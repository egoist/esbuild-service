package main

import (
	"fmt"
	"os"

	"github.com/egoist/esbuild-service/api"
	"github.com/egoist/esbuild-service/builder"
	"github.com/egoist/esbuild-service/logger"
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	b := builder.NewBuilder()

	r.GET("/build/*pkg", api.CreateBuildHandler(b))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	url := fmt.Sprintf("localhost:%s", port)
	logger.Logger.Fatal(r.Run(url))
}
