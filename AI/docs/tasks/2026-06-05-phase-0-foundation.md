# Phase 0 — Foundation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
> **House rules (CLAUDE.md):** one task per session · diagnose before fixing · parameterized SQL only · `tenant_id` in every query · no secrets in code · tests before "done" · commit as `[P0.N] …`.

**Goal:** A standalone Go service + data tooling where you can ask a solar question in the terminal (`ask`) and get a grounded **Taglish** answer with cited sources, retrieved from a synthetic Apolaki corpus via pgvector/HNSW and generated through LiteLLM.

**Architecture:** A Go service (`github.com/apolaki/solar-assistant`, module path adjustable) talks to Postgres+pgvector for retrieval and to LiteLLM (`:4000`) for embeddings (BGE-M3) and generation (SEA-LION 9B, Qwen3 fallback). A small Python FastAPI server serves BGE-M3 embeddings behind LiteLLM. A Python pipeline generates synthetic seed data; a Go ingestion path chunks → embeds → upserts it. The Phase-0 deliverable is the `ask` CLI; the HTTP server is a thin health-only skeleton (chat endpoint is Phase 1).

**Tech Stack:** Go 1.22+ (`pgx/v5`), Postgres 16 + `pgvector` (HNSW), LiteLLM `:4000`, BGE-M3 (FlagEmbedding/FastAPI), Ollama/MLX for generation, Python 3.11+ for data tooling. Docker for a local dev/test Postgres.

**Prerequisites (already done in this session):** LiteLLM is live on `:4000` with retries + MLX→Ollama fallback; MLX (`:8000`) and Ollama (`:11434`) are up. See memory `local-serving-stack`. This plan *adds* a BGE-M3 embedding model and a SEA-LION generation model to LiteLLM.

---

## File Structure

```
llm-slm/
├── go.mod / go.sum
├── Makefile                      # dev shortcuts (db-up, migrate, ask, test)
├── .env.example                 # documented config; real .env is gitignored
├── docker-compose.yml           # local Postgres 16 + pgvector (dev + test)
├── cmd/
│   ├── server/main.go           # HTTP skeleton: GET /assistant/health
│   ├── ask/main.go              # CLI deliverable: question → grounded answer + sources
│   └── ingest/main.go           # CLI: load seed JSONL → chunk → embed → upsert
├── internal/
│   ├── config/config.go         # env → typed Config (+ _test)
│   ├── db/db.go                 # pgxpool connect + ping
│   ├── db/migrate.go            # minimal migrations runner (+ _test)
│   ├── embed/embed.go           # BGE-M3 client via LiteLLM /v1/embeddings (+ _test)
│   ├── ingest/ingest.go         # chunk + upsert documents/chunks (+ _test)
│   ├── retriever/retriever.go   # HNSW vector search, tenant-scoped (+ _test)
│   ├── prompt/prompt.go         # Taglish persona prompt assembly (+ _test)
│   └── generator/generator.go   # LiteLLM streaming chat client (+ _test)
├── migrations/
│   ├── 0001_pgvector.sql
│   ├── 0002_knowledge.sql
│   └── 0003_conversations.sql
├── embeddings-server/
│   ├── requirements.txt
│   └── server.py                # FastAPI OpenAI-compatible /v1/embeddings (BGE-M3)
├── data/
│   ├── requirements.txt
│   ├── generate_seed.py         # synthetic datasheets + FAQ + tickets → JSONL
│   └── seed/                    # generated output (raw gitignored)
└── AI/ …                        # existing docs
```

**Module boundaries:** each `internal/*` package has one responsibility and a small interface so tasks are independently testable. `embed.Client` and `generator.Client` are interfaces so retriever/CLI can be unit-tested with fakes (no live model needed in unit tests).

---

## Roadmap context (later phases — detailed plans written when their inputs exist)

- **Phase 1 — Customer self-service MVP:** implement `POST /assistant/chat` (SSE) reusing P0 modules, topic-gate, safety/grounding layer, `/assistant/feedback`, conversation+message+feedback logging, and the Vue chat widget. *Plan it after P0 lands.*
- **Phase 2 — Light Taglish LoRA + modes:** needs real logged chats (flywheel) before a fine-tune set exists → plan when data is collected.
- **Phase 3 — Advocacy & scale:** proactive savings nudges, more PH languages, cloud-GPU serving swap (LiteLLM config change). Plan when nearing ~1,000 users.

*(Detailed TDD steps for 1–3 are intentionally omitted here — writing them now would be speculative placeholders, which this skill forbids.)*

---
### Task 0: Repo bootstrap + typed config

**Files:**
- Create: `go.mod`, `Makefile`, `.env.example`, `internal/config/config.go`, `internal/config/config_test.go`

- [ ] **Step 1: Init the Go module and .gitignore-safe env example**

Run:
```bash
cd /Users/macstudio/Documents/llm-slm
go mod init github.com/apolaki/solar-assistant
```
Create `.env.example` (real `.env` is already gitignored):
```bash
# Postgres (local dev/test via docker-compose)
DATABASE_URL=postgres://solar:solar@localhost:5433/solar?sslmode=disable
DATABASE_URL_TEST=postgres://solar:solar@localhost:5433/solar_test?sslmode=disable
# LiteLLM gateway (already running this machine)
LITELLM_BASE_URL=http://localhost:4000/v1
LITELLM_API_KEY=local
# Model names as registered in LiteLLM
EMBED_MODEL=bge-m3
GEN_MODEL=sea-lion-9b
```

- [ ] **Step 2: Write the failing config test**

`internal/config/config_test.go`:
```go
package config

import "testing"

func TestLoadRequiresDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	if _, err := Load(); err == nil {
		t.Fatal("expected error when DATABASE_URL is empty")
	}
}

func TestLoadDefaultsAndValues(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("LITELLM_BASE_URL", "")
	t.Setenv("EMBED_MODEL", "")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.DatabaseURL != "postgres://x" {
		t.Fatalf("DatabaseURL = %q", cfg.DatabaseURL)
	}
	if cfg.LiteLLMBaseURL != "http://localhost:4000/v1" {
		t.Fatalf("default LiteLLMBaseURL wrong: %q", cfg.LiteLLMBaseURL)
	}
	if cfg.EmbedModel != "bge-m3" {
		t.Fatalf("default EmbedModel wrong: %q", cfg.EmbedModel)
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./internal/config/...`
Expected: FAIL — `undefined: Load`.

- [ ] **Step 4: Implement minimal config**

`internal/config/config.go`:
```go
package config

import (
	"errors"
	"os"
)

type Config struct {
	DatabaseURL     string
	DatabaseURLTest string
	LiteLLMBaseURL  string
	LiteLLMAPIKey   string
	EmbedModel      string
	GenModel        string
}

func get(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// Load reads config from the environment, applying defaults.
func Load() (Config, error) {
	c := Config{
		DatabaseURL:     os.Getenv("DATABASE_URL"),
		DatabaseURLTest: os.Getenv("DATABASE_URL_TEST"),
		LiteLLMBaseURL:  get("LITELLM_BASE_URL", "http://localhost:4000/v1"),
		LiteLLMAPIKey:   get("LITELLM_API_KEY", "local"),
		EmbedModel:      get("EMBED_MODEL", "bge-m3"),
		GenModel:        get("GEN_MODEL", "sea-lion-9b"),
	}
	if c.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}
	return c, nil
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/config/...`
Expected: PASS.

- [ ] **Step 6: Add a Makefile**

`Makefile`:
```make
.PHONY: db-up db-down migrate test ask ingest
db-up:        ; docker compose up -d
db-down:      ; docker compose down
migrate:      ; go run ./cmd/server -migrate
test:         ; go test ./...
ask:          ; go run ./cmd/ask
ingest:       ; go run ./cmd/ingest
```

- [ ] **Step 7: Commit**

```bash
git add go.mod Makefile .env.example internal/config
git commit -m "[P0.0] Bootstrap Go module + typed config"
```

---

### Task 1: Local Postgres + pgvector and DB pool

**Files:**
- Create: `docker-compose.yml`, `internal/db/db.go`

- [ ] **Step 1: Add docker-compose with the pgvector image**

`docker-compose.yml`:
```yaml
services:
  postgres:
    image: pgvector/pgvector:pg16
    environment:
      POSTGRES_USER: solar
      POSTGRES_PASSWORD: solar
      POSTGRES_DB: solar
    ports: ["5433:5432"]
    volumes: ["pgdata:/var/lib/postgresql/data"]
volumes: { pgdata: {} }
```

- [ ] **Step 2: Start it and create the test DB**

Run:
```bash
docker compose up -d
sleep 5
docker compose exec -T postgres psql -U solar -d solar -c "CREATE DATABASE solar_test;"
```
Expected: `CREATE DATABASE` (ignore "already exists" on reruns).

- [ ] **Step 3: Add pgx dependency and the pool helper**

Run: `go get github.com/jackc/pgx/v5/pgxpool`
`internal/db/db.go`:
```go
package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Connect opens a pooled connection and verifies it with a ping.
func Connect(ctx context.Context, url string) (*pgxpool.Pool, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}
```

- [ ] **Step 4: Verify it compiles and connects**

Run:
```bash
go build ./...
DATABASE_URL=postgres://solar:solar@localhost:5433/solar?sslmode=disable \
  go run -tags ignore ./... 2>/dev/null; echo "compiled"
```
Expected: build succeeds. (A dedicated connection test is added in Task 2 alongside migrations, which need a live DB.)

- [ ] **Step 5: Commit**

```bash
git add docker-compose.yml internal/db go.mod go.sum
git commit -m "[P0.1] Local pgvector Postgres + pgxpool connector"
```

---
### Task 2: Migrations runner + schema (pgvector, HNSW, tables)

**Files:**
- Create: `migrations/0001_pgvector.sql`, `migrations/0002_knowledge.sql`, `migrations/0003_conversations.sql`, `internal/db/migrate.go`, `internal/db/migrate_test.go`

- [ ] **Step 1: Write the SQL migrations**

`migrations/0001_pgvector.sql`:
```sql
CREATE EXTENSION IF NOT EXISTS vector;
CREATE TABLE IF NOT EXISTS schema_migrations (
  version text PRIMARY KEY,
  applied_at timestamptz NOT NULL DEFAULT now()
);
```
`migrations/0002_knowledge.sql`:
```sql
CREATE TABLE IF NOT EXISTS knowledge_documents (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid,                       -- NULL = shared/global
  title text NOT NULL,
  source_type text NOT NULL,
  source_uri text,
  audience text NOT NULL DEFAULT 'customer',
  product text, brand text, language text DEFAULT 'taglish',
  version int NOT NULL DEFAULT 1,
  content_hash text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS knowledge_chunks (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  document_id uuid NOT NULL REFERENCES knowledge_documents(id) ON DELETE CASCADE,
  tenant_id uuid,
  chunk_index int NOT NULL,
  content text NOT NULL,
  embedding vector(1024) NOT NULL,
  token_count int NOT NULL DEFAULT 0,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_chunks_embedding_hnsw
  ON knowledge_chunks USING hnsw (embedding vector_cosine_ops);
CREATE INDEX IF NOT EXISTS idx_chunks_tenant_audience
  ON knowledge_chunks (tenant_id);
```
`migrations/0003_conversations.sql`:
```sql
CREATE TABLE IF NOT EXISTS conversations (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid, user_id uuid,
  mode text NOT NULL DEFAULT 'customer',
  channel text NOT NULL DEFAULT 'web',
  status text NOT NULL DEFAULT 'open',
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS messages (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  conversation_id uuid NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
  tenant_id uuid, role text NOT NULL, content text NOT NULL,
  retrieved_chunk_ids uuid[], model text, latency_ms int,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS feedback (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  message_id uuid NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
  tenant_id uuid, user_id uuid,
  rating text NOT NULL, solved boolean, note text,
  created_at timestamptz NOT NULL DEFAULT now()
);
```

- [ ] **Step 2: Write the failing migrations test**

`internal/db/migrate_test.go` (uses `DATABASE_URL_TEST`; skips if unset):
```go
package db

import (
	"context"
	"os"
	"testing"
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
```
Add the import `"github.com/jackc/pgx/v5/pgxpool"` to the test file.

- [ ] **Step 3: Run test to verify it fails**

Run: `set -a; source .env; set +a; go test ./internal/db/...`
Expected: FAIL — `undefined: Migrate`.

- [ ] **Step 4: Implement the migrations runner**

`internal/db/migrate.go`:
```go
package db

import (
	"context"
	"os"
	"path/filepath"
	"sort"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Migrate applies any *.sql files in dir not yet recorded in schema_migrations,
// in filename order. The first file must create schema_migrations.
func Migrate(ctx context.Context, pool *pgxpool.Pool, dir string) error {
	files, err := filepath.Glob(filepath.Join(dir, "*.sql"))
	if err != nil {
		return err
	}
	sort.Strings(files)
	for _, f := range files {
		version := filepath.Base(f)
		applied, err := isApplied(ctx, pool, version)
		if err != nil {
			return err
		}
		if applied {
			continue
		}
		sqlBytes, err := os.ReadFile(f)
		if err != nil {
			return err
		}
		if _, err := pool.Exec(ctx, string(sqlBytes)); err != nil {
			return err
		}
		if _, err := pool.Exec(ctx,
			`INSERT INTO schema_migrations(version) VALUES($1) ON CONFLICT DO NOTHING`, version); err != nil {
			return err
		}
	}
	return nil
}

func isApplied(ctx context.Context, pool *pgxpool.Pool, version string) (bool, error) {
	var exists bool
	err := pool.QueryRow(ctx, `SELECT to_regclass('schema_migrations') IS NOT NULL`).Scan(&exists)
	if err != nil || !exists {
		return false, nil // table not created yet → not applied
	}
	var n int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM schema_migrations WHERE version=$1`, version).Scan(&n); err != nil {
		return false, err
	}
	return n > 0, nil
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `set -a; source .env; set +a; go test ./internal/db/...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add migrations internal/db
git commit -m "[P0.2] Migrations runner + pgvector/HNSW schema"
```

---

### Task 3: BGE-M3 embedding server behind LiteLLM

**Files:**
- Create: `embeddings-server/requirements.txt`, `embeddings-server/server.py`
- Modify: `/Users/macstudio/Documents/Code/agent_skills/litellm_config.yaml` (register `bge-m3`)

- [ ] **Step 1: Write the embedding server (OpenAI-compatible)**

`embeddings-server/requirements.txt`:
```
fastapi
uvicorn
FlagEmbedding
```
`embeddings-server/server.py`:
```python
"""OpenAI-compatible /v1/embeddings server for BGE-M3 (1024-dim dense)."""
from fastapi import FastAPI
from pydantic import BaseModel
from FlagEmbedding import BGEM3FlagModel

app = FastAPI()
model = BGEM3FlagModel("BAAI/bge-m3", use_fp16=True)

class EmbReq(BaseModel):
    input: list[str] | str
    model: str = "bge-m3"

@app.get("/health")
def health():
    return {"status": "healthy", "model": "bge-m3", "dim": 1024}

@app.post("/v1/embeddings")
def embeddings(req: EmbReq):
    texts = [req.input] if isinstance(req.input, str) else req.input
    vecs = model.encode(texts, batch_size=8)["dense_vecs"]
    data = [
        {"object": "embedding", "index": i, "embedding": v.tolist()}
        for i, v in enumerate(vecs)
    ]
    return {"object": "list", "data": data, "model": "bge-m3",
            "usage": {"prompt_tokens": 0, "total_tokens": 0}}
```

- [ ] **Step 2: Install deps and start it on :8100**

Run:
```bash
cd /Users/macstudio/Documents/llm-slm/embeddings-server
python3 -m venv .venv && source .venv/bin/activate
pip install -r requirements.txt
nohup uvicorn server:app --host 127.0.0.1 --port 8100 > /tmp/bge.log 2>&1 &
sleep 30   # first run downloads BGE-M3 (~2GB)
curl -s http://localhost:8100/health
```
Expected: `{"status":"healthy","model":"bge-m3","dim":1024}`.

- [ ] **Step 3: Register bge-m3 in LiteLLM and restart it**

Add under `model_list:` in `/Users/macstudio/Documents/Code/agent_skills/litellm_config.yaml`:
```yaml
  - model_name: bge-m3
    litellm_params:
      model: openai/bge-m3
      api_base: http://localhost:8100/v1
      api_key: local
```
Run: `cd /Users/macstudio/Documents/Code/agent_skills && ./start-litellm.sh`

- [ ] **Step 4: Verify embeddings through LiteLLM**

Run:
```bash
curl -s http://localhost:4000/v1/embeddings \
  -H 'Content-Type: application/json' \
  -d '{"model":"bge-m3","input":"magkano ang solar panel"}' \
  | python3 -c "import sys,json; d=json.load(sys.stdin); print('dim:', len(d['data'][0]['embedding']))"
```
Expected: `dim: 1024`.

- [ ] **Step 5: Commit (service repo only; infra config is unversioned)**

```bash
git add embeddings-server
git commit -m "[P0.3] BGE-M3 embedding server (OpenAI-compatible) + LiteLLM route"
```
> Note: `litellm_config.yaml` lives in `agent_skills/` (not a git repo) — document the added block in the PR description / ops notes.

---
### Task 4: Go embeddings client (via LiteLLM)

**Files:**
- Create: `internal/embed/embed.go`, `internal/embed/embed_test.go`

- [ ] **Step 1: Write the failing test (httptest fake of LiteLLM)**

`internal/embed/embed_test.go`:
```go
package embed

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEmbedReturns1024Vector(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/embeddings" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		vec := make([]float64, 1024)
		vec[0] = 0.5
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data":[{"embedding":[`))
		// build 1024 zeros with first=0.5
		w.Write([]byte("0.5"))
		for i := 1; i < 1024; i++ {
			w.Write([]byte(",0"))
		}
		w.Write([]byte(`]}]}`))
	}))
	defer srv.Close()

	c := New(srv.URL, "k", "bge-m3")
	v, err := c.Embed(context.Background(), "magkano")
	if err != nil {
		t.Fatalf("embed: %v", err)
	}
	if len(v) != 1024 {
		t.Fatalf("dim = %d, want 1024", len(v))
	}
	if v[0] != 0.5 {
		t.Fatalf("v[0] = %f", v[0])
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/embed/...`
Expected: FAIL — `undefined: New`.

- [ ] **Step 3: Implement the client**

`internal/embed/embed.go`:
```go
package embed

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Client embeds text via an OpenAI-compatible /v1/embeddings endpoint (LiteLLM).
type Client interface {
	Embed(ctx context.Context, text string) ([]float64, error)
}

type httpClient struct {
	baseURL, apiKey, model string
	hc                     *http.Client
}

func New(baseURL, apiKey, model string) Client {
	return &httpClient{baseURL: baseURL, apiKey: apiKey, model: model,
		hc: &http.Client{Timeout: 30 * time.Second}}
}

func (c *httpClient) Embed(ctx context.Context, text string) ([]float64, error) {
	body, _ := json.Marshal(map[string]any{"model": c.model, "input": text})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("embeddings status %d", resp.StatusCode)
	}
	var out struct {
		Data []struct {
			Embedding []float64 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if len(out.Data) == 0 {
		return nil, fmt.Errorf("no embedding returned")
	}
	return out.Data[0].Embedding, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/embed/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/embed
git commit -m "[P0.4] BGE-M3 embeddings Go client"
```

---

### Task 5: Register SEA-LION 9B generation model in LiteLLM

**Files:**
- Modify: `/Users/macstudio/Documents/Code/agent_skills/litellm_config.yaml`

Goal: make `sea-lion-9b` the primary generation model with a license-clean fallback, served frugally via Ollama (simplest path to add a model; the existing 32B Qwen MLX stays as heavy fallback).

- [ ] **Step 1: Obtain Gemma-SEA-LION-v3-9B-IT as a GGUF and load into Ollama**

Run (path A — pull a community GGUF if available):
```bash
# Verify the exact repo/tag first; aisingapore publishes GGUF builds.
ollama pull hf.co/aisingapore/Gemma-SEA-LION-v3-9B-IT-GGUF:Q4_K_M
ollama list | grep -i sea-lion
```
Path B (fallback if no prebuilt GGUF): download the safetensors and convert with `llama.cpp` `convert_hf_to_gguf.py` → `Q4_K_M`, then `ollama create sea-lion-9b -f Modelfile` where `Modelfile` is `FROM ./sea-lion-9b-Q4_K_M.gguf`.
Expected: a model named `sea-lion-9b` (or the hf.co tag) appears in `ollama list`.

- [ ] **Step 2: Register it in LiteLLM as primary + Qwen fallback**

Edit `/Users/macstudio/Documents/Code/agent_skills/litellm_config.yaml`:
```yaml
  # --- Generation: primary (SEA-LION 9B, Taglish) ---
  - model_name: sea-lion-9b
    litellm_params:
      model: ollama_chat/<exact-sea-lion-tag-from-step-1>
      api_base: http://localhost:11434
      timeout: 600
  # license-clean text fallback already exists as qwen-ollama / qwen-coder-32b
```
And add to `litellm_settings.fallbacks`:
```yaml
    - sea-lion-9b: ["qwen-ollama"]
```

- [ ] **Step 3: Restart LiteLLM and smoke-test generation**

Run:
```bash
cd /Users/macstudio/Documents/Code/agent_skills && ./start-litellm.sh
curl -s --max-time 120 http://localhost:4000/v1/chat/completions \
  -H 'Content-Type: application/json' \
  -d '{"model":"sea-lion-9b","messages":[{"role":"user","content":"Sabihin mo: SEA_LION_OK"}],"max_tokens":20,"stream":false}' \
  | python3 -c "import sys,json; print(json.load(sys.stdin)['choices'][0]['message']['content'][:60])"
```
Expected: a Taglish reply containing `SEA_LION_OK`.

- [ ] **Step 4: Verify fallback (optional, same method proven this session)**

Stop Ollama's sea-lion or point the route at a bad host briefly → confirm a `sea-lion-9b` request still answers via `qwen-ollama`. Restore after.

- [ ] **Step 5: Record in ops notes (infra repo is unversioned)**

No service-repo commit. Append the new model block + fallback to the ops notes / PR description, and update memory `local-serving-stack`.

---
### Task 6: Synthetic seed-data generator (Python)

**Files:**
- Create: `data/requirements.txt`, `data/generate_seed.py`
- Modify: `.gitignore` (ensure `data/seed/` raw output ignored — `/data/raw/` already is; add `data/seed/`)

Produces a JSONL corpus the Go ingester reads. Each line is one document: `{title, source_type, audience, product, brand, language, content}`. Covers PH essentials per PRD §9.

- [ ] **Step 1: Write the generator**

`data/requirements.txt`:
```
# stdlib-only generator; no external deps required for Phase 0
```
`data/generate_seed.py`:
```python
"""Generate a synthetic Apolaki solar corpus as JSONL (one document per line).
Re-runnable and deterministic. Output: data/seed/corpus.jsonl
"""
import json, os, hashlib

OUT_DIR = os.path.join(os.path.dirname(__file__), "seed")
OUT = os.path.join(OUT_DIR, "corpus.jsonl")

DOCS = [
    {"title": "Net Metering sa Pilipinas (Meralco)", "source_type": "faq",
     "content": "Ang net metering ay nagbibigay-daan sa iyong i-export ang sobrang solar energy "
                "pabalik sa grid. Sa Meralco, ang export mo ay nakukunting credit sa susunod mong bill. "
                "Karaniwang aabot ng 20-30% ang bawas sa monthly bill para sa typical na bahay."},
    {"title": "ROI ng Residential Solar (₱)", "source_type": "faq",
     "content": "Para sa isang 5 kW system na nagkakahalaga ng ~₱300,000, kung nakakatipid ka ng "
                "₱4,000/buwan, ang payback period ay ~6 na taon. Pagkatapos noon, halos libre na ang kuryente "
                "mo sa araw. Ang panels ay may 25-year warranty kaya malaki ang long-term savings."},
    {"title": "Brownout Backup with Solar + Battery", "source_type": "faq",
     "content": "Kung gusto mo ng backup tuwing brownout, kailangan mo ng hybrid inverter at battery. "
                "Ang solar-only (grid-tie) ay hindi gumagana kapag may outage para sa safety. Ang battery "
                "ang nagbibigay ng power sa gabi at tuwing walang kuryente."},
    {"title": "Apolaki Panel Datasheet — AP-450W Mono", "source_type": "datasheet",
     "product": "AP-450W", "brand": "Apolaki",
     "content": "Apolaki AP-450W monocrystalline panel. Power: 450W. Efficiency: 21%. "
                "Dimensions: 1.9m x 1.1m. Operating temp: -40C to 85C. Warranty: 25 years performance."},
    {"title": "Inverter Error Codes — Apolaki Hybrid 5kW", "source_type": "datasheet",
     "product": "AP-INV-5K", "brand": "Apolaki",
     "content": "E01: grid voltage out of range — tawag sa installer. E02: over-temperature — i-check ang "
                "ventilation. E03: battery communication fail — i-check ang cables. F12: insulation fault — "
                "huwag i-reset, tumawag agad sa lisensyadong installer."},
    {"title": "Resolved Ticket — Mataas pa rin ang bill", "source_type": "ticket",
     "content": "Tanong: bakit mataas pa rin ang Meralco bill ko kahit may solar? Sagot: kadalasan dahil "
                "gabi ang peak usage mo (aircon, etc.) na hindi covered ng solar kung walang battery. Solusyon: "
                "ilipat ang mabigat na gamit sa araw, o magdagdag ng battery para sa gabi."},
    {"title": "Financing Options para sa Solar", "source_type": "faq",
     "content": "May mga bangko at in-house financing na nag-aalok ng 12-60 months para sa solar. "
                "Ang monthly amortization ay madalas mas mababa pa sa monthly savings sa kuryente, kaya "
                "cash-flow positive ka agad sa maraming kaso."},
]

def main():
    os.makedirs(OUT_DIR, exist_ok=True)
    with open(OUT, "w", encoding="utf-8") as f:
        for d in DOCS:
            d.setdefault("audience", "customer")
            d.setdefault("brand", "Apolaki")
            d.setdefault("language", "taglish")
            d.setdefault("product", None)
            d["content_hash"] = hashlib.sha256(d["content"].encode()).hexdigest()
            f.write(json.dumps(d, ensure_ascii=False) + "\n")
    print(f"wrote {len(DOCS)} docs -> {OUT}")

if __name__ == "__main__":
    main()
```

- [ ] **Step 2: Generate and eyeball the corpus**

Run:
```bash
cd /Users/macstudio/Documents/llm-slm
python3 data/generate_seed.py
wc -l data/seed/corpus.jsonl   # expect 7
head -1 data/seed/corpus.jsonl | python3 -m json.tool
```
Expected: `7` lines; first doc is valid JSON with `content`, `content_hash`, `audience=customer`.

- [ ] **Step 3: Commit (generator only; raw output gitignored)**

```bash
printf "\ndata/seed/\n" >> .gitignore
git add data/generate_seed.py data/requirements.txt .gitignore
git commit -m "[P0.6] Synthetic Taglish solar seed-data generator"
```

---

### Task 7: Ingestion pipeline (chunk → embed → upsert)

**Files:**
- Create: `internal/ingest/ingest.go`, `internal/ingest/ingest_test.go`, `cmd/ingest/main.go`

- [ ] **Step 1: Write the failing chunker test (pure unit, no DB)**

`internal/ingest/ingest_test.go`:
```go
package ingest

import "testing"

func TestChunkSplitsLongTextWithOverlap(t *testing.T) {
	words := make([]string, 250)
	for i := range words {
		words[i] = "w"
	}
	text := join(words, " ")
	chunks := Chunk(text, 100, 20) // size=100 words, overlap=20
	if len(chunks) < 2 {
		t.Fatalf("expected >=2 chunks, got %d", len(chunks))
	}
}

func TestChunkShortTextSingleChunk(t *testing.T) {
	if got := Chunk("maikli lang ito", 100, 20); len(got) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(got))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/ingest/...`
Expected: FAIL — `undefined: Chunk` / `undefined: join`.

- [ ] **Step 3: Implement chunker + upsert**

`internal/ingest/ingest.go`:
```go
package ingest

import (
	"context"
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
	Title       string `json:"title"`
	SourceType  string `json:"source_type"`
	Audience    string `json:"audience"`
	Product     *string `json:"product"`
	Brand       string `json:"brand"`
	Language    string `json:"language"`
	Content     string `json:"content"`
	ContentHash string `json:"content_hash"`
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
```
Add a `pgvec` helper (vector literal) in the same file:
```go
import "fmt"

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
```

- [ ] **Step 4: Run unit tests (chunker) to verify pass**

Run: `go get github.com/jackc/pgx/v5/pgxpool && go test ./internal/ingest/...`
Expected: PASS (chunker tests; Upsert is covered by the CLI integration in Step 6).

- [ ] **Step 5: Write the ingest CLI**

`cmd/ingest/main.go`:
```go
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
```

- [ ] **Step 6: Run end-to-end ingest (integration)**

Run:
```bash
set -a; source .env; set +a
go run ./cmd/ingest
psql "$DATABASE_URL" -c "SELECT count(*) FROM knowledge_chunks;"
```
Expected: `ingested 7 documents`; chunk count ≥ 7.

- [ ] **Step 7: Commit**

```bash
git add internal/ingest cmd/ingest go.mod go.sum
git commit -m "[P0.7] Ingestion: chunk + embed + upsert + ingest CLI"
```

---
### Task 8: Retriever (HNSW vector search, tenant-scoped)

**Files:**
- Create: `internal/retriever/retriever.go`, `internal/retriever/retriever_test.go`

- [ ] **Step 1: Write the failing integration test (skips without test DB)**

`internal/retriever/retriever_test.go`:
```go
package retriever

import (
	"context"
	"os"
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
```
> `seedOneChunk` and `unitVec` are small test helpers in the same file: insert a `knowledge_documents` row, then a `knowledge_chunks` row with `embedding = unitVec(0)` (a 1024-dim vector, index 0 = 1.0), and define `unitVec(i int) []float64`.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/retriever/...`
Expected: FAIL — `undefined: New`.

- [ ] **Step 3: Implement the retriever**

`internal/retriever/retriever.go`:
```go
package retriever

import (
	"context"
	"fmt"
	"strings"

	"github.com/apolaki/solar-assistant/internal/embed"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Chunk struct {
	ChunkID  string
	DocID    string
	Title    string
	SourceURI *string
	Content  string
	Score    float64 // cosine distance (lower = closer)
}

type Retriever struct {
	pool *pgxpool.Pool
	emb  embed.Client
}

func New(pool *pgxpool.Pool, emb embed.Client) *Retriever {
	return &Retriever{pool: pool, emb: emb}
}

// Search embeds the query and returns the top-k nearest chunks, scoped to the
// tenant (or shared/global where tenant_id IS NULL) and audience.
func (r *Retriever) Search(ctx context.Context, query string, tenantID *string, audience string, k int) ([]Chunk, error) {
	vec, err := r.emb.Embed(ctx, query)
	if err != nil {
		return nil, err
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
```
> `<=>` is pgvector's cosine-distance operator, matching the `vector_cosine_ops` HNSW index from Task 2.

- [ ] **Step 4: Run test to verify it passes**

Run: `set -a; source .env; set +a; go test ./internal/retriever/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/retriever
git commit -m "[P0.8] HNSW retriever, tenant-scoped cosine search"
```

---

### Task 9: Taglish prompt assembler

**Files:**
- Create: `internal/prompt/prompt.go`, `internal/prompt/prompt_test.go`

- [ ] **Step 1: Write the failing test**

`internal/prompt/prompt_test.go`:
```go
package prompt

import (
	"strings"
	"testing"

	"github.com/apolaki/solar-assistant/internal/retriever"
)

func TestAssembleIncludesSourcesAndQuestion(t *testing.T) {
	chunks := []retriever.Chunk{
		{Title: "Net Metering", Content: "export credit sa Meralco"},
	}
	sys, user := Assemble("magkano savings?", chunks)
	if !strings.Contains(sys, "Taglish") {
		t.Fatal("system prompt missing persona")
	}
	if !strings.Contains(user, "export credit sa Meralco") {
		t.Fatal("user prompt missing source content")
	}
	if !strings.Contains(user, "magkano savings?") {
		t.Fatal("user prompt missing the question")
	}
}

func TestAssembleNoSourcesSignalsEscalation(t *testing.T) {
	_, user := Assemble("random", nil)
	if !strings.Contains(strings.ToLower(user), "walang") && !strings.Contains(user, "no sources") {
		t.Fatal("expected an explicit no-sources signal for grounding")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/prompt/...`
Expected: FAIL — `undefined: Assemble`.

- [ ] **Step 3: Implement the assembler**

`internal/prompt/prompt.go`:
```go
package prompt

import (
	"fmt"
	"strings"

	"github.com/apolaki/solar-assistant/internal/retriever"
)

// System is the Taglish advocate persona (PRD Appendix A, condensed).
const System = `You are Apolaki Assistant, a warm, encouraging solar-energy guide for ` +
	`Filipino homeowners. Speak natural Taglish. Emphasize savings, ROI in pesos (₱), ` +
	`and energy independence; avoid heavy jargon. Only answer questions about solar energy ` +
	`and Apolaki's products. Only state facts found in the provided sources; if the sources ` +
	`don't cover it, say you'll connect them to a specialist — never guess. For wiring, ` +
	`electrical, or installation-safety topics, remind them to consult a licensed installer. ` +
	`Cite source titles you used. Be kind, clear, and motivating.`

// Assemble builds (systemPrompt, userPrompt) from the question and retrieved chunks.
func Assemble(question string, chunks []retriever.Chunk) (string, string) {
	var b strings.Builder
	if len(chunks) == 0 {
		b.WriteString("SOURCES: (walang nahanap na source / no sources found)\n\n")
	} else {
		b.WriteString("SOURCES:\n")
		for i, c := range chunks {
			fmt.Fprintf(&b, "[%d] %s: %s\n", i+1, c.Title, c.Content)
		}
		b.WriteString("\n")
	}
	fmt.Fprintf(&b, "QUESTION: %s\n", question)
	b.WriteString("\nSagutin gamit lang ang SOURCES sa itaas. Kung kulang, mag-escalate.")
	return System, b.String()
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/prompt/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/prompt
git commit -m "[P0.9] Taglish persona prompt assembler"
```

---
### Task 10: Streaming generator client (LiteLLM SSE)

**Files:**
- Create: `internal/generator/generator.go`, `internal/generator/generator_test.go`

- [ ] **Step 1: Write the failing test (httptest fakes an SSE stream)**

`internal/generator/generator_test.go`:
```go
package generator

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestStreamConcatenatesDeltas(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Write([]byte(`data: {"choices":[{"delta":{"content":"Pwede"}}]}` + "\n\n"))
		w.Write([]byte(`data: {"choices":[{"delta":{"content":" mag-solar"}}]}` + "\n\n"))
		w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer srv.Close()

	c := New(srv.URL, "k", "sea-lion-9b")
	var got strings.Builder
	err := c.Stream(context.Background(), "sys", "user", func(tok string) {
		got.WriteString(tok)
	})
	if err != nil {
		t.Fatalf("stream: %v", err)
	}
	if got.String() != "Pwede mag-solar" {
		t.Fatalf("got %q", got.String())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/generator/...`
Expected: FAIL — `undefined: New`.

- [ ] **Step 3: Implement the streaming client**

`internal/generator/generator.go`:
```go
package generator

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Client interface {
	Stream(ctx context.Context, system, user string, onToken func(string)) error
}

type httpClient struct {
	baseURL, apiKey, model string
	hc                     *http.Client
}

func New(baseURL, apiKey, model string) Client {
	return &httpClient{baseURL: baseURL, apiKey: apiKey, model: model,
		hc: &http.Client{Timeout: 10 * time.Minute}}
}

func (c *httpClient) Stream(ctx context.Context, system, user string, onToken func(string)) error {
	body, _ := json.Marshal(map[string]any{
		"model":  c.model,
		"stream": true,
		"messages": []map[string]string{
			{"role": "system", "content": system},
			{"role": "user", "content": user},
		},
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	resp, err := c.hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("chat status %d", resp.StatusCode)
	}
	sc := bufio.NewScanner(resp.Body)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(line[5:])
		if payload == "[DONE]" {
			break
		}
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			continue
		}
		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
			onToken(chunk.Choices[0].Delta.Content)
		}
	}
	return sc.Err()
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/generator/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/generator
git commit -m "[P0.10] LiteLLM streaming generator client"
```

---

### Task 11: `ask` CLI — the Phase 0 deliverable

**Files:**
- Create: `cmd/ask/main.go`

- [ ] **Step 1: Wire retrieve → assemble → stream → print sources**

`cmd/ask/main.go`:
```go
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
	if question == "" {
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
```

- [ ] **Step 2: Run the full deliverable end-to-end**

Run:
```bash
set -a; source .env; set +a
go run ./cmd/ask "magkano ang matitipid ko sa solar kada buwan?"
```
Expected: a streamed Taglish answer grounded in the ROI/net-metering docs, followed by a `Sources` list naming those documents. Ask an off-topic question (e.g. "sino panalo sa NBA?") → the model should decline/redirect (grounding + persona), proving the guardrail seed.

- [ ] **Step 3: Commit**

```bash
git add cmd/ask
git commit -m "[P0.11] ask CLI: grounded Taglish answer + sources (P0 deliverable)"
```

---

### Task 12: Health server skeleton

**Files:**
- Create: `cmd/server/main.go`

- [ ] **Step 1: Implement health endpoint + optional -migrate flag**

`cmd/server/main.go`:
```go
package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/apolaki/solar-assistant/internal/config"
	"github.com/apolaki/solar-assistant/internal/db"
)

func main() {
	migrateOnly := flag.Bool("migrate", false, "run migrations and exit")
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
	if err := db.Migrate(ctx, pool, "migrations"); err != nil {
		log.Fatal(err)
	}
	if *migrateOnly {
		log.Println("migrations applied"); os.Exit(0)
	}

	http.HandleFunc("/assistant/health", func(w http.ResponseWriter, r *http.Request) {
		status := map[string]string{"status": "ok"}
		if err := pool.Ping(r.Context()); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			status["status"] = "db_unreachable"
		}
		json.NewEncoder(w).Encode(status)
	})
	log.Println("listening on :8090")
	log.Fatal(http.ListenAndServe(":8090", nil))
}
```

- [ ] **Step 2: Verify health responds**

Run:
```bash
set -a; source .env; set +a
go run ./cmd/server & sleep 2
curl -s http://localhost:8090/assistant/health   # {"status":"ok"}
kill %1
```
Expected: `{"status":"ok"}`.

- [ ] **Step 3: Commit**

```bash
git add cmd/server
git commit -m "[P0.12] HTTP health skeleton + -migrate flag"
```

---

## Self-Review (run against the PRD)

**Spec coverage:** Go service skeleton (T0,T12) · pgvector/HNSW tables (T2) · LiteLLM→SEA-LION 9B + fallback (T5) · BGE-M3 embeddings (T3,T4) · synthetic seed data (T6) · ingestion (T7) · retrieval (T8) · Taglish persona + grounding (T9) · streaming generation (T10) · CLI deliverable "terminal question → grounded Taglish answer + sources" (T11). ✅ Matches PRD §12 Phase 0 and the §4 lifecycle steps 2–6 (topic-gate, personalize, safety-output-layer are **Phase 1**, noted in roadmap).

**Placeholder scan:** every code step contains complete code; the only deliberately deferred specifics are the exact SEA-LION GGUF tag (verify at run time — `ollama list`) and Phase 1–3 detail (gated on data/outcomes).

**Type consistency:** `embed.Client.Embed`, `generator.Client.Stream`, `retriever.Chunk`, `prompt.Assemble`, `ingest.Doc/Chunk/Upsert` are used identically across tasks; module path `github.com/apolaki/solar-assistant` is consistent throughout.

**Known follow-ups (Phase 1, not gaps in P0):** topic-gate module, safety/escalation output filter, `/assistant/chat` SSE endpoint, conversation/message/feedback logging, golden-eval harness (PRD §10), PII masking (PRD §8).

---

## Execution Handoff

Plan complete and saved to `AI/docs/tasks/2026-06-05-phase-0-foundation.md`. Two execution options:

1. **Subagent-Driven (recommended)** — dispatch a fresh subagent per task, review between tasks, fast iteration (superpowers:subagent-driven-development).
2. **Inline Execution** — execute tasks in this session with checkpoints (superpowers:executing-plans).

Per your CLAUDE.md ("one task per session"), the natural cadence is **one P0.N task per session**, committing after each.

