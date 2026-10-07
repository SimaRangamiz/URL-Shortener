package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"url-shortener/internal/link"
	"url-shortener/internal/store"
)

type ShortenRequest struct {
	URL string `json:"url"`
}

type ShortenResponse struct {
	Code     string `json:"code"`
	ShortURL string `json:"short_url"`
}

type Handler struct {
	store   store.Store
	baseURL string
}

func NewHandler(s store.Store, baseURL string) *Handler {
	return &Handler{
		store:   s,
		baseURL: strings.TrimRight(baseURL, "/"),
	}
}

func (h *Handler) Shorten(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ShortenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.URL == "" {
		http.Error(w, `{"error":"invalid JSON or missing url"}`, http.StatusBadRequest)
		return
	}

	normalizedURL, err := link.Validate(req.URL)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	code, err := h.store.GetCodeOrShorten(normalizedURL)
	if err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	resp := ShortenResponse{
		Code:     code,
		ShortURL: h.baseURL + "/" + code,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) Router(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/")

	if path == "api/shorten" {
		h.Shorten(w, r)
		return
	}

	if r.Method == http.MethodGet && path != "" {
		longURL, exists := h.store.GetURL(path)
		if !exists {
			http.NotFound(w, r)
			return
		}
		http.Redirect(w, r, longURL, http.StatusFound)
		return
	}

	http.NotFound(w, r)
}