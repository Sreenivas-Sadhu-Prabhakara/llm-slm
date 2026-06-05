// Command distill generates self-distillation training examples for the Taglish
// voice LoRA. For each question it retrieves (per mode audience), generates the
// gold answer with the FULL persona via LiteLLM, and writes a RawExample whose
// input uses the SHORT persona. Reuses the production retrieval/prompt/generation
// path so the distillation inputs match prod exactly. Output: training/data/raw.jsonl.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"log"
	"os"
	"strings"

	"github.com/apolaki/solar-assistant/internal/config"
	"github.com/apolaki/solar-assistant/internal/db"
	"github.com/apolaki/solar-assistant/internal/distill"
	"github.com/apolaki/solar-assistant/internal/embed"
	"github.com/apolaki/solar-assistant/internal/generator"
	"github.com/apolaki/solar-assistant/internal/prompt"
	"github.com/apolaki/solar-assistant/internal/retriever"
)

type question struct {
	Category string `json:"category"`
	Mode     string `json:"mode"`
	Question string `json:"question"`
}

func main() {
	in := flag.String("in", "training/questions.jsonl", "question bank jsonl")
	out := flag.String("out", "training/data/raw.jsonl", "raw examples jsonl")
	topK := flag.Int("k", 4, "retrieval depth")
	limit := flag.Int("limit", 0, "limit number of questions (0 = all; for smoke runs)")
	flag.Parse()

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
	gen := generator.New(cfg.LiteLLMBaseURL, cfg.LiteLLMAPIKey, cfg.GenModel)

	questions := readQuestions(*in)
	if *limit > 0 && *limit < len(questions) {
		questions = questions[:*limit]
	}

	if err := os.MkdirAll("training/data", 0o755); err != nil {
		log.Fatal(err)
	}
	f, err := os.Create(*out)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	enc := json.NewEncoder(f)

	for i, q := range questions {
		m := prompt.ModeByName(q.Mode)
		var chunks []retriever.Chunk
		// nosource → deliberately empty context so the model must decline.
		if q.Category != "nosource" {
			chunks, err = r.Search(ctx, q.Question, nil, m.Audience, *topK)
			if err != nil {
				log.Fatalf("retrieve %q: %v", q.Question, err)
			}
		}
		sysFull, user := prompt.AssembleFor(m, q.Question, chunks)
		var gold strings.Builder
		if err := gen.Stream(ctx, sysFull, user, func(t string) { gold.WriteString(t) }); err != nil {
			log.Fatalf("generate %q: %v", q.Question, err)
		}
		ex := distill.Build(m, q.Category, q.Question, chunks, strings.TrimSpace(gold.String()))
		if err := enc.Encode(ex); err != nil {
			log.Fatal(err)
		}
		log.Printf("[%d/%d] %s (%s) -> %d chars", i+1, len(questions), q.Category, m.Name, gold.Len())
	}
	log.Printf("wrote %d examples -> %s", len(questions), *out)
}

func readQuestions(path string) []question {
	f, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	var out []question
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var q question
		if err := json.Unmarshal([]byte(line), &q); err != nil {
			log.Fatalf("bad question line: %v", err)
		}
		out = append(out, q)
	}
	if err := sc.Err(); err != nil {
		log.Fatal(err)
	}
	return out
}
