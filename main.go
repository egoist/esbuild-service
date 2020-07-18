package main

import (
	"log"

	handler "github.com/egoist/esbuild-service/api"
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	r.GET("/build", handler.Build)

	log.Println("Open http://localhost:8080")
	log.Fatal(r.Run(":8080"))
}
