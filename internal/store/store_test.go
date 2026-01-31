package store

import (
	"os"
	"testing"

	"nickandperla.net/losp/internal/expr"
)

func TestMemoryStore(t *testing.T) {
	s := NewMemory()
	defer s.Close()

	// Test Put and Get
	err := s.Put("test", expr.Text{Value: "hello"})
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	got, err := s.Get("test")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.String() != "hello" {
		t.Errorf("expected 'hello', got '%s'", got.String())
	}

	// Test Delete
	err = s.Delete("test")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	got, err = s.Get("test")
	if err != nil {
		t.Fatalf("Get after delete failed: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil after delete, got '%s'", got.String())
	}
}

func TestSQLiteStore(t *testing.T) {
	// Create temp file
	f, err := os.CreateTemp("", "losp-test-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	path := f.Name()
	f.Close()
	defer os.Remove(path)

	s, err := NewSQLite(path)
	if err != nil {
		t.Fatalf("Failed to create SQLite store: %v", err)
	}

	// Test Put and Get
	err = s.Put("test", expr.Text{Value: "world"})
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	got, err := s.Get("test")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.String() != "world" {
		t.Errorf("expected 'world', got '%s'", got.String())
	}

	// Close and reopen to verify persistence
	s.Close()

	s2, err := NewSQLite(path)
	if err != nil {
		t.Fatalf("Failed to reopen SQLite store: %v", err)
	}
	defer s2.Close()

	got, err = s2.Get("test")
	if err != nil {
		t.Fatalf("Get after reopen failed: %v", err)
	}
	if got.String() != "world" {
		t.Errorf("expected 'world' after reopen, got '%s'", got.String())
	}
}
