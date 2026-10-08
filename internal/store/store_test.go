package store

import (
	"errors"
	"testing"
)

func TestMemoryStore(t *testing.T) {
	s := NewMemoryStore()

	code, err := s.GetCodeOrShorten("https://example.com")
	if err != nil || code == "" {
		t.Fatalf("failed to shorten url: %v", err)
	}

	code2, err := s.GetCodeOrShorten("https://example.com")
	if err != nil || code != code2 {
		t.Fatalf("expected same code %s, got %s", code, code2)
	}

	url, exists := s.GetURL(code)
	if !exists || url != "https://example.com" {
		t.Fatalf("expected https://example.com, got %s", url)
	}

	_, exists = s.GetURL("unknown")
	if exists {
		t.Fatalf("expected non-existent code to return false")
	}

	l, err := s.GetLink(code)
	if err != nil || l.OriginalURL != "https://example.com" {
		t.Fatalf("expected valid link struct, got error: %v", err)
	}

	_, err = s.GetLink("unknown")
	if err == nil || !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}