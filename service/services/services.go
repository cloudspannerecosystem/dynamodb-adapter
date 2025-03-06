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
	"strconv"
	"strings"
	"sync"

	"cloud.google.com/go/spanner"
	"github.com/ahmetb/go-linq"
	"github.com/cloudspannerecosystem/dynamodb-adapter/config"
	"github.com/cloudspannerecosystem/dynamodb-adapter/models"
	"github.com/cloudspannerecosystem/dynamodb-adapter/pkg/errors"
	"github.com/cloudspannerecosystem/dynamodb-adapter/pkg/logger"
	"github.com/cloudspannerecosystem/dynamodb-adapter/storage"
	"github.com/cloudspannerecosystem/dynamodb-adapter/utils"
)

type Storage interface {
	GetSpannerClient() (*spanner.Client, error)
	SpannerTransactWritePut(ctx context.Context, table string, m map[string]interface{}, eval *models.Eval, expr *models.UpdateExpressionCondition, txn *spanner.ReadWriteTransaction) (map[string]interface{}, *spanner.Mutation, error)
	SpannerGet(ctx context.Context, tableName string, pKeys, sKeys interface{}, projectionCols []string) (map[string]interface{}, error)
	TransactWriteSpannerDel(ctx context.Context, table string, m map[string]interface{}, eval *models.Eval, expr *models.UpdateExpressionCondition, txn *spanner.ReadWriteTransaction) (*spanner.Mutation, error)
	TransactWriteSpannerAdd(ctx context.Context, table string, n map[string]interface{}, eval *models.Eval, expr *models.UpdateExpressionCondition, txn *spanner.ReadWriteTransaction) (map[string]interface{}, *spanner.Mutation, error)
	TransactWriteSpannerRemove(ctx context.Context, table string, m map[string]interface{}, eval *models.Eval, expr *models.UpdateExpressionCondition, colsToRemove []string, txn *spanner.ReadWriteTransaction) (*spanner.Mutation, error)
}
type Service interface {
	MayIReadOrWrite(tableName string, isWrite bool, user string) bool
	TransactWritePut(ctx context.Context, tableName string, putObj map[string]interface{}, expr *models.UpdateExpressionCondition, conditionExp string, expressionAttr, oldRes map[string]interface{}, txn *spanner.ReadWriteTransaction) (map[string]interface{}, *spanner.Mutation, error)
	TransactWriteDel(ctx context.Context, tableName string, attrMap map[string]interface{}, condExpression string, expressionAttr map[string]interface{}, expr *models.UpdateExpressionCondition, txn *spanner.ReadWriteTransaction) (map[string]interface{}, *spanner.Mutation, error)
	TransactWriteAdd(ctx context.Context, tableName string, attrMap map[string]interface{}, condExpression string, m, expressionAttr map[string]interface{}, expr *models.UpdateExpressionCondition, oldRes map[string]interface{}, txn *spanner.ReadWriteTransaction) (map[string]interface{}, *spanner.Mutation, error)
	TransactWriteRemove(ctx context.Context, tableName string, updateAttr models.UpdateAttr, actionValue string, expr *models.UpdateExpressionCondition, oldRes map[string]interface{}, txn *spanner.ReadWriteTransaction) (map[string]interface{}, *spanner.Mutation, error)
	GetWithProjection(ctx context.Context, tableName string, primaryKeyMap map[string]interface{}, projectionExpression string, expressionAttributeNames map[string]string) (map[string]interface{}, error)
}

type spannerService struct {
	spannerClient *spanner.Client
	st            Storage
}

var service Service // Service interface variable

var once sync.Once

func SetServiceInstance(s Service) {
	service = s
}

func GetServiceInstance() Service {
	once.Do(func() {
		storageInstance := storage.GetStorageInstance()
		spannerClient, err := storageInstance.GetSpannerClient()
		if err != nil {
			logger.LogError("Failed to initialize Spanner client: %v", err)
			panic(err)
		}

		service = &spannerService{
			spannerClient: spannerClient,
			st:            storageInstance,
		}
	})
	return service
}

// MayIReadOrWrite for checking the operation is allowed or not
func (s *spannerService) MayIReadOrWrite(table string, IsMutation bool, operation string) bool {
	return true
}

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
func Put(ctx context.Context, tableName string, putObj map[string]interface{}, expr *models.UpdateExpressionCondition, conditionExp string, expressionAttr, oldRes map[string]interface{}) (map[string]interface{}, error) {
	tableConf, err := config.GetTableConf(tableName)
	if err != nil {
		return nil, err
	}

	tableName = tableConf.ActualTable
	e, err := utils.CreateConditionExpression(conditionExp, expressionAttr)
	if err != nil {
		return nil, err
	}
	newResp, err := storage.GetStorageInstance().SpannerPut(ctx, tableName, putObj, e, expr)
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
	res, err := storage.GetStorageInstance().SpannerGet(ctx, tableName, attrMap[pKey], attrMap[sKey], nil)
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
func BatchPut(ctx context.Context, tableName string, arrAttrMap []map[string]interface{}) error {
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
	err = storage.GetStorageInstance().SpannerBatchPut(ctx, tableName, arrAttrMap)
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
func (s *spannerService) GetWithProjection(ctx context.Context, tableName string, primaryKeyMap map[string]interface{}, projectionExpression string, expressionAttributeNames map[string]string) (map[string]interface{}, error) {
	if primaryKeyMap == nil {
		return nil, errors.New("ValidationException")
	}
	tableConf, err := config.GetTableConf(tableName)
	if err != nil {
		return nil, err
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

	if whereClause != "WHERE " {
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
	whereClause += expression
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

// TransactWritePut manages a transactional put operation in Spanner, ensuring old data is fetched and conditions are evaluated.
func (s *spannerService) TransactWritePut(ctx context.Context, tableName string, putObj map[string]interface{}, expr *models.UpdateExpressionCondition, conditionExp string, expressionAttr, oldRes map[string]interface{}, txn *spanner.ReadWriteTransaction) (map[string]interface{}, *spanner.Mutation, error) {
	// Fetch the table configuration to retrieve partition and sort keys
	tableConf, err := config.GetTableConf(tableName)
	if err != nil {
		return nil, nil, err
	}

	// Update tableName to its actual value from the table configuration
	tableName = tableConf.ActualTable

	// Create the condition expression for the transaction
	e, err := utils.CreateConditionExpression(conditionExp, expressionAttr)
	if err != nil {
		return nil, nil, err
	}

	// Perform the transactional write operation
	newResp, mut, err := s.st.SpannerTransactWritePut(ctx, tableName, putObj, e, expr, txn)
	if err != nil {
		return nil, nil, err
	}

	// If there is no old response, return early with nil mutation
	if oldRes == nil {
		return oldRes, nil, nil
	}

	// Combine the old response with the new response
	updateResp := map[string]interface{}{}
	for k, v := range oldRes {
		updateResp[k] = v
	}
	for k, v := range newResp {
		updateResp[k] = v
	}

	// Return the updated response and the mutation
	return newResp, mut, nil
}

// TransactWriteDel performs a transactional delete on Spanner
func (s *spannerService) TransactWriteDel(ctx context.Context, tableName string, attrMap map[string]interface{}, condExpression string, expressionAttr map[string]interface{}, expr *models.UpdateExpressionCondition, txn *spanner.ReadWriteTransaction) (map[string]interface{}, *spanner.Mutation, error) {
	// Fetch the table configuration and update the table name
	tableConf, err := config.GetTableConf(tableName)
	if err != nil {
		return nil, nil, err
	}

	tableName = tableConf.ActualTable

	// Create the condition expression for the transaction
	e, err := utils.CreateConditionExpression(condExpression, expressionAttr)
	if err != nil {
		return nil, nil, err
	}

	// Perform the transactional write operation
	mut, err := s.st.TransactWriteSpannerDel(ctx, tableName, expressionAttr, e, expr, txn)
	if err != nil {
		return nil, nil, err
	}

	// Retrieve the previous response before the delete operation
	sKey := tableConf.SortKey
	pKey := tableConf.PartitionKey
	res, err := s.st.SpannerGet(ctx, tableName, attrMap[pKey], attrMap[sKey], nil)
	if err != nil {
		return nil, nil, err
	}
	return res, mut, nil
}

// TransactWriteAdd performs a transactional add operation in Spanner, ensuring old data is fetched and conditions are evaluated.
func (s *spannerService) TransactWriteAdd(ctx context.Context, tableName string, attrMap map[string]interface{}, condExpression string, m, expressionAttr map[string]interface{}, expr *models.UpdateExpressionCondition, oldRes map[string]interface{}, txn *spanner.ReadWriteTransaction) (map[string]interface{}, *spanner.Mutation, error) {
	// Fetch the table configuration to retrieve the actual table name
	tableConf, err := config.GetTableConf(tableName)
	if err != nil {
		return nil, nil, err
	}
	tableName = tableConf.ActualTable

	// Create the condition expression for the transaction
	e, err := utils.CreateConditionExpression(condExpression, expressionAttr)
	if err != nil {
		return nil, nil, err
	}

	// Perform the transactional add operation
	newResp, mut, err := s.st.TransactWriteSpannerAdd(ctx, tableName, m, e, expr, txn)
	if err != nil {
		return nil, nil, err
	}

	// If there is no old response, return early with nil mutation
	if oldRes == nil {
		return newResp, nil, nil
	}

	// Combine the old response with the new response
	updateResp := map[string]interface{}{}
	for k, v := range oldRes {
		updateResp[k] = v
	}
	for k, v := range newResp {
		updateResp[k] = v
	}

	// Return the updated response and the mutation
	return updateResp, mut, nil
}

// TransactWriteRemove performs a transactional remove operation in Spanner, ensuring old data is fetched and conditions are evaluated.
//
// It takes the following parameters:
// - ctx: the context of the request
// - tableName: the name of the table
// - updateAttr: the update attribute
// - actionValue: the action value
// - expr: the expression
// - oldRes: the old response
// - txn: the transaction
//
// It returns a map of the updated response, a mutation, and an error.
func (s *spannerService) TransactWriteRemove(ctx context.Context, tableName string, updateAttr models.UpdateAttr, actionValue string, expr *models.UpdateExpressionCondition, oldRes map[string]interface{}, txn *spanner.ReadWriteTransaction) (map[string]interface{}, *spanner.Mutation, error) {
	actionValue = strings.ReplaceAll(actionValue, " ", "")
	colsToRemove := strings.Split(actionValue, ",")
	tableConf, err := config.GetTableConf(tableName)
	if err != nil {
		return nil, nil, err
	}
	tableName = tableConf.ActualTable
	e, err := utils.CreateConditionExpression(updateAttr.ConditionExpression, updateAttr.ExpressionAttributeMap)
	if err != nil {
		return nil, nil, err
	}
	mut, err := s.st.TransactWriteSpannerRemove(ctx, tableName, updateAttr.PrimaryKeyMap, e, expr, colsToRemove, txn)
	if err != nil {
		return nil, nil, err
	}
	if oldRes == nil {
		return oldRes, nil, nil
	}
	updateResp := map[string]interface{}{}
	for k, v := range oldRes {
		updateResp[k] = v
	}

	// remove the columns from the old response
	for i := 0; i < len(colsToRemove); i++ {
		delete(updateResp, colsToRemove[i])
	}
	return updateResp, mut, nil
}

// TransactWriteDelete - This function is used to delete an item in a table.
// It takes the context of the request, the name of the table, the primary key map,
// the condition expression, the attribute map, the expression, and the transaction.
// It returns a mutation and an error.
func TransactWriteDelete(ctx context.Context, tableName string, primaryKeyMap map[string]interface{}, condExpression string, attrMap map[string]interface{}, expr *models.UpdateExpressionCondition, txn *spanner.ReadWriteTransaction) (*spanner.Mutation, error) {
	tableConf, err := config.GetTableConf(tableName)
	if err != nil {
		return nil, err
	}
	tableName = tableConf.ActualTable
	e, err := utils.CreateConditionExpression(condExpression, attrMap)
	if err != nil {
		return nil, err
	}
	// Call the storage instance to delete the item
	mut, err := storage.GetStorageInstance().TransactWriteSpannerDelete(ctx, tableName, primaryKeyMap, e, expr, txn)
	return mut, err
}
