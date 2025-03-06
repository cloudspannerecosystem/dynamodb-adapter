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

package services

import (
	"context"
	"reflect"
	"testing"

	"cloud.google.com/go/spanner"
	"github.com/cloudspannerecosystem/dynamodb-adapter/models"
	"github.com/cloudspannerecosystem/dynamodb-adapter/storage"
	"github.com/cloudspannerecosystem/dynamodb-adapter/utils"
	"github.com/stretchr/testify/mock"
	"gopkg.in/go-playground/assert.v1"
)

func init() {
	models.TableColumnMap = map[string][]string{
		"testTable": {"first", "second", "third", "fourth"},
	}
}
func Test_getSpannerProjections(t *testing.T) {

	tests := []struct {
		testName                 string
		projectionExpression     string
		table                    string
		expressionAttributeNames map[string]string
		want                     []string
	}{
		{
			"empty projectionExpression",
			"",
			"testTable",
			nil,
			nil,
		},
		{
			"Empty expressionAttributeNames",
			"#f, second, third",
			"testTable",
			nil,
			[]string{"second", "third"},
		},
		{
			"wrong expressionAttributeNames present",
			"#f, second, third",
			"testTable",
			map[string]string{"#f": "fir"},
			[]string{"second", "third"},
		},
		{
			"correct expressionAttributeNames present",
			"#f, second, third",
			"testTable",
			map[string]string{"#f": "first"},
			[]string{"first", "second", "third"},
		},
		{
			"only projectionExpression",
			"first, second, third",
			"testTable",
			nil,
			[]string{"first", "second", "third"},
		},
		{
			"wrong projectionExpression",
			"firs, secod, thir",
			"testTable",
			nil,
			[]string{},
		},
		{
			"wrong table",
			"first, second, third",
			"testTabl",
			nil,
			[]string{},
		},
	}

	for _, tc := range tests {
		got := getSpannerProjections(tc.projectionExpression, tc.table, tc.expressionAttributeNames)
		assert.Equal(t, got, tc.want)
	}
}

func Test_createSpannerQuery(t *testing.T) {

	tests := []struct {
		testName     string
		queryModel   *models.Query
		partionkey   string
		primaryKey   string
		secondaryKey string
		want1        spanner.Statement
		want2        []string
		want3        bool
		want4        int64
	}{
		{
			"empty queryModel",
			nil,
			"first",
			"first",
			"second",
			spanner.Statement{},
			[]string{},
			false,
			0,
		},
		{
			"queryModel is present but without projectionExpression",
			&models.Query{
				TableName: "testTable",
			},
			"first",
			"first",
			"second",
			spanner.Statement{
				SQL:    "SELECT testTable.`first`,testTable.`second`,testTable.`third`,testTable.`fourth` FROM testTable WHERE second is not null  ORDER BY second DESC  LIMIT 5000 ",
				Params: make(map[string]interface{}),
			},
			[]string{"first", "second", "third", "fourth"},
			false,
			0,
		},
		{
			"queryModel is present but with projectionExpression",
			&models.Query{
				TableName:            "testTable",
				ProjectionExpression: "first, second",
			},
			"first",
			"first",
			"second",
			spanner.Statement{
				SQL:    "SELECT testTable.`first`,testTable.`second` FROM testTable WHERE second is not null  ORDER BY second DESC  LIMIT 5000 ",
				Params: make(map[string]interface{}),
			},
			[]string{"first", "second"},
			false,
			0,
		},
		{
			"queryModel is present but with projectionExpression & ExpressionAttributeNames",
			&models.Query{
				TableName:                "testTable",
				ProjectionExpression:     "#f, second",
				ExpressionAttributeNames: map[string]string{"#f": "first"},
			},
			"first",
			"first",
			"second",
			spanner.Statement{
				SQL:    "SELECT testTable.`first`,testTable.`second` FROM testTable WHERE second is not null  ORDER BY second DESC  LIMIT 5000 ",
				Params: make(map[string]interface{}),
			},
			[]string{"first", "second"},
			false,
			0,
		},
		{
			"queryModel is present but with projectionExpression & wrong ExpressionAttributeNames",
			&models.Query{
				TableName:                "testTable",
				ProjectionExpression:     "#f, second",
				ExpressionAttributeNames: map[string]string{"#fir": "first"},
			},
			"first",
			"first",
			"second",
			spanner.Statement{
				SQL:    "SELECT testTable.`second`,testTable.`first` FROM testTable WHERE second is not null  ORDER BY second DESC  LIMIT 5000 ",
				Params: make(map[string]interface{}),
			},
			[]string{"second", "first"},
			false,
			0,
		},
		{
			"only count",
			&models.Query{
				TableName:                "testTable",
				ProjectionExpression:     "#f, second",
				ExpressionAttributeNames: map[string]string{"#f": "first"},
				OnlyCount:                true,
			},
			"first",
			"first",
			"second",
			spanner.Statement{
				SQL:    "SELECT COUNT(first) AS count FROM testTable WHERE second is not null  ",
				Params: make(map[string]interface{}),
			},
			[]string{"count"},
			true,
			0,
		},
		{
			"with offset",
			&models.Query{
				TableName:                "testTable",
				ProjectionExpression:     "#f, second",
				ExpressionAttributeNames: map[string]string{"#f": "first"},
				StartFrom: map[string]interface{}{
					"offset": float64(10),
				},
			},
			"first",
			"first",
			"second",
			spanner.Statement{
				SQL:    "SELECT testTable.`first`,testTable.`second` FROM testTable WHERE second is not null  ORDER BY second DESC  LIMIT 5000  OFFSET 10",
				Params: make(map[string]interface{}),
			},
			[]string{"first", "second"},
			false,
			10,
		},
		{
			"with offset other than float64",
			&models.Query{
				TableName:                "testTable",
				ProjectionExpression:     "#f, second",
				ExpressionAttributeNames: map[string]string{"#f": "first"},
				StartFrom: map[string]interface{}{
					"offset": 10,
				},
			},
			"first",
			"first",
			"second",
			spanner.Statement{
				SQL:    "SELECT testTable.`first`,testTable.`second` FROM testTable WHERE second is not null  ORDER BY second DESC  LIMIT 5000 ",
				Params: make(map[string]interface{}),
			},
			[]string{"first", "second"},
			false,
			0,
		},
		{
			"range expression present",
			&models.Query{
				TableName:                "testTable",
				ProjectionExpression:     "#f, second",
				ExpressionAttributeNames: map[string]string{"#f": "first"},
				RangeExp:                 "first > :val1",
				RangeValMap: map[string]interface{}{
					":val1": float64(5),
				},
			},
			"first",
			"first",
			"second",
			spanner.Statement{
				SQL: "SELECT testTable.`first`,testTable.`second` FROM testTable WHERE second is not null  AND first > @rangeExp1 ORDER BY second DESC  LIMIT 5000 ",
				Params: map[string]interface{}{
					"rangeExp1": float64(5),
				},
			},
			[]string{"first", "second"},
			false,
			0,
		},
		{
			"filter expression present",
			&models.Query{
				TableName:                "testTable",
				ProjectionExpression:     "#f, second",
				ExpressionAttributeNames: map[string]string{"#f": "first"},
				FilterExp:                "fourth > :val1",
				RangeValMap: map[string]interface{}{
					":val1": float64(5),
				},
			},
			"first",
			"first",
			"second",
			spanner.Statement{
				SQL: "SELECT testTable.`first`,testTable.`second` FROM testTable WHERE second is not null  AND fourth > @filterExp1 ORDER BY second DESC  LIMIT 5000 ",
				Params: map[string]interface{}{
					"filterExp1": float64(5),
				},
			},
			[]string{"first", "second"},
			false,
			0,
		},
		{
			"filter & range expression both present",
			&models.Query{
				TableName:                "testTable",
				ProjectionExpression:     "#f, second",
				ExpressionAttributeNames: map[string]string{"#f": "first"},
				FilterExp:                "fourth > :val1",
				RangeExp:                 "first > :val2",
				RangeValMap: map[string]interface{}{
					":val1": float64(5),
					":val2": float64(4),
				},
			},
			"first",
			"first",
			"second",
			spanner.Statement{
				SQL: "SELECT testTable.`first`,testTable.`second` FROM testTable WHERE second is not null  AND first > @rangeExp1 AND fourth > @filterExp1 ORDER BY second DESC  LIMIT 5000 ",
				Params: map[string]interface{}{
					"filterExp1": float64(5),
					"rangeExp1":  float64(4),
				},
			},
			[]string{"first", "second"},
			false,
			0,
		},
		{
			"limit present",
			&models.Query{
				TableName:                "testTable",
				ProjectionExpression:     "#f, second",
				ExpressionAttributeNames: map[string]string{"#f": "first"},
				FilterExp:                "fourth > :val1",
				RangeExp:                 "first > :val2",
				RangeValMap: map[string]interface{}{
					":val1": float64(5),
					":val2": float64(4),
				},
				Limit: 100,
			},
			"first",
			"first",
			"second",
			spanner.Statement{
				SQL: "SELECT testTable.`first`,testTable.`second` FROM testTable WHERE second is not null  AND first > @rangeExp1 AND fourth > @filterExp1 ORDER BY second DESC  LIMIT 100",
				Params: map[string]interface{}{
					"filterExp1": float64(5),
					"rangeExp1":  float64(4),
				},
			},
			[]string{"first", "second"},
			false,
			0,
		},
	}

	for _, tc := range tests {
		got1, got2, got3, got4, _, _ := createSpannerQuery(tc.queryModel, tc.partionkey, tc.primaryKey, tc.secondaryKey)

		assert.Equal(t, got1, tc.want1)
		assert.Equal(t, got2, tc.want2)
		assert.Equal(t, got3, tc.want3)
		assert.Equal(t, got4, tc.want4)
	}
}

func Test_parseSpannerColumns(t *testing.T) {
	tests := []struct {
		testName     string
		queryModel   *models.Query
		partitionkey string
		primaryKey   string
		secondaryKey string
		want1        []string
		want2        string
		want3        bool
	}{
		{
			"empty queryModel",
			nil,
			"",
			"",
			"",
			[]string{},
			"",
			false,
		},
		{
			"onlyCount present",
			&models.Query{
				OnlyCount: true,
			},
			"first",
			"first",
			"second",
			[]string{"count"},
			"COUNT(first) AS count",
			true,
		},
		{
			"Empty Query Model",
			&models.Query{},
			"first",
			"first",
			"second",
			nil,
			"",
			false,
		},
		{
			"Only table Name present",
			&models.Query{
				TableName: "testTable",
			},
			"first",
			"first",
			"second",
			[]string{"first", "second", "third", "fourth"},
			"testTable.`first`,testTable.`second`,testTable.`third`,testTable.`fourth`",
			false,
		},
		{
			"table with projection expression",
			&models.Query{
				TableName:            "testTable",
				ProjectionExpression: "first, third, fourth",
			},
			"first",
			"first",
			"second",
			[]string{"first", "third", "fourth", "second"},
			"testTable.`first`,testTable.`third`,testTable.`fourth`,testTable.`second`",
			false,
		},
		{
			"table with wrong projection expression",
			&models.Query{
				TableName:            "testTable",
				ProjectionExpression: "first, second , third, four",
			},
			"first",
			"first",
			"second",
			[]string{"first", "second", "third"},
			"testTable.`first`,testTable.`second`,testTable.`third`",
			false,
		},
		{
			"projectionexpression & ExpressionAttributeNames both present",
			&models.Query{
				TableName:            "testTable",
				ProjectionExpression: "first, #s, third",
				ExpressionAttributeNames: map[string]string{
					"#s": "second",
				},
			},
			"first",
			"first",
			"second",
			[]string{"first", "second", "third"},
			"testTable.`first`,testTable.`second`,testTable.`third`",
			false,
		},
		{
			"projectionexpression & ExpressionAttributeNames both present",
			&models.Query{
				TableName:            "testTable",
				ProjectionExpression: "first, #s, third, #fr",
				ExpressionAttributeNames: map[string]string{
					"#s":  "second",
					"#fr": "fourth",
				},
			},
			"first",
			"first",
			"second",
			[]string{"first", "second", "third", "fourth"},
			"testTable.`first`,testTable.`second`,testTable.`third`,testTable.`fourth`",
			false,
		},
	}

	for _, tc := range tests {
		got1, got2, got3, _ := parseSpannerColumns(tc.queryModel, tc.partitionkey, tc.primaryKey, tc.secondaryKey)

		assert.Equal(t, got1, tc.want1)
		assert.Equal(t, got2, tc.want2)
		assert.Equal(t, got3, tc.want3)
	}
}

func Test_parseSpannerTableName(t *testing.T) {
	tests := []struct {
		testName   string
		queryModel *models.Query
		want       string
	}{
		{
			"Empty Query model",
			&models.Query{},
			"",
		},
		{
			"TableName present",
			&models.Query{
				TableName: "testTable",
			},
			"testTable",
		},
		{
			"IndexName passed",
			&models.Query{
				TableName: "testTable",
				IndexName: "SecondaryIndex",
			},
			"testTable@{FORCE_INDEX=SecondaryIndex}",
		},
	}

	for _, tc := range tests {
		got := parseSpannerTableName(tc.queryModel)
		assert.Equal(t, got, tc.want)
	}
}

func Test_parseSpannerCondition(t *testing.T) {
	tests := []struct {
		testName   string
		queryModel *models.Query
		pKey       string
		sKey       string
		want1      string
		want2      map[string]interface{}
	}{
		{
			"Empty Query model",
			&models.Query{},
			"",
			"",
			" ",
			make(map[string]interface{}),
		},
		{
			"queryModel with only tableName",
			&models.Query{
				TableName: "testTable",
			},
			"first",
			"second",
			"WHERE second is not null ",
			make(map[string]interface{}),
		},
		{
			"rangeExpression present",
			&models.Query{
				TableName: "testTable",
				RangeExp:  "first = :val1",
				RangeValMap: map[string]interface{}{
					":val1": float64(61),
				},
			},
			"first",
			"second",
			"WHERE second is not null  AND first = @rangeExp1",
			map[string]interface{}{
				"rangeExp1": float64(61),
			},
		},
		{
			"FilterExpression present",
			&models.Query{
				TableName: "testTable",
				FilterExp: "fourth = :val1",
				RangeValMap: map[string]interface{}{
					":val1": float64(61),
				},
			},
			"first",
			"second",
			"WHERE second is not null  AND fourth = @filterExp1",
			map[string]interface{}{
				"filterExp1": float64(61),
			},
		},
		{
			"FilterExpression & range both present",
			&models.Query{
				TableName: "testTable",
				RangeExp:  "fourth = :val1",
				FilterExp: "fourth = :val2",
				RangeValMap: map[string]interface{}{
					":val1": float64(61),
					":val2": float64(34),
				},
			},
			"first",
			"second",
			"WHERE second is not null  AND fourth = @rangeExp1 AND fourth = @filterExp1",
			map[string]interface{}{
				"filterExp1": float64(34),
				"rangeExp1":  float64(61),
			},
		},
	}

	for _, tc := range tests {
		got1, got2 := parseSpannerCondition(tc.queryModel, tc.pKey, tc.sKey)
		assert.Equal(t, got1, tc.want1)
		assert.Equal(t, got2, tc.want2)
	}
}

func Test_parseOffset(t *testing.T) {
	tests := []struct {
		testName   string
		queryModel *models.Query
		want1      string
		want2      int64
	}{
		{
			"Empty Query Model",
			&models.Query{},
			"",
			0,
		},
		{
			"StartFrom object present",
			&models.Query{
				StartFrom: map[string]interface{}{
					"offset": float64(10),
				},
			},
			" OFFSET 10",
			10,
		},
		{
			"StartFrom without float64 value",
			&models.Query{
				StartFrom: map[string]interface{}{
					"offset": 10,
				},
			},
			"",
			0,
		},
	}

	for _, tc := range tests {
		got1, got2 := parseOffset(tc.queryModel)
		assert.Equal(t, got1, tc.want1)
		assert.Equal(t, got2, tc.want2)
	}
}

func Test_parseSpannerSorting(t *testing.T) {
	tests := []struct {
		testName     string
		query        *models.Query
		isCountQuery bool
		pKey         string
		sKey         string
		want         string
	}{
		{
			"empty Query & skey",
			&models.Query{},
			false,
			"",
			"",
			" ",
		},
		{
			"empty Query but skey present",
			&models.Query{},
			false,
			"first",
			"second",
			" ORDER BY second DESC ",
		},
		{
			"empty Query but skey present",
			&models.Query{
				SortAscending: true,
			},
			false,
			"first",
			"second",
			" ORDER BY second ASC ",
		},
		{
			"isCountQuery is true",
			&models.Query{
				SortAscending: true,
			},
			true,
			"first",
			"second",
			" ",
		},
	}

	for _, tc := range tests {
		got := parseSpannerSorting(tc.query, tc.isCountQuery, tc.pKey, tc.sKey)
		assert.Equal(t, got, tc.want)
	}

}

func Test_parseLimit(t *testing.T) {
	tests := []struct {
		testName     string
		queryModel   *models.Query
		isCountQuery bool
		want         string
	}{
		{
			"Empty Query Model",
			&models.Query{},
			false,
			" LIMIT 5000 ",
		},
		{
			"isCountQuery is true",
			&models.Query{},
			true,
			"",
		},
		{
			"custom Limit testcase",
			&models.Query{
				Limit: 100,
			},
			false,
			" LIMIT 100",
		},
		{
			"custom Limit with isCountQuery is true testcase",
			&models.Query{
				Limit: 100,
			},
			true,
			"",
		},
	}

	for _, tc := range tests {
		got := parseLimit(tc.queryModel, tc.isCountQuery)
		assert.Equal(t, got, tc.want)
	}

}

type MockStorage struct {
	mock.Mock
}

func (m *MockStorage) SpannerTransactWritePut(ctx context.Context, tableName string, putObj map[string]interface{}, e *models.Eval, expr *models.UpdateExpressionCondition, txn *spanner.ReadWriteTransaction) (map[string]interface{}, *spanner.Mutation, error) {
	args := m.Called(ctx, tableName, putObj, e, expr, txn)
	return args.Get(0).(map[string]interface{}), args.Get(1).(*spanner.Mutation), args.Error(2)
}

func (m *MockStorage) SpannerGet(ctx context.Context, tableName string, pKeys, sKeys interface{}, projectionCols []string) (map[string]interface{}, error) {
	args := m.Called(ctx, tableName, pKeys, sKeys, projectionCols)
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockStorage) TransactWriteSpannerDel(ctx context.Context, table string, n map[string]interface{}, eval *models.Eval, expr *models.UpdateExpressionCondition, txn *spanner.ReadWriteTransaction) (*spanner.Mutation, error) {
	args := m.Called(ctx, table, n, eval, expr, txn)
	return args.Get(0).(*spanner.Mutation), args.Error(1)
}

func (m *MockStorage) TransactWriteSpannerAdd(ctx context.Context, table string, n map[string]interface{}, eval *models.Eval, expr *models.UpdateExpressionCondition, txn *spanner.ReadWriteTransaction) (map[string]interface{}, *spanner.Mutation, error) {
	args := m.Called(ctx, table, n, eval, expr, txn)
	return args.Get(0).(map[string]interface{}), args.Get(1).(*spanner.Mutation), args.Error(2)
}

func (m *MockStorage) TransactWriteSpannerRemove(ctx context.Context, table string, n map[string]interface{}, eval *models.Eval, expr *models.UpdateExpressionCondition, colsToRemove []string, txn *spanner.ReadWriteTransaction) (*spanner.Mutation, error) {
	args := m.Called(ctx, table, n, eval, expr, txn)
	return args.Get(0).(*spanner.Mutation), args.Error(1)
}

type MockConfig struct{}

func (m *MockConfig) GetTableConf(tableName string) (models.TableConfig, error) {
	return models.TableConfig{ActualTable: tableName}, nil
}

func mockCreateConditionExpression(conditionExp string, expressionAttr map[string]interface{}) (*models.Eval, error) {
	return &models.Eval{}, nil
}

type MockServices struct {
	mock.Mock
}

func (m *MockStorage) GetSpannerClient() (*spanner.Client, error) {
	args := m.Called()
	client, _ := args.Get(0).(*spanner.Client)
	return client, args.Error(1)
}

func TestTransactWritePut(t *testing.T) {
	ctx := context.Background()
	mockTxn := &spanner.ReadWriteTransaction{}
	utils.CreateConditionExpressionFunc = mockCreateConditionExpression

	// Ensure the mock is reset after the test
	defer func() { utils.CreateConditionExpressionFunc = utils.CreateConditionExpression }()

	tableName := "TestTable"
	putObj := map[string]interface{}{"Name": "John"}
	oldRes := map[string]interface{}{"Age": 30}
	expr := &models.UpdateExpressionCondition{}
	conditionExp := "#age > :minAge"
	expressionAttr := map[string]interface{}{":minAge": 18}
	mockStorage := new(MockStorage)

	models.DbConfigMap = map[string]models.TableConfig{
		"TestTable": {
			ActualTable:  "TestTable",
			PartitionKey: "id",
		},
	}
	models.TableDDL = make(map[string]map[string]string)
	models.TableDDL["TestTable"] = map[string]string{
		"id":   "INT64",
		"Name": "STRING",
		"Age":  "INT64",
	}

	mockStorageInstance := &storage.Storage{}
	storage.SetStorageInstance(mockStorageInstance)

	mockStorage.On("SpannerTransactWritePut", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mockTxn).
		Return(map[string]interface{}{"Name": "John", "Age": 30}, &spanner.Mutation{}, nil)

	svc := &spannerService{
		st: mockStorage, // Assign the mock storage to the struct field
	}
	result, _, _ := svc.TransactWritePut(ctx, tableName, putObj, expr, conditionExp, expressionAttr, oldRes, mockTxn)

	expected := map[string]interface{}{
		"Name": "John",
		"Age":  30,
	}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected result %+v, got %+v", expected, result)
	}

	mockStorage.AssertExpectations(t)
}

func TestTransactWriteDel(t *testing.T) {
	ctx := context.Background()
	mockTxn := &spanner.ReadWriteTransaction{}
	utils.CreateConditionExpressionFunc = mockCreateConditionExpression

	defer func() { utils.CreateConditionExpressionFunc = utils.CreateConditionExpression }()

	tableName := "TestTable"
	attrMap := map[string]interface{}{"id": 1}
	conditionExp := "#age > :minAge"
	expressionAttr := map[string]interface{}{":minAge": 18}
	expr := &models.UpdateExpressionCondition{}
	mockStorage := new(MockStorage)

	models.DbConfigMap = map[string]models.TableConfig{
		"TestTable": {
			ActualTable:  "TestTable",
			PartitionKey: "id",
		},
	}

	mockStorage.On("TransactWriteSpannerDel", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mockTxn).
		Return(&spanner.Mutation{}, nil)
	mockStorage.On("SpannerGet", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(map[string]interface{}{"Age": 30, "Name": "John"}, nil)

	svc := &spannerService{st: mockStorage}
	result, _, _ := svc.TransactWriteDel(ctx, tableName, attrMap, conditionExp, expressionAttr, expr, mockTxn)

	expected := map[string]interface{}{"Age": 30, "Name": "John"}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected result %+v, got %+v", expected, result)
	}

	mockStorage.AssertExpectations(t)
}

func TestTransactWriteAdd(t *testing.T) {
	ctx := context.Background()
	mockTxn := &spanner.ReadWriteTransaction{}
	utils.CreateConditionExpressionFunc = mockCreateConditionExpression
	defer func() { utils.CreateConditionExpressionFunc = utils.CreateConditionExpression }()

	tableName := "TestTable"
	attrMap := map[string]interface{}{"Name": "John"}
	oldRes := map[string]interface{}{"Age": 30}
	conditionExp := "#age > :minAge"
	expressionAttr := map[string]interface{}{":minAge": 18}
	expr := &models.UpdateExpressionCondition{}
	mockStorage := new(MockStorage)

	mockStorage.On("TransactWriteSpannerAdd", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mockTxn).
		Return(map[string]interface{}{"Name": "John"}, &spanner.Mutation{}, nil)

	svc := &spannerService{st: mockStorage}
	result, _, _ := svc.TransactWriteAdd(ctx, tableName, attrMap, conditionExp, attrMap, expressionAttr, expr, oldRes, mockTxn)

	expected := map[string]interface{}{"Name": "John", "Age": 30}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected result %+v, got %+v", expected, result)
	}

	mockStorage.AssertExpectations(t)
}

func TestTransactWriteRemove(t *testing.T) {
	ctx := context.Background()
	mockTxn := &spanner.ReadWriteTransaction{}
	utils.CreateConditionExpressionFunc = mockCreateConditionExpression
	defer func() { utils.CreateConditionExpressionFunc = utils.CreateConditionExpression }()

	tableName := "TestTable"
	oldRes := map[string]interface{}{"Age": 30, "Name": "John"}
	expr := &models.UpdateExpressionCondition{}
	actionValue := "Name"
	updateAttr := models.UpdateAttr{PrimaryKeyMap: map[string]interface{}{}}
	mockStorage := new(MockStorage)

	mockStorage.On("TransactWriteSpannerRemove", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mockTxn).
		Return(&spanner.Mutation{}, nil)

	svc := &spannerService{st: mockStorage}
	result, _, _ := svc.TransactWriteRemove(ctx, tableName, updateAttr, actionValue, expr, oldRes, mockTxn)

	expected := map[string]interface{}{"Age": 30}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected result %+v, got %+v", expected, result)
	}

	mockStorage.AssertExpectations(t)
}
