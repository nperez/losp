package store

import (
	"strings"
	"sync"

	"nickandperla.net/losp/internal/expr"
)

// Memory is an in-memory store for testing.
type Memory struct {
	mu       sync.RWMutex
	data     map[string]expr.Expr
	metadata map[string]string

	// Corpus support
	corpora    map[string]bool              // corpus name -> exists
	members    map[string][]string          // corpus name -> member names
	ftsContent map[string]map[string]string // corpus name -> expr name -> content
	embeddings map[string]map[string][]float32
	vecIndexes map[string][]byte
}

// NewMemory creates a new in-memory store.
func NewMemory() *Memory {
	return &Memory{
		data:       make(map[string]expr.Expr),
		metadata:   make(map[string]string),
		corpora:    make(map[string]bool),
		members:    make(map[string][]string),
		ftsContent: make(map[string]map[string]string),
		embeddings: make(map[string]map[string][]float32),
		vecIndexes: make(map[string][]byte),
	}
}

// Get retrieves an expression by name.
func (m *Memory) Get(name string) (expr.Expr, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if e, ok := m.data[name]; ok {
		return e, nil
	}
	return nil, nil
}

// Put stores an expression by name.
func (m *Memory) Put(name string, e expr.Expr) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[name] = e
	return nil
}

// Delete removes an expression by name.
func (m *Memory) Delete(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, name)
	return nil
}

// Close is a no-op for memory store.
func (m *Memory) Close() error {
	return nil
}

// GetMetadata retrieves a metadata value by key.
func (m *Memory) GetMetadata(key string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.metadata[key], nil
}

// SetMetadata stores a metadata value by key.
func (m *Memory) SetMetadata(key, value string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.metadata[key] = value
	return nil
}

// CorpusExists checks if a corpus exists.
func (m *Memory) CorpusExists(name string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.corpora[name], nil
}

// CreateCorpus creates a corpus.
func (m *Memory) CreateCorpus(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.corpora[name] = true
	return nil
}

// AddCorpusMember adds an expression to a corpus.
func (m *Memory) AddCorpusMember(corpus, exprName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, n := range m.members[corpus] {
		if n == exprName {
			return nil
		}
	}
	m.members[corpus] = append(m.members[corpus], exprName)
	return nil
}

// GetCorpusMembers returns all members of a corpus.
func (m *Memory) GetCorpusMembers(corpus string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.members[corpus], nil
}

// CreateFTSTable is a no-op for memory (FTS is simulated).
func (m *Memory) CreateFTSTable(corpus string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.ftsContent[corpus] == nil {
		m.ftsContent[corpus] = make(map[string]string)
	}
	return nil
}

// UpdateFTSContent stores content for FTS simulation.
func (m *Memory) UpdateFTSContent(corpus, exprName, content string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.ftsContent[corpus] == nil {
		m.ftsContent[corpus] = make(map[string]string)
	}
	m.ftsContent[corpus][exprName] = content
	return nil
}

// SearchFTS performs a simple substring search (simulates FTS5 MATCH).
func (m *Memory) SearchFTS(corpus, query string, limit int) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	content := m.ftsContent[corpus]
	if content == nil {
		return nil, nil
	}
	// Simple word-based matching: check if any query word appears in content
	queryWords := strings.Fields(strings.ToLower(query))
	var results []string
	for name, text := range content {
		lower := strings.ToLower(text)
		for _, w := range queryWords {
			if strings.Contains(lower, w) {
				results = append(results, name)
				break
			}
		}
		if len(results) >= limit {
			break
		}
	}
	return results, nil
}

// StoreEmbedding stores an embedding vector.
func (m *Memory) StoreEmbedding(corpus, exprName string, vector []float32) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.embeddings[corpus] == nil {
		m.embeddings[corpus] = make(map[string][]float32)
	}
	m.embeddings[corpus][exprName] = vector
	return nil
}

// GetEmbeddings retrieves all embeddings for a corpus.
func (m *Memory) GetEmbeddings(corpus string) (map[string][]float32, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.embeddings[corpus], nil
}

// StoreVectorIndex stores a serialized HNSW index.
func (m *Memory) StoreVectorIndex(corpus string, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.vecIndexes[corpus] = data
	return nil
}

// GetVectorIndex retrieves a serialized HNSW index.
func (m *Memory) GetVectorIndex(corpus string) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.vecIndexes[corpus], nil
}

// CorpusStore is the interface for corpus-related database operations.
type CorpusStore interface {
	CorpusExists(name string) (bool, error)
	CreateCorpus(name string) error
	AddCorpusMember(corpus, exprName string) error
	GetCorpusMembers(corpus string) ([]string, error)
	CreateFTSTable(corpus string) error
	UpdateFTSContent(corpus, exprName, content string) error
	SearchFTS(corpus, query string, limit int) ([]string, error)
	StoreEmbedding(corpus, exprName string, vector []float32) error
	GetEmbeddings(corpus string) (map[string][]float32, error)
	StoreVectorIndex(corpus string, data []byte) error
	GetVectorIndex(corpus string) ([]byte, error)
}

// Verify both implementations satisfy CorpusStore.
var (
	_ CorpusStore = (*SQLite)(nil)
	_ CorpusStore = (*Memory)(nil)
)

