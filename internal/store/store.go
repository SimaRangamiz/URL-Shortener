package store

import (
	"errors"
	"url-shortener/internal/link"
)

var ErrNotFound = errors.New("code not found")

type Store interface {
	GetCodeOrShorten(longURL string) (string, error)
	GetURL(code string) (string, bool)
	GetLink(code string) (link.Link, error)
}