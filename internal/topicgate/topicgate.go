// Package topicgate is the first guardrail layer: a cheap, LLM-free check that
// a message is solar-related so off-topic queries are politely redirected
// without paying for a generation call (PRD §5 step 2).
package topicgate

import "strings"

// keywords are lower-cased EN + Filipino/Taglish solar terms. A message is
// considered solar-related if it contains any of them as a substring.
var keywords = []string{
	"solar", "panel", "baterya", "battery", "inverter", "kuryente",
	"electric", "meralco", "net metering", "netmetering", "metering",
	"roi", "tipid", "matitipid", "savings", "renewable", "sikat ng araw",
	"sunlight", "off-grid", "off grid", "on-grid", "brownout", "watt",
	"kw", "kwh", "kilowatt", "apolaki", "installer", "installation",
	"instala", "wiring", "rooftop", "bubong", "enerhiya", "photovoltaic",
	"financing", "hulugan",
	// installer/buyer technical + product vocabulary (modes beyond customer):
	"torque", "clamp", "mounting", "mppt", "voltage", "voltahe", "ampere",
	"commission", "datasheet", "warranty", "ap-450", "ap-inv",
}

// IsSolarRelated reports whether q plausibly concerns solar energy or Apolaki.
func IsSolarRelated(q string) bool {
	s := strings.ToLower(q)
	for _, kw := range keywords {
		if strings.Contains(s, kw) {
			return true
		}
	}
	return false
}

// Redirect is the canned, warm Taglish reply for off-topic messages.
func Redirect() string {
	return "Pasensya na, solar energy at Apolaki products lang ang kaya kong tulungan. " +
		"May tanong ka ba tungkol sa solar panels, savings, o net metering? Tutulungan kita! ☀️"
}
