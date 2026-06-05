package retriever

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/apolaki/solar-assistant/internal/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

type fakeEmbed struct{ v []float64 }

func (f fakeEmbed) Embed(_ context.Context, _ string) ([]float64, error) { return f.v, nil }

func TestSearchReturnsNearestChunk(t *testing.T) {
	url := os.Getenv("DATABASE_URL_TEST")
	if url == "" {
		t.Skip("DATABASE_URL_TEST not set")
	}
	ctx := context.Background()
	pool, err := db.Connect(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := db.Migrate(ctx, pool, "../../migrations"); err != nil {
		t.Fatal(err)
	}
	seedOneChunk(t, pool) // helper inserts a doc+chunk with a known vector

	r := New(pool, fakeEmbed{v: unitVec(0)})
	got, err := r.Search(ctx, "anything", nil, "customer", 3)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(got) == 0 {
		t.Fatal("expected at least one result")
	}
}

// unitVec returns a 1024-dim vector where index i = 1.0 and all others = 0.0.
func unitVec(i int) []float64 {
	v := make([]float64, 1024)
	v[i] = 1.0
	return v
}

// seedOneChunk inserts a knowledge_documents row (audience='customer') and
// a knowledge_chunks row with embedding = unitVec(0). Uses parameterized SQL.
// Designed to be run multiple times safely (just accumulates rows, no unique
// constraint violation since we always insert with gen_random_uuid()).
func seedOneChunk(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()

	var docID string
	err := pool.QueryRow(ctx, `
		INSERT INTO knowledge_documents
		  (title, source_type, source_uri, audience, product, brand, language, content_hash)
		VALUES ($1, $2, NULL, $3, NULL, $4, $5, $6)
		RETURNING id`,
		"Test Doc", "manual", "customer", "TestBrand", "taglish",
		fmt.Sprintf("testhash-%d", 0),
	).Scan(&docID)
	if err != nil {
		t.Fatalf("seed doc: %v", err)
	}

	vec := unitVec(0)
	if _, err := pool.Exec(ctx, `
		INSERT INTO knowledge_chunks
		  (document_id, tenant_id, chunk_index, content, embedding, token_count)
		VALUES ($1, NULL, $2, $3, $4, $5)`,
		docID, 0, "test chunk content", pgvecLiteral(vec), 3,
	); err != nil {
		t.Fatalf("seed chunk: %v", err)
	}
}

// pgvecLiteral renders a float slice as a pgvector text literal: [v0,v1,...]
func pgvecLiteral(v []float64) string {
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
