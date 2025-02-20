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
	"fmt"
	"runtime/debug"

	"github.com/cloudspannerecosystem/dynamodb-adapter/pkg/errors"
	"github.com/gin-gonic/gin"
)

// PanicHandler is global handler for all type of panic
func PanicHandler(c *gin.Context) {
	if e := recover(); e != nil {
		stack := string(debug.Stack())
		fmt.Println(stack)
		c.JSON(errors.New("ServerInternalError", e, stack).HTTPResponse(e))
	}
}
