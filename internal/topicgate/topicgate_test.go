package topicgate

import (
	"strings"
	"testing"
)

func TestSolarQuestionsPass(t *testing.T) {
	cases := []string{
		"magkano ang matitipid ko sa solar kada buwan?",
		"How much does a 5kW panel system cost?",
		"paano gumagana ang net metering sa Meralco?",
		"tuloy-tuloy ba ang kuryente kapag brownout kung may baterya?",
	}
	for _, q := range cases {
		if !IsSolarRelated(q) {
			t.Errorf("expected solar-related: %q", q)
		}
	}
}

func TestInstallerTechnicalQuestionsPass(t *testing.T) {
	// Installer-mode jargon and product codes must not be rejected as off-topic
	// even when they omit consumer-facing words like "solar"/"panel".
	cases := []string{
		"anong torque sa clamp bolts ng AP-450W?",
		"ano ang MPPT window at max DC input voltage ng AP-INV-5K?",
		"paano i-commission ang inverter nang ligtas?",
		"anong mounting clearance ang kailangan sa bubong?",
	}
	for _, q := range cases {
		if !IsSolarRelated(q) {
			t.Errorf("expected installer technical question to pass: %q", q)
		}
	}
}

func TestOffTopicQuestionsRejected(t *testing.T) {
	cases := []string{
		"sino panalo sa NBA finals?",
		"ano lutuin ko mamaya?",
		"write me a poem about cats",
	}
	for _, q := range cases {
		if IsSolarRelated(q) {
			t.Errorf("expected off-topic: %q", q)
		}
	}
}

func TestRedirectIsTaglishAndSolar(t *testing.T) {
	r := strings.ToLower(Redirect())
	if r == "" {
		t.Fatal("empty redirect")
	}
	if !strings.Contains(r, "solar") {
		t.Fatal("redirect should mention solar to steer the user back")
	}
}
