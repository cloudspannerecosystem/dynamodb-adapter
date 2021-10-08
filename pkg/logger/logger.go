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

package logger

import (
	"log"
	"os"

	"go.uber.org/zap"
)

var env = os.Getenv("ACTIVE_ENV")
var logger *zap.SugaredLogger

// init - this will init logger in the project
func init() {
	config := zap.NewProductionConfig()
	config.DisableStacktrace = true

	if env != "PRODUCTION" {
		config = zap.NewDevelopmentConfig()
	}

	tmp, err := config.Build()
	if err != nil {
		log.Fatal(err)
	}
	logger = tmp.Sugar()
	defer func() {
		err = logger.Sync()
	}()
	if err != nil {
		log.Fatal(err)
	}
}

// LogError - This is error level log
func LogError(message ...interface{}) {
	logger.Error(message)
}

// LogErrorF - This is error level log
func LogErrorF(format string, message ...interface{}) {
	logger.Errorf(format, message...)
}

// LogInfo - This is Info level log
func LogInfo(message ...interface{}) {
	logger.Info(message)
}

// LogWarn - This is Warn level log
func LogWarn(message ...interface{}) {
	logger.Warn(message)
}

// LogDebug - This is debug level log
func LogDebug(message ...interface{}) {
	logger.Debug(message)
}

// LogFatal - This log error and fatal it
func LogFatal(message ...interface{}) {
	logger.Fatal(message)
}

// ErrorLogging - This log error and fatal it
func ErrorLogging(message ...interface{}) {
	logger.Error(message)
}
