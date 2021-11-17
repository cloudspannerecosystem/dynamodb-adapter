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

// Package config implements the functions for reading
// configuration files and saving them into Golang objects
package config

import (
	"encoding/json"
	"os"
	"strings"
	"sync"

	rice "github.com/GeertJohan/go.rice"
	"github.com/cloudspannerecosystem/dynamodb-adapter/models"
	"github.com/cloudspannerecosystem/dynamodb-adapter/pkg/errors"
	"github.com/cloudspannerecosystem/dynamodb-adapter/pkg/logger"
	"github.com/cloudspannerecosystem/dynamodb-adapter/utils"
)

// Configuration struct
type Configuration struct {
	GoogleProjectID string
	SpannerDb       string
	QueryLimit      int64
}

// ConfigurationMap pointer
var ConfigurationMap *Configuration

// DbConfigMap dynamo to Spanner
var DbConfigMap map[string]models.TableConfig

var once sync.Once

func init() {
	ConfigurationMap = new(Configuration)
}

// InitConfig loads ConfigurationMap and DbConfigMap in memory based on
// ACTIVE_ENV. If ACTIVE_ENV is not set or and empty string the environment
// is defaulted to staging.
// 
// These config files are read from rice-box
func InitConfig(box *rice.Box) {
	once.Do(func() {
		env := strings.ToLower(os.Getenv("ACTIVE_ENV"))
		if env == "" {
			env = "staging"
		}

		ConfigurationMap = new(Configuration)

		ba, err := box.Bytes(env + "/tables.json")
		if err != nil {
			logger.LogFatal(err)
		}
		err = json.Unmarshal(ba, &DbConfigMap)
		if err != nil {
			logger.LogFatal(err)
		}
		ba, err = box.Bytes(env + "/config.json")
		if err != nil {
			logger.LogFatal(err)
		}
		err = json.Unmarshal(ba, ConfigurationMap)
		if err != nil {
			logger.LogFatal(err)
		}
		ba, err = box.Bytes(env + "/spanner.json")
		if err != nil {
			logger.LogFatal(err)
		}
		tmp := make(map[string]string)
		err = json.Unmarshal(ba, &tmp)
		if err != nil {
			logger.LogFatal(err)
		}
		for k, v := range tmp {
			models.SpannerTableMap[utils.ChangeTableNameForSpanner(k)] = v
		}
	})
}

//GetTableConf returns table configuration from global map object
func GetTableConf(tableName string) (models.TableConfig, error) {
	tableConf, ok := DbConfigMap[tableName]
	if !ok {
		return models.TableConfig{}, errors.New("ResourceNotFoundException", tableName)
	}
	if tableConf.ActualTable == "" {
		tableConf.ActualTable = tableName
		return tableConf, nil
	} else if tableConf.ActualTable != "" {
		actualTable := tableConf.ActualTable
		tableConf = DbConfigMap[actualTable]
		tableConf.ActualTable = actualTable
		return tableConf, nil
	}
	return models.TableConfig{}, errors.New("ResourceNotFoundException", tableName)
}
