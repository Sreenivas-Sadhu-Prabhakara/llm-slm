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

    # --- Buyer-audience docs ---
    {"title": "Anong kW ang Bagay sa Bill Mo? (Sizing Guide)", "source_type": "faq",
     "audience": "buyer",
     "content": "Para makapili ng tamang system size, tingnan mo muna ang average monthly Meralco bill mo. "
                "Kung ~₱6,000-8,000/buwan ang bill mo, sapat na ang 5 kW system (mga 11 piraso ng AP-450W panels) "
                "na nagkakahalaga ng ~₱300,000 at makakatipid ka ng humigit-kumulang ₱4,000/buwan. "
                "Kung ~₱3,000-4,000/buwan lang, pwede na ang 3 kW. Kung ₱10,000 pataas, mag-isip ka ng 8-10 kW. "
                "Rule of thumb: bawat 1 kW ng panels ay kumukuha ng mga 6-7 sqm ng bubong na walang lilim."},
    {"title": "Total Cost of Ownership ng 5 kW System (₱)", "source_type": "faq",
     "audience": "buyer",
     "content": "Sa pagbili ng solar, huwag lang sticker price ang tingnan. Halimbawa sa 5 kW grid-tie system na "
                "~₱300,000 cash: kasama na dito ang AP-450W panels, ang AP-INV-5K inverter, mounting, at installation. "
                "Kung financing ang kukunin mo (12-60 months), ang amortization ay madalas mas mababa pa sa "
                "₱4,000/buwan mong savings, kaya cash-flow positive ka agad. Sa loob ng 25-year panel warranty, "
                "ang payback ay ~6 na taon at libre na halos ang kuryente sa araw pagkatapos. Kung gusto mo ng backup "
                "tuwing brownout, magdagdag ng battery na may hiwalay na gastos."},
    {"title": "Ano ang Hanapin Kapag Bibili ng Solar (Warranty Checklist)", "source_type": "faq",
     "audience": "buyer",
     "content": "Bago ka pumirma sa kontrata, i-tsek ang mga ito: (1) Panel performance warranty — ang Apolaki "
                "AP-450W ay may 25-year performance warranty at 21% efficiency. (2) Inverter warranty — ang AP-INV-5K "
                "hybrid inverter ay dapat may malinaw na warranty period at after-sales support sa Pilipinas. "
                "(3) Installer na lisensyado at may net metering experience sa Meralco. (4) Itemized quote na nakikita "
                "mo ang presyo ng panels, inverter, at labor — iwasan ang vague na 'package price'. (5) Maintenance at "
                "monitoring — tanungin kung paano mo makikita ang generation ng system mo."},

    # --- Installer-audience docs ---
    {"title": "Installation Spec — Apolaki AP-450W Panel Mounting", "source_type": "datasheet",
     "audience": "installer", "product": "AP-450W", "brand": "Apolaki",
     "content": "AP-450W monocrystalline module. Dimensions: 1.9m x 1.1m, weight ~23 kg. Frame: anodized aluminum, "
                "drainage holes dapat nakaharap pababa. Mounting: gumamit ng minimum 4 na clamp points sa long-edge rails; "
                "i-torque ang module clamp bolts sa 16-20 Nm (huwag sobrahan para hindi mabasag ang frame). "
                "Mag-iwan ng minimum 10mm gap sa pagitan ng modules para sa thermal expansion at 100mm clearance mula sa "
                "roof surface para sa airflow. Operating temp range: -40C to 85C. Iwasan ang anumang lilim sa cells — "
                "kahit partial shading ay malaki ang bawas sa string output."},
    {"title": "Commissioning Guide — Apolaki AP-INV-5K Hybrid", "source_type": "datasheet",
     "audience": "installer", "product": "AP-INV-5K", "brand": "Apolaki",
     "content": "AP-INV-5K hybrid inverter commissioning. PV input: i-confirm na ang open-circuit voltage (Voc) ng "
                "string ay nasa loob ng MPPT window at hindi lalampas sa max DC input voltage ng unit kahit sa pinakamalamig "
                "na umaga (gamitin ang -40C temperature coefficient sa pagkalkula). I-verify ang correct DC polarity bago "
                "i-energize. AC side: dapat tugma ang grid voltage at frequency; kapag E01 (grid voltage out of range), "
                "i-check ang AC wiring at grid connection. Para sa E03 (battery comms fail), tiyakin ang tamang RS485/CAN "
                "cable pinout sa battery. Sundin ang sequence: PV off, AC off, ikabit ang battery, AC on, tapos PV on."},
    {"title": "Safety at Code Reminders para sa Solar Installers", "source_type": "datasheet",
     "audience": "installer", "brand": "Apolaki",
     "content": "Safety reminders bago at habang nag-i-install: (1) DC side ay live tuwing may araw — gumamit ng "
                "DC-rated PPE at huwag kailanman i-disconnect ang DC connectors under load; gamitin ang DC isolator. "
                "(2) Mag-install ng DC at AC isolators na accessible para sa emergency shutdown ayon sa Philippine "
                "Electrical Code (PEC). (3) Proper grounding/earthing ng panel frames at mounting rails ay mandatory para "
                "sa lightning at fault protection. (4) F12 (insulation fault) sa AP-INV-5K — huwag i-reset; i-isolate at "
                "i-check ang DC insulation resistance bago tumawag ng senior technician. (5) Grid-tie inverter ay dapat "
                "may anti-islanding protection — kapag may grid outage, dapat awtomatikong mag-shutdown ang output para sa "
                "kaligtasan ng grid linemen."},
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
