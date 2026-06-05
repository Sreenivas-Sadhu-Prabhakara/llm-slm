package prompt

import (
	"fmt"
	"strings"

	"github.com/apolaki/solar-assistant/internal/retriever"
)

// System is the Taglish advocate persona (PRD Appendix A, condensed).
const System = `You are Apolaki Assistant, a warm, encouraging solar-energy guide for ` +
	`Filipino homeowners. Speak natural Taglish. Emphasize savings, ROI in pesos (₱), ` +
	`and energy independence; avoid heavy jargon. Only answer questions about solar energy ` +
	`and Apolaki's products. Only state facts found in the provided sources; if the sources ` +
	`don't cover it, say you'll connect them to a specialist — never guess. For wiring, ` +
	`electrical, or installation-safety topics, remind them to consult a licensed installer. ` +
	`Cite source titles you used. Be kind, clear, and motivating.`

// Assemble builds (systemPrompt, userPrompt) from the question and retrieved chunks.
func Assemble(question string, chunks []retriever.Chunk) (string, string) {
	var b strings.Builder
	if len(chunks) == 0 {
		b.WriteString("SOURCES: (walang nahanap na source / no sources found)\n\n")
	} else {
		b.WriteString("SOURCES:\n")
		for i, c := range chunks {
			fmt.Fprintf(&b, "[%d] %s: %s\n", i+1, c.Title, c.Content)
		}
		b.WriteString("\n")
	}
	fmt.Fprintf(&b, "QUESTION: %s\n", question)
	b.WriteString("\nSagutin gamit lang ang SOURCES sa itaas. Kung kulang, mag-escalate.")
	return System, b.String()
}
