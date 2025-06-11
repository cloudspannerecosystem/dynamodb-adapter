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

package v1

import (
	"bytes"
	"io"
	"runtime/debug"

	"github.com/cloudspannerecosystem/dynamodb-adapter/pkg/errors"
	"github.com/cloudspannerecosystem/dynamodb-adapter/pkg/logger"
	"github.com/gin-gonic/gin"
)

// PanicHandler is global handler for all type of panic
func PanicHandler(c *gin.Context) {
	if e := recover(); e != nil {
		stack := string(debug.Stack())
		logger.Error("panic stack trace", stack)
		c.JSON(errors.New("ServerInternalError", e, stack).HTTPResponse(e))
	}
}

func LogRequestResponse() gin.HandlerFunc {
	return func(c *gin.Context) {
		// --- Log Request ---
		var reqBodyBytes []byte
		if c.Request.Body != nil {
			reqBodyBytes, _ = io.ReadAll(c.Request.Body)
			reqLog := map[string]interface{}{
				"message":     "HTTP Request",
				"method":      c.Request.Method,
				"path":        c.Request.URL.Path,
				"headers":     "REDACTED",
				"requestBody": string(reqBodyBytes),
			}
			logger.Debug(reqLog)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(reqBodyBytes))
		}

		// --- Log Response ---
		respBody := &bytes.Buffer{}
		blw := &bodyLogWriter{body: respBody, ResponseWriter: c.Writer}
		c.Writer = blw

		c.Next()

		respLog := map[string]interface{}{
			"message":      "HTTP Response",
			"method":       c.Request.Method,
			"path":         c.Request.URL.Path,
			"status":       c.Writer.Status(),
			"responseBody": respBody,
		}
		logger.Debug(respLog)
	}
}

// Helper to capture response body
type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *bodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}
