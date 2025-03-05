// Copyright 2020 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package services implements services for getting data from Spanner
// and streaming data into pubsub
package services

import (
	"context"
	"fmt"
	"hash/fnv"
	"regexp"
	"strconv"
	"strings"

	"cloud.google.com/go/spanner"
	"github.com/ahmetb/go-linq"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/cloudspannerecosystem/dynamodb-adapter/config"
	"github.com/cloudspannerecosystem/dynamodb-adapter/models"
	"github.com/cloudspannerecosystem/dynamodb-adapter/pkg/errors"
	"github.com/cloudspannerecosystem/dynamodb-adapter/pkg/logger"
	"github.com/cloudspannerecosystem/dynamodb-adapter/storage"
	translator "github.com/cloudspannerecosystem/dynamodb-adapter/translator/utils"
	"github.com/cloudspannerecosystem/dynamodb-adapter/utils"
)

const (
	regexPattern = `^[a-zA-Z_][a-zA-Z0-9_.]*(\.[a-zA-Z_][a-zA-Z0-9_.]*)+\s*=\s*@\w+$`
)

var (
	re = regexp.MustCompile(regexPattern)
)

var (
	// Regular expressions to match the beginning of the query
	selectRegex = regexp.MustCompile(`(?i)^\s*SELECT`)
	insertRegex = regexp.MustCompile(`(?i)^\s*INSERT`)
	updateRegex = regexp.MustCompile(`(?i)^\s*UPDATE`)
	deleteRegex = regexp.MustCompile(`(?i)^\s*DELETE`)
)

// getSpannerProjections makes a projection array of columns
func getSpannerProjections(projectionExpression, table string, expressionAttributeNames map[string]string) []string {
	if projectionExpression == "" {
		return nil
	}
	expressionAttributes := expressionAttributeNames
	projections := strings.Split(projectionExpression, ",")
	projectionCols := []string{}
	for _, pro := range projections {
		pro = strings.TrimSpace(pro)
		if val, ok := expressionAttributes[pro]; ok {
			projectionCols = append(projectionCols, val)
		} else {
			projectionCols = append(projectionCols, pro)
		}
	}

	linq.From(projectionCols).IntersectByT(linq.From(models.TableColumnMap[utils.ChangeTableNameForSpanner(table)]), func(str string) string {
		return str
	}).ToSlice(&projectionCols)
	return projectionCols
}

// Put writes an object to Spanner
func Put(ctx context.Context, tableName string, putObj map[string]interface{}, expr *models.UpdateExpressionCondition, conditionExp string, expressionAttr, oldRes map[string]interface{}, spannerRow map[string]interface{}) (map[string]interface{}, error) {
	tableConf, err := config.GetTableConf(tableName)
	if err != nil {
		return nil, err
	}

	tableName = tableConf.ActualTable
	e, err := utils.CreateConditionExpression(conditionExp, expressionAttr)
	if err != nil {
		return nil, err
	}
	newResp, err := storage.GetStorageInstance().SpannerPut(ctx, tableName, putObj, e, expr, spannerRow)
	if err != nil {
		return nil, err
	}

	if oldRes == nil {
		return oldRes, nil
	}
	updateResp := map[string]interface{}{}
	for k, v := range oldRes {
		updateResp[k] = v
	}
	for k, v := range newResp {
		updateResp[k] = v
	}
	return updateResp, nil
}

// Add checks the expression for converting the data
func Add(ctx context.Context, tableName string, attrMap map[string]interface{}, condExpression string, m, expressionAttr map[string]interface{}, expr *models.UpdateExpressionCondition, oldRes map[string]interface{}) (map[string]interface{}, error) {
	tableConf, err := config.GetTableConf(tableName)
	if err != nil {
		return nil, err
	}
	tableName = tableConf.ActualTable

	e, err := utils.CreateConditionExpression(condExpression, expressionAttr)
	if err != nil {
		return nil, err
	}

	newResp, err := storage.GetStorageInstance().SpannerAdd(ctx, tableName, m, e, expr)
	if err != nil {
		return nil, err
	}
	if oldRes == nil {
		return newResp, nil
	}
	updateResp := map[string]interface{}{}
	for k, v := range oldRes {
		updateResp[k] = v
	}
	for k, v := range newResp {
		updateResp[k] = v
	}

	return updateResp, nil
}

// Del checks the expression for saving the data
func Del(ctx context.Context, tableName string, attrMap map[string]interface{}, condExpression string, expressionAttr map[string]interface{}, expr *models.UpdateExpressionCondition) (map[string]interface{}, error) {
	logger.LogDebug(expressionAttr)
	tableConf, err := config.GetTableConf(tableName)
	if err != nil {
		return nil, err
	}

	tableName = tableConf.ActualTable

	e, err := utils.CreateConditionExpression(condExpression, expressionAttr)
	if err != nil {
		return nil, err
	}

	err = storage.GetStorageInstance().SpannerDel(ctx, tableName, expressionAttr, e, expr)
	if err != nil {
		return nil, err
	}
	sKey := tableConf.SortKey
	pKey := tableConf.PartitionKey
	res, _, err := storage.GetStorageInstance().SpannerGet(ctx, tableName, attrMap[pKey], attrMap[sKey], nil)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// BatchGet for batch operation for getting data
func BatchGet(ctx context.Context, tableName string, keyMapArray []map[string]interface{}) ([]map[string]interface{}, error) {
	if len(keyMapArray) == 0 {
		var resp = make([]map[string]interface{}, 0)
		return resp, nil
	}
	tableConf, err := config.GetTableConf(tableName)
	if err != nil {
		return nil, err
	}
	tableName = tableConf.ActualTable

	var pValues []interface{}
	var sValues []interface{}
	for i := 0; i < len(keyMapArray); i++ {
		pValue := keyMapArray[i][tableConf.PartitionKey]
		if tableConf.SortKey != "" {
			sValue := keyMapArray[i][tableConf.SortKey]
			sValues = append(sValues, sValue)
		}
		pValues = append(pValues, pValue)
	}
	return storage.GetStorageInstance().SpannerBatchGet(ctx, tableName, pValues, sValues, nil)
}

// BatchPut writes bulk records to Spanner
func BatchPut(ctx context.Context, tableName string, arrAttrMap []map[string]interface{}, spannerRow []map[string]interface{}) error {
	if len(arrAttrMap) <= 0 {
		return errors.New("ValidationException")
	}

	oldRes, err := BatchGet(ctx, tableName, arrAttrMap)
	if err != nil {
		return err
	}
	tableConf, err := config.GetTableConf(tableName)
	if err != nil {
		return err
	}
	tableName = tableConf.ActualTable
	err = storage.GetStorageInstance().SpannerBatchPut(ctx, tableName, arrAttrMap, spannerRow)
	if err != nil {
		return err
	}
	go func() {
		if len(oldRes) == len(arrAttrMap) {
			for i := 0; i < len(arrAttrMap); i++ {
				go StreamDataToThirdParty(oldRes[i], arrAttrMap[i], tableName)
			}
		} else {
			for i := 0; i < len(arrAttrMap); i++ {
				go StreamDataToThirdParty(nil, arrAttrMap[i], tableName)
			}

		}
	}()
	return nil
}

// GetWithProjection get table data with projection
func GetWithProjection(ctx context.Context, tableName string, primaryKeyMap map[string]interface{}, projectionExpression string, expressionAttributeNames map[string]string) (map[string]interface{}, map[string]interface{}, error) {
	if primaryKeyMap == nil {
		return nil, nil, errors.New("ValidationException")
	}
	tableConf, err := config.GetTableConf(tableName)
	if err != nil {
		return nil, nil, err
	}

	tableName = tableConf.ActualTable

	projectionCols := getSpannerProjections(projectionExpression, tableName, expressionAttributeNames)
	pValue := primaryKeyMap[tableConf.PartitionKey]
	var sValue interface{}
	if tableConf.SortKey != "" {
		sValue = primaryKeyMap[tableConf.SortKey]
	}
	return storage.GetStorageInstance().SpannerGet(ctx, tableName, pValue, sValue, projectionCols)
}

// QueryAttributes from Spanner
func QueryAttributes(ctx context.Context, query models.Query) (map[string]interface{}, string, error) {
	tableConf, err := config.GetTableConf(query.TableName)
	if err != nil {
		return nil, "", err
	}
	var sKey string
	var pKey string
	tPKey := tableConf.PartitionKey
	tSKey := tableConf.SortKey
	if query.IndexName != "" {
		conf := tableConf.Indices[query.IndexName]
		query.IndexName = strings.Replace(query.IndexName, "-", "_", -1)

		if tableConf.ActualTable != query.TableName {
			query.TableName = tableConf.ActualTable
		}

		sKey = conf.SortKey
		pKey = conf.PartitionKey
	} else {
		sKey = tableConf.SortKey
		pKey = tableConf.PartitionKey
	}
	if pKey == "" {
		pKey = tPKey
		sKey = tSKey
	}

	originalLimit := query.Limit
	query.Limit = originalLimit + 1

	stmt, cols, isCountQuery, offset, hash, err := createSpannerQuery(&query, tPKey, pKey, sKey)
	if err != nil {
		return nil, hash, err
	}
	logger.LogDebug(stmt)
	resp, err := storage.GetStorageInstance().ExecuteSpannerQuery(ctx, query.TableName, cols, isCountQuery, stmt)
	if err != nil {
		return nil, hash, err
	}
	if isCountQuery {
		return resp[0], hash, nil
	}
	finalResp := make(map[string]interface{})
	length := len(resp)
	if length == 0 {
		finalResp["Count"] = 0
		finalResp["Items"] = []map[string]interface{}{}
		finalResp["LastEvaluatedKey"] = nil
		return finalResp, hash, nil
	}
	if int64(length) > originalLimit {
		finalResp["Count"] = length - 1
		last := resp[length-2]
		if sKey != "" {
			finalResp["LastEvaluatedKey"] = map[string]interface{}{"offset": originalLimit + offset, pKey: last[pKey], tPKey: last[tPKey], sKey: last[sKey], tSKey: last[tSKey]}
		} else {
			finalResp["LastEvaluatedKey"] = map[string]interface{}{"offset": originalLimit + offset, pKey: last[pKey], tPKey: last[tPKey]}
		}
		finalResp["Items"] = resp[:length-1]
	} else {
		if query.StartFrom != nil && length-1 == 1 {
			finalResp["Items"] = resp
		} else {
			finalResp["Items"] = resp
		}
		finalResp["Count"] = length
		finalResp["Items"] = resp
		finalResp["LastEvaluatedKey"] = nil
	}
	return finalResp, hash, nil
}

func createSpannerQuery(query *models.Query, tPkey, pKey, sKey string) (spanner.Statement, []string, bool, int64, string, error) {
	stmt := spanner.Statement{}
	cols, colstr, isCountQuery, err := parseSpannerColumns(query, tPkey, pKey, sKey)
	if err != nil {
		return stmt, cols, isCountQuery, 0, "", err
	}
	tableName := parseSpannerTableName(query)
	whereCondition, m := parseSpannerCondition(query, pKey, sKey)
	offsetString, offset := parseOffset(query)
	orderBy := parseSpannerSorting(query, isCountQuery, pKey, sKey)
	limitClause := parseLimit(query, isCountQuery)
	finalQuery := "SELECT " + colstr + " FROM " + tableName + " " + whereCondition + orderBy + limitClause + offsetString
	stmt.SQL = finalQuery
	h := fnv.New64a()
	h.Write([]byte(finalQuery))
	val := h.Sum64()
	rs := strconv.FormatUint(val, 10)
	stmt.Params = m
	return stmt, cols, isCountQuery, offset, rs, nil
}

func parseSpannerColumns(query *models.Query, tPkey, pKey, sKey string) ([]string, string, bool, error) {
	if query == nil {
		return []string{}, "", false, errors.New("Query is not present")
	}
	colStr := ""
	if query.OnlyCount {
		return []string{"count"}, "COUNT(" + pKey + ") AS count", true, nil
	}
	table := utils.ChangeTableNameForSpanner(query.TableName)
	var cols []string
	if query.ProjectionExpression != "" {
		cols = getSpannerProjections(query.ProjectionExpression, query.TableName, query.ExpressionAttributeNames)
		insertPKey := true
		for i := 0; i < len(cols); i++ {
			if cols[i] == pKey {
				insertPKey = false
				break
			}
		}
		if insertPKey {
			cols = append(cols, pKey)
		}
		if sKey != "" {
			insertSKey := true
			for i := 0; i < len(cols); i++ {
				if cols[i] == sKey {
					insertSKey = false
					break
				}
			}
			if insertSKey {
				cols = append(cols, sKey)
			}
		}
		if tPkey != pKey {
			insertSKey := true
			for i := 0; i < len(cols); i++ {
				if cols[i] == tPkey {
					insertSKey = false
					break
				}
			}
			if insertSKey {
				cols = append(cols, tPkey)
			}
		}

	} else {
		cols = models.TableColumnMap[table]
	}
	for i := 0; i < len(cols); i++ {
		if cols[i] == "commit_timestamp" {
			continue
		}
		colStr += table + ".`" + cols[i] + "`,"
	}
	colStr = strings.Trim(colStr, ",")
	return cols, colStr, false, nil
}

func parseSpannerTableName(query *models.Query) string {
	tableName := utils.ChangeTableNameForSpanner(query.TableName)
	if query.IndexName != "" {
		tableName += "@{FORCE_INDEX=" + query.IndexName + "}"
	}
	return tableName
}

func parseSpannerCondition(query *models.Query, pKey, sKey string) (string, map[string]interface{}) {
	params := make(map[string]interface{})
	whereClause := "WHERE "

	if sKey != "" {
		whereClause += sKey + " is not null "
	}

	if query.RangeExp != "" {
		whereClause, query.RangeExp = createWhereClause(whereClause, query.RangeExp, "rangeExp", query.RangeValMap, params)
	}

	if query.FilterExp != "" {
		whereClause, query.FilterExp = createWhereClause(whereClause, query.FilterExp, "filterExp", query.RangeValMap, params)
	}

	if whereClause == "WHERE " {
		whereClause = " "
	}
	return whereClause, params
}

func createWhereClause(whereClause string, expression string, queryVar string, RangeValueMap map[string]interface{}, params map[string]interface{}) (string, string) {
	_, _, expression = utils.ParseBeginsWith(expression)
	expression = strings.ReplaceAll(expression, "begins_with", "STARTS_WITH")
	trimmedString := strings.TrimSpace(whereClause)
	if whereClause != "WHERE " && !strings.HasSuffix(trimmedString, "AND") {
		whereClause += " AND "
	}
	count := 1
	for k, v := range RangeValueMap {
		if strings.Contains(expression, k) {
			str := queryVar + strconv.Itoa(count)
			expression = strings.ReplaceAll(expression, k, "@"+str)
			params[str] = v
			count++
		}
	}
	// Handle JSON paths if the expression is structured correctly
	if re.MatchString(expression) {
		expression := strings.TrimSpace(expression)
		expressionParts := strings.Split(expression, "=")
		expressionParts[0] = strings.TrimSpace(expressionParts[0])
		jsonFields := strings.Split(expressionParts[0], ".")

		// Construct new JSON_VALUE expression
		newExpression := fmt.Sprintf("JSON_VALUE(%s, '$.%s') = %s", jsonFields[0], strings.Join(jsonFields[1:], "."), expressionParts[1])
		whereClause = whereClause + " " + newExpression
	} else if expression != "" {
		whereClause = whereClause + expression
	}
	return whereClause, expression
}

func parseOffset(query *models.Query) (string, int64) {
	logger.LogDebug(query)
	if query.StartFrom != nil {
		offset, ok := query.StartFrom["offset"].(float64)
		if ok {
			return " OFFSET " + strconv.FormatInt(int64(offset), 10), int64(offset)
		}
	}
	return "", 0
}

func parseSpannerSorting(query *models.Query, isCountQuery bool, pKey, sKey string) string {
	if isCountQuery {
		return " "
	}
	if sKey == "" {
		return " "
	}

	if query.SortAscending {
		return " ORDER BY " + sKey + " ASC "
	}
	return " ORDER BY " + sKey + " DESC "
}

func parseLimit(query *models.Query, isCountQuery bool) string {
	if isCountQuery {
		return ""
	}
	if query.Limit == 0 {
		return " LIMIT 5000 "
	}
	return " LIMIT " + strconv.FormatInt(query.Limit, 10)
}

// BatchGetWithProjection from Spanner
func BatchGetWithProjection(ctx context.Context, tableName string, keyMapArray []map[string]interface{}, projectionExpression string, expressionAttributeNames map[string]string) ([]map[string]interface{}, error) {
	if len(keyMapArray) == 0 {
		var resp = make([]map[string]interface{}, 0)
		return resp, nil
	}
	tableConf, err := config.GetTableConf(tableName)
	if err != nil {
		return nil, err
	}
	tableName = tableConf.ActualTable

	projectionCols := getSpannerProjections(projectionExpression, tableName, expressionAttributeNames)
	var pValues []interface{}
	var sValues []interface{}
	for i := 0; i < len(keyMapArray); i++ {
		pValue := keyMapArray[i][tableConf.PartitionKey]
		if tableConf.SortKey != "" {
			sValue := keyMapArray[i][tableConf.SortKey]
			sValues = append(sValues, sValue)
		}
		pValues = append(pValues, pValue)
	}
	return storage.GetStorageInstance().SpannerBatchGet(ctx, tableName, pValues, sValues, projectionCols)
}

// Delete service
func Delete(ctx context.Context, tableName string, primaryKeyMap map[string]interface{}, condExpression string, attrMap map[string]interface{}, expr *models.UpdateExpressionCondition) error {
	tableConf, err := config.GetTableConf(tableName)
	if err != nil {
		return err
	}
	tableName = tableConf.ActualTable
	e, err := utils.CreateConditionExpression(condExpression, attrMap)
	if err != nil {
		return err
	}
	return storage.GetStorageInstance().SpannerDelete(ctx, tableName, primaryKeyMap, e, expr)
}

// BatchDelete service
func BatchDelete(ctx context.Context, tableName string, keyMapArray []map[string]interface{}) error {
	tableConf, err := config.GetTableConf(tableName)
	if err != nil {
		return err
	}
	oldRes, _ := BatchGet(ctx, tableName, keyMapArray)

	tableName = tableConf.ActualTable
	err = storage.GetStorageInstance().SpannerBatchDelete(ctx, tableName, keyMapArray)
	if err != nil {
		return err
	}
	go func() {
		if len(oldRes) == len(keyMapArray) {
			for i := 0; i < len(keyMapArray); i++ {
				go StreamDataToThirdParty(oldRes[i], keyMapArray[i], tableName)
			}
		} else {
			for i := 0; i < len(keyMapArray); i++ {
				go StreamDataToThirdParty(nil, keyMapArray[i], tableName)
			}

		}
	}()
	return nil
}

// Scan service
func Scan(ctx context.Context, scanData models.ScanMeta) (map[string]interface{}, error) {
	query := models.Query{}
	query.TableName = scanData.TableName
	query.Limit = scanData.Limit
	if query.Limit == 0 {
		query.Limit = models.GlobalConfig.Spanner.QueryLimit
	}
	query.StartFrom = scanData.StartFrom
	query.RangeValMap = scanData.ExpressionAttributeMap
	query.IndexName = scanData.IndexName
	query.FilterExp = scanData.FilterExpression
	query.ExpressionAttributeNames = scanData.ExpressionAttributeNames
	query.OnlyCount = scanData.OnlyCount
	query.ProjectionExpression = scanData.ProjectionExpression

	for k, v := range query.ExpressionAttributeNames {
		query.FilterExp = strings.ReplaceAll(query.FilterExp, k, v)
	}

	rs, _, err := QueryAttributes(ctx, query)
	return rs, err
}

// Remove for remove operation in update
func Remove(ctx context.Context, tableName string, updateAttr models.UpdateAttr, actionValue string, expr *models.UpdateExpressionCondition, oldRes map[string]interface{}) (map[string]interface{}, error) {
	actionValue = strings.ReplaceAll(actionValue, " ", "")
	colsToRemove := strings.Split(actionValue, ",")
	tableConf, err := config.GetTableConf(tableName)
	if err != nil {
		return nil, err
	}
	tableName = tableConf.ActualTable
	e, err := utils.CreateConditionExpression(updateAttr.ConditionExpression, updateAttr.ExpressionAttributeMap)
	if err != nil {
		return nil, err
	}
	err = storage.GetStorageInstance().SpannerRemove(ctx, tableName, updateAttr.PrimaryKeyMap, e, expr, colsToRemove, oldRes)
	if err != nil {
		return nil, err
	}
	if oldRes == nil {
		return oldRes, nil
	}
	updateResp := map[string]interface{}{}
	for k, v := range oldRes {
		updateResp[k] = v
	}
	for _, target := range colsToRemove {
		if strings.Contains(target, "[") && strings.Contains(target, "]") {
			// Handle list index removal
			listAttr, idx := utils.ParseListRemoveTarget(target)
			if idx != -1 {
				if list, ok := oldRes[listAttr].([]interface{}); ok {
					oldRes[listAttr] = utils.RemoveListElement(list, idx)
				}
			} else {
				// Handle invalid list index format
				return nil, fmt.Errorf("invalid list index format for target %q", target)
			}
		} else {
			// Handle direct column removal
			delete(updateResp, target)
		}
	}
	return updateResp, nil
}

// ExecuteStatement service API handler function
func ExecuteStatement(ctx context.Context, executeStatement models.ExecuteStatement) (map[string]interface{}, error) {

	query := strings.TrimSpace(executeStatement.Statement) // Remove any leading or trailing whitespace
	queryUpper := strings.ToUpper(query)

	switch {
	case selectRegex.MatchString(queryUpper):
		return ExecuteStatementForSelect(ctx, executeStatement)
	case insertRegex.MatchString(queryUpper):
		return ExecuteStatementForInsert(ctx, executeStatement)
	case updateRegex.MatchString(queryUpper):
		return ExecuteStatementForUpdate(ctx, executeStatement)
	case deleteRegex.MatchString(queryUpper):
		return ExecuteStatementForDelete(ctx, executeStatement)
	default:
		return nil, nil
	}

}

// parsePartiQlToSpannerforSelect converts a PartiQL statement with parameters to a Spanner SQL statement with parameters.
// Parameters:
// - ctx: The context for managing request-scoped values, cancelations, and timeouts.
// - executeStatement: The object containing the PartiQL query string and parameters to be translated.
//
// Returns:
// - spanner.Statement: A Google Cloud Spanner statement ready to be executed.
// - error: An error object, if an error occurs during translation or parameter conversion.
func parsePartiQlToSpannerforSelect(ctx context.Context, executeStatement models.ExecuteStatement) (spanner.Statement, error) {
	stmt := spanner.Statement{}
	paramMap := make(map[string]interface{})
	var err error

	translatorObj := translator.Translator{}

	queryMap, err := translatorObj.ToSpannerSelect(executeStatement.Statement)
	if err != nil {
		return stmt, err
	}

	queryStmt := queryMap.SpannerQuery
	if executeStatement.Limit != 0 {
		queryMap.Limit = strconv.FormatInt(executeStatement.Limit, 10)
	}

	err = handleParameters(executeStatement.Parameters, queryMap.Where, &paramMap, &queryStmt)
	if err != nil {
		return stmt, err
	}

	stmt.SQL = queryMap.SpannerQuery
	stmt.Params = paramMap
	return stmt, nil
}
func handleParameters(parameters []*dynamodb.AttributeValue, whereConditions []translator.Condition, paramMap *map[string]interface{}, queryStmt *string) error {
	for i, val := range parameters {
		if val.S != nil {
			(*paramMap)[whereConditions[i].Column] = *val.S
			*queryStmt = strings.Replace(*queryStmt, "?", "@val"+strconv.Itoa(i), 1)
		} else if val.N != nil {
			floatVal, err := strconv.ParseFloat(*val.N, 64)
			if err != nil {
				return fmt.Errorf("failed to convert the string type to the float")
			}
			(*paramMap)[whereConditions[i].Column] = floatVal
			*queryStmt = strings.Replace(*queryStmt, "?", "@val"+strconv.Itoa(i), 1)
		} else if val.BOOL != nil {
			(*paramMap)[whereConditions[i].Column] = *val.BOOL
			*queryStmt = strings.Replace(*queryStmt, "?", "@val"+strconv.Itoa(i), 1)
		} else if val.SS != nil {
			ss := make([]interface{}, len(val.SS))
			for index, v := range val.SS {
				ss[index] = *v
			}
			(*paramMap)[whereConditions[i].Column] = ss
			*queryStmt = strings.Replace(*queryStmt, "?", "@val"+strconv.Itoa(i), 1)
		} else {
			return fmt.Errorf("unsupported datatype")
		}
	}
	return nil
}

// ExecuteStatementForSelect executes a select statement on a Spanner database, converting a PartiQL statement to a Spanner statement.
//
// Parameters:
// - ctx: Context for managing request-scoped values, cancellations, and timeouts.
// - executeStatement: Contains the PartiQL select statement and parameters to be executed.
//
// Returns:
// - map[string]interface{}: A map containing the fetched items under the key "Items".
// - error: An error object, if any issues arise during the execution process.
func ExecuteStatementForSelect(ctx context.Context, executeStatement models.ExecuteStatement) (map[string]interface{}, error) {
	spannerStatement, err := parsePartiQlToSpannerforSelect(ctx, executeStatement)
	if err != nil {
		return nil, err

	}
	resp, err := storage.GetStorageInstance().ExecuteSpannerQuery(ctx, executeStatement.TableName, []string{}, false, spannerStatement)
	if err != nil {
		return nil, err
	}
	finalResp := make(map[string]interface{})
	finalResp["Items"] = resp
	return finalResp, nil
}

// ExecuteStatementForInsert executes an insert statement on a Spanner database by converting a PartiQL insert statement
// to a Spanner compatible format and then performing the insert operation.
//
// Parameters:
// - ctx: The context for managing request-scoped values, cancellations, and timeouts.
// - executeStatement: Contains the PartiQL insert statement and the attributes to be inserted.
//
// Returns:
// - map[string]interface{}: A map containing the result of the insert operation.
// - error: An error object, if any issues arise during the execution process.
func ExecuteStatementForInsert(ctx context.Context, executeStatement models.ExecuteStatement) (map[string]interface{}, error) {
	translatorObj := translator.Translator{}
	parsedQueryObj, err := translatorObj.ToSpannerInsert(executeStatement.Statement)
	if err != nil {
		return nil, err
	}

	colDLL, ok := models.TableDDL[utils.ChangeTableNameForSpanner(executeStatement.TableName)]
	if !ok {
		return nil, fmt.Errorf("ResourceNotFoundException: %s", executeStatement.TableName)
	}

	newMap := make(map[string]interface{})
	attrParams := executeStatement.AttrParams

	columns := parsedQueryObj.Columns
	values := parsedQueryObj.Values

	// Determine the number of parameters to use and iterate accordingly
	var paramsCount int
	if len(attrParams) > 0 {
		paramsCount = len(attrParams)
	} else {
		paramsCount = len(columns)
	}

	for i := 0; i < paramsCount; i++ {
		var value interface{}
		if len(attrParams) > 0 {
			value = attrParams[i]
		} else {
			value = values[i]
		}

		columnName := columns[i]
		columnType := colDLL[columnName]

		convertedValue, err := convertType(columnName, value, columnType)
		if err != nil {
			return nil, err
		}
		newMap[columnName] = convertedValue
	}
	result, err := Put(ctx, executeStatement.TableName, newMap, nil, "", nil, nil, nil)
	if err != nil {
		return result, err
	}
	return result, nil
}

// ExecuteStatementForUpdate executes an update statement on a Spanner database by converting a PartiQL update statement
// to a Spanner compatible format and performing the update operation.
//
// Parameters:
// - ctx: The context for managing request-scoped values, cancellations, and timeouts.
// - executeStatement: Contains the PartiQL update statement and the parameters for the update.
//
// Returns:
// - map[string]interface{}: A map containing the result of the update operation or nil if successful.
// - error: An error object, if any issues arise during the execution process.
func ExecuteStatementForUpdate(ctx context.Context, executeStatement models.ExecuteStatement) (map[string]interface{}, error) {
	translatorObj := translator.Translator{}
	parsedQueryObj, err := translatorObj.ToSpannerUpdate(executeStatement.Statement)
	if err != nil {
		return nil, err
	}
	paramMap := make(map[string]interface{})
	if len(executeStatement.Parameters) > 0 {
		j := len(parsedQueryObj.UpdateSetValues)
		for i, val := range parsedQueryObj.UpdateSetValues {
			colDLL, ok := models.TableDDL[utils.ChangeTableNameForSpanner(executeStatement.TableName)]
			if !ok {
				return nil, fmt.Errorf("ResourceNotFoundException: %s", executeStatement.TableName)
			}
			convertedValue, err := convertType(val.Column, executeStatement.AttrParams[i], colDLL[val.Column])
			if err != nil {
				return nil, err
			}
			paramMap[val.Column] = convertedValue

		}
		for _, val := range parsedQueryObj.Clauses {
			colDLL, ok := models.TableDDL[utils.ChangeTableNameForSpanner(executeStatement.TableName)]
			if !ok {
				return nil, fmt.Errorf("ResourceNotFoundException: %s", executeStatement.TableName)
			}
			convertedValue, err := convertType(val.Column, executeStatement.AttrParams[j], colDLL[val.Column])
			if err != nil {
				return nil, err
			}
			paramMap[val.Column] = convertedValue
			j++
		}
		parsedQueryObj.Params = paramMap
	}
	res, err := storage.GetStorageInstance().InsertUpdateOrDeleteStatement(ctx, parsedQueryObj)
	if err != nil {
		return res, err
	}
	return nil, err
}

// ExecuteStatementForDelete executes a delete statement on a Spanner database by converting a PartiQL delete statement
// to a Spanner compatible format and performing the delete operation.
//
// Parameters:
// - ctx: The context for managing request-scoped values, cancellations, and timeouts.
// - executeStatement: Contains the PartiQL delete statement and the parameters for the deletion.
//
// Returns:
// - map[string]interface{}: A map containing the result of the delete operation.
// - error: An error object, if any issues arise during the execution process.
func ExecuteStatementForDelete(ctx context.Context, executeStatement models.ExecuteStatement) (map[string]interface{}, error) {

	translatorObj := translator.Translator{}
	parsedQueryObj, err := translatorObj.ToSpannerDelete(executeStatement.Statement)
	if err != nil {
		return nil, err
	}
	newMap := make(map[string]interface{})
	if len(executeStatement.AttrParams) > 0 {
		for i, val := range parsedQueryObj.Clauses {
			colDLL, ok := models.TableDDL[utils.ChangeTableNameForSpanner(executeStatement.TableName)]
			if !ok {
				return nil, fmt.Errorf("ResourceNotFoundException: %s", executeStatement.TableName)
			}
			convertedValue, err := convertType(val.Column, executeStatement.AttrParams[i], colDLL[val.Column])
			if err != nil {
				return nil, err
			}
			newMap[val.Column] = convertedValue
		}
		parsedQueryObj.Params = newMap
	}

	res, err := storage.GetStorageInstance().InsertUpdateOrDeleteStatement(ctx, parsedQueryObj)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func convertType(columnName string, val interface{}, columntype string) (interface{}, error) {
	switch columntype {
	case "S":
		// Ensure the value is a string
		return utils.TrimSingleQuotes(fmt.Sprintf("%v", val)), nil

	case "N":
		// Convert to float64
		floatValue, err := strconv.ParseFloat(fmt.Sprintf("%v", val), 64)
		if err != nil {
			return nil, fmt.Errorf("error converting to float64: %v", err)
		}
		return floatValue, nil

	case "BOOL":
		// Convert to boolean
		boolValue, err := strconv.ParseBool(fmt.Sprintf("%v", val))
		if err != nil {
			return nil, fmt.Errorf("error converting to bool: %v", err)
		}
		return boolValue, nil
	case "L":
		// Convert to list (array or slice in Go)
		listValue, ok := val.([]interface{})
		if !ok {
			return nil, fmt.Errorf("expected a list for column %s, but got: %v", columnName, val)
		}
		return listValue, nil // Return the slice as is or manipulate as neede

	default:
		return nil, fmt.Errorf("unsupported data type: %s", columntype)
	}

}
