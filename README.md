# Apolaki Solar Assistant (`llm-slm`)

A self-hosted, **solar-only** support assistant (the "solar brain") for the Apolaki solar
marketplace. It advocates for domestic solar adoption — ROI / energy independence in **PHP**,
spoken in **Taglish** — grounded in Apolaki's own docs via RAG, at **≈ ₱0 per query**.

Runs on a Mac Studio behind **LiteLLM**, using **SEA-LION 9B** for generation and **BGE-M3**
for embeddings, with **pgvector + HNSW** for retrieval.

## Architecture

```
client → POST /assistant/chat (SSE)
  → auth (stub JWT / dev headers)
  → topic gate (solar-only, LLM-free)         internal/topicgate
  → retrieve (pgvector HNSW, per-mode audience) internal/retriever + internal/embed
  → assemble persona + sources prompt          internal/prompt
  → generate (LiteLLM → SEA-LION, streaming)   internal/generator
  → grounded-or-escalate safety                internal/safety
  → log turn + 👍/👎 feedback (flywheel)        internal/chatlog
```

- **Modes (one brain, many personas):** `customer` / `buyer` / `installer` — each retrieves
  from its own document audience and uses a tailored Taglish persona.
- **3-layer guardrails:** topic gate → grounded-only retrieval → safety/escalation.

## Phases

- **Phase 0 — Foundation:** Go service + pgvector/HNSW RAG + BGE-M3 + SEA-LION via LiteLLM,
  synthetic Taglish seed corpus, `ask` CLI, health server.
- **Phase 1 — Customer MVP:** streaming `/assistant/chat`, feedback, stub-JWT, browser test page.
- **Phase 2 — Modes + Taglish voice LoRA:** buyer/installer modes; a **prompt-distillation
  LoRA** (self-distilled SEA-LION 9B) that reproduces the full persona from a *short* system
  prompt, served MLX-native as the LiteLLM primary (Ollama GGUF fallback). See
  `AI/docs/PRDs/` and `AI/docs/tasks/`.

## Layout

```
cmd/         ask · ingest · server · distill (training-data generator)
internal/    config db migrate embed ingest retriever prompt generator
             topicgate safety personalizer chatlog httpapi distill
data/        synthetic Taglish seed corpus generator
training/    Taglish voice LoRA pipeline (Python + MLX): questions → curate → train → eval gate
AI/          PRDs, task plans, master_plan.md (session memory)
```

## Run (local dev)

```bash
# infra: Postgres (pgvector), LiteLLM :4000, BGE-M3 :8100  (see AI/master_plan.md)
cp .env.example .env
go run ./cmd/server -migrate          # apply migrations
set -a; source .env; set +a
python3 data/generate_seed.py && go run ./cmd/ingest   # seed + ingest
go run ./cmd/server                   # serves chat UI at http://localhost:8090/
go test ./...                         # full suite
```

> Engineering norms, model stack, and serving details live in `CLAUDE.md` and `AI/`.
