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

package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/cloudspannerecosystem/dynamodb-adapter/api"
	"github.com/cloudspannerecosystem/dynamodb-adapter/apitesting"
	"github.com/cloudspannerecosystem/dynamodb-adapter/initializer"
	"github.com/cloudspannerecosystem/dynamodb-adapter/models"
	httpexpect "github.com/gavv/httpexpect/v2"
	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v2"
)

// database name used in all the test cases
var databaseName string
var readConfigFile = os.ReadFile

// params for TestGetItemAPI
var (
	getItemTest1 = models.GetItemMeta{
		TableName: "employee",
	}
	getItemTest1_1 = models.GetItemMeta{
		TableName: "employee",
		Key: map[string]*dynamodb.AttributeValue{
			"emp_id": {N: aws.String("")},
		},
	}
	getItemTest2 = models.GetItemMeta{
		TableName: "employee",
		Key: map[string]*dynamodb.AttributeValue{
			"emp_id": {N: aws.String("2")},
		},
	}
	getItemTest2Output = `{"Item":{"address":{"S":"Ney York"},"age":{"N":"20"},"emp_id":{"N":"2"},"first_name":{"S":"Catalina"},"last_name":{"S":"Smith"}}}`
	getItemTest3       = models.GetItemMeta{
		TableName: "employee",
		Key: map[string]*dynamodb.AttributeValue{
			"emp_id": {N: aws.String("2")},
		},
		ProjectionExpression: "emp_id, address",
	}
	getItemTest3Output = `{"Item":{"address":{"S":"Ney York"},"emp_id":{"N":"2"}}}`

	getItemTest4 = models.GetItemMeta{
		TableName: "employee",
		Key: map[string]*dynamodb.AttributeValue{
			"emp_id": {N: aws.String("2")},
		},
		ProjectionExpression: "#emp, address",
		ExpressionAttributeNames: map[string]string{
			"#emp": "emp_id",
		},
	}
	getItemTest5 = models.GetItemMeta{
		TableName: "employee",
		Key: map[string]*dynamodb.AttributeValue{
			"emp_id": {N: aws.String("2")},
		},
		ProjectionExpression: "#emp, address",
	}
	getItemTest5Output = `{"Item":{"address":{"S":"Ney York"}}}`
)

// params for TestGetBatchAPI
var (
	TestGetBatch1Name = "1: wrong url"
	TestGetBatch1     = models.BatchGetMeta{
		RequestItems: map[string]models.BatchGetWithProjectionMeta{
			"employee": {},
		},
	}

	TestGetBatch2Name = "2: only table name"
	TestGetBatch2     = models.BatchGetMeta{
		RequestItems: map[string]models.BatchGetWithProjectionMeta{
			"employee": {},
		},
	}
	TestGetBatch2Output = `{"Responses":{"employee":[]}}`

	TestGetBatch3Name = "3: Keys present for 1 table"
	TestGetBatch3     = models.BatchGetMeta{
		RequestItems: map[string]models.BatchGetWithProjectionMeta{
			"employee": {
				Keys: []map[string]*dynamodb.AttributeValue{
					{"emp_id": {N: aws.String("1")}},
					{"emp_id": {N: aws.String("5")}},
					{"emp_id": {N: aws.String("3")}},
				},
			},
		},
	}
	TestGetBatch3Output = `{"Responses":{"employee":[{"address":{"S":"Shamli"},"age":{"N":"10"},"emp_id":{"N":"1"},"first_name":{"S":"Marc"},"last_name":{"S":"Richards"}},{"address":{"S":"Pune"},"age":{"N":"30"},"emp_id":{"N":"3"},"first_name":{"S":"Alice"},"last_name":{"S":"Trentor"}},{"address":{"S":"London"},"age":{"N":"50"},"emp_id":{"N":"5"},"first_name":{"S":"David"},"last_name":{"S":"Lomond"}}]}}`

	TestGetBatch4Name = "4: Keys present for 2 table"
	TestGetBatch4     = models.BatchGetMeta{
		RequestItems: map[string]models.BatchGetWithProjectionMeta{
			"employee": {
				Keys: []map[string]*dynamodb.AttributeValue{
					{"emp_id": {N: aws.String("1")}},
					{"emp_id": {N: aws.String("5")}},
					{"emp_id": {N: aws.String("3")}},
				},
			},
			"department": {
				Keys: []map[string]*dynamodb.AttributeValue{
					{"d_id": {N: aws.String("100")}},
					{"d_id": {N: aws.String("300")}},
				},
			},
		},
	}
	TestGetBatch4Output = `{"Responses":{"department":[{"d_id":{"N":"100"},"d_name":{"S":"Engineering"},"d_specialization":{"S":"CSE, ECE, Civil"}},{"d_id":{"N":"300"},"d_name":{"S":"Culture"},"d_specialization":{"S":"History"}}],"employee":[{"address":{"S":"Shamli"},"age":{"N":"10"},"emp_id":{"N":"1"},"first_name":{"S":"Marc"},"last_name":{"S":"Richards"}},{"address":{"S":"Pune"},"age":{"N":"30"},"emp_id":{"N":"3"},"first_name":{"S":"Alice"},"last_name":{"S":"Trentor"}},{"address":{"S":"London"},"age":{"N":"50"},"emp_id":{"N":"5"},"first_name":{"S":"David"},"last_name":{"S":"Lomond"}}]}}`

	TestGetBatch5Name = "5: ProjectionExpression without ExpressionAttributeNames for 1 table"
	TestGetBatch5     = models.BatchGetMeta{
		RequestItems: map[string]models.BatchGetWithProjectionMeta{
			"employee": {
				Keys: []map[string]*dynamodb.AttributeValue{
					{"emp_id": {N: aws.String("1")}},
					{"emp_id": {N: aws.String("5")}},
					{"emp_id": {N: aws.String("3")}},
				},
				ProjectionExpression: "emp_id, address, first_name, last_name",
			},
		},
	}
	TestGetBatch5Output = `{"Responses":{"employee":[{"address":{"S":"Shamli"},"emp_id":{"N":"1"},"first_name":{"S":"Marc"},"last_name":{"S":"Richards"}},{"address":{"S":"Pune"},"emp_id":{"N":"3"},"first_name":{"S":"Alice"},"last_name":{"S":"Trentor"}},{"address":{"S":"London"},"emp_id":{"N":"5"},"first_name":{"S":"David"},"last_name":{"S":"Lomond"}}]}}`

	TestGetBatch6Name = "6: ProjectionExpression without ExpressionAttributeNames for 2 table"
	TestGetBatch6     = models.BatchGetMeta{
		RequestItems: map[string]models.BatchGetWithProjectionMeta{
			"employee": {
				Keys: []map[string]*dynamodb.AttributeValue{
					{"emp_id": {N: aws.String("1")}},
					{"emp_id": {N: aws.String("5")}},
					{"emp_id": {N: aws.String("3")}},
				},
				ProjectionExpression: "emp_id, address, first_name, last_name",
			},
			"department": {
				Keys: []map[string]*dynamodb.AttributeValue{
					{"d_id": {N: aws.String("100")}},
					{"d_id": {N: aws.String("300")}},
				},
				ProjectionExpression: "d_id, d_name, d_specialization",
			},
		},
	}
	TestGetBatch6Output = `{"Responses":{"department":[{"d_id":{"N":"100"},"d_name":{"S":"Engineering"},"d_specialization":{"S":"CSE, ECE, Civil"}},{"d_id":{"N":"300"},"d_name":{"S":"Culture"},"d_specialization":{"S":"History"}}],"employee":[{"address":{"S":"Shamli"},"emp_id":{"N":"1"},"first_name":{"S":"Marc"},"last_name":{"S":"Richards"}},{"address":{"S":"Pune"},"emp_id":{"N":"3"},"first_name":{"S":"Alice"},"last_name":{"S":"Trentor"}},{"address":{"S":"London"},"emp_id":{"N":"5"},"first_name":{"S":"David"},"last_name":{"S":"Lomond"}}]}}`

	TestGetBatch7Name = "7: ProjectionExpression with ExpressionAttributeNames for 1 table"
	TestGetBatch7     = models.BatchGetMeta{
		RequestItems: map[string]models.BatchGetWithProjectionMeta{
			"employee": {
				Keys: []map[string]*dynamodb.AttributeValue{
					{"emp_id": {N: aws.String("1")}},
					{"emp_id": {N: aws.String("5")}},
					{"emp_id": {N: aws.String("3")}},
				},
				ProjectionExpression: "#emp, #add, first_name, last_name",
				ExpressionAttributeNames: map[string]string{
					"#emp": "emp_id",
					"#add": "address",
				},
			},
		},
	}
	TestGetBatch7Output = `{"Responses":{"employee":[{"address":{"S":"Shamli"},"emp_id":{"N":"1"},"first_name":{"S":"Marc"},"last_name":{"S":"Richards"}},{"address":{"S":"Pune"},"emp_id":{"N":"3"},"first_name":{"S":"Alice"},"last_name":{"S":"Trentor"}},{"address":{"S":"London"},"emp_id":{"N":"5"},"first_name":{"S":"David"},"last_name":{"S":"Lomond"}}]}}`

	TestGetBatch8Name = "8: ProjectionExpression with ExpressionAttributeNames for 2 table"
	TestGetBatch8     = models.BatchGetMeta{
		RequestItems: map[string]models.BatchGetWithProjectionMeta{
			"employee": {
				Keys: []map[string]*dynamodb.AttributeValue{
					{"emp_id": {N: aws.String("1")}},
					{"emp_id": {N: aws.String("5")}},
					{"emp_id": {N: aws.String("3")}},
				},
				ProjectionExpression: "#emp, #add, first_name, last_name",
				ExpressionAttributeNames: map[string]string{
					"#emp": "emp_id",
					"#add": "address",
				},
			},
			"department": {
				Keys: []map[string]*dynamodb.AttributeValue{
					{"d_id": {N: aws.String("100")}},
					{"d_id": {N: aws.String("300")}},
				},
				ProjectionExpression: "d_id, #dn, #ds",
				ExpressionAttributeNames: map[string]string{
					"#ds": "d_specialization",
					"#dn": "d_name",
				},
			},
		},
	}
	TestGetBatch8Output = `{"Responses":{"department":[{"d_id":{"N":"100"},"d_name":{"S":"Engineering"},"d_specialization":{"S":"CSE, ECE, Civil"}},{"d_id":{"N":"300"},"d_name":{"S":"Culture"},"d_specialization":{"S":"History"}}],"employee":[{"address":{"S":"Shamli"},"emp_id":{"N":"1"},"first_name":{"S":"Marc"},"last_name":{"S":"Richards"}},{"address":{"S":"Pune"},"emp_id":{"N":"3"},"first_name":{"S":"Alice"},"last_name":{"S":"Trentor"}},{"address":{"S":"London"},"emp_id":{"N":"5"},"first_name":{"S":"David"},"last_name":{"S":"Lomond"}}]}}`

	TestGetBatch9Name = "9: ProjectionExpression but ExpressionAttributeNames not present"
	TestGetBatch9     = models.BatchGetMeta{
		RequestItems: map[string]models.BatchGetWithProjectionMeta{
			"employee": {
				Keys: []map[string]*dynamodb.AttributeValue{
					{"emp_id": {N: aws.String("1")}},
					{"emp_id": {N: aws.String("5")}},
					{"emp_id": {N: aws.String("3")}},
				},
				ProjectionExpression: "#emp, #add, first_name, last_name",
			},
		},
	}
	TestGetBatch9Output = `{"Responses":{"employee":[{"first_name":{"S":"Marc"},"last_name":{"S":"Richards"}},{"first_name":{"S":"Alice"},"last_name":{"S":"Trentor"}},{"first_name":{"S":"David"},"last_name":{"S":"Lomond"}}]}}`

	TestGetBatch10Name = "10: Wrong Keys"
	TestGetBatch10     = models.BatchGetMeta{
		RequestItems: map[string]models.BatchGetWithProjectionMeta{
			"employee": {
				Keys: []map[string]*dynamodb.AttributeValue{
					{"emp_id": {S: aws.String("1")}},
					{"emp_id": {N: aws.String("5")}},
					{"emp_id": {N: aws.String("3")}},
				},
			},
		},
	}
)

// test Data for Query API
var (
	//empty 404
	queryTestCase0 = models.Query{}

	//only table name
	queryTestCase1 = models.Query{
		TableName: "employee",
	}

	//table & projection expression
	queryTestCase2 = models.Query{
		TableName:            "employee",
		ProjectionExpression: "emp_id, first_name, #last ",
	}

	//projection expression with ExpressionAttributeNames
	queryTestCase3 = models.Query{
		TableName: "employee",
		ExpressionAttributeNames: map[string]string{
			"#last": "last_name",
			"#emp":  "emp_id",
		},
		ProjectionExpression: "#emp, first_name, #last ",
	}

	// KeyconditionExpression
	queryTestCase4 = models.Query{
		TableName: "employee",
		ExpressionAttributeNames: map[string]string{
			"#last": "last_name",
			"#emp":  "emp_id",
		},
		ProjectionExpression: "#emp, first_name, #last ",
		RangeExp:             "#emp = :val1 ",
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":val1": {N: aws.String("2")},
		},
	}

	//(400 bad request) KeyconditionExpression without ExpressionAttributeValues
	queryTestCase5 = models.Query{
		TableName: "employee",
		ExpressionAttributeNames: map[string]string{
			"#last": "last_name",
			"#emp":  "emp_id",
		},
		ProjectionExpression: "#emp, first_name, #last ",
		RangeExp:             "#emp = :val1",
	}

	//with filter experssion
	queryTestCase6 = models.Query{
		TableName: "employee",
		ExpressionAttributeNames: map[string]string{
			"#last": "last_name",
			"#emp":  "emp_id",
		},
		ProjectionExpression: "#emp, first_name, #last ",
		RangeExp:             "#emp = :val1",
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":val1": {N: aws.String("3")},
			":last": {S: aws.String("Trentor")},
		},
		FilterExp: "last_name = :last",
	}

	//(400 bad request) filter expression but value not present
	queryTestCase7 = models.Query{
		TableName: "employee",
		ExpressionAttributeNames: map[string]string{
			"#last": "last_name",
			"#emp":  "emp_id",
		},
		ProjectionExpression: "#emp, first_name, #last ",
		RangeExp:             "#emp = :val1",
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":val1": {N: aws.String("3")},
		},
		FilterExp: "last_name = :last",
	}

	//only filter expression
	queryTestCase8 = models.Query{
		TableName: "employee",
		ExpressionAttributeNames: map[string]string{
			"#last": "last_name",
			"#emp":  "emp_id",
		},
		ProjectionExpression: "#emp, first_name, #last ",
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":last": {S: aws.String("Trentor")},
		},
		FilterExp: "last_name = :last",
	}

	//ScanIndexForward with filter & Keyconditions expression
	queryTestCase9 = models.Query{
		TableName: "employee",
		ExpressionAttributeNames: map[string]string{
			"#last": "last_name",
			"#emp":  "emp_id",
		},
		ProjectionExpression: "#emp, first_name, #last ",
		RangeExp:             "#emp = :val1",
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":val1": {N: aws.String("3")},
			":last": {S: aws.String("Trentor")},
		},
		FilterExp:     "last_name = :last",
		SortAscending: true,
	}

	//with ScanIndexForward only
	queryTestCase10 = models.Query{
		TableName:     "employee",
		SortAscending: true,
	}

	//with Limit
	queryTestCase11 = models.Query{
		TableName: "employee",
		Limit:     4,
	}

	//with Limit & ScanIndexForward
	queryTestCase12 = models.Query{
		TableName:     "employee",
		SortAscending: true,
		Limit:         4,
	}

	//only count
	queryTestCase13 = models.Query{
		TableName: "employee",
		Select:    "COUNT",
	}

	//count with other attributes present
	queryTestCase14 = models.Query{
		TableName: "employee",
		ExpressionAttributeNames: map[string]string{
			"#last": "last_name",
			"#emp":  "emp_id",
		},
		ProjectionExpression: "#emp, first_name, #last ",
		RangeExp:             "#emp = :val1",
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":val1": {N: aws.String("3")},
			":last": {S: aws.String("Trentor")},
		},
		FilterExp: "last_name = :last",
		Select:    "COUNT",
		Limit:     4,
	}

	//Select with other than count
	queryTestCase15 = models.Query{
		TableName: "employee",
		ExpressionAttributeNames: map[string]string{
			"#last": "last_name",
			"#emp":  "emp_id",
		},
		ProjectionExpression: "#emp, first_name, #last ",
		RangeExp:             "#emp = :val1",
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":val1": {N: aws.String("3")},
			":last": {S: aws.String("Trentor")},
		},
		FilterExp: "last_name = :last",
		Select:    "ALL",
	}

	//all attributes
	queryTestCase16 = models.Query{
		TableName: "employee",
		ExpressionAttributeNames: map[string]string{
			"#last": "last_name",
			"#emp":  "emp_id",
		},
		ProjectionExpression: "#emp, first_name, #last ",
		RangeExp:             "#emp = :val1",
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":val1": {N: aws.String("3")},
			":last": {S: aws.String("Trentor")},
		},
		FilterExp:     "last_name = :last",
		Select:        "COUNT",
		SortAscending: true,
		Limit:         4,
	}

	queryTestCaseOutput1 = `{"Count":5,"Items":[{"address":{"S":"Shamli"},"age":{"N":"10"},"emp_id":{"N":"1"},"first_name":{"S":"Marc"},"last_name":{"S":"Richards"}},{"address":{"S":"Ney York"},"age":{"N":"20"},"emp_id":{"N":"2"},"first_name":{"S":"Catalina"},"last_name":{"S":"Smith"}},{"address":{"S":"Pune"},"age":{"N":"30"},"emp_id":{"N":"3"},"first_name":{"S":"Alice"},"last_name":{"S":"Trentor"}},{"address":{"S":"Silicon Valley"},"age":{"N":"40"},"emp_id":{"N":"4"},"first_name":{"S":"Lea"},"last_name":{"S":"Martin"}},{"address":{"S":"London"},"age":{"N":"50"},"emp_id":{"N":"5"},"first_name":{"S":"David"},"last_name":{"S":"Lomond"}}]}`

	queryTestCaseOutput2 = `{"Count":5,"Items":[{"emp_id":{"N":"1"},"first_name":{"S":"Marc"}},{"emp_id":{"N":"2"},"first_name":{"S":"Catalina"}},{"emp_id":{"N":"3"},"first_name":{"S":"Alice"}},{"emp_id":{"N":"4"},"first_name":{"S":"Lea"}},{"emp_id":{"N":"5"},"first_name":{"S":"David"}}]}`

	queryTestCaseOutput3 = `{"Count":5,"Items":[{"emp_id":{"N":"1"},"first_name":{"S":"Marc"},"last_name":{"S":"Richards"}},{"emp_id":{"N":"2"},"first_name":{"S":"Catalina"},"last_name":{"S":"Smith"}},{"emp_id":{"N":"3"},"first_name":{"S":"Alice"},"last_name":{"S":"Trentor"}},{"emp_id":{"N":"4"},"first_name":{"S":"Lea"},"last_name":{"S":"Martin"}},{"emp_id":{"N":"5"},"first_name":{"S":"David"},"last_name":{"S":"Lomond"}}]}`

	queryTestCaseOutput4 = `{"Count":1,"Items":[{"emp_id":{"N":"2"},"first_name":{"S":"Catalina"},"last_name":{"S":"Smith"}}]}`

	queryTestCaseOutput6 = `{"Count":1,"Items":[{"emp_id":{"N":"3"},"first_name":{"S":"Alice"},"last_name":{"S":"Trentor"}}]}`

	queryTestCaseOutput8 = `{"Count":1,"Items":[{"emp_id":{"N":"3"},"first_name":{"S":"Alice"},"last_name":{"S":"Trentor"}}]}`

	queryTestCaseOutput9 = `{"Count":1,"Items":[{"emp_id":{"N":"3"},"first_name":{"S":"Alice"},"last_name":{"S":"Trentor"}}]}`

	queryTestCaseOutput10 = `{"Count":5,"Items":[{"address":{"S":"Shamli"},"age":{"N":"10"},"emp_id":{"N":"1"},"first_name":{"S":"Marc"},"last_name":{"S":"Richards"}},{"address":{"S":"Ney York"},"age":{"N":"20"},"emp_id":{"N":"2"},"first_name":{"S":"Catalina"},"last_name":{"S":"Smith"}},{"address":{"S":"Pune"},"age":{"N":"30"},"emp_id":{"N":"3"},"first_name":{"S":"Alice"},"last_name":{"S":"Trentor"}},{"address":{"S":"Silicon Valley"},"age":{"N":"40"},"emp_id":{"N":"4"},"first_name":{"S":"Lea"},"last_name":{"S":"Martin"}},{"address":{"S":"London"},"age":{"N":"50"},"emp_id":{"N":"5"},"first_name":{"S":"David"},"last_name":{"S":"Lomond"}}]}`

	queryTestCaseOutput11 = `{"Count":4,"Items":[{"address":{"S":"Shamli"},"age":{"N":"10"},"emp_id":{"N":"1"},"first_name":{"S":"Marc"},"last_name":{"S":"Richards"}},{"address":{"S":"Ney York"},"age":{"N":"20"},"emp_id":{"N":"2"},"first_name":{"S":"Catalina"},"last_name":{"S":"Smith"}},{"address":{"S":"Pune"},"age":{"N":"30"},"emp_id":{"N":"3"},"first_name":{"S":"Alice"},"last_name":{"S":"Trentor"}},{"address":{"S":"Silicon Valley"},"age":{"N":"40"},"emp_id":{"N":"4"},"first_name":{"S":"Lea"},"last_name":{"S":"Martin"}}],"LastEvaluatedKey":{"emp_id":{"N":"4"},"offset":{"N":"4"}}}`

	queryTestCaseOutput12 = `{"Count":4,"Items":[{"address":{"S":"Shamli"},"age":{"N":"10"},"emp_id":{"N":"1"},"first_name":{"S":"Marc"},"last_name":{"S":"Richards"}},{"address":{"S":"Ney York"},"age":{"N":"20"},"emp_id":{"N":"2"},"first_name":{"S":"Catalina"},"last_name":{"S":"Smith"}},{"address":{"S":"Pune"},"age":{"N":"30"},"emp_id":{"N":"3"},"first_name":{"S":"Alice"},"last_name":{"S":"Trentor"}},{"address":{"S":"Silicon Valley"},"age":{"N":"40"},"emp_id":{"N":"4"},"first_name":{"S":"Lea"},"last_name":{"S":"Martin"}}],"LastEvaluatedKey":{"emp_id":{"N":"4"},"offset":{"N":"4"}}}`

	queryTestCaseOutput13 = `{"Count":5,"Items":[]}`

	queryTestCaseOutput14 = `{"Count":1,"Items":[]}`

	queryTestCaseOutput15 = `{"Count":1,"Items":[{"emp_id":{"N":"3"},"first_name":{"S":"Alice"},"last_name":{"S":"Trentor"}}]}`

	queryTestCaseOutput16 = `{"Count":1,"Items":[]}`
)

// Test Data for Scan API
var (
	ScanTestCase1Name = "1: Wrong URL"
	ScanTestCase1     = models.ScanMeta{
		TableName: "employee",
	}

	ScanTestCase2Name = "2: Only Table Name passed"
	ScanTestCase2     = models.ScanMeta{
		TableName: "employee",
	}
	ScanTestCase2Output = `{"Count":5,"Items":[{"address":{"S":"Shamli"},"age":{"N":"10"},"emp_id":{"N":"1"},"first_name":{"S":"Marc"},"last_name":{"S":"Richards"}},{"address":{"S":"Ney York"},"age":{"N":"20"},"emp_id":{"N":"2"},"first_name":{"S":"Catalina"},"last_name":{"S":"Smith"}},{"address":{"S":"Pune"},"age":{"N":"30"},"emp_id":{"N":"3"},"first_name":{"S":"Alice"},"last_name":{"S":"Trentor"}},{"address":{"S":"Silicon Valley"},"age":{"N":"40"},"emp_id":{"N":"4"},"first_name":{"S":"Lea"},"last_name":{"S":"Martin"}},{"address":{"S":"London"},"age":{"N":"50"},"emp_id":{"N":"5"},"first_name":{"S":"David"},"last_name":{"S":"Lomond"}}]}`

	ScanTestCase3Name = "3: With Limit Attribute"
	ScanTestCase3     = models.ScanMeta{
		TableName: "employee",
		Limit:     3,
	}
	ScanTestCase3Output = `{"Count":3,"Items":[{"address":{"S":"Shamli"},"age":{"N":"10"},"emp_id":{"N":"1"},"first_name":{"S":"Marc"},"last_name":{"S":"Richards"}},{"address":{"S":"Ney York"},"age":{"N":"20"},"emp_id":{"N":"2"},"first_name":{"S":"Catalina"},"last_name":{"S":"Smith"}},{"address":{"S":"Pune"},"age":{"N":"30"},"emp_id":{"N":"3"},"first_name":{"S":"Alice"},"last_name":{"S":"Trentor"}}],"LastEvaluatedKey":{"emp_id":{"N":"3"},"offset":{"N":"3"}}}`

	ScanTestCase4Name = "4: With Projection Expression"
	ScanTestCase4     = models.ScanMeta{
		TableName:            "employee",
		ProjectionExpression: "address, emp_id, first_name",
	}
	ScanTestCase4Output = `{"Count":5,"Items":[{"address":{"S":"Shamli"},"emp_id":{"N":"1"},"first_name":{"S":"Marc"}},{"address":{"S":"Ney York"},"emp_id":{"N":"2"},"first_name":{"S":"Catalina"}},{"address":{"S":"Pune"},"emp_id":{"N":"3"},"first_name":{"S":"Alice"}},{"address":{"S":"Silicon Valley"},"emp_id":{"N":"4"},"first_name":{"S":"Lea"}},{"address":{"S":"London"},"emp_id":{"N":"5"},"first_name":{"S":"David"}}]}`

	ScanTestCase5Name = "5: With Projection Expression & limit"
	ScanTestCase5     = models.ScanMeta{
		TableName:            "employee",
		Limit:                3,
		ProjectionExpression: "address, emp_id, first_name",
	}
	ScanTestCase5Output = `{"Count":3,"Items":[{"address":{"S":"Shamli"},"emp_id":{"N":"1"},"first_name":{"S":"Marc"}},{"address":{"S":"Ney York"},"emp_id":{"N":"2"},"first_name":{"S":"Catalina"}},{"address":{"S":"Pune"},"emp_id":{"N":"3"},"first_name":{"S":"Alice"}}],"LastEvaluatedKey":{"emp_id":{"N":"3"},"offset":{"N":"3"}}}`

	ScanTestCase6Name = "6: Projection Expression without ExpressionAttributeNames"
	ScanTestCase6     = models.ScanMeta{
		TableName: "employee",
		Limit:     3,
		ExclusiveStartKey: map[string]*dynamodb.AttributeValue{
			"emp_id": {N: aws.String("4")},
			"offset": {N: aws.String("3")},
		},
		ProjectionExpression: "address, #ag, emp_id, first_name, last_name",
	}
	ScanTestCase6Output = `{"Count":2,"Items":[{"address":{"S":"Silicon Valley"},"emp_id":{"N":"4"},"first_name":{"S":"Lea"},"last_name":{"S":"Martin"}},{"address":{"S":"London"},"emp_id":{"N":"5"},"first_name":{"S":"David"},"last_name":{"S":"Lomond"}}]}`

	ScanTestCase7Name = "7: Projection Expression with ExpressionAttributeNames"
	ScanTestCase7     = models.ScanMeta{
		TableName:                "employee",
		ExpressionAttributeNames: map[string]string{"#ag": "age"},
		Limit:                    3,
		ProjectionExpression:     "address, #ag, emp_id, first_name, last_name",
	}
	ScanTestCase7Output = `{"Count":3,"Items":[{"address":{"S":"Shamli"},"age":{"N":"10"},"emp_id":{"N":"1"},"first_name":{"S":"Marc"},"last_name":{"S":"Richards"}},{"address":{"S":"Ney York"},"age":{"N":"20"},"emp_id":{"N":"2"},"first_name":{"S":"Catalina"},"last_name":{"S":"Smith"}},{"address":{"S":"Pune"},"age":{"N":"30"},"emp_id":{"N":"3"},"first_name":{"S":"Alice"},"last_name":{"S":"Trentor"}}],"LastEvaluatedKey":{"emp_id":{"N":"3"},"offset":{"N":"3"}}}`

	//400 Bad request
	ScanTestCase8Name = "8: Filter Expression without ExpressionAttributeValues"
	ScanTestCase8     = models.ScanMeta{
		TableName: "employee",
		ExclusiveStartKey: map[string]*dynamodb.AttributeValue{
			"emp_id": {N: aws.String("4")},
			"offset": {N: aws.String("3")},
		},
		FilterExpression: "age > :val1",
	}

	ScanTestCase9Name = "9: Filter Expression with ExpressionAttributeValues"
	ScanTestCase9     = models.ScanMeta{
		TableName: "employee",
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":val1": {N: aws.String("10")},
		},
		FilterExpression: "age > :val1",
	}
	ScanTestCase9Output = `{"Count":4,"Items":[{"address":{"S":"Ney York"},"age":{"N":"20"},"emp_id":{"N":"2"},"first_name":{"S":"Catalina"},"last_name":{"S":"Smith"}},{"address":{"S":"Pune"},"age":{"N":"30"},"emp_id":{"N":"3"},"first_name":{"S":"Alice"},"last_name":{"S":"Trentor"}},{"address":{"S":"Silicon Valley"},"age":{"N":"40"},"emp_id":{"N":"4"},"first_name":{"S":"Lea"},"last_name":{"S":"Martin"}},{"address":{"S":"London"},"age":{"N":"50"},"emp_id":{"N":"5"},"first_name":{"S":"David"},"last_name":{"S":"Lomond"}}]}`

	//400 bad request
	ScanTestCase10Name = "10: FilterExpression & ExpressionAttributeValues without ExpressionAttributeNames"
	ScanTestCase10     = models.ScanMeta{
		TableName: "employee",
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":val1": {N: aws.String("10")},
		},
		FilterExpression: "#ag > :val1",
	}

	ScanTestCase11Name = "11: FilterExpression & ExpressionAttributeValues with ExpressionAttributeNames"
	ScanTestCase11     = models.ScanMeta{
		TableName:                "employee",
		ExpressionAttributeNames: map[string]string{"#ag": "age"},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":val1": {N: aws.String("10")},
		},
		FilterExpression: "age > :val1",
	}
	ScanTestCase11Output = `{"Count":4,"Items":[{"address":{"S":"Ney York"},"age":{"N":"20"},"emp_id":{"N":"2"},"first_name":{"S":"Catalina"},"last_name":{"S":"Smith"}},{"address":{"S":"Pune"},"age":{"N":"30"},"emp_id":{"N":"3"},"first_name":{"S":"Alice"},"last_name":{"S":"Trentor"}},{"address":{"S":"Silicon Valley"},"age":{"N":"40"},"emp_id":{"N":"4"},"first_name":{"S":"Lea"},"last_name":{"S":"Martin"}},{"address":{"S":"London"},"age":{"N":"50"},"emp_id":{"N":"5"},"first_name":{"S":"David"},"last_name":{"S":"Lomond"}}]}`

	ScanTestCase12Name = "12: With ExclusiveStartKey"
	ScanTestCase12     = models.ScanMeta{
		TableName: "employee",
		ExclusiveStartKey: map[string]*dynamodb.AttributeValue{
			"emp_id": {N: aws.String("4")},
			"offset": {N: aws.String("3")},
		},
		Limit: 3,
	}
	ScanTestCase12Output = `{"Count":2,"Items":[{"address":{"S":"Silicon Valley"},"age":{"N":"40"},"emp_id":{"N":"4"},"first_name":{"S":"Lea"},"last_name":{"S":"Martin"}},{"address":{"S":"London"},"age":{"N":"50"},"emp_id":{"N":"5"},"first_name":{"S":"David"},"last_name":{"S":"Lomond"}}]}`

	ScanTestCase13Name = "13: With Count"
	ScanTestCase13     = models.ScanMeta{
		TableName: "employee",
		Limit:     3,
		Select:    "COUNT",
	}
	ScanTestCase13Output = `{"Count":5,"Items":[]}`
)

// Test Data for UpdateItem API
var (

	//200 Status check
	UpdateItemTestCase1Name = "1: Only TableName passed"
	UpdateItemTestCase1     = models.UpdateAttr{
		TableName: "employee",
	}

	UpdateItemTestCase2Name = "2: Update Expression with ExpressionAttributeValues"
	UpdateItemTestCase2     = models.UpdateAttr{
		TableName: "employee",
		Key: map[string]*dynamodb.AttributeValue{
			"emp_id": {N: aws.String("1")},
		},
		UpdateExpression: "SET age = :age",
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":age": {N: aws.String("10")},
		},
		ReturnValues: "ALL_NEW",
	}
	UpdateItemTestCase2Output = `{"Attributes":{"address":{"S":"Shamli"},"age":{"N":"10"},"emp_id":{"N":"1"},"first_name":{"S":"Marc"},"last_name":{"S":"Richards"}}}`

	UpdateItemTestCase3Name = "2: UpdateExpression, ExpressionAttributeValues with ExpressionAttributeNames"
	UpdateItemTestCase3     = models.UpdateAttr{
		TableName: "employee",
		Key: map[string]*dynamodb.AttributeValue{
			"emp_id": {N: aws.String("1")},
		},
		UpdateExpression: "SET #ag = :age",
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":age": {N: aws.String("10")},
		},
		ExpressionAttributeNames: map[string]string{
			"#ag": "age",
		},
	}
	UpdateItemTestCase3Output = `{"Attributes":{"address":{"S":"Shamli"},"age":{"N":"10"},"emp_id":{"N":"1"},"first_name":{"S":"Marc"},"last_name":{"S":"Richards"}}}`

	UpdateItemTestCase4Name = "4: Update Expression without ExpressionAttributeValues"
	UpdateItemTestCase4     = models.UpdateAttr{
		TableName: "employee",
		Key: map[string]*dynamodb.AttributeValue{
			"emp_id": {N: aws.String("1")},
		},
		UpdateExpression: "SET age = :age",
	}

	//400 bad request
	UpdateItemTestCase5Name = "5: UpdateExpression,ExpressionAttributeValues without Key"
	UpdateItemTestCase5     = models.UpdateAttr{
		TableName:        "employee",
		UpdateExpression: "SET age = :age",
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":age": {N: aws.String("10")},
		},
	}

	UpdateItemTestCase6Name = "6: UpdateExpression,ExpressionAttributeValues without ExpressionAttributeNames"
	UpdateItemTestCase6     = models.UpdateAttr{
		TableName: "employee",
		Key: map[string]*dynamodb.AttributeValue{
			"emp_id": {N: aws.String("1")},
		},
		UpdateExpression: "SET age = :age",
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":age": {N: aws.String("10")},
		},
	}

	UpdateItemTestCase7Name = "7: Correct ConditionExpression "
	UpdateItemTestCase7     = models.UpdateAttr{
		TableName: "employee",
		Key: map[string]*dynamodb.AttributeValue{
			"emp_id": {N: aws.String("1")},
		},
		ConditionExpression: "#ag > :val2",
		UpdateExpression:    "SET age = :age",
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":age":  {N: aws.String("10")},
			":val2": {N: aws.String("9")},
		},
		ExpressionAttributeNames: map[string]string{
			"#ag": "age",
		},
	}
	UpdateItemTestCase7Output = `{"Attributes":{"address":{"S":"Shamli"},"age":{"N":"10"},"emp_id":{"N":"1"},"first_name":{"S":"Marc"},"last_name":{"S":"Richards"}}}`

	//400 bad request
	UpdateItemTestCase8Name = "8: Wrong ConditionExpression"
	UpdateItemTestCase8     = models.UpdateAttr{
		TableName: "employee",
		Key: map[string]*dynamodb.AttributeValue{
			"emp_id": {N: aws.String("1")},
		},
		ConditionExpression: "#ag < :val2",
		UpdateExpression:    "SET age = :age",
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":age":  {N: aws.String("10")},
			":val2": {N: aws.String("9")},
		},
		ExpressionAttributeNames: map[string]string{
			"#ag": "age",
		},
	}

	//400 Bad request
	UpdateItemTestCase9Name = "9: ConditionExpression without ExpressionAttributeValues value"
	UpdateItemTestCase9     = models.UpdateAttr{
		TableName: "employee",
		Key: map[string]*dynamodb.AttributeValue{
			"emp_id": {N: aws.String("1")},
		},
		ConditionExpression: "#ag < :val2",
		UpdateExpression:    "SET age = :age",
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":age": {N: aws.String("10")},
		},
		ExpressionAttributeNames: map[string]string{
			"#ag": "age",
		},
	}

	//400 bad request
	UpdateItemTestCase10Name = "10: ConditionExpression without ExpressionAttributeNames value"
	UpdateItemTestCase10     = models.UpdateAttr{
		TableName: "employee",
		Key: map[string]*dynamodb.AttributeValue{
			"emp_id": {N: aws.String("1")},
		},
		ConditionExpression: "#ag < :val2",
		UpdateExpression:    "SET age = :age",
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":age":  {N: aws.String("10")},
			":val2": {N: aws.String("9")},
		},
	}
)

// Test Data for PutItem API
var (
	//400 bad request
	PutItemTestCase1Name = "1: only tablename passed"
	PutItemTestCase1     = models.Meta{
		TableName: "employe",
	}

	PutItemTestCase2Name = "2: Item Value to be updated"
	PutItemTestCase2     = models.Meta{
		TableName: "employee",
		Item: map[string]*dynamodb.AttributeValue{
			"emp_id": {N: aws.String("1")},
			"age":    {N: aws.String("11")},
		},
	}
	PutItemTestCase2Output = `{"Attributes":{"address":{"S":"Shamli"},"age":{"N":"10"},"emp_id":{"N":"1"},"first_name":{"S":"Marc"},"last_name":{"S":"Richards"}}}`

	PutItemTestCase3Name = "3: ConditionExpression with ExpressionAttributeValues & ExpressionAttributeNames"
	PutItemTestCase3     = models.Meta{
		TableName: "employee",
		Item: map[string]*dynamodb.AttributeValue{
			"emp_id": {N: aws.String("1")},
			"age":    {N: aws.String("10")},
		},
		ConditionExpression: "#ag > :val2",
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":val2": {N: aws.String("10")},
		},
		ExpressionAttributeNames: map[string]string{
			"#ag": "age",
		},
	}
	PutItemTestCase3Output = `{"Attributes":{"address":{"S":"Shamli"},"age":{"N":"11"},"emp_id":{"N":"1"},"first_name":{"S":"Marc"},"last_name":{"S":"Richards"}}}`

	PutItemTestCase4Name = "4: ConditionExpression with ExpressionAttributeValues"
	PutItemTestCase4     = models.Meta{
		TableName: "employee",
		Item: map[string]*dynamodb.AttributeValue{
			"emp_id": {N: aws.String("1")},
			"age":    {N: aws.String("11")},
		},
		ConditionExpression: "age > :val2",
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":val2": {N: aws.String("9")},
		},
	}
	PutItemTestCase4Output = `{"Attributes":{"address":{"S":"Shamli"},"age":{"N":"10"},"emp_id":{"N":"1"},"first_name":{"S":"Marc"},"last_name":{"S":"Richards"}}}`

	//400 bad request
	PutItemTestCase5Name = "5: ConditionExpression without ExpressionAttributeValues"
	PutItemTestCase5     = models.Meta{
		TableName: "employee",
		Item: map[string]*dynamodb.AttributeValue{
			"emp_id": {N: aws.String("1")},
			"age":    {N: aws.String("10")},
		},
		ConditionExpression: "#ag > :val2",
		ExpressionAttributeNames: map[string]string{
			"#ag": "age",
		},
	}

	//400 bad request
	PutItemTestCase6Name = "6: ConditionExpression without ExpressionAttributeNames"
	PutItemTestCase6     = models.Meta{
		TableName: "employee",
		Item: map[string]*dynamodb.AttributeValue{
			"emp_id": {N: aws.String("1")},
			"age":    {N: aws.String("10")},
		},
		ConditionExpression: "#ag > :val2",
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":val2": {N: aws.String("9")},
		},
	}

	//400 bad request
	PutItemTestCase7Name = "7: ConditionExpression without ExpressionAttributeValues & ExpressionAttributeNames "
	PutItemTestCase7     = models.Meta{
		TableName: "employee",
		Item: map[string]*dynamodb.AttributeValue{
			"emp_id": {N: aws.String("1")},
		},
		ConditionExpression: "age > 9",
	}

	//400 bad request
	PutItemTestCase8Name = "Item is not present"
	PutItemTestCase8     = models.Meta{
		TableName:           "employee",
		ConditionExpression: "#ag > :val2",
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":age":  {N: aws.String("10")},
			":val2": {N: aws.String("9")},
		},
		ExpressionAttributeNames: map[string]string{
			"#ag": "age",
		},
	}

	PutItemTestCase9Name = "9: Changing the values to initial state"
	PutItemTestCase9     = models.Meta{
		TableName: "employee",
		Item: map[string]*dynamodb.AttributeValue{
			"emp_id": {N: aws.String("1")},
			"age":    {N: aws.String("10")},
		},
	}
)

// Test Data DeleteItem API
var (
	DeleteItemTestCase1Name = "1: Only TableName passed"
	DeleteItemTestCase1     = models.Delete{
		TableName: "employee",
	}

	DeleteItemTestCase2Name = "2: Correct Key passed"
	DeleteItemTestCase2     = models.Delete{
		TableName: "employee",
		Key: map[string]*dynamodb.AttributeValue{
			"emp_id": {N: aws.String("2")},
		},
	}
	DeleteItemTestCase2Output = `{"Attributes":{"address":{"S":"Ney York"},"age":{"N":"20"},"emp_id":{"N":"2"},"first_name":{"S":"Catalina"},"last_name":{"S":"Smith"}}}`

	DeleteItemTestCase3Name = "3: Icorrect Key passed"
	DeleteItemTestCase3     = models.Delete{
		TableName: "employee",
		Key: map[string]*dynamodb.AttributeValue{
			"emp_id": {N: aws.String("2")},
		},
	}

	DeleteItemTestCase4Name = "4: ConditionExpression with ExpressionAttributeValues"
	DeleteItemTestCase4     = models.Delete{
		TableName: "employee",
		Key: map[string]*dynamodb.AttributeValue{
			"emp_id": {N: aws.String("3")},
		},
		ConditionExpression: "age > :val2",
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":val2": {N: aws.String("9")},
		},
	}
	DeleteItemTestCase4Output = `{"Attributes":{"address":{"S":"Pune"},"age":{"N":"30"},"emp_id":{"N":"3"},"first_name":{"S":"Alice"},"last_name":{"S":"Trentor"}}}`

	DeleteItemTestCase5Name = "5: ConditionExpressionNames with ExpressionAttributeNames & ExpressionAttributeValues"
	DeleteItemTestCase5     = models.Delete{
		TableName: "employee",
		Key: map[string]*dynamodb.AttributeValue{
			"emp_id": {N: aws.String("4")},
		},
		ConditionExpression: "#ag > :val2",
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":val2": {N: aws.String("19")},
		},
		ExpressionAttributeNames: map[string]string{
			"#ag": "age",
		},
	}
	DeleteItemTestCase5Output = `{"Attributes":{"address":{"S":"Silicon Valley"},"age":{"N":"40"},"emp_id":{"N":"4"},"first_name":{"S":"Lea"},"last_name":{"S":"Martin"}}}`

	DeleteItemTestCase6Name = "6: ConditionExpressionNames without ExpressionAttributeValues"
	DeleteItemTestCase6     = models.Delete{
		TableName: "employee",
		Key: map[string]*dynamodb.AttributeValue{
			"emp_id": {N: aws.String("2")},
		},
		ConditionExpression: "#ag > :val2",
		ExpressionAttributeNames: map[string]string{
			"#ag": "age",
		},
	}

	DeleteItemTestCase7Name = "7: ConditionExpressionNames without ExpressionAttributeNames"
	DeleteItemTestCase7     = models.Delete{
		TableName: "employee",
		Key: map[string]*dynamodb.AttributeValue{
			"emp_id": {N: aws.String("2")},
		},
		ConditionExpression: "#ag > :val2",
		ExpressionAttributeNames: map[string]string{
			"#ag": "age",
		},
	}

	DeleteItemTestCase8Name = "8: ConditionExpressionNames with ExpressionAttributeNames & ExpressionAttributeValue"
	DeleteItemTestCase8     = models.Delete{
		TableName: "employee",
		Key: map[string]*dynamodb.AttributeValue{
			"emp_id": {N: aws.String("2")},
		},
		ConditionExpression: "#ag > :val2",
	}
)

// test Data for BatchWriteItem API
var (
	BatchWriteItemTestCase1Name = "1: Only Table name passed"
	BatchWriteItemTestCase1     = models.BatchWriteItem{
		RequestItems: map[string][]models.BatchWriteSubItems{
			"employee": {},
		},
	}

	BatchWriteItemTestCase2Name = "2: Batch Put Request for one table"
	BatchWriteItemTestCase2     = models.BatchWriteItem{
		RequestItems: map[string][]models.BatchWriteSubItems{
			"employee": {
				{
					PutReq: models.BatchPutItem{
						Item: map[string]*dynamodb.AttributeValue{
							"emp_id":     {N: aws.String("6")},
							"age":        {N: aws.String("60")},
							"address":    {S: aws.String("London")},
							"first_name": {S: aws.String("David")},
							"last_name":  {S: aws.String("Root")},
						},
					},
				},
				{
					PutReq: models.BatchPutItem{
						Item: map[string]*dynamodb.AttributeValue{
							"emp_id":     {N: aws.String("7")},
							"age":        {N: aws.String("70")},
							"address":    {S: aws.String("Paris")},
							"first_name": {S: aws.String("Marc")},
							"last_name":  {S: aws.String("Ponting")},
						},
					},
				},
			},
		},
	}

	BatchWriteItemTestCase3Name = "3: Batch Delete Request for one Table"
	BatchWriteItemTestCase3     = models.BatchWriteItem{
		RequestItems: map[string][]models.BatchWriteSubItems{
			"employee": {
				{
					DelReq: models.BatchDeleteItem{
						Key: map[string]*dynamodb.AttributeValue{
							"emp_id": {N: aws.String("6")},
						},
					},
				},
				{
					DelReq: models.BatchDeleteItem{
						Key: map[string]*dynamodb.AttributeValue{
							"emp_id": {N: aws.String("7")},
						},
					},
				},
			},
		},
	}

	BatchWriteItemTestCase4Name = "4: Batch Delete Request for one table for those keys which are not present"
	BatchWriteItemTestCase4     = models.BatchWriteItem{
		RequestItems: map[string][]models.BatchWriteSubItems{
			"employee": {
				{
					DelReq: models.BatchDeleteItem{
						Key: map[string]*dynamodb.AttributeValue{
							"emp_id": {N: aws.String("6")},
						},
					},
				},
				{
					DelReq: models.BatchDeleteItem{
						Key: map[string]*dynamodb.AttributeValue{
							"emp_id": {N: aws.String("7")},
						},
					},
				},
			},
		},
	}

	BatchWriteItemTestCase5Name = "5: Batch Put Request for 2 tables"
	BatchWriteItemTestCase5     = models.BatchWriteItem{
		RequestItems: map[string][]models.BatchWriteSubItems{
			"employee": {
				{
					PutReq: models.BatchPutItem{
						Item: map[string]*dynamodb.AttributeValue{
							"emp_id":     {N: aws.String("6")},
							"age":        {N: aws.String("60")},
							"address":    {S: aws.String("London")},
							"first_name": {S: aws.String("David")},
							"last_name":  {S: aws.String("Root")},
						},
					},
				},
				{
					PutReq: models.BatchPutItem{
						Item: map[string]*dynamodb.AttributeValue{
							"emp_id":     {N: aws.String("7")},
							"age":        {N: aws.String("70")},
							"address":    {S: aws.String("Paris")},
							"first_name": {S: aws.String("Marc")},
							"last_name":  {S: aws.String("Ponting")},
						},
					},
				},
			},
			"department": {
				{
					PutReq: models.BatchPutItem{
						Item: map[string]*dynamodb.AttributeValue{
							"d_id":             {N: aws.String("400")},
							"d_name":           {S: aws.String("Sports")},
							"d_specialization": {S: aws.String("Cricket")},
						},
					},
				},
				{
					PutReq: models.BatchPutItem{
						Item: map[string]*dynamodb.AttributeValue{
							"d_id":             {N: aws.String("500")},
							"d_name":           {S: aws.String("Welfare")},
							"d_specialization": {S: aws.String("Students")},
						},
					},
				},
			},
		},
	}

	BatchWriteItemTestCase6Name = "6: Batch Delete Request for 2 tables"
	BatchWriteItemTestCase6     = models.BatchWriteItem{
		RequestItems: map[string][]models.BatchWriteSubItems{
			"employee": {
				{
					DelReq: models.BatchDeleteItem{
						Key: map[string]*dynamodb.AttributeValue{
							"emp_id": {N: aws.String("6")},
						},
					},
				},
				{
					DelReq: models.BatchDeleteItem{
						Key: map[string]*dynamodb.AttributeValue{
							"emp_id": {N: aws.String("7")},
						},
					},
				},
			},
			"department": {
				{
					DelReq: models.BatchDeleteItem{
						Key: map[string]*dynamodb.AttributeValue{
							"d_id": {N: aws.String("400")},
						},
					},
				},
				{
					DelReq: models.BatchDeleteItem{
						Key: map[string]*dynamodb.AttributeValue{
							"d_id": {N: aws.String("500")},
						},
					},
				},
			},
		},
	}

	BatchWriteItemTestCase7Name = "7: Batch Put & Delete Request for 1 table"
	BatchWriteItemTestCase7     = models.BatchWriteItem{
		RequestItems: map[string][]models.BatchWriteSubItems{
			"employee": {
				{
					PutReq: models.BatchPutItem{
						Item: map[string]*dynamodb.AttributeValue{
							"emp_id":     {N: aws.String("6")},
							"age":        {N: aws.String("60")},
							"address":    {S: aws.String("London")},
							"first_name": {S: aws.String("David")},
							"last_name":  {S: aws.String("Root")},
						},
					},
				},
				{
					PutReq: models.BatchPutItem{
						Item: map[string]*dynamodb.AttributeValue{
							"emp_id":     {N: aws.String("7")},
							"age":        {N: aws.String("70")},
							"address":    {S: aws.String("Paris")},
							"first_name": {S: aws.String("Marc")},
							"last_name":  {S: aws.String("Ponting")},
						},
					},
				},
				{
					DelReq: models.BatchDeleteItem{
						Key: map[string]*dynamodb.AttributeValue{
							"emp_id": {N: aws.String("6")},
						},
					},
				},
				{
					DelReq: models.BatchDeleteItem{
						Key: map[string]*dynamodb.AttributeValue{
							"emp_id": {N: aws.String("7")},
						},
					},
				},
			},
			"department": {
				{
					PutReq: models.BatchPutItem{
						Item: map[string]*dynamodb.AttributeValue{
							"d_id":             {N: aws.String("400")},
							"d_name":           {S: aws.String("Sports")},
							"d_specialization": {S: aws.String("Cricket")},
						},
					},
				},
				{
					PutReq: models.BatchPutItem{
						Item: map[string]*dynamodb.AttributeValue{
							"d_id":             {N: aws.String("500")},
							"d_name":           {S: aws.String("Welfare")},
							"d_specialization": {S: aws.String("Students")},
						},
					},
				},
				{
					DelReq: models.BatchDeleteItem{
						Key: map[string]*dynamodb.AttributeValue{
							"d_id": {N: aws.String("400")},
						},
					},
				},
				{
					DelReq: models.BatchDeleteItem{
						Key: map[string]*dynamodb.AttributeValue{
							"d_id": {N: aws.String("500")},
						},
					},
				},
			},
		},
	}

	BatchWriteItemTestCase8Name = "8: Batch Put & Delete Request for 2 table"
	BatchWriteItemTestCase8     = models.BatchWriteItem{
		RequestItems: map[string][]models.BatchWriteSubItems{
			"employee": {
				{
					PutReq: models.BatchPutItem{
						Item: map[string]*dynamodb.AttributeValue{
							"emp_id":     {N: aws.String("6")},
							"age":        {N: aws.String("60")},
							"address":    {S: aws.String("London")},
							"first_name": {S: aws.String("David")},
							"last_name":  {S: aws.String("Root")},
						},
					},
				},
				{
					PutReq: models.BatchPutItem{
						Item: map[string]*dynamodb.AttributeValue{
							"emp_id":     {N: aws.String("7")},
							"age":        {N: aws.String("70")},
							"address":    {S: aws.String("Paris")},
							"first_name": {S: aws.String("Marc")},
							"last_name":  {S: aws.String("Ponting")},
						},
					},
				},
				{
					DelReq: models.BatchDeleteItem{
						Key: map[string]*dynamodb.AttributeValue{
							"emp_id": {N: aws.String("6")},
						},
					},
				},
				{
					DelReq: models.BatchDeleteItem{
						Key: map[string]*dynamodb.AttributeValue{
							"emp_id": {N: aws.String("7")},
						},
					},
				},
			},
			"department": {
				{
					PutReq: models.BatchPutItem{
						Item: map[string]*dynamodb.AttributeValue{
							"d_id":             {N: aws.String("400")},
							"d_name":           {S: aws.String("Sports")},
							"d_specialization": {S: aws.String("Cricket")},
						},
					},
				},
				{
					PutReq: models.BatchPutItem{
						Item: map[string]*dynamodb.AttributeValue{
							"d_id":             {N: aws.String("500")},
							"d_name":           {S: aws.String("Welfare")},
							"d_specialization": {S: aws.String("Students")},
						},
					},
				},
				{
					DelReq: models.BatchDeleteItem{
						Key: map[string]*dynamodb.AttributeValue{
							"d_id": {N: aws.String("400")},
						},
					},
				},
				{
					DelReq: models.BatchDeleteItem{
						Key: map[string]*dynamodb.AttributeValue{
							"d_id": {N: aws.String("500")},
						},
					},
				},
			},
		},
	}

	BatchWriteItemTestCase9Name = "9: Batch Put Request for wrong Table"
	BatchWriteItemTestCase9     = models.BatchWriteItem{
		RequestItems: map[string][]models.BatchWriteSubItems{
			"employee1": {
				{
					PutReq: models.BatchPutItem{
						Item: map[string]*dynamodb.AttributeValue{
							"emp_id":     {N: aws.String("6")},
							"age":        {N: aws.String("60")},
							"address":    {S: aws.String("London")},
							"first_name": {S: aws.String("David")},
							"last_name":  {S: aws.String("Root")},
						},
					},
				},
				{
					PutReq: models.BatchPutItem{
						Item: map[string]*dynamodb.AttributeValue{
							"emp_id":     {N: aws.String("7")},
							"age":        {N: aws.String("70")},
							"address":    {S: aws.String("Paris")},
							"first_name": {S: aws.String("Marc")},
							"last_name":  {S: aws.String("Ponting")},
						},
					},
				},
			},
		},
	}

	BatchWriteItemTestCase10Name = "10: Batch Delete Request for wrong table"
	BatchWriteItemTestCase10     = models.BatchWriteItem{
		RequestItems: map[string][]models.BatchWriteSubItems{
			"employee1": {
				{
					DelReq: models.BatchDeleteItem{
						Key: map[string]*dynamodb.AttributeValue{
							"emp_id": {N: aws.String("6")},
						},
					},
				},
				{
					DelReq: models.BatchDeleteItem{
						Key: map[string]*dynamodb.AttributeValue{
							"emp_id": {N: aws.String("7")},
						},
					},
				},
			},
		},
	}
)

func handlerInitFunc() *gin.Engine {
	initErr := initializer.InitAll()
	if initErr != nil {
		log.Fatalln(initErr)
	}
	r := gin.Default()
	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "Server is up and running!",
		})
	})
	r.NoRoute(func(c *gin.Context) {
		c.JSON(404, gin.H{"code": "RouteNotFound"})
	})
	api.InitAPI(r)
	return r
}

func createPostTestCase(name, url, dynamoAction, outputString string, input interface{}) apitesting.APITestCase {
	return apitesting.APITestCase{
		Name:    name,
		ReqType: "POST",
		PopulateHeaders: func(ctx context.Context, t *testing.T) map[string]string {
			return map[string]string{
				"Content-Type": "application/json",
				"X-Amz-Target": "DynamoDB_20120810." + dynamoAction,
			}
		},
		ResourcePath: func(ctx context.Context, t *testing.T) string { return url },
		PopulateJSON: func(ctx context.Context, t *testing.T) interface{} {
			return input
		},
		ExpHTTPStatus: http.StatusOK,
		ValidateResponse: func(ctx context.Context, t *testing.T, resp *httpexpect.Response) context.Context {
			resp.Body().Equal(outputString)
			return ctx
		},
	}
}

func createStatusCheckPostTestCase(name, url, dynamoAction string, httpStatus int, input interface{}) apitesting.APITestCase {
	return apitesting.APITestCase{
		Name:    name,
		ReqType: "POST",
		PopulateHeaders: func(ctx context.Context, t *testing.T) map[string]string {
			return map[string]string{
				"Content-Type": "application/json",
				"X-Amz-Target": "DynamoDB_20120810." + dynamoAction,
			}
		},
		ResourcePath: func(ctx context.Context, t *testing.T) string { return url },
		PopulateJSON: func(ctx context.Context, t *testing.T) interface{} {
			return input
		},
		ExpHTTPStatus: httpStatus,
	}
}

func init() {
	config, err := LoadConfig("../config.yaml")
	if err != nil {
		log.Fatalf("Error loading configuration: %v", err)
	}
	// Build the Spanner database name
	databaseName = fmt.Sprintf(
		"projects/%s/instances/%s/databases/%s",
		config.Spanner.ProjectID, config.Spanner.InstanceID, config.Spanner.DatabaseName,
	)
}

func LoadConfig(filename string) (*models.Config, error) {
	data, err := readConfigFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config models.Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}

func testGetItemAPI(t *testing.T) {
	apitest := apitesting.APITest{
		// APIEndpointURL: apiURL + "/" + version,
		GetHTTPHandler: func(ctx context.Context, t *testing.T) http.Handler {
			return handlerInitFunc()
		},
	}
	tests := []apitesting.APITestCase{
		{
			Name:    "Wrong URL (404 Error)",
			ReqType: "POST",
			PopulateHeaders: func(ctx context.Context, t *testing.T) map[string]string {
				return map[string]string{
					"Content-Type": "application/json",
					"X-Amz-Target": "DynamoDB_20120810.GetItem",
				}
			},
			ResourcePath: func(ctx context.Context, t *testing.T) string { return "/v1/GetIte" },
			PopulateJSON: func(ctx context.Context, t *testing.T) interface{} {
				return getItemTest2
			},
			ExpHTTPStatus: http.StatusNotFound,
		},
		{
			Name:    "Wrong Parameter(Bad Request)",
			ReqType: "POST",
			PopulateHeaders: func(ctx context.Context, t *testing.T) map[string]string {
				return map[string]string{
					"Content-Type": "application/json",
					"X-Amz-Target": "DynamoDB_20120810.GetItem",
				}
			},
			ResourcePath: func(ctx context.Context, t *testing.T) string { return "/v1" },
			PopulateJSON: func(ctx context.Context, t *testing.T) interface{} {
				return getItemTest1
			},
			ExpHTTPStatus: http.StatusBadRequest,
		},
		{
			Name:    "Wrong Parameter(Key value is not passed)",
			ReqType: "POST",
			PopulateHeaders: func(ctx context.Context, t *testing.T) map[string]string {
				return map[string]string{
					"Content-Type": "application/json",
					"X-Amz-Target": "DynamoDB_20120810.GetItem",
				}
			},
			ResourcePath: func(ctx context.Context, t *testing.T) string { return "/v1" },
			PopulateJSON: func(ctx context.Context, t *testing.T) interface{} {
				return getItemTest1_1
			},
			ExpHTTPStatus: http.StatusBadRequest,
		},
		createPostTestCase("Crorect Data TestCase", "/v1", "GetItem", getItemTest2Output, getItemTest2),
		createPostTestCase("Crorect data with Projection param Testcase", "/v1", "GetItem", getItemTest3Output, getItemTest3),
		createPostTestCase("Crorect data with  ExpressionAttributeNames Testcase", "/v1", "GetItem", getItemTest3Output, getItemTest4),
		createPostTestCase("Crorect data with  ExpressionAttributeNames values not passed Testcase", "/v1", "GetItem", getItemTest5Output, getItemTest5),
	}
	apitest.RunTests(t, tests)
}

func testGetBatchAPI(t *testing.T) {
	apitest := apitesting.APITest{
		GetHTTPHandler: func(ctx context.Context, t *testing.T) http.Handler {
			return handlerInitFunc()
		},
	}
	tests := []apitesting.APITestCase{
		{
			Name:    TestGetBatch1Name,
			ReqType: "POST",
			PopulateHeaders: func(ctx context.Context, t *testing.T) map[string]string {
				return map[string]string{
					"Content-Type": "application/json",
					"X-Amz-Target": "DynamoDB_20120810.BatchGetItem",
				}
			},
			ResourcePath: func(ctx context.Context, t *testing.T) string { return "/v1/BatchGetIt" },
			PopulateJSON: func(ctx context.Context, t *testing.T) interface{} {
				return TestGetBatch1
			},
			ExpHTTPStatus: http.StatusNotFound,
		},
		{
			Name:    TestGetBatch10Name,
			ReqType: "POST",
			PopulateHeaders: func(ctx context.Context, t *testing.T) map[string]string {
				return map[string]string{
					"Content-Type": "application/json",
					"X-Amz-Target": "DynamoDB_20120810.BatchGetItem",
				}
			},
			ResourcePath: func(ctx context.Context, t *testing.T) string { return "/v1" },
			PopulateJSON: func(ctx context.Context, t *testing.T) interface{} {
				return TestGetBatch10
			},
			ExpHTTPStatus: http.StatusBadRequest,
		},
		createPostTestCase(TestGetBatch2Name, "/v1", "BatchGetItem", TestGetBatch2Output, TestGetBatch2),
		createPostTestCase(TestGetBatch3Name, "/v1", "BatchGetItem", TestGetBatch3Output, TestGetBatch3),
		createPostTestCase(TestGetBatch4Name, "/v1", "BatchGetItem", TestGetBatch4Output, TestGetBatch4),
		createPostTestCase(TestGetBatch5Name, "/v1", "BatchGetItem", TestGetBatch5Output, TestGetBatch5),
		createPostTestCase(TestGetBatch6Name, "/v1", "BatchGetItem", TestGetBatch6Output, TestGetBatch6),
		createPostTestCase(TestGetBatch7Name, "/v1", "BatchGetItem", TestGetBatch7Output, TestGetBatch7),
		createPostTestCase(TestGetBatch8Name, "/v1", "BatchGetItem", TestGetBatch8Output, TestGetBatch8),
		createPostTestCase(TestGetBatch9Name, "/v1", "BatchGetItem", TestGetBatch9Output, TestGetBatch9),
	}
	apitest.RunTests(t, tests)
}

func testQueryAPI(t *testing.T) {
	apitest := apitesting.APITest{
		GetHTTPHandler: func(ctx context.Context, t *testing.T) http.Handler {
			return handlerInitFunc()
		},
	}
	tests := []apitesting.APITestCase{
		{
			Name:    "Wrong URL (404 Error)",
			ReqType: "POST",
			PopulateHeaders: func(ctx context.Context, t *testing.T) map[string]string {
				return map[string]string{
					"Content-Type": "application/json",
					"X-Amz-Target": "DynamoDB_20120810.Query",
				}
			},
			ResourcePath: func(ctx context.Context, t *testing.T) string { return "/v1/Quer" },
			PopulateJSON: func(ctx context.Context, t *testing.T) interface{} {
				return queryTestCase0
			},
			ExpHTTPStatus: http.StatusNotFound,
		},
		{
			Name:    "Wrong Parameter(Bad Request)",
			ReqType: "POST",
			PopulateHeaders: func(ctx context.Context, t *testing.T) map[string]string {
				return map[string]string{
					"Content-Type": "application/json",
					"X-Amz-Target": "DynamoDB_20120810.Query",
				}
			},
			ResourcePath: func(ctx context.Context, t *testing.T) string { return "/v1" },
			PopulateJSON: func(ctx context.Context, t *testing.T) interface{} {
				return queryTestCase0
			},
			ExpHTTPStatus: http.StatusBadRequest,
		},
		{
			Name:    "KeyconditionExpression without ExpressionAttributeValues",
			ReqType: "POST",
			PopulateHeaders: func(ctx context.Context, t *testing.T) map[string]string {
				return map[string]string{
					"Content-Type": "application/json",
					"X-Amz-Target": "DynamoDB_20120810.Query",
				}
			},
			ResourcePath: func(ctx context.Context, t *testing.T) string { return "/v1" },
			PopulateJSON: func(ctx context.Context, t *testing.T) interface{} {
				return queryTestCase5
			},
			ExpHTTPStatus: http.StatusBadRequest,
		},
		{
			Name:    "filter expression but value not present",
			ReqType: "POST",
			PopulateHeaders: func(ctx context.Context, t *testing.T) map[string]string {
				return map[string]string{
					"Content-Type": "application/json",
					"X-Amz-Target": "DynamoDB_20120810.Query",
				}
			},
			ResourcePath: func(ctx context.Context, t *testing.T) string { return "/v1" },
			PopulateJSON: func(ctx context.Context, t *testing.T) interface{} {
				return queryTestCase7
			},
			ExpHTTPStatus: http.StatusBadRequest,
		},
		createPostTestCase("Only table name passed", "/v1", "Query", queryTestCaseOutput1, queryTestCase1),
		createPostTestCase("table & projection Expression", "/v1", "Query", queryTestCaseOutput2, queryTestCase2),
		createPostTestCase("projection expression with ExpressionAttributeNames", "/v1", "Query", queryTestCaseOutput3, queryTestCase3),
		createPostTestCase("KeyconditionExpression ", "/v1", "Query", queryTestCaseOutput4, queryTestCase4),
		createPostTestCase("KeyconditionExpression & filterExperssion", "/v1", "Query", queryTestCaseOutput6, queryTestCase6),
		createPostTestCase("only filter expression", "/v1", "Query", queryTestCaseOutput8, queryTestCase8),
		createPostTestCase("with ScanIndexForward and other attributes", "/v1", "Query", queryTestCaseOutput9, queryTestCase9),
		createPostTestCase("with only ScanIndexForward ", "/v1", "Query", queryTestCaseOutput10, queryTestCase10),
		createPostTestCase("with Limit", "/v1", "Query", queryTestCaseOutput11, queryTestCase11),
		createPostTestCase("with Limit & ScanIndexForward", "/v1", "Query", queryTestCaseOutput12, queryTestCase12),
		createPostTestCase("only count", "/v1", "Query", queryTestCaseOutput13, queryTestCase13),
		createPostTestCase("count with other attributes present", "/v1", "Query", queryTestCaseOutput14, queryTestCase14),
		createPostTestCase("Select with other than count", "/v1", "Query", queryTestCaseOutput15, queryTestCase15),
		createPostTestCase("all attributes", "/v1", "Query", queryTestCaseOutput16, queryTestCase16),
	}
	apitest.RunTests(t, tests)
}

func testScanAPI(t *testing.T) {
	apitest := apitesting.APITest{
		GetHTTPHandler: func(ctx context.Context, t *testing.T) http.Handler {
			return handlerInitFunc()
		},
	}
	tests := []apitesting.APITestCase{
		{
			Name:    ScanTestCase1Name,
			ReqType: "POST",
			PopulateHeaders: func(ctx context.Context, t *testing.T) map[string]string {
				return map[string]string{
					"Content-Type": "application/json",
					"X-Amz-Target": "DynamoDB_20120810.Scan",
				}
			},
			ResourcePath: func(ctx context.Context, t *testing.T) string { return "/v1/Sca" },
			PopulateJSON: func(ctx context.Context, t *testing.T) interface{} {
				return ScanTestCase1
			},
			ExpHTTPStatus: http.StatusNotFound,
		},
		{
			Name:    ScanTestCase8Name,
			ReqType: "POST",
			PopulateHeaders: func(ctx context.Context, t *testing.T) map[string]string {
				return map[string]string{
					"Content-Type": "application/json",
					"X-Amz-Target": "DynamoDB_20120810.Scan",
				}
			},
			ResourcePath: func(ctx context.Context, t *testing.T) string { return "/v1" },
			PopulateJSON: func(ctx context.Context, t *testing.T) interface{} {
				return ScanTestCase8
			},
			ExpHTTPStatus: http.StatusBadRequest,
		},
		{
			Name:    ScanTestCase10Name,
			ReqType: "POST",
			PopulateHeaders: func(ctx context.Context, t *testing.T) map[string]string {
				return map[string]string{
					"Content-Type": "application/json",
					"X-Amz-Target": "DynamoDB_20120810.Scan",
				}
			},
			ResourcePath: func(ctx context.Context, t *testing.T) string { return "/v1" },
			PopulateJSON: func(ctx context.Context, t *testing.T) interface{} {
				return ScanTestCase10
			},
			ExpHTTPStatus: http.StatusBadRequest,
		},
		createPostTestCase(ScanTestCase2Name, "/v1", "Query", ScanTestCase2Output, ScanTestCase2),
		createPostTestCase(ScanTestCase3Name, "/v1", "Query", ScanTestCase3Output, ScanTestCase3),
		createPostTestCase(ScanTestCase4Name, "/v1", "Query", ScanTestCase4Output, ScanTestCase4),
		createPostTestCase(ScanTestCase5Name, "/v1", "Query", ScanTestCase5Output, ScanTestCase5),
		createPostTestCase(ScanTestCase6Name, "/v1", "Query", ScanTestCase6Output, ScanTestCase6),
		createPostTestCase(ScanTestCase7Name, "/v1", "Query", ScanTestCase7Output, ScanTestCase7),
		createPostTestCase(ScanTestCase9Name, "/v1", "Query", ScanTestCase9Output, ScanTestCase9),
		createPostTestCase(ScanTestCase11Name, "/v1", "Query", ScanTestCase11Output, ScanTestCase11),
		createPostTestCase(ScanTestCase12Name, "/v1", "Query", ScanTestCase12Output, ScanTestCase12),
		createPostTestCase(ScanTestCase13Name, "/v1", "Query", ScanTestCase13Output, ScanTestCase13),
	}
	apitest.RunTests(t, tests)
}

func testUpdateItemAPI(t *testing.T) {
	apitest := apitesting.APITest{
		GetHTTPHandler: func(ctx context.Context, t *testing.T) http.Handler {
			return handlerInitFunc()
		},
	}
	tests := []apitesting.APITestCase{
		createStatusCheckPostTestCase(UpdateItemTestCase1Name, "/v1", "UpdateItem", http.StatusOK, UpdateItemTestCase1),
		createStatusCheckPostTestCase(UpdateItemTestCase4Name, "/v1", "UpdateItem", http.StatusOK, UpdateItemTestCase4),
		createStatusCheckPostTestCase(UpdateItemTestCase6Name, "/v1", "UpdateItem", http.StatusOK, UpdateItemTestCase6),
		createStatusCheckPostTestCase(UpdateItemTestCase5Name, "/v1", "UpdateItem", http.StatusBadRequest, UpdateItemTestCase5),
		createStatusCheckPostTestCase(UpdateItemTestCase8Name, "/v1", "UpdateItem", http.StatusBadRequest, UpdateItemTestCase8),
		createStatusCheckPostTestCase(UpdateItemTestCase9Name, "/v1", "UpdateItem", http.StatusBadRequest, UpdateItemTestCase9),
		createStatusCheckPostTestCase(UpdateItemTestCase10Name, "/v1", "UpdateItem", http.StatusBadRequest, UpdateItemTestCase10),
		createPostTestCase(UpdateItemTestCase2Name, "/v1", "UpdateItem", UpdateItemTestCase2Output, UpdateItemTestCase2),
		createPostTestCase(UpdateItemTestCase3Name, "/v1", "UpdateItem", UpdateItemTestCase3Output, UpdateItemTestCase3),
		createPostTestCase(UpdateItemTestCase7Name, "/v1", "UpdateItem", UpdateItemTestCase7Output, UpdateItemTestCase7),
	}
	apitest.RunTests(t, tests)
}

func testPutItemAPI(t *testing.T) {
	apitest := apitesting.APITest{
		// APIEndpointURL: apiURL + "/" + version,
		GetHTTPHandler: func(ctx context.Context, t *testing.T) http.Handler {
			return handlerInitFunc()
		},
	}
	tests := []apitesting.APITestCase{
		createStatusCheckPostTestCase(PutItemTestCase1Name, "/v1", "PutItem", http.StatusBadRequest, PutItemTestCase1),
		createStatusCheckPostTestCase(PutItemTestCase5Name, "/v1", "PutItem", http.StatusBadRequest, PutItemTestCase5),
		createStatusCheckPostTestCase(PutItemTestCase6Name, "/v1", "PutItem", http.StatusBadRequest, PutItemTestCase6),
		createStatusCheckPostTestCase(PutItemTestCase7Name, "/v1", "PutItem", http.StatusBadRequest, PutItemTestCase7),
		createStatusCheckPostTestCase(PutItemTestCase8Name, "/v1", "PutItem", http.StatusBadRequest, PutItemTestCase8),
		createPostTestCase(PutItemTestCase2Name, "/v1", "PutItem", PutItemTestCase2Output, PutItemTestCase2),
		createPostTestCase(PutItemTestCase3Name, "/v1", "PutItem", PutItemTestCase3Output, PutItemTestCase3),
		createPostTestCase(PutItemTestCase4Name, "/v1", "PutItem", PutItemTestCase4Output, PutItemTestCase4),
		createStatusCheckPostTestCase(PutItemTestCase9Name, "/v1", "PutItem", http.StatusOK, PutItemTestCase9),
	}
	apitest.RunTests(t, tests)
}

func testDeleteItemAPI(t *testing.T) {
	apitest := apitesting.APITest{
		// APIEndpointURL: apiURL + "/" + version,
		GetHTTPHandler: func(ctx context.Context, t *testing.T) http.Handler {
			return handlerInitFunc()
		},
	}
	tests := []apitesting.APITestCase{
		createStatusCheckPostTestCase(DeleteItemTestCase1Name, "/v1", "DeleteItem", http.StatusBadRequest, DeleteItemTestCase1),
		createStatusCheckPostTestCase(DeleteItemTestCase6Name, "/v1", "DeleteItem", http.StatusBadRequest, DeleteItemTestCase6),
		createStatusCheckPostTestCase(DeleteItemTestCase7Name, "/v1", "DeleteItem", http.StatusBadRequest, DeleteItemTestCase7),
		createStatusCheckPostTestCase(DeleteItemTestCase8Name, "/v1", "DeleteItem", http.StatusBadRequest, DeleteItemTestCase8),
		createPostTestCase(DeleteItemTestCase2Name, "/v1", "DeleteItem", DeleteItemTestCase2Output, DeleteItemTestCase2),
		createStatusCheckPostTestCase(DeleteItemTestCase3Name, "/v1", "DeleteItem", http.StatusOK, DeleteItemTestCase3),
		createPostTestCase(DeleteItemTestCase4Name, "/v1", "DeleteItem", DeleteItemTestCase4Output, DeleteItemTestCase4),
		createPostTestCase(DeleteItemTestCase5Name, "/v1", "DeleteItem", DeleteItemTestCase5Output, DeleteItemTestCase5),
	}
	apitest.RunTests(t, tests)
}

func testBatchWriteItemAPI(t *testing.T) {
	apitest := apitesting.APITest{
		GetHTTPHandler: func(ctx context.Context, t *testing.T) http.Handler {
			return handlerInitFunc()
		},
	}
	tests := []apitesting.APITestCase{
		createStatusCheckPostTestCase(BatchWriteItemTestCase1Name, "/v1", "BatchWriteItem", http.StatusOK, BatchWriteItemTestCase1),
		createStatusCheckPostTestCase(BatchWriteItemTestCase2Name, "/v1", "BatchWriteItem", http.StatusOK, BatchWriteItemTestCase2),
		createStatusCheckPostTestCase(BatchWriteItemTestCase3Name, "/v1", "BatchWriteItem", http.StatusOK, BatchWriteItemTestCase3),
		createStatusCheckPostTestCase(BatchWriteItemTestCase4Name, "/v1", "BatchWriteItem", http.StatusOK, BatchWriteItemTestCase4),
		createStatusCheckPostTestCase(BatchWriteItemTestCase5Name, "/v1", "BatchWriteItem", http.StatusOK, BatchWriteItemTestCase5),
		createStatusCheckPostTestCase(BatchWriteItemTestCase6Name, "/v1", "BatchWriteItem", http.StatusOK, BatchWriteItemTestCase6),
		createStatusCheckPostTestCase(BatchWriteItemTestCase7Name, "/v1", "BatchWriteItem", http.StatusOK, BatchWriteItemTestCase7),
		createStatusCheckPostTestCase(BatchWriteItemTestCase8Name, "/v1", "BatchWriteItem", http.StatusOK, BatchWriteItemTestCase8),
		createStatusCheckPostTestCase(BatchWriteItemTestCase9Name, "/v1", "BatchWriteItem", http.StatusBadRequest, BatchWriteItemTestCase9),
		createStatusCheckPostTestCase(BatchWriteItemTestCase10Name, "/v1", "BatchWriteItem", http.StatusBadRequest, BatchWriteItemTestCase10),
	}
	apitest.RunTests(t, tests)
}

func TestApi(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration tests in short mode")
	}

	// this is done to maintain the order of the test cases
	var testNames = []string{
		"GetItemAPI",
		"GetBatchAPI",
		"QueryAPI",
		"ScanAPI",
		"UpdateItemAPI",
		"PutItemAPI",
		"DeleteItemAPI",
		"BatchWriteItemAPI",
	}

	var tests = map[string]func(t *testing.T){
		"GetItemAPI":        testGetItemAPI,
		"GetBatchAPI":       testGetBatchAPI,
		"QueryAPI":          testQueryAPI,
		"ScanAPI":           testScanAPI,
		"UpdateItemAPI":     testUpdateItemAPI,
		"PutItemAPI":        testPutItemAPI,
		"DeleteItemAPI":     testDeleteItemAPI,
		"BatchWriteItemAPI": testBatchWriteItemAPI,
	}

	// run the tests
	for _, testName := range testNames {
		t.Run(testName, tests[testName])
	}
}
