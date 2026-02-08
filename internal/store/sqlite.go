package store

import (
	"database/sql"
	"encoding/binary"
	"fmt"
	"math"
	"sync"

	"nickandperla.net/losp/internal/expr"
)

// Current schema version
const SchemaVersion = "2"

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

	if version == "" || version == "1" {
		// New DB or migrate from v1 to v2: add corpus tables
		if err := s.migrateToV2(); err != nil {
			db.Close()
			return nil, err
		}
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

// migrateToV2 creates corpus-related tables.
func (s *SQLite) migrateToV2() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS corpora (
			name TEXT PRIMARY KEY
		);
		CREATE TABLE IF NOT EXISTS corpus_members (
			corpus_name TEXT NOT NULL,
			expr_name TEXT NOT NULL,
			PRIMARY KEY (corpus_name, expr_name),
			FOREIGN KEY (corpus_name) REFERENCES corpora(name)
		);
		CREATE TABLE IF NOT EXISTS embeddings (
			corpus_name TEXT NOT NULL,
			expr_name TEXT NOT NULL,
			vector BLOB NOT NULL,
			PRIMARY KEY (corpus_name, expr_name)
		);
		CREATE TABLE IF NOT EXISTS vector_indexes (
			corpus_name TEXT PRIMARY KEY,
			index_data BLOB NOT NULL
		);
	`)
	return err
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

// CorpusExists checks if a corpus exists in the database.
func (s *SQLite) CorpusExists(name string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var n string
	err := s.db.QueryRow("SELECT name FROM corpora WHERE name = ?", name).Scan(&n)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// CreateCorpus creates a corpus entry in the database.
func (s *SQLite) CreateCorpus(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(`
		INSERT OR IGNORE INTO corpora (name) VALUES (?)
	`, name)
	return err
}

// AddCorpusMember adds an expression to a corpus.
func (s *SQLite) AddCorpusMember(corpus, exprName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(`
		INSERT OR IGNORE INTO corpus_members (corpus_name, expr_name) VALUES (?, ?)
	`, corpus, exprName)
	return err
}

// GetCorpusMembers returns all expression names in a corpus.
func (s *SQLite) GetCorpusMembers(corpus string) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rows, err := s.db.Query("SELECT expr_name FROM corpus_members WHERE corpus_name = ?", corpus)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var members []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		members = append(members, name)
	}
	return members, rows.Err()
}

// CreateFTSTable creates the FTS5 virtual table for a corpus.
func (s *SQLite) CreateFTSTable(corpus string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(fmt.Sprintf(
		`CREATE VIRTUAL TABLE IF NOT EXISTS "corpus_fts_%s" USING fts5(expr_name, content)`,
		corpus,
	))
	return err
}

// UpdateFTSContent inserts or updates FTS content for an expression.
func (s *SQLite) UpdateFTSContent(corpus, exprName, content string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Delete old entry then insert new one (FTS5 upsert pattern)
	table := fmt.Sprintf(`"corpus_fts_%s"`, corpus)
	_, _ = s.db.Exec(fmt.Sprintf(`DELETE FROM %s WHERE expr_name = ?`, table), exprName)
	_, err := s.db.Exec(fmt.Sprintf(`INSERT INTO %s (expr_name, content) VALUES (?, ?)`, table), exprName, content)
	return err
}

// SearchFTS performs a full-text search on a corpus.
func (s *SQLite) SearchFTS(corpus, query string, limit int) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	table := fmt.Sprintf(`"corpus_fts_%s"`, corpus)
	rows, err := s.db.Query(
		fmt.Sprintf(`SELECT expr_name FROM %s WHERE %s MATCH ? ORDER BY rank LIMIT ?`, table, table),
		query, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var results []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		results = append(results, name)
	}
	return results, rows.Err()
}

// StoreEmbedding stores a float32 vector as a BLOB for an expression in a corpus.
func (s *SQLite) StoreEmbedding(corpus, exprName string, vector []float32) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	blob := float32sToBytes(vector)
	_, err := s.db.Exec(`
		INSERT INTO embeddings (corpus_name, expr_name, vector) VALUES (?, ?, ?)
		ON CONFLICT(corpus_name, expr_name) DO UPDATE SET vector = excluded.vector
	`, corpus, exprName, blob)
	return err
}

// GetEmbeddings retrieves all embeddings for a corpus.
func (s *SQLite) GetEmbeddings(corpus string) (map[string][]float32, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rows, err := s.db.Query("SELECT expr_name, vector FROM embeddings WHERE corpus_name = ?", corpus)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[string][]float32)
	for rows.Next() {
		var name string
		var blob []byte
		if err := rows.Scan(&name, &blob); err != nil {
			return nil, err
		}
		result[name] = bytesToFloat32s(blob)
	}
	return result, rows.Err()
}

// StoreVectorIndex stores a serialized HNSW index for a corpus.
func (s *SQLite) StoreVectorIndex(corpus string, data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(`
		INSERT INTO vector_indexes (corpus_name, index_data) VALUES (?, ?)
		ON CONFLICT(corpus_name) DO UPDATE SET index_data = excluded.index_data
	`, corpus, data)
	return err
}

// GetVectorIndex retrieves a serialized HNSW index for a corpus.
func (s *SQLite) GetVectorIndex(corpus string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var data []byte
	err := s.db.QueryRow("SELECT index_data FROM vector_indexes WHERE corpus_name = ?", corpus).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return data, nil
}

// float32sToBytes converts a float32 slice to a byte slice.
func float32sToBytes(fs []float32) []byte {
	buf := make([]byte, len(fs)*4)
	for i, f := range fs {
		binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(f))
	}
	return buf
}

// bytesToFloat32s converts a byte slice to a float32 slice.
func bytesToFloat32s(b []byte) []float32 {
	fs := make([]float32, len(b)/4)
	for i := range fs {
		fs[i] = math.Float32frombits(binary.LittleEndian.Uint32(b[i*4:]))
	}
	return fs
}
