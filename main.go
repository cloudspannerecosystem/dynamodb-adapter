package main

import (
	"log"
	"net/http"

	rice "github.com/GeertJohan/go.rice"
	"github.com/cloudspannerecosystem/dynamodb-adapter/api"
	"github.com/cloudspannerecosystem/dynamodb-adapter/docs"
	"github.com/cloudspannerecosystem/dynamodb-adapter/initializer"
	"github.com/cloudspannerecosystem/dynamodb-adapter/storage"
	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/swaggo/gin-swagger/swaggerFiles"
)

// starting point of the application

// @title dynamodb-adapter APIs
// @description This is a API collection for dynamodb-adapter
// @version 1.0
// @host localhost:9050
// @BasePath /v1
func main() {

	// This will pack config-files folder inside binary
	// you need rice utility for it
	box := rice.MustFindBox("config-files")

	initErr := initializer.InitAll(box)
	if initErr != nil {
		log.Fatalln(initErr)
	}
	r := gin.Default()
	pprof.Register(r)
	r.GET("/doc/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	docs.SwaggerInfo.Host = ""
	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "Server is up and running!",
		})
	})
	r.NoRoute(func(c *gin.Context) {
		c.JSON(404, gin.H{"code": "RouteNotFound"})
	})
	api.InitAPI(r)
	go func() {
		err := r.Run(":9050")
		if err != nil {
			log.Fatal(err)
		}
	}()
	storage.GetStorageInstance().Close()
}
