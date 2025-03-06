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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	"cloud.google.com/go/spanner"
	"github.com/antonmedv/expr"
	"github.com/cloudspannerecosystem/dynamodb-adapter/models"
	"github.com/cloudspannerecosystem/dynamodb-adapter/pkg/errors"
	"github.com/cloudspannerecosystem/dynamodb-adapter/pkg/logger"
)

var listRemoveTargetRegex = regexp.MustCompile(`(.*)\[(\d+)\]`)
var base64Regexp = regexp.MustCompile("^([A-Za-z0-9+/]{4})*([A-Za-z0-9+/]{3}=|[A-Za-z0-9+/]{2}==)?$")

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
				case []interface{}:
					// Handle lists by converting them to JSON for easier evaluation
					listBytes, err := json.Marshal(v)
					if err != nil {
						return nil, errors.New("InvalidListException", err.Error(), tokens[i])
					}
					str = string(listBytes)
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

// parseListRemoveTarget parses a list attribute target and its index from the action value.
// It returns the attribute name and index.
// Example: listAttr[2]
func ParseListRemoveTarget(target string) (string, int) {
	matches := listRemoveTargetRegex.FindStringSubmatch(target)
	if len(matches) == 3 {
		index, err := strconv.Atoi(matches[2])
		if err != nil {
			return target, -1
		}
		return matches[1], index
	}
	return target, -1
}

// removeListElement removes an element from a list at the specified index.
// If the index is invalid, it returns the original list.
func RemoveListElement(list []interface{}, idx int) []interface{} {
	if idx < 0 || idx >= len(list) {
		return list // Return original list for invalid indices
	}
	return append(list[:idx], list[idx+1:]...)
}

// IsValidJSONObject checks if a string is a valid JSON object
func IsValidJSONObject(s string) error {
	var js map[string]interface{}
	err := json.Unmarshal([]byte(s), &js)
	return err
}

func IsValidBase64(s string) bool {
	if _, err := base64.StdEncoding.DecodeString(s); err != nil {
		return false
	}
	return true
}
func ParseBytes(r *spanner.Row, i int, k string) (map[string]interface{}, error) {
	var s []byte
	singleRowImg := make(map[string]interface{})
	err := r.Column(i, &s)
	if err != nil {
		if strings.Contains(err.Error(), "ambiguous column name") {
			return nil, err
		}
		return nil, errors.New("ValidationException", err, k)
	}
	if len(s) > 0 {
		var m interface{}
		err := json.Unmarshal(s, &m)
		if err != nil {
			logger.LogError(err, string(s))
			singleRowImg[k] = string(s)
		}
		val1, ok := m.(string)
		if ok {
			if base64Regexp.MatchString(val1) {
				ba, err := base64.StdEncoding.DecodeString(val1)
				if err == nil {
					var sample interface{}
					err = json.Unmarshal(ba, &sample)
					if err == nil {
						singleRowImg[k] = sample

					} else {
						singleRowImg[k] = string(s)

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
		singleRowImg[k] = m

	}
	return singleRowImg, err
}

func ParseNestedJSON(value interface{}) interface{} {
	switch v := value.(type) {
	case map[string]interface{}:
		m := make(map[string]interface{})
		for key, val := range v {
			m[key] = ParseNestedJSON(val)
		}
		return map[string]interface{}{"M": m} // Wrap around with "M"
	case []interface{}:
		for i, item := range v {
			v[i] = ParseNestedJSON(item)
		}
		return v
	case string:
		// Check for base64 encoding
		if base64Regexp.MatchString(v) {
			ba, err := base64.StdEncoding.DecodeString(v)
			if err == nil {
				return ParseNestedJSON(string(ba)) // Convert bytes back to string
			}
		}
		return v // Keep string as is
	case float64:
		return v
	default:
		return v
	}
}

// UpdateFieldByPath navigates the nested JSON structure to update the desired field.
func UpdateFieldByPath(data map[string]interface{}, path string, newValue interface{}) bool {
	keys := strings.Split(path, ".")
	keys = keys[1:]
	// Traverse to the deepest map
	current := data
	for i, key := range keys {
		if i == len(keys)-1 {
			// If it's the last key, perform the update
			current[key] = newValue
			return true
		}

		// Traverse deeper into the map structure
		if next, ok := current[key].(map[string]interface{}); ok {
			current = next
		} else {
			// Path is invalid if we can't find the next map level
			log.Printf("Invalid path: key %s not found\n", key)
			return false
		}
	}
	return false
}
