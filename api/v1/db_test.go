package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/cloudspannerecosystem/dynamodb-adapter/models"
	"github.com/cloudspannerecosystem/dynamodb-adapter/service/services"
	"github.com/cloudspannerecosystem/dynamodb-adapter/storage"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"
	"github.com/tj/assert"
)

// MockService struct
type MockService struct {
	mock.Mock
}

type MockStorage struct {
	mock.Mock
}
type MockSpannerClient struct {
	mock.Mock
}

func (m *MockStorage) GetStorageInstance(ctx context.Context, tableName string, keys []map[string]interface{}, projection string, expressionNames map[string]string) ([]map[string]interface{}, error) {
	args := m.Called(ctx, tableName, keys, projection, expressionNames)
	return args.Get(0).([]map[string]interface{}), args.Error(1)
}

// Mock TransactGetItems method
func (m *MockService) TransactGetItem(ctx context.Context, getRequest models.GetItemRequest, keyMapArray []map[string]interface{}, projectionExpression string, expressionAttributeNames map[string]string) ([]map[string]interface{}, error) {
	args := m.Called(ctx, getRequest, keyMapArray, projectionExpression, expressionAttributeNames)
	return args.Get(0).([]map[string]interface{}), args.Error(1)
}

func (m *MockService) MayIReadOrWrite(tableName string, isWrite bool, user string) bool {
	args := m.Called(tableName, isWrite, user)
	return args.Bool(0)
}

func (m *MockService) ChangeMaptoDynamoMap(input interface{}) (map[string]interface{}, error) {
	args := m.Called(input)
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

// Test case for TransactGetItems

func TestTransactGetItems_ValidRequestWithMultipleItems(t *testing.T) {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)

	mockSvc := new(MockService)
	mockStorage := new(MockStorage)
	mockStorageInstance := &storage.Storage{}

	mockStorage.On("GetSpannerClient").Return(&MockSpannerClient{})
	mockStorage.On("InitializeDriver").Return()

	// Mocking service methods
	mockStorage.On("GetStorageInstance").Return(mockStorage)
	mockSvc.On("MayIReadOrWrite", "employee", false, "").Return(true)

	// Mock TransactGetItems response
	mockSvc.On("TransactGetItem",
		mock.Anything,
		mock.Anything,
		mock.Anything, // Use mock.Anything for keyMapArray for now
		mock.Anything, // Correctly match the empty string
		mock.Anything,
	).Return([]map[string]interface{}{
		{
			"Item": map[string]interface{}{
				"L": []interface{}{
					map[string]interface{}{
						"emp_id":     map[string]interface{}{"N": "1"},
						"first_name": map[string]interface{}{"S": "John"},
						"last_name":  map[string]interface{}{"S": "Doe"},
					},
				},
			},
		},
		{
			"Item": map[string]interface{}{
				"L": []interface{}{
					map[string]interface{}{
						"emp_id":     map[string]interface{}{"N": "2"},
						"first_name": map[string]interface{}{"S": "Jane"},
						"last_name":  map[string]interface{}{"S": "Smith"},
					},
				},
			},
		},
	}, nil).Twice()

	h := &APIHandler{svc: mockSvc}
	// Create request payload
	transactGetMeta := models.TransactGetItemsRequest{
		TransactItems: []models.TransactGetItem{
			{
				Get: models.GetItemRequest{
					TableName: "employee",
					Keys: map[string]*dynamodb.AttributeValue{
						"emp_id": {N: aws.String("1")},
					},
				},
			},
			{
				Get: models.GetItemRequest{
					TableName: "employee",
					Keys: map[string]*dynamodb.AttributeValue{
						"emp_id": {N: aws.String("2")},
					},
				},
			},
		},
	}
	//models.TableColChangeMap[tableName]
	reqBody, _ := json.Marshal(transactGetMeta)

	// Set request in context
	c.Request, _ = http.NewRequest("POST", "/transact-get-items", bytes.NewBuffer(reqBody))

	// Set mock service in context
	services.SetServiceInstance(mockSvc)
	storage.SetStorageInstance(mockStorageInstance)
	//	apiHandler := NewAPIHandler(mockService)
	h.TransactGetItems(c)

	// Assertions
	assert.Equal(t, http.StatusOK, recorder.Code)

	var responseBody models.TransactGetItemsResponse // Use the correct response type
	err := json.Unmarshal(recorder.Body.Bytes(), &responseBody)
	assert.NoError(t, err, "Response should be valid JSON")
	// Access the Responses field
	responses := responseBody.Responses

	// Check that response contains expected employee data
	expectedEmployees := []map[string]interface{}{
		{"emp_id": map[string]interface{}{"N": "1"}, "first_name": map[string]interface{}{"S": "John"}, "last_name": map[string]interface{}{"S": "Doe"}},
		{"emp_id": map[string]interface{}{"N": "2"}, "first_name": map[string]interface{}{"S": "Jane"}, "last_name": map[string]interface{}{"S": "Smith"}},
	}

	assert.Equal(t, len(expectedEmployees), len(responses), "Response should contain correct number of items")
	for _, response := range responses {

		employeeData, exists := response.Item["L"].([]interface{})
		assert.True(t, exists, "Response should contain 'L' key at the top level")

		for _, item := range employeeData {
			itemMap, ok := item.(map[string]interface{})
			assert.True(t, ok, "Item should be a map")

			nestedItem, exists := itemMap["Item"].(map[string]interface{})
			assert.True(t, exists, "Item should contain 'Item' key")

			nestedL, exists := nestedItem["L"].(map[string]interface{})
			assert.True(t, exists, "Item should contain 'L' key")

			finalList, exists := nestedL["L"].([]interface{})
			assert.True(t, exists, "Final 'L' should be a list")

			assert.Equal(t, 1, len(finalList), "Final list should contain one employee")

			finalEmployeeMap, exists := finalList[0].(map[string]interface{})
			assert.True(t, exists, "Final list should contain a map with employee attributes")

			expectedEmployees := []map[string]interface{}{
				{"emp_id": map[string]interface{}{"N": map[string]interface{}{"S": "1"}},
					"first_name": map[string]interface{}{"S": map[string]interface{}{"S": "John"}},
					"last_name":  map[string]interface{}{"S": map[string]interface{}{"S": "Doe"}}},
				{"emp_id": map[string]interface{}{"N": map[string]interface{}{"S": "2"}},
					"first_name": map[string]interface{}{"S": map[string]interface{}{"S": "Jane"}},
					"last_name":  map[string]interface{}{"S": map[string]interface{}{"S": "Smith"}}},
			}

			assert.Contains(t, expectedEmployees, finalEmployeeMap, "Response should match expected employees")
		}
	}

	// Verify that the mock expectations were met
	mockSvc.AssertExpectations(t)
}
