package eval

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/coder/hnsw"
	"nickandperla.net/losp/internal/expr"
	"nickandperla.net/losp/internal/provider"
	"nickandperla.net/losp/internal/store"
)

func builtinCorpus(e *Evaluator, argsRaw string) (expr.Expr, error) {
	args, err := e.parseArgs(argsRaw)
	if err != nil {
		return nil, err
	}
	if len(args) < 1 {
		return expr.Empty{}, nil
	}
	name := strings.TrimSpace(args[0])
	if name == "" {
		return expr.Empty{}, nil
	}

	// Try to load from database if not already in registry
	if cs := corpusStore(e); cs != nil {
		exists, err := cs.CorpusExists(name)
		if err != nil {
			return nil, err
		}
		if exists && e.corpusRegistry.GetByName(name) == nil {
			// Load from DB
			c := &Corpus{
				name:       name,
				embeddings: make(map[string][]float32),
			}
			members, err := cs.GetCorpusMembers(name)
			if err != nil {
				return nil, err
			}
			c.members = members

			// Load embeddings
			embs, err := cs.GetEmbeddings(name)
			if err != nil {
				return nil, err
			}
			if embs != nil {
				c.embeddings = embs
			}

			// Load HNSW index
			indexData, err := cs.GetVectorIndex(name)
			if err != nil {
				return nil, err
			}
			if indexData != nil {
				g := hnsw.NewGraph[string]()
				if err := g.Import(bytes.NewReader(indexData)); err == nil {
					c.hnswGraph = g
					c.vecReady = true
				}
			}

			// Check if FTS table exists by trying a search
			c.ftsReady = ftsTableExists(cs, name)

			e.corpusRegistry.SetCorpus(name, c)
		} else if !exists {
			// Create in DB
			if err := cs.CreateCorpus(name); err != nil {
				return nil, err
			}
		}
	}

	handleID := e.corpusRegistry.GetOrCreate(name)
	return expr.Text{Value: handleID}, nil
}

func builtinAdd(e *Evaluator, argsRaw string) (expr.Expr, error) {
	args, err := e.parseArgs(argsRaw)
	if err != nil {
		return nil, err
	}
	if len(args) < 2 {
		return expr.Empty{}, nil
	}

	handleID := strings.TrimSpace(args[0])
	exprName := strings.TrimSpace(args[1])

	c := e.corpusRegistry.Get(handleID)
	if c == nil {
		return expr.Empty{}, nil
	}

	c.AddMember(exprName)

	if cs := corpusStore(e); cs != nil {
		if err := cs.AddCorpusMember(c.name, exprName); err != nil {
			return nil, err
		}
	}

	return expr.Empty{}, nil
}

func builtinIndex(e *Evaluator, argsRaw string) (expr.Expr, error) {
	args, err := e.parseArgs(argsRaw)
	if err != nil {
		return nil, err
	}
	if len(args) < 1 {
		return expr.Empty{}, nil
	}

	handleID := strings.TrimSpace(args[0])
	c := e.corpusRegistry.Get(handleID)
	if c == nil {
		return expr.Empty{}, nil
	}

	cs := corpusStore(e)
	if cs == nil {
		return expr.Empty{}, nil
	}

	if err := cs.CreateFTSTable(c.name); err != nil {
		return nil, err
	}

	for _, member := range c.members {
		val := e.namespace.Get(member)
		content := val.String()
		if err := cs.UpdateFTSContent(c.name, member, content); err != nil {
			return nil, err
		}
	}

	c.ftsReady = true
	return expr.Empty{}, nil
}

func builtinSearch(e *Evaluator, argsRaw string) (expr.Expr, error) {
	args, err := e.parseArgs(argsRaw)
	if err != nil {
		return nil, err
	}
	if len(args) < 2 {
		return expr.Empty{}, nil
	}

	handleID := strings.TrimSpace(args[0])
	query := strings.TrimSpace(args[1])

	c := e.corpusRegistry.Get(handleID)
	if c == nil || !c.ftsReady {
		return expr.Empty{}, nil
	}

	cs := corpusStore(e)
	if cs == nil {
		return expr.Empty{}, nil
	}

	limit := searchLimit(e)
	results, err := cs.SearchFTS(c.name, query, limit)
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return expr.Empty{}, nil
	}
	return expr.Text{Value: strings.Join(results, "\n")}, nil
}

func builtinEmbed(e *Evaluator, argsRaw string) (expr.Expr, error) {
	args, err := e.parseArgs(argsRaw)
	if err != nil {
		return nil, err
	}
	if len(args) < 1 {
		return expr.Empty{}, nil
	}

	handleID := strings.TrimSpace(args[0])
	c := e.corpusRegistry.Get(handleID)
	if c == nil {
		return expr.Empty{}, nil
	}

	ep, ok := e.provider.(provider.EmbeddingProvider)
	if !ok {
		return nil, fmt.Errorf("current provider does not support embeddings")
	}

	// Collect texts that need embedding
	var toEmbed []string
	var toEmbedNames []string
	for _, member := range c.members {
		if _, exists := c.embeddings[member]; exists {
			continue
		}
		val := e.namespace.Get(member)
		toEmbed = append(toEmbed, val.String())
		toEmbedNames = append(toEmbedNames, member)
	}

	if len(toEmbed) > 0 {
		vectors, err := ep.Embed(toEmbed)
		if err != nil {
			return nil, err
		}

		cs := corpusStore(e)
		for i, name := range toEmbedNames {
			if i < len(vectors) {
				c.embeddings[name] = vectors[i]
				if cs != nil {
					if err := cs.StoreEmbedding(c.name, name, vectors[i]); err != nil {
						return nil, err
					}
				}
			}
		}
	}

	// Build HNSW graph from all embeddings
	g := hnsw.NewGraph[string]()
	for name, vec := range c.embeddings {
		g.Add(hnsw.MakeNode(name, vec))
	}
	c.hnswGraph = g
	c.vecReady = true

	// Serialize and persist
	if cs := corpusStore(e); cs != nil {
		var buf bytes.Buffer
		if err := g.Export(&buf); err != nil {
			return nil, err
		}
		if err := cs.StoreVectorIndex(c.name, buf.Bytes()); err != nil {
			return nil, err
		}
	}

	return expr.Empty{}, nil
}

func builtinSimilar(e *Evaluator, argsRaw string) (expr.Expr, error) {
	args, err := e.parseArgs(argsRaw)
	if err != nil {
		return nil, err
	}
	if len(args) < 2 {
		return expr.Empty{}, nil
	}

	handleID := strings.TrimSpace(args[0])
	query := strings.TrimSpace(args[1])

	c := e.corpusRegistry.Get(handleID)
	if c == nil || !c.vecReady || c.hnswGraph == nil {
		return expr.Empty{}, nil
	}

	ep, ok := e.provider.(provider.EmbeddingProvider)
	if !ok {
		return nil, fmt.Errorf("current provider does not support embeddings")
	}

	// Embed the query
	vectors, err := ep.Embed([]string{query})
	if err != nil {
		return nil, err
	}
	if len(vectors) == 0 {
		return expr.Empty{}, nil
	}

	limit := searchLimit(e)
	results := c.hnswGraph.Search(vectors[0], limit)

	if len(results) == 0 {
		return expr.Empty{}, nil
	}

	var names []string
	for _, r := range results {
		names = append(names, r.Key)
	}
	return expr.Text{Value: strings.Join(names, "\n")}, nil
}

// corpusStore type-asserts the evaluator's store to CorpusStore.
func corpusStore(e *Evaluator) store.CorpusStore {
	if e.store == nil {
		return nil
	}
	cs, _ := e.store.(store.CorpusStore)
	return cs
}

// searchLimit returns the SEARCH_LIMIT setting as an int.
func searchLimit(e *Evaluator) int {
	s := e.GetSetting("SEARCH_LIMIT", "10")
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 {
		return 10
	}
	return n
}

// ftsTableExists checks if the FTS table exists for a corpus.
func ftsTableExists(cs store.CorpusStore, name string) bool {
	// Try a search; if the table doesn't exist, it will error
	_, err := cs.SearchFTS(name, "test", 1)
	return err == nil
}
