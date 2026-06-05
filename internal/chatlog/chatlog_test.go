package chatlog

import (
	"context"
	"os"
	"testing"

	"github.com/apolaki/solar-assistant/internal/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

func testPool(t *testing.T) *pgxpool.Pool {
	url := os.Getenv("DATABASE_URL_TEST")
	if url == "" {
		t.Skip("DATABASE_URL_TEST not set")
	}
	p, err := db.Connect(context.Background(), url)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	if err := db.Migrate(context.Background(), p, "../../migrations"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return p
}

func TestLogTurnAndFeedbackPersist(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()
	l := New(pool)

	convID, err := l.StartConversation(ctx, nil, nil, "customer", "web")
	if err != nil {
		t.Fatalf("start conversation: %v", err)
	}
	if convID == "" {
		t.Fatal("empty conversation id")
	}

	chunkIDs := []string{"11111111-1111-1111-1111-111111111111"}
	userMsgID, asstMsgID, err := l.LogTurn(ctx, convID, nil,
		"magkano ang solar?", "Ayon sa sources...", "sea-lion-9b", chunkIDs, 1234)
	if err != nil {
		t.Fatalf("log turn: %v", err)
	}
	if userMsgID == "" || asstMsgID == "" {
		t.Fatal("expected both message ids")
	}

	solved := false
	if err := l.RecordFeedback(ctx, asstMsgID, nil, nil, "down", &solved, "mali sagot"); err != nil {
		t.Fatalf("record feedback: %v", err)
	}

	var msgs int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM messages WHERE conversation_id=$1`, convID).Scan(&msgs); err != nil {
		t.Fatalf("count messages: %v", err)
	}
	if msgs != 2 {
		t.Fatalf("expected 2 messages, got %d", msgs)
	}

	var fb int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM feedback WHERE message_id=$1`, asstMsgID).Scan(&fb); err != nil {
		t.Fatalf("count feedback: %v", err)
	}
	if fb != 1 {
		t.Fatalf("expected 1 feedback row, got %d", fb)
	}
}
