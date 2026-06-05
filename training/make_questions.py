"""Generate a deterministic Taglish question bank for self-distillation.
Categories: customer/buyer/installer (grounded), nosource (must decline),
safety (must escalate). Output: training/questions.jsonl  ({category,mode,question}).
"""
import json, os

OUT = os.path.join(os.path.dirname(__file__), "questions.jsonl")

# (category, mode, [questions]) — concrete, paraphrased for diversity.
CUSTOMER = [
    "magkano ang matitipid ko kada buwan sa {kw} kW solar?",
    "paano gumagana ang net metering sa Meralco?",
    "worth it ba ang solar kung {bill} ang bill ko kada buwan?",
    "ano ang payback period ng solar para sa bahay?",
    "tuloy ba ang kuryente kapag brownout kung may baterya?",
    "ilang taon ang warranty ng solar panels?",
    "bakit mataas pa rin ang Meralco bill kahit may solar?",
    "kailangan ko ba ng baterya o grid-tie lang?",
    "gaano kabilis ang installation ng solar sa bahay?",
    "makakatipid ba talaga ako sa solar sa {bill} bill?",
]
BUYER = [
    "anong kW system ang bagay sa {bill} na bill?",
    "magkano ang total cost ng {kw} kW system?",
    "may financing ba o hulugan para sa solar?",
    "ano ang dapat hanapin sa warranty bago bumili?",
    "ilang panel ang kailangan para sa {kw} kW?",
    "alin ang mas sulit, mas malaking system o baterya?",
    "ano ang kasama sa quote ng isang installation?",
    "anong brand ng inverter ang inirerekomenda niyo?",
    "magkano ang downpayment para sa {kw} kW system?",
]
INSTALLER = [
    "anong torque sa clamp bolts ng AP-450W?",
    "anong mounting clearance at gap ang kailangan ng AP-450W?",
    "ano ang MPPT window at max DC input voltage ng AP-INV-5K?",
    "paano i-commission ang AP-INV-5K nang ligtas?",
    "ano ang gagawin sa E01 error ng inverter?",
    "ano ang sequence sa pag-energize ng hybrid inverter?",
    "anong PEC code requirements sa grounding ng panel frames?",
    "anong gauge ng DC cable ang gamitin sa AP-450W string?",
    "ilang panel ang pwede sa isang MPPT string ng AP-INV-5K?",
    "anong AWG ng grounding conductor para sa AP-450W array?",
    "paano i-set ang anti-islanding sa AP-INV-5K?",
    "ano ang max series fuse rating ng AP-450W?",
    "anong tilt angle ang optimal para sa Metro Manila?",
    "paano i-troubleshoot ang E03 battery comms fail?",
]
NOSOURCE = [  # off-topic / out-of-KB → model must decline, not invent
    "sino ang panalo sa NBA finals?",
    "ano ang recipe ng adobo?",
    "magkano ang bitcoin ngayon?",
    "ano ang kapital ng France?",
    "pwede mo ba akong tulungan sa math homework?",
    "ano ang lagay ng panahon bukas?",
    "sino ang pangulo ng Pilipinas noong 1990?",
    "ano ang score ng Ginebra kahapon?",
    "ano ang plot ng latest Marvel movie?",
    "paano gumawa ng website?",
    "magkano ang ticket papuntang Japan?",
    "ano ang meaning ng panaginip ko kagabi?",
]
SAFETY = [  # wiring/electrical → must escalate to licensed installer
    "paano mag-wiring ng solar panel sa bahay ko mismo?",
    "pwede ko bang i-connect mismo ang inverter sa main breaker?",
    "paano ko aayusin ang sarili kong solar electrical fault?",
    "safe bang ako mismo mag-install ng DC isolator?",
    "paano ko i-bypass ang insulation fault sa inverter?",
    "pwede ko bang palitan ang breaker ng solar ko mag-isa?",
    "pwede ko bang gamitin ang regular na wire sa DC side?",
    "okay lang bang ako mismo umakyat at mag-mount ng panel?",
    "paano ko i-disconnect ang live DC connector mag-isa?",
    "kailangan ko ba ng permit para mag-wire mismo?",
    "pwede ko bang i-ground sa tubo ng tubig ang panel?",
    "safe bang i-reset ko ang insulation fault mag-isa?",
]

KW = ["3", "5", "8", "10"]
BILL = ["₱3,000", "₱4,000", "₱6,000", "₱10,000"]


def expand(templates):
    out = []
    for t in templates:
        if "{kw}" in t:
            for kw in KW:
                out.append(t.replace("{kw}", kw))
        elif "{bill}" in t:
            for b in BILL:
                out.append(t.replace("{bill}", b))
        else:
            out.append(t)
    return out


def main():
    rows = []
    for cat, mode, tmpls in [
        ("customer", "customer", CUSTOMER),
        ("buyer", "buyer", BUYER),
        ("installer", "installer", INSTALLER),
        ("nosource", "customer", NOSOURCE),
        ("safety", "customer", SAFETY),
    ]:
        for q in expand(tmpls):
            rows.append({"category": cat, "mode": mode, "question": q})
    # Deterministic order, no shuffling.
    with open(OUT, "w", encoding="utf-8") as f:
        for r in rows:
            f.write(json.dumps(r, ensure_ascii=False) + "\n")
    print(f"wrote {len(rows)} questions -> {OUT}")


if __name__ == "__main__":
    main()
