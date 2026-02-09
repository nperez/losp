package store

import (
	"database/sql"
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

func TestMemoryVersioning(t *testing.T) {
	s := NewMemory()

	// Put creates version 1
	s.Put("X", expr.Text{Value: "first"})
	got, _ := s.Get("X")
	if got.String() != "first" {
		t.Errorf("expected 'first', got '%s'", got.String())
	}

	// Put again with different value creates version 2
	s.Put("X", expr.Text{Value: "second"})
	got, _ = s.Get("X")
	if got.String() != "second" {
		t.Errorf("expected 'second', got '%s'", got.String())
	}

	// Put with same value is a no-op (dedup)
	s.Put("X", expr.Text{Value: "second"})

	// GetHistory returns newest-first
	entries, err := s.GetHistory("X", 0)
	if err != nil {
		t.Fatalf("GetHistory failed: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Version != 2 || entries[0].Value != "second" {
		t.Errorf("entry[0]: expected v2 'second', got v%d '%s'", entries[0].Version, entries[0].Value)
	}
	if entries[1].Version != 1 || entries[1].Value != "first" {
		t.Errorf("entry[1]: expected v1 'first', got v%d '%s'", entries[1].Version, entries[1].Value)
	}

	// GetHistory with limit
	entries, err = s.GetHistory("X", 1)
	if err != nil {
		t.Fatalf("GetHistory with limit failed: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry with limit, got %d", len(entries))
	}
	if entries[0].Version != 2 {
		t.Errorf("expected v2 with limit, got v%d", entries[0].Version)
	}

	// GetHistory on nonexistent returns nil
	entries, err = s.GetHistory("nope", 0)
	if err != nil {
		t.Fatalf("GetHistory nonexistent failed: %v", err)
	}
	if entries != nil {
		t.Errorf("expected nil for nonexistent, got %v", entries)
	}

	// Delete removes all versions
	s.Delete("X")
	entries, err = s.GetHistory("X", 0)
	if err != nil {
		t.Fatalf("GetHistory after delete failed: %v", err)
	}
	if entries != nil {
		t.Errorf("expected nil after delete, got %v", entries)
	}
}

func TestSQLiteVersioning(t *testing.T) {
	f, err := os.CreateTemp("", "losp-ver-test-*.db")
	if err != nil {
		t.Fatalf("temp file: %v", err)
	}
	path := f.Name()
	f.Close()
	defer os.Remove(path)

	s, err := NewSQLite(path)
	if err != nil {
		t.Fatalf("NewSQLite: %v", err)
	}
	defer s.Close()

	// First put creates version 1
	s.Put("X", expr.Text{Value: "first"})
	got, _ := s.Get("X")
	if got.String() != "first" {
		t.Errorf("expected 'first', got '%s'", got.String())
	}

	// Second put with different value creates version 2
	s.Put("X", expr.Text{Value: "second"})
	got, _ = s.Get("X")
	if got.String() != "second" {
		t.Errorf("expected 'second', got '%s'", got.String())
	}

	// Same value is a no-op
	s.Put("X", expr.Text{Value: "second"})

	// GetHistory returns newest first
	entries, err := s.GetHistory("X", 0)
	if err != nil {
		t.Fatalf("GetHistory: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Version != 2 || entries[0].Value != "second" {
		t.Errorf("entry[0]: expected v2 'second', got v%d '%s'", entries[0].Version, entries[0].Value)
	}
	if entries[1].Version != 1 || entries[1].Value != "first" {
		t.Errorf("entry[1]: expected v1 'first', got v%d '%s'", entries[1].Version, entries[1].Value)
	}
	// Timestamps should be non-empty
	if entries[0].Ts == "" {
		t.Error("expected non-empty timestamp")
	}

	// GetHistory with limit
	entries, _ = s.GetHistory("X", 1)
	if len(entries) != 1 {
		t.Fatalf("expected 1 with limit, got %d", len(entries))
	}

	// Delete removes all versions
	s.Delete("X")
	entries, _ = s.GetHistory("X", 0)
	if len(entries) != 0 {
		t.Errorf("expected 0 after delete, got %d", len(entries))
	}
}

func TestSQLiteMigrationV2toV3(t *testing.T) {
	f, err := os.CreateTemp("", "losp-migrate-test-*.db")
	if err != nil {
		t.Fatalf("temp file: %v", err)
	}
	path := f.Name()
	f.Close()
	defer os.Remove(path)

	// Create a v2 database manually
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	db.Exec(`
		CREATE TABLE expressions (name TEXT PRIMARY KEY, value TEXT NOT NULL);
		CREATE TABLE metadata (key TEXT PRIMARY KEY, value TEXT NOT NULL);
		INSERT INTO metadata (key, value) VALUES ('schema_version', '2');
		INSERT INTO expressions (name, value) VALUES ('MyExpr', 'hello world');
		CREATE TABLE corpora (name TEXT PRIMARY KEY);
		CREATE TABLE corpus_members (corpus_name TEXT NOT NULL, expr_name TEXT NOT NULL, PRIMARY KEY (corpus_name, expr_name));
		CREATE TABLE embeddings (corpus_name TEXT NOT NULL, expr_name TEXT NOT NULL, vector BLOB NOT NULL, PRIMARY KEY (corpus_name, expr_name));
		CREATE TABLE vector_indexes (corpus_name TEXT PRIMARY KEY, index_data BLOB NOT NULL);
	`)
	db.Close()

	// Open with NewSQLite — should migrate to v3
	s, err := NewSQLite(path)
	if err != nil {
		t.Fatalf("NewSQLite after migration: %v", err)
	}
	defer s.Close()

	// Verify existing data preserved
	got, err := s.Get("MyExpr")
	if err != nil {
		t.Fatalf("Get after migration: %v", err)
	}
	if got == nil || got.String() != "hello world" {
		t.Errorf("expected 'hello world' after migration, got '%v'", got)
	}

	// Verify history works (existing row became version 1)
	entries, err := s.GetHistory("MyExpr", 0)
	if err != nil {
		t.Fatalf("GetHistory after migration: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry after migration, got %d", len(entries))
	}
	if entries[0].Version != 1 || entries[0].Value != "hello world" {
		t.Errorf("unexpected entry: v%d '%s'", entries[0].Version, entries[0].Value)
	}

	// New puts should version correctly
	s.Put("MyExpr", expr.Text{Value: "updated"})
	entries, _ = s.GetHistory("MyExpr", 0)
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries after update, got %d", len(entries))
	}
}
