package main

import (
	"flag"
	"log"
	"net/http"

	"url-shortener/internal/handler"
	"url-shortener/internal/store"
)

func main() {
	addr := flag.String("addr", ":8080", "Server listen address")
	baseURL := flag.String("", "http://localhost:8080", "")
	flag.Parse()

	memStore := store.NewMemoryStore()
	h := handler.NewHandler(memStore, *baseURL)

	mux := http.NewServeMux()
	mux.HandleFunc("/", h.Router)

	log.Printf("Server starting on %s (Base URL: %s)...", *addr, *baseURL)
	if err := http.ListenAndServe(*addr, mux); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}