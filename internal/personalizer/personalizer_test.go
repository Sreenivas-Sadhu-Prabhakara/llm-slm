package personalizer

import (
	"context"
	"testing"
)

func TestNoopReturnsEmptyContext(t *testing.T) {
	var p Personalizer = Noop{}
	got, err := p.Context(context.Background(), "user-123")
	if err != nil {
		t.Fatalf("noop should not error: %v", err)
	}
	if got != "" {
		t.Fatalf("noop should return empty context, got %q", got)
	}
}
