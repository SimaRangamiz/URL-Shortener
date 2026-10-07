package link

import (
	"errors"
	"testing"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"Valid HTTP", "http://example.com", false},
		{"Valid HTTPS", "https://example.com/path/", false},
		{"Invalid Empty", "", true},
		{"Invalid Scheme", "ftp://example.com", true},
		{"Invalid URL", "not-a-url", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Validate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if tt.wantErr && !errors.Is(err, ErrInvalidURL) {
				t.Errorf("expected ErrInvalidURL, got %v", err)
			}
		})
	}
}

func TestGenerateCode(t *testing.T) {
	code1 := GenerateCode("https://example.com", 0)
	code2 := GenerateCode("https://example.com", 0)
	code3 := GenerateCode("https://example.com", 1)

	if code1 != code2 {
		t.Errorf("expected same code for same input, got %s and %s", code1, code2)
	}

	if code1 == code3 {
		t.Errorf("expected different code for different attempt, got %s", code1)
	}

	if len(code1) != 6 {
		t.Errorf("expected code length 6, got %d", len(code1))
	}
}