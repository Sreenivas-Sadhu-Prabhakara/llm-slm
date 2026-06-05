package embed

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEmbedReturns1024Vector(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/embeddings" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data":[{"embedding":[`))
		// build 1024 floats with first=0.5, rest 0
		w.Write([]byte("0.5"))
		for i := 1; i < 1024; i++ {
			w.Write([]byte(",0"))
		}
		w.Write([]byte(`]}]}`))
	}))
	defer srv.Close()

	c := New(srv.URL, "k", "bge-m3")
	v, err := c.Embed(context.Background(), "magkano")
	if err != nil {
		t.Fatalf("embed: %v", err)
	}
	if len(v) != 1024 {
		t.Fatalf("dim = %d, want 1024", len(v))
	}
	if v[0] != 0.5 {
		t.Fatalf("v[0] = %f", v[0])
	}
}
