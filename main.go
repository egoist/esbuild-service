package main

import (
	"log"
	"net/http"

	handler "github.com/egoist/esbuild-service/api"
)

func main() {
	log.Println("Open http://localhost:8080")
	http.HandleFunc("/", handler.Handler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
