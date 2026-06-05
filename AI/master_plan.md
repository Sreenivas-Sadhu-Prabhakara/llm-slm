# Apolaki Solar Assistant — Master Plan

> Session-to-session memory. **Start each session** by reading this + `AGENTS.md` (when added) + the PRD.

## Project
Custom, **solar-only** support assistant (the "solar brain") for the **Apolaki** solar marketplace (web + mobile).
**North star:** advocacy for **domestic solar adoption** — ROI / energy independence in **PHP**, spoken in **Taglish**.
Self-hosted on the **Mac Studio** behind **LiteLLM**, **RAG** over Apolaki docs. **≈ ₱0 per query.**

## Status — 2026-06-05
- ✅ Design / PRD complete and **approved** → `AI/docs/PRDs/2026-06-05-apolaki-solar-assistant-design.md`
- ⏳ **Next:** implementation plan (all phases 0–3) via the writing-plans workflow
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
- Generate the **implementation plan** (writing-plans) → save under `AI/docs/tasks/`.
- Then begin **Phase 0**, one task per session per the ai-wf loop.

## Task Log
| Date | Task | Status |
|------|------|--------|
| 2026-06-05 | Brainstorm + design / PRD | ✅ Done |
| 2026-06-05 | git init + scaffold (PRD, master_plan, .gitignore) | ✅ Done |
| — | Implementation plan (phases 0–3) | ⏳ Next |
