package store

import (
	"errors"
	"fmt"

	"gorm.io/gorm"

	"url-shortener/internal/link"
)

type DBStore struct {
	db *gorm.DB
}

func NewDBStore(db *gorm.DB) (*DBStore, error) {
	if err := db.AutoMigrate(&link.Link{}); err != nil {
		return nil, fmt.Errorf("failed to automigrate database: %w", err)
	}
	return &DBStore{db: db}, nil
}

func (s *DBStore) GetCodeOrShorten(longURL string) (string, error) {
	for attempt := 0; attempt < 5; attempt++ {
		code := link.GenerateCode(longURL, attempt)

		var existing link.Link
		err := s.db.Where("code = ?", code).First(&existing).Error
		if err == nil {
			if existing.OriginalURL == longURL {
				return existing.Code, nil
			}
			continue 
		}

		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return "", fmt.Errorf("database query error: %w", err)
		}

		newLink := link.Link{
			Code:        code,
			OriginalURL: longURL,
		}

		if err := s.db.Create(&newLink).Error; err != nil {
			return "", fmt.Errorf("database insert error: %w", err)
		}

		return code, nil
	}

	return "", fmt.Errorf("failed to generate unique code due to hash collisions")
}

func (s *DBStore) GetURL(code string) (string, bool) {
	var l link.Link
	err := s.db.Where("code = ?", code).First(&l).Error
	if err != nil {
		return "", false
	}
	return l.OriginalURL, true
}

func (s *DBStore) GetLink(code string) (link.Link, error) {
	var l link.Link
	err := s.db.Where("code = ?", code).First(&l).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return link.Link{}, fmt.Errorf("db lookup: %w", ErrNotFound)
		}
		return link.Link{}, err
	}
	return l, nil
}