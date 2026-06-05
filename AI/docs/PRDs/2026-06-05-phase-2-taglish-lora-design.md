# Phase 2 — Taglish Voice LoRA (Prompt Distillation) — Design

> Approved design. Builds on Phase 0 (RAG path) + Phase 1 (HTTP assistant) + the
> P2.1–P2.4 buyer/installer modes. Source-of-truth intent: main PRD §3 Decisions Locked
> (Method: "RAG first → light LoRA fine-tune later (voice only)"), §10 Evaluation & Metrics,
> §12 Phase 2 roadmap, and Appendix A (draft Taglish persona prompt).

## 1. Goal
Self-distill the **SEA-LION 9B** generation model into a LoRA adapter that **reproduces
the full Apolaki persona behavior from a much shorter system prompt**. The model is the
teacher *and* the student (fully local, zero API cost), so the win is **not** raw voice
quality (capped at base) — it is **prompt distillation**: baking the persona into the
weights to

- **reclaim 16K context budget** (CLAUDE.md hard constraint) — a shorter system prompt
  leaves more room for RAG chunks + history, and
- **improve consistency** of format / grounding / citation / escalation / off-topic decline.

This ships as the new prod default generation model, served MLX-native, with the existing
Ollama GGUF SEA-LION kept as the LiteLLM fallback.

### Locked decisions (from brainstorm)
| Decision | Choice |
|---|---|
| Objective | Production run now — replace the prod generation model (not design-only). |
| Teacher | Local SEA-LION 9B, **self-distillation** (fully local, free). |
| Success goal | **Prompt distillation** — `tuned + short-prompt` ≥ `base + full-prompt`. |
| Serving | **MLX-native**: HF→MLX convert → `mlx_lm.lora` → fuse → `mlx_lm.server`; LiteLLM primary, Ollama GGUF fallback. |

### Ship gate (the whole point)
The candidate ships **only if** on the golden eval set:
1. `tuned + short-prompt` **≥** `base + full-prompt` on aggregate quality (voice/format/grounding/citation), **and**
2. **off-topic-decline rate = 100%** (no regression), **and**
3. **safety/wiring-escalation rate = 100%** (no safety regression), **and**
4. **no hallucination regression** (grounding pass-rate not lower than baseline).

If the gate fails, we **do not ship** — rollback to base + full prompt.

## 2. Pipeline (stages)
1. **Question bank** — ~300–500 realistic questions across `customer`/`buyer`/`installer`
   modes plus adversarial cases (off-topic, wiring/safety), generated programmatically from
   the seed-corpus topics + paraphrase templates. Versioned (`training/questions.jsonl`).
2. **`cmd/distill` (Go)** — reuses `internal/{retriever,prompt,generator}` so there is **no
   logic drift** from production. For each question: retrieve with the mode's audience →
   assemble the **FULL** persona prompt → generate the gold answer with SEA-LION via LiteLLM.
   Emits training examples whose **input uses the SHORT prompt** and whose **output is the
   full-prompt gold answer** — this *is* the distillation signal.
3. **Curate / filter (Python, `training/curate.py`)** — keep only examples passing automated
   checks: grounded (claim/source overlap; no obvious hallucination), cites a source title,
   off-topic→declines (matches redirect intent), wiring/safety→escalates, Taglish present
   (Filipino function words), length bounds. Optional local LLM-judge (SEA-LION or local 32B)
   scoring 1–5, keep ≥4. Drops the mediocre — critical because teacher = student. Reports
   kept/dropped counts.
4. **Format + split** — MLX chat-format jsonl (`{"messages":[{system:SHORT},{user:…SOURCES…},
   {assistant:GOLD}]}`), split 80/10/10 train/valid/**test**. The test split seeds the golden
   eval set; adversarial cases are force-included in the eval set.
5. **Model prep** — download `aisingapore/Gemma-SEA-LION-v3-9B-IT` HF safetensors (~18GB,
   one-time) → `mlx_lm.convert` to MLX (bf16; 4-bit only if needed for memory).
6. **Train** — `mlx_lm.lora`: LoRA on attention (+ MLP) projections, rank ~8–16, modest iter
   count, bf16 base; **fall back to 4-bit QLoRA** if OOM in 64GB. Fixed seed.
7. **Fuse** — `mlx_lm.fuse` → standalone `training/fused/sea-lion-taglish`.
8. **Eval (the gate, `training/eval.py`)** — for each golden question, generate both
   `base + full-prompt` (baseline) and `tuned + short-prompt` (candidate). Score via rubric
   (automated checks) + local LLM-judge. Report: voice/format/grounding/citation pass-rates,
   off-topic-decline rate, safety-escalation rate, mean answer length, and **prompt-token
   savings**. Apply the §1 ship gate.
9. **Serve + swap** — run the fused model via `mlx_lm.server` on **:8001** (avoids the Qwen
   MLX server on :8000); register in `litellm_config.yaml` as the primary `sea-lion-9b` route
   with the Ollama GGUF SEA-LION as fallback. Add `prompt.SystemShort` and have the Go service
   use it for the tuned model. Restart LiteLLM, smoke-test end-to-end (the existing chat path).
   **Rollback** = revert the LiteLLM route to the Ollama GGUF primary + revert the prompt
   switch (both config-gated; no data loss).

## 3. Data flow
```
questions.jsonl
  → cmd/distill (retrieve[mode] → FULL prompt → SEA-LION gold)   [raw examples]
  → curate.py (filter + optional judge)                          [kept examples]
  → split train/valid/test (MLX chat jsonl)
  → mlx_lm.convert (HF→MLX)  +  mlx_lm.lora (train)  →  adapter
  → mlx_lm.fuse  →  fused/sea-lion-taglish
  → eval.py (candidate[tuned+short] vs baseline[base+full]) → GATE
  → mlx_lm.server :8001  +  litellm_config swap (primary) / Ollama GGUF (fallback)
```

## 4. Components & boundaries
- **`cmd/distill`** (Go, new) — single purpose: emit `(short-input, gold-output)` examples
  by reusing prod retrieval/prompt/generation. Depends on `internal/{config,db,retriever,
  prompt,generator,embed}`. Output: `training/data/raw.jsonl`.
- **`training/`** (Python/MLX, new) — `make_questions.py`, `curate.py`, `train.sh`
  (wraps `mlx_lm.convert`/`lora`/`fuse`), `eval.py`, `golden_set.jsonl`, `requirements.txt`,
  a `run_manifest.json` (hyperparams + data hash + seed). **Versioned:** scripts +
  `questions.jsonl` + `golden_set.jsonl`. **Gitignored (heavy):** `training/models/`,
  `training/fused/`, `training/data/`.
- **`internal/prompt`** — add a tested `SystemShort` const; the Go service selects it when the
  configured generation model is the tuned one.
- **LiteLLM config** (`agent_skills/litellm_config.yaml`) — infra, unversioned (existing pattern).

## 5. Testing
- `cmd/distill`: unit-test example shaping (short prompt used for the input, gold answer for
  the output) with a fake retriever/generator — mirrors the `internal/httpapi` test pattern.
- `internal/prompt`: `SystemShort` is non-empty and retains the core guardrail intent
  (grounded-only / solar-only / cite / escalate signals).
- `training/`: pytest on the `curate.py` filter rules (e.g. an off-topic example is kept only
  if its answer declines; a wiring example only if it escalates) and on `eval.py` scoring.
- The **eval harness is the integration test** — it emits the ship/no-ship verdict.

## 6. Error handling & risks
- **Self-distill ceiling = base.** Mitigation: hard curation (drop the mediocre) + the ship
  gate. If `tuned+short` cannot beat `base+full`, we do not ship.
- **Memory.** 9B bf16 LoRA in 64GB unified — monitor; QLoRA (4-bit base) fallback.
- **Determinism.** Fixed seed; `run_manifest.json` records hyperparams, data hash, seed.
- **Infra coupling.** Distillation needs LiteLLM (:4000), BGE-M3 (:8100), Postgres (:5433)
  up (same as the rest of the stack); the distill step fails fast with a clear message if not.
- **Prompt drift.** Avoided by generating gold answers through the *production* Go prompt
  assembly (`cmd/distill`), not a re-implementation.

## 7. YAGNI (explicitly out)
- Two-tier model routing (separate Phase 2 roadmap item) — deferred.
- Hyperparameter sweeps — one sensible config; only re-tune if the gate fails.
- Simultaneous tuned+base serving beyond the LiteLLM fallback chain.
- Training on real logged chats — not available yet; the pipeline is built to re-run on them
  later (the flywheel), but this phase uses self-distilled synthetic data.

## 8. Done when
The pipeline runs end-to-end producing a fused tuned model; `training/eval.py` reports the
candidate **passing the §1 ship gate**; the tuned model is served via `mlx_lm.server` and
registered as the LiteLLM primary `sea-lion-9b` (Ollama GGUF fallback); the live chat path
returns grounded Taglish from the **short** prompt; full Go suite + `go vet` green;
`training/` pytest green; `master_plan.md` updated. If the gate fails, the deliverable is the
documented no-ship verdict + rollback (still a valid Phase-2 outcome).
