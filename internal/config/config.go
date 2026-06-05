package config

import (
	"errors"
	"os"
)

type Config struct {
	DatabaseURL     string
	DatabaseURLTest string
	LiteLLMBaseURL  string
	LiteLLMAPIKey   string
	EmbedModel      string
	GenModel        string
	JWTSecret       string // ASSISTANT_JWT_SECRET; empty = permissive dev auth
}

func get(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// Load reads config from the environment, applying defaults.
func Load() (Config, error) {
	c := Config{
		DatabaseURL:     os.Getenv("DATABASE_URL"),
		DatabaseURLTest: os.Getenv("DATABASE_URL_TEST"),
		LiteLLMBaseURL:  get("LITELLM_BASE_URL", "http://localhost:4000/v1"),
		LiteLLMAPIKey:   get("LITELLM_API_KEY", "local"),
		EmbedModel:      get("EMBED_MODEL", "bge-m3"),
		GenModel:        get("GEN_MODEL", "sea-lion-9b"),
		JWTSecret:       os.Getenv("ASSISTANT_JWT_SECRET"),
	}
	if c.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}
	return c, nil
}
