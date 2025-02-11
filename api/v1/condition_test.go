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

package v1

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/cloudspannerecosystem/dynamodb-adapter/models"
	"gopkg.in/go-playground/assert.v1"
)

func TestBetween(t *testing.T) {
	tests := []struct {
		testName, strValue, firstStr, secondStr, want string
	}{
		{"Empty Value", "", "s", "l", ""},
		{"Correct Values", "school", "sc", "ol", "ho"},
		{"Correct Values with 2 similar words", "stool", "o", "l", "o"},
		{"Empty first string", "school", "", "l", "schoo"},
		{"Empty second string", "school", "o", "", ""},
	}

	for _, tc := range tests {
		got := between(tc.strValue, tc.firstStr, tc.secondStr)
		assert.Equal(t, got, tc.want)
	}
}

func TestBefore(t *testing.T) {
	tests := []struct {
		testName, valueStr, searchStr, want string
	}{
		{"Empty Value", "", "s", ""},
		{"Correct Values", "school", "ho", "sc"},
		{"Empty string values", "school", "", ""},
	}

	for _, tc := range tests {
		got := before(tc.valueStr, tc.searchStr)
		assert.Equal(t, got, tc.want)
	}
}

func TestAfter(t *testing.T) {
	tests := []struct {
		testName, valueStr, searchStr, want string
	}{
		{"Empty Value", "", "s", ""},
		{"Correct Values", "school", "ho", "ol"},
		{"Empty string values", "school", "", ""},
	}

	for _, tc := range tests {
		got := after(tc.valueStr, tc.searchStr)
		assert.Equal(t, got, tc.want)

	}
}

func TestDeleteEmpty(t *testing.T) {
	tests := []struct {
		testName string
		inputArr []string
		want     []string
	}{
		{
			"Empty array",
			[]string{},
			nil,
		},
		{
			"Only spaces present in array",
			[]string{"", "", ""},
			nil,
		},
		{
			"Spaces present in the middle",
			[]string{"frist", "", "second", "", "third", "fourth"},
			[]string{"frist", "second", "third", "fourth"},
		},
		{
			"Spaces present in initial position",
			[]string{"", "", "", "frist", "second", "third", "fourth"},
			[]string{"frist", "second", "third", "fourth"},
		},
		{
			"Spaces present in end position",
			[]string{"frist", "second", "third", "fourth", "", "", ""},
			[]string{"frist", "second", "third", "fourth"},
		},
	}

	for _, tc := range tests {
		got := deleteEmpty(tc.inputArr)
		assert.Equal(t, got, tc.want)
	}
}

func Test_parseUpdateExpresstion(t *testing.T) {
	tests := []struct {
		testName string
		input    string
		want     *models.UpdateExpressionCondition
	}{
		{
			"empty input value",
			"",
			nil,
		},
		{
			"wrong syntax",
			"if_exists(name, :val2",
			nil,
		},
		{
			"if_exists only",
			"if_exists(name, :val2)",
			&models.UpdateExpressionCondition{
				Field:     []string{"name"},
				Value:     []string{":val2"},
				Condition: []string{"if_exists"},
				ActionVal: "%:val2%",
			},
		},
		{
			"if_not_exists only",
			"if_not_exists(name, :anyVal)",
			&models.UpdateExpressionCondition{
				Field:     []string{"name"},
				Value:     []string{":anyVal"},
				Condition: []string{"if_not_exists"},
				ActionVal: "%:anyVal%",
			},
		},
		{
			"if_exists and if_not_exists both",
			"if_exists(id, :val1) && if_not_exists(name, :val2)",
			&models.UpdateExpressionCondition{
				Field:     []string{"name", "id"},
				Value:     []string{":val2", ":val1"},
				Condition: []string{"if_not_exists", "if_exists"},
				ActionVal: "%:val1% && %:val2%",
			},
		},
		{
			"if_exists, if_not_exists and other values",
			"age >:ag && if_exists(id, :val1) && if_not_exists(name, :val2)",
			&models.UpdateExpressionCondition{
				Field:     []string{"name", "id"},
				Value:     []string{":val2", ":val1"},
				Condition: []string{"if_not_exists", "if_exists"},
				ActionVal: "age >:ag && %:val1% && %:val2%",
			},
		},
	}

	for _, tc := range tests {
		got := parseUpdateExpresstion(tc.input)
		assert.Equal(t, got, tc.want)
	}
}

func Test_extractOperations(t *testing.T) {
	tests := []struct {
		testName    string
		inputString string
		want        map[string]string
	}{
		{
			"Empty input string",
			"",
			nil,
		},
		{
			"Only SET Operation",
			"SET name = :val1, age = :val2",
			map[string]string{
				"SET": "name = :val1, age = :val2",
			},
		},
		{
			"Lowercase set Operation",
			"set name = :val1, age = :val2",
			map[string]string{
				"SET": "name = :val1, age = :val2",
			},
		},
		{
			"Only ADD Operation",
			"ADD age :val1",
			map[string]string{
				"ADD": "age :val1",
			},
		},
		{
			"Title case Add Operation",
			"Add age :val1",
			map[string]string{
				"ADD": "age :val1",
			},
		},
		{
			"Only REMOVE Operation",
			"REMOVE address",
			map[string]string{
				"REMOVE": "address",
			},
		},
		{
			"Mixed case ReMoVe Operation",
			"ReMoVe address",
			map[string]string{
				"REMOVE": "address",
			},
		},
		{
			"Only DELETE Operation",
			"DELETE Color :p",
			map[string]string{
				"DELETE": "Color :p",
			},
		},
		{
			"Lower case delete Operation",
			"delete Color :p",
			map[string]string{
				"DELETE": "Color :p",
			},
		},
	}

	for _, tc := range tests {
		got := extractOperations(tc.inputString)
		assert.Equal(t, got, tc.want)
	}
}

func TestReplaceHashRangeExpr(t *testing.T) {
	tests := []struct {
		testName string
		input    models.Query
		want     models.Query
	}{
		{
			"empty input ",
			models.Query{},
			models.Query{},
		},
		{
			"empty ExpressionAttributeNames ",
			models.Query{
				ExpressionAttributeNames: nil,
				RangeExp:                 "#e = :val1",
				FilterExp:                "#ag > :val2",
			},
			models.Query{
				ExpressionAttributeNames: nil,
				RangeExp:                 "#e = :val1",
				FilterExp:                "#ag > :val2",
			},
		},
		{
			"Correct Input",
			models.Query{
				ExpressionAttributeNames: map[string]string{
					"#e":  "emp_id",
					"#ag": "age",
				},
				RangeExp:  "#e = :val1",
				FilterExp: "#ag > :val2",
			},
			models.Query{
				ExpressionAttributeNames: map[string]string{
					"#e":  "emp_id",
					"#ag": "age",
				},
				RangeExp:  "emp_id = :val1",
				FilterExp: "age > :val2",
			},
		},
	}

	for _, tc := range tests {
		got := ReplaceHashRangeExpr(tc.input)
		assert.Equal(t, got, tc.want)
	}
}

func TestConvertDynamoToMap(t *testing.T) {
	tests := []struct {
		testName       string
		dynamodbObject map[string]*dynamodb.AttributeValue
		want           map[string]interface{}
	}{
		{
			"empty dynamodbObject",
			nil,
			nil,
		},
		{
			"dynamodbObject with String present",
			map[string]*dynamodb.AttributeValue{
				"address":    {S: aws.String("Ney York")},
				"first_name": {S: aws.String("Catalina")},
				"last_name":  {S: aws.String("Smith")},
			},
			map[string]interface{}{
				"address":    "Ney York",
				"first_name": "Catalina",
				"last_name":  "Smith",
			},
		},
		{
			"dynamodbObject with diffent type of params",
			map[string]*dynamodb.AttributeValue{
				"emp_id":     {N: aws.String("2")},
				"age":        {N: aws.String("20")},
				"address":    {S: aws.String("Ney York")},
				"first_name": {S: aws.String("Catalina")},
				"last_name":  {S: aws.String("Smith")},
				"subjects": {L: []*dynamodb.AttributeValue{
					{S: aws.String("Maths")},
					{S: aws.String("Physics")},
					{S: aws.String("Chemistry")},
				}},
			},
			map[string]interface{}{
				"emp_id":     float64(2),
				"age":        float64(20),
				"address":    "Ney York",
				"first_name": "Catalina",
				"last_name":  "Smith",
				"subjects":   []interface{}{"Maths", "Physics", "Chemistry"},
			},
		},
	}

	for _, tc := range tests {
		got, _ := ConvertDynamoToMap("", tc.dynamodbObject)
		assert.Equal(t, got, tc.want)
	}
}

func TestChangeMaptoDynamoMap(t *testing.T) {
	tests := []struct {
		testName string
		input    interface{}
		want     map[string]interface{}
	}{
		{
			"empty input",
			nil,
			nil,
		},
		{
			"Only String Values for input",
			map[string]interface{}{
				"address": "London",
				"name":    "Richard",
			},
			map[string]interface{}{
				"address": map[string]interface{}{"S": "London"},
				"name":    map[string]interface{}{"S": "Richard"},
			},
		},
		{
			"Mixed data types for input",
			map[string]interface{}{
				"address": "London",
				"name":    "Richard",
				"age":     20,
				"value":   float64(10),
				"array":   []string{"first", "second", "third"},
			},
			map[string]interface{}{
				"address": map[string]interface{}{"S": "London"},
				"name":    map[string]interface{}{"S": "Richard"},
				"age":     map[string]interface{}{"N": "20"},
				"value":   map[string]interface{}{"N": "10"},
				"array": map[string]interface{}{
					"L": []map[string]interface{}{
						{"S": "first"},
						{"S": "second"},
						{"S": "third"},
					},
				},
			},
		},
	}

	for _, tc := range tests {
		got, _ := ChangeMaptoDynamoMap(tc.input)
		assert.Equal(t, got, tc.want)
	}
}
