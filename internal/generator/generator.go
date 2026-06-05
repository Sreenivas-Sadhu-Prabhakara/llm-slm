package generator

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Client streams a chat completion from an OpenAI-compatible endpoint (LiteLLM).
type Client interface {
	// Stream posts a (system, user) chat and invokes onToken for each content
	// delta as it arrives, returning once the stream reaches [DONE].
	Stream(ctx context.Context, system, user string, onToken func(string)) error
}

type httpClient struct {
	baseURL, apiKey, model string
	hc                     *http.Client
}

// New returns a Client that posts to baseURL+"/chat/completions" using the
// given API key and model (e.g. "sea-lion-9b"). The 10-minute timeout covers
// slow local GGUF generation behind LiteLLM's MLX→Ollama fallback.
func New(baseURL, apiKey, model string) Client {
	return &httpClient{baseURL: baseURL, apiKey: apiKey, model: model,
		hc: &http.Client{Timeout: 10 * time.Minute}}
}

func (c *httpClient) Stream(ctx context.Context, system, user string, onToken func(string)) error {
	body, err := json.Marshal(map[string]any{
		"model":  c.model,
		"stream": true,
		"messages": []map[string]string{
			{"role": "system", "content": system},
			{"role": "user", "content": user},
		},
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	resp, err := c.hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("chat status %d", resp.StatusCode)
	}
	sc := bufio.NewScanner(resp.Body)
	// SSE data lines can exceed bufio's default 64KB cap; allow up to 1MB.
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(line[len("data:"):])
		if payload == "[DONE]" {
			break
		}
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			continue // skip keep-alives / non-JSON frames
		}
		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
			onToken(chunk.Choices[0].Delta.Content)
		}
	}
	return sc.Err()
}
