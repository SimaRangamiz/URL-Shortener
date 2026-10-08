package store

import (
	"path/filepath"
	"testing"
	"url-shortener/internal/link"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestDBStoreRestart(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_links.db")

	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent), 
	}

	db1, err := gorm.Open(sqlite.Open(dbPath), gormConfig)
	if err != nil {
		t.Fatalf("failed to open sqlite db: %v", err)
	}

	store1, err := NewDBStore(db1)
	if err != nil {
		t.Fatalf("failed to init db store: %v", err)
	}

	testURL := "https://go.dev/doc/"
	code1, err := store1.GetCodeOrShorten(testURL)
	if err != nil {
		t.Fatalf("failed to shorten url: %v", err)
	}

	linkMeta, err := store1.GetLink(code1)
	if err != nil || linkMeta.OriginalURL != testURL {
		t.Fatalf("failed to get link metadata from db store: %v", err)
	}

	_, err = store1.GetLink("nonexistent")
	if err == nil {
		t.Fatalf("expected error for nonexistent code in GetLink")
	}

	sqlDB1, err := db1.DB()
	if err == nil {
		_ = sqlDB1.Close()
	}

	db2, err := gorm.Open(sqlite.Open(dbPath), gormConfig)
	if err != nil {
		t.Fatalf("failed to reopen sqlite db: %v", err)
	}

	sqlDB2, err := db2.DB()
	if err == nil {
		defer sqlDB2.Close()
	}

	store2, err := NewDBStore(db2)
	if err != nil {
		t.Fatalf("failed to init db store 2: %v", err)
	}

	url2, exists := store2.GetURL(code1)
	if !exists || url2 != testURL {
		t.Fatalf("restart simulation failed: expected %s, got %s (exists=%v)", testURL, url2, exists)
	}

	code2, err := store2.GetCodeOrShorten(testURL)
	if err != nil || code2 != code1 {
		t.Fatalf("idempotency broken after restart: expected code %s, got %s", code1, code2)
	}
}

func TestDBStoreCoverage(t *testing.T) {
		t.Run("AutoMigrate Error on Closed DB", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "closed.db")

		db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		})
		if err != nil {
			t.Fatalf("failed to open sqlite: %v", err)
		}

		sqlDB, _ := db.DB()
		sqlDB.Close() 

		_, err = NewDBStore(db)
		if err == nil {
			t.Errorf("expected error for AutoMigrate on closed db, got nil")
		}
	})

	t.Run("DB Query Errors on Closed DB", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "queries.db")

		db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		})
		if err != nil {
			t.Fatalf("failed to open sqlite: %v", err)
		}

		s, err := NewDBStore(db)
		if err != nil {
			t.Fatalf("failed to init db store: %v", err)
		}

		sqlDB, _ := db.DB()
		sqlDB.Close() 

		_, err = s.GetCodeOrShorten("https://example.com/test")
		if err == nil {
			t.Errorf("expected error when querying closed db, got nil")
		}

		_, exists := s.GetURL("somecode")
		if exists {
			t.Errorf("expected exists=false on closed db")
		}

		_, err = s.GetLink("somecode")
		if err == nil {
			t.Errorf("expected error on closed db for GetLink")
		}
	})

		t.Run("Collision Handling", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "collision.db")

		db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		})
		if err != nil {
			t.Fatalf("failed to open sqlite: %v", err)
		}

		sqlDB, err := db.DB()
		if err == nil {
			defer sqlDB.Close()
		}

		s, err := NewDBStore(db)
		if err != nil {
			t.Fatalf("failed to init db store: %v", err)
		}

		testURL := "https://example.com/collision-test"
		code := link.GenerateCode(testURL, 0)

		db.Create(&link.Link{
			Code:        code,
			OriginalURL: testURL,
		})

		gotCode, err := s.GetCodeOrShorten(testURL)
		if err != nil || gotCode != code {
			t.Errorf("expected code %s, got %s (err: %v)", code, gotCode, err)
		}
	})
}