package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"url-shortener/internal/store"
)

func TestShortenAndRedirect(t *testing.T) {
	s := store.NewMemoryStore()
	h := NewHandler(s, "http://localhost:8080")

	reqBody := []byte(`{"url":"https://go.dev/doc/"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewBuffer(reqBody))
	w := httptest.NewRecorder()

	h.Router(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", w.Code)
	}

	var resp ShortenResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Code == "" || resp.ShortURL == "" {
		t.Fatalf("empty response fields")
	}

	req2 := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewBuffer(reqBody))
	w2 := httptest.NewRecorder()

	h.Router(w2, req2)

	var resp2 ShortenResponse
	json.Unmarshal(w2.Body.Bytes(), &resp2)

	if resp.Code != resp2.Code {
		t.Fatalf("expected idempotency same code %s, got %s", resp.Code, resp2.Code)
	}

	reqRedirect := httptest.NewRequest(http.MethodGet, "/"+resp.Code, nil)
	wRedirect := httptest.NewRecorder()

	h.Router(wRedirect, reqRedirect)

	if wRedirect.Code != http.StatusFound {
		t.Fatalf("expected 302 Found, got %d", wRedirect.Code)
	}

	if loc := wRedirect.Header().Get("Location"); loc != "https://go.dev/doc" {
		t.Fatalf("expected location https://go.dev/doc, got %s", loc)
	}
}

func TestBadRequestScenarios(t *testing.T) {
	s := store.NewMemoryStore()
	h := NewHandler(s, "http://localhost:8080")

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{"Empty Body", ``, http.StatusBadRequest},
		{"Invalid JSON", `{"url":}`, http.StatusBadRequest},
		{"Empty URL", `{"url":""}`, http.StatusBadRequest},
		{"Unsupported Scheme", `{"url":"ftp://golang.org"}`, http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewBufferString(tt.body))
			w := httptest.NewRecorder()

			h.Router(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}

func TestConcurrentSameURL(t *testing.T) {
	s := store.NewMemoryStore()
	h := NewHandler(s, "http://localhost:8080")

	const goroutines = 50
	var wg sync.WaitGroup
	codes := make([]string, goroutines)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			reqBody := []byte(`{"url":"https://example.com/concurrent"}`)
			req := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewBuffer(reqBody))
			w := httptest.NewRecorder()

			h.Router(w, req)

			var resp ShortenResponse
			json.Unmarshal(w.Body.Bytes(), &resp)
			codes[idx] = resp.Code
		}(i)
	}

	wg.Wait()

	firstCode := codes[0]
	for _, c := range codes {
		if c != firstCode {
			t.Fatalf("expected all codes to be %s, but got %s", firstCode, c)
		}
	}
}