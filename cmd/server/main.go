package main

import (
	"flag"
	"log"
	"net/http"
	"time"

	"url-shortener/internal/handler"
	"url-shortener/internal/store"
)

func main() {
	addr := flag.String("addr", ":8080", "Server listen address")
	baseURL := flag.String("base", "http://localhost:8080", "Base URL for short links")
	flag.Parse()

	memStore := store.NewMemoryStore()
	h := handler.NewHandler(memStore, *baseURL)

	mux := http.NewServeMux()
	mux.HandleFunc("/", h.Router)

	srv := &http.Server{
		Addr:         *addr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,  
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second, 
	}

	log.Printf("Server starting on %s (Base URL: %s)...", *addr, *baseURL)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed to start: %v", err)
	}
}