package v1

import (
	"fmt"
	"runtime/debug"

	"github.com/cloudspannerecosystem/dynamodb-adapter/pkg/errors"
	"github.com/gin-gonic/gin"
)

func PanicHandler(c *gin.Context) {
	if e := recover(); e != nil {
		stack := string(debug.Stack())
		fmt.Println(stack)
		c.JSON(errors.New("ServerInternalError", e, stack).HTTPResponse(e))
	}
}
