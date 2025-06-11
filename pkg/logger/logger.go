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

	"github.com/cloudspannerecosystem/dynamodb-adapter/models"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var logger *zap.SugaredLogger

// init sets up a default development logger so that the global logger is never nil.
// This ensures logging calls are safe in tests and before Init() is called at runtime.
func init() {
	l, _ := zap.NewDevelopment()
	logger = l.Sugar()
}

// Init initializes the global zap logger using configuration from models.GlobalConfig.
//
// It sets up a zap logger with production settings, structured for Google Cloud Logging compatibility.
// The log level is determined by models.GlobalConfig.LogLevel. If the value is invalid, it defaults to INFO.
// The logger outputs to stdout/stderr, uses RFC3339 timestamps, and includes stack traces in the "stack_trace" field for error-level logs and above.
//
// This function must be called after models.GlobalConfig is populated with configuration values.
func Init() {
	var config zap.Config
	if models.GlobalConfig.GinMode == "debug" {
		config = zap.NewDevelopmentConfig()
	} else {
		config = zap.NewProductionConfig()
		config.DisableStacktrace = false
		config.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
		config.EncoderConfig.EncodeTime = zapcore.RFC3339TimeEncoder
		config.EncoderConfig.LevelKey = "severity"
		config.EncoderConfig.MessageKey = "message"
		config.EncoderConfig.StacktraceKey = "stack_trace"
		config.EncoderConfig.TimeKey = "timestamp"
		config.ErrorOutputPaths = []string{"stderr"}
		config.OutputPaths = []string{"stdout"}
	}

	level, err := zapcore.ParseLevel(models.GlobalConfig.LogLevel)
	if err != nil {
		log.Printf("Invalid LOG_LEVEL '%s', defaulting to INFO: %v", models.GlobalConfig.LogLevel, err)
		level = zapcore.InfoLevel
	}
	config.Level = zap.NewAtomicLevelAt(level)

	zapLogger, err := config.Build(zap.AddCaller(), zap.AddCallerSkip(1))
	if err != nil {
		log.Fatalf("can't initialize zap logger: %v", err)
	}

	logger = zapLogger.Sugar()
}

// Desugar returns the underlying zap.Logger from the package logger.
func Desugar() *zap.Logger {
	return logger.Desugar()
}

func Error(message ...interface{}) {
	logger.Error(message...)
}

func Errorf(format string, message ...interface{}) {
	logger.Errorf(format, message...)
}

func Info(message ...interface{}) {
	logger.Info(message...)
}

func Infof(format string, message ...interface{}) {
	logger.Infof(format, message...)
}

func Warn(message ...interface{}) {
	logger.Warn(message...)
}

func Warnf(format string, message ...interface{}) {
	logger.Warnf(format, message...)
}

func Debug(message ...interface{}) {
	logger.Debug(message...)
}

func Debugf(format string, message ...interface{}) {
	logger.Debugf(format, message...)
}

func Fatal(message ...interface{}) {
	logger.Fatal(message...)
}

func Fatalf(format string, message ...interface{}) {
	logger.Fatalf(format, message...)
}
