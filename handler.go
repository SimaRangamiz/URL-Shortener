package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

var urlStore = make(map[string]string)

type Request struct {
	URL string `json:"url"`
}

type Response struct {
	ShortURL string `json:"short_url"`
}

func Handler(w http.ResponseWriter, r *http.Request) {

	var req Request
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil || req.URL == "" {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	code := generateRandomCode(6)

	urlStore[code] = req.URL

	resp := Response{
		ShortURL: fmt.Sprintf("http://localhost:8080/%s", code),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

func redirectHandler(w http.ResponseWriter, r *http.Request) {
	code := strings.TrimPrefix(r.URL.Path, "/")

	if code == "" {
		http.Error(w, "Code is required", http.StatusBadRequest)
		return
	}

	originalURL, exists := urlStore[code]
	if !exists {
		http.Error(w, "URL not found", http.StatusNotFound)
		return
	}

	http.Redirect(w, r, originalURL, http.StatusFound)
}