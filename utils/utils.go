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

package utils

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/antonmedv/expr"
	"github.com/cloudspannerecosystem/dynamodb-adapter/models"
	"github.com/cloudspannerecosystem/dynamodb-adapter/pkg/errors"
)

// GetFieldNameFromConditionalExpression returns the field name from conditional expression
func GetFieldNameFromConditionalExpression(conditionalExpression string) string {
	if strings.Contains(conditionalExpression, "attribute_exists") {
		return GetStringInBetween(conditionalExpression, "(", ")")
	}
	if strings.Contains(conditionalExpression, "attribute_not_exists") {
		return GetStringInBetween(conditionalExpression, "(", ")")
	}
	return conditionalExpression
}

// GetStringInBetween Returns empty string if no start string found
func GetStringInBetween(str string, start string, end string) (result string) {
	s := strings.Index(str, start)
	if s == -1 {
		return
	}
	s += len(start)
	e := strings.Index(str, end)
	if s >= e {
		return ""
	}
	return str[s:e]
}

// CreateConditionExpression - create evelute condition from condition
func CreateConditionExpression(condtionExpression string, expressionAttr map[string]interface{}) (*models.Eval, error) {
	if condtionExpression == "" {
		e := new(models.Eval)
		return e, nil
	}
	condtionExpression = strings.TrimSpace(condtionExpression)
	condtionExpression = strings.ReplaceAll(condtionExpression, "( ", "(")
	condtionExpression = strings.ReplaceAll(condtionExpression, " )", ")")
	tokens := strings.Split(condtionExpression, " ")
	sb := strings.Builder{}
	evalTokens := []string{}
	cols := []string{}
	ts := []string{}
	var err error
	for i := 0; i < len(tokens); i++ {
		if i%2 == 0 {
			if strings.Contains(tokens[i], ":") {
				v, ok := expressionAttr[tokens[i]]
				if !ok {
					return nil, errors.New("ResourceNotFoundException", expressionAttr, tokens[i])
				}
				str := fmt.Sprint(v)
				_, ok = v.(string)
				if ok {
					str = "\"" + str + "\""
				}
				switch v.(type) {
				case float64:
					str = fmt.Sprintf("%f", v)
				case int64:
					str = fmt.Sprintf("%d", v)
				}
				sb.WriteString(str)
				sb.WriteString(" ")
				continue
			}
			t := "TOKEN" + strconv.Itoa(i)
			col := GetFieldNameFromConditionalExpression(tokens[i])
			sb.WriteString(t)
			sb.WriteString(" ")
			evalTokens = append(evalTokens, tokens[i])
			cols = append(cols, col)
			ts = append(ts, t)
		} else {
			sb.WriteString(tokens[i])
			sb.WriteString(" ")
		}
	}
	e := new(models.Eval)
	str := sb.String()
	str = strings.ReplaceAll(str, " = ", " == ")
	str = strings.ReplaceAll(str, " OR ", " || ")
	str = strings.ReplaceAll(str, " or ", " || ")
	str = strings.ReplaceAll(str, " and ", " && ")
	str = strings.ReplaceAll(str, " AND ", " && ")
	str = strings.ReplaceAll(str, " <> ", " != ")

	e.Cond, err = expr.Compile(str)
	if err != nil {
		return nil, errors.New("ConditionalCheckFailedException", err.Error(), str)
	}
	e.Attributes = evalTokens
	e.Cols = cols
	e.Tokens = ts
	e.ValueMap = make(map[string]interface{}, len(evalTokens))
	return e, nil
}

// EvaluateExpression - evalute expression
func EvaluateExpression(expression *models.Eval) (bool, error) {
	if expression == nil || expression.Cond == nil {
		return true, nil
	}
	if expression.ValueMap == nil {
		return false, nil
	}

	val, err := expr.Run(expression.Cond, expression.ValueMap)
	if err != nil {
		return false, errors.New("ConditionalCheckFailedException", err.Error())
	}
	status, ok := val.(bool)
	if !status || !ok {
		return false, errors.New("ConditionalCheckFailedException")
	}
	return status, nil
}

var replaceMap = map[string]string{"EQ": "=", "LT": "<", "GT": ">", "LE": "<=", "GE": ">="}

// ParseBeginsWith ..
func ParseBeginsWith(rangeExpression string) (string, string, string) {
	index := strings.Index(rangeExpression, "begins_with")
	if index > -1 {
		start := -1
		end := -1
		for i := index; i < len(rangeExpression); i++ {
			if rangeExpression[i] == '(' && start == -1 {
				start = i
			}
			if rangeExpression[i] == ')' && end == -1 {
				end = i
				break
			}
		}
		bracketValue := rangeExpression[start+1 : end]
		tokens := strings.Split(bracketValue, ",")
		return strings.TrimSpace(tokens[0]), strings.TrimSpace(tokens[1]), rangeExpression
	}
	for k, v := range replaceMap {
		rangeExpression = strings.ReplaceAll(rangeExpression, k, v)
	}

	return "", "", rangeExpression
}

// ChangeTableNameForSpanner - ReplaceAll the hyphens (-) with underscore for given table name
// https://cloud.google.com/spanner/docs/data-definition-language#naming_conventions
func ChangeTableNameForSpanner(tableName string) string {
	tableName = strings.ReplaceAll(tableName, "-", "_")
	return tableName
}

// Convert DynamoDB data types to equivalent Spanner types
func ConvertDynamoTypeToSpannerType(dynamoType string) string {
	switch dynamoType {
	case "S":
		return "STRING(MAX)"
	case "N":
		return "FLOAT64"
	case "B":
		return "BYTES(MAX)"
	case "BOOL":
		return "BOOL"
	case "NULL":
		return "NULL"
	case "SS":
		return "ARRAY<STRING(MAX)>"
	case "NS":
		return "ARRAY<FLOAT64>"
	case "BS":
		return "ARRAY<BYTES(MAX)>"
	case "M":
		return "JSON"
	case "L":
		return "JSON"
	default:
		return "STRING(MAX)"
	}
}

// RemoveDuplicatesString removes duplicates from a []string
func RemoveDuplicatesString(input []string) []string {
	seen := make(map[string]struct{})
	var result []string

	for _, val := range input {
		if _, exists := seen[val]; !exists {
			seen[val] = struct{}{}
			result = append(result, val)
		}
	}
	return result
}

// RemoveDuplicatesFloat removes duplicates from a []float64
func RemoveDuplicatesFloat(input []float64) []float64 {
	seen := make(map[float64]struct{})
	var result []float64

	for _, val := range input {
		if _, exists := seen[val]; !exists {
			seen[val] = struct{}{}
			result = append(result, val)
		}
	}
	return result
}

// RemoveDuplicatesByteSlice removes duplicates from a [][]byte
func RemoveDuplicatesByteSlice(input [][]byte) [][]byte {
	seen := make(map[string]struct{})
	var result [][]byte

	for _, val := range input {
		key := string(val) // Convert byte slice to string for map key
		if _, exists := seen[key]; !exists {
			seen[key] = struct{}{}
			result = append(result, val)
		}
	}
	return result
}
