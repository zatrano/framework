package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// OpenAIDriver calls an OpenAI-compatible chat/embeddings HTTP API.
type OpenAIDriver struct {
	BaseURL    string
	APIKey     string
	Model      string
	HTTPClient *http.Client
	name       string
}

// OpenAI returns an OpenAIDriver with the default API base URL.
func OpenAI(apiKey string) Driver {
	return &OpenAIDriver{
		BaseURL: "https://api.openai.com/v1",
		APIKey:  apiKey,
		Model:   "gpt-4o-mini",
		name:    "openai",
	}
}

// OpenAICompatible returns an OpenAI-protocol driver for Ollama, OpenRouter, Azure proxies, etc.
func OpenAICompatible(baseURL, apiKey, model string) Driver {
	if model == "" {
		model = "gpt-4o-mini"
	}
	return &OpenAIDriver{
		BaseURL: baseURL,
		APIKey:  apiKey,
		Model:   model,
		name:    "openai_compatible",
	}
}

func (d *OpenAIDriver) Name() string {
	if d != nil && d.name != "" {
		return d.name
	}
	return "openai"
}

func (d *OpenAIDriver) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	base := strings.TrimRight(d.BaseURL, "/")
	if base == "" {
		base = "https://api.openai.com/v1"
	}
	model := req.Model
	if model == "" || model == "zatrano-fake-1" {
		model = d.Model
	}
	if model == "" {
		model = "gpt-4o-mini"
	}

	body := map[string]any{
		"model":    model,
		"messages": req.Messages,
	}
	if req.Temperature != nil {
		body["temperature"] = *req.Temperature
	}
	if req.MaxTokens > 0 {
		body["max_tokens"] = req.MaxTokens
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	client := d.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/chat/completions", bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if d.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+d.APIKey)
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("ai: openai status %d: %s", resp.StatusCode, strings.TrimSpace(string(payload)))
	}

	return parseOpenAIChatResponse(payload, model)
}

func (d *OpenAIDriver) Embed(ctx context.Context, req EmbedRequest) (*EmbedResponse, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	base := strings.TrimRight(d.BaseURL, "/")
	if base == "" {
		base = "https://api.openai.com/v1"
	}
	model := req.Model
	if model == "" || model == "zatrano-fake-1" {
		model = d.Model
	}
	if model == "" {
		model = "text-embedding-3-small"
	}
	body, err := json.Marshal(map[string]any{
		"model": model,
		"input": req.Input,
	})
	if err != nil {
		return nil, err
	}
	client := d.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if d.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+d.APIKey)
	}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("ai: openai embeddings status %d: %s", resp.StatusCode, strings.TrimSpace(string(payload)))
	}
	var parsed struct {
		Model string `json:"model"`
		Data  []struct {
			Embedding []float64 `json:"embedding"`
		} `json:"data"`
		Usage Usage `json:"usage"`
	}
	if err := json.Unmarshal(payload, &parsed); err != nil {
		return nil, err
	}
	out := make([][]float64, 0, len(parsed.Data))
	for _, row := range parsed.Data {
		out = append(out, row.Embedding)
	}
	if parsed.Model == "" {
		parsed.Model = model
	}
	return &EmbedResponse{Model: parsed.Model, Embeddings: out, Usage: parsed.Usage}, nil
}

type openAIChatResponse struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Created int64  `json:"created"`
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
	Usage Usage `json:"usage"`
}

func parseOpenAIChatResponse(payload []byte, fallbackModel string) (*ChatResponse, error) {
	var parsed openAIChatResponse
	if err := json.Unmarshal(payload, &parsed); err != nil {
		return nil, err
	}
	if len(parsed.Choices) == 0 {
		return nil, fmt.Errorf("ai: openai response missing choices")
	}
	model := parsed.Model
	if model == "" {
		model = fallbackModel
	}
	created := time.Now().UTC()
	if parsed.Created > 0 {
		created = time.Unix(parsed.Created, 0).UTC()
	}
	return &ChatResponse{
		ID:      parsed.ID,
		Model:   model,
		Message: parsed.Choices[0].Message,
		Usage:   parsed.Usage,
		Created: created,
	}, nil
}
