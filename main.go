package main

import (
	handler "github.com/egoist/esbuild-service/api"
	"github.com/egoist/esbuild-service/logger"
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	r.GET("/build/*pkg", handler.Build)

	logger.Logger.Info("Open http://localhost:8080")
	logger.Logger.Fatal(r.Run(":8080"))
}
