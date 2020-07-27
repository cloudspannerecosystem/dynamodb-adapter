package apitesting

import (
	"context"
	"net/http"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/cloudspannerecosystem/dynamodb-adapter/models"
	httpexpect "github.com/gavv/httpexpect/v2"
)

const (
	apiURL  = "http://127.0.0.1:9050"
	version = "v1"
)

var (
	getItemTest1 = models.GetItemMeta{
		TableName: "employee",
	}
	getItemTest2 = models.GetItemMeta{
		TableName: "employee",
		Key: map[string]*dynamodb.AttributeValue{
			"emp_id": {N: aws.String("2")},
		},
	}
	getItemTest2Output = `{"address":"Ney York","age":20,"emp_id":2,"first_name":"Catalina","last_name":"Smith"}`
	getItemTest3       = models.GetItemMeta{
		TableName: "employee",
		Key: map[string]*dynamodb.AttributeValue{
			"emp_id": {N: aws.String("2")},
		},
		ProjectionExpression: "emp_id, address",
	}
	getItemTest3Output = `{"address":"Ney York","emp_id":2}`
)

func TestGetItemAPI(t *testing.T) {
	apitest := APITest{
		APIEndpointURL: apiURL + "/" + version,
	}
	tests := []APITestCase{
		{
			Name:    "Wrong URL (404 Error)",
			ReqType: "POST",
			PopulateHeaders: func(ctx context.Context, t *testing.T) map[string]string {
				return map[string]string{
					"Content-Type": "application/json",
				}
			},
			ResourcePath: func(ctx context.Context, t *testing.T) string { return "/GetIte" },
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
			ResourcePath: func(ctx context.Context, t *testing.T) string { return "/GetItem" },
			PopulateJSON: func(ctx context.Context, t *testing.T) interface{} {
				return getItemTest1
			},
			ExpHTTPStatus: http.StatusBadRequest,
		},
		{
			Name:    "Crorect Data TestCase",
			ReqType: "POST",
			PopulateHeaders: func(ctx context.Context, t *testing.T) map[string]string {
				return map[string]string{
					"Content-Type": "application/json",
				}
			},
			ResourcePath: func(ctx context.Context, t *testing.T) string { return "/GetItem" },
			PopulateJSON: func(ctx context.Context, t *testing.T) interface{} {
				return getItemTest2
			},
			ExpHTTPStatus: http.StatusOK,
			ValidateResponse: func(ctx context.Context, t *testing.T, resp *httpexpect.Response) context.Context {
				resp.Body().Equal(getItemTest2Output)
				return ctx
			},
		},
		{
			Name:    "Crorect data with Projection param Testcase",
			ReqType: "POST",
			PopulateHeaders: func(ctx context.Context, t *testing.T) map[string]string {
				return map[string]string{
					"Content-Type": "application/json",
				}
			},
			ResourcePath: func(ctx context.Context, t *testing.T) string { return "/GetItem" },
			PopulateJSON: func(ctx context.Context, t *testing.T) interface{} {
				return getItemTest3
			},
			ExpHTTPStatus: http.StatusOK,
			ValidateResponse: func(ctx context.Context, t *testing.T, resp *httpexpect.Response) context.Context {
				resp.Body().Equal(getItemTest3Output)
				return ctx
			},
		},
	}
	apitest.RunTests(t, tests)
}
