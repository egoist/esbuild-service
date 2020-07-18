package main

import (
	"github.com/egoist/esbuild-service/api"
	"github.com/egoist/esbuild-service/builder"
	"github.com/egoist/esbuild-service/logger"
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	b := builder.NewBuilder()

	r.GET("/build/*pkg", api.CreateBuildHandler(b))

	logger.Logger.Info("Open http://localhost:8080")
	logger.Logger.Fatal(r.Run(":8080"))
}
