// Package safety is the grounding + safety guardrail layer (PRD §5 step 7).
// It decides whether the assistant must escalate to a human (no good source)
// and whether a licensed-installer disclaimer should be appended.
package safety

import (
	"strings"

	"github.com/apolaki/solar-assistant/internal/retriever"
)

// RelevanceThreshold is the maximum cosine distance at which a retrieved chunk
// still counts as relevant. Chunks are scored as cosine distance (0 = identical,
// up to 2 = opposite); beyond this the retrieval is treated as "no good source".
const RelevanceThreshold = 0.6

// electricalTerms trigger a licensed-installer safety disclaimer.
var electricalTerms = []string{
	"wiring", "wire", "kable", "electrical", "kuryente connection", "breaker",
	"voltage", "grounding wire", "install", "instala", "mount", "rooftop",
	"circuit", "fuse", "konekta sa kuryente",
}

// Decision is the outcome of a safety check.
type Decision struct {
	Escalate   bool   // true => hand off to a human instead of answering from weak/absent sources
	Disclaimer string // non-empty => append this licensed-installer reminder
}

// Escalation is the canned Taglish hand-off message.
func Escalation() string {
	return "Pasensya na, wala akong sapat na impormasyon diyan. Ikokonekta kita sa isang " +
		"Apolaki specialist para masagot nang tama ang tanong mo. 🙏"
}

// Check evaluates retrieved chunks against question and returns the safety Decision.
func Check(chunks []retriever.Chunk, question string) Decision {
	d := Decision{}
	if !hasRelevant(chunks) {
		d.Escalate = true
		return d
	}
	if mentionsElectrical(question) {
		d.Disclaimer = "⚠️ Para sa wiring, electrical, o installation, kumonsulta muna sa " +
			"isang lisensyadong installer para sa kaligtasan."
	}
	return d
}

func hasRelevant(chunks []retriever.Chunk) bool {
	for _, c := range chunks {
		if c.Score <= RelevanceThreshold {
			return true
		}
	}
	return false
}

func mentionsElectrical(question string) bool {
	s := strings.ToLower(question)
	for _, t := range electricalTerms {
		if strings.Contains(s, t) {
			return true
		}
	}
	return false
}
