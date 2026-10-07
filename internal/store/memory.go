package store

import (
	"fmt"
	"sync"
	"time"

	"url-shortener/internal/link"
)

type MemoryStore struct {
	mu        sync.RWMutex
	codeToURL map[string]link.Link
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		codeToURL: make(map[string]link.Link),
	}
}

func (s *MemoryStore) GetCodeOrShorten(longURL string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for attempt := 0; attempt < 5; attempt++ {
		code := link.GenerateCode(longURL, attempt)

		existing, found := s.codeToURL[code]
		if !found {
			s.codeToURL[code] = link.Link{
				Code:        code,
				OriginalURL: longURL,
				CreatedAt:   time.Now().UTC(),
			}
			return code, nil
		}

		if existing.OriginalURL == longURL {
			return code, nil
		}
	}

	return "", fmt.Errorf("failed to generate unique code due to hash collisions")
}

func (s *MemoryStore) GetURL(code string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	l, exists := s.codeToURL[code]
	return l.OriginalURL, exists
}

func (s *MemoryStore) GetLink(code string) (link.Link, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	l, exists := s.codeToURL[code]
	if !exists {
		return link.Link{}, ErrNotFound
	}
	return l, nil
}