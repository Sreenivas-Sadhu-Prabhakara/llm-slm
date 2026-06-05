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
    # grounded categories: must cite a source and not be a blanket refusal.
    return has_citation(ex) and not declines(gold)


def to_message(ex: dict) -> dict:
    return {"messages": [
        {"role": "system", "content": ex["system_short"]},
        {"role": "user", "content": ex["user"]},
        {"role": "assistant", "content": ex["gold"]},
    ]}


def _bucket(row: dict) -> int:
    h = hashlib.sha256(row["question"].encode()).hexdigest()
    return int(h[:8], 16) % 10  # 0-9


def split(rows: list) -> dict:
    """Deterministic 80/10/10 by hash bucket of the question."""
    train, valid, test = [], [], []
    for r in rows:
        b = _bucket(r)
        (test if b == 0 else valid if b == 1 else train).append(r)
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
