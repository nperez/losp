package store

import (
	"database/sql"
	"fmt"
	"sync"

	"nickandperla.net/losp/internal/expr"
)

// Current schema version
const SchemaVersion = "1"

// SQLite is a SQLite-backed store.
type SQLite struct {
	mu sync.Mutex
	db *sql.DB
}

// NewSQLite creates a new SQLite store at the given path.
func NewSQLite(path string) (*SQLite, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	// Create tables if not exists
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS expressions (
			name TEXT PRIMARY KEY,
			value TEXT NOT NULL
		);
		CREATE TABLE IF NOT EXISTS metadata (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		);
	`)
	if err != nil {
		db.Close()
		return nil, err
	}

	s := &SQLite{db: db}

	// Check/set schema version (use unlocked versions since we're in init)
	version, err := s.getMetadataUnlocked("schema_version")
	if err != nil {
		db.Close()
		return nil, err
	}

	if version == "" {
		// First time - set schema version
		if err := s.setMetadataUnlocked("schema_version", SchemaVersion); err != nil {
			db.Close()
			return nil, err
		}
	} else if version != SchemaVersion {
		db.Close()
		return nil, fmt.Errorf("unsupported schema version: %s (expected %s)", version, SchemaVersion)
	}

	return s, nil
}

// Get retrieves an expression by name.
func (s *SQLite) Get(name string) (expr.Expr, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var value string
	err := s.db.QueryRow("SELECT value FROM expressions WHERE name = ?", name).Scan(&value)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return expr.Text{Value: value}, nil
}

// Put stores an expression by name.
func (s *SQLite) Put(name string, e expr.Expr) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	value := ""
	if e != nil {
		value = e.String()
	}

	_, err := s.db.Exec(`
		INSERT INTO expressions (name, value) VALUES (?, ?)
		ON CONFLICT(name) DO UPDATE SET value = excluded.value
	`, name, value)
	return err
}

// Delete removes an expression by name.
func (s *SQLite) Delete(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec("DELETE FROM expressions WHERE name = ?", name)
	return err
}

// Close closes the database connection.
func (s *SQLite) Close() error {
	return s.db.Close()
}

// GetMetadata retrieves a metadata value by key.
func (s *SQLite) GetMetadata(key string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.getMetadataUnlocked(key)
}

// getMetadataUnlocked retrieves metadata without locking (caller must hold lock).
func (s *SQLite) getMetadataUnlocked(key string) (string, error) {
	var value string
	err := s.db.QueryRow("SELECT value FROM metadata WHERE key = ?", key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return value, nil
}

// SetMetadata stores a metadata value by key.
func (s *SQLite) SetMetadata(key, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.setMetadataUnlocked(key, value)
}

// setMetadataUnlocked stores metadata without locking (caller must hold lock).
func (s *SQLite) setMetadataUnlocked(key, value string) error {
	_, err := s.db.Exec(`
		INSERT INTO metadata (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value
	`, key, value)
	return err
}
