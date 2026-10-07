package handler

import (
	"encoding/json"
	"net/http"
	"strings"
	"errors"
	"time"
	
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

type LinkMetadataResponse struct {
	URL       string `json:"url"`
	CreatedAt string `json:"created_at"` 
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func NewHandler(s store.Store, baseURL string) *Handler {
	return &Handler{
		store:   s,
		baseURL: strings.TrimRight(baseURL, "/"),
	}
}

func (h *Handler) jSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ErrorResponse{Error: msg})
}

func (h *Handler) Shorten(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.jSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req ShortenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.URL == "" {
		h.jSONError(w, http.StatusBadRequest, "invalid JSON or missing url")
		return
	}

	normalizedURL, err := link.Validate(req.URL)
	if err != nil {
		if errors.Is(err, link.ErrInvalidURL) {
			h.jSONError(w, http.StatusBadRequest, err.Error())
			return
		}
		h.jSONError(w, http.StatusBadRequest, "invalid URL input")
		return
	}

	code, err := h.store.GetCodeOrShorten(normalizedURL)
	if err != nil {
		h.jSONError(w, http.StatusInternalServerError, "internal server error")
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

	if strings.HasPrefix(path, "api/v1/links/") {
		code := strings.TrimPrefix(path, "api/v1/links/")
		if code == "" {
			h.jSONError(w, http.StatusNotFound, "missing code")
			return
		}
		h.GetMetadata(w, r, code)
		return
	}

	if r.Method == http.MethodGet && path != "" {
		longURL, exists := h.store.GetURL(path)
		if !exists {
			h.jSONError(w, http.StatusNotFound, "link not found")
			return
		}
		http.Redirect(w, r, longURL, http.StatusFound)
		return
	}

	h.jSONError(w, http.StatusNotFound, "route not found")
}


func (h *Handler) GetMetadata(w http.ResponseWriter, r *http.Request, code string) {
	if r.Method != http.MethodGet {
		h.jSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	l, err := h.store.GetLink(code)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			h.jSONError(w, http.StatusNotFound, "link not found")
			return
		}
		h.jSONError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	resp := LinkMetadataResponse{
		URL:       l.OriginalURL,
		CreatedAt: l.CreatedAt.Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}