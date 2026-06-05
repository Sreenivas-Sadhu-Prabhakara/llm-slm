# Phase 1 — Customer Self-Service MVP (Backend) — Design

> Approved design. Builds on the Phase 0 foundation (`retriever`, `prompt`, `generator`, `embed`, `db`).
> Source of truth: PRD §5 (request lifecycle) and §6 (service design).

## 1. Goal
Expose the Phase 0 RAG path as the real Apolaki assistant HTTP service: a streaming
chat endpoint with a solar-only topic gate, grounded-or-escalate safety, turn logging,
and a feedback hook — plus a local browser test page standing in for the (out-of-repo)
Vue widget.

## 2. Scope
**In:** topic gate, safety/grounding filter, personalizer seam, turn logging, `POST /assistant/chat`
(SSE), `POST /assistant/feedback`, `GET /assistant/health`, static HTML chat page, stub-JWT middleware.

**Out (deliberately):**
- The production Vue chat widget — lives in the Apolaki monorepo (PRD non-goal "no code inside the monorepo"). Replaced here by a local static HTML page.
- Real JWT issuance — no Apolaki auth server locally. Middleware validates if a signing key is configured, else runs permissive (dev) and reads `tenant_id`/`user_id` from headers.
- Real personalization — `solar-service` does not exist locally. We ship a `Personalizer` interface + `Noop` impl; a real client drops in later with no call-site change.

## 3. Decisions (this phase)
| Decision | Choice | Rationale |
|---|---|---|
| Auth | Optional/stub middleware | No local auth server; wire the seam, stay permissive in dev. Tenant/user from `X-Tenant-Id`/`X-User-Id` headers or dev defaults. |
| Personalizer | Interface + `Noop` | Keep PRD lifecycle step 4 without inventing data. |
| Test UI | Static HTML chat page at `GET /` | Real in-browser SSE verification of Phase 1. |
| Topic gate | Cheap keyword/heuristic (no LLM call) | PRD: "don't call the big model" for off-topic; fast + free. |
| Migrations | None new | P0.2 schema already has conversations/messages/feedback with the needed columns. |

## 4. Modules (each single-purpose, TDD)
- **`internal/topicgate`** — `IsSolarRelated(q string) bool` + `Redirect() string` (canned Taglish). Keyword set covers EN+Filipino solar terms (solar, panel, baterya, kuryente, Meralco, net metering, ROI, tipid, kuryente bill…).
- **`internal/safety`** — `Check(chunks []retriever.Chunk, question string) Decision` where `Decision{Escalate bool, Disclaimer string}`. Escalate when no chunk clears a min cosine score; append licensed-installer disclaimer when question hits electrical/wiring/installation terms.
- **`internal/personalizer`** — `type Personalizer interface { Context(ctx, userID string) (string, error) }`; `Noop` returns `("", nil)`.
- **`internal/chatlog`** — wraps the pool:
  - `StartConversation(ctx, tenantID, userID *string, mode, channel) (convID string, err error)`
  - `LogTurn(ctx, convID string, tenantID *string, question, answer, model string, chunkIDs []string, latencyMs int) (userMsgID, asstMsgID string, err error)`
  - `RecordFeedback(ctx, messageID string, tenantID, userID *string, rating string, solved *bool, note string) error`
  - All parameterized; `tenant_id` NULL = shared.
- **`internal/httpapi`** — `Handler(deps)` returns an `http.Handler` (ServeMux):
  - `POST /assistant/chat` — body `{conversation_id?, message}`; topic-gate → (redirect SSE | retrieve→personalize→assemble→stream→safety-disclaimer→log); response header `X-Message-Id` for feedback; SSE `event: error` on failure.
  - `POST /assistant/feedback` — body `{message_id, rating, solved?, note?}` → `chatlog.RecordFeedback`.
  - `GET /assistant/health` — pool ping (moved from cmd/server).
  - `GET /` — embedded static HTML chat page (uses `fetch` + `ReadableStream` to consume SSE).
  - `authMiddleware` — wraps all `/assistant/*`: if signing key set, validate Bearer JWT (HS256) → tenant/user; else permissive, headers/dev defaults.

## 5. Request flow (PRD §5)
```
POST /assistant/chat
  → auth (tenant_id, user_id)
  → topicgate.IsSolarRelated?
      no  → SSE the Taglish redirect, log turn (model="topic-gate"), done
      yes → retriever.Search(tenant, "customer", k)
          → personalizer.Context (Noop today)
          → prompt.Assemble(question, chunks)  [+ personal context]
          → generator.Stream → SSE deltas to client (accumulate answer)
          → safety.Check → if escalate, stream escalation tail; else append disclaimer if any
          → chatlog.LogTurn(question, chunkIDs, answer, model, latency)
          → trailer: source titles + X-Message-Id
```

## 6. Error handling
- Handlers return correct status codes; bodies validated at the boundary.
- Once SSE has started, errors are surfaced as `event: error` then the stream closes.
- Logging failures are **non-fatal**: logged server-side, never break the user's answer.
- Personalizer errors are non-fatal (degrade to no personal context).

## 7. Testing
- Unit (no infra): topicgate, safety, personalizer.
- Handler tests (httptest + fake retriever/generator interfaces): chat happy path streams + sets `X-Message-Id`; off-topic returns redirect without invoking the generator; feedback validates input.
- One live end-to-end smoke against real infra: open `GET /`, ask a grounded question (streamed Taglish + sources), thumbs-down, confirm rows in `messages`/`feedback`.

## 8. Out-of-scope follow-ups (tracked, not built here)
- Real JWT verification against Apolaki's key; real `solar-service` personalizer client; the monorepo Vue widget; admin `POST /assistant/ingest` (Phase 0 `cmd/ingest` already covers ingestion via CLI).
