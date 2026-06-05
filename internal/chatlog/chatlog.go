// Package chatlog persists each assistant turn and its feedback to Postgres,
// feeding the eval + future fine-tune flywheel (PRD §5 step 8). All queries are
// parameterized and tenant-scoped (tenant_id NULL = shared).
package chatlog

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Logger writes conversations, messages, and feedback.
type Logger struct {
	pool *pgxpool.Pool
}

// New returns a Logger backed by pool.
func New(pool *pgxpool.Pool) *Logger { return &Logger{pool: pool} }

// StartConversation creates a conversation row and returns its id.
func (l *Logger) StartConversation(ctx context.Context, tenantID, userID *string, mode, channel string) (string, error) {
	var id string
	err := l.pool.QueryRow(ctx,
		`INSERT INTO conversations (tenant_id, user_id, mode, channel)
		 VALUES ($1, $2, $3, $4) RETURNING id`,
		tenantID, userID, mode, channel).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("start conversation: %w", err)
	}
	return id, nil
}

// LogTurn records the user question and the assistant answer as two messages and
// returns their ids. chunkIDs are the retrieved source chunk ids for the answer.
func (l *Logger) LogTurn(ctx context.Context, convID string, tenantID *string,
	question, answer, model string, chunkIDs []string, latencyMs int) (userMsgID, asstMsgID string, err error) {
	err = l.pool.QueryRow(ctx,
		`INSERT INTO messages (conversation_id, tenant_id, role, content)
		 VALUES ($1, $2, 'user', $3) RETURNING id`,
		convID, tenantID, question).Scan(&userMsgID)
	if err != nil {
		return "", "", fmt.Errorf("log user message: %w", err)
	}
	err = l.pool.QueryRow(ctx,
		`INSERT INTO messages (conversation_id, tenant_id, role, content,
		                       retrieved_chunk_ids, model, latency_ms)
		 VALUES ($1, $2, 'assistant', $3, $4::uuid[], $5, $6) RETURNING id`,
		convID, tenantID, answer, chunkIDs, model, latencyMs).Scan(&asstMsgID)
	if err != nil {
		return "", "", fmt.Errorf("log assistant message: %w", err)
	}
	return userMsgID, asstMsgID, nil
}

// RecordFeedback stores a 👍/👎 ("up"/"down") plus optional solved flag and note
// for an assistant message.
func (l *Logger) RecordFeedback(ctx context.Context, messageID string, tenantID, userID *string,
	rating string, solved *bool, note string) error {
	_, err := l.pool.Exec(ctx,
		`INSERT INTO feedback (message_id, tenant_id, user_id, rating, solved, note)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		messageID, tenantID, userID, rating, solved, note)
	if err != nil {
		return fmt.Errorf("record feedback: %w", err)
	}
	return nil
}
