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
