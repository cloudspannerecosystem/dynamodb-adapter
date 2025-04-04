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
	getItemTest2Output = `{"Item":{"address":{"S":"New York"},"age":{"N":"20"},"emp_id":{"N":"2"},"first_name":{"S":"Catalina"},"last_name":{"S":"Smith"},"phone_numbers":{"SS":["+1333333333"]},"profile_pics":{"BS":["U29tZUJ5dGVzRGF0YTM="]},"salaries":{"NS":["3000"]}}}`

	getItemTest3 = models.GetItemMeta{
		TableName: "employee",
		Key: map[string]*dynamodb.AttributeValue{
			"emp_id": {N: aws.String("2")},
		},
		ProjectionExpression: "emp_id, address, phone_numbers, salaries",
	}
	getItemTest3Output = `{"Item":{"address":{"S":"New York"},"emp_id":{"N":"2"},"phone_numbers":{"SS":["+1333333333"]},"salaries":{"NS":["3000"]}}}`

	getItemTest4 = models.GetItemMeta{
		TableName: "employee",
		Key: map[string]*dynamodb.AttributeValue{
			"emp_id": {N: aws.String("2")},
		},
		ProjectionExpression: "#emp, address, profile_pics",
		ExpressionAttributeNames: map[string]string{
			"#emp": "emp_id",
		},
	}
	getItemTest4Output = `{"Item":{"address":{"S":"New York"},"emp_id":{"N":"2"},"profile_pics":{"BS":["U29tZUJ5dGVzRGF0YTM="]}}}`

	getItemTest5 = models.GetItemMeta{
		TableName: "employee",
		Key: map[string]*dynamodb.AttributeValue{
			"emp_id": {N: aws.String("2")},
		},
		ProjectionExpression: "#emp, address",
	}
	getItemTest5Output = `{"Item":{"address":{"S":"New York"}}}`

	getItemTest6 = models.GetItemMeta{
		TableName: "department",
		Key: map[string]*dynamodb.AttributeValue{
			"d_id": {N: aws.String("200")}, // Assuming d_id 200 has a NULL d_name
		},
	}
	getItemTest6Output = `{"Item":{"d_id":{"N":"200"},"d_name":{"NULL":true},"d_specialization":{"S":"BA"}}}`
	getItemTestForList = models.GetItemMeta{
		TableName: "test_table",
		Key: map[string]*dynamodb.AttributeValue{
			"rank_list": {S: aws.String("rank_list")},
		},
	}
	getItemTestForListOutput = `{"Item":{"category":{"S":"category"},"id":{"S":"testing"},"list_type":{"L":[{"S":"John Doe"},{"S":"62536"},{"BOOL":true}]},"rank_list":{"S":"rank_list"},"updated_at":{"S":"2024-12-04T11:02:02Z"}}}`
	getItemTestForMap        = models.GetItemMeta{
		TableName: "mapdynamo",
		Key: map[string]*dynamodb.AttributeValue{
			"guid": {S: aws.String("123e4567-e89b-12d3-a456-value001")},
		},
	}
	getItemTestForMapOutput = `{"Item":{"address":{"M":{"active":{"BOOL":true},"additional_details":{"M":{"additional_details_2":{"M":{"landmark_field":{"S":"near water tank road"},"landmark_field_number":{"N":"1001"}}},"apartment_number":{"S":"5B"},"landmark":{"S":"Near Central Park"},"landmark notes":{"B":"YmluYXJ5X2RhdGE="}}},"mobilenumber":{"N":"9035599089"},"notes":{"B":"YmluYXJ5X2RhdGE="},"permanent_address":{"S":"789 Elm St, Springfield, SP"},"present_address":{"S":"101 Maple Ave, Metropolis, MP"}}},"contact_ranking_list":{"S":"1,2,3"},"context":{"S":"user-profile"},"guid":{"S":"123e4567-e89b-12d3-a456-value001"},"name":{"S":"Jane Smith"}}}`
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
	TestGetBatch3Output = `{"Responses":{"employee":[{"address":{"S":"Shamli"},"age":{"N":"10"},"emp_id":{"N":"1"},"first_name":{"S":"Marc"},"last_name":{"S":"Richards"},"phone_numbers":{"SS":["+1111111111","+1222222222"]},"profile_pics":{"BS":["U29tZUJ5dGVzRGF0YTE=","U29tZUJ5dGVzRGF0YTI="]},"salaries":{"NS":["1000.5","2000.75"]}},{"address":{"S":"Pune"},"age":{"N":"30"},"emp_id":{"N":"3"},"first_name":{"S":"Alice"},"last_name":{"S":"Trentor"},"phone_numbers":{"SS":["+1444444444","+1555555555"]},"profile_pics":{"BS":["U29tZUJ5dGVzRGF0YTQ=","U29tZUJ5dGVzRGF0YTU="]},"salaries":{"NS":["4000.25","5000.5","6000.75"]}},{"address":{"S":"London"},"age":{"N":"50"},"emp_id":{"N":"5"},"first_name":{"S":"David"},"last_name":{"S":"Lomond"},"phone_numbers":{"SS":["+1777777777","+1888888888","+1999999999"]},"profile_pics":{"BS":["U29tZUJ5dGVzRGF0YTc=","U29tZUJ5dGVzRGF0YTg="]},"salaries":{"NS":["9000.5"]}}]}}`

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
	TestGetBatch4Output = `{"Responses":{"department":[{"d_id":{"N":"100"},"d_name":{"S":"Engineering"},"d_specialization":{"S":"CSE, ECE, Civil"}},{"d_id":{"N":"300"},"d_name":{"S":"Culture"},"d_specialization":{"S":"History"}}],"employee":[{"address":{"S":"Shamli"},"age":{"N":"10"},"emp_id":{"N":"1"},"first_name":{"S":"Marc"},"last_name":{"S":"Richards"},"phone_numbers":{"SS":["+1111111111","+1222222222"]},"profile_pics":{"BS":["U29tZUJ5dGVzRGF0YTE=","U29tZUJ5dGVzRGF0YTI="]},"salaries":{"NS":["1000.5","2000.75"]}},{"address":{"S":"Pune"},"age":{"N":"30"},"emp_id":{"N":"3"},"first_name":{"S":"Alice"},"last_name":{"S":"Trentor"},"phone_numbers":{"SS":["+1444444444","+1555555555"]},"profile_pics":{"BS":["U29tZUJ5dGVzRGF0YTQ=","U29tZUJ5dGVzRGF0YTU="]},"salaries":{"NS":["4000.25","5000.5","6000.75"]}},{"address":{"S":"London"},"age":{"N":"50"},"emp_id":{"N":"5"},"first_name":{"S":"David"},"last_name":{"S":"Lomond"},"phone_numbers":{"SS":["+1777777777","+1888888888","+1999999999"]},"profile_pics":{"BS":["U29tZUJ5dGVzRGF0YTc=","U29tZUJ5dGVzRGF0YTg="]},"salaries":{"NS":["9000.5"]}}]}}`

	TestGetBatch5Name = "5: ProjectionExpression without ExpressionAttributeNames for 1 table"
	TestGetBatch5     = models.BatchGetMeta{
		RequestItems: map[string]models.BatchGetWithProjectionMeta{
			"employee": {
				Keys: []map[string]*dynamodb.AttributeValue{
					{"emp_id": {N: aws.String("1")}},
					{"emp_id": {N: aws.String("5")}},
					{"emp_id": {N: aws.String("3")}},
				},
				ProjectionExpression: "emp_id, address, first_name, last_name, phone_numbers, profile_pics, address",
			},
		},
	}
	TestGetBatch5Output = `{"Responses":{"employee":[{"address":{"S":"Shamli"},"emp_id":{"N":"1"},"first_name":{"S":"Marc"},"last_name":{"S":"Richards"},"phone_numbers":{"SS":["+1111111111","+1222222222"]},"profile_pics":{"BS":["U29tZUJ5dGVzRGF0YTE=","U29tZUJ5dGVzRGF0YTI="]}},{"address":{"S":"Pune"},"emp_id":{"N":"3"},"first_name":{"S":"Alice"},"last_name":{"S":"Trentor"},"phone_numbers":{"SS":["+1444444444","+1555555555"]},"profile_pics":{"BS":["U29tZUJ5dGVzRGF0YTQ=","U29tZUJ5dGVzRGF0YTU="]}},{"address":{"S":"London"},"emp_id":{"N":"5"},"first_name":{"S":"David"},"last_name":{"S":"Lomond"},"phone_numbers":{"SS":["+1777777777","+1888888888","+1999999999"]},"profile_pics":{"BS":["U29tZUJ5dGVzRGF0YTc=","U29tZUJ5dGVzRGF0YTg="]}}]}}`

	TestGetBatch6Name = "6: ProjectionExpression without ExpressionAttributeNames for 2 table"
	TestGetBatch6     = models.BatchGetMeta{
		RequestItems: map[string]models.BatchGetWithProjectionMeta{
			"employee": {
				Keys: []map[string]*dynamodb.AttributeValue{
					{"emp_id": {N: aws.String("1")}},
					{"emp_id": {N: aws.String("5")}},
					{"emp_id": {N: aws.String("3")}},
				},
				ProjectionExpression: "emp_id, address, first_name, last_name, phone_numbers, profile_pics, address",
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
	TestGetBatch6Output = `{"Responses":{"department":[{"d_id":{"N":"100"},"d_name":{"S":"Engineering"},"d_specialization":{"S":"CSE, ECE, Civil"}},{"d_id":{"N":"300"},"d_name":{"S":"Culture"},"d_specialization":{"S":"History"}}],"employee":[{"address":{"S":"Shamli"},"emp_id":{"N":"1"},"first_name":{"S":"Marc"},"last_name":{"S":"Richards"},"phone_numbers":{"SS":["+1111111111","+1222222222"]},"profile_pics":{"BS":["U29tZUJ5dGVzRGF0YTE=","U29tZUJ5dGVzRGF0YTI="]}},{"address":{"S":"Pune"},"emp_id":{"N":"3"},"first_name":{"S":"Alice"},"last_name":{"S":"Trentor"},"phone_numbers":{"SS":["+1444444444","+1555555555"]},"profile_pics":{"BS":["U29tZUJ5dGVzRGF0YTQ=","U29tZUJ5dGVzRGF0YTU="]}},{"address":{"S":"London"},"emp_id":{"N":"5"},"first_name":{"S":"David"},"last_name":{"S":"Lomond"},"phone_numbers":{"SS":["+1777777777","+1888888888","+1999999999"]},"profile_pics":{"BS":["U29tZUJ5dGVzRGF0YTc=","U29tZUJ5dGVzRGF0YTg="]}}]}}`

	TestGetBatch7Name = "7: ProjectionExpression with ExpressionAttributeNames for 1 table"
	TestGetBatch7     = models.BatchGetMeta{
		RequestItems: map[string]models.BatchGetWithProjectionMeta{
			"employee": {
				Keys: []map[string]*dynamodb.AttributeValue{
					{"emp_id": {N: aws.String("1")}},
					{"emp_id": {N: aws.String("5")}},
					{"emp_id": {N: aws.String("3")}},
				},
				ProjectionExpression: "#emp, #add, first_name, last_name, phone_numbers, profile_pics, address",
				ExpressionAttributeNames: map[string]string{
					"#emp": "emp_id",
					"#add": "address",
				},
			},
		},
	}
	TestGetBatch7Output = `{"Responses":{"employee":[{"address":{"S":"Shamli"},"emp_id":{"N":"1"},"first_name":{"S":"Marc"},"last_name":{"S":"Richards"},"phone_numbers":{"SS":["+1111111111","+1222222222"]},"profile_pics":{"BS":["U29tZUJ5dGVzRGF0YTE=","U29tZUJ5dGVzRGF0YTI="]}},{"address":{"S":"Pune"},"emp_id":{"N":"3"},"first_name":{"S":"Alice"},"last_name":{"S":"Trentor"},"phone_numbers":{"SS":["+1444444444","+1555555555"]},"profile_pics":{"BS":["U29tZUJ5dGVzRGF0YTQ=","U29tZUJ5dGVzRGF0YTU="]}},{"address":{"S":"London"},"emp_id":{"N":"5"},"first_name":{"S":"David"},"last_name":{"S":"Lomond"},"phone_numbers":{"SS":["+1777777777","+1888888888","+1999999999"]},"profile_pics":{"BS":["U29tZUJ5dGVzRGF0YTc=","U29tZUJ5dGVzRGF0YTg="]}}]}}`

	TestGetBatch8Name = "8: ProjectionExpression with ExpressionAttributeNames for 2 table"
	TestGetBatch8     = models.BatchGetMeta{
		RequestItems: map[string]models.BatchGetWithProjectionMeta{
			"employee": {
				Keys: []map[string]*dynamodb.AttributeValue{
					{"emp_id": {N: aws.String("1")}},
					{"emp_id": {N: aws.String("5")}},
					{"emp_id": {N: aws.String("3")}},
				},
				ProjectionExpression: "#emp, #add, first_name, last_name, phone_numbers, profile_pics, address",
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
	TestGetBatch8Output = `{"Responses":{"department":[{"d_id":{"N":"100"},"d_name":{"S":"Engineering"},"d_specialization":{"S":"CSE, ECE, Civil"}},{"d_id":{"N":"300"},"d_name":{"S":"Culture"},"d_specialization":{"S":"History"}}],"employee":[{"address":{"S":"Shamli"},"emp_id":{"N":"1"},"first_name":{"S":"Marc"},"last_name":{"S":"Richards"},"phone_numbers":{"SS":["+1111111111","+1222222222"]},"profile_pics":{"BS":["U29tZUJ5dGVzRGF0YTE=","U29tZUJ5dGVzRGF0YTI="]}},{"address":{"S":"Pune"},"emp_id":{"N":"3"},"first_name":{"S":"Alice"},"last_name":{"S":"Trentor"},"phone_numbers":{"SS":["+1444444444","+1555555555"]},"profile_pics":{"BS":["U29tZUJ5dGVzRGF0YTQ=","U29tZUJ5dGVzRGF0YTU="]}},{"address":{"S":"London"},"emp_id":{"N":"5"},"first_name":{"S":"David"},"last_name":{"S":"Lomond"},"phone_numbers":{"SS":["+1777777777","+1888888888","+1999999999"]},"profile_pics":{"BS":["U29tZUJ5dGVzRGF0YTc=","U29tZUJ5dGVzRGF0YTg="]}}]}}`

	TestGetBatch9Name = "9: ProjectionExpression but ExpressionAttributeNames not present"
	TestGetBatch9     = models.BatchGetMeta{
		RequestItems: map[string]models.BatchGetWithProjectionMeta{
			"employee": {
				Keys: []map[string]*dynamodb.AttributeValue{
					{"emp_id": {N: aws.String("1")}},
					{"emp_id": {N: aws.String("5")}},
					{"emp_id": {N: aws.String("3")}},
				},
				ProjectionExpression: "#emp, #add, first_name, last_name phone_numbers, profile_pics, address",
			},
		},
	}
	TestGetBatch9Output = `{"Responses":{"employee":[{"address":{"S":"Shamli"},"first_name":{"S":"Marc"},"profile_pics":{"BS":["U29tZUJ5dGVzRGF0YTE=","U29tZUJ5dGVzRGF0YTI="]}},{"address":{"S":"Pune"},"first_name":{"S":"Alice"},"profile_pics":{"BS":["U29tZUJ5dGVzRGF0YTQ=","U29tZUJ5dGVzRGF0YTU="]}},{"address":{"S":"London"},"first_name":{"S":"David"},"profile_pics":{"BS":["U29tZUJ5dGVzRGF0YTc=","U29tZUJ5dGVzRGF0YTg="]}}]}}`

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
	TestGetBatchForListName = "Test: BatchGet for Multiple Keys in List Dyanmo table"
	TestGetBatchForList     = models.BatchGetMeta{
		RequestItems: map[string]models.BatchGetWithProjectionMeta{
			"test_table": {
				Keys: []map[string]*dynamodb.AttributeValue{
					{"rank_list": {S: aws.String("rank_list")}},
					{"rank_list": {S: aws.String("rank_list1")}},
					{"rank_list": {S: aws.String("rank_list2")}},
				},
			},
		},
	}
	TestGetBatch11Name = "11: Keys present for 1 table for Map Data"
	TestGetBatch11     = models.BatchGetMeta{
		RequestItems: map[string]models.BatchGetWithProjectionMeta{
			"mapdynamo": {
				Keys: []map[string]*dynamodb.AttributeValue{
					{"guid": {S: aws.String("123e4567-e89b-12d3-a456-value001")}, "context": {S: aws.String("user-profile")}},
				},
			},
		},
	}
	TestGetBatchForListOutput = `{"Responses":{"test_table":[{"category":{"S":"category"},"id":{"S":"testing"},"list_type":{"L":[{"S":"John Doe"},{"S":"62536"},{"BOOL":true}]},"rank_list":{"S":"rank_list"},"updated_at":{"S":"2024-12-04T11:02:02Z"}},{"category":{"S":"category1"},"id":{"S":"id"},"list_type":{"L":[{"S":"string_value"},{"S":"12345"},{"BOOL":true},{"L":[{"N":"1"},{"N":"2"},{"N":"3"}]},{"S":"testing"}]},"rank_list":{"S":"rank_list1"},"updated_at":{"S":"2024-12-04T11:02:02Z"}},{"category":{"S":"category2"},"id":{"S":"id2"},"list_type":{"L":[{"S":"test"},{"S":"dummy_value"},{"S":"62536"}]},"rank_list":{"S":"rank_list2"},"updated_at":{"S":"2024-12-04T11:02:02Z"}}]}}`
	TestGetBatch11Output      = `{"Responses":{"mapdynamo":[{"address":{"M":{"active":{"BOOL":true},"additional_details":{"M":{"additional_details_2":{"M":{"landmark_field":{"S":"near water tank road"},"landmark_field_number":{"N":"1001"}}},"apartment_number":{"S":"5B"},"landmark":{"S":"Near Central Park"},"landmark notes":{"B":"YmluYXJ5X2RhdGE="}}},"mobilenumber":{"N":"9035599089"},"notes":{"B":"YmluYXJ5X2RhdGE="},"permanent_address":{"S":"789 Elm St, Springfield, SP"},"present_address":{"S":"101 Maple Ave, Metropolis, MP"}}},"contact_ranking_list":{"S":"1,2,3"},"context":{"S":"user-profile"},"guid":{"S":"123e4567-e89b-12d3-a456-value001"},"name":{"S":"Jane Smith"}}]}}`
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

	queryTestCaseOutput1 = `{"Count":5,"Items":[{"address":{"S":"Shamli"},"age":{"N":"10"},"emp_id":{"N":"1"},"first_name":{"S":"Marc"},"last_name":{"S":"Richards"},"phone_numbers":{"SS":["+1111111111","+1222222222"]},"profile_pics":{"BS":["U29tZUJ5dGVzRGF0YTE=","U29tZUJ5dGVzRGF0YTI="]},"salaries":{"NS":["1000.5","2000.75"]}},{"address":{"S":"New York"},"age":{"N":"20"},"emp_id":{"N":"2"},"first_name":{"S":"Catalina"},"last_name":{"S":"Smith"},"phone_numbers":{"SS":["+1333333333"]},"profile_pics":{"BS":["U29tZUJ5dGVzRGF0YTM="]},"salaries":{"NS":["3000"]}},{"address":{"S":"Pune"},"age":{"N":"30"},"emp_id":{"N":"3"},"first_name":{"S":"Alice"},"last_name":{"S":"Trentor"},"phone_numbers":{"SS":["+1444444444","+1555555555"]},"profile_pics":{"BS":["U29tZUJ5dGVzRGF0YTQ=","U29tZUJ5dGVzRGF0YTU="]},"salaries":{"NS":["4000.25","5000.5","6000.75"]}},{"address":{"S":"Silicon Valley"},"age":{"N":"40"},"emp_id":{"N":"4"},"first_name":{"S":"Lea"},"last_name":{"S":"Martin"},"phone_numbers":{"SS":["+1666666666"]},"profile_pics":{"BS":["U29tZUJ5dGVzRGF0YTY="]},"salaries":{"NS":["7000","8000.25"]}},{"address":{"S":"London"},"age":{"N":"50"},"emp_id":{"N":"5"},"first_name":{"S":"David"},"last_name":{"S":"Lomond"},"phone_numbers":{"SS":["+1777777777","+1888888888","+1999999999"]},"profile_pics":{"BS":["U29tZUJ5dGVzRGF0YTc=","U29tZUJ5dGVzRGF0YTg="]},"salaries":{"NS":["9000.5"]}}]}`

	queryTestCaseOutput2 = `{"Count":5,"Items":[{"emp_id":{"N":"1"},"first_name":{"S":"Marc"}},{"emp_id":{"N":"2"},"first_name":{"S":"Catalina"}},{"emp_id":{"N":"3"},"first_name":{"S":"Alice"}},{"emp_id":{"N":"4"},"first_name":{"S":"Lea"}},{"emp_id":{"N":"5"},"first_name":{"S":"David"}}]}`

	queryTestCaseOutput3 = `{"Count":5,"Items":[{"emp_id":{"N":"1"},"first_name":{"S":"Marc"},"last_name":{"S":"Richards"}},{"emp_id":{"N":"2"},"first_name":{"S":"Catalina"},"last_name":{"S":"Smith"}},{"emp_id":{"N":"3"},"first_name":{"S":"Alice"},"last_name":{"S":"Trentor"}},{"emp_id":{"N":"4"},"first_name":{"S":"Lea"},"last_name":{"S":"Martin"}},{"emp_id":{"N":"5"},"first_name":{"S":"David"},"last_name":{"S":"Lomond"}}]}`

	queryTestCaseOutput4 = `{"Count":1,"Items":[{"emp_id":{"N":"2"},"first_name":{"S":"Catalina"},"last_name":{"S":"Smith"}}]}`

	queryTestCaseOutput6 = `{"Count":1,"Items":[{"emp_id":{"N":"3"},"first_name":{"S":"Alice"},"last_name":{"S":"Trentor"}}]}`

	queryTestCaseOutput8 = `{"Count":1,"Items":[{"emp_id":{"N":"3"},"first_name":{"S":"Alice"},"last_name":{"S":"Trentor"}}]}`

	queryTestCaseOutput9 = `{"Count":1,"Items":[{"emp_id":{"N":"3"},"first_name":{"S":"Alice"},"last_name":{"S":"Trentor"}}]}`

	queryTestCaseOutput10 = `{"Count":5,"Items":[{"address":{"S":"Shamli"},"age":{"N":"10"},"emp_id":{"N":"1"},"first_name":{"S":"Marc"},"last_name":{"S":"Richards"},"phone_numbers":{"SS":["+1111111111","+1222222222"]},"profile_pics":{"BS":["U29tZUJ5dGVzRGF0YTE=","U29tZUJ5dGVzRGF0YTI="]},"salaries":{"NS":["1000.5","2000.75"]}},{"address":{"S":"New York"},"age":{"N":"20"},"emp_id":{"N":"2"},"first_name":{"S":"Catalina"},"last_name":{"S":"Smith"},"phone_numbers":{"SS":["+1333333333"]},"profile_pics":{"BS":["U29tZUJ5dGVzRGF0YTM="]},"salaries":{"NS":["3000"]}},{"address":{"S":"Pune"},"age":{"N":"30"},"emp_id":{"N":"3"},"first_name":{"S":"Alice"},"last_name":{"S":"Trentor"},"phone_numbers":{"SS":["+1444444444","+1555555555"]},"profile_pics":{"BS":["U29tZUJ5dGVzRGF0YTQ=","U29tZUJ5dGVzRGF0YTU="]},"salaries":{"NS":["4000.25","5000.5","6000.75"]}},{"address":{"S":"Silicon Valley"},"age":{"N":"40"},"emp_id":{"N":"4"},"first_name":{"S":"Lea"},"last_name":{"S":"Martin"},"phone_numbers":{"SS":["+1666666666"]},"profile_pics":{"BS":["U29tZUJ5dGVzRGF0YTY="]},"salaries":{"NS":["7000","8000.25"]}},{"address":{"S":"London"},"age":{"N":"50"},"emp_id":{"N":"5"},"first_name":{"S":"David"},"last_name":{"S":"Lomond"},"phone_numbers":{"SS":["+1777777777","+1888888888","+1999999999"]},"profile_pics":{"BS":["U29tZUJ5dGVzRGF0YTc=","U29tZUJ5dGVzRGF0YTg="]},"salaries":{"NS":["9000.5"]}}]}`

	queryTestCaseOutput11 = `{"Count":4,"Items":[{"address":{"S":"Shamli"},"age":{"N":"10"},"emp_id":{"N":"1"},"first_name":{"S":"Marc"},"last_name":{"S":"Richards"},"phone_numbers":{"SS":["+1111111111","+1222222222"]},"profile_pics":{"BS":["U29tZUJ5dGVzRGF0YTE=","U29tZUJ5dGVzRGF0YTI="]},"salaries":{"NS":["1000.5","2000.75"]}},{"address":{"S":"New York"},"age":{"N":"20"},"emp_id":{"N":"2"},"first_name":{"S":"Catalina"},"last_name":{"S":"Smith"},"phone_numbers":{"SS":["+1333333333"]},"profile_pics":{"BS":["U29tZUJ5dGVzRGF0YTM="]},"salaries":{"NS":["3000"]}},{"address":{"S":"Pune"},"age":{"N":"30"},"emp_id":{"N":"3"},"first_name":{"S":"Alice"},"last_name":{"S":"Trentor"},"phone_numbers":{"SS":["+1444444444","+1555555555"]},"profile_pics":{"BS":["U29tZUJ5dGVzRGF0YTQ=","U29tZUJ5dGVzRGF0YTU="]},"salaries":{"NS":["4000.25","5000.5","6000.75"]}},{"address":{"S":"Silicon Valley"},"age":{"N":"40"},"emp_id":{"N":"4"},"first_name":{"S":"Lea"},"last_name":{"S":"Martin"},"phone_numbers":{"SS":["+1666666666"]},"profile_pics":{"BS":["U29tZUJ5dGVzRGF0YTY="]},"salaries":{"NS":["7000","8000.25"]}}],"LastEvaluatedKey":{"emp_id":{"N":"4"},"offset":{"N":"4"}}}`

	queryTestCaseOutput12 = `{"Count":4,"Items":[{"address":{"S":"Shamli"},"age":{"N":"10"},"emp_id":{"N":"1"},"first_name":{"S":"Marc"},"last_name":{"S":"Richards"},"phone_numbers":{"SS":["+1111111111","+1222222222"]},"profile_pics":{"BS":["U29tZUJ5dGVzRGF0YTE=","U29tZUJ5dGVzRGF0YTI="]},"salaries":{"NS":["1000.5","2000.75"]}},{"address":{"S":"New York"},"age":{"N":"20"},"emp_id":{"N":"2"},"first_name":{"S":"Catalina"},"last_name":{"S":"Smith"},"phone_numbers":{"SS":["+1333333333"]},"profile_pics":{"BS":["U29tZUJ5dGVzRGF0YTM="]},"salaries":{"NS":["3000"]}},{"address":{"S":"Pune"},"age":{"N":"30"},"emp_id":{"N":"3"},"first_name":{"S":"Alice"},"last_name":{"S":"Trentor"},"phone_numbers":{"SS":["+1444444444","+1555555555"]},"profile_pics":{"BS":["U29tZUJ5dGVzRGF0YTQ=","U29tZUJ5dGVzRGF0YTU="]},"salaries":{"NS":["4000.25","5000.5","6000.75"]}},{"address":{"S":"Silicon Valley"},"age":{"N":"40"},"emp_id":{"N":"4"},"first_name":{"S":"Lea"},"last_name":{"S":"Martin"},"phone_numbers":{"SS":["+1666666666"]},"profile_pics":{"BS":["U29tZUJ5dGVzRGF0YTY="]},"salaries":{"NS":["7000","8000.25"]}}],"LastEvaluatedKey":{"emp_id":{"N":"4"},"offset":{"N":"4"}}}`

	queryTestCaseOutput13 = `{"Count":5,"Items":[]}`

	queryTestCaseOutput14 = `{"Count":1,"Items":[]}`

	queryTestCaseOutput15 = `{"Count":1,"Items":[{"emp_id":{"N":"3"},"first_name":{"S":"Alice"},"last_name":{"S":"Trentor"}}]}`

	queryTestCaseOutput16 = `{"Count":1,"Items":[]}`

	queryTestCase17 = models.Query{
		TableName: "department",
		RangeExp:  "d_id =:val1",
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":val1": {N: aws.String("200")}, // d_id 200 has NULL d_name
		},
	}
	queryTestCaseOutput17 = `{"Count":1,"Items":[{"d_id":{"N":"200"},"d_name":{"NULL":true},"d_specialization":{"S":"BA"}}]}`
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
	ScanTestCase2Output = `{"Count":5,"Items":[{"address":{"S":"Shamli"},"age":{"N":"10"},"emp_id":{"N":"1"},"first_name":{"S":"Marc"},"last_name":{"S":"Richards"},"phone_numbers":{"SS":["+1111111111","+1222222222"]},"profile_pics":{"BS":["U29tZUJ5dGVzRGF0YTE=","U29tZUJ5dGVzRGF0YTI="]},"salaries":{"NS":["1000.5","2000.75"]}},{"address":{"S":"New York"},"age":{"N":"20"},"emp_id":{"N":"2"},"first_name":{"S":"Catalina"},"last_name":{"S":"Smith"},"phone_numbers":{"SS":["+1333333333"]},"profile_pics":{"BS":["U29tZUJ5dGVzRGF0YTM="]},"salaries":{"NS":["3000"]}},{"address":{"S":"Pune"},"age":{"N":"30"},"emp_id":{"N":"3"},"first_name":{"S":"Alice"},"last_name":{"S":"Trentor"},"phone_numbers":{"SS":["+1444444444","+1555555555"]},"profile_pics":{"BS":["U29tZUJ5dGVzRGF0YTQ=","U29tZUJ5dGVzRGF0YTU="]},"salaries":{"NS":["4000.25","5000.5","6000.75"]}},{"address":{"S":"Silicon Valley"},"age":{"N":"40"},"emp_id":{"N":"4"},"first_name":{"S":"Lea"},"last_name":{"S":"Martin"},"phone_numbers":{"SS":["+1666666666"]},"profile_pics":{"BS":["U29tZUJ5dGVzRGF0YTY="]},"salaries":{"NS":["7000","8000.25"]}},{"address":{"S":"London"},"age":{"N":"50"},"emp_id":{"N":"5"},"first_name":{"S":"David"},"last_name":{"S":"Lomond"},"phone_numbers":{"SS":["+1777777777","+1888888888","+1999999999"]},"profile_pics":{"BS":["U29tZUJ5dGVzRGF0YTc=","U29tZUJ5dGVzRGF0YTg="]},"salaries":{"NS":["9000.5"]}}]}`

	ScanTestCase3Name = "3: With Limit Attribute"
	ScanTestCase3     = models.ScanMeta{
		TableName: "employee",
		Limit:     3,
	}
	ScanTestCase3Output = `{"Count":3,"Items":[{"address":{"S":"Shamli"},"age":{"N":"10"},"emp_id":{"N":"1"},"first_name":{"S":"Marc"},"last_name":{"S":"Richards"},"phone_numbers":{"SS":["+1111111111","+1222222222"]},"profile_pics":{"BS":["U29tZUJ5dGVzRGF0YTE=","U29tZUJ5dGVzRGF0YTI="]},"salaries":{"NS":["1000.5","2000.75"]}},{"address":{"S":"New York"},"age":{"N":"20"},"emp_id":{"N":"2"},"first_name":{"S":"Catalina"},"last_name":{"S":"Smith"},"phone_numbers":{"SS":["+1333333333"]},"profile_pics":{"BS":["U29tZUJ5dGVzRGF0YTM="]},"salaries":{"NS":["3000"]}},{"address":{"S":"Pune"},"age":{"N":"30"},"emp_id":{"N":"3"},"first_name":{"S":"Alice"},"last_name":{"S":"Trentor"},"phone_numbers":{"SS":["+1444444444","+1555555555"]},"profile_pics":{"BS":["U29tZUJ5dGVzRGF0YTQ=","U29tZUJ5dGVzRGF0YTU="]},"salaries":{"NS":["4000.25","5000.5","6000.75"]}}],"LastEvaluatedKey":{"emp_id":{"N":"3"},"offset":{"N":"3"}}}`

	ScanTestCase4Name = "4: With Projection Expression"
	ScanTestCase4     = models.ScanMeta{
		TableName:            "employee",
		ProjectionExpression: "address, emp_id, first_name",
	}
	ScanTestCase4Output = `{"Count":5,"Items":[{"address":{"S":"Shamli"},"emp_id":{"N":"1"},"first_name":{"S":"Marc"}},{"address":{"S":"New York"},"emp_id":{"N":"2"},"first_name":{"S":"Catalina"}},{"address":{"S":"Pune"},"emp_id":{"N":"3"},"first_name":{"S":"Alice"}},{"address":{"S":"Silicon Valley"},"emp_id":{"N":"4"},"first_name":{"S":"Lea"}},{"address":{"S":"London"},"emp_id":{"N":"5"},"first_name":{"S":"David"}}]}`

	ScanTestCase5Name = "5: With Projection Expression & limit"
	ScanTestCase5     = models.ScanMeta{
		TableName:            "employee",
		Limit:                3,
		ProjectionExpression: "address, emp_id, first_name",
	}
	ScanTestCase5Output = `{"Count":3,"Items":[{"address":{"S":"Shamli"},"emp_id":{"N":"1"},"first_name":{"S":"Marc"}},{"address":{"S":"New York"},"emp_id":{"N":"2"},"first_name":{"S":"Catalina"}},{"address":{"S":"Pune"},"emp_id":{"N":"3"},"first_name":{"S":"Alice"}}],"LastEvaluatedKey":{"emp_id":{"N":"3"},"offset":{"N":"3"}}}`

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
	ScanTestCase7Output = `{"Count":3,"Items":[{"address":{"S":"Shamli"},"age":{"N":"10"},"emp_id":{"N":"1"},"first_name":{"S":"Marc"},"last_name":{"S":"Richards"}},{"address":{"S":"New York"},"age":{"N":"20"},"emp_id":{"N":"2"},"first_name":{"S":"Catalina"},"last_name":{"S":"Smith"}},{"address":{"S":"Pune"},"age":{"N":"30"},"emp_id":{"N":"3"},"first_name":{"S":"Alice"},"last_name":{"S":"Trentor"}}],"LastEvaluatedKey":{"emp_id":{"N":"3"},"offset":{"N":"3"}}}`

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
	ScanTestCase9Output = `{"Count":4,"Items":[{"address":{"S":"New York"},"age":{"N":"20"},"emp_id":{"N":"2"},"first_name":{"S":"Catalina"},"last_name":{"S":"Smith"},"phone_numbers":{"SS":["+1333333333"]},"profile_pics":{"BS":["U29tZUJ5dGVzRGF0YTM="]},"salaries":{"NS":["3000"]}},{"address":{"S":"Pune"},"age":{"N":"30"},"emp_id":{"N":"3"},"first_name":{"S":"Alice"},"last_name":{"S":"Trentor"},"phone_numbers":{"SS":["+1444444444","+1555555555"]},"profile_pics":{"BS":["U29tZUJ5dGVzRGF0YTQ=","U29tZUJ5dGVzRGF0YTU="]},"salaries":{"NS":["4000.25","5000.5","6000.75"]}},{"address":{"S":"Silicon Valley"},"age":{"N":"40"},"emp_id":{"N":"4"},"first_name":{"S":"Lea"},"last_name":{"S":"Martin"},"phone_numbers":{"SS":["+1666666666"]},"profile_pics":{"BS":["U29tZUJ5dGVzRGF0YTY="]},"salaries":{"NS":["7000","8000.25"]}},{"address":{"S":"London"},"age":{"N":"50"},"emp_id":{"N":"5"},"first_name":{"S":"David"},"last_name":{"S":"Lomond"},"phone_numbers":{"SS":["+1777777777","+1888888888","+1999999999"]},"profile_pics":{"BS":["U29tZUJ5dGVzRGF0YTc=","U29tZUJ5dGVzRGF0YTg="]},"salaries":{"NS":["9000.5"]}}]}`

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
	ScanTestCase11Output = `{"Count":4,"Items":[{"address":{"S":"New York"},"age":{"N":"20"},"emp_id":{"N":"2"},"first_name":{"S":"Catalina"},"last_name":{"S":"Smith"},"phone_numbers":{"SS":["+1333333333"]},"profile_pics":{"BS":["U29tZUJ5dGVzRGF0YTM="]},"salaries":{"NS":["3000"]}},{"address":{"S":"Pune"},"age":{"N":"30"},"emp_id":{"N":"3"},"first_name":{"S":"Alice"},"last_name":{"S":"Trentor"},"phone_numbers":{"SS":["+1444444444","+1555555555"]},"profile_pics":{"BS":["U29tZUJ5dGVzRGF0YTQ=","U29tZUJ5dGVzRGF0YTU="]},"salaries":{"NS":["4000.25","5000.5","6000.75"]}},{"address":{"S":"Silicon Valley"},"age":{"N":"40"},"emp_id":{"N":"4"},"first_name":{"S":"Lea"},"last_name":{"S":"Martin"},"phone_numbers":{"SS":["+1666666666"]},"profile_pics":{"BS":["U29tZUJ5dGVzRGF0YTY="]},"salaries":{"NS":["7000","8000.25"]}},{"address":{"S":"London"},"age":{"N":"50"},"emp_id":{"N":"5"},"first_name":{"S":"David"},"last_name":{"S":"Lomond"},"phone_numbers":{"SS":["+1777777777","+1888888888","+1999999999"]},"profile_pics":{"BS":["U29tZUJ5dGVzRGF0YTc=","U29tZUJ5dGVzRGF0YTg="]},"salaries":{"NS":["9000.5"]}}]}`

	ScanTestCase12Name = "12: With ExclusiveStartKey"
	ScanTestCase12     = models.ScanMeta{
		TableName: "employee",
		ExclusiveStartKey: map[string]*dynamodb.AttributeValue{
			"emp_id": {N: aws.String("4")},
			"offset": {N: aws.String("3")},
		},
		Limit: 3,
	}
	ScanTestCase12Output = `{"Count":2,"Items":[{"address":{"S":"Silicon Valley"},"age":{"N":"40"},"emp_id":{"N":"4"},"first_name":{"S":"Lea"},"last_name":{"S":"Martin"},"phone_numbers":{"SS":["+1666666666"]},"profile_pics":{"BS":["U29tZUJ5dGVzRGF0YTY="]},"salaries":{"NS":["7000","8000.25"]}},{"address":{"S":"London"},"age":{"N":"50"},"emp_id":{"N":"5"},"first_name":{"S":"David"},"last_name":{"S":"Lomond"},"phone_numbers":{"SS":["+1777777777","+1888888888","+1999999999"]},"profile_pics":{"BS":["U29tZUJ5dGVzRGF0YTc=","U29tZUJ5dGVzRGF0YTg="]},"salaries":{"NS":["9000.5"]}}]}`

	ScanTestCase13Name = "13: With Count"
	ScanTestCase13     = models.ScanMeta{
		TableName: "employee",
		Limit:     3,
		Select:    "COUNT",
	}
	ScanTestCase13Output = `{"Count":5,"Items":[]}`

	ScanTestCase14Name = "14: NULL Value"
	ScanTestCase14     = models.ScanMeta{
		TableName:        "department",
		FilterExpression: "d_id = :val1",
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":val1": {N: aws.String("200")}, // Filter for NULL d_name
		},
	}
	ScanTestCase14Output = `{"Count":1,"Items":[{"d_id":{"N":"200"},"d_name":{"NULL":true},"d_specialization":{"S":"BA"}}],"LastEvaluatedKey":null}`
	ScanTestCaseListName = "15: List Type"
	ScanTestCaseList     = models.ScanMeta{
		TableName: "test_table",
		Limit:     2,
		Select:    "COUNT",
	}
	ScanTestCaseListOutput = `{"Count":3,"Items":[]}`
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
		UpdateExpression: "SET age = :age, phone_numbers = :phone_numbers, salaries = :salaries, profile_pics = :profile_pics",
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":age": {N: aws.String("10")},
			":phone_numbers": {SS: aws.StringSlice([]string{
				"+1111111111", "+1222222222", "+1111111111", "+1222222222",
			})},
			":salaries": {NS: aws.StringSlice([]string{
				"1000.5", "2000.75", "1000.5", "2000.75",
			})},
			"profile_pics": {BS: [][]byte{[]byte("SomeBytesData1"), []byte("SomeBytesData2"), []byte("SomeBytesData1"), []byte("SomeBytesData2")}},
		},
		ReturnValues: "ALL_NEW",
	}

	UpdateItemTestCase2Output = `{"Attributes":{"address":{"S":"Shamli"},"age":{"N":"10"},"emp_id":{"N":"1"},"first_name":{"S":"Marc"},"last_name":{"S":"Richards"},"phone_numbers":{"SS":["+1111111111","+1222222222"]},"profile_pics":{"BS":["U29tZUJ5dGVzRGF0YTE=","U29tZUJ5dGVzRGF0YTI="]},"salaries":{"NS":["1000.5","2000.75"]}}}`

	UpdateItemTestCase3Name = "3: UpdateExpression, ExpressionAttributeValues with ExpressionAttributeNames"
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
	UpdateItemTestCase3Output = `{"Attributes":{"address":{"S":"Shamli"},"age":{"N":"10"},"emp_id":{"N":"1"},"first_name":{"S":"Marc"},"last_name":{"S":"Richards"},"phone_numbers":{"SS":["+1111111111","+1222222222"]},"profile_pics":{"BS":["U29tZUJ5dGVzRGF0YTE=","U29tZUJ5dGVzRGF0YTI="]},"salaries":{"NS":["1000.5","2000.75"]}}}`

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
	UpdateItemTestCase7Output = `{"Attributes":{"address":{"S":"Shamli"},"age":{"N":"10"},"emp_id":{"N":"1"},"first_name":{"S":"Marc"},"last_name":{"S":"Richards"},"phone_numbers":{"SS":["+1111111111","+1222222222"]},"profile_pics":{"BS":["U29tZUJ5dGVzRGF0YTE=","U29tZUJ5dGVzRGF0YTI="]},"salaries":{"NS":["1000.5","2000.75"]}}}`

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

	UpdateItemTestForListName = "Test: UpdateItem for List Data"
	UpdateItemTestForList     = models.UpdateAttr{
		TableName: "test_table",
		Key: map[string]*dynamodb.AttributeValue{
			"rank_list": {S: aws.String("rank_list2")},
		},
		UpdateExpression: "SET #lt[1] = :newValue",
		ExpressionAttributeNames: map[string]string{
			"#lt": "list_type",
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":newValue": {S: aws.String("updated_value")},
		},
		ReturnValues: "UPDATED_ALL",
	}
	UpdateItemTestForListOutput = `{"Attributes":{"category":{"S":"category2"},"id":{"S":"id2"},"list_type":{"S":"[\"test\",\"updated_value\",\"62536\"]"},"rank_list":{"S":"rank_list2"},"updated_at":{"S":"2024-12-04T11:02:02Z"}}}`
	UpdateItemTestCase11Name    = "6: UpdateItem for Map w"
	UpdateItemTestCase11        = models.UpdateAttr{
		TableName: "mapdynamo",
		Key: map[string]*dynamodb.AttributeValue{
			"guid":    {S: aws.String("123e4567-e89b-12d3-a456-value011")},
			"context": {S: aws.String("user-profile")},
		},
		UpdateExpression: "SET #a.#b.#c.#d = :newValue",
		ExpressionAttributeNames: map[string]string{
			"#a": "address",
			"#b": "additional_details",
			"#c": "additional_details_2",
			"#d": "landmark_field",
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":newValue": {S: aws.String("near water tank road")},
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
			"emp_id":        {N: aws.String("1")},
			"age":           {N: aws.String("11")},
			"phone_numbers": {SS: aws.StringSlice([]string{"+1111111111", "+1222222222", "+1111111111"})},
			"profile_pics":  {BS: [][]byte{[]byte("SomeBytesData1"), []byte("SomeBytesData2"), []byte("SomeBytesData1")}},
			"salaries":      {NS: aws.StringSlice([]string{"1000.5", "2000.75", "1000.5"})},
		},
	}

	PutItemTestCase2Output = `{"Attributes":{"address":{"S":"Shamli"},"age":{"N":"10"},"emp_id":{"N":"1"},"first_name":{"S":"Marc"},"last_name":{"S":"Richards"},"phone_numbers":{"SS":["+1111111111","+1222222222"]},"profile_pics":{"BS":["U29tZUJ5dGVzRGF0YTE=","U29tZUJ5dGVzRGF0YTI="]},"salaries":{"NS":["1000.5","2000.75"]}}}`

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
	PutItemTestCase3Output = `{"Attributes":{"address":{"S":"Shamli"},"age":{"N":"11"},"emp_id":{"N":"1"},"first_name":{"S":"Marc"},"last_name":{"S":"Richards"},"phone_numbers":{"SS":["+1222222222",+1111111111""]},"profile_pics":{"BS":["U29tZUJ5dGVzRGF0YTE=","U29tZUJ5dGVzRGF0YTI="]},"salaries":{"NS":["1000.5","2000.75"]}}}`

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
	PutItemTestCase4Output = `{"Attributes":{"address":{"S":"Shamli"},"age":{"N":"10"},"emp_id":{"N":"1"},"first_name":{"S":"Marc"},"last_name":{"S":"Richards"},"phone_numbers":{"SS":["+1222222222",+1111111111""]},"profile_pics":{"BS":["U29tZUJ5dGVzRGF0YTE=","U29tZUJ5dGVzRGF0YTI="]},"salaries":{"NS":["1000.5","2000.75"]}}}`

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

	PutItemTestForListName = "Test: PutItem with List Data"
	PutItemTestForList     = models.Meta{
		TableName: "test_table",
		Item: map[string]*dynamodb.AttributeValue{
			"updated_at": {
				S: aws.String("2025-01-21T10:00:00Z"),
			},
			"rank_list": {
				S: aws.String("new_rank_list"),
			},
			"category": {
				S: aws.String("new_category"),
			},
			"id": {
				S: aws.String("new_id"),
			},
			"list_type": {
				L: []*dynamodb.AttributeValue{
					{S: aws.String("list_value1")},
					{N: aws.String("100")},
					{BOOL: aws.Bool(false)},
					{M: map[string]*dynamodb.AttributeValue{
						"key1": {S: aws.String("value1")},
					}},
				},
			},
		},
	}
	PutItemTestCase10Name = "10: Item to be inserted"
	PutItemTestCase10     = models.Meta{
		TableName: "mapdynamo",
		Item: map[string]*dynamodb.AttributeValue{
			"guid": {
				S: aws.String("123e4567-e89b-12d3-a456-value001"),
			},
			"context": {
				S: aws.String("user-profile"),
			},
			"contact_ranking_list": {
				S: aws.String("1,2,3"),
			},
			"name": {
				S: aws.String("Jane Smith"),
			},
			"address": {
				M: map[string]*dynamodb.AttributeValue{
					"permanent_address": {
						S: aws.String("789 Elm St, Springfield, SP"),
					},
					"present_address": {
						S: aws.String("101 Maple Ave, Metropolis, MP"),
					},
					"mobilenumber": {
						N: aws.String("9035599089"),
					},
					"active": {
						BOOL: aws.Bool(true),
					},
					"notes": {
						B: []byte("YmluYXJ5X2RhdGE="),
					},
					"additional_details": {
						M: map[string]*dynamodb.AttributeValue{
							"landmark": {
								S: aws.String("Near Central Park"),
							},
							"apartment_number": {
								S: aws.String("5B"),
							},
							"landmark notes": {
								B: []byte("YmluYXJ5X2RhdGE="),
							},
							"additional_details_2": {
								M: map[string]*dynamodb.AttributeValue{
									"landmark_field": {
										S: aws.String("near water tank road"),
									},
									"landmark_field_number": {
										N: aws.String("1001"),
									},
								},
							},
						},
					},
				},
			},
		},
	}
	PutItemTestForListOutput = `{"Attributes":{}}`
	PutItemTestCase10Output  = `{"Attributes":{"address":{"M":{"active":{"BOOL":true},"additional_details":{"M":{"additional_details_2":{"M":{"landmark_field":{"S":"near water tank road"},"landmark_field_number":{"N":"1001"}}},"apartment_number":{"S":"5B"},"landmark":{"S":"Near Central Park"},"landmark notes":{"B":"YmluYXJ5X2RhdGE="}}},"mobilenumber":{"N":"9035599089"},"notes":{"B":"YmluYXJ5X2RhdGE="},"permanent_address":{"S":"789 Elm St, Springfield, SP"},"present_address":{"S":"101 Maple Ave, Metropolis, MP"}}},"contact_ranking_list":{"S":"1,2,3"},"context":{"S":"user-profile"},"guid":{"S":"123e4567-e89b-12d3-a456-value001"},"name":{"S":"Jane Smith"}}}`
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
	DeleteItemTestCase2Output = `{"Attributes":{"address":{"S":"New York"},"age":{"N":"20"},"emp_id":{"N":"2"},"first_name":{"S":"Catalina"},"last_name":{"S":"Smith"},"phone_numbers":{"SS":["+1333333333"]},"profile_pics":{"BS":["U29tZUJ5dGVzRGF0YTM="]},"salaries":{"NS":["3000"]}}}`

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
	DeleteItemTestCase4Output = `{"Attributes":{"address":{"S":"Pune"},"age":{"N":"30"},"emp_id":{"N":"3"},"first_name":{"S":"Alice"},"last_name":{"S":"Trentor"},"phone_numbers":{"SS":["+1444444444","+1555555555"]},"profile_pics":{"BS":["U29tZUJ5dGVzRGF0YTQ=","U29tZUJ5dGVzRGF0YTU="]},"salaries":{"NS":["4000.25","5000.5","6000.75"]}}}`

	DeleteItemTestCase5Name = "5: ConditionExpressionNames with ExpressionAttributeNames & ExpressionAttributeValues"
	DeleteItemTestCase5     = models.Delete{
		TableName: "employee",
		Key: map[string]*dynamodb.AttributeValue{
			"emp_id": {N: aws.String("4")},
		},
		ConditionExpression: "#ag > :val2",
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":val2": {N: aws.String("19.0")},
		},
		ExpressionAttributeNames: map[string]string{
			"#ag": "age",
		},
	}
	DeleteItemTestCase5Output = `{"Attributes":{"address":{"S":"Silicon Valley"},"age":{"N":"40"},"emp_id":{"N":"4"},"first_name":{"S":"Lea"},"last_name":{"S":"Martin"},"phone_numbers":{"SS":["+1666666666"]},"profile_pics":{"BS":["U29tZUJ5dGVzRGF0YTY="]},"salaries":{"NS":["7000","8000.25"]}}}`

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
	DeleteItemTestCaseListName = "ConditionExpression with ExpressionAttributeValues for List"
	DeleteItemTestCaseList     = models.Delete{
		TableName: "test_table",
		Key: map[string]*dynamodb.AttributeValue{
			"rank_list": {S: aws.String("rank_list")},
		},
	}
	DeleteItemTestCaseListOutput = `{"Attributes":{"category":{"S":"category"},"id":{"S":"testing"},"list_type":{"L":[{"S":"John Doe"},{"S":"62536"},{"BOOL":true}]},"rank_list":{"S":"rank_list"},"updated_at":{"S":"2024-12-04T11:02:02Z"}}}`
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
							"emp_id":        {N: aws.String("6")},
							"age":           {N: aws.String("60")},
							"address":       {S: aws.String("London")},
							"first_name":    {S: aws.String("David")},
							"last_name":     {S: aws.String("Root")},
							"phone_numbers": {SS: []*string{aws.String("+1777777777"), aws.String("+1888888888")}},
							"profile_pics":  {BS: [][]byte{[]byte("U29tZUJ5dGVzRGF0YTc="), []byte("U29tZUJ5dGVzRGF0YTg=")}},
							"salaries":      {NS: []*string{aws.String("9000.50"), aws.String("10000.75")}},
						},
					},
				},
				{
					PutReq: models.BatchPutItem{
						Item: map[string]*dynamodb.AttributeValue{
							"emp_id":        {N: aws.String("7")},
							"age":           {N: aws.String("70")},
							"address":       {S: aws.String("Paris")},
							"first_name":    {S: aws.String("Marc")},
							"last_name":     {S: aws.String("Ponting")},
							"phone_numbers": {SS: []*string{aws.String("+1999999999"), aws.String("+2111111111")}},
							"profile_pics":  {BS: [][]byte{[]byte("U29tZUJ5dGVzRGF0YTk="), []byte("U29tZUJ5dGVzRGF0YTEw=")}},
							"salaries":      {NS: []*string{aws.String("11000"), aws.String("12000.25")}},
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
			"mapdynamo": {
				{
					PutReq: models.BatchPutItem{
						Item: map[string]*dynamodb.AttributeValue{
							"address": {
								M: map[string]*dynamodb.AttributeValue{
									"permanent_address": {
										S: aws.String("789 Elm St, Springfield, SP"),
									},
									"present_address": {
										S: aws.String("101 Maple Ave, Metropolis, MP"),
									},
									"mobilenumber": {
										N: aws.String("9035599089"),
									},
									"active": {
										BOOL: aws.Bool(true),
									},
									"notes": {
										B: []byte("YmluYXJ5X2RhdGE="),
									},
									"additional_details": {
										M: map[string]*dynamodb.AttributeValue{
											"landmark": {
												S: aws.String("Near Central Park"),
											},
											"apartment_number": {
												S: aws.String("5B"),
											},
											"landmark notes": {
												B: []byte("YmluYXJ5X2RhdGE="),
											},
											"additional_details_2": {
												M: map[string]*dynamodb.AttributeValue{
													"landmark_field": {
														S: aws.String("street 7"),
													},
													"landmark_field_number": {
														N: aws.String("1001"),
													},
												},
											},
										},
									},
								},
							},
							"contact_ranking_list": {
								S: aws.String("1,2,3"),
							},
							"context": {
								S: aws.String("user-profile"),
							},
							"guid": {
								S: aws.String("123e4567-e89b-12d3-a456-value002"),
							},
							"name": {
								S: aws.String("Jane Smith"),
							},
						},
					},
				},
				{
					PutReq: models.BatchPutItem{
						Item: map[string]*dynamodb.AttributeValue{
							"address": {
								M: map[string]*dynamodb.AttributeValue{
									"permanent_address": {
										S: aws.String("789 Elm St, Springfield, SP"),
									},
									"present_address": {
										S: aws.String("101 Maple Ave, Metropolis, MP"),
									},
									"mobilenumber": {
										N: aws.String("9035599089"),
									},
									"active": {
										BOOL: aws.Bool(true),
									},
									"notes": {
										B: []byte("YmluYXJ5X2RhdGE="),
									},
									"additional_details": {
										M: map[string]*dynamodb.AttributeValue{
											"landmark": {
												S: aws.String("Near Central Park"),
											},
											"apartment_number": {
												S: aws.String("5B"),
											},
											"landmark notes": {
												B: []byte("YmluYXJ5X2RhdGE="),
											},
											"additional_details_2": {
												M: map[string]*dynamodb.AttributeValue{
													"landmark_field": {
														S: aws.String("street 7"),
													},
													"landmark_field_number": {
														N: aws.String("1001"),
													},
												},
											},
										},
									},
								},
							},
							"contact_ranking_list": {
								S: aws.String("1,2,3"),
							},
							"context": {
								S: aws.String("user-profile"),
							},
							"guid": {
								S: aws.String("123e4567-e89b-12d3-a456-value003"),
							},
							"name": {
								S: aws.String("Jane Smith"),
							},
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
							"emp_id": {N: aws.String("4")},
						},
					},
				},
				{
					DelReq: models.BatchDeleteItem{
						Key: map[string]*dynamodb.AttributeValue{
							"emp_id": {N: aws.String("5")},
						},
					},
				},
			},
		},
	}

	BatchWriteItemTestCaseListName = "1: Insert and Delete Items in Batch"
	BatchWriteItemTestCaseList     = models.BatchWriteItem{
		RequestItems: map[string][]models.BatchWriteSubItems{
			"test_table": {
				{
					PutReq: models.BatchPutItem{
						Item: map[string]*dynamodb.AttributeValue{
							"id":        {S: aws.String("test_id1")},
							"rank_list": {S: aws.String("rank_list4")},
							"list_type": {
								L: []*dynamodb.AttributeValue{
									{S: aws.String("value1")},
									{N: aws.String("123")},
									{BOOL: aws.Bool(true)},
								},
							},
						},
					},
				},
			},
		},
	}
	BatchWriteItemTestCase11Name = "11: Batch Delete Request for 1 tables"
	BatchWriteItemTestCase11     = models.BatchWriteItem{
		RequestItems: map[string][]models.BatchWriteSubItems{
			"mapdynamo": {
				{
					DelReq: models.BatchDeleteItem{
						Key: map[string]*dynamodb.AttributeValue{
							"guid":    {S: aws.String("123e4567-e89b-12d3-a456-value002")},
							"context": {S: aws.String("user-profile")},
						},
					},
				},
				{
					DelReq: models.BatchDeleteItem{
						Key: map[string]*dynamodb.AttributeValue{
							"rank_list": {S: aws.String("rank_list")},
							"guid":      {S: aws.String("123e4567-e89b-12d3-a456-value003")},
							"context":   {S: aws.String("user-profile")},
						},
					},
				},
			},
		},
	}
	BatchWriteItemTestCase1Output = `{
		"UnprocessedItems": {},
		"ConsumedCapacity": [
			{
				"TableName": "test_table",
				"CapacityUnits": 2
			}
		]
	}`
)

var (
	TestTransactGet1Name = "1: wrong url"
	TestTransactGet1     = models.TransactGetItemsRequest{
		TransactItems: []models.TransactGetItem{
			{Get: models.GetItemRequest{
				TableName: "your_table_name", // add the actual table name here
				Keys: map[string]*dynamodb.AttributeValue{
					"emp_id": {N: aws.String("1")},
				},
			}},
		},
	}

	TestTransactGet2Name = "2: valid request with one item"
	TestTransactGet2     = models.TransactGetItemsRequest{
		TransactItems: []models.TransactGetItem{
			{Get: models.GetItemRequest{
				TableName: "employee", // add the actual table name here
				Keys: map[string]*dynamodb.AttributeValue{
					"emp_id": {N: aws.String("1")},
				},
			}},
		},
	}
	TestTransactGet2Output = `{"Responses":[{"TableName":"employee","Item":{"L":[{"address":{"S":"Shamli"},"age":{"N":"10"},"emp_id":{"N":"1"},"first_name":{"S":"Marc"},"last_name":{"S":"Richards"},"phone_numbers":{"SS":["+1111111111","+1222222222"]},"profile_pics":{"BS":["U29tZUJ5dGVzRGF0YTE=","U29tZUJ5dGVzRGF0YTI="]},"salaries":{"NS":["1000.5","2000.75"]}}]}}]}`

	TestTransactGet3Name = "3: valid request with multiple items"
	TestTransactGet3     = models.TransactGetItemsRequest{
		TransactItems: []models.TransactGetItem{
			{Get: models.GetItemRequest{
				TableName: "employee", // add the actual table name here
				Keys: map[string]*dynamodb.AttributeValue{
					"emp_id": {N: aws.String("1")},
				},
			}},
			{Get: models.GetItemRequest{
				TableName: "department", // add the actual table name here
				Keys: map[string]*dynamodb.AttributeValue{
					"d_id": {N: aws.String("200")},
				},
			}},
		},
	}
	TestTransactGet3Output = `{"Responses":[{"TableName":"employee","Item":{"L":[{"address":{"S":"Shamli"},"age":{"N":"10"},"emp_id":{"N":"1"},"first_name":{"S":"Marc"},"last_name":{"S":"Richards"},"phone_numbers":{"SS":["+1111111111","+1222222222"]},"profile_pics":{"BS":["U29tZUJ5dGVzRGF0YTE=","U29tZUJ5dGVzRGF0YTI="]},"salaries":{"NS":["1000.5","2000.75"]}}]}},{"TableName":"department","Item":{"L":[{"d_id":{"N":"200"},"d_name":{"S":"Arts"},"d_specialization":{"S":"BA"}}]}}]}`

	TestTransactGet4Name = "4: valid request with ProjectionExpression"
	TestTransactGet4     = models.TransactGetItemsRequest{
		TransactItems: []models.TransactGetItem{
			{Get: models.GetItemRequest{
				TableName: "employee", // add the actual table name here
				Keys: map[string]*dynamodb.AttributeValue{
					"emp_id": {N: aws.String("1")},
				},
				ProjectionExpression: "first_name, last_name",
				ExpressionAttributeNames: map[string]string{
					"#fn": "first_name",
					"#ln": "last_name",
				},
			}},
		},
	}
	TestTransactGet4Output = `{"Responses":[{"TableName":"employee","Item":{"L":[{"first_name":{"S":"Marc"},"last_name":{"S":"Richards"}}]}}]}`

	TestTransactGet5Name = "5: valid request with ProjectionExpression and multiple items"
	TestTransactGet5     = models.TransactGetItemsRequest{
		TransactItems: []models.TransactGetItem{
			{Get: models.GetItemRequest{
				TableName: "employee", // add the actual table name here
				Keys: map[string]*dynamodb.AttributeValue{
					"emp_id": {N: aws.String("1")},
				},
				ProjectionExpression: "first_name, last_name",
				ExpressionAttributeNames: map[string]string{
					"#fn": "first_name",
					"#ln": "last_name",
				},
			}},
			{Get: models.GetItemRequest{
				TableName: "department", // add the actual table name here
				Keys: map[string]*dynamodb.AttributeValue{
					"d_id": {N: aws.String("200")},
				},
				ProjectionExpression: "d_name, d_specialization",
				ExpressionAttributeNames: map[string]string{
					"#dn": "d_name",
					"#ds": "d_specialization",
				},
			}},
		},
	}
	TestTransactGet5Output = `{"Responses":[{"TableName":"employee","Item":{"L":[{"first_name":{"S":"Marc"},"last_name":{"S":"Richards"}}]}},{"TableName":"department","Item":{"L":[{"d_name":{"S":"Arts"},"d_specialization":{"S":"BA"}}]}}]}`
)

var (
	TestTransactWrite1Name = "1: wrong url"
	TestTransactWrite1     = models.TransactWriteItemsRequest{
		TransactItems: []models.TransactWriteItem{
			{Put: models.PutItemRequest{
				TableName: "your_table_name", // add the actual table name here
				Item: map[string]*dynamodb.AttributeValue{
					"emp_id": {N: aws.String("1")},
					// ... other attributes
				},
			}},
		},
	}

	TestTransactWrite2Name = "2: valid request with one Put item"
	TestTransactWrite2     = models.TransactWriteItemsRequest{
		TransactItems: []models.TransactWriteItem{
			{Put: models.PutItemRequest{
				TableName: "employee",
				Item: map[string]*dynamodb.AttributeValue{
					"emp_id":     {N: aws.String("5")},
					"first_name": {S: aws.String("John")},
					"last_name":  {S: aws.String("Doe")},
					"age":        {N: aws.String("30")},
					"address":    {S: aws.String("123 Main St")},
				},
			}},
		},
	}
	TestTransactWrite2Output = `[{"Put":{"address":{"S":"123 Main St"},"age":{"N":"30"},"emp_id":{"N":"5"},"first_name":{"S":"John"},"last_name":{"S":"Doe"}}}]`

	TestTransactWrite3Name = "3: valid request with multiple items (Put, Update, Delete)"
	TestTransactWrite3     = models.TransactWriteItemsRequest{
		TransactItems: []models.TransactWriteItem{
			{Put: models.PutItemRequest{
				TableName: "employee",
				Item: map[string]*dynamodb.AttributeValue{
					"emp_id":     {N: aws.String("6")},
					"first_name": {S: aws.String("Alice")},
					"last_name":  {S: aws.String("Smith")},
					"age":        {N: aws.String("25")},
					"address":    {S: aws.String("456 Oak Ave")},
				},
			}},
			{Update: models.UpdateAttr{
				TableName: "employee",
				Key: map[string]*dynamodb.AttributeValue{
					"emp_id": {N: aws.String("5")},
				},
				UpdateExpression: "SET age = :age",
				ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
					":age": {N: aws.String("31")},
				},
			}},
			{Delete: models.DeleteItemRequest{
				TableName: "employee",
				Key: map[string]*dynamodb.AttributeValue{
					"emp_id": {N: aws.String("1")},
				},
			}},
		},
	}
	TestTransactWrite3Output = `[{"Put":{"address":{"S":"456 Oak Ave"},"age":{"N":"25"},"emp_id":{"N":"6"},"first_name":{"S":"Alice"},"last_name":{"S":"Smith"}}},{"Update":{"Attributes":{}}},{"Delete":{}}]`
	TestTransactWrite4Name   = "4: valid request with ConditionCheck"
	TestTransactWrite4       = models.TransactWriteItemsRequest{
		TransactItems: []models.TransactWriteItem{
			{ConditionCheck: models.ConditionCheckRequest{
				TableName: "employee",
				Key: map[string]*dynamodb.AttributeValue{
					"emp_id": {N: aws.String("5")},
				},
				ConditionExpression: "age = :age",
				ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
					":age": {N: aws.String("30")},
				},
			}},
			{Put: models.PutItemRequest{
				TableName: "employee",
				Item: map[string]*dynamodb.AttributeValue{
					"emp_id":     {N: aws.String("7")},
					"first_name": {S: aws.String("Bob")},
					"last_name":  {S: aws.String("Johnson")},
					"age":        {N: aws.String("40")},
					"address":    {S: aws.String("789 Elm St")},
				},
			}},
		},
	}
	TestTransactWrite4Output = `[{"Put":{"address":{"S":"789 Elm St"},"age":{"N":"40"},"emp_id":{"N":"7"},"first_name":{"S":"Bob"},"last_name":{"S":"Johnson"}}}]`
)

// test Data for ExecuteStatement API
var (
	ExecuteStatementTestCase1Name = "1: Insert Query for ExecuteStatement"
	ExecuteStatementCase1         = models.ExecuteStatement{
		Statement: "INSERT INTO employee VALUE {'emp_id': 9, 'first_name': 'Marc', 'last_name': 'Richards1', 'age': 10, 'address': 'Shamli'}",
	}

	ExecuteStatementTestCase2Name = "2: Insert Query for ExecuteStatement"
	ExecuteStatementCase2         = models.ExecuteStatement{
		Statement: "INSERT INTO employee VALUE {'emp_id': ?, 'first_name': ?, 'last_name': ?, 'age': ?, 'address': ?}",
		Parameters: []*dynamodb.AttributeValue{
			{
				N: aws.String("10"),
			},
			{
				S: aws.String("Shoaib"),
			},
			{
				S: aws.String("Jarman"),
			},
			{
				N: aws.String("32"),
			},
			{
				S: aws.String("Rampur"),
			},
		},
	}
	ExecuteStatementTestCase3Name = "3: Select Query for ExecuteStatement Non Parameterised"
	ExecuteStatementCase3         = models.ExecuteStatement{
		Limit:     10,
		Statement: "SELECT * FROM employee WHERE emp_id = 1",
	}
	ExecuteStatementCase3Output = `{"Items":[{"address":{"S":"Shamli"},"age":{"N":"10"},"emp_id":{"N":"1"},"first_name":{"S":"Marc"},"last_name":{"S":"Richards"}}]}`

	ExecuteStatementTestCase4Name = "4: Select Query for ExecuteStatement Parameterised Statement"
	ExecuteStatementCase4         = models.ExecuteStatement{
		Limit:     10,
		Statement: "SELECT * FROM employee WHERE emp_id = ?",
		Parameters: []*dynamodb.AttributeValue{
			{
				N: aws.String("1"),
			},
		},
	}
	ExecuteStatementTestCase5Name = "5: Update Query for ExecuteStatement Non Parameterised"
	ExecuteStatementCase5         = models.ExecuteStatement{
		Statement: "UPDATE employee SET age = 11, address = 'New Shamli' WHERE emp_id = 9",
	}

	ExecuteStatementTestCase6Name = "5: Update Query for ExecuteStatement Non Parameterised"
	ExecuteStatementCase6         = models.ExecuteStatement{
		Statement: "UPDATE employee SET address = ?, age = ? WHERE emp_id = ?",
		Parameters: []*dynamodb.AttributeValue{
			{
				S: aws.String("Bengaluru"),
			},
			{
				N: aws.String("33"),
			},
			{
				N: aws.String("10"),
			},
		},
	}

	ExecuteStatementTestCase7Name = "5: Delete Query for ExecuteStatement Non Parameterised"
	ExecuteStatementCase7         = models.ExecuteStatement{
		Statement: "DELETE FROM employee WHERE emp_id = 9",
	}

	ExecuteStatementTestCase8Name = "5: Delete Query for ExecuteStatement Non Parameterised"
	ExecuteStatementCase8         = models.ExecuteStatement{
		Statement: "DELETE FROM employee WHERE emp_id = ?",
		Parameters: []*dynamodb.AttributeValue{
			{
				N: aws.String("10"),
			},
		},
	}
)

func handlerInitFunc() *gin.Engine {
	initErr := initializer.InitAll("../config.yaml")
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
		createPostTestCase("Crorect data with  ExpressionAttributeNames Testcase", "/v1", "GetItem", getItemTest4Output, getItemTest4),
		createPostTestCase("Crorect data with  ExpressionAttributeNames values not passed Testcase", "/v1", "GetItem", getItemTest5Output, getItemTest5),
		createPostTestCase("Crorect Data TestCase for Map", "/v1", "GetItem", getItemTestForMapOutput, getItemTestForMap),
		createPostTestCase("Correct data with NULL value Testcase", "/v1", "GetItem", getItemTest6Output, getItemTest6),
		createPostTestCase("Crorect data for List Data Type", "/v1", "GetItem", getItemTestForListOutput, getItemTestForList),
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
		createPostTestCase(TestGetBatchForListName, "/v1", "BatchGetItem", TestGetBatchForListOutput, TestGetBatchForList),
		createPostTestCase(TestGetBatch11Name, "/v1", "BatchGetItem", TestGetBatch11Output, TestGetBatch11),
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
		createPostTestCase("Query with NULL value in KeyConditionExpression", "/v1", "Query", queryTestCaseOutput17, queryTestCase17),
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
		createPostTestCase(ScanTestCase14Name, "/v1", "Scan", ScanTestCase14Output, ScanTestCase14),
		createPostTestCase(ScanTestCaseListName, "/v1", "Query", ScanTestCaseListOutput, ScanTestCaseList),
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
		createPostTestCase(UpdateItemTestForListName, "/v1", "UpdateItem", UpdateItemTestForListOutput, UpdateItemTestForList),
		createStatusCheckPostTestCase(UpdateItemTestCase11Name, "/v1", "UpdateItem", http.StatusOK, UpdateItemTestCase11),
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
		createPostTestCase(PutItemTestForListName, "/v1", "PutItem", PutItemTestForListOutput, PutItemTestForList),
		createPostTestCase(PutItemTestCase10Name, "/v1", "PutItem", PutItemTestCase10Output, PutItemTestCase10),
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
		createPostTestCase(DeleteItemTestCaseListName, "/v1", "DeleteItem", DeleteItemTestCaseListOutput, DeleteItemTestCaseList),
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
		createStatusCheckPostTestCase(BatchWriteItemTestCase9Name, "/v1", "BatchWriteItem", http.StatusOK, BatchWriteItemTestCase9),
		createStatusCheckPostTestCase(BatchWriteItemTestCase10Name, "/v1", "BatchWriteItem", http.StatusBadRequest, BatchWriteItemTestCase10),
		createStatusCheckPostTestCase(BatchWriteItemTestCaseListName, "/v1", "BatchWriteItem", http.StatusOK, BatchWriteItemTestCaseList),
		createStatusCheckPostTestCase(BatchWriteItemTestCase11Name, "/v1", "BatchWriteItem", http.StatusOK, BatchWriteItemTestCase11),
	}
	apitest.RunTests(t, tests)
}
func testExecuteStatementAPI(t *testing.T) {
	apitest := apitesting.APITest{
		GetHTTPHandler: func(ctx context.Context, t *testing.T) http.Handler {
			return handlerInitFunc()
		},
	}
	tests := []apitesting.APITestCase{
		createStatusCheckPostTestCase(ExecuteStatementTestCase1Name, "/v1", "ExecuteStatement", http.StatusOK, ExecuteStatementCase1),
		createStatusCheckPostTestCase(ExecuteStatementTestCase2Name, "/v1", "ExecuteStatement", http.StatusOK, ExecuteStatementCase2),
		createStatusCheckPostTestCase(ExecuteStatementTestCase5Name, "/v1", "ExecuteStatement", http.StatusOK, ExecuteStatementCase5),
		createStatusCheckPostTestCase(ExecuteStatementTestCase6Name, "/v1", "ExecuteStatement", http.StatusOK, ExecuteStatementCase6),

		createPostTestCase(ExecuteStatementTestCase3Name, "/v1", "ExecuteStatement", ExecuteStatementCase3Output, ExecuteStatementCase3),
		createPostTestCase(ExecuteStatementTestCase4Name, "/v1", "ExecuteStatement", ExecuteStatementCase3Output, ExecuteStatementCase4),

		createStatusCheckPostTestCase(ExecuteStatementTestCase7Name, "/v1", "ExecuteStatement", http.StatusOK, ExecuteStatementCase7),
		createStatusCheckPostTestCase(ExecuteStatementTestCase8Name, "/v1", "ExecuteStatement", http.StatusOK, ExecuteStatementCase8),
	}
	apitest.RunTests(t, tests)
}

func testTransactGetAPI(t *testing.T) {
	apitest := apitesting.APITest{
		GetHTTPHandler: func(ctx context.Context, t *testing.T) http.Handler {
			return handlerInitFunc()
		},
	}
	tests := []apitesting.APITestCase{
		{
			Name:    TestTransactGet1Name,
			ReqType: "POST",
			PopulateHeaders: func(ctx context.Context, t *testing.T) map[string]string {
				return map[string]string{
					"Content-Type": "application/json",
					"X-Amz-Target": "DynamoDB_20120810.TransactGetItems",
				}
			},
			ResourcePath: func(ctx context.Context, t *testing.T) string { return "/v1/WrongPath" },
			PopulateJSON: func(ctx context.Context, t *testing.T) interface{} {
				return TestTransactGet1
			},
			ExpHTTPStatus: http.StatusNotFound,
		},
		createPostTestCase(TestTransactGet2Name, "/v1", "TransactGetItems", TestTransactGet2Output, TestTransactGet2),
		createPostTestCase(TestTransactGet3Name, "/v1", "TransactGetItems", TestTransactGet3Output, TestTransactGet3),
		createPostTestCase(TestTransactGet4Name, "/v1", "TransactGetItems", TestTransactGet4Output, TestTransactGet4),
		createPostTestCase(TestTransactGet5Name, "/v1", "TransactGetItems", TestTransactGet5Output, TestTransactGet5),
	}
	apitest.RunTests(t, tests)
}

func testTransactWriteItemsAPI(t *testing.T) {
	apitest := apitesting.APITest{
		GetHTTPHandler: func(ctx context.Context, t *testing.T) http.Handler {
			return handlerInitFunc()
		},
	}
	tests := []apitesting.APITestCase{
		{
			Name:    TestTransactWrite1Name,
			ReqType: "POST",
			PopulateHeaders: func(ctx context.Context, t *testing.T) map[string]string {
				return map[string]string{
					"Content-Type": "application/json",
					"X-Amz-Target": "DynamoDB_20120810.TransactWriteItems",
				}
			},
			ResourcePath: func(ctx context.Context, t *testing.T) string { return "/v1/WrongPath" },
			PopulateJSON: func(ctx context.Context, t *testing.T) interface{} {
				return TestTransactWrite1
			},
			ExpHTTPStatus: http.StatusNotFound,
		},
		createPostTestCase(TestTransactWrite2Name, "/v1", "TransactWriteItems", TestTransactWrite2Output, TestTransactWrite2),
		createPostTestCase(TestTransactWrite3Name, "/v1", "TransactWriteItems", TestTransactWrite3Output, TestTransactWrite3),
		createPostTestCase(TestTransactWrite4Name, "/v1", "TransactWriteItems", TestTransactWrite4Output, TestTransactWrite4),
	}
	apitest.RunTests(t, tests)
}

func TestApi(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration tests in short mode")
	}

	config, err := LoadConfig("../config.yaml")
	if err != nil {
		log.Fatalf("Error loading configuration: %v", err)
	}
	// Build the Spanner database name
	databaseName = fmt.Sprintf(
		"projects/%s/instances/%s/databases/%s",
		config.Spanner.ProjectID, config.Spanner.InstanceID, config.Spanner.DatabaseName,
	)

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
		"TransactGetItems",
		"TestTransactWriteItemsAPI",
	}

	var tests = map[string]func(t *testing.T){
		"GetItemAPI":                testGetItemAPI,
		"GetBatchAPI":               testGetBatchAPI,
		"QueryAPI":                  testQueryAPI,
		"ScanAPI":                   testScanAPI,
		"UpdateItemAPI":             testUpdateItemAPI,
		"PutItemAPI":                testPutItemAPI,
		"DeleteItemAPI":             testDeleteItemAPI,
		"BatchWriteItemAPI":         testBatchWriteItemAPI,
		"TransactGetItems":          testTransactGetAPI,
		"TestTransactWriteItemsAPI": testTransactWriteItemsAPI,
		"ExecuteStatementAPI":       testExecuteStatementAPI,
	}

	// run the tests
	for _, testName := range testNames {
		t.Run(testName, tests[testName])
	}
}
