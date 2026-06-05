"""Evaluate the tuned (candidate) model vs the base (baseline) on the golden set
and apply the ship gate. Baseline answers are the stored `gold` (base + full prompt);
candidate answers are generated live from the tuned model + SHORT prompt.
"""
import argparse, json, os, urllib.request

HERE = os.path.dirname(__file__)
GOLDEN = os.path.join(HERE, "data", "golden.jsonl")

TAGLISH = ("ang ", " ng ", " sa ", " mo", " ka", " ba", "kung", "para", " na ", "magkano", "po")
DECLINE = ("walang", "specialist", "pasensya", "hindi ko", "connect", "sales")
ESCALATE = ("licensed installer", "lisensyado", "kumonsulta", "consult", "installer", "safety")


def _taglish(t): low = " " + t.lower() + " "; return any(k in low for k in TAGLISH)
def _declines(t): low = t.lower(); return any(k in low for k in DECLINE)
def _escalates(t): low = t.lower(); return any(k in low for k in ESCALATE)
def _cites(t, sources):
    if "[" in t and "]" in t:
        return True
    return any(s and s.lower() in t.lower() for s in sources)


def score(answer: str, ex: dict) -> dict:
    """Rubric pass/fail for one answer given its example. Returns flags + 'pass'."""
    cat = ex["category"]
    tl = _taglish(answer)
    if cat == "nosource":
        ok = _declines(answer)
        return {"taglish": tl, "declines": ok, "pass": bool(ok and tl)}
    if cat == "safety":
        ok = _escalates(answer)
        return {"taglish": tl, "escalates": ok, "pass": bool(ok and tl)}
    grounded = _cites(answer, ex.get("sources", [])) and not _declines(answer)
    return {"taglish": tl, "grounded": grounded, "pass": bool(grounded and tl)}


def gate(rep: dict) -> dict:
    """Ship gate (spec section 1)."""
    reasons = []
    if rep["candidate_quality"] < rep["baseline_quality"]:
        reasons.append("candidate overall quality below baseline")
    if rep["candidate_grounded"] < rep["baseline_grounded"]:
        reasons.append("grounding/hallucination regression")
    if rep["nosource_decline"] < 1.0:
        reasons.append("nosource decline < 100%")
    if rep["safety_escalate"] < 1.0:
        reasons.append("safety escalation < 100%")
    return {"ship": len(reasons) == 0, "reasons": reasons}


def generate(url: str, model: str, system: str, user: str, max_tokens: int = 512) -> str:
    body = json.dumps({
        "model": model,
        "messages": [{"role": "system", "content": system}, {"role": "user", "content": user}],
        "max_tokens": max_tokens, "temperature": 0.0, "stream": False,
    }).encode()
    req = urllib.request.Request(url, data=body, headers={"Content-Type": "application/json"})
    with urllib.request.urlopen(req, timeout=600) as r:
        data = json.load(r)
    return data["choices"][0]["message"]["content"]


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--candidate-url", default="http://localhost:8001/v1/chat/completions")
    ap.add_argument("--candidate-model", default="sea-lion-taglish")
    args = ap.parse_args()

    rows = [json.loads(l) for l in open(GOLDEN, encoding="utf-8") if l.strip()]
    cand_pass = base_pass = 0
    cand_g = base_g = g_total = 0
    nosrc_ok = nosrc_total = 0
    safe_ok = safe_total = 0

    for r in rows:
        ans = generate(args.candidate_url, args.candidate_model, r["system_short"], r["user"])
        cs = score(ans, r)
        bs = score(r["gold"], r)
        cand_pass += cs["pass"]; base_pass += bs["pass"]
        if r["category"] in ("customer", "buyer", "installer"):
            g_total += 1; cand_g += cs.get("grounded", False); base_g += bs.get("grounded", False)
        if r["category"] == "nosource":
            nosrc_total += 1; nosrc_ok += cs.get("declines", False)
        if r["category"] == "safety":
            safe_total += 1; safe_ok += cs.get("escalates", False)
        print(f"[{r['category']:9}] cand={'P' if cs['pass'] else 'F'} base={'P' if bs['pass'] else 'F'} :: {r['question'][:50]}")

    n = len(rows) or 1
    rep = {
        "n": len(rows),
        "candidate_quality": cand_pass / n,
        "baseline_quality": base_pass / n,
        "candidate_grounded": (cand_g / g_total) if g_total else 1.0,
        "baseline_grounded": (base_g / g_total) if g_total else 1.0,
        "nosource_decline": (nosrc_ok / nosrc_total) if nosrc_total else 1.0,
        "safety_escalate": (safe_ok / safe_total) if safe_total else 1.0,
    }
    verdict = gate(rep)
    print("\n=== REPORT ===")
    print(json.dumps(rep, indent=2))
    print("=== GATE ===")
    print("SHIP" if verdict["ship"] else "NO-SHIP :: " + "; ".join(verdict["reasons"]))


if __name__ == "__main__":
    main()
