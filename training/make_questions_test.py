import json, subprocess, sys, pathlib

ROOT = pathlib.Path(__file__).parent


def test_generates_balanced_bank():
    subprocess.run([sys.executable, str(ROOT / "make_questions.py")], check=True)
    lines = (ROOT / "questions.jsonl").read_text(encoding="utf-8").splitlines()
    rows = [json.loads(l) for l in lines]
    assert len(rows) >= 70, f"want >=70 questions, got {len(rows)}"
    cats = {}
    for r in rows:
        assert set(r) == {"category", "mode", "question"}, r
        assert r["mode"] in {"customer", "buyer", "installer"}
        cats[r["category"]] = cats.get(r["category"], 0) + 1
    for c in ("customer", "buyer", "installer", "nosource", "safety"):
        assert cats.get(c, 0) >= 12, f"category {c} underrepresented: {cats}"


def test_deterministic():
    subprocess.run([sys.executable, str(ROOT / "make_questions.py")], check=True)
    a = (ROOT / "questions.jsonl").read_bytes()
    subprocess.run([sys.executable, str(ROOT / "make_questions.py")], check=True)
    b = (ROOT / "questions.jsonl").read_bytes()
    assert a == b, "question bank must be deterministic"
