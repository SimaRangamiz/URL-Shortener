package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
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

type LinkMetadataResponse struct {
	URL       string `json:"url"`
	CreatedAt string `json:"created_at"`
}

type ErrorResponse struct {
	Error string `json:"error"`
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

func (h *Handler) jsonError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(ErrorResponse{Error: msg})
}

func (h *Handler) Shorten(w http.ResponseWriter, r *http.Request) {
	var req ShortenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.URL == "" {
		h.jsonError(w, http.StatusBadRequest, "invalid JSON or missing url")
		return
	}

	normalizedURL, err := link.Validate(req.URL)
	if err != nil {
		if errors.Is(err, link.ErrInvalidURL) {
			h.jsonError(w, http.StatusBadRequest, err.Error())
			return
		}
		h.jsonError(w, http.StatusBadRequest, "invalid URL input")
		return
	}

	code, err := h.store.GetCodeOrShorten(normalizedURL)
	if err != nil {
		h.jsonError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	resp := ShortenResponse{
		Code:     code,
		ShortURL: h.baseURL + "/" + code,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *Handler) Redirect(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	if code == "" {
		h.jsonError(w, http.StatusNotFound, "missing code")
		return
	}

	longURL, exists := h.store.GetURL(code)
	if !exists {
		h.jsonError(w, http.StatusNotFound, "link not found")
		return
	}

	http.Redirect(w, r, longURL, http.StatusFound)
}

func (h *Handler) GetMetadata(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	if code == "" {
		h.jsonError(w, http.StatusNotFound, "missing code")
		return
	}

	l, err := h.store.GetLink(code)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			h.jsonError(w, http.StatusNotFound, "link not found")
			return
		}
		h.jsonError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	resp := LinkMetadataResponse{
		URL:       l.OriginalURL,
		CreatedAt: l.CreatedAt.Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/shorten", h.Shorten)
	mux.HandleFunc("GET /api/v1/links/{code}", h.GetMetadata)
	mux.HandleFunc("GET /{code}", h.Redirect)
}