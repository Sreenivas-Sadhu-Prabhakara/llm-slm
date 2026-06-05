package ingest

import "testing"

func TestChunkSplitsLongTextWithOverlap(t *testing.T) {
	words := make([]string, 250)
	for i := range words {
		words[i] = "w"
	}
	text := join(words, " ")
	chunks := Chunk(text, 100, 20) // size=100 words, overlap=20
	if len(chunks) < 2 {
		t.Fatalf("expected >=2 chunks, got %d", len(chunks))
	}
}

func TestChunkShortTextSingleChunk(t *testing.T) {
	if got := Chunk("maikli lang ito", 100, 20); len(got) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(got))
	}
}
