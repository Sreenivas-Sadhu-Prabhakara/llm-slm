package retriever

import (
	"context"
	"fmt"
	"strings"

	"github.com/apolaki/solar-assistant/internal/embed"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Chunk is a single result from a vector search.
type Chunk struct {
	ChunkID   string
	DocID     string
	Title     string
	SourceURI *string
	Content   string
	Score     float64 // cosine distance (lower = closer)
}

// Retriever performs tenant-scoped HNSW vector similarity search.
type Retriever struct {
	pool *pgxpool.Pool
	emb  embed.Client
}

// New returns a Retriever backed by pool and emb.
func New(pool *pgxpool.Pool, emb embed.Client) *Retriever {
	return &Retriever{pool: pool, emb: emb}
}

// Search embeds the query and returns the top-k nearest chunks, scoped to the
// tenant (or shared/global where tenant_id IS NULL) and audience.
func (r *Retriever) Search(ctx context.Context, query string, tenantID *string, audience string, k int) ([]Chunk, error) {
	if k <= 0 {
		return nil, fmt.Errorf("retriever: k must be > 0, got %d", k)
	}
	vec, err := r.emb.Embed(ctx, query)
	if err != nil {
		return nil, err
	}
	if len(vec) == 0 {
		return nil, fmt.Errorf("retriever: embed returned empty vector")
	}
	rows, err := r.pool.Query(ctx, `
		SELECT c.id, d.id, d.title, d.source_uri, c.content,
		       c.embedding <=> $1 AS score
		FROM knowledge_chunks c
		JOIN knowledge_documents d ON d.id = c.document_id
		WHERE (c.tenant_id = $2 OR c.tenant_id IS NULL)
		  AND d.audience = $3
		ORDER BY c.embedding <=> $1
		LIMIT $4`,
		pgvec(vec), tenantID, audience, k,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Chunk
	for rows.Next() {
		var c Chunk
		if err := rows.Scan(&c.ChunkID, &c.DocID, &c.Title, &c.SourceURI, &c.Content, &c.Score); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// pgvec renders a float slice as a pgvector text literal [v0,v1,...]. It is
// bound as a query parameter; pgvector implicitly casts text → vector. Phase 0
// approach — revisit with pgvector-go typed binding if the implicit cast is dropped.
func pgvec(v []float64) string {
	b := strings.Builder{}
	b.WriteByte('[')
	for i, f := range v {
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, "%g", f)
	}
	b.WriteByte(']')
	return b.String()
}
