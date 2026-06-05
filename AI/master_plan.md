# Apolaki Solar Assistant — Master Plan

> Session-to-session memory. **Start each session** by reading this + `AGENTS.md` (when added) + the PRD.

## Project
Custom, **solar-only** support assistant (the "solar brain") for the **Apolaki** solar marketplace (web + mobile).
**North star:** advocacy for **domestic solar adoption** — ROI / energy independence in **PHP**, spoken in **Taglish**.
Self-hosted on the **Mac Studio** behind **LiteLLM**, **RAG** over Apolaki docs. **≈ ₱0 per query.**

## Status — 2026-06-05 (Phase 0 COMPLETE)
- ✅ Design / PRD complete and **approved** → `AI/docs/PRDs/2026-06-05-apolaki-solar-assistant-design.md`
- ✅ **Local serving stack hardened** (fixed timeouts/peer-resets): LiteLLM `:4000` live with retries + MLX→Ollama fallback; MLX real streaming. See memory `local-serving-stack`.
- ✅ **Phase 0 implementation plan written** → `AI/docs/tasks/2026-06-05-phase-0-foundation.md` (13 bite-sized TDD tasks P0.0–P0.12). Phases 1–3 kept as roadmap (detail gated on P0 outcomes + real data).
- ✅ **P0.0 done** — Go module `github.com/apolaki/solar-assistant`, `.env.example`, Makefile, `internal/config` (TDD, passing). Commit `7175b3d`.
- ✅ **P0.1 done** — `docker-compose.yml` (pgvector pg16 :5433), `internal/db.Connect` (pgx/v5), verified vs PG 16.14 + pgvector 0.8.2. Commit `c3d1d03`. **Env note:** Docker = **Colima** (run `colima start`) + standalone **`docker-compose`** (no `docker compose` plugin).
- ✅ **P0.2 done** — `internal/db.Migrate` (idempotent, `schema_migrations`-tracked) + schema: knowledge_documents/chunks (vector(1024)+HNSW), conversations/messages/feedback. Integration test green. Commit `23e54a4`.
- ✅ **P0.3 done** — `embeddings-server/` (FastAPI, OpenAI-compatible `/v1/embeddings`, BGE-M3 1024-dim dense) on `:8100`; registered `bge-m3` in LiteLLM (`:4000`), restarted, verified `dim: 1024` end-to-end. Commit `4a3e3ec`. **Notes:** model cached at `~/.cache/huggingface/hub/models--BAAI--bge-m3` (~4.3GB, downloaded once); venv had no console scripts so start via `.venv/bin/python -m uvicorn server:app --host 127.0.0.1 --port 8100`; `litellm_config.yaml` (in `agent_skills/`, unversioned) gained a `bge-m3` route block.
- ✅ **P0.4 done** — `internal/embed` Go client (`New`/`Embed`) hits OpenAI-compatible `/embeddings`; TDD via httptest fake, returns 1024-dim `[]float64`. `go vet` clean; full suite (config/db/embed) green. Commit `97d7fbd`.
- ✅ **P0.5 done** — registered `sea-lion-9b` generation route in LiteLLM (`:4000`): `ollama_chat/hf.co/aisingapore/Gemma-SEA-LION-v3-9B-IT-GGUF:Q4_K_M` (5.8GB GGUF, pulled into Ollama), 600s timeout, fallback `sea-lion-9b → qwen-ollama`. Restarted LiteLLM, smoke-test green (Taglish `SEA_LION_OK`, `model: sea-lion-9b`). **Infra-only** (no service-repo code); `litellm_config.yaml` in `agent_skills/` (unversioned). Memory `local-serving-stack` updated.
- ✅ **P0.6 done** — `data/generate_seed.py` (stdlib-only, deterministic) + `data/requirements.txt`; writes `data/seed/corpus.jsonl` (7 Taglish docs: 4 FAQ + 2 datasheet + 1 ticket), each defaulting `audience=customer`/`brand=Apolaki`/`language=taglish`/`product` + sha256 `content_hash`. Verified 7 valid JSON lines + identical output on re-run; `data/seed/` added to `.gitignore` (raw output unversioned). Commit `48e3542`.
- ✅ **P0.7 done** — `internal/ingest` (`Chunk` word-based size/overlap; `Upsert` inserts doc row → chunks → BGE-M3 embeddings via LiteLLM, parameterized SQL, `tenant_id` NULL = shared, pgvector text-literal via `pgvec`) + `cmd/ingest` (reads JSONL, migrates, ingests). TDD: chunker unit tests green. Integration verified end-to-end: `ingested 7 documents` → 7 docs / 7 chunks / embedding dim 1024 in pgvector. Full suite + `go vet` clean. Commit `942c2fe`. **Note:** Upsert is insert-only (no content_hash dedup yet) — re-running duplicates; fine for Phase 0.
- ✅ **P0.8 done** — `internal/retriever` (`New`/`Search`): embeds the question via BGE-M3, runs tenant-scoped top-k cosine (`<=>`) over `knowledge_chunks` joined to `knowledge_documents`, returns `Chunk{ChunkID,DocID,Title,SourceURI,Content,Score}`. Input guards + parameterized SQL; `tenant_id` NULL = shared. TDD green, review addressed (input guards, cast comment, stronger test). Commits `399cdb8`, `1abb250`.
- ✅ **P0.9 done** — `internal/prompt` (`Assemble(question, chunks) -> (system, user)`): `System` const holds the condensed Taglish advocate persona (grounded-only, solar-only, cite sources, escalate on safety/wiring); user prompt lists numbered `SOURCES` + `QUESTION` with an explicit no-sources signal (`walang nahanap`) for grounding. TDD: tests green, `go vet` clean. Commit `d00c4fc`.
- ✅ **P0.10 done** — `internal/generator` (`New(baseURL,key,model)` + `Stream(ctx, system, user, onToken)`): streaming POST to OpenAI-compatible `/chat/completions`, invokes `onToken` per content delta until `[DONE]`; 1MB scanner buffer for large SSE frames, skips keep-alive/non-JSON frames, 10-min timeout for slow local GGUF behind MLX→Ollama fallback. Mirrors `internal/embed`. TDD: httptest fake SSE green, `go vet`/build clean. Commit `f78780c`.
- ✅ **P0.11 done** — `cmd/ask` (the Phase 0 deliverable): wires `config.Load → db.Connect → retriever.Search` (top-4, customer audience, shared tenant) `→ prompt.Assemble → generator.Stream` (token-by-token to stdout) `→` print cited source titles. **Verified end-to-end vs live infra + 7 seed docs:** "magkano matitipid… kada buwan?" → grounded Taglish answer citing the ROI doc (~₱4,000/mo on a 5 kW system); "sino panalo sa NBA finals?" → declines + redirects (solar-only guardrail holds). Commit `273c789`.
- ✅ **P0.12 done** — `cmd/server`: connects pool, runs idempotent migrations on boot, serves `GET /assistant/health` (`pool.Ping` → `{"status":"ok"}` | 503 `db_unreachable`) on `:8090`; `-migrate` flag applies migrations and exits (deploy/CI). Verified live: health `{"status":"ok"}`; `-migrate` → "migrations applied", exit 0. Commit `19ef1c5`.
- ✅ **Phase 0 — Foundation COMPLETE** (P0.0–P0.12): Go service + pgvector/HNSW RAG + BGE-M3 embeddings + SEA-LION generation via LiteLLM + synthetic Taglish seed + `ask` CLI test harness + health server. Full suite green, `go vet` clean. **First end-to-end grounded Taglish answer with citations + working solar-only guardrail.**
- ⏳ Phase 1 — Customer self-service MVP (Vue widget, guardrails, logging + feedback) — **next; needs a Phase 1 plan written before coding**
- 🔶 Phase 2 — Light Taglish LoRA fine-tune + buyer/installer modes — **started**
  - ✅ **P2.1 done** — `internal/prompt/modes.go`: `Mode{Name,Audience,System}` + `Customer`/`Buyer`/`Installer` personas + `ModeByName` (defaults Customer). `prompt.Assemble` now delegates to new `AssembleFor(mode,…)`. `cmd/server`'s `/assistant/chat` reads `mode` from the request body and threads it through retrieval (`mode.Audience`), persona (`AssembleFor`), and turn logging (`StartConversation(…, mode, …)`). Dropped now-dead `Deps.Audience`. TDD green, full suite + `go vet` clean. **Backend only** — see follow-ups.
  - ✅ **P2.2 done** — seed corpus extended in `data/generate_seed.py`: +3 buyer docs (sizing guide, TCO/financing, warranty checklist) +3 installer docs (AP-450W mounting spec, AP-INV-5K commissioning, PEC safety/code reminders), all Taglish + Apolaki-branded. Now 13 docs (7 customer / 3 buyer / 3 installer), deterministic re-run verified; re-ingested into pgvector (truncate + `go run ./cmd/ingest` → 13 docs / 13 chunks, audiences confirmed). Buyer/installer modes now retrieve real chunks.
  - ✅ **P2.3 done** — `internal/httpapi/index.html` test page gained a customer/buyer/installer `<select id="mode">`; chat request body now sends `mode`. `//go:embed` picks it up automatically. Build/vet/httpapi tests green.
  - ✅ **P2.4 done** — topic-gate fix found during live smoke: installer technical jargon/product codes (torque, clamp, MPPT, voltage, AP-450W, AP-INV-5K…) were wrongly redirected as off-topic. Added installer/buyer technical vocabulary to `internal/topicgate` keywords (TDD: failing test → fix). Off-topic cases still rejected.
  - ✅ **Live e2e verified** — server on :8090; each mode retrieves strictly its own audience (installer→3 installer datasheets, buyer→3 buyer FAQs, customer→customer FAQs); `conversations.mode` persisted per turn (installer/buyer/customer rows confirmed); installer Q "anong torque…AP-450W?" → grounded "16-20 Nm" citing the installer mounting spec.
  - ✅ **P2-LoRA done — Taglish voice LoRA (prompt distillation), SHIPPED.** Plan: `AI/docs/tasks/2026-06-05-phase-2-taglish-lora.md`; spec: `AI/docs/PRDs/2026-06-05-phase-2-taglish-lora-design.md`. Self-distilled SEA-LION 9B so the tuned model reproduces the full persona from a SHORT system prompt (frees 16K context).
    - **Pipeline:** `internal/prompt` short personas + `AssembleForShort`; `internal/distill` + `cmd/distill` (reuses prod retrieval/prompt/generation → 78 self-distilled examples); `training/` Python+MLX (`make_questions.py` 78-q bank, `curate.py` kept 64/78 → train 47 / valid 8 / test 9 stratified, `lora_config.yaml`+`train.sh`, `eval.py` gate). Trained via `mlx_lm.lora` (400 iters, rank 16, train loss 0.21→0.027; val rose after iter 200 = mild overfit, iter-200 ckpt kept as fallback), fused → `training/fused/sea-lion-taglish` (17GB bf16).
    - **Eval GATE = SHIP:** candidate (tuned+short) vs baseline (base+full) on golden set: quality 1.0 = 1.0, grounding 1.0 = 1.0, nosource-decline 100%, safety-escalate 100%, answer-rate 0.83 (within guard). Prompt distillation preserved all guardrail behaviors.
    - **Served:** `mlx_lm.server :8001` → LiteLLM primary `sea-lion-9b` (Ollama GGUF demoted to `sea-lion-9b-gguf` fallback, then qwen-ollama); `USE_SHORT_PROMPT=true`. **Live e2e:** installer→"16-20 Nm" (datasheet cite), buyer→"₱300,000" (TCO cite), off-topic→redirect. Full Go suite + `go vet` + 12 `training/` pytest green.
    - **Rollback:** `USE_SHORT_PROMPT=false` + revert LiteLLM `sea-lion-9b` route to GGUF primary + restart LiteLLM (config-gated, no data loss).
  - ⏳ **Follow-ups:** retrain on REAL logged chats once the flywheel accumulates (pipeline is built to re-run); optional two-tier routing; larger question bank for a richer golden eval.
- ⬜ Phase 3 — Advocacy features + scale (cloud GPU once past ~1,000 users)

## Locked Decisions (see PRD §3)
- **Architecture:** standalone **Go** service (own repo), RAG via **pgvector + HNSW** in Apolaki's Postgres.
- **Models:** **Gemma-SEA-LION-v3-9B-IT** (primary), **Qwen3-7B** (fallback), **BGE-M3** embeddings — served via **LiteLLM**.
- **Phase 1 = homeowners**, **Taglish**; **synthetic seed data now**, real tickets later (flywheel).
- **Hosting:** Mac Studio = production **until ~1,000 users**; expected **< 10k calls/day**.
- **Guardrails:** 3-layer solar-only (topic gate → grounded-only → safety/escalate).

## Next Session
- **Phase 0 is complete** — the whole P0.0–P0.12 backlog is done and verified end-to-end. The next backlog item is **Phase 1 (Customer self-service MVP)**, which per the ai-wf loop should start with a **brainstorm + Phase 1 implementation plan** (`AI/docs/tasks/…-phase-1-*.md`) before any code — Phase 1 detail was deliberately gated on Phase 0 outcomes + real data. Likely scope: Vue chat widget, HTTP `/assistant/ask` endpoint (wrap the `ask` path behind the server), 3-layer guardrails hardening, conversation/message logging + thumbs feedback.
- Re-ingest if DB was reset: `set -a; source .env; set +a; python3 data/generate_seed.py && go run ./cmd/ingest` (insert-only — truncate `knowledge_chunks`/`knowledge_documents` first to avoid duplicates).
- Confirm infra up first: `colima start && docker-compose up -d` (Postgres), `curl :4000/health/liveliness`, `:8000/health`, `:11434/api/tags`, and the **embeddings server** `:8100/health` (start: `cd embeddings-server && nohup .venv/bin/python -m uvicorn server:app --host 127.0.0.1 --port 8100 > /tmp/bge.log 2>&1 &`). None auto-start after reboot.
- Run Go tests that touch the DB with env loaded: `set -a; source .env; set +a; go test ./...`.

## Task Log
| Date | Task | Status |
|------|------|--------|
| 2026-06-05 | Brainstorm + design / PRD | ✅ Done |
| 2026-06-05 | git init + scaffold (PRD, master_plan, .gitignore) | ✅ Done |
| 2026-06-05 | Fix local-serving timeouts/peer-resets (LiteLLM proxy + MLX real streaming + fallback) | ✅ Done |
| 2026-06-05 | Phase 0 implementation plan (P0.0–P0.12) | ✅ Done |
| 2026-06-05 | P0.0 — Go module + typed config (TDD) | ✅ Done |
| 2026-06-05 | P0.1 — pgvector Postgres + db pool | ✅ Done |
| 2026-06-05 | P0.2 — migrations runner + HNSW schema | ✅ Done |
| 2026-06-05 | P0.3 — BGE-M3 embedding server + LiteLLM route | ✅ Done |
| 2026-06-05 | P0.4 — BGE-M3 embeddings Go client (internal/embed) | ✅ Done |
| 2026-06-05 | P0.5 — Register SEA-LION 9B generation model in LiteLLM | ✅ Done |
| 2026-06-05 | P0.6 — Synthetic seed-data generator (Python) | ✅ Done |
| 2026-06-05 | P0.7 — Ingestion: chunk + embed + upsert + ingest CLI | ✅ Done |
| 2026-06-05 | P0.8 — Retriever (HNSW vector search, tenant-scoped) | ✅ Done |
| 2026-06-05 | P0.9 — Taglish persona prompt assembler | ✅ Done |
| 2026-06-05 | P0.10 — Streaming generator client (LiteLLM SSE) | ✅ Done |
| 2026-06-05 | P0.11 — `ask` CLI (Phase 0 deliverable) | ✅ Done |
| 2026-06-05 | P0.12 — HTTP health skeleton + `-migrate` flag | ✅ Done |
| 2026-06-05 | **Phase 0 — Foundation** | ✅ **COMPLETE** |
| 2026-06-05 | P2.1 — buyer/installer mode plumbing (prompt modes + chat wiring) | ✅ Done |
| 2026-06-05 | P2.2 — buyer/installer seed docs + re-ingest | ✅ Done |
| 2026-06-05 | P2.3 — mode selector in HTML test page | ✅ Done |
| 2026-06-05 | P2.4 — topic-gate installer/buyer vocabulary + live e2e verify | ✅ Done |
| 2026-06-05 | P2-LoRA — Taglish voice LoRA (self-distill, prompt distillation): pipeline + train + eval gate | ✅ **SHIPPED** |
