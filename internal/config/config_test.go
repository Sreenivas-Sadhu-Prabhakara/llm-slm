package config

import "testing"

func TestLoadRequiresDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	if _, err := Load(); err == nil {
		t.Fatal("expected error when DATABASE_URL is empty")
	}
}

func TestLoadDefaultsAndValues(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("LITELLM_BASE_URL", "")
	t.Setenv("EMBED_MODEL", "")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.DatabaseURL != "postgres://x" {
		t.Fatalf("DatabaseURL = %q", cfg.DatabaseURL)
	}
	if cfg.LiteLLMBaseURL != "http://localhost:4000/v1" {
		t.Fatalf("default LiteLLMBaseURL wrong: %q", cfg.LiteLLMBaseURL)
	}
	if cfg.EmbedModel != "bge-m3" {
		t.Fatalf("default EmbedModel wrong: %q", cfg.EmbedModel)
	}
}
