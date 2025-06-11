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
	"context"
	"fmt"
	"strings"

	"log"

	"github.com/cloudspannerecosystem/dynamodb-adapter/models"
	"github.com/cloudspannerecosystem/dynamodb-adapter/pkg/errors"
	"github.com/cloudspannerecosystem/dynamodb-adapter/utils"
	"github.com/joho/godotenv"
	"github.com/spf13/viper"
	"golang.org/x/oauth2/google"
)

var (
	proxyReleaseVersion string
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

type DefaultConfigProvider struct{}

type ConfigProvider interface {
	GetTableConf(tableName string) (models.TableConfig, error)
}

func InitConfig(filepath string) {
	GlobalConfig, err := LoadConfig(filepath)
	if err != nil {
		log.Printf("failed to read config file: %v", err)
	}
	GlobalConfig.UserAgent = "dynamodb-adapter/" + proxyReleaseVersion
	models.GlobalConfig = GlobalConfig
}

func LoadConfig(configPath string) (*models.Config, error) {
	v := viper.New()
	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")

	// Allow env vars to override (e.g., SPANNER_PROJECT_ID)
	_ = godotenv.Overload(".env")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg models.Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Default SPANNER_PROJECT_ID to Google ADC project if not set
	if cfg.Spanner.ProjectID == "" || cfg.Spanner.ProjectID == "SPANNER_PROJECT_ID" {
		creds, err := google.FindDefaultCredentials(context.Background())
		if err != nil {
			log.Fatalf("SPANNER_PROJECT_ID is not set and ADC lookup failed: %v", err)
		}
		if creds.ProjectID == "" {
			log.Fatal("SPANNER_PROJECT_ID is not set and ADC did not provide a project ID")
		}
		cfg.Spanner.ProjectID = creds.ProjectID
	}
	// Default LogLevel based on Gin mode
	if cfg.LogLevel == "" {
		if cfg.GinMode == "debug" {
			cfg.LogLevel = "debug"
		} else {
			cfg.LogLevel = "info"
		}
	}

	return &cfg, nil
}

// GetTableConf returns table configuration from global map object
func GetTableConf(tableName string) (models.TableConfig, error) {
	tableConf, ok := models.DbConfigMap[utils.ChangeTableNameForSpanner(tableName)]
	if !ok {
		return models.TableConfig{}, errors.New("ResourceNotFoundException", tableName)
	}
	if tableConf.ActualTable == "" {
		tableConf.ActualTable = tableName
	}
	return tableConf, nil
}
