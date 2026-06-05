package db

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func testPool(t *testing.T) *pgxpool.Pool {
	url := os.Getenv("DATABASE_URL_TEST")
	if url == "" {
		t.Skip("DATABASE_URL_TEST not set")
	}
	p, err := Connect(context.Background(), url)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	return p
}

func TestMigrateCreatesChunksTable(t *testing.T) {
	p := testPool(t)
	defer p.Close()
	if err := Migrate(context.Background(), p, "../../migrations"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	var n int
	err := p.QueryRow(context.Background(),
		`SELECT count(*) FROM information_schema.tables WHERE table_name='knowledge_chunks'`).Scan(&n)
	if err != nil || n != 1 {
		t.Fatalf("expected knowledge_chunks table, n=%d err=%v", n, err)
	}
}
