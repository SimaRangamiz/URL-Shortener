package store

import (
	"fmt"
	"testing"
)

func BenchmarkGetCodeOrShorten(b *testing.B) {
	s := NewMemoryStore()

	b.ReportAllocs()

	for i := 0; b.Loop(); i++ {
		url := fmt.Sprintf("https://example.com/page/%d", i)
		_, _ = s.GetCodeOrShorten(url)
	}
}

func BenchmarkGetURL(b *testing.B) {
	s := NewMemoryStore()
	code, _ := s.GetCodeOrShorten("https://go.dev/doc/")

	b.ReportAllocs()

	for b.Loop() {
		_, _ = s.GetURL(code)
	}
}