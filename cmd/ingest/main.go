package main

import (
	"bufio"
	"context"
	"encoding/json"
	"log"
	"os"

	"github.com/apolaki/solar-assistant/internal/config"
	"github.com/apolaki/solar-assistant/internal/db"
	"github.com/apolaki/solar-assistant/internal/embed"
	"github.com/apolaki/solar-assistant/internal/ingest"
)

func main() {
	path := "data/seed/corpus.jsonl"
	if len(os.Args) > 1 {
		path = os.Args[1]
	}
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()
	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	if err := db.Migrate(ctx, pool, "migrations"); err != nil {
		log.Fatal(err)
	}
	emb := embed.New(cfg.LiteLLMBaseURL, cfg.LiteLLMAPIKey, cfg.EmbedModel)

	f, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	n := 0
	for sc.Scan() {
		var d ingest.Doc
		if err := json.Unmarshal(sc.Bytes(), &d); err != nil {
			log.Fatal(err)
		}
		if err := ingest.Upsert(ctx, pool, emb, d); err != nil {
			log.Fatal(err)
		}
		n++
	}
	log.Printf("ingested %d documents", n)
}
