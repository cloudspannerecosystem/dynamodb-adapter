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

	"github.com/antonmedv/expr"
	"github.com/cloudspannerecosystem/dynamodb-adapter/models"
	"github.com/cloudspannerecosystem/dynamodb-adapter/pkg/errors"
	"github.com/cloudspannerecosystem/dynamodb-adapter/pkg/logger"
)

var listRemoveTargetRegex = regexp.MustCompile(`(.*)\[(\d+)\]`)
var base64Regexp = regexp.MustCompile("^([A-Za-z0-9+/]{4})*([A-Za-z0-9+/]{3}=|[A-Za-z0-9+/]{2}==)?$")

var CreateConditionExpressionFunc = CreateConditionExpression

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

// stripWrappingParens removes unnecessary wrapping or unmatched parentheses from a string.
//
// - If the string is wrapped in balanced parentheses, it removes the outermost pair(s).
// - If there are unmatched leading or trailing parentheses, it removes them until balanced.
// - It preserves valid, balanced parentheses inside the string.
//
// Examples:
//
//	stripWrappingParens("(foo)")            // "foo"
//	stripWrappingParens("((foo))")          // "foo"
//	stripWrappingParens("(foo")             // "foo"
//	stripWrappingParens("foo)")             // "foo"
//	stripWrappingParens("((foo)")           // "foo"
//	stripWrappingParens("(size(bar))")      // "size(bar)"
//	stripWrappingParens("(size(bar)")       // "size(bar)"
//	stripWrappingParens("size(bar))")       // "size(bar)"
//	stripWrappingParens("size(bar)")        // "size(bar)"
//	stripWrappingParens("foo(bar(baz))")    // "foo(bar(baz))"
//	stripWrappingParens("((foo(bar)))")     // "foo(bar)"
func stripWrappingParens(s string) string {
	s = strings.TrimSpace(s)
	// Remove balanced wrapping parentheses
	for {
		changed := false
		// Remove balanced outermost parens
		for strings.HasPrefix(s, "(") && strings.HasSuffix(s, ")") && len(s) > 1 {
			inner := s[1 : len(s)-1]
			if countParens(inner) == 0 {
				s = strings.TrimSpace(inner)
				changed = true
			} else {
				break
			}
		}
		// Remove unmatched leading paren
		if strings.HasPrefix(s, "(") && countParens(s) > 0 {
			s = strings.TrimSpace(s[1:])
			changed = true
		}
		// Remove unmatched trailing paren
		if strings.HasSuffix(s, ")") && countParens(s) < 0 {
			s = strings.TrimSpace(s[:len(s)-1])
			changed = true
		}
		if !changed {
			break
		}
	}
	return s
}

// Helper: returns open parens minus close parens
func countParens(s string) int {
	open := 0
	close := 0
	for _, c := range s {
		if c == '(' {
			open++
		} else if c == ')' {
			close++
		}
	}
	return open - close
}

// cleanExpressionSpacing removes unnecessary spaces between function names and their opening parenthesis
// in a DynamoDB expression, while preserving spaces for logical operators like AND, OR, etc. This is needed
// since logic below splits tokens by spaces.
//
// For example, it converts:
//
//	"attribute_exists (foo) AND begins_with (bar, :val)"
//
// to:
//
//	"attribute_exists(foo) AND begins_with(bar, :val)"
//
// Logical operators (AND, OR, etc.) are not affected, so their spacing remains
func cleanExpressionSpacing(expression string) string {
	// Regex to find words followed by any whitespace (space, tab, etc.) and (
	re := regexp.MustCompile(`\b(\w+)\s+\(`)

	// List of logical operators to exclude
	logicalOps := map[string]bool{
		"AND": true,
		"and": true,
		"OR":  true,
		"or":  true,
	}

	// Use ReplaceAllStringFunc to process each match
	return re.ReplaceAllStringFunc(expression, func(m string) string {
		// Extract the word before whitespace+(
		// m is like "attribute_exists (" or "attribute_exists\t("
		parts := regexp.MustCompile(`\s+`).Split(m, 2)
		if len(parts) != 2 {
			return m
		}
		word := parts[0]
		if logicalOps[word] {
			// If logical operator, don't change spacing
			return m
		}
		// Otherwise remove all whitespace before '('
		return word + "("
	})
}

// CreateConditionExpression - create evelute condition from condition
func CreateConditionExpression(condtionExpression string, expressionAttr map[string]interface{}) (*models.Eval, error) {
	if condtionExpression == "" {
		e := new(models.Eval)
		return e, nil
	}
	logger.Debug("Original condition expression:", condtionExpression)
	condtionExpression = strings.TrimSpace(condtionExpression)
	condtionExpression = strings.ReplaceAll(condtionExpression, "( ", "(")
	condtionExpression = strings.ReplaceAll(condtionExpression, " )", ")")
	condtionExpression = strings.ReplaceAll(condtionExpression, "NOT ", "!")
	condtionExpression = cleanExpressionSpacing(condtionExpression)
	tokens := strings.Split(condtionExpression, " ")
	sb := strings.Builder{}
	evalTokens := []string{}
	cols := []string{}
	ts := []string{}
	var err error
	for i := 0; i < len(tokens); i++ {
		token := tokens[i]
		if i%2 == 0 {
			isNegated := false
			if strings.HasPrefix(token, "!") {
				isNegated = true
				token = strings.TrimPrefix(token, "!")
			}
			token = stripWrappingParens(token)
			if strings.Contains(token, ":") {
				v, ok := expressionAttr[token]
				if !ok {
					return nil, errors.New("ResourceNotFoundException", expressionAttr, token)
				}
				str := fmt.Sprint(v)
				_, ok = v.(string)
				if ok {
					str = "\"" + str + "\""
				}
				switch v.(type) { // TODO: Support timestamp and big.rat
				case float64:
					str = fmt.Sprintf("%f", v)
				case int64:
					str = fmt.Sprintf("%d", v)
				case []interface{}:
					// Handle lists by converting them to JSON for easier evaluation
					listBytes, err := json.Marshal(v)
					if err != nil {
						return nil, errors.New("InvalidListException", err.Error(), token)
					}
					str = string(listBytes)
				}
				sb.WriteString(str)
				sb.WriteString(" ")
				continue
			}

			t := "TOKEN" + strconv.Itoa(i)
			col := GetFieldNameFromConditionalExpression(token)
			if isNegated {
				sb.WriteString("!(" + t + ") ")
			} else {
				sb.WriteString(t + " ")
			}
			evalTokens = append(evalTokens, token)
			cols = append(cols, col)
			ts = append(ts, t)
		} else {
			sb.WriteString(token)
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
// Only used by initialization code to create tables
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
				// Attempt to unmarshal the base64 decoded bytes as JSON
				var m interface{}
				if err := json.Unmarshal(ba, &m); err == nil {
					return ParseNestedJSON(m)
				}
				return ba
			}
		}
		// Keep string as is if not base64 encoded
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

func TrimSingleQuotes(s string) string {
	// Check if the string starts and ends with single quotes
	if strings.HasPrefix(s, "'") && strings.HasSuffix(s, "'") {
		// Remove the quotes from the beginning and end
		s = s[1 : len(s)-1]
	}
	return s
}
