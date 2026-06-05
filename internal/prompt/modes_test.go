package prompt

import (
	"strings"
	"testing"

	"github.com/apolaki/solar-assistant/internal/retriever"
)

func TestModeByNameDefaultsToCustomer(t *testing.T) {
	if ModeByName("").Name != "customer" {
		t.Fatal("empty mode should default to customer")
	}
	if ModeByName("nonsense").Name != "customer" {
		t.Fatal("unknown mode should default to customer")
	}
	if ModeByName("installer").Name != "installer" {
		t.Fatal("installer mode not resolved")
	}
	if ModeByName("buyer").Audience != "buyer" {
		t.Fatal("buyer mode audience wrong")
	}
}

func TestAssembleForUsesModePersona(t *testing.T) {
	chunks := []retriever.Chunk{{Title: "Spec", Content: "450W mono"}}

	bSys, _ := AssembleFor(Buyer, "alin ang pinakasulit?", chunks)
	if !strings.Contains(strings.ToLower(bSys), "buyer") && !strings.Contains(strings.ToLower(bSys), "bibili") {
		t.Fatalf("buyer persona missing buyer framing: %q", bSys)
	}

	iSys, _ := AssembleFor(Installer, "anong mounting torque?", chunks)
	if !strings.Contains(strings.ToLower(iSys), "installer") && !strings.Contains(strings.ToLower(iSys), "technical") {
		t.Fatalf("installer persona missing installer framing: %q", iSys)
	}

	// All modes keep the grounding instruction in the user prompt.
	_, user := AssembleFor(Installer, "q", chunks)
	if !strings.Contains(user, "450W mono") {
		t.Fatal("source content missing from user prompt")
	}
}

func TestAssembleStillCustomer(t *testing.T) {
	sys, _ := Assemble("magkano?", nil)
	if sys != Customer.System {
		t.Fatal("Assemble should use the customer persona")
	}
}
