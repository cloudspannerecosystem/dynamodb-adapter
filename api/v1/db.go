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

// Package v1 implements version-1 for DynamoDB-adapter APIs
package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/cloudspannerecosystem/dynamodb-adapter/config"
	"github.com/cloudspannerecosystem/dynamodb-adapter/models"
	"github.com/cloudspannerecosystem/dynamodb-adapter/pkg/errors"
	"github.com/cloudspannerecosystem/dynamodb-adapter/pkg/logger"
	"github.com/cloudspannerecosystem/dynamodb-adapter/service/services"
	"github.com/cloudspannerecosystem/dynamodb-adapter/storage"
	"github.com/gin-gonic/gin"
	"github.com/opentracing/opentracing-go"
)

type APIHandler struct {
	svc services.Service
}

func NewAPIHandler(svc services.Service) *APIHandler {
	return &APIHandler{svc: svc}
}

// InitDBAPI - routes for apis
func InitDBAPI(r *gin.Engine) {
	svc := services.GetServiceInstance()

	// Create API handler with dependency injection
	apiHandler := NewAPIHandler(svc)
	r.POST("/v1", apiHandler.RouteRequest)
}

// RouteRequest - parse X-Amz-Target and call appropiate handler
func (h *APIHandler) RouteRequest(c *gin.Context) {
	var amzTarget = c.Request.Header.Get("X-Amz-Target")
	switch strings.Split(amzTarget, ".")[1] {
	case "BatchGetItem":
		h.BatchGetItem(c)
	case "BatchWriteItem":
		h.BatchWriteItem(c)
	case "DeleteItem":
		h.DeleteItem(c)
	case "GetItem":
		h.GetItemMeta(c)
	case "PutItem":
		h.UpdateMeta(c)
	case "Query":
		h.QueryTable(c)
	case "Scan":
		h.Scan(c)
	case "UpdateItem":
		h.Update(c)
	case "TransactGetItems":
		h.TransactGetItems(c)
	default:
		c.JSON(errors.New("ValidationException", "Invalid X-Amz-Target header value of "+amzTarget).
			HTTPResponse("X-Amz-Target Header not supported"))
	}
}

func addParentSpanID(c *gin.Context, span opentracing.Span) opentracing.Span {
	parentSpanID := c.Request.Header.Get("X-B3-Spanid")
	traceID := c.Request.Header.Get("X-B3-Traceid")
	serviceName := c.Request.Header.Get("service-name")
	span = span.SetTag("parentSpanId", parentSpanID)
	span = span.SetTag("traceId", traceID)
	span = span.SetTag("service-name", serviceName)
	return span
}

// UpdateMeta Writes a record
// @Description Writes a record
// @Summary Writes record
// @ID put
// @Produce  json
// @Success 200 {object} gin.H
// @Param requestBody body models.Meta true "Please add request body of type models.Meta"
// @Failure 500 {object} gin.H "{"errorMessage":"We had a problem with our server. Try again later.","errorCode":"E0001"}"
// @Router /put/ [post]
// @Failure 401 {object} gin.H "{"errorMessage":"API access not allowed","errorCode": "E0005"}"
func (h *APIHandler) UpdateMeta(c *gin.Context) {
	defer PanicHandler(c)
	defer c.Request.Body.Close()
	carrier := opentracing.HTTPHeadersCarrier(c.Request.Header)
	spanContext, err := opentracing.GlobalTracer().Extract(opentracing.HTTPHeaders, carrier)
	if err != nil || spanContext == nil {
		logger.LogDebug(err)
	}
	span, ctx := opentracing.StartSpanFromContext(c.Request.Context(), c.Request.URL.RequestURI(), opentracing.ChildOf(spanContext))
	c.Request = c.Request.WithContext(ctx)
	defer span.Finish()
	addParentSpanID(c, span)
	var meta models.Meta
	if err := c.ShouldBindJSON(&meta); err != nil {
		c.JSON(errors.New("ValidationException", err).HTTPResponse(meta))
	} else {
		if allow := h.svc.MayIReadOrWrite(meta.TableName, true, "UpdateMeta"); !allow {
			c.JSON(http.StatusOK, gin.H{})
			return
		}
		logger.LogDebug(meta)
		meta.AttrMap, err = ConvertDynamoToMap(meta.TableName, meta.Item)
		if err != nil {
			c.JSON(errors.New("ValidationException", err).HTTPResponse(meta))
			return
		}
		meta.ExpressionAttributeMap, err = ConvertDynamoToMap(meta.TableName, meta.ExpressionAttributeValues)
		if err != nil {
			c.JSON(errors.New("ValidationException", err).HTTPResponse(meta))
			return
		}

		for k, v := range meta.ExpressionAttributeNames {
			meta.ConditionExpression = strings.ReplaceAll(meta.ConditionExpression, k, v)
		}

		res, err := put(c.Request.Context(), meta.TableName, meta.AttrMap, nil, meta.ConditionExpression, meta.ExpressionAttributeMap)
		if err != nil {
			c.JSON(errors.HTTPResponse(err, meta))
		} else {
			var output map[string]interface{}
			if meta.ReturnValues == "NONE" {
				output = nil
			} else {
				output, _ = ChangeMaptoDynamoMap(ChangeResponseToOriginalColumns(meta.TableName, res))
				output = map[string]interface{}{"Attributes": output}
			}

			c.JSON(http.StatusOK, output)
		}
	}
}

func put(ctx context.Context, tableName string, putObj map[string]interface{}, expr *models.UpdateExpressionCondition, conditionExp string, expressionAttr map[string]interface{}) (map[string]interface{}, error) {
	tableConf, err := config.GetTableConf(tableName)
	if err != nil {
		return nil, err
	}
	sKey := tableConf.SortKey
	pKey := tableConf.PartitionKey
	var oldResp map[string]interface{}

	oldResp, err = storage.GetStorageInstance().SpannerGet(ctx, tableName, putObj[pKey], putObj[sKey], nil)
	if err != nil {
		return nil, err
	}
	res, err := services.Put(ctx, tableName, putObj, nil, conditionExp, expressionAttr, oldResp)
	if err != nil {
		return nil, err
	}
	go services.StreamDataToThirdParty(oldResp, res, tableName)
	return oldResp, nil
}

func queryResponse(query models.Query, c *gin.Context, svc services.Service) {
	defer PanicHandler(c)
	defer c.Request.Body.Close()
	span, ctx := opentracing.StartSpanFromContext(c.Request.Context(), c.Request.URL.RequestURI())
	c.Request = c.Request.WithContext(ctx)
	defer span.Finish()
	var err1 error
	if allow := svc.MayIReadOrWrite(query.TableName, false, ""); !allow {
		c.JSON(http.StatusOK, gin.H{})
		return
	}

	if query.Select == "COUNT" {
		query.OnlyCount = true
	}

	query.StartFrom, err1 = ConvertDynamoToMap(query.TableName, query.ExclusiveStartKey)
	if err1 != nil {
		c.JSON(errors.New("ValidationException", err1).HTTPResponse(query))
		return
	}
	query.RangeValMap, err1 = ConvertDynamoToMap(query.TableName, query.ExpressionAttributeValues)
	if err1 != nil {
		c.JSON(errors.New("ValidationException", err1).HTTPResponse(query))
		return
	}

	if query.Limit == 0 {
		query.Limit = models.GlobalConfig.Spanner.QueryLimit
	}
	query.ExpressionAttributeNames = ChangeColumnToSpannerExpressionName(query.TableName, query.ExpressionAttributeNames)
	query = ReplaceHashRangeExpr(query)
	res, hash, err := services.QueryAttributes(c.Request.Context(), query)
	if err == nil {
		finalResult := make(map[string]interface{})
		changedOutput := ChangeQueryResponseColumn(query.TableName, res)
		if _, ok := changedOutput["Items"]; ok && changedOutput["Items"] != nil {
			changedOutput["Items"], err = ChangeMaptoDynamoMap(changedOutput["Items"])
			if err != nil {
				c.JSON(errors.HTTPResponse(err, "ItemsChangeError"))
			}
		}
		if _, ok := changedOutput["Items"].(map[string]interface{})["L"]; ok {
			finalResult["Count"] = changedOutput["Count"]
			finalResult["Items"] = changedOutput["Items"].(map[string]interface{})["L"]
		}

		if _, ok := changedOutput["LastEvaluatedKey"]; ok && changedOutput["LastEvaluatedKey"] != nil {
			finalResult["LastEvaluatedKey"], err = ChangeMaptoDynamoMap(changedOutput["LastEvaluatedKey"])
			if err != nil {
				c.JSON(errors.HTTPResponse(err, "LastEvaluatedKeyChangeError"))
			}
		}

		c.JSON(http.StatusOK, finalResult)
	} else {
		c.JSON(errors.HTTPResponse(err, query))
	}
	if hash != "" {
		span.SetTag("qHash", hash)
	}
}

// QueryTable queries a table
// @Description Query a table
// @Summary Query a table
// @ID query-table
// @Produce  json
// @Success 200 {object} gin.H
// @Param requestBody body models.Query true "Please add request body of type models.Query"
// @Failure 500 {object} gin.H "{"errorMessage":"We had a problem with our server. Try again later.","errorCode":"E0001"}"
// @Router /query/ [post]
// @Failure 401 {object} gin.H "{"errorMessage":"API access not allowed","errorCode": "E0005"}"
func (h *APIHandler) QueryTable(c *gin.Context) {
	defer PanicHandler(c)
	defer c.Request.Body.Close()
	carrier := opentracing.HTTPHeadersCarrier(c.Request.Header)
	spanContext, err := opentracing.GlobalTracer().Extract(opentracing.HTTPHeaders, carrier)
	if err != nil || spanContext == nil {
		logger.LogDebug(err)
	}
	span, ctx := opentracing.StartSpanFromContext(c.Request.Context(), c.Request.URL.RequestURI(), opentracing.ChildOf(spanContext))
	c.Request = c.Request.WithContext(ctx)
	defer span.Finish()
	addParentSpanID(c, span)
	var query models.Query
	if err := c.ShouldBindJSON(&query); err != nil {
		c.JSON(errors.New("ValidationException", err).HTTPResponse(query))
	} else {
		logger.LogInfo(query)
		queryResponse(query, c, h.svc)
	}
}

// GetItemMeta to get with projections
// @Description Get a record with projections
// @Summary Get a record with projections
// @ID get-with-projection
// @Produce  json
// @Success 200 {object} gin.H
// @Param requestBody body models.GetWithProjectionMeta true "Please add request body of type models.GetWithProjectionMeta"
// @Failure 500 {object} gin.H "{"errorMessage":"We had a problem with our server. Try again later.","errorCode":"E0001"}"
// @Router /getWithProjection/ [post]
// @Failure 401 {object} gin.H "{"errorMessage":"API access not allowed","errorCode": "E0005"}"
func (h *APIHandler) GetItemMeta(c *gin.Context) {
	defer PanicHandler(c)
	defer c.Request.Body.Close()
	carrier := opentracing.HTTPHeadersCarrier(c.Request.Header)
	spanContext, err := opentracing.GlobalTracer().Extract(opentracing.HTTPHeaders, carrier)
	if err != nil || spanContext == nil {
		logger.LogDebug(err)
	}
	span, ctx := opentracing.StartSpanFromContext(c.Request.Context(), c.Request.URL.RequestURI(), opentracing.ChildOf(spanContext))
	c.Request = c.Request.WithContext(ctx)
	defer span.Finish()
	span = addParentSpanID(c, span)
	var getItemMeta models.GetItemMeta
	if err := c.ShouldBindJSON(&getItemMeta); err != nil {
		c.JSON(errors.New("ValidationException", err).HTTPResponse(getItemMeta))
	} else {
		span.SetTag("table", getItemMeta.TableName)
		logger.LogDebug(getItemMeta)
		if allow := h.svc.MayIReadOrWrite(getItemMeta.TableName, false, ""); !allow {
			c.JSON(http.StatusOK, gin.H{})
			return
		}
		getItemMeta.PrimaryKeyMap, err = ConvertDynamoToMap(getItemMeta.TableName, getItemMeta.Key)
		if err != nil {
			c.JSON(errors.New("ValidationException", err).HTTPResponse(getItemMeta))
			return
		}
		getItemMeta.ExpressionAttributeNames = ChangeColumnToSpannerExpressionName(getItemMeta.TableName, getItemMeta.ExpressionAttributeNames)
		res, rowErr := services.GetWithProjection(c.Request.Context(), getItemMeta.TableName, getItemMeta.PrimaryKeyMap, getItemMeta.ProjectionExpression, getItemMeta.ExpressionAttributeNames)
		if rowErr == nil {
			changedColumns := ChangeResponseToOriginalColumns(getItemMeta.TableName, res)
			output, err := ChangeMaptoDynamoMap(changedColumns)
			if err != nil {
				c.JSON(errors.HTTPResponse(err, "OutputChangedError"))
			}
			output = map[string]interface{}{
				"Item": output,
			}
			c.JSON(http.StatusOK, output)
		} else {
			c.JSON(errors.HTTPResponse(rowErr, getItemMeta))
		}
	}
}

// BatchGetItem to get with projections
// @Description Request items in a batch with projections.
// @Summary Request items in a batch with projections.
// @ID batch-get-with-projection
// @Produce  json
// @Success 200 {object} gin.H
// @Param requestBody body models.BatchGetWithProjectionMeta true "Please add request body of type models.BatchGetWithProjectionMeta"
// @Failure 500 {object} gin.H "{"errorMessage":"We had a problem with our server. Try again later.","errorCode":"E0001"}"
// @Router /batchGetWithProjection/ [post]
// @Failure 401 {object} gin.H "{"errorMessage":"API access not allowed","errorCode": "E0005"}"
func (h *APIHandler) BatchGetItem(c *gin.Context) {
	start := time.Now()
	defer PanicHandler(c)
	defer c.Request.Body.Close()
	carrier := opentracing.HTTPHeadersCarrier(c.Request.Header)
	spanContext, err := opentracing.GlobalTracer().Extract(opentracing.HTTPHeaders, carrier)
	if err != nil || spanContext == nil {
		logger.LogDebug(err)
	}
	span, ctx := opentracing.StartSpanFromContext(c.Request.Context(), c.Request.URL.RequestURI(), opentracing.ChildOf(spanContext))
	c.Request = c.Request.WithContext(ctx)
	defer span.Finish()
	span = addParentSpanID(c, span)

	var batchGetMeta models.BatchGetMeta
	if err1 := c.ShouldBindJSON(&batchGetMeta); err1 != nil {
		c.JSON(errors.New("ValidationException", err1).HTTPResponse(batchGetMeta))
	} else {
		output := make(map[string]interface{})

		for k, v := range batchGetMeta.RequestItems {
			batchGetWithProjectionMeta := v
			batchGetWithProjectionMeta.TableName = k
			logger.LogDebug(batchGetWithProjectionMeta)
			if allow := h.svc.MayIReadOrWrite(batchGetWithProjectionMeta.TableName, false, ""); !allow {
				c.JSON(http.StatusOK, []gin.H{})
				return
			}
			var singleOutput interface{}
			singleOutput, span, err = batchGetDataSingleTable(c.Request.Context(), batchGetWithProjectionMeta, span)
			if err != nil {
				c.JSON(errors.HTTPResponse(err, batchGetWithProjectionMeta))
			}
			currOutput, err := ChangeMaptoDynamoMap(singleOutput)
			if err != nil {
				c.JSON(errors.HTTPResponse(err, batchGetWithProjectionMeta))
			}
			output[k] = currOutput["L"]
		}

		c.JSON(http.StatusOK, map[string]interface{}{"Responses": output})

		if time.Since(start) > time.Second*1 {
			go fmt.Println("BatchGetCall", batchGetMeta)
		}
	}
}

func batchGetDataSingleTable(ctx context.Context, batchGetWithProjectionMeta models.BatchGetWithProjectionMeta, span opentracing.Span) (interface{}, opentracing.Span, error) {

	var err1 error
	batchGetWithProjectionMeta.KeyArray, err1 = ConvertDynamoArrayToMapArray(batchGetWithProjectionMeta.TableName, batchGetWithProjectionMeta.Keys)
	if err1 != nil {
		return nil, nil, errors.New("ValidationException", err1.Error())
	}
	batchGetWithProjectionMeta.ExpressionAttributeNames = ChangeColumnToSpannerExpressionName(batchGetWithProjectionMeta.TableName, batchGetWithProjectionMeta.ExpressionAttributeNames)
	res, err2 := services.BatchGetWithProjection(ctx, batchGetWithProjectionMeta.TableName, batchGetWithProjectionMeta.KeyArray, batchGetWithProjectionMeta.ProjectionExpression, batchGetWithProjectionMeta.ExpressionAttributeNames)

	span = span.SetTag("table", batchGetWithProjectionMeta.TableName)
	span = span.SetTag("batchRequestCount", len(batchGetWithProjectionMeta.Keys))
	span = span.SetTag("batchResponseCount", len(res))

	if err2 != nil {
		return nil, span, err2
	}
	return ChangesArrayResponseToOriginalColumns(batchGetWithProjectionMeta.TableName, res), span, nil
}

// DeleteItem  ...
// @Description Delete Item from table
// @Summary Delete Item from table
// @ID delete-row
// @Produce  json
// @Success 200 {object} gin.H
// @Param requestBody body models.Delete true "Please add request body of type models.Delete"
// @Failure 500 {object} gin.H "{"errorMessage":"We had a problem with our server. Try again later.","errorCode":"E0001"}"
// @Router /deleteItem/ [post]
// @Failure 401 {object} gin.H "{"errorMessage":"API access not allowed","errorCode": "E0005"}"
func (h *APIHandler) DeleteItem(c *gin.Context) {
	defer PanicHandler(c)
	defer c.Request.Body.Close()
	carrier := opentracing.HTTPHeadersCarrier(c.Request.Header)
	spanContext, err := opentracing.GlobalTracer().Extract(opentracing.HTTPHeaders, carrier)
	if err != nil || spanContext == nil {
		logger.LogDebug(err)
	}
	span, ctx := opentracing.StartSpanFromContext(c.Request.Context(), c.Request.URL.RequestURI(), opentracing.ChildOf(spanContext))
	c.Request = c.Request.WithContext(ctx)
	defer span.Finish()
	addParentSpanID(c, span)
	var deleteItem models.Delete
	if err := c.ShouldBindJSON(&deleteItem); err != nil {
		c.JSON(errors.New("ValidationException", err).HTTPResponse(deleteItem))
	} else {
		logger.LogDebug(deleteItem)
		if allow := h.svc.MayIReadOrWrite(deleteItem.TableName, true, "DeleteItem"); !allow {
			c.JSON(http.StatusOK, gin.H{})
			return
		}
		deleteItem.PrimaryKeyMap, err = ConvertDynamoToMap(deleteItem.TableName, deleteItem.Key)
		if err != nil {
			c.JSON(errors.New("ValidationException", err).HTTPResponse(deleteItem))
			return
		}
		deleteItem.ExpressionAttributeMap, err = ConvertDynamoToMap(deleteItem.TableName, deleteItem.ExpressionAttributeValues)
		if err != nil {
			c.JSON(errors.New("ValidationException", err).HTTPResponse(deleteItem))
			return
		}

		for k, v := range deleteItem.ExpressionAttributeNames {
			deleteItem.ConditionExpression = strings.ReplaceAll(deleteItem.ConditionExpression, k, v)
		}

		oldRes, _ := services.GetWithProjection(c.Request.Context(), deleteItem.TableName, deleteItem.PrimaryKeyMap, "", nil)
		err := services.Delete(c.Request.Context(), deleteItem.TableName, deleteItem.PrimaryKeyMap, deleteItem.ConditionExpression, deleteItem.ExpressionAttributeMap, nil)
		if err == nil {
			output, _ := ChangeMaptoDynamoMap(ChangeResponseToOriginalColumns(deleteItem.TableName, oldRes))
			c.JSON(http.StatusOK, map[string]interface{}{"Attributes": output})
			go services.StreamDataToThirdParty(oldRes, nil, deleteItem.TableName)
		} else {
			c.JSON(errors.HTTPResponse(err, deleteItem))
		}
	}
}

// Scan record from table
// @Description Scan records from table
// @Summary Scan records from table
// @ID scan
// @Produce  json
// @Success 200 {object} gin.H
// @Param requestBody body models.ScanMeta true "Please add request body of type models.ScanMeta"
// @Failure 500 {object} gin.H "{"errorMessage":"We had a problem with our server. Try again later.","errorCode":"E0001"}"
// @Router /scan/ [post]
// @Failure 401 {object} gin.H "{"errorMessage":"API access not allowed","errorCode": "E0005"}"
func (h *APIHandler) Scan(c *gin.Context) {
	defer PanicHandler(c)
	defer c.Request.Body.Close()
	carrier := opentracing.HTTPHeadersCarrier(c.Request.Header)
	spanContext, err := opentracing.GlobalTracer().Extract(opentracing.HTTPHeaders, carrier)
	if err != nil || spanContext == nil {
		logger.LogDebug(err)
	}
	span, ctx := opentracing.StartSpanFromContext(c.Request.Context(), c.Request.URL.RequestURI(), opentracing.ChildOf(spanContext))
	c.Request = c.Request.WithContext(ctx)
	defer span.Finish()
	addParentSpanID(c, span)
	var meta models.ScanMeta
	if err := c.ShouldBindJSON(&meta); err != nil {
		c.JSON(errors.New("ValidationException", err).HTTPResponse(meta))
	} else {
		if allow := h.svc.MayIReadOrWrite(meta.TableName, false, ""); !allow {
			c.JSON(http.StatusOK, gin.H{})
			return
		}

		meta.StartFrom, err = ConvertDynamoToMap(meta.TableName, meta.ExclusiveStartKey)
		if err != nil {
			c.JSON(errors.New("ValidationException", err).HTTPResponse(meta))
			return
		}

		meta.ExpressionAttributeMap, err = ConvertDynamoToMap(meta.TableName, meta.ExpressionAttributeValues)
		if err != nil {
			c.JSON(errors.New("ValidationException", err).HTTPResponse(meta))
			return
		}
		if meta.Select == "COUNT" {
			meta.OnlyCount = true
		}

		logger.LogDebug(meta)
		res, err := services.Scan(c.Request.Context(), meta)
		if err == nil {
			changedOutput := ChangeQueryResponseColumn(meta.TableName, res)
			if _, ok := changedOutput["Items"]; ok && changedOutput["Items"] != nil {
				itemsOutput, err := ChangeMaptoDynamoMap(changedOutput["Items"])
				if err != nil {
					c.JSON(errors.HTTPResponse(err, "ItemsChangeError"))
				}
				changedOutput["Items"] = itemsOutput["L"]
			}
			if _, ok := changedOutput["LastEvaluatedKey"]; ok && changedOutput["LastEvaluatedKey"] != nil {
				changedOutput["LastEvaluatedKey"], err = ChangeMaptoDynamoMap(changedOutput["LastEvaluatedKey"])
				if err != nil {
					c.JSON(errors.HTTPResponse(err, "LastEvaluatedKeyChangeError"))
				}
			}
			jsonData, _ := json.Marshal(res)
			c.JSON(http.StatusOK, json.RawMessage(jsonData))
		} else {
			c.JSON(errors.HTTPResponse(err, meta))
		}
	}
}

// Update updates a record in Spanner
// @Description updates a record in Spanner
// @Summary updates a record in Spanner
// @ID update
// @Produce  json
// @Success 200 {object} gin.H
// @Param requestBody body models.UpdateAttr true "Please add request body of type models.UpdateAttr"
// @Failure 500 {object} gin.H "{"errorMessage":"We had a problem with our server. Try again later.","errorCode":"E0001"}"
// @Router /update/ [post]
// @Failure 401 {object} gin.H "{"errorMessage":"API access not allowed","errorCode": "E0005"}"
func (h *APIHandler) Update(c *gin.Context) {
	defer PanicHandler(c)
	defer c.Request.Body.Close()
	carrier := opentracing.HTTPHeadersCarrier(c.Request.Header)
	spanContext, err := opentracing.GlobalTracer().Extract(opentracing.HTTPHeaders, carrier)
	if err != nil || spanContext == nil {
		logger.LogDebug(err)
	}
	span, ctx := opentracing.StartSpanFromContext(c.Request.Context(), c.Request.URL.RequestURI(), opentracing.ChildOf(spanContext))
	c.Request = c.Request.WithContext(ctx)
	defer span.Finish()
	addParentSpanID(c, span)
	var updateAttr models.UpdateAttr
	if err := c.ShouldBindJSON(&updateAttr); err != nil {
		c.JSON(errors.New("ValidationException", err).HTTPResponse(updateAttr))
	} else {
		if allow := h.svc.MayIReadOrWrite(updateAttr.TableName, true, "update"); !allow {
			c.JSON(http.StatusOK, gin.H{})
			return
		}
		updateAttr.PrimaryKeyMap, err = ConvertDynamoToMap(updateAttr.TableName, updateAttr.Key)
		if err != nil {
			c.JSON(errors.New("ValidationException", err).HTTPResponse(updateAttr))
			return
		}
		updateAttr.ExpressionAttributeMap, err = ConvertDynamoToMap(updateAttr.TableName, updateAttr.ExpressionAttributeValues)
		if err != nil {
			c.JSON(errors.New("ValidationException", err).HTTPResponse(updateAttr))
			return
		}
		resp, err := UpdateExpression(c.Request.Context(), updateAttr)
		if err != nil {
			c.JSON(errors.HTTPResponse(err, updateAttr))
		} else {
			c.JSON(http.StatusOK, resp)
		}
	}
}

// BatchWriteItem put & delete items in/from table
// @Description Batch Write Item for putting and deleting data in/from table
// @Summary Batch Write Items from table
// @ID batch-write-rows
// @Produce  json
// @Success 200 {object} gin.H
// @Param requestBody body models.BatchWriteItem true "Please add request body of type models.BatchWriteItem"
// @Failure 500 {object} gin.H "{"errorMessage":"We had a problem with our server. Try again later.","errorCode":"E0001"}"
// @Router /BatchWriteItem/ [post]
// @Failure 401 {object} gin.H "{"errorMessage":"API access not allowed","errorCode": "E0005"}"
func (h *APIHandler) BatchWriteItem(c *gin.Context) {
	defer PanicHandler(c)
	defer c.Request.Body.Close()
	carrier := opentracing.HTTPHeadersCarrier(c.Request.Header)
	spanContext, err := opentracing.GlobalTracer().Extract(opentracing.HTTPHeaders, carrier)
	if err != nil || spanContext == nil {
		logger.LogDebug(err)
	}
	span, ctx := opentracing.StartSpanFromContext(c.Request.Context(), c.Request.URL.RequestURI(), opentracing.ChildOf(spanContext))
	c.Request = c.Request.WithContext(ctx)
	defer span.Finish()
	addParentSpanID(c, span)
	var batchWriteItem models.BatchWriteItem
	var unprocessedBatchWriteItems models.BatchWriteItemResponse

	if err1 := c.ShouldBindJSON(&batchWriteItem); err1 != nil {
		c.JSON(errors.New("ValidationException", err1).HTTPResponse(batchWriteItem))
	} else {
		for key, value := range batchWriteItem.RequestItems {
			if allow := h.svc.MayIReadOrWrite(key, true, "BatchWriteItem"); !allow {
				c.JSON(http.StatusOK, gin.H{})
				return
			}
			var putData models.BatchMetaUpdate
			putData.TableName = key

			var deleteData models.BulkDelete
			deleteData.TableName = key

			for _, v := range value {
				if v.PutReq.Item != nil {
					putData.DynamoObject = append(putData.DynamoObject, v.PutReq.Item)
				}

				if v.DelReq.Key != nil {
					deleteData.DynamoObject = append(deleteData.DynamoObject, v.DelReq.Key)
				}
			}

			if putData.DynamoObject != nil {
				err = batchUpdateItems(c.Request.Context(), putData)
				if err != nil {
					for _, v := range value {
						if v.PutReq.Item != nil {
							unprocessedBatchWriteItems.UnprocessedItems[key] = append(unprocessedBatchWriteItems.UnprocessedItems[key], v)
						}
					}
				}
			}

			if deleteData.DynamoObject != nil {
				err = batchDeleteItems(c.Request.Context(), deleteData)
				if err != nil {
					for _, v := range value {
						if v.DelReq.Key != nil {
							unprocessedBatchWriteItems.UnprocessedItems[key] = append(unprocessedBatchWriteItems.UnprocessedItems[key], v)
						}
					}
				}
			}
		}

		c.JSON(http.StatusOK, unprocessedBatchWriteItems)
	}
}

func batchDeleteItems(con context.Context, bulkDelete models.BulkDelete) error {
	var err error
	bulkDelete.PrimaryKeyMapArray, err = ConvertDynamoArrayToMapArray(bulkDelete.TableName, bulkDelete.DynamoObject)
	if err != nil {
		return err
	}
	err = services.BatchDelete(con, bulkDelete.TableName, bulkDelete.PrimaryKeyMapArray)
	if err != nil {
		return err
	}
	return nil
}

func batchUpdateItems(con context.Context, batchMetaUpdate models.BatchMetaUpdate) error {
	var err error
	batchMetaUpdate.ArrAttrMap, err = ConvertDynamoArrayToMapArray(batchMetaUpdate.TableName, batchMetaUpdate.DynamoObject)
	if err != nil {
		return err
	}
	err = services.BatchPut(con, batchMetaUpdate.TableName, batchMetaUpdate.ArrAttrMap)
	if err != nil {
		return err
	}
	return nil
}

// TransactGetItems to get with projections
// @Description Request items in a batch with projections.
// @Summary Request items in a batch with projections.
// @ID transact-get-with-projection
// @Produce  json
// @Success 200 {object} gin.H
// @Param requestBody body models.TransactGetItemsRequest true "Please add request body of type models.TransactGetItemsRequest"
// @Failure 500 {object} gin.H "{"errorMessage":"We had a problem with our server. Try again later.","errorCode":"E0001"}"
// @Router /transactGetItems/ [post]
// @Failure 401 {object} gin.H "{"errorMessage":"API access not allowed","errorCode": "E0005"}"
func (h *APIHandler) TransactGetItems(c *gin.Context) {
	start := time.Now()
	ctx := c.Request.Context()

	// Parse request body into struct
	var transactGetMeta models.TransactGetItemsRequest
	if err := c.ShouldBindJSON(&transactGetMeta); err != nil {
		c.JSON(errors.New("ValidationException", err).HTTPResponse(transactGetMeta))
		return
	}
	// Iterate over each transact item
	for _, transactItem := range transactGetMeta.TransactItems {
		getRequest := transactItem.Get

		// Validate read permissions
		if allow := h.svc.MayIReadOrWrite(getRequest.TableName, false, ""); !allow {
			c.JSON(http.StatusOK, gin.H{"Responses": []gin.H{}})
			return
		}
	}
	// Fetch data from Spanner
	output, err := transactGetDataSingleTable(ctx, transactGetMeta, h.svc)
	if err != nil {
		c.JSON(errors.HTTPResponse(err, transactGetMeta))
		return
	}

	var currOutput []models.ResponseItem
	for _, row := range output {
		if row["Item"] != nil {
			dataMap, ok := row["Item"].(map[string]interface{})
			if !ok {
				c.JSON(errors.New("Invalid data format").HTTPResponse(transactGetMeta))
				return
			}
			convertedMap, err := ChangeMaptoDynamoMap(dataMap)
			if err != nil {
				c.JSON(errors.HTTPResponse(err, transactGetMeta))
				return
			}
			currOutput = append(currOutput, models.ResponseItem{
				TableName: row["TableName"],
				Item:      map[string]interface{}{"L": []interface{}{convertedMap}},
			})
		} else {
			c.JSON(errors.New("ValidationException").HTTPResponse(transactGetMeta))
		}
	}
	// Send final response
	c.JSON(http.StatusOK, gin.H{"Responses": currOutput})

	// Log slow transactions
	if time.Since(start) > time.Second*1 {
		go fmt.Println("TransactGetCall", transactGetMeta)
	}
}

// TransactGetDataSingleTable - fetch data from Spanner using Spanner TransactGetItems function
//
// This function takes a context, a TransactGetItemsRequest, a service, and returns a slice of maps and an error.
// The function first gets the table configuration using the table name from the TransactGetItemsRequest.
// Then it converts the projection expression to a slice of column names.
// Then it creates two slices, pValues and sValues, to store the partition key and the sort key values.
// Finally, it calls the SpannerTransactGetItems function on the Storage interface to fetch the data from Spanner.
func transactGetDataSingleTable(ctx context.Context, transactGetMeta models.TransactGetItemsRequest, svc services.Service) ([]map[string]interface{}, error) {
	// Convert DynamoDB Keys to Spanner KeyArray
	var err1 error

	tableProjectionCols := make(map[string][]string)
	pValues := make(map[string]interface{})
	sValues := make(map[string]interface{})

	// Iterate over the TransactGetItemsRequest
	for _, transactItem := range transactGetMeta.TransactItems {
		// Get the GetItemRequest
		getRequest := transactItem.Get

		// Convert the DynamoDB KeyArray to a Spanner-style KeyArray
		getRequest.KeyArray, err1 = ConvertDynamoArrayToMapArray(getRequest.TableName, []map[string]*dynamodb.AttributeValue{getRequest.Keys})
		if err1 != nil {
			return nil, nil
		}

		// Change ExpressionAttributeNames to Spanner-style
		getRequest.ExpressionAttributeNames = ChangeColumnToSpannerExpressionName(getRequest.TableName, getRequest.ExpressionAttributeNames)

		// Get the projection columns
		projectionCols, pvalues, svalues, _ := svc.TransactGetProjectionCols(ctx, getRequest)
		tableProjectionCols[getRequest.TableName] = projectionCols
		pValues[getRequest.TableName] = pvalues
		sValues[getRequest.TableName] = svalues
	}

	// Fetch data from Spanner
	return svc.TransactGetItem(ctx, tableProjectionCols, pValues, sValues)
}
