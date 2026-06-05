# Apolaki Solar Assistant — Design / PRD

- **Date:** 2026-06-05
- **Status:** Approved design (all open questions resolved in review) → ready for implementation planning
- **Owner:** sadhu.sreenivas@gmail.com
- **Codename:** Apolaki Solar Assistant (the "solar brain")
- **Home:** **its own repo** — `Documents/llm-slm` (separate from the Apolaki monorepo); integrates with Apolaki over HTTP
- **Integrates with:** `Code/apolaki-udpated-app` (Vue 3 + Vite frontend, Go microservices, Postgres, LiteLLM)

---

## 1. Mission & Summary

Build a **custom, solar-only support assistant** for the Apolaki solar marketplace (web + mobile), self-hosted and frugal. Its north star is **advocacy for domestic (residential) solar adoption** — making homeowners confident about energy independence and ROI (in PHP), not just answering dry technical questions.

It is **one shared "solar brain"** (one model + one knowledge base) that serves multiple audiences through different *modes* (system prompt + retrieval filter + UI entry point), shipped in **phases**. Phase 1 serves **customer self-service** for homeowners.

The assistant answers **grounded in Apolaki's own documents** (Retrieval-Augmented Generation), speaks **Taglish**, runs on **hardware already owned** (Mac Studio M2 Max, 64 GB) behind the existing **LiteLLM** proxy, and costs **≈ ₱0 per query**.

---

## 2. Goals & Non-Goals

### Goals
- Answer residential solar questions accurately, grounded in Apolaki's docs, in warm Taglish.
- Deflect routine support load and actively **advocate** solar adoption (ROI/energy-independence framing).
- Personalize using the logged-in user's own system specs/monitoring (via `solar-service`).
- Stay **strictly solar-only**; escalate to a human when unsure rather than improvise.
- Run at near-zero marginal cost; keep the hosting choice swappable.
- Build a data flywheel (logged chats + feedback) that funds a later light fine-tune.

### Non-Goals (YAGNI)
- ❌ Not a general-purpose chatbot. Off-topic queries are politely redirected.
- ❌ No training a model from scratch. We use open small models + RAG.
- ❌ No fine-tuning in Phase 1 (no curated data yet). Fine-tune is Phase 2, on real logged chats.
- ❌ No separate vector database. We use `pgvector` inside Postgres.
- ❌ No code inside the Apolaki monorepo — this is a **standalone repo/service**.
- ❌ No native mobile app work in Phase 1 — API-first; the mobile PWA reuses the web endpoint.
- ❌ No cloud GPU spend until we pass ~1,000 users.

---

## 3. Decisions Locked

| Decision | Choice | Rationale |
|---|---|---|
| **Mission** | Advocacy engine for domestic solar adoption | User's explicit north star; matches brand voice (ROI, energy independence, PHP) |
| **Phase 1 audience** | Customer self-service (homeowners) | Highest volume, doubles as advocacy, best data flywheel |
| **Language** | Taglish (English ↔ Filipino code-switching) | Warm, homeowner-friendly; Filipino well-supported by SEA-LION |
| **Hosting** | **Mac Studio (M2 Max, 64 GB) behind LiteLLM — production until ~1,000 users**, then revisit cloud GPU | Zero per-query cost; comfortably serves <10k calls/day; swappable with no app change |
| **Expected volume** | **< 10,000 calls/day** | ~0.12 calls/s avg; fits a single 9B on the Mac with headroom |
| **Method** | RAG first → light LoRA fine-tune later (voice only) | Frugal, always factually fresh; fine-tune trained on real logged chats |
| **Architecture** | One shared brain, phased rollout | Build once, serve all audiences via modes |
| **Service language / home** | **Go**, in its **own repo** (`llm-slm`) | Matches existing services; standalone, integrates over HTTP |
| **Generation model** | **Gemma-SEA-LION-v3-9B-IT** (primary), **Qwen3-7B** (fallback) | SEA-LION is #1 for Filipino on SEA-HELM; both open-source, cleared for commercial use |
| **Embedding model** | **BGE-M3** (1024-dim) | Multilingual (incl. Filipino), self-hosted quality matches paid APIs, pgvector-native |
| **Vector store + index** | `pgvector` in Postgres, **HNSW index** | No new infra; HNSW = best recall/latency for a frequently-updated, modest-size KB (no training/rebuild step) |
| **Seed data** | **Create synthetic/sample data now**; real customer tickets later via the flywheel | Lets us build and test RAG immediately without waiting for real data |

---

## 4. Architecture

One new service, **zero new infrastructure**. `solar-assistant-service` is a **standalone Go service** (own repo) that talks to the Apolaki platform over HTTP and reuses its Postgres (+ `pgvector`), LiteLLM, and JWT auth.

```
 Apolaki app  (Vue web / mobile PWA)
      │   POST /assistant/chat   (SSE stream, platform JWT)
      ▼
┌──────────────────────────────────────────────────┐
│  solar-assistant-service  (new, Go, own repo)    │
│                                                  │
│  1. Topic gate ........ solar-related? → else polite Taglish redirect
│  2. Retrieve .......... pgvector (Postgres, HNSW), tenant-scoped
│  3. Personalize ....... pull THIS user's system specs → solar-service
│  4. Assemble prompt ... Taglish advocate persona + sources + history
│  5. Generate .......... LiteLLM :4000 → SEA-LION 9B on the Mac (SSE)
│  6. Safety + grounding. no good source? → escalate, don't improvise
│  7. Log turn + 👍/👎 ... feeds fine-tune & eval flywheel
└──────────────────────────────────────────────────┘
      │   streamed answer  +  citations  +  "talk to a human" button
      ▼
 Homeowner gets grounded, friendly Taglish solar help
```

### Why pgvector + HNSW (not a separate vector DB)
Postgres is already running; `pgvector` adds semantic search *in the same database* — no new service, transactional with existing data. **HNSW** is the strategic index here: best recall + low query latency, **no training step**, and graceful incremental inserts as new docs land — ideal for a frequently-updated, modest-size KB (well under 100k chunks) at <10k queries/day. (IVFFlat would need a training pass and periodic `REINDEX`, with weaker small-scale recall.)

### Inference serving
- Models run **4-bit** on the Mac via **MLX** (Apple-Silicon fast path) or **Ollama**; obtain/convert a 4-bit MLX or GGUF build of Gemma-SEA-LION-9B in Phase 0.
- **LiteLLM (:4000)** is the single gateway: app and service always call one stable endpoint, so models can be swapped/routed with no app change. LiteLLM handles timeouts, retries, and a fallback model.
- **Capacity:** <10k calls/day is ~0.12/s average; even over a 7-hour peak window it's <0.5/s. A single 9B 4-bit on the M2 Max handles this comfortably with LiteLLM queueing — consistent with "Mac is production until ~1,000 users."
- **Optional two-tier routing (Phase 2):** default to the fast 9B; route detected "hard" queries to the already-installed 32B for a quality boost — frugal, not wasteful.

---

## 5. Request Lifecycle (customer self-service)

1. User opens the chat widget and sends a message. Request carries: message, recent history, platform **JWT**, `tenant_id`, `user_id`.
2. **Topic gate:** classify whether the message is solar-related (cheap keyword + embedding/LLM check). If not → polite Taglish redirect; don't call the big model.
3. **Retrieve:** embed the query with **BGE-M3**; HNSW vector-search `knowledge_chunks`, filtered to `tenant_id` **∪** shared/global docs, plus audience=`customer`. Return top-k chunks with source metadata.
4. **Personalize (optional):** if the question is about the user's own system, fetch specs/monitoring from `solar-service` to enable "your 5 kW system…" answers.
5. **Assemble prompt:** system prompt (Taglish advocate persona, solar-only, brand voice, PHP, no jargon, cite sources, escalate when unsure) + retrieved chunks + user context + history.
6. **Generate:** stream via **LiteLLM** → SEA-LION 9B on the Mac → **SSE** to the client.
7. **Safety + grounding:** if retrieval found nothing relevant, the model **escalates** ("let me connect you to a specialist") instead of inventing. Hands-on/electrical content gets a **licensed-installer disclaimer**.
8. **Log:** persist the full turn (question, retrieved chunk IDs, answer, model, latency) + a **👍/👎 + "solved?"** hook → the future fine-tune dataset and eval set.

---

## 6. Service Design (Go)

### Endpoints
| Method · Path | Purpose | Auth |
|---|---|---|
| `POST /assistant/chat` | Send a message, receive a streamed (SSE) grounded answer + citations | User JWT |
| `POST /assistant/feedback` | Record 👍/👎 + "solved?" for a message | User JWT |
| `POST /assistant/ingest` | Ingest/refresh knowledge documents (triggers embedding) | Admin JWT |
| `GET /assistant/health` | Liveness/readiness (model reachable, DB reachable) | none |

### Internal modules
`topic-gate` → `retriever` (pgvector/HNSW) → `personalizer` (solar-service client) → `prompt-assembler` (Taglish persona templates per mode) → `generator` (LiteLLM streaming client) → `safety-filter` (grounding check + disclaimers) → `logger`.

### Data model (Postgres, multi-tenant)
All tables carry `tenant_id` per the platform's multi-tenant rule; queries are tenant-scoped and **parameterized**.

```
knowledge_documents
  id, tenant_id (NULL = shared/global), title, source_type, source_uri,
  audience (customer|buyer|installer|agent), product, brand, language,
  version, content_hash, created_at, updated_at

knowledge_chunks
  id, document_id → knowledge_documents.id, tenant_id, chunk_index,
  content, embedding vector(1024), token_count, created_at
  -- HNSW index on embedding; btree on (tenant_id, audience)

conversations
  id, tenant_id, user_id, mode (customer|buyer|installer|agent),
  channel (web|mobile), created_at, status (open|escalated|closed)

messages
  id, conversation_id, tenant_id, role (user|assistant|system),
  content, retrieved_chunk_ids[], model, latency_ms, created_at

feedback
  id, message_id, tenant_id, user_id, rating (up|down),
  solved (bool|null), note, created_at
```

> `vector(1024)` matches BGE-M3's dense dimension; pin it once confirmed against the exact model build in Phase 0.

### Engineering standards (per CLAUDE.md)
- Parameterized SQL only; `tenant_id` in every query.
- Input validation at the API boundary (message length, mode enum, IDs).
- Error handling on every external call (DB, LiteLLM, solar-service) with timeouts + graceful degradation (LiteLLM down → escalate to human, never silent 500).
- Unit + integration tests per module; golden-set eval (see §10).
- No secrets in code; reuse platform config/secret conventions.

---

## 7. Model Stack

| Role | Model | License | Notes |
|---|---|---|---|
| Generation (primary) | **Gemma-SEA-LION-v3-9B-IT** | Open source, cleared for commercial use (Gemma terms) | #1 for Filipino on SEA-HELM; 4-bit ≈ ~6 GB |
| Generation (fallback) | **Qwen3-7B (Instruct)** | Open source (Apache-2.0) | Strong multilingual; license-clean fallback |
| Embeddings | **BGE-M3** | MIT | Multilingual incl. Filipino; 1024-dim; dense+sparse+multi-vector |
| Gateway | **LiteLLM** | Open source | Single endpoint, routing, fallback, rate-limit |
| Escalation tier (opt., Ph2) | existing 32B (qwen-dev) | — | Route only "hard" queries here |

**Memory budget (64 GB unified):** 9B 4-bit (~6 GB) + BGE-M3 (~2 GB) ≈ ~8 GB, leaving large headroom for KV cache + concurrency. The 32B (~20 GB) fits as an optional escalation tier.

**Why 9B, not the 32B by default:** a Filipino-tuned 9B gives better Taglish, ~3× faster responses, and more concurrent users — frugal *and* higher-quality for this task.

---

## 8. Solar-Only Guardrails & Safety

Enforced in **three layers**:
1. **Input topic-gate:** non-solar questions get a polite Taglish redirect; the big model isn't called.
2. **Retrieval-grounding:** the model answers *only* from retrieved Apolaki sources; if nothing relevant is found, it **escalates to a human** instead of improvising.
3. **Output safety layer:** hands-on/electrical content gets a **"consult a licensed installer"** disclaimer; a **human-handoff** path is always offered.

**Privacy (PH Data Privacy Act / RA 10173):** real tickets (added later) may contain PII. Ingestion strips/masks PII where feasible; conversation logs are tenant-scoped and access-controlled; document a data-retention policy before ingesting real tickets. (Synthetic seed data carries no real PII.)

---

## 9. Data Plan — synthetic now, real later

1. **Create synthetic/sample seed data now** *(Python pipeline)* — generate a realistic starter corpus so we can build and test RAG immediately:
   - Sample **panel/inverter datasheets** + **error-code tables**.
   - A homeowner **FAQ** on PH essentials — **net metering, Meralco/local-utility bills, brownout backup, financing, ROI in ₱**.
   - A set of **synthetic resolved tickets** (realistic Taglish Q→A pairs).
   Pipeline: generate/load → clean → chunk → embed (BGE-M3) → upsert. Re-runnable, versioned, tagged by audience/product/brand/language/source.
2. **Real data later:** actual customer tickets and docs replace/augment the synthetic corpus as they arrive.
3. **Flywheel:** every chat + 👍/👎 is logged → after enough good interactions, build (a) a **LoRA fine-tune set** for Taglish/Apolaki voice and (b) a **golden eval set**. This activates Phase 2's light fine-tune cheaply, on real data.

---

## 10. Evaluation & Metrics

- **Golden eval set:** curated Q→ideal-answer pairs (seeded from the synthetic corpus, grown with real chats), run on every change. Tracks: accuracy, grounding (claims supported by sources?), refusal correctness (escalates when it should?), Taglish quality.
- **Product metrics:** deflection rate, CSAT / 👍 ratio, escalation rate, latency (p50/p95), cost (≈ ₱0 marginal).
- **Regression gate:** a change ships only if the golden eval doesn't regress beyond a set threshold.

---

## 11. Integration with Apolaki

- **Standalone service:** lives in its own repo (`llm-slm`); communicates with the Apolaki platform over HTTP. Not embedded in the monorepo.
- **Frontend:** a Vue 3 chat-widget component (Tailwind, matches design system). Streams via SSE; renders **citations** + a **"talk to a human"** button.
- **Mobile:** the PWA reuses the **same `/assistant/chat` endpoint** — no separate work in Phase 1.
- **Auth/context:** reuse the platform **JWT**, `tenant_id`, `user_id`.
- **Escalation:** "talk to a human" hands off to the existing support queue; later the *same brain* powers the **agent-copilot** persona.

---

## 12. Phased Roadmap

### Phase 0 — Foundation (internal)
Standalone Go service skeleton + `pgvector`/HNSW tables + LiteLLM wired to SEA-LION 9B + BGE-M3 embeddings + **synthetic seed-data generation** + ingestion + a CLI test harness.
**Deliverable:** ask a solar question in the terminal → get a grounded Taglish answer with sources.

### Phase 1 — Customer self-service MVP
Vue chat widget, solar-only guardrails, citations, human escalation, conversation logging + 👍/👎. Launch **low-risk** (beta badge / gated cohort).
**Deliverable:** homeowners get instant Taglish solar help in-app; measure deflection + CSAT.

### Phase 2 — Light Taglish fine-tune + expand
LoRA fine-tune (voice only) on collected chats; add **buyer/pre-sales** and **installer** modes (same brain, new prompts/filters); optional two-tier model routing.

### Phase 3 — Advocacy & scale
Proactive advocacy (personalized savings nudges — "your roof ≈ ₱X/yr"); more PH languages (Tagalog, Cebuano); move serving to a cloud GPU **once past ~1,000 users** (a LiteLLM config swap, no app change).

---

## 13. Cost & Operations

- **Marginal cost ≈ ₱0:** owned hardware + open models + pgvector inside existing Postgres.
- **Real investment:** time (build + synthetic-data curation) + electricity.
- **Capacity:** <10k calls/day fits the Mac comfortably; **Mac is the production server until ~1,000 users**, then revisit cloud GPU.
- **Ops:** LiteLLM provides one stable endpoint, timeouts, retries, fallback. The Mac must be reachable/always-on for production.
- **Cloud trigger:** passing ~1,000 users, sustained concurrency the Mac can't meet, or a 24/7 SLA requirement.

---

## 14. Risks & Mitigations

| Risk | Mitigation |
|---|---|
| Synthetic data ≠ real questions | Realistic Taglish samples to start; flywheel swaps in real tickets fast; eval catches drift |
| Hallucination / wrong solar advice | Grounded-only answers; escalate-when-unsure; safety disclaimers |
| Taglish quality of base model | SEA-LION (#1 Filipino); Phase-2 LoRA tunes voice; Qwen3 fallback |
| Mac throughput / availability | <10k/day fits; LiteLLM queue + graceful "talk to a human" fallback; cloud swap past 1k users |
| PII in real tickets (PDPA) | Strip/mask on ingest; tenant-scoped, access-controlled logs; retention policy before real data |

---

## 15. Resolved in Review (2026-06-05)

1. **Production hosting:** Mac Studio serves production **until ~1,000 users**, then revisit cloud GPU.
2. **Licenses:** Gemma-SEA-LION and Qwen3 are **open-source and cleared for commercial use**.
3. **Seed data:** **create synthetic/sample data now**; real customer tickets come later via the flywheel.
4. **Service home:** **separate repo** (`llm-slm`), outside the Apolaki monorepo; integrates over HTTP.
5. **Volume:** **< 10,000 calls/day** expected.
6. **Vector index:** **HNSW** (best recall/latency for a frequently-updated, modest-size KB; no training/rebuild step).

---

## Appendix A — Draft Taglish system prompt (persona)

> You are **Apolaki Assistant**, a warm, encouraging solar-energy guide for Filipino homeowners. Speak natural **Taglish**. Your goal is to help people feel confident about going solar — emphasize **savings, ROI in pesos (₱), and energy independence**. Avoid heavy technical jargon; explain simply. **Only answer questions about solar energy and Apolaki's products/services.** If a question is off-topic, gently steer back to solar. **Only state facts found in the provided sources**; if the sources don't cover it, say you'll connect them to a specialist — never guess. For anything involving wiring, electrical work, or installation safety, remind them to consult a **licensed installer**. Always be kind, clear, and motivating.

## Appendix B — Draft `/assistant/chat` contract

```
POST /assistant/chat   (Authorization: Bearer <JWT>)
Request:
  { "conversation_id": "uuid|null", "message": "string",
    "mode": "customer", "channel": "web|mobile" }
Response: text/event-stream (SSE)
  event: token   data: {"text": "..."}            // streamed tokens
  event: sources data: [{"title","source_uri","chunk_id"}]
  event: done    data: {"message_id","escalate": false}
Errors: 400 (validation), 401 (auth), 503 (model unavailable → escalate)
```
