"""Generate a synthetic Apolaki solar corpus as JSONL (one document per line).
Re-runnable and deterministic. Output: data/seed/corpus.jsonl
"""
import json, os, hashlib

OUT_DIR = os.path.join(os.path.dirname(__file__), "seed")
OUT = os.path.join(OUT_DIR, "corpus.jsonl")

DOCS = [
    {"title": "Net Metering sa Pilipinas (Meralco)", "source_type": "faq",
     "content": "Ang net metering ay nagbibigay-daan sa iyong i-export ang sobrang solar energy "
                "pabalik sa grid. Sa Meralco, ang export mo ay nakukunting credit sa susunod mong bill. "
                "Karaniwang aabot ng 20-30% ang bawas sa monthly bill para sa typical na bahay."},
    {"title": "ROI ng Residential Solar (₱)", "source_type": "faq",
     "content": "Para sa isang 5 kW system na nagkakahalaga ng ~₱300,000, kung nakakatipid ka ng "
                "₱4,000/buwan, ang payback period ay ~6 na taon. Pagkatapos noon, halos libre na ang kuryente "
                "mo sa araw. Ang panels ay may 25-year warranty kaya malaki ang long-term savings."},
    {"title": "Brownout Backup with Solar + Battery", "source_type": "faq",
     "content": "Kung gusto mo ng backup tuwing brownout, kailangan mo ng hybrid inverter at battery. "
                "Ang solar-only (grid-tie) ay hindi gumagana kapag may outage para sa safety. Ang battery "
                "ang nagbibigay ng power sa gabi at tuwing walang kuryente."},
    {"title": "Apolaki Panel Datasheet — AP-450W Mono", "source_type": "datasheet",
     "product": "AP-450W", "brand": "Apolaki",
     "content": "Apolaki AP-450W monocrystalline panel. Power: 450W. Efficiency: 21%. "
                "Dimensions: 1.9m x 1.1m. Operating temp: -40C to 85C. Warranty: 25 years performance."},
    {"title": "Inverter Error Codes — Apolaki Hybrid 5kW", "source_type": "datasheet",
     "product": "AP-INV-5K", "brand": "Apolaki",
     "content": "E01: grid voltage out of range — tawag sa installer. E02: over-temperature — i-check ang "
                "ventilation. E03: battery communication fail — i-check ang cables. F12: insulation fault — "
                "huwag i-reset, tumawag agad sa lisensyadong installer."},
    {"title": "Resolved Ticket — Mataas pa rin ang bill", "source_type": "ticket",
     "content": "Tanong: bakit mataas pa rin ang Meralco bill ko kahit may solar? Sagot: kadalasan dahil "
                "gabi ang peak usage mo (aircon, etc.) na hindi covered ng solar kung walang battery. Solusyon: "
                "ilipat ang mabigat na gamit sa araw, o magdagdag ng battery para sa gabi."},
    {"title": "Financing Options para sa Solar", "source_type": "faq",
     "content": "May mga bangko at in-house financing na nag-aalok ng 12-60 months para sa solar. "
                "Ang monthly amortization ay madalas mas mababa pa sa monthly savings sa kuryente, kaya "
                "cash-flow positive ka agad sa maraming kaso."},
]

def main():
    os.makedirs(OUT_DIR, exist_ok=True)
    with open(OUT, "w", encoding="utf-8") as f:
        for d in DOCS:
            d.setdefault("audience", "customer")
            d.setdefault("brand", "Apolaki")
            d.setdefault("language", "taglish")
            d.setdefault("product", None)
            d["content_hash"] = hashlib.sha256(d["content"].encode()).hexdigest()
            f.write(json.dumps(d, ensure_ascii=False) + "\n")
    print(f"wrote {len(DOCS)} docs -> {OUT}")

if __name__ == "__main__":
    main()
