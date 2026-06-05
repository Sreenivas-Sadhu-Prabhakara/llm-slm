// Package personalizer is the seam for PRD §5 step 4: fetching the logged-in
// user's own system specs/monitoring (from solar-service) so the assistant can
// say "your 5 kW system…". solar-service does not exist locally, so Phase 1
// ships only the interface and an inert Noop; a real HTTP client drops in later
// with no change at the call site.
package personalizer

import "context"

// Personalizer returns extra grounding context about a specific user, or "" if
// none is available. Errors are advisory: callers degrade to no personalization.
type Personalizer interface {
	Context(ctx context.Context, userID string) (string, error)
}

// Noop is the Phase 1 implementation: it never personalizes.
type Noop struct{}

// Context always returns no personalization.
func (Noop) Context(ctx context.Context, userID string) (string, error) { return "", nil }
