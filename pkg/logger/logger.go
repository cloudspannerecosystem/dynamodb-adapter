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
	"fmt"
	"log"

	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var env = os.Getenv("ACTIVE_ENV")
var logger *zap.SugaredLogger
var errorLogger *zap.SugaredLogger

// init - this will init logger in the project
func init() {
	devConfig := zap.NewDevelopmentConfig()
	devConfig.DisableStacktrace = true
	w := MyWriter{}
	tmp, err := devConfig.Build(zap.AddCallerSkip(1), zap.WrapCore(func(zapcore.Core) zapcore.Core {
		return zapcore.NewCore(zapcore.NewJSONEncoder(devConfig.EncoderConfig), zapcore.AddSync(w), devConfig.Level)
	}))
	if err != nil {
		log.Fatal(err)
	}

	logger = tmp.Sugar()

	prodLogger := zap.NewProductionConfig()
	prodLogger.DisableStacktrace = true
	tempProd, err := prodLogger.Build(zap.AddCallerSkip(2), zap.WrapCore(func(zapcore.Core) zapcore.Core {
		return zapcore.NewCore(zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()), zapcore.AddSync(w), devConfig.Level)
	}))
	if err != nil {
		log.Fatal(err)
	}
	errorLogger = tempProd.Sugar()
	defer errorLogger.Sync()
	defer logger.Sync()
}

// LogError - This is error level log
func LogError(message ...interface{}) {
	if env != "PRODUCTION" {
		logger.Error(message)
	} else {
		errorLogger.Error(message)
	}
}

// LogErrorF - This is error level log
func LogErrorF(format string, message ...interface{}) {
	if env != "PRODUCTION" {
		logger.Errorf(format, message...)
	} else {
		errorLogger.Errorf(format, message...)
	}
}

// LogInfo - This is Info level log
func LogInfo(message ...interface{}) {
	if env != "PRODUCTION" {
		logger.Info(message)
	}
}

// LogWarn - This is Warn level log
func LogWarn(message ...interface{}) {
	logger.Warn(message)
}

// LogDebug - This is debug level log
func LogDebug(message ...interface{}) {
	if env != "PRODUCTION" {
		logger.Debug(message)
	}
}

// LogFatal - This log error and fatal it
func LogFatal(message ...interface{}) {
	errorLogger.Fatal(message)
}

// ErrorLogging - This log error and fatal it
func ErrorLogging(message ...interface{}) {
	errorLogger.Error(message)
}

// MyWriter - MyWriter
type MyWriter struct{}

func (m MyWriter) Write(ba []byte) (int, error) {
	if env == "PRODUCTION" {
		go fmt.Println(string(ba))
	} else {
		fmt.Println(string(ba))
	}
	return len(ba), nil
}
