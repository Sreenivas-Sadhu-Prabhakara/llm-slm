package distill

import (
	"testing"

	"github.com/apolaki/solar-assistant/internal/prompt"
	"github.com/apolaki/solar-assistant/internal/retriever"
)

func TestBuildShapesExample(t *testing.T) {
	chunks := []retriever.Chunk{
		{Title: "AP-450W Spec", Content: "450W mono"},
		{Title: "Mounting", Content: "16-20 Nm"},
	}
	ex := Build(prompt.Installer, "installer", "anong torque?", chunks, "16-20 Nm po.")

	if ex.Mode != "installer" || ex.Category != "installer" {
		t.Fatalf("mode/category wrong: %+v", ex)
	}
	if ex.Gold != "16-20 Nm po." {
		t.Fatalf("gold not carried through: %q", ex.Gold)
	}
	// Input system must be the SHORT persona (distillation target).
	if ex.SystemShort != prompt.Installer.Short {
		t.Fatalf("SystemShort should be the short persona, got %q", ex.SystemShort)
	}
	if len(ex.Sources) != 2 || ex.Sources[0] != "AP-450W Spec" {
		t.Fatalf("sources not collected: %+v", ex.Sources)
	}
}
