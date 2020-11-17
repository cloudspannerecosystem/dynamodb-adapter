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
	"github.com/aws/aws-sdk-go/service/dynamodb"
	// "github.com/zhouzhuojie/conditions"

	"github.com/antonmedv/expr/vm"

	"sync"
)

// Meta struct
type Meta struct {
	TableName                 string                              `json:"TableName"`
	AttrMap                   map[string]interface{}              `json:"attrMap"`
	ReturnValues              string                              `json:"ReturnValues"`
	ConditionExpression       string                              `json:"ConditionExpression"`
	ExpressionAttributeMap    map[string]interface{}              `json:"ExpressionAttributeMap"`
	ExpressionAttributeNames  map[string]string                   `json:"ExpressionAttributeNames"`
	ExpressionAttributeValues map[string]*dynamodb.AttributeValue `json:"ExpressionAttributeValues"`
	Item                      map[string]*dynamodb.AttributeValue `json:"Item"`
}

// GetKeyMeta struct
type GetKeyMeta struct {
	Key          string                              `json:"key"`
	Type         string                              `json:"type"`
	DynamoObject map[string]*dynamodb.AttributeValue `json:"dynamoObject"`
}

// SetKeyMeta struct
type SetKeyMeta struct {
	Key          string                              `json:"key"`
	Type         string                              `json:"type"`
	Value        string                              `json:"value"`
	DynamoObject map[string]*dynamodb.AttributeValue `json:"dynamoObject"`
}

// BatchMetaUpdate struct
type BatchMetaUpdate struct {
	TableName    string                                `json:"tableName"`
	ArrAttrMap   []map[string]interface{}              `json:"arrAttrMap"`
	DynamoObject []map[string]*dynamodb.AttributeValue `json:"dynamoObject"`
}

// BatchMeta struct
type BatchMeta struct {
	TableName    string                                `json:"tableName"`
	KeyArray     []map[string]interface{}              `json:"keyArray"`
	DynamoObject []map[string]*dynamodb.AttributeValue `json:"dynamoObject"`
}

// GetItemMeta struct
type GetItemMeta struct {
	TableName                string                              `json:"TableName"`
	PrimaryKeyMap            map[string]interface{}              `json:"PrimaryKeyMap"`
	ProjectionExpression     string                              `json:"ProjectionExpression"`
	ExpressionAttributeNames map[string]string                   `json:"ExpressionAttributeNames"`
	Key                      map[string]*dynamodb.AttributeValue `json:"Key"`
}

//BatchGetMeta struct
type BatchGetMeta struct {
	RequestItems map[string]BatchGetWithProjectionMeta `json:"RequestItems"`
}

// BatchGetWithProjectionMeta struct
type BatchGetWithProjectionMeta struct {
	TableName                string                                `json:"tableName"`
	KeyArray                 []map[string]interface{}              `json:"keyArray"`
	ProjectionExpression     string                                `json:"projectionExpression"`
	ExpressionAttributeNames map[string]string                     `json:"expressionAttributeNames"`
	Keys                     []map[string]*dynamodb.AttributeValue `json:"Keys"`
}

// Delete struct
type Delete struct {
	TableName                 string                              `json:"tableName"`
	PrimaryKeyMap             map[string]interface{}              `json:"primaryKeyMap"`
	ConditionExpression       string                              `json:"ConditionExpression"`
	ExpressionAttributeMap    map[string]interface{}              `json:"ExpressionAttributeMap"`
	Key                       map[string]*dynamodb.AttributeValue `json:"Key"`
	ExpressionAttributeValues map[string]*dynamodb.AttributeValue `json:"ExpressionAttributeValues"`
	ExpressionAttributeNames  map[string]string                   `json:"ExpressionAttributeNames"`
}

// BulkDelete struct
type BulkDelete struct {
	TableName          string                                `json:"tableName"`
	PrimaryKeyMapArray []map[string]interface{}              `json:"keyArray"`
	DynamoObject       []map[string]*dynamodb.AttributeValue `json:"dynamoObject"`
}

// Query struct
type Query struct {
	TableName                 string                              `json:"tableName"`
	IndexName                 string                              `json:"indexName"`
	OnlyCount                 bool                                `json:"onlyCount"`
	Limit                     int64                               `json:"limit"`
	SortAscending             bool                                `json:"ScanIndexForward"`
	StartFrom                 map[string]interface{}              `json:"startFrom"`
	ProjectionExpression      string                              `json:"ProjectionExpression"`
	ExpressionAttributeNames  map[string]string                   `json:"ExpressionAttributeNames"`
	FilterExp                 string                              `json:"FilterExpression"`
	RangeExp                  string                              `json:"KeyConditionExpression"`
	RangeValMap               map[string]interface{}              `json:"rangeValMap"`
	ExpressionAttributeValues map[string]*dynamodb.AttributeValue `json:"ExpressionAttributeValues"`
	ExclusiveStartKey         map[string]*dynamodb.AttributeValue `json:"ExclusiveStartKey"`
	Select                    string                              `json:"Select"`
}

// UpdateAttr struct
type UpdateAttr struct {
	TableName                 string                              `json:"tableName"`
	PrimaryKeyMap             map[string]interface{}              `json:"primaryKeyMap"`
	ReturnValues              string                              `json:"returnValues"`
	UpdateExpression          string                              `json:"updateExpression"`
	ConditionExpression       string                              `json:"ConditionExpression"`
	ExpressionAttributeMap    map[string]interface{}              `json:"attrVals"`
	ExpressionAttributeNames  map[string]string                   `json:"ExpressionAttributeNames"`
	Key                       map[string]*dynamodb.AttributeValue `json:"Key"`
	ExpressionAttributeValues map[string]*dynamodb.AttributeValue `json:"ExpressionAttributeValues"`
}

//ScanMeta for Scan request
type ScanMeta struct {
	TableName                 string                              `json:"tableName"`
	IndexName                 string                              `json:"indexName"`
	OnlyCount                 bool                                `json:"onlyCount"`
	Select                    string                              `json:"Select"`
	Limit                     int64                               `json:"limit"`
	StartFrom                 map[string]interface{}              `json:"startFrom"`
	ExclusiveStartKey         map[string]*dynamodb.AttributeValue `json:"ExclusiveStartKey"`
	FilterExpression          string                              `json:"FilterExpression"`
	ProjectionExpression      string                              `json:"ProjectionExpression"`
	ExpressionAttributeNames  map[string]string                   `json:"ExpressionAttributeNames"`
	ExpressionAttributeMap    map[string]interface{}              `json:"ExpressionAttributeMap"`
	ExpressionAttributeValues map[string]*dynamodb.AttributeValue `json:"ExpressionAttributeValues"`
}

// TableConfig for Configuration table
type TableConfig struct {
	PartitionKey     string                 `json:"partitionKey,omitempty"`
	SortKey          string                 `json:"sortKey,omitempty"`
	Indices          map[string]TableConfig `json:"indices,omitempty"`
	GCSSourcePath    string                 `json:"gcsSourcePath,omitempty"`
	DDBIndexName     string                 `json:"ddbIndexName,omitempty"`
	SpannerIndexName string                 `json:"table,omitempty"`
	IsPadded         bool                   `json:"isPadded,omitempty"`
	IsComplement     bool                   `json:"isComplement,omitempty"`
	TableSource      string                 `json:"tableSource,omitempty"`
	ActualTable      string                 `json:"actualTable,omitempty"`
}

//BatchWriteItem for Batch Operation
type BatchWriteItem struct {
	RequestItems map[string][]BatchWriteSubItems `json:"RequestItems"`
}

//BatchWriteSubItems is for BatchWriteItem
type BatchWriteSubItems struct {
	DelReq BatchDeleteItem `json:"DeleteRequest"`
	PutReq BatchPutItem    `json:"PutRequest"`
}

//BatchDeleteItem is for BatchWriteSubItems
type BatchDeleteItem struct {
	Key map[string]*dynamodb.AttributeValue `json:"Key"`
}

//BatchPutItem is for BatchWriteSubItems
type BatchPutItem struct {
	Item map[string]*dynamodb.AttributeValue `json:"Item"`
}

// TableDDL - This contains the DDL
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
	TableDDL["dynamodb_adapter_table_ddl"] = map[string]string{"tableName": "STRING(MAX)", "column": "STRING(MAX)", "dataType": "STRING(MAX)", "originalColumn": "STRING(MAX)"}
	TableDDL["dynamodb_adapter_config_manager"] = map[string]string{"tableName": "STRING(MAX)", "config": "STRING(MAX)", "cronTime": "STRING(MAX)", "uniqueValue": "STRING(MAX)", "enabledStream": "STRING(MAX)", "pubsubTopic": "STRING(MAX)"}
	TableColumnMap = make(map[string][]string)
	TableColumnMap["dynamodb_adapter_table_ddl"] = []string{"tableName", "column", "dataType", "originalColumn"}
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

type dynamodbAdapterTableDdl struct {
	Table    string
	Column   string
	DataType string
}

// DBAudit for db auditing data
type DBAudit struct {
	ID           string                 `json:"id,omitempty"`
	Timestamp    int64                  `json:"timestamp,omitempty"`
	AWSResponse  map[string]interface{} `json:"awsResponse,omitempty"`
	GCPResponse  map[string]interface{} `json:"gcpResponse,omitempty"`
	TableName    string                 `json:"tableName,omitempty"`
	PartitionKey string                 `json:"partitionKey,omitempty"`
	SortKey      string                 `json:"sortKey,omitempty"`
	Operation    string                 `json:"operation"`
	GCPDbErr     string                 `json:"gcpDbErr"`
	AWSDbErr     string                 `json:"awsDbErr"`
	Payload      string                 `json:"payload"`
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
	OldImage       map[string]interface{} `json:"oldImage"`
	NewImage       map[string]interface{} `json:"newImage"`
	Keys           map[string]interface{} `json:"keys"`
	Timestamp      int64                  `json:"timestamp"`
	Table          string                 `json:"tableName"`
	EventName      string                 `json:"eventName"`
	SequenceNumber int64                  `json:"sequenceNumber"`
	EventID        string                 `json:"eventId"`
	EventSourceArn string                 `json:"eventSourceArn"`
}
