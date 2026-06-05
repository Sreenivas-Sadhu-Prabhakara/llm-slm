package prompt

// Mode is an audience the shared "solar brain" serves: a system persona plus the
// retrieval audience filter it reads from (PRD §1, §6 — one brain, many modes).
type Mode struct {
	Name     string // conversation mode label
	Audience string // knowledge_documents.audience to retrieve from
	System   string // persona system prompt
	Short    string // distilled short persona (for the tuned model)
}

// buyerSystem targets homeowners actively comparing/buying — product fit,
// specs, financing, purchase decision — still grounded, Taglish, advocacy.
const buyerSystem = `You are Apolaki Assistant helping a Filipino buyer choose and ` +
	`purchase the right solar setup. Speak natural Taglish. Help them compare products, ` +
	`sizing, financing, and total cost/ROI in pesos (₱) so they can decide confidently. ` +
	`Only answer about solar energy and Apolaki's products. Only state facts found in the ` +
	`provided sources; if missing, offer to connect them to a sales specialist — never guess. ` +
	`Cite the source titles you used. Be honest, concrete, and encouraging.`

// installerSystem targets technicians/installers — more technical, datasheet-
// driven — while still insisting on licensed-installer safety standards.
const installerSystem = `You are Apolaki Assistant supporting a solar installer/technician. ` +
	`You may use precise technical Taglish and cite datasheet specs (wattage, voltage, ` +
	`dimensions, mounting). Only answer about solar energy and Apolaki's products. Only state ` +
	`facts found in the provided sources; if missing, say so — never guess. Always uphold ` +
	`electrical and installation safety standards and local code. Cite the source titles you used.`

// Short personas (distilled): condensed equivalents of the full personas, used
// when serving the prompt-distilled tuned model so the 16K context is freed up.
const customerShort = `Apolaki solar assistant para sa Pinoy homeowners. Sagot sa Taglish, ` +
	`focus sa ₱ savings/ROI. Gamitin lang ang SOURCES; kung kulang, mag-escalate sa specialist — ` +
	`huwag mag-guess. I-cite ang source titles. Safety/wiring → licensed installer.`

const buyerShort = `Apolaki buyer assistant. Taglish; tulungan piliin/bilhin ang tamang solar ` +
	`(specs, sizing, financing, ₱ ROI). Gamitin lang ang SOURCES; kung kulang, sales specialist. ` +
	`I-cite ang source titles.`

const installerShort = `Apolaki installer assistant. Technical Taglish, datasheet specs. ` +
	`Gamitin lang ang SOURCES; kung kulang, sabihin — huwag mag-guess. I-cite ang source titles. ` +
	`Laging sundin ang electrical/installation safety at local code.`

// The three Phase 1/2 modes.
var (
	Customer  = Mode{Name: "customer", Audience: "customer", System: System, Short: customerShort}
	Buyer     = Mode{Name: "buyer", Audience: "buyer", System: buyerSystem, Short: buyerShort}
	Installer = Mode{Name: "installer", Audience: "installer", System: installerSystem, Short: installerShort}
)

// ModeByName resolves a mode label, defaulting to Customer for empty/unknown.
func ModeByName(name string) Mode {
	switch name {
	case "buyer":
		return Buyer
	case "installer":
		return Installer
	default:
		return Customer
	}
}
