package api

import (
	v1 "github.com/cloudspannerecosystem/dynamodb-adapter/api/v1"

	"github.com/gin-gonic/gin"
)

// InitAPI - initialize api
func InitAPI(g *gin.Engine) {
	r := g.Group("/v1")
	v1.InitDBAPI(r)

}
