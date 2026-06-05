package generator

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestStreamConcatenatesDeltas(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Write([]byte(`data: {"choices":[{"delta":{"content":"Pwede"}}]}` + "\n\n"))
		w.Write([]byte(`data: {"choices":[{"delta":{"content":" mag-solar"}}]}` + "\n\n"))
		w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer srv.Close()

	c := New(srv.URL, "k", "sea-lion-9b")
	var got strings.Builder
	err := c.Stream(context.Background(), "sys", "user", func(tok string) {
		got.WriteString(tok)
	})
	if err != nil {
		t.Fatalf("stream: %v", err)
	}
	if got.String() != "Pwede mag-solar" {
		t.Fatalf("got %q", got.String())
	}
}
