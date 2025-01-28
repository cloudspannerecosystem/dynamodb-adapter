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
	"fmt"
	"log"
	"os"

	"github.com/cloudspannerecosystem/dynamodb-adapter/models"
	"github.com/cloudspannerecosystem/dynamodb-adapter/pkg/errors"
	"gopkg.in/yaml.v2"
)

// Configuration struct
type Configuration struct {
	GoogleProjectID string
	SpannerDb       string
	QueryLimit      int64
}

// ConfigurationMap pointer
var ConfigurationMap *Configuration

func init() {
	ConfigurationMap = new(Configuration)
}

var readFile = os.ReadFile

func InitConfig() {
	GlobalConfig, err := loadConfig("config.yaml")
	if err != nil {
		log.Printf("failed to read config file: %v", err)
	}
	models.GlobalConfig = GlobalConfig
}

func loadConfig(filename string) (*models.Config, error) {
	data, err := readFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Unmarshal YAML data into config struct
	var config models.Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}

// GetTableConf returns table configuration from global map object
func GetTableConf(tableName string) (models.TableConfig, error) {
	tableConf, ok := models.DbConfigMap[tableName]
	if !ok {
		return models.TableConfig{}, errors.New("ResourceNotFoundException", tableName)
	}
	if tableConf.ActualTable == "" {
		tableConf.ActualTable = tableName
		return tableConf, nil
	} else if tableConf.ActualTable != "" {
		actualTable := tableConf.ActualTable
		tableConf = models.DbConfigMap[actualTable]
		tableConf.ActualTable = actualTable
		return tableConf, nil
	}
	return models.TableConfig{}, errors.New("ResourceNotFoundException", tableName)
}
