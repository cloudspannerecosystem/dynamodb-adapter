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

package storage

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ahmetb/go-linq"
	"github.com/cloudspannerecosystem/dynamodb-adapter/config"
	"github.com/cloudspannerecosystem/dynamodb-adapter/models"
	"github.com/cloudspannerecosystem/dynamodb-adapter/pkg/errors"
	"github.com/cloudspannerecosystem/dynamodb-adapter/pkg/logger"
	"github.com/cloudspannerecosystem/dynamodb-adapter/utils"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

var base64Regexp = regexp.MustCompile("^([A-Za-z0-9+/]{4})*([A-Za-z0-9+/]{3}=|[A-Za-z0-9+/]{2}==)?$")

// SpannerBatchGet - fetch all rows
func (s Storage) SpannerBatchGet(ctx context.Context, tableName string, pKeys, sKeys []interface{}, projectionCols []string) ([]map[string]interface{}, error) {
	var keySet []spanner.KeySet

	for i := range pKeys {
		if len(sKeys) == 0 || sKeys[i] == nil {
			keySet = append(keySet, spanner.Key{pKeys[i]})
		} else {
			keySet = append(keySet, spanner.Key{pKeys[i], sKeys[i]})
		}
	}
	if len(projectionCols) == 0 {
		var ok bool
		projectionCols, ok = models.TableColumnMap[utils.ChangeTableNameForSpanner(tableName)]
		if !ok {
			return nil, errors.New("ResourceNotFoundException", tableName)
		}
	}
	colDLL, ok := models.TableDDL[utils.ChangeTableNameForSpanner(tableName)]
	if !ok {
		return nil, errors.New("ResourceNotFoundException", tableName)
	}
	tableName = utils.ChangeTableNameForSpanner(tableName)
	client := s.getSpannerClient(tableName)
	itr := client.Single().Read(ctx, tableName, spanner.KeySets(keySet...), projectionCols)
	defer itr.Stop()
	allRows := []map[string]interface{}{}
	for {
		r, err := itr.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}
			return nil, errors.New("ValidationException", err)
		}
		singleRow, err := parseRow(r, colDLL)
		if err != nil {
			return nil, err
		}
		if len(singleRow) > 0 {
			allRows = append(allRows, singleRow)
		}
	}
	return allRows, nil
}

// SpannerGet - get with spanner
func (s Storage) SpannerGet(ctx context.Context, tableName string, pKeys, sKeys interface{}, projectionCols []string) (map[string]interface{}, error) {
	var key spanner.Key
	if sKeys == nil {
		key = spanner.Key{pKeys}
	} else {
		key = spanner.Key{pKeys, sKeys}
	}
	if len(projectionCols) == 0 {
		var ok bool
		projectionCols, ok = models.TableColumnMap[utils.ChangeTableNameForSpanner(tableName)]
		if !ok {
			return nil, errors.New("ResourceNotFoundException", tableName)
		}
	}
	colDLL, ok := models.TableDDL[utils.ChangeTableNameForSpanner(tableName)]
	if !ok {
		return nil, errors.New("ResourceNotFoundException", tableName)
	}
	tableName = utils.ChangeTableNameForSpanner(tableName)
	client := s.getSpannerClient(tableName)
	row, err := client.Single().ReadRow(ctx, tableName, key, projectionCols)
	if err := errors.AssignError(err); err != nil {
		return nil, errors.New("ResourceNotFoundException", tableName, key, err)
	}

	return parseRow(row, colDLL)
}

// ExecuteSpannerQuery - this will execute query on spanner database
func (s Storage) ExecuteSpannerQuery(ctx context.Context, table string, cols []string, isCountQuery bool, stmt spanner.Statement) ([]map[string]interface{}, error) {

	colDLL, ok := models.TableDDL[utils.ChangeTableNameForSpanner(table)]

	if !ok {
		return nil, errors.New("ResourceNotFoundException", table)
	}

	itr := s.getSpannerClient(table).Single().WithTimestampBound(spanner.ExactStaleness(time.Second*10)).Query(ctx, stmt)

	defer itr.Stop()
	allRows := []map[string]interface{}{}
	for {
		r, err := itr.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, errors.New("ResourceNotFoundException", err)
		}
		if isCountQuery {
			var count int64
			err := r.ColumnByName("count", &count)
			if err != nil {
				return nil, err
			}
			singleRow := map[string]interface{}{"Count": count, "Items": []map[string]interface{}{}, "LastEvaluatedKey": nil}
			allRows = append(allRows, singleRow)
			break
		}
		singleRow, err := parseRow(r, colDLL)
		if err != nil {
			return nil, err
		}
		allRows = append(allRows, singleRow)
	}

	return allRows, nil
}

// SpannerPut - Spanner put insert a single object
func (s Storage) SpannerPut(ctx context.Context, table string, m map[string]interface{}, eval *models.Eval, expr *models.UpdateExpressionCondition) (map[string]interface{}, error) {
	update := map[string]interface{}{}
	_, err := s.getSpannerClient(table).ReadWriteTransaction(ctx, func(ctx context.Context, t *spanner.ReadWriteTransaction) error {
		tmpMap := map[string]interface{}{}
		for k, v := range m {
			switch v := v.(type) {
			case []interface{}:
				// Serialize lists to JSON
				jsonValue, err := json.Marshal(v)
				if err != nil {
					return fmt.Errorf("failed to serialize column %s to JSON: %v", k, err)
				}
				tmpMap[k] = string(jsonValue)
			default:
				// Assign other types as-is
				tmpMap[k] = v
			}
		}
		if len(eval.Attributes) > 0 || expr != nil {
			status, err := evaluateConditionalExpression(ctx, t, table, tmpMap, eval, expr)
			if err != nil {
				return err
			}
			if !status {
				return errors.New("ConditionalCheckFailedException", eval, expr)
			}
		}
		table = utils.ChangeTableNameForSpanner(table)
		for k, v := range tmpMap {
			update[k] = v
		}
		return s.performPutOperation(ctx, t, table, tmpMap)
	})
	return update, err
}

// SpannerDelete - this will delete the data
func (s Storage) SpannerDelete(ctx context.Context, table string, m map[string]interface{}, eval *models.Eval, expr *models.UpdateExpressionCondition) error {
	_, err := s.getSpannerClient(table).ReadWriteTransaction(ctx, func(ctx context.Context, t *spanner.ReadWriteTransaction) error {
		tmpMap := map[string]interface{}{}
		for k, v := range m {
			tmpMap[k] = v
		}
		if len(eval.Attributes) > 0 || expr != nil {
			status, err := evaluateConditionalExpression(ctx, t, table, tmpMap, eval, expr)
			if err != nil {
				return err
			}
			if !status {
				return errors.New("ConditionalCheckFailedException", tmpMap, expr)
			}
		}
		tableConf, err := config.GetTableConf(table)
		if err != nil {
			return err
		}
		table = utils.ChangeTableNameForSpanner(table)

		pKey := tableConf.PartitionKey
		pValue, ok := tmpMap[pKey]
		if !ok {
			return errors.New("ResourceNotFoundException", pKey)
		}
		var key spanner.Key
		sKey := tableConf.SortKey
		if sKey != "" {
			sValue, ok := tmpMap[sKey]
			if !ok {
				return errors.New("ResourceNotFoundException", pKey)
			}
			key = spanner.Key{pValue, sValue}

		} else {
			key = spanner.Key{pValue}
		}

		mutation := spanner.Delete(table, key)
		err = t.BufferWrite([]*spanner.Mutation{mutation})
		if e := errors.AssignError(err); e != nil {
			return e
		}
		return nil
	})
	return err
}

// SpannerBatchDelete - this delete the data in batch
func (s Storage) SpannerBatchDelete(ctx context.Context, table string, keys []map[string]interface{}) error {
	tableConf, err := config.GetTableConf(table)
	if err != nil {
		return err
	}
	table = utils.ChangeTableNameForSpanner(table)

	pKey := tableConf.PartitionKey
	ms := make([]*spanner.Mutation, len(keys))
	sKey := tableConf.SortKey
	for i := 0; i < len(keys); i++ {
		m := keys[i]
		pValue, ok := m[pKey]
		if !ok {
			return errors.New("ResourceNotFoundException", pKey)
		}
		var key spanner.Key
		if sKey != "" {
			sValue, ok := m[sKey]
			if !ok {
				return errors.New("ResourceNotFoundException", sKey)
			}
			key = spanner.Key{pValue, sValue}

		} else {
			key = spanner.Key{pValue}
		}
		ms[i] = spanner.Delete(table, key)
	}
	_, err = s.getSpannerClient(table).Apply(ctx, ms)
	if err != nil {
		return errors.New("ResourceNotFoundException", err)
	}
	return nil
}

// SpannerAdd - Spanner Add functionality like update attribute
func (s Storage) SpannerAdd(ctx context.Context, table string, m map[string]interface{}, eval *models.Eval, expr *models.UpdateExpressionCondition) (map[string]interface{}, error) {
	tableConf, err := config.GetTableConf(table)
	if err != nil {
		return nil, err
	}
	colDLL, ok := models.TableDDL[utils.ChangeTableNameForSpanner(table)]
	if !ok {
		return nil, errors.New("ResourceNotFoundException", table)
	}
	pKey := tableConf.PartitionKey
	var pValue interface{}
	var sValue interface{}
	sKey := tableConf.SortKey

	cols := []string{}
	var key spanner.Key
	var m1 = make(map[string]interface{})

	for k, v := range m {
		m1[k] = v
		if k == pKey {
			pValue = v
			delete(m, k)
			continue
		}
		if k == sKey {
			delete(m, k)
			sValue = v
			continue
		}
		cols = append(cols, k)
	}
	if sValue != nil {
		key = spanner.Key{pValue, sValue}
	} else {
		key = spanner.Key{pValue}
	}

	updatedObj := map[string]interface{}{}
	_, err = s.getSpannerClient(table).ReadWriteTransaction(ctx, func(ctx context.Context, t *spanner.ReadWriteTransaction) error {
		tmpMap := map[string]interface{}{}
		for k, v := range m1 {
			tmpMap[k] = v
		}

		if len(eval.Attributes) > 0 || expr != nil {
			status, _ := evaluateConditionalExpression(ctx, t, table, tmpMap, eval, expr)
			if !status {
				return errors.New("ConditionalCheckFailedException")
			}
		}
		table = utils.ChangeTableNameForSpanner(table)

		r, err := t.ReadRow(ctx, table, key, cols)
		if err != nil {
			return errors.New("ResourceNotFoundException", err)
		}
		rs, err := parseRow(r, colDLL)
		if err != nil {
			return err
		}

		for k, v := range tmpMap {
			if existingVal, ok := rs[k]; ok {
				switch existingVal := existingVal.(type) {
				case int64:
					// Handling int64
					v2, ok := v.(float64)
					if !ok {
						strV, ok := v.(string)
						if !ok {
							return errors.New("ValidationException", reflect.TypeOf(v).String())
						}
						v2, err = strconv.ParseFloat(strV, 64)
						if err != nil {
							return errors.New("ValidationException", reflect.TypeOf(v).String())
						}
					}
					tmpMap[k] = existingVal + int64(v2)

				case float64:
					// Handling float64
					v2, ok := v.(float64)
					if !ok {
						strV, ok := v.(string)
						if !ok {
							return errors.New("ValidationException", reflect.TypeOf(v).String())
						}
						v2, err = strconv.ParseFloat(strV, 64)
						if err != nil {
							return errors.New("ValidationException", reflect.TypeOf(v).String())
						}
					}
					tmpMap[k] = existingVal + v2

				default:
					logger.LogDebug(reflect.TypeOf(v).String())
				}
			}
		}

		// Add partition and sort keys to the updated object
		tmpMap[pKey] = pValue
		if sValue != nil {
			tmpMap[sKey] = sValue
		}

		ddl := models.TableDDL[table]
		for k, v := range tmpMap {
			updatedObj[k] = v
			t, ok := ddl[k]
			if t == "BYTES(MAX)" && ok {
				ba, err := json.Marshal(v)
				if err != nil {
					return errors.New("ValidationException", err)
				}
				tmpMap[k] = ba
			}
			switch v := v.(type) {
			case []interface{}:
				// Serialize lists to JSON
				jsonValue, err := json.Marshal(v)
				if err != nil {
					return fmt.Errorf("failed to serialize column %s to JSON: %v", k, err)
				}
				tmpMap[k] = string(jsonValue)
			default:
				// Assign other types as-is
				tmpMap[k] = v
			}
		}

		mutation := spanner.InsertOrUpdateMap(table, tmpMap)
		err = t.BufferWrite([]*spanner.Mutation{mutation})
		if err != nil {
			return errors.New("ResourceNotFoundException", err)
		}

		return nil
	})

	return updatedObj, err
}

func (s Storage) SpannerDel(ctx context.Context, table string, m map[string]interface{}, eval *models.Eval, expr *models.UpdateExpressionCondition) error {
	tableConf, err := config.GetTableConf(table)
	if err != nil {
		return err
	}
	colDLL, ok := models.TableDDL[utils.ChangeTableNameForSpanner(table)]
	if !ok {
		return errors.New("ResourceNotFoundException", table)
	}
	pKey := tableConf.PartitionKey
	var pValue interface{}
	var sValue interface{}
	sKey := tableConf.SortKey

	cols := []string{}
	var key spanner.Key
	var m1 = make(map[string]interface{})

	// Process primary and secondary keys
	for k, v := range m {
		m1[k] = v
		if k == pKey {
			pValue = v
			delete(m, k)
			continue
		}
		if k == sKey {
			delete(m, k)
			sValue = v
			continue
		}
		cols = append(cols, k)
	}
	if sValue != nil {
		key = spanner.Key{pValue, sValue}
	} else {
		key = spanner.Key{pValue}
	}

	_, err = s.getSpannerClient(table).ReadWriteTransaction(ctx, func(ctx context.Context, t *spanner.ReadWriteTransaction) error {
		tmpMap := map[string]interface{}{}
		for k, v := range m {
			tmpMap[k] = v
		}

		// Evaluate conditional expressions
		if len(eval.Attributes) > 0 || expr != nil {
			status, _ := evaluateConditionalExpression(ctx, t, table, m1, eval, expr)
			if !status {
				return errors.New("ConditionalCheckFailedException")
			}
		}

		table = utils.ChangeTableNameForSpanner(table)

		// Read the row
		r, err := t.ReadRow(ctx, table, key, cols)
		if err != nil {
			return errors.New("ResourceNotFoundException", err)
		}
		rs, err := parseRow(r, colDLL)
		if err != nil {
			return err
		}

		// Process and merge data for deletion
		for k, v := range tmpMap {
			v1, ok := rs[k]
			if ok {
				switch v1.(type) {
				case []interface{}:
					var ifaces1 []interface{}
					ba, ok := v.([]byte)
					if ok {
						err = json.Unmarshal(ba, &ifaces1)
						if err != nil {
							logger.LogError(err, string(ba))
						}
					} else {
						ifaces1 = v.([]interface{})
					}
					m1 := map[interface{}]struct{}{}
					ifaces := v1.([]interface{})
					for i := 0; i < len(ifaces); i++ {
						m1[reflect.ValueOf(ifaces[i]).Interface()] = struct{}{}
					}
					for i := 0; i < len(ifaces1); i++ {

						delete(m1, reflect.ValueOf(ifaces1[i]).Interface())
					}
					ifaces = []interface{}{}
					for k := range m1 {
						ifaces = append(ifaces, k)
					}
					tmpMap[k] = ifaces
				default:
					logger.LogDebug(reflect.TypeOf(v).String())
				}
			}
		}
		tmpMap[pKey] = pValue
		if sValue != nil {
			tmpMap[sKey] = sValue
		}

		ddl := models.TableDDL[table]

		// Handle special cases like BYTES(MAX) columns
		for k, v := range tmpMap {
			t, ok := ddl[k]
			if t == "BYTES(MAX)" && ok {
				ba, err := json.Marshal(v)
				if err != nil {
					return errors.New("ValidationException", err)
				}
				tmpMap[k] = ba
			}
		}

		// Perform the delete operation by updating the row
		mutation := spanner.InsertOrUpdateMap(table, tmpMap)
		err = t.BufferWrite([]*spanner.Mutation{mutation})
		if err != nil {
			return errors.New("ResourceNotFoundException", err)
		}
		return nil
	})
	return err
}

// SpannerRemove - Spanner Remove functionality like update attribute
func (s Storage) SpannerRemove(ctx context.Context, table string, m map[string]interface{}, eval *models.Eval, expr *models.UpdateExpressionCondition, colsToRemove []string, oldRes map[string]interface{}) error {
	_, err := s.getSpannerClient(table).ReadWriteTransaction(ctx, func(ctx context.Context, t *spanner.ReadWriteTransaction) error {
		tmpMap := map[string]interface{}{}
		for k, v := range m {
			tmpMap[k] = v
		}
		if len(eval.Attributes) > 0 || expr != nil {
			status, _ := evaluateConditionalExpression(ctx, t, table, m, eval, expr)
			if !status {
				return errors.New("ConditionalCheckFailedException")
			}
		}

		// Process each removal target
		for _, target := range colsToRemove {
			if strings.Contains(target, "[") && strings.Contains(target, "]") {
				// Handle list element removal
				listAttr, idx := parseListRemoveTarget(target)

				if val, ok := oldRes[listAttr]; ok {
					if list, ok := val.([]interface{}); ok {
						oldList := list
						oldRes[listAttr] = removeListElement(list, idx)
						tmpMap[listAttr] = oldList
					}
				}
			} else if strings.Contains(target, ".") {
				// Handle map key removal
				mapAttr, key := parseMapRemoveTarget(target)
				if val, ok := oldRes[mapAttr]; ok {
					if m, ok := val.(map[string]interface{}); ok {
						oldMap := m
						delete(m, key)
						oldRes[mapAttr] = m
						tmpMap[mapAttr] = oldMap
					}
				}
			} else {
				// Direct column removal from oldRes
				delete(oldRes, target)
			}
		}
		// Handle special cases like BYTES(MAX) columns
		for k, v := range tmpMap {
			switch v := v.(type) {
			case []interface{}:
				// Serialize lists to JSON
				jsonValue, err := json.Marshal(v)
				if err != nil {
					return fmt.Errorf("failed to serialize column %s to JSON: %v", k, err)
				}
				tmpMap[k] = string(jsonValue)
			default:
				// Assign other types as-is
				tmpMap[k] = v
			}
		}

		table = utils.ChangeTableNameForSpanner(table)
		mutation := spanner.InsertOrUpdateMap(table, tmpMap)
		err := t.BufferWrite([]*spanner.Mutation{mutation})
		if err != nil {
			return errors.New("ResourceNotFoundException", err)
		}
		return nil
	})
	return err
}

func parseListRemoveTarget(target string) (string, int) {
	// Example: listAttr[2]
	re := regexp.MustCompile(`(.*)\[(\d+)\]`)
	matches := re.FindStringSubmatch(target)
	if len(matches) == 3 {
		index, _ := strconv.Atoi(matches[2])
		return matches[1], index
	}
	return target, -1
}

func removeListElement(list []interface{}, idx int) []interface{} {
	if idx < 0 || idx >= len(list) {
		return list // Return original list for invalid indices
	}
	return append(list[:idx], list[idx+1:]...)
}

func parseMapRemoveTarget(target string) (string, string) {
	// Example: mapAttr.key
	parts := strings.SplitN(target, ".", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return target, ""
}

// SpannerBatchPut - this insert or update data in batch
func (s Storage) SpannerBatchPut(ctx context.Context, table string, m []map[string]interface{}) error {
	mutations := make([]*spanner.Mutation, len(m))
	ddl := models.TableDDL[utils.ChangeTableNameForSpanner(table)]
	table = utils.ChangeTableNameForSpanner(table)
	for i := 0; i < len(m); i++ {
		for k, v := range m[i] {
			t, ok := ddl[k]
			if t == "BYTES(MAX)" && ok {
				ba, err := json.Marshal(v)
				if err != nil {
					return errors.New("ValidationException", err)
				}
				m[i][k] = ba
			}
			switch v := v.(type) {
			case []interface{}:
				// Serialize lists to JSON
				jsonValue, err := json.Marshal(v)
				if err != nil {
					return fmt.Errorf("failed to serialize column %s to JSON: %v", k, err)
				}
				m[i][k] = string(jsonValue)
			default:
				// Assign other types as-is
				m[i][k] = v
			}
		}
		mutations[i] = spanner.InsertOrUpdateMap(table, m[i])
	}
	_, err := s.getSpannerClient(table).Apply(ctx, mutations)
	if err != nil {
		return errors.New("ResourceNotFoundException", err.Error())
	}
	return nil
}

func (s Storage) performPutOperation(ctx context.Context, t *spanner.ReadWriteTransaction, table string, m map[string]interface{}) error {
	ddl := models.TableDDL[table]
	for k, v := range m {
		t, ok := ddl[k]
		if t == "BYTES(MAX)" && ok {
			ba, err := json.Marshal(v)
			if err != nil {
				return errors.New("ValidationException", err)
			}
			m[k] = ba
		}
	}

	mutation := spanner.InsertOrUpdateMap(table, m)

	mutations := []*spanner.Mutation{mutation}

	err := t.BufferWrite(mutations)
	if e := errors.AssignError(err); e != nil {
		return e
	}
	return nil
}

func evaluateConditionalExpression(ctx context.Context, t *spanner.ReadWriteTransaction, table string, m map[string]interface{}, e *models.Eval, expr *models.UpdateExpressionCondition) (bool, error) {
	colDDL, ok := models.TableDDL[utils.ChangeTableNameForSpanner(table)]
	if !ok {
		return false, errors.New("ResourceNotFoundException", table)
	}
	tableConf, err := config.GetTableConf(table)
	if err != nil {
		return false, err
	}

	pKey := tableConf.PartitionKey
	pValue, ok := m[pKey]
	if !ok {
		return false, errors.New("ValidationException", pKey)
	}
	var key spanner.Key
	sKey := tableConf.SortKey
	if sKey != "" {
		sValue, ok := m[sKey]
		if !ok {
			return false, errors.New("ValidationException", sKey)
		}
		key = spanner.Key{pValue, sValue}

	} else {
		key = spanner.Key{pValue}
	}
	var cols []string
	if expr != nil {
		cols = append(e.Cols, expr.Field...)
		for k := range expr.AddValues {
			cols = append(e.Cols, k)
		}
	} else {
		cols = e.Cols
	}

	linq.From(cols).IntersectByT(linq.From(models.TableColumnMap[utils.ChangeTableNameForSpanner(table)]), func(str string) string {
		return str
	}).ToSlice(&cols)
	r, err := t.ReadRow(ctx, utils.ChangeTableNameForSpanner(table), key, cols)
	if e := errors.AssignError(err); e != nil {
		return false, e
	}
	rowMap, err := parseRow(r, colDDL)
	if err != nil {
		return false, err
	}
	if expr != nil {
		for index := 0; index < len(expr.Field); index++ {
			colName := expr.Field[index]
			if strings.HasPrefix(colName, "size(") {
				// Extract attribute name from size function
				sizeRegex := regexp.MustCompile(`size\((\w+)\)`)
				matches := sizeRegex.FindStringSubmatch(colName)
				if len(matches) == 2 {
					colName = matches[1] // Extracted column name
				}
			}
			status := evaluateStatementFromRowMap(expr.Condition[index], colName, rowMap)
			tmp, ok := status.(bool)
			if !ok || !tmp {
				if v1, ok := expr.AddValues[expr.Field[index]]; ok {

					tmp, ok := rowMap[expr.Field[index]].(float64)
					if ok {
						m[expr.Field[index]] = tmp + v1
						err = checkInifinty(m[expr.Field[index]].(float64), expr)
						if err != nil {
							return false, err
						}
					}
				} else {
					delete(m, expr.Field[index])
				}
			} else {
				if v1, ok := expr.AddValues[expr.Field[index]]; ok {
					tmp, ok := m[expr.Field[index]].(float64)
					if ok {
						m[expr.Field[index]] = tmp + v1
						err = checkInifinty(m[expr.Field[index]].(float64), expr)
						if err != nil {
							return false, err
						}
					}
				}
			}
			delete(expr.AddValues, expr.Field[index])
		}
		for k, v := range expr.AddValues {
			val, ok := rowMap[k].(float64)
			if ok {
				m[k] = val + v
				err = checkInifinty(m[k].(float64), expr)
				if err != nil {
					return false, err
				}

			} else {
				m[k] = v
			}
		}
	}
	for i := 0; i < len(e.Attributes); i++ {
		e.ValueMap[e.Tokens[i]] = evaluateStatementFromRowMap(e.Attributes[i], e.Cols[i], rowMap)
	}

	status, err := utils.EvaluateExpression(e)
	if err != nil {
		return false, err
	}

	return status, nil
}

func evaluateStatementFromRowMap(conditionalExpression, colName string, rowMap map[string]interface{}) interface{} {
	if strings.HasPrefix(conditionalExpression, "attribute_not_exists") || strings.HasPrefix(conditionalExpression, "if_not_exists") {
		if len(rowMap) == 0 {
			return true
		}
		_, ok := rowMap[colName]
		return !ok
	}
	if strings.HasPrefix(conditionalExpression, "attribute_exists") || strings.HasPrefix(conditionalExpression, "if_exists") {
		if len(rowMap) == 0 {
			return false
		}
		_, ok := rowMap[colName]
		return ok
	}
	// Handle size() function
	if strings.HasPrefix(conditionalExpression, "size(") {
		sizeRegex := regexp.MustCompile(`size\((\w+)\)`)
		matches := sizeRegex.FindStringSubmatch(conditionalExpression)
		if len(matches) == 2 {
			attributeName := matches[1]

			// Check if the attribute exists in rowMap
			val, ok := rowMap[attributeName]
			if !ok {
				return errors.New("Attribute not found in row")
			}

			// Ensure the attribute is a list and calculate its size
			switch v := val.(type) {
			case []interface{}:
				return len(v) // Return the size of the list
			default:
				return errors.New("size() function is only valid for list attributes")
			}
		} else {
			return errors.New("Invalid size() function syntax")
		}
	}
	return rowMap[conditionalExpression]
}

// parseRow - Converts Spanner row and datatypes to a map removing null columns from the result.
func parseRow(r *spanner.Row, colDDL map[string]string) (map[string]interface{}, error) {
	singleRow := make(map[string]interface{})
	if r == nil {
		return singleRow, nil
	}

	cols := r.ColumnNames()
	for i, k := range cols {
		if k == "" || k == "commit_timestamp" {
			continue
		}
		v, ok := colDDL[k]
		if !ok {
			return nil, errors.New("ResourceNotFoundException", k)
		}
		switch v {
		case "S":
			var s spanner.NullString
			err := r.Column(i, &s)
			if err != nil {
				if strings.Contains(err.Error(), "ambiguous column name") {
					continue
				}
				return nil, errors.New("ValidationException", err, k)
			}
			if !s.IsNull() {
				singleRow[k] = s.StringVal
			}
		case "B":
			var s []byte
			err := r.Column(i, &s)
			if err != nil {
				if strings.Contains(err.Error(), "ambiguous column name") {
					continue
				}
				return nil, errors.New("ValidationException", err, k)
			}
			if len(s) > 0 {
				var m interface{}
				err := json.Unmarshal(s, &m)
				if err != nil {
					logger.LogError(err, string(s))
					singleRow[k] = string(s)
					continue
				}
				val1, ok := m.(string)
				if ok {
					if base64Regexp.MatchString(val1) {
						ba, err := base64.StdEncoding.DecodeString(val1)
						if err == nil {
							var sample interface{}
							err = json.Unmarshal(ba, &sample)
							if err == nil {
								singleRow[k] = sample
								continue
							} else {
								singleRow[k] = string(s)
								continue
							}
						}
					}
				}

				if mp, ok := m.(map[string]interface{}); ok {
					for k, v := range mp {
						if val, ok := v.(string); ok {
							if base64Regexp.MatchString(val) {
								ba, err := base64.StdEncoding.DecodeString(val)
								if err == nil {
									var sample interface{}
									err = json.Unmarshal(ba, &sample)
									if err == nil {
										mp[k] = sample
										m = mp
									}
								}
							}
						}
					}
				}
				singleRow[k] = m
			}
		case "N":
			var s spanner.NullFloat64
			err := r.Column(i, &s)
			if err != nil {
				if strings.Contains(err.Error(), "ambiguous column name") {
					continue
				}
				return nil, errors.New("ValidationException", err, k)

			}
			if !s.IsNull() {
				singleRow[k] = s.Float64
			}
		case "NUmeric":
			var s spanner.NullNumeric
			err := r.Column(i, &s)
			if err != nil {
				if strings.Contains(err.Error(), "ambiguous column name") {
					continue
				}
				return nil, errors.New("ValidationException", err, k)
			}
			if !s.IsNull() {
				if s.Numeric.IsInt() {
					tmp, _ := s.Numeric.Float64()
					singleRow[k] = int64(tmp)
				} else {
					singleRow[k], _ = s.Numeric.Float64()
				}
			}
		case "BOOL":
			var s spanner.NullBool
			err := r.Column(i, &s)
			if err != nil {
				if strings.Contains(err.Error(), "ambiguous column name") {
					continue
				}
				return nil, errors.New("ValidationException", err, k)

			}
			if !s.IsNull() {
				singleRow[k] = s.Bool
			}
		case "L":
			var jsonValue spanner.NullJSON
			err := r.Column(i, &jsonValue)
			if err != nil {
				if strings.Contains(err.Error(), "ambiguous column name") {
					continue
				}
				return nil, errors.New("ValidationException", err, k)
			}
			if !jsonValue.IsNull() {
				parsed := parseDynamoDBJSON(jsonValue.Value)
				singleRow[k] = parsed
			}

		}
	}
	return singleRow, nil
}

func parseDynamoDBJSON(value interface{}) interface{} {
	if value == nil {
		return nil
	}

	switch v := value.(type) {
	case map[string]interface{}:
		for key, val := range v {
			switch key {
			case "S": // String
				return val.(string)
			case "N": // Number
				num, _ := strconv.ParseFloat(val.(string), 64)
				return num
			case "BOOL": // Boolean
				return val.(bool)
			case "M": // Map (nested object)
				result := make(map[string]interface{})
				for k, nestedVal := range val.(map[string]interface{}) {
					result[k] = parseDynamoDBJSON(nestedVal)
				}
				return result
			case "L": // List
				list := val.([]interface{})
				result := make([]interface{}, len(list))
				for i, item := range list {
					result[i] = parseDynamoDBJSON(item) // Recursively parse each list item
				}
				return result
			}
		}
	case []interface{}: // Handle direct list structures
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = parseDynamoDBJSON(item)
		}
		return result
	}

	return value // Return as-is for unsupported types
}

func checkInifinty(value float64, logData interface{}) error {
	if math.IsInf(value, 1) {
		return errors.New("ValidationException", "value found is infinity", logData)
	}
	if math.IsInf(value, -1) {
		return errors.New("ValidationException", "value found is infinity", logData)
	}

	return nil
}
