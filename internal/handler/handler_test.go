package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"url-shortener/internal/link"
	"url-shortener/internal/store"
)

type FakeStore struct {
	links map[string]link.Link
}

func NewFakeStore() *FakeStore {
	return &FakeStore{links: make(map[string]link.Link)}
}

func (f *FakeStore) GetCodeOrShorten(longURL string) (string, error) {
	for code, l := range f.links {
		if l.OriginalURL == longURL {
			return code, nil
		}
	}
	code := "fake123"
	f.links[code] = link.Link{
		Code:        code,
		OriginalURL: longURL,
		CreatedAt:   time.Now().UTC(),
	}
	return code, nil
}

func (f *FakeStore) GetURL(code string) (string, bool) {
	l, exists := f.links[code]
	return l.OriginalURL, exists
}

func (f *FakeStore) GetLink(code string) (link.Link, error) {
	l, exists := f.links[code]
	if !exists {
		return link.Link{}, store.ErrNotFound
	}
	return l, nil
}

func TestMetadataRouteWithFakeStore(t *testing.T) {
	fake := NewFakeStore()
	h := NewHandler(fake, "http://localhost:8080")

	fake.links["testCode"] = link.Link{
		Code:        "testCode",
		OriginalURL: "https://example.com/test",
		CreatedAt:   time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC),
	}

	t.Run("Metadata 200 OK", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/links/testCode", nil)
		w := httptest.NewRecorder()

		h.Router(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", w.Code)
		}

		var resp LinkMetadataResponse
		json.Unmarshal(w.Body.Bytes(), &resp)

		if resp.URL != "https://example.com/test" {
			t.Errorf("expected URL https://example.com/test, got %s", resp.URL)
		}
		if resp.CreatedAt != "2026-01-15T12:00:00Z" {
			t.Errorf("expected RFC3339 time format, got %s", resp.CreatedAt)
		}
	})

	t.Run("Metadata 404 Not Found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/links/unknownCode", nil)
		w := httptest.NewRecorder()

		h.Router(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404 Not Found, got %d", w.Code)
		}
	})
}

func TestErrorMappingWithErrorsIs(t *testing.T) {
	s := store.NewMemoryStore()
	h := NewHandler(s, "http://localhost:8080")

	t.Run("ErrInvalidURL maps to 400 Bad Request", func(t *testing.T) {
		reqBody := []byte(`{"url":"not-a-valid-url"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewBuffer(reqBody))
		w := httptest.NewRecorder()

		h.Router(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request, got %d", w.Code)
		}
	})

	t.Run("ErrNotFound maps to 404 Not Found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/links/nonexistent", nil)
		w := httptest.NewRecorder()

		h.Router(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404 Not Found, got %d", w.Code)
		}
	})
}

func TestIdempotencyAfterRefactor(t *testing.T) {
	s := store.NewMemoryStore()
	h := NewHandler(s, "http://localhost:8080")

	reqBody := []byte(`{"url":"https://go.dev/doc/"}`)

	req1 := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewBuffer(reqBody))
	w1 := httptest.NewRecorder()
	h.Router(w1, req1)

	var resp1 ShortenResponse
	json.Unmarshal(w1.Body.Bytes(), &resp1)

	req2 := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewBuffer(reqBody))
	w2 := httptest.NewRecorder()
	h.Router(w2, req2)

	var resp2 ShortenResponse
	json.Unmarshal(w2.Body.Bytes(), &resp2)

	if resp1.Code != resp2.Code {
		t.Fatalf("idempotency broken: expected code %s, got %s", resp1.Code, resp2.Code)
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