package main

import (
	"log"
	"net/http"
	"os"
	"sOPown3d/internal/server"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	srv, err := server.New("127.0.0.1:" + port)
	if err != nil {
		log.Fatalf("failed to create server: %v", err)
	}

	log.Printf("=== sOPown3d C2 ===\nListening on http://%s\n", srv.Addr)

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}
