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
	"math"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/cloudspannerecosystem/dynamodb-adapter/config"
	"github.com/cloudspannerecosystem/dynamodb-adapter/models"
	"github.com/cloudspannerecosystem/dynamodb-adapter/pkg/errors"
	"github.com/cloudspannerecosystem/dynamodb-adapter/pkg/logger"
	"github.com/cloudspannerecosystem/dynamodb-adapter/utils"

	"cloud.google.com/go/spanner"
	"github.com/ahmetb/go-linq"
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
			tmpMap[k] = v
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
			v1, ok := rs[k]
			if ok {
				switch v1.(type) {
				case int64:
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
						err = checkInifinty(v2, strV)
						if err != nil {
							return err
						}
					}
					tmpMap[k] = v1.(int64) + int64(v2)
					err = checkInifinty(float64(m[k].(int64)), m)
					if err != nil {
						return err
					}
				case float64:
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
						err = checkInifinty(v2, strV)
						if err != nil {
							return err
						}
					}
					tmpMap[k] = v1.(float64) + v2
					err = checkInifinty(m[k].(float64), m)
					if err != nil {
						return err
					}

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
						m1[ifaces[i]] = struct{}{}
					}
					for i := 0; i < len(ifaces1); i++ {
						m1[ifaces1[i]] = struct{}{}
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

// SpannerDel for delete operation on Spanner
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
		if len(eval.Attributes) > 0 || expr != nil {
			status, _ := evaluateConditionalExpression(ctx, t, table, m1, eval, expr)
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
func (s Storage) SpannerRemove(ctx context.Context, table string, m map[string]interface{}, eval *models.Eval, expr *models.UpdateExpressionCondition, colsToRemove []string) error {

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
		var null spanner.NullableValue
		for _, col := range colsToRemove {
			tmpMap[col] = null
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
			status := evaluateStatementFromRowMap(expr.Condition[index], expr.Field[index], rowMap)
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

		var err error
		switch v {
		case "S":
			err = parseStringColumn(r, i, k, singleRow)
		case "B":
			err = parseBytesColumn(r, i, k, singleRow)
		case "N":
			err = parseNumericColumn(r, i, k, singleRow)
		case "BOOL":
			err = parseBoolColumn(r, i, k, singleRow)
		case "SS":
			err = parseStringArrayColumn(r, i, k, singleRow)
		case "BS":
			err = parseByteArrayColumn(r, i, k, singleRow)
		case "NS":
			err = parseNumberArrayColumn(r, i, k, singleRow)
		default:
			return nil, errors.New("TypeNotFound", err, k)
		}

		if err != nil {
			return nil, errors.New("ValidationException", err, k)
		}
	}
	return singleRow, nil
}

func parseStringColumn(r *spanner.Row, idx int, col string, row map[string]interface{}) error {
	var s spanner.NullString
	err := r.Column(idx, &s)
	if err != nil && !strings.Contains(err.Error(), "ambiguous column name") {
		return err
	}
	if !s.IsNull() {
		row[col] = s.StringVal
	}
	return nil
}

func parseBytesColumn(r *spanner.Row, idx int, col string, row map[string]interface{}) error {
	var s []byte
	err := r.Column(idx, &s)
	if err != nil && !strings.Contains(err.Error(), "ambiguous column name") {
		return err
	}

	if len(s) > 0 {
		var m interface{}
		if err := json.Unmarshal(s, &m); err != nil {
			// Instead of an error while unmarshalling fall back to the raw string.
			row[col] = string(s)
			return nil
		}
		m = processDecodedData(m)
		row[col] = m
	}
	return nil
}

func parseNumericColumn(r *spanner.Row, idx int, col string, row map[string]interface{}) error {
	var s spanner.NullFloat64
	err := r.Column(idx, &s)
	if err != nil && !strings.Contains(err.Error(), "ambiguous column name") {
		return err
	}
	if !s.IsNull() {
		row[col] = s.Float64
	}
	return nil
}

func parseBoolColumn(r *spanner.Row, idx int, col string, row map[string]interface{}) error {
	var s spanner.NullBool
	err := r.Column(idx, &s)
	if err != nil && !strings.Contains(err.Error(), "ambiguous column name") {
		return err
	}
	if !s.IsNull() {
		row[col] = s.Bool
	}
	return nil
}

func parseStringArrayColumn(r *spanner.Row, idx int, col string, row map[string]interface{}) error {
	var s []spanner.NullString
	err := r.Column(idx, &s)
	if err != nil && !strings.Contains(err.Error(), "ambiguous column name") {
		return err
	}
	var temp []string
	for _, val := range s {
		temp = append(temp, val.StringVal)
	}
	if len(s) > 0 {
		row[col] = temp
	}
	return nil
}

func parseByteArrayColumn(r *spanner.Row, idx int, col string, row map[string]interface{}) error {
	var b [][]byte
	err := r.Column(idx, &b)
	if err != nil && !strings.Contains(err.Error(), "ambiguous column name") {
		return err
	}
	if len(b) > 0 {
		row[col] = b
	}
	return nil
}

func parseNumberArrayColumn(r *spanner.Row, idx int, col string, row map[string]interface{}) error {
	var nums []spanner.NullFloat64
	err := r.Column(idx, &nums)
	if err != nil && !strings.Contains(err.Error(), "ambiguous column name") {
		return err
	}
	var temp []float64
	for _, val := range nums {
		if val.Valid {
			temp = append(temp, val.Float64)
		}
	}
	if len(nums) > 0 {
		row[col] = temp
	}
	return nil
}

func processDecodedData(m interface{}) interface{} {
	if val, ok := m.(string); ok && base64Regexp.MatchString(val) {
		if ba, err := base64.StdEncoding.DecodeString(val); err == nil {
			var sample interface{}
			if err := json.Unmarshal(ba, &sample); err == nil {
				return sample
			}
		}
	}
	if mp, ok := m.(map[string]interface{}); ok {
		for k, v := range mp {
			if val, ok := v.(string); ok && base64Regexp.MatchString(val) {
				if ba, err := base64.StdEncoding.DecodeString(val); err == nil {
					var sample interface{}
					if err := json.Unmarshal(ba, &sample); err == nil {
						mp[k] = sample
					}
				}
			}
		}
	}
	return m
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
