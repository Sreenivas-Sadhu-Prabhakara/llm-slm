package prompt

import (
	"strings"
	"testing"

	"github.com/apolaki/solar-assistant/internal/retriever"
)

func TestAssembleIncludesSourcesAndQuestion(t *testing.T) {
	chunks := []retriever.Chunk{
		{Title: "Net Metering", Content: "export credit sa Meralco"},
	}
	sys, user := Assemble("magkano savings?", chunks)
	if !strings.Contains(sys, "Taglish") {
		t.Fatal("system prompt missing persona")
	}
	if !strings.Contains(user, "export credit sa Meralco") {
		t.Fatal("user prompt missing source content")
	}
	if !strings.Contains(user, "magkano savings?") {
		t.Fatal("user prompt missing the question")
	}
}

func TestAssembleNoSourcesSignalsEscalation(t *testing.T) {
	_, user := Assemble("random", nil)
	if !strings.Contains(strings.ToLower(user), "walang") && !strings.Contains(user, "no sources") {
		t.Fatal("expected an explicit no-sources signal for grounding")
	}
}
