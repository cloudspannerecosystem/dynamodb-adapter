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

// Package initializer initialize the project by initalizing configuration
// Creating DB connection and reading configuration tables from Spanner
package initializer

import (
	"github.com/cloudspannerecosystem/dynamodb-adapter/config"
	"github.com/cloudspannerecosystem/dynamodb-adapter/service/services"
	"github.com/cloudspannerecosystem/dynamodb-adapter/service/spanner"
	"github.com/cloudspannerecosystem/dynamodb-adapter/storage"
)

// InitAll - this will initialize all the project object
// Config, storage and all other global objects are initialize
func InitAll() error {
	config.InitConfig()
	storage.InitializeDriver()
	err := spanner.ParseDDL(true)
	if err != nil {
		return err
	}
	services.StartConfigManager()
	services.InitStream()
	return nil
}
