// Command ask is the Phase 0 deliverable: a CLI that answers a solar question
// in grounded Taglish by wiring retrieve -> assemble -> stream -> print sources.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/apolaki/solar-assistant/internal/config"
	"github.com/apolaki/solar-assistant/internal/db"
	"github.com/apolaki/solar-assistant/internal/embed"
	"github.com/apolaki/solar-assistant/internal/generator"
	"github.com/apolaki/solar-assistant/internal/prompt"
	"github.com/apolaki/solar-assistant/internal/retriever"
)

func main() {
	question := strings.Join(os.Args[1:], " ")
	if strings.TrimSpace(question) == "" {
		log.Fatal("usage: ask <your solar question>")
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

	emb := embed.New(cfg.LiteLLMBaseURL, cfg.LiteLLMAPIKey, cfg.EmbedModel)
	r := retriever.New(pool, emb)
	chunks, err := r.Search(ctx, question, nil, "customer", 4)
	if err != nil {
		log.Fatal(err)
	}

	sys, user := prompt.Assemble(question, chunks)
	gen := generator.New(cfg.LiteLLMBaseURL, cfg.LiteLLMAPIKey, cfg.GenModel)

	fmt.Println("--- Sagot (Taglish) ---")
	if err := gen.Stream(ctx, sys, user, func(tok string) { fmt.Print(tok) }); err != nil {
		log.Fatal(err)
	}
	fmt.Println("\n\n--- Sources ---")
	if len(chunks) == 0 {
		fmt.Println("(walang nahanap — escalate to a specialist)")
	}
	for i, c := range chunks {
		fmt.Printf("[%d] %s\n", i+1, c.Title)
	}
}
