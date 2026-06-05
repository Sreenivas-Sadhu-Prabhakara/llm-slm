package safety

import (
	"testing"

	"github.com/apolaki/solar-assistant/internal/retriever"
)

func TestEscalatesWhenNoChunks(t *testing.T) {
	d := Check(nil, "magkano ang solar?")
	if !d.Escalate {
		t.Fatal("expected escalation with no sources")
	}
}

func TestEscalatesWhenAllChunksAreFarApart(t *testing.T) {
	chunks := []retriever.Chunk{
		{Title: "Unrelated", Content: "x", Score: 0.92},
		{Title: "Also far", Content: "y", Score: 0.88},
	}
	d := Check(chunks, "magkano ang solar?")
	if !d.Escalate {
		t.Fatal("expected escalation when best chunk is beyond the relevance threshold")
	}
}

func TestGroundedQuestionDoesNotEscalate(t *testing.T) {
	chunks := []retriever.Chunk{{Title: "ROI", Content: "tipid", Score: 0.21}}
	d := Check(chunks, "magkano ang matitipid ko?")
	if d.Escalate {
		t.Fatal("did not expect escalation for a well-grounded question")
	}
	if d.Disclaimer != "" {
		t.Fatal("did not expect an installer disclaimer for a savings question")
	}
}

func TestElectricalQuestionGetsInstallerDisclaimer(t *testing.T) {
	chunks := []retriever.Chunk{{Title: "Install", Content: "...", Score: 0.2}}
	d := Check(chunks, "paano ko i-wiring ang inverter sa panel?")
	if d.Escalate {
		t.Fatal("grounded electrical question should not escalate")
	}
	if d.Disclaimer == "" {
		t.Fatal("expected a licensed-installer disclaimer for a wiring question")
	}
}
