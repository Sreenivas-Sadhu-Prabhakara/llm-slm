package embed

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Client embeds text via an OpenAI-compatible /v1/embeddings endpoint (LiteLLM).
type Client interface {
	Embed(ctx context.Context, text string) ([]float64, error)
}

type httpClient struct {
	baseURL, apiKey, model string
	hc                     *http.Client
}

// New returns a Client that posts to baseURL+"/embeddings" using the given
// API key and model (e.g. "bge-m3").
func New(baseURL, apiKey, model string) Client {
	return &httpClient{baseURL: baseURL, apiKey: apiKey, model: model,
		hc: &http.Client{Timeout: 30 * time.Second}}
}

func (c *httpClient) Embed(ctx context.Context, text string) ([]float64, error) {
	body, err := json.Marshal(map[string]any{"model": c.model, "input": text})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("embeddings status %d", resp.StatusCode)
	}
	var out struct {
		Data []struct {
			Embedding []float64 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if len(out.Data) == 0 {
		return nil, fmt.Errorf("no embedding returned")
	}
	return out.Data[0].Embedding, nil
}
