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
func (m *MockService) TransactGetItem(ctx context.Context, tableProjectionCols map[string][]string, pValues map[string]interface{}, sValues map[string]interface{}) ([]map[string]interface{}, error) {
	args := m.Called(ctx, tableProjectionCols, pValues, sValues)
	return args.Get(0).([]map[string]interface{}), args.Error(1)
}

func (m *MockService) TransactGetProjectionCols(ctx context.Context, transactGetMeta models.GetItemRequest) ([]string, []interface{}, []interface{}, error) {
	args := m.Called(ctx, transactGetMeta)
	return args.Get(0).([]string), args.Get(1).([]interface{}), args.Get(2).([]interface{}), args.Error(3)
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

	mockSvc.On("TransactGetProjectionCols",
		mock.Anything,
		mock.AnythingOfType("models.GetItemRequest"),
	).Return(
		[]string{},
		[]interface{}{},
		[]interface{}{},
		nil,
	)

	response1 := []map[string]interface{}{
		{
			"Item": map[string]interface{}{
				"emp_id":     map[string]interface{}{"N": "1"},
				"first_name": map[string]interface{}{"S": "John"},
				"last_name":  map[string]interface{}{"S": "Doe"},
			},
		},
	}
	// Mock TransactGetItems response
	mockSvc.On("TransactGetItem", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(response1, nil).Once()

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

	mockSvc.AssertNumberOfCalls(t, "TransactGetItem", 1)
	mockSvc.AssertNumberOfCalls(t, "TransactGetProjectionCols", 2)
	mockSvc.AssertExpectations(t)
}
