package store

import (
	"errors"
	"testing"
)

func TestMemoryStore(t *testing.T) {
	s := NewMemoryStore()

	// 1. Test GetCodeOrShorten
	code, err := s.GetCodeOrShorten("https://example.com")
	if err != nil || code == "" {
		t.Fatalf("failed to shorten url: %v", err)
	}

	// Test Idempotency (same URL -> same code)
	code2, err := s.GetCodeOrShorten("https://example.com")
	if err != nil || code != code2 {
		t.Fatalf("expected same code %s, got %s", code, code2)
	}

	// 2. Test GetURL
	url, exists := s.GetURL(code)
	if !exists || url != "https://example.com" {
		t.Fatalf("expected https://example.com, got %s", url)
	}

	_, exists = s.GetURL("unknown")
	if exists {
		t.Fatalf("expected non-existent code to return false")
	}

	// 3. Test GetLink
	l, err := s.GetLink(code)
	if err != nil || l.OriginalURL != "https://example.com" {
		t.Fatalf("expected valid link struct, got error: %v", err)
	}

	// 4. Test GetLink Not Found
	_, err = s.GetLink("unknown")
	if err == nil || !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}