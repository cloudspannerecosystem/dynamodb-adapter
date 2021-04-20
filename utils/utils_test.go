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
	"testing"

	"github.com/antonmedv/expr"
	"github.com/cloudspannerecosystem/dynamodb-adapter/models"
	"gopkg.in/go-playground/assert.v1"
)

func TestGetStringInBetween(t *testing.T) {
	tests := []struct {
		testName, strValue, firstStr, secondStr, want string
	}{
		{"Empty Value for String", "", "s", "l", ""},
		{"Correct Values here", "school", "sc", "ol", "ho"},
		{"Correct Values with 2 similar letters", "stool", "o", "l", "o"},
		{"Empty 1st string", "school", "", "l", "schoo"},
		{"Empty 2nd string", "school", "o", "", ""},
	}

	for _, tc := range tests {
		got := GetStringInBetween(tc.strValue, tc.firstStr, tc.secondStr)
		assert.Equal(t, got, tc.want)
	}
}

func TestGetFieldNameFromConditionalExpression(t *testing.T) {
	tests := []struct {
		testName, condExpr, want string
	}{
		{"Empty Value", "", ""},
		{"Any String passed", "Any stirng", "Any stirng"},
		{"String with attribute_exists ", "attribute_exists(name)", "name"},
		{"String with attribute_not_exists", "attribute_not_exists(some_field)", "some_field"},
	}

	for _, tc := range tests {
		got := GetFieldNameFromConditionalExpression(tc.condExpr)
		assert.Equal(t, got, tc.want)
	}
}

func TestCreateConditionExpression(t *testing.T) {
	cond1, _ := expr.Compile(`TOKEN0 > "20" && TOKEN4 `)

	tests := []struct {
		testName            string
		conditionExpression string
		attributeMap        map[string]interface{}
		want                *models.Eval
	}{
		{
			"empty Conditonal Expression",
			"",
			nil,
			new(models.Eval),
		},
		{
			"Attribute map not present",
			"age > :val AND attribute_exists(c)",
			nil,
			nil,
		},
		{
			"Conditonal Expression with attributeMap",
			"age > :val AND attribute_exists(c)",
			map[string]interface{}{":val": "20"},
			&models.Eval{
				Cond:       cond1,
				Attributes: []string{"age", "attribute_exists(c)"},
				Cols:       []string{"age", "c"},
				Tokens:     []string{"TOKEN0", "TOKEN4"},
				ValueMap:   make(map[string]interface{}),
			},
		},
	}

	for _, tc := range tests {
		got, _ := CreateConditionExpression(tc.conditionExpression, tc.attributeMap)
		assert.Equal(t, got, tc.want)
	}
}

func TestEvaluateExpression(t *testing.T) {
	cond1, _ := expr.Compile(`TOKEN0 > "20" && TOKEN4 `)
	tests := []struct {
		testName string
		input    *models.Eval
		want     bool
	}{
		{
			"No Input",
			nil,
			true,
		},
		{
			"Cond is nil in input",
			&models.Eval{
				Cond:       nil,
				Attributes: []string{"age", "attribute_exists(c)"},
				Cols:       []string{"age", "c"},
				Tokens:     []string{"TOKEN0", "TOKEN4"},
				ValueMap:   make(map[string]interface{}),
			},
			true,
		},
		{
			"ValueMap is nil",
			&models.Eval{
				Cond:       cond1,
				Attributes: []string{"age", "attribute_exists(c)"},
				Cols:       []string{"age", "c"},
				Tokens:     []string{"TOKEN0", "TOKEN4"},
				ValueMap:   nil,
			},
			false,
		},
		{
			"Correct Params",
			&models.Eval{
				Cond:       cond1,
				Attributes: []string{"age", "attribute_exists(c)"},
				Cols:       []string{"age", "c"},
				Tokens:     []string{"TOKEN0", "TOKEN4"},
				ValueMap: map[string]interface{}{
					"TOKEN0": "age",
					"TOKEN4": true,
				},
			},
			true,
		},
	}

	// EvaluateExpression()
	for _, tc := range tests {
		got, _ := EvaluateExpression(tc.input)
		assert.Equal(t, got, tc.want)
	}
}

func TestParseBeginsWith(t *testing.T) {
	tests := []struct {
		testName, rangeExpression string
		want                      map[string]string
	}{
		{
			"Empty rangeExpression",
			"",
			map[string]string{
				"first":  "",
				"second": "",
				"third":  "",
			},
		},
		{
			"rangeExpression with begins_with()",
			"begins_with(name, :val)",
			map[string]string{
				"first":  "name",
				"second": ":val",
				"third":  "begins_with(name, :val)",
			},
		},
		{
			"ragneEpression without begins_with()",
			"age > 20",
			map[string]string{
				"first":  "",
				"second": "",
				"third":  "age > 20",
			},
		},
		{
			"ragneEpression with special symbols GT",
			"age GT 20",
			map[string]string{
				"first":  "",
				"second": "",
				"third":  "age > 20",
			},
		},
		{
			"ragneEpression with special symbols LT",
			"age GT 20",
			map[string]string{
				"first":  "",
				"second": "",
				"third":  "age > 20",
			},
		},
	}

	for _, tc := range tests {
		first, second, third := ParseBeginsWith(tc.rangeExpression)
		assert.Equal(t, first, tc.want["first"])
		assert.Equal(t, second, tc.want["second"])
		assert.Equal(t, third, tc.want["third"])
	}
}
