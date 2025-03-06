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
	"strconv"
	"strings"
	"sync/atomic"

	"cloud.google.com/go/spanner"
	"github.com/cloudspannerecosystem/dynamodb-adapter/models"
	"github.com/cloudspannerecosystem/dynamodb-adapter/pkg/logger"
	"github.com/cloudspannerecosystem/dynamodb-adapter/storage"
	"github.com/robfig/cron"
)

var percentMap = make(map[string]int64)
var counterTableIndex = make(map[string]int)
var counters []int64
var ctx = context.Background()

var c *cron.Cron

// StartConfigManager starts config mangager for fetching the config manager data after certain time
func StartConfigManager() {
	c = cron.New()
	err := c.AddFunc("@every "+models.ConfigController.CornTime+"m", fetchConfigData)
	if err != nil {
		logger.LogError(err)
	}
	c.Start()
	fetchConfigData()
}

// MayIReadOrWrite for checking the operation is allowed or not
func MayIReadOrWrite(table string, IsMutation bool, operation string) bool {
	return true
}

func fetchConfigData() {
	logger.LogDebug("Fetching starts")
	stmt := spanner.Statement{}
	stmt.SQL = "SELECT * FROM dynamodb_adapter_config_manager"
	data, err := storage.GetStorageInstance().ExecuteSpannerQuery(ctx, "dynamodb_adapter_config_manager", []string{"tableName", "config", "cronTime", "uniqueValue", "enabledStream"}, false, stmt)
	if err != nil {
		models.ConfigController.StopConfigManager = true
		logger.LogDebug(err)
		return
	}
	if len(data) == 0 {
		models.ConfigController.StopConfigManager = true
		return
	}
	m := data[0]

	uniqueValue, _ := m["uniqueValue"].(string)
	if models.ConfigController.UniqueVal == uniqueValue {
		logger.LogDebug("No Changes in config detected", models.ConfigController.UniqueVal, "--", uniqueValue)
		return
	}
	logger.LogDebug("Changes in config detected", models.ConfigController.UniqueVal, "--", uniqueValue)

	cronTime, ok := m["cronTime"].(string)
	if !ok {
		models.ConfigController.StopConfigManager = true
		c.Stop()
		return
	}
	if cronTime == "0" {
		models.ConfigController.StopConfigManager = true
		c.Stop()
		return
	}
	if models.ConfigController.CornTime != cronTime {
		c.Stop()
		models.ConfigController.CornTime = cronTime
		StartConfigManager()
		return
	}
	models.ConfigController.UniqueVal = uniqueValue
	models.ConfigController.Mux.Lock()
	defer models.ConfigController.Mux.Unlock()
	models.ConfigController.ReadMap = map[string]struct{}{}
	models.ConfigController.WriteMap = map[string]struct{}{}
	percentMap = make(map[string]int64)
	counterTableIndex = make(map[string]int)
	counters = make([]int64, len(data))
	count := 0
	for _, tableConf := range data {
		tableName := tableConf["tableName"].(string)
		config := tableConf["config"].(string)
		parseConfig(tableName, config, count)
		enableStream, ok := tableConf["enabledStream"].(string)
		if ok && enableStream == "1" {
			models.ConfigController.StreamEnable[tableName] = struct{}{}
		} else {
			delete(models.ConfigController.StreamEnable, tableName)
		}

		count++
	}
}

func parseConfig(table string, config string, count int) {
	tokens := strings.Split(config, ",")
	if len(tokens) >= 1 && tokens[0] == "1" {
		models.ConfigController.ReadMap[table] = struct{}{}
	}
	if len(tokens) >= 2 && tokens[1] == "1" {
		models.ConfigController.WriteMap[table] = struct{}{}
	}
	if len(tokens) > 2 {
		if tokens[2] != "" {
			percent, err := strconv.ParseInt(tokens[2], 10, 64)
			if err == nil {
				percentMap[table] = percent
				counterTableIndex[table] = count
				counters[count] = 0
				atomic.StoreInt64(&counters[count], 0)
			}
		}
	}
}

// IsStreamEnabled checks if a table is enabled for streaming or not
func IsStreamEnabled(tableName string) bool {
	models.ConfigController.Mux.RLock()
	defer models.ConfigController.Mux.RUnlock()
	_, ok := models.ConfigController.StreamEnable[tableName]
	return ok
}
