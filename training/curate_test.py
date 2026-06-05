import importlib.util, pathlib

spec = importlib.util.spec_from_file_location("curate", pathlib.Path(__file__).parent / "curate.py")
curate = importlib.util.module_from_spec(spec)
spec.loader.exec_module(curate)


def test_is_taglish():
    assert curate.is_taglish("magkano ang matitipid mo sa solar")
    assert not curate.is_taglish("zzzz qqqq")


def test_declines_detects_refusal():
    assert curate.declines("Pasensya, solar lang ang kaya kong sagutin, walang source dito.")
    assert not curate.declines("Ang torque ay 16-20 Nm.")


def test_escalates_detects_safety():
    assert curate.escalates("Mag-consult sa licensed installer para sa wiring na ito.")
    assert not curate.escalates("Ang savings mo ay ₱4,000 kada buwan.")


def test_keep_routes_by_category():
    grounded = {"category": "installer", "gold": "Ang torque ay 16-20 Nm. (Source: [1] Mounting)",
                "sources": ["Mounting"], "system_short": "x", "user": "u"}
    assert curate.keep(grounded)

    nosrc_good = {"category": "nosource", "gold": "Pasensya, walang source — connect kita sa specialist.",
                  "sources": [], "system_short": "x", "user": "u"}
    nosrc_bad = {"category": "nosource", "gold": "Ang kapital ng France ay Paris.",
                 "sources": [], "system_short": "x", "user": "u"}
    assert curate.keep(nosrc_good)
    assert not curate.keep(nosrc_bad)

    safety_good = {"category": "safety", "gold": "Para sa wiring, kumonsulta sa licensed installer.",
                   "sources": [], "system_short": "x", "user": "u"}
    safety_bad = {"category": "safety", "gold": "Sige, i-connect mo ang red wire sa breaker.",
                  "sources": [], "system_short": "x", "user": "u"}
    assert curate.keep(safety_good)
    assert not curate.keep(safety_bad)


def test_to_message_uses_short_system():
    ex = {"system_short": "SHORT", "user": "U", "gold": "G"}
    m = curate.to_message(ex)
    assert m["messages"][0] == {"role": "system", "content": "SHORT"}
    assert m["messages"][1]["role"] == "user" and m["messages"][1]["content"] == "U"
    assert m["messages"][2] == {"role": "assistant", "content": "G"}


def test_split_is_deterministic_and_disjoint():
    rows = [{"question": f"q{i}", "x": i} for i in range(100)]
    a = curate.split(rows)
    b = curate.split(rows)
    assert a == b
    tr, va, te = a["train"], a["valid"], a["test"]
    keys = lambda rs: {r["question"] for r in rs}
    assert keys(tr) & keys(va) == set() and keys(tr) & keys(te) == set() and keys(va) & keys(te) == set()
    assert len(tr) + len(va) + len(te) == 100
