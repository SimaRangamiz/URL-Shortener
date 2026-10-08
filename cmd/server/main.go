package main

import (
	"flag"
	"log"
	"net/http"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"url-shortener/internal/handler"
	"url-shortener/internal/store"
)

func main() {
	addr := flag.String("addr", ":8080", "Server listen address")
	baseURL := flag.String("base", "http://localhost:8080", "Base URL for short links")
	storeType := flag.String("store", "memory", "Store type: memory or sqlite")
	dbPath := flag.String("db-path", "links.db", "SQLite database file path")
	flag.Parse()

	var currentStore store.Store

	if *storeType == "sqlite" {
		db, err := gorm.Open(sqlite.Open(*dbPath), &gorm.Config{})
		if err != nil {
			log.Fatalf("failed to connect to sqlite database: %v", err)
		}
		dbStore, err := store.NewDBStore(db)
		if err != nil {
			log.Fatalf("failed to create db store: %v", err)
		}
		currentStore = dbStore
		log.Printf("Using persistent SQLite store at %s", *dbPath)
	} else {
		currentStore = store.NewMemoryStore()
		log.Println("Using in-memory store")
	}

	h := handler.NewHandler(currentStore, *baseURL)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

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