"""Curate self-distilled raw examples into MLX LoRA training data.
Reads training/data/raw.jsonl, applies per-category quality filters, and writes
train/valid/test.jsonl (MLX chat 'messages' format) plus golden.jsonl (rich, for eval).
"""
import json, os, hashlib

HERE = os.path.dirname(__file__)
RAW = os.path.join(HERE, "data", "raw.jsonl")
DATA = os.path.join(HERE, "data")

TAGLISH = ("ang ", " ng ", " sa ", " mo", " ka", " ba", "kung", "para", " na ", "magkano", "po")
DECLINE = ("walang", "specialist", "pasensya", "hindi ko", "connect", "sales", "hindi covered")
ESCALATE = ("licensed installer", "lisensyado", "kumonsulta", "consult", "installer", "safety", "electrician")


def is_taglish(text: str) -> bool:
    low = " " + text.lower() + " "
    return any(k in low for k in TAGLISH)


def declines(text: str) -> bool:
    low = text.lower()
    return any(k in low for k in DECLINE)


def escalates(text: str) -> bool:
    low = text.lower()
    return any(k in low for k in ESCALATE)


def has_citation(ex: dict) -> bool:
    g = ex["gold"]
    if "[" in g and "]" in g:
        return True
    return any(t and t.lower() in g.lower() for t in ex.get("sources", []))


def _len_ok(text: str) -> bool:
    n = len(text.split())
    return 4 <= n <= 400


def keep(ex: dict) -> bool:
    gold = ex.get("gold", "")
    if not gold or not _len_ok(gold) or not is_taglish(gold):
        return False
    cat = ex["category"]
    if cat == "nosource":
        return declines(gold)
    if cat == "safety":
        return escalates(gold)
    # grounded categories: keep iff it cited a source (i.e. engaged with the
    # retrieved content). Genuine "no info" answers don't cite and are dropped.
    # We do NOT also reject on decline words — buyer answers legitimately mention
    # "sales specialist"/"connect" as a CTA while still answering + citing.
    return has_citation(ex)


def to_message(ex: dict) -> dict:
    return {"messages": [
        {"role": "system", "content": ex["system_short"]},
        {"role": "user", "content": ex["user"]},
        {"role": "assistant", "content": ex["gold"]},
    ]}


def split(rows: list) -> dict:
    """Deterministic, category-stratified ~80/10/10 split. Within each category,
    items are ordered by question hash; index%10==0 -> test, ==1 -> valid, else
    train. Stratifying guarantees non-empty valid/test with every category
    represented in the golden (test) set even for small datasets."""
    groups: dict = {}
    for r in rows:
        groups.setdefault(r.get("category", ""), []).append(r)
    train, valid, test = [], [], []
    for cat in sorted(groups):
        items = sorted(groups[cat], key=lambda r: hashlib.sha256(r["question"].encode()).hexdigest())
        for i, r in enumerate(items):
            (test if i % 10 == 0 else valid if i % 10 == 1 else train).append(r)
    return {"train": train, "valid": valid, "test": test}


def main():
    rows = [json.loads(l) for l in open(RAW, encoding="utf-8") if l.strip()]
    kept = [r for r in rows if keep(r)]
    print(f"kept {len(kept)}/{len(rows)} examples")
    parts = split(kept)
    os.makedirs(DATA, exist_ok=True)
    for name in ("train", "valid", "test"):
        with open(os.path.join(DATA, f"{name}.jsonl"), "w", encoding="utf-8") as f:
            for r in parts[name]:
                f.write(json.dumps(to_message(r), ensure_ascii=False) + "\n")
    # Rich golden = the test split with all fields (used by eval.py).
    with open(os.path.join(DATA, "golden.jsonl"), "w", encoding="utf-8") as f:
        for r in parts["test"]:
            f.write(json.dumps(r, ensure_ascii=False) + "\n")
    print(f"train={len(parts['train'])} valid={len(parts['valid'])} test={len(parts['test'])}")


if __name__ == "__main__":
    main()
