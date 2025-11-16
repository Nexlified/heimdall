package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	r := chi.NewRouter()

	// Add standard middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Heartbeat("/health"))

	// Simple "Hello World" route
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write(byte("Heimdall is guarding the Bifröst!"))
	})

	port := "8080"
	log.Printf("Heimdall server starting on port %s...", port)
	
	// Start the server
	if err := http.ListenAndServe(":"+port, r); err!= nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
