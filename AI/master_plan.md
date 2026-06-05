# Apolaki Solar Assistant — Master Plan

> Session-to-session memory. **Start each session** by reading this + `AGENTS.md` (when added) + the PRD.

## Project
Custom, **solar-only** support assistant (the "solar brain") for the **Apolaki** solar marketplace (web + mobile).
**North star:** advocacy for **domestic solar adoption** — ROI / energy independence in **PHP**, spoken in **Taglish**.
Self-hosted on the **Mac Studio** behind **LiteLLM**, **RAG** over Apolaki docs. **≈ ₱0 per query.**

## Status — 2026-06-05
- ✅ Design / PRD complete and **approved** → `AI/docs/PRDs/2026-06-05-apolaki-solar-assistant-design.md`
- ✅ **Local serving stack hardened** (fixed timeouts/peer-resets): LiteLLM `:4000` live with retries + MLX→Ollama fallback; MLX real streaming. See memory `local-serving-stack`.
- ✅ **Phase 0 implementation plan written** → `AI/docs/tasks/2026-06-05-phase-0-foundation.md` (13 bite-sized TDD tasks P0.0–P0.12). Phases 1–3 kept as roadmap (detail gated on P0 outcomes + real data).
- ✅ **P0.0 done** — Go module `github.com/apolaki/solar-assistant`, `.env.example`, Makefile, `internal/config` (TDD, passing). Commit `7175b3d`.
- ⏳ **Next:** execute **P0.1** (local pgvector Postgres + pgxpool connector).
- ⬜ Phase 0 — Foundation (Go service + RAG + synthetic data + CLI test harness)
- ⬜ Phase 1 — Customer self-service MVP (Vue widget, guardrails, logging + feedback)
- ⬜ Phase 2 — Light Taglish LoRA fine-tune + buyer/installer modes
- ⬜ Phase 3 — Advocacy features + scale (cloud GPU once past ~1,000 users)

## Locked Decisions (see PRD §3)
- **Architecture:** standalone **Go** service (own repo), RAG via **pgvector + HNSW** in Apolaki's Postgres.
- **Models:** **Gemma-SEA-LION-v3-9B-IT** (primary), **Qwen3-7B** (fallback), **BGE-M3** embeddings — served via **LiteLLM**.
- **Phase 1 = homeowners**, **Taglish**; **synthetic seed data now**, real tickets later (flywheel).
- **Hosting:** Mac Studio = production **until ~1,000 users**; expected **< 10k calls/day**.
- **Guardrails:** 3-layer solar-only (topic gate → grounded-only → safety/escalate).

## Next Session
- Execute **P0.1** (docker pgvector Postgres + `internal/db` pool) from `AI/docs/tasks/2026-06-05-phase-0-foundation.md`, one task per session per the ai-wf loop.
- Before coding, confirm services are up: `curl :4000/health/liveliness`, `:8000/health`, `:11434/api/tags` (rerun `agent_skills/start-*.sh` after reboot — they don't auto-start yet).

## Task Log
| Date | Task | Status |
|------|------|--------|
| 2026-06-05 | Brainstorm + design / PRD | ✅ Done |
| 2026-06-05 | git init + scaffold (PRD, master_plan, .gitignore) | ✅ Done |
| 2026-06-05 | Fix local-serving timeouts/peer-resets (LiteLLM proxy + MLX real streaming + fallback) | ✅ Done |
| 2026-06-05 | Phase 0 implementation plan (P0.0–P0.12) | ✅ Done |
| 2026-06-05 | P0.0 — Go module + typed config (TDD) | ✅ Done |
| — | P0.1 — pgvector Postgres + db pool | ⏳ Next |
