package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"testing"

	rice "github.com/GeertJohan/go.rice"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/cloudspannerecosystem/dynamodb-adapter/api"
	"github.com/cloudspannerecosystem/dynamodb-adapter/apitesting"
	"github.com/cloudspannerecosystem/dynamodb-adapter/initializer"
	"github.com/cloudspannerecosystem/dynamodb-adapter/models"
	httpexpect "github.com/gavv/httpexpect/v2"
	"github.com/gin-gonic/gin"
)

const (
	apiURL  = "http://127.0.0.1:9050"
	version = "v1"
)

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

func initFunc() *gin.Engine {
	box := rice.MustFindBox("config-files")

	initErr := initializer.InitAll(box)
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

func createPostTestCase(name, url, outputString string, input interface{}) apitesting.APITestCase {
	return apitesting.APITestCase{
		Name:    name,
		ReqType: "POST",
		PopulateHeaders: func(ctx context.Context, t *testing.T) map[string]string {
			return map[string]string{
				"Content-Type": "application/json",
			}
		},
		ResourcePath: func(ctx context.Context, t *testing.T) string { return url },
		PopulateJSON: func(ctx context.Context, t *testing.T) interface{} {
			return input
		},
		ExpHTTPStatus: http.StatusOK,
		ValidateResponse: func(ctx context.Context, t *testing.T, resp *httpexpect.Response) context.Context {
			fmt.Println(resp.Body())
			resp.Body().Equal(outputString)
			return ctx
		},
	}
}

func TestGetItemAPI(t *testing.T) {
	apitest := apitesting.APITest{
		// APIEndpointURL: apiURL + "/" + version,
		GetHTTPHandler: func(ctx context.Context, t *testing.T) http.Handler {
			return initFunc()
		},
	}
	tests := []apitesting.APITestCase{
		{
			Name:    "Wrong URL (404 Error)",
			ReqType: "POST",
			PopulateHeaders: func(ctx context.Context, t *testing.T) map[string]string {
				return map[string]string{
					"Content-Type": "application/json",
				}
			},
			ResourcePath: func(ctx context.Context, t *testing.T) string { return "/v1/GetIte" },
			PopulateJSON: func(ctx context.Context, t *testing.T) interface{} {
				return getItemTest2
			},
			ExpHTTPStatus: http.StatusNotFound,
		},
		{
			Name:    "Wrong Pramamerter(Bad Request)",
			ReqType: "POST",
			PopulateHeaders: func(ctx context.Context, t *testing.T) map[string]string {
				return map[string]string{
					"Content-Type": "application/json",
				}
			},
			ResourcePath: func(ctx context.Context, t *testing.T) string { return "/v1/GetItem" },
			PopulateJSON: func(ctx context.Context, t *testing.T) interface{} {
				return getItemTest1
			},
			ExpHTTPStatus: http.StatusBadRequest,
		},
		{
			Name:    "Wrong Pramamerter(Key value is not passed)",
			ReqType: "POST",
			PopulateHeaders: func(ctx context.Context, t *testing.T) map[string]string {
				return map[string]string{
					"Content-Type": "application/json",
				}
			},
			ResourcePath: func(ctx context.Context, t *testing.T) string { return "/v1/GetItem" },
			PopulateJSON: func(ctx context.Context, t *testing.T) interface{} {
				return getItemTest1
			},
			ExpHTTPStatus: http.StatusBadRequest,
		},
		createPostTestCase("Crorect Data TestCase", "/v1/GetItem", getItemTest2Output, getItemTest2),
		createPostTestCase("Crorect data with Projection param Testcase", "/v1/GetItem", getItemTest3Output, getItemTest3),
		createPostTestCase("Crorect data with  ExpressionAttributeNames Testcase", "/v1/GetItem", getItemTest3Output, getItemTest4),
		createPostTestCase("Crorect data with  ExpressionAttributeNames values not passed Testcase", "/v1/GetItem", getItemTest5Output, getItemTest5),
	}
	apitest.RunTests(t, tests)
}

func TestGetBatchAPI(t *testing.T) {
	apitest := apitesting.APITest{
		// APIEndpointURL: apiURL + "/" + version,
		GetHTTPHandler: func(ctx context.Context, t *testing.T) http.Handler {
			return initFunc()
		},
	}
	tests := []apitesting.APITestCase{
		{
			Name:    TestGetBatch1Name,
			ReqType: "POST",
			PopulateHeaders: func(ctx context.Context, t *testing.T) map[string]string {
				return map[string]string{
					"Content-Type": "application/json",
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
				}
			},
			ResourcePath: func(ctx context.Context, t *testing.T) string { return "/v1/BatchGetItem" },
			PopulateJSON: func(ctx context.Context, t *testing.T) interface{} {
				return TestGetBatch10
			},
			ExpHTTPStatus: http.StatusBadRequest,
		},
		createPostTestCase(TestGetBatch2Name, "/v1/BatchGetItem", TestGetBatch2Output, TestGetBatch2),
		createPostTestCase(TestGetBatch3Name, "/v1/BatchGetItem", TestGetBatch3Output, TestGetBatch3),
		createPostTestCase(TestGetBatch4Name, "/v1/BatchGetItem", TestGetBatch4Output, TestGetBatch4),
		createPostTestCase(TestGetBatch5Name, "/v1/BatchGetItem", TestGetBatch5Output, TestGetBatch5),
		createPostTestCase(TestGetBatch6Name, "/v1/BatchGetItem", TestGetBatch6Output, TestGetBatch6),
		createPostTestCase(TestGetBatch7Name, "/v1/BatchGetItem", TestGetBatch7Output, TestGetBatch7),
		createPostTestCase(TestGetBatch8Name, "/v1/BatchGetItem", TestGetBatch8Output, TestGetBatch8),
		createPostTestCase(TestGetBatch9Name, "/v1/BatchGetItem", TestGetBatch9Output, TestGetBatch9),
	}

	apitest.RunTests(t, tests)
}
