package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"project-management-tools/internal/adapter/driving/httpserver"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	router := httpserver.NewRouter()

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	log.Printf("server running on port %s", port)

	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
