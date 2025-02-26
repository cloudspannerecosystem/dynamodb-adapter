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

// Package models implements all the structs required by application
package models

import (
	"sync"

	"github.com/antonmedv/expr/vm"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

type SpannerConfig struct {
	ProjectID        string `yaml:"project_id"`
	InstanceID       string `yaml:"instance_id"`
	DatabaseName     string `yaml:"database_name"`
	QueryLimit       int64  `yaml:"query_limit"`
	DynamoQueryLimit int32  `yaml:"dynamo_query_limit"` //dynamo_query_limit
}

type Config struct {
	Spanner SpannerConfig `yaml:"spanner"`
}

var GlobalConfig *Config

// Meta struct
type Meta struct {
	TableName                 string                              `json:"TableName"`
	AttrMap                   map[string]interface{}              `json:"AttrMap"`
	ReturnValues              string                              `json:"ReturnValues"`
	ConditionExpression       string                              `json:"ConditionExpression"`
	ExpressionAttributeMap    map[string]interface{}              `json:"ExpressionAttributeMap"`
	ExpressionAttributeNames  map[string]string                   `json:"ExpressionAttributeNames"`
	ExpressionAttributeValues map[string]*dynamodb.AttributeValue `json:"ExpressionAttributeValues"`
	Item                      map[string]*dynamodb.AttributeValue `json:"Item"`
}

// GetKeyMeta struct
type GetKeyMeta struct {
	Key          string                              `json:"Key"`
	Type         string                              `json:"Type"`
	DynamoObject map[string]*dynamodb.AttributeValue `json:"DynamoObject"`
}

// SetKeyMeta struct
type SetKeyMeta struct {
	Key          string                              `json:"Key"`
	Type         string                              `json:"Type"`
	Value        string                              `json:"Value"`
	DynamoObject map[string]*dynamodb.AttributeValue `json:"DynamoObject"`
}

// BatchMetaUpdate struct
type BatchMetaUpdate struct {
	TableName    string                                `json:"TableName"`
	ArrAttrMap   []map[string]interface{}              `json:"ArrAttrMap"`
	DynamoObject []map[string]*dynamodb.AttributeValue `json:"DynamoObject"`
}

// BatchMeta struct
type BatchMeta struct {
	TableName    string                                `json:"TableName"`
	KeyArray     []map[string]interface{}              `json:"KeyArray"`
	DynamoObject []map[string]*dynamodb.AttributeValue `json:"DynamoObject"`
}

// GetItemMeta struct
type GetItemMeta struct {
	TableName                string                              `json:"TableName"`
	PrimaryKeyMap            map[string]interface{}              `json:"PrimaryKeyMap"`
	ProjectionExpression     string                              `json:"ProjectionExpression"`
	ExpressionAttributeNames map[string]string                   `json:"ExpressionAttributeNames"`
	Key                      map[string]*dynamodb.AttributeValue `json:"Key"`
}

// BatchGetMeta struct
type BatchGetMeta struct {
	RequestItems map[string]BatchGetWithProjectionMeta `json:"RequestItems"`
}

// BatchGetWithProjectionMeta struct
type BatchGetWithProjectionMeta struct {
	TableName                string                                `json:"TableName"`
	KeyArray                 []map[string]interface{}              `json:"KeyArray"`
	ProjectionExpression     string                                `json:"ProjectionExpression"`
	ExpressionAttributeNames map[string]string                     `json:"ExpressionAttributeNames"`
	Keys                     []map[string]*dynamodb.AttributeValue `json:"Keys"`
}

// Delete struct
type Delete struct {
	TableName                 string                              `json:"TableName"`
	PrimaryKeyMap             map[string]interface{}              `json:"PrimaryKeyMap"`
	ConditionExpression       string                              `json:"ConditionExpression"`
	ExpressionAttributeMap    map[string]interface{}              `json:"ExpressionAttributeMap"`
	Key                       map[string]*dynamodb.AttributeValue `json:"Key"`
	ExpressionAttributeValues map[string]*dynamodb.AttributeValue `json:"ExpressionAttributeValues"`
	ExpressionAttributeNames  map[string]string                   `json:"ExpressionAttributeNames"`
}

// BulkDelete struct
type BulkDelete struct {
	TableName          string                                `json:"TableName"`
	PrimaryKeyMapArray []map[string]interface{}              `json:"KeyArray"`
	DynamoObject       []map[string]*dynamodb.AttributeValue `json:"DynamoObject"`
}

// Query struct
type Query struct {
	TableName                 string                              `json:"TableName"`
	IndexName                 string                              `json:"IndexName"`
	OnlyCount                 bool                                `json:"OnlyCount"`
	Limit                     int64                               `json:"Limit"`
	SortAscending             bool                                `json:"ScanIndexForward"`
	StartFrom                 map[string]interface{}              `json:"StartFrom"`
	ProjectionExpression      string                              `json:"ProjectionExpression"`
	ExpressionAttributeNames  map[string]string                   `json:"ExpressionAttributeNames"`
	FilterExp                 string                              `json:"FilterExpression"`
	RangeExp                  string                              `json:"KeyConditionExpression"`
	RangeValMap               map[string]interface{}              `json:"RangeValMap"`
	ExpressionAttributeValues map[string]*dynamodb.AttributeValue `json:"ExpressionAttributeValues"`
	ExclusiveStartKey         map[string]*dynamodb.AttributeValue `json:"ExclusiveStartKey"`
	Select                    string                              `json:"Select"`
}

// UpdateAttr struct
type UpdateAttr struct {
	TableName                 string                              `json:"TableName"`
	PrimaryKeyMap             map[string]interface{}              `json:"PrimaryKeyMap"`
	ReturnValues              string                              `json:"ReturnValues"`
	UpdateExpression          string                              `json:"UpdateExpression"`
	ConditionExpression       string                              `json:"ConditionExpression"`
	ExpressionAttributeMap    map[string]interface{}              `json:"AttrVals"`
	ExpressionAttributeNames  map[string]string                   `json:"ExpressionAttributeNames"`
	Key                       map[string]*dynamodb.AttributeValue `json:"Key"`
	ExpressionAttributeValues map[string]*dynamodb.AttributeValue `json:"ExpressionAttributeValues"`
}

// ScanMeta for Scan request
type ScanMeta struct {
	TableName                 string                              `json:"TableName"`
	IndexName                 string                              `json:"IndexName"`
	OnlyCount                 bool                                `json:"OnlyCount"`
	Select                    string                              `json:"Select"`
	Limit                     int64                               `json:"Limit"`
	StartFrom                 map[string]interface{}              `json:"StartFrom"`
	ExclusiveStartKey         map[string]*dynamodb.AttributeValue `json:"ExclusiveStartKey"`
	FilterExpression          string                              `json:"FilterExpression"`
	ProjectionExpression      string                              `json:"ProjectionExpression"`
	ExpressionAttributeNames  map[string]string                   `json:"ExpressionAttributeNames"`
	ExpressionAttributeMap    map[string]interface{}              `json:"ExpressionAttributeMap"`
	ExpressionAttributeValues map[string]*dynamodb.AttributeValue `json:"ExpressionAttributeValues"`
}

// TableConfig for Configuration table
type TableConfig struct {
	PartitionKey     string                 `json:"PartitionKey,omitempty"`
	SortKey          string                 `json:"SortKey,omitempty"`
	Indices          map[string]TableConfig `json:"Indices,omitempty"`
	GCSSourcePath    string                 `json:"GcsSourcePath,omitempty"`
	DDBIndexName     string                 `json:"DdbIndexName,omitempty"`
	SpannerIndexName string                 `json:"SpannerIndexName,omitempty"`
	IsPadded         bool                   `json:"IsPadded,omitempty"`
	IsComplement     bool                   `json:"IsComplement,omitempty"`
	TableSource      string                 `json:"TableSource,omitempty"`
	ActualTable      string                 `json:"ActualTable,omitempty"`
}

// BatchWriteItem for Batch Operation
type BatchWriteItem struct {
	RequestItems map[string][]BatchWriteSubItems `json:"RequestItems"`
}

// BatchWriteItemResponse for Batch Operation
type BatchWriteItemResponse struct {
	UnprocessedItems map[string][]BatchWriteSubItems `json:"UnprocessedItems"`
}

// BatchWriteSubItems is for BatchWriteItem
type BatchWriteSubItems struct {
	DelReq BatchDeleteItem `json:"DeleteRequest"`
	PutReq BatchPutItem    `json:"PutRequest"`
}

// BatchDeleteItem is for BatchWriteSubItems
type BatchDeleteItem struct {
	Key map[string]*dynamodb.AttributeValue `json:"Key"`
}

// BatchPutItem is for BatchWriteSubItems
type BatchPutItem struct {
	Item map[string]*dynamodb.AttributeValue `json:"Item"`
}

var DbConfigMap map[string]TableConfig

// TableDDL - this contains the DDL
var TableDDL map[string]map[string]string

// TableColumnMap - this contains the list of columns for the tables
var TableColumnMap map[string][]string

// TableColChangeMap for changed columns map
var TableColChangeMap map[string]struct{}

// ColumnToOriginalCol for Original column map
var ColumnToOriginalCol map[string]string

// OriginalColResponse for Original Column Response
var OriginalColResponse map[string]string

func init() {
	TableDDL = make(map[string]map[string]string)
	TableDDL["dynamodb_adapter_table_ddl"] = map[string]string{"tableName": "S", "column": "S", "dynamoDataType": "S", "originalColumn": "S", "partitionKey": "S", "sortKey": "S", "spannerIndexName": "S", "actualTable": "S", "spannerDataType": "S"}
	TableDDL["dynamodb_adapter_config_manager"] = map[string]string{"tableName": "STRING(MAX)", "config": "STRING(MAX)", "cronTime": "STRING(MAX)", "uniqueValue": "STRING(MAX)", "enabledStream": "STRING(MAX)", "pubsubTopic": "STRING(MAX)"}
	TableColumnMap = make(map[string][]string)
	TableColumnMap["dynamodb_adapter_table_ddl"] = []string{"tableName", "column", "dynamoDataType", "originalColumn", "partitionKey", "sortKey", "spannerIndexName", "actualTable", "spannerDataType"}
	TableColumnMap["dynamodb_adapter_config_manager"] = []string{"tableName", "config", "cronTime", "uniqueValue", "enabledStream", "pubsubTopic"}
	TableColChangeMap = make(map[string]struct{})
	ColumnToOriginalCol = make(map[string]string)
	OriginalColResponse = make(map[string]string)
}

// Eval for Evaluation expression
type Eval struct {
	Cond       *vm.Program
	Attributes []string
	Cols       []string
	Tokens     []string
	ValueMap   map[string]interface{}
}

// UpdateExpressionCondition for Update Condition
type UpdateExpressionCondition struct {
	Field     []string
	Value     []string
	Condition []string
	ActionVal string
	AddValues map[string]float64
}

// ConfigControllerModel for Config controller
type ConfigControllerModel struct {
	Mux               sync.RWMutex
	UniqueVal         string
	CornTime          string
	StopConfigManager bool
	ReadMap           map[string]struct{}
	WriteMap          map[string]struct{}
	StreamEnable      map[string]struct{}
	PubSubTopic       map[string]string
}

// ConfigController object for ConfigControllerModel
var ConfigController *ConfigControllerModel

// SpannerTableMap for spanner column map
var SpannerTableMap = make(map[string]string)

func init() {
	ConfigController = new(ConfigControllerModel)
	ConfigController.CornTime = "1"
	ConfigController.Mux = sync.RWMutex{}
	ConfigController.ReadMap = make(map[string]struct{})
	ConfigController.WriteMap = make(map[string]struct{})
	ConfigController.StreamEnable = make(map[string]struct{})
	ConfigController.PubSubTopic = make(map[string]string)
}

// StreamDataModel for streaming data
type StreamDataModel struct {
	OldImage       map[string]interface{} `json:"OldImage"`
	NewImage       map[string]interface{} `json:"NewImage"`
	Keys           map[string]interface{} `json:"Keys"`
	Timestamp      int64                  `json:"Timestamp"`
	Table          string                 `json:"TableName"`
	EventName      string                 `json:"EventName"`
	SequenceNumber int64                  `json:"SequenceNumber"`
	EventID        string                 `json:"EventId"`
	EventSourceArn string                 `json:"EventSourceArn"`
}

// TransactGetItemsRequest represents the input structure for TransactGetItems API.
type TransactGetItemsRequest struct {
	TransactItems          []TransactGetItem `json:"TransactItems"`
	ReturnConsumedCapacity string            `json:"ReturnConsumedCapacity,omitempty"`
}

// TransactGetItem represents a single Get operation inside TransactGetItems.
type TransactGetItem struct {
	Get GetItemRequest `json:"Get"`
}

// GetItemRequest represents the structure of a Get request.
type GetItemRequest struct {
	TableName                string                              `json:"TableName"`
	Keys                     map[string]*dynamodb.AttributeValue `json:"Key"`
	KeyArray                 []map[string]interface{}            `json:"KeyArray"`
	ProjectionExpression     string                              `json:"ProjectionExpression,omitempty"`
	ExpressionAttributeNames map[string]string                   `json:"ExpressionAttributeNames,omitempty"`
}

// ConsumedCapacity represents the consumed capacity of a DynamoDB operation.
type ConsumedCapacity struct {
	TableName     string  `json:"TableName"`
	CapacityUnits float64 `json:"CapacityUnits"`
}

type TransactGetItemResponse struct {
	TableName string                 `json:"TableName"`
	Item      map[string]interface{} `json:"Item"`
}

// TransactGetItemsResponse represents the overall response structure for multiple TransactGetItems.
type TransactGetItemsResponse struct {
	Responses []TransactGetItemResponse `json:"Responses"`
}
