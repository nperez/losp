// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (c) 2023-2026 Nicholas R. Perez

package eval

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/coder/hnsw"
)

// Corpus is a searchable collection of expressions.
type Corpus struct {
	name       string
	members    []string
	hnswGraph  *hnsw.Graph[string]
	embeddings map[string][]float32
	ftsReady   bool
	vecReady   bool
}

// CorpusRegistry manages corpus handles across evaluators.
type CorpusRegistry struct {
	mu      sync.Mutex
	corpora map[string]*Corpus // corpus name -> corpus
	handles map[string]string  // handle ID -> corpus name
	counter atomic.Int64
}

// NewCorpusRegistry creates a new corpus registry.
func NewCorpusRegistry() *CorpusRegistry {
	return &CorpusRegistry{
		corpora: make(map[string]*Corpus),
		handles: make(map[string]string),
	}
}

// GetOrCreate returns an existing corpus or creates a new one.
// Returns the handle ID.
func (r *CorpusRegistry) GetOrCreate(name string) string {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if corpus already exists in registry
	if _, ok := r.corpora[name]; ok {
		// Return a new handle for the existing corpus
		return r.newHandleLocked(name)
	}

	// Create new corpus
	r.corpora[name] = &Corpus{
		name:       name,
		embeddings: make(map[string][]float32),
	}
	return r.newHandleLocked(name)
}

// newHandleLocked creates a new handle pointing to the named corpus.
// Caller must hold r.mu.
func (r *CorpusRegistry) newHandleLocked(name string) string {
	id := fmt.Sprintf("_corpus_%d", r.counter.Add(1))
	r.handles[id] = name
	return id
}

// Get retrieves a corpus by handle ID.
func (r *CorpusRegistry) Get(handleID string) *Corpus {
	r.mu.Lock()
	defer r.mu.Unlock()
	name, ok := r.handles[handleID]
	if !ok {
		return nil
	}
	return r.corpora[name]
}

// GetByName retrieves a corpus by name.
func (r *CorpusRegistry) GetByName(name string) *Corpus {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.corpora[name]
}

// SetCorpus stores a corpus directly (used when loading from DB).
func (r *CorpusRegistry) SetCorpus(name string, c *Corpus) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.corpora[name] = c
}

// AddMember adds an expression name to the corpus membership list.
func (c *Corpus) AddMember(name string) {
	for _, m := range c.members {
		if m == name {
			return // already a member
		}
	}
	c.members = append(c.members, name)
}

// Members returns the corpus member list.
func (c *Corpus) Members() []string {
	return c.members
}

// SetFTSReady marks the FTS index as built.
func (c *Corpus) SetFTSReady(ready bool) {
	c.ftsReady = ready
}

// FTSReady returns whether the FTS index is built.
func (c *Corpus) FTSReady() bool {
	return c.ftsReady
}

// SetVecReady marks the vector index as built.
func (c *Corpus) SetVecReady(ready bool) {
	c.vecReady = ready
}

// VecReady returns whether the vector index is built.
func (c *Corpus) VecReady() bool {
	return c.vecReady
}

// SetEmbedding stores an embedding vector for an expression.
func (c *Corpus) SetEmbedding(name string, vec []float32) {
	c.embeddings[name] = vec
}

// GetEmbedding retrieves an embedding vector for an expression.
func (c *Corpus) GetEmbedding(name string) ([]float32, bool) {
	v, ok := c.embeddings[name]
	return v, ok
}

// Embeddings returns all embeddings.
func (c *Corpus) Embeddings() map[string][]float32 {
	return c.embeddings
}

// SetHNSWGraph sets the HNSW graph for vector search.
func (c *Corpus) SetHNSWGraph(g *hnsw.Graph[string]) {
	c.hnswGraph = g
}

// HNSWGraph returns the HNSW graph.
func (c *Corpus) HNSWGraph() *hnsw.Graph[string] {
	return c.hnswGraph
}
