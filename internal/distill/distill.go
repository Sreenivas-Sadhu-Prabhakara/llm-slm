// Package distill shapes self-distillation training examples for the Taglish
// voice LoRA: the input carries the SHORT persona, the output is the gold answer
// generated with the FULL persona (prompt distillation). It reuses internal/prompt
// so the training inputs match production exactly.
package distill

import (
	"github.com/apolaki/solar-assistant/internal/prompt"
	"github.com/apolaki/solar-assistant/internal/retriever"
)

// RawExample is one distillation record (serialized to raw.jsonl).
type RawExample struct {
	Mode        string   `json:"mode"`
	Category    string   `json:"category"`
	Question    string   `json:"question"`
	SystemShort string   `json:"system_short"`
	User        string   `json:"user"`
	Gold        string   `json:"gold"`
	Sources     []string `json:"sources"`
}

// Build assembles a RawExample. The short persona is the distillation INPUT; gold
// is the answer produced by the caller with the FULL persona.
func Build(m prompt.Mode, category, question string, chunks []retriever.Chunk, gold string) RawExample {
	sysShort, user := prompt.AssembleForShort(m, question, chunks)
	titles := make([]string, 0, len(chunks))
	for _, c := range chunks {
		titles = append(titles, c.Title)
	}
	return RawExample{
		Mode:        m.Name,
		Category:    category,
		Question:    question,
		SystemShort: sysShort,
		User:        user,
		Gold:        gold,
		Sources:     titles,
	}
}
