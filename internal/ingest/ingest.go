package ingest

import (
	"context"
	"fmt"
	"strings"

	"github.com/apolaki/solar-assistant/internal/embed"
	"github.com/jackc/pgx/v5/pgxpool"
)

func join(s []string, sep string) string { return strings.Join(s, sep) }

// Chunk splits text into ~size-word chunks with overlap words of context.
func Chunk(text string, size, overlap int) []string {
	words := strings.Fields(text)
	if len(words) <= size {
		return []string{strings.TrimSpace(text)}
	}
	var chunks []string
	for start := 0; start < len(words); start += size - overlap {
		end := start + size
		if end > len(words) {
			end = len(words)
		}
		chunks = append(chunks, join(words[start:end], " "))
		if end == len(words) {
			break
		}
	}
	return chunks
}

// Doc is one input document from the seed JSONL.
type Doc struct {
	Title       string  `json:"title"`
	SourceType  string  `json:"source_type"`
	Audience    string  `json:"audience"`
	Product     *string `json:"product"`
	Brand       string  `json:"brand"`
	Language    string  `json:"language"`
	Content     string  `json:"content"`
	ContentHash string  `json:"content_hash"`
}

// Upsert ingests one document: insert doc row, chunk, embed, insert chunks.
// tenantID nil => shared/global. Parameterized SQL only.
func Upsert(ctx context.Context, pool *pgxpool.Pool, emb embed.Client, d Doc) error {
	var docID string
	err := pool.QueryRow(ctx, `
		INSERT INTO knowledge_documents
		  (title, source_type, source_uri, audience, product, brand, language, content_hash)
		VALUES ($1,$2,NULL,$3,$4,$5,$6,$7)
		RETURNING id`,
		d.Title, d.SourceType, d.Audience, d.Product, d.Brand, d.Language, d.ContentHash,
	).Scan(&docID)
	if err != nil {
		return err
	}
	for i, ch := range Chunk(d.Content, 120, 24) {
		vec, err := emb.Embed(ctx, ch)
		if err != nil {
			return err
		}
		if _, err := pool.Exec(ctx, `
			INSERT INTO knowledge_chunks
			  (document_id, tenant_id, chunk_index, content, embedding, token_count)
			VALUES ($1, NULL, $2, $3, $4, $5)`,
			docID, i, ch, pgvec(vec), len(strings.Fields(ch)),
		); err != nil {
			return err
		}
	}
	return nil
}

// pgvec renders a float slice as a pgvector literal: [0.1,0.2,...]
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
