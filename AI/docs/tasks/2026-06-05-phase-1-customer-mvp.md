# Phase 1 — Customer Self-Service MVP — Task Plan

TDD, one module per commit. Builds on Phase 0. Spec: `AI/docs/PRDs/2026-06-05-phase-1-customer-mvp-design.md`.

- **P1.1 — `internal/topicgate`**: `IsSolarRelated(q) bool` + `Redirect() string`. Tests: solar Q → true; "sino panalo sa NBA?" → false; redirect is Taglish. *(no infra)*
- **P1.2 — `internal/safety`**: `Check(chunks, question) Decision{Escalate, Disclaimer}`. Tests: empty/low-score chunks → escalate; "paano mag-wiring?" → installer disclaimer; normal grounded Q → neither. *(no infra)*
- **P1.3 — `internal/personalizer`**: `Personalizer` interface + `Noop`. Test: Noop returns empty, nil. *(no infra)*
- **P1.4 — `internal/chatlog`**: `StartConversation`/`LogTurn`/`RecordFeedback` over conversations/messages/feedback. Test: integration insert + read-back (needs DB).
- **P1.5 — `internal/httpapi`**: handlers + stub-JWT middleware + embedded HTML page. Tests: httptest with fake retriever/generator — chat streams + `X-Message-Id`; off-topic → redirect, generator not called; feedback input validation. *(no infra)*
- **P1.6 — wire `cmd/server` + e2e**: mount handlers; live smoke (browser chat + feedback) → rows in `messages`/`feedback`.

## Done when
Browser chat page returns streamed grounded Taglish + sources; off-topic redirects; thumbs-down persists; full suite + `go vet` green.
