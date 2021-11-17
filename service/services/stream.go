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
	"encoding/json"
	"sync"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/cloudspannerecosystem/dynamodb-adapter/config"
	"github.com/cloudspannerecosystem/dynamodb-adapter/models"
	"github.com/cloudspannerecosystem/dynamodb-adapter/pkg/logger"
	uuid "github.com/satori/go.uuid"
)

var pubsubClient *pubsub.Client
var mClients = map[string]*pubsub.Topic{}
var mux = &sync.Mutex{}

// InitStream for initializing the stream
func InitStream() {
	var err error
	pubsubClient, err = pubsub.NewClient(context.Background(), config.ConfigurationMap.GoogleProjectID)
	if err != nil {
		logger.LogFatal(err)
	}
}

// StreamDataToThirdParty for streaming data to any third party source
func StreamDataToThirdParty(oldImage, newImage map[string]interface{}, tableName string) {
	if !IsStreamEnabled(tableName) {
		return
	}
	if len(oldImage) == 0 && len(newImage) == 0 {
		return
	}
	streamObj := models.StreamDataModel{}
	tableConf, err := config.GetTableConf(tableName)
	if err == nil {
		streamObj.Keys = map[string]interface{}{}
		if len(oldImage) > 0 {
			streamObj.Keys[tableConf.PartitionKey] = oldImage[tableConf.PartitionKey]
			if tableConf.SortKey != "" {
				streamObj.Keys[tableConf.SortKey] = oldImage[tableConf.SortKey]
			}
		} else {
			streamObj.Keys[tableConf.PartitionKey] = newImage[tableConf.PartitionKey]
			if tableConf.SortKey != "" {
				streamObj.Keys[tableConf.SortKey] = newImage[tableConf.SortKey]
			}
		}
	}
	streamObj.EventID = uuid.NewV1().String()
	streamObj.EventSourceArn = "arn:aws:dynamodb:us-east-2:123456789012:table/" + tableName
	streamObj.OldImage = oldImage
	streamObj.NewImage = newImage
	streamObj.Timestamp = time.Now().UnixNano()
	streamObj.SequenceNumber = streamObj.Timestamp
	streamObj.Table = tableName
	if len(oldImage) == 0 {
		streamObj.EventName = "INSERT"
	} else if len(newImage) == 0 {
		streamObj.EventName = "REMOVE"
	} else {
		streamObj.EventName = "MODIFY"
	}
	connectors(&streamObj)
}

func connectors(streamObj *models.StreamDataModel) {
	go pubsubPublish(streamObj)
}

func pubsubPublish(streamObj *models.StreamDataModel) {
	var err error
	topicName, status := IsPubSubAllowed(streamObj.Table)
	if !status {
		return
	}
	mux.Lock()
	defer mux.Unlock()
	topic, ok := mClients[topicName]
	if !ok {
		topic = pubsubClient.
			TopicInProject(topicName, config.ConfigurationMap.GoogleProjectID)
		mClients[topicName] = topic
	}
	message := &pubsub.Message{}
	message.Data, err = json.Marshal(streamObj)
	if err != nil {
		logger.LogError(err)
	}
	_, err = topic.Publish(context.Background(), message).Get(ctx)
	if err != nil {
		logger.LogError(err)
	}
}
