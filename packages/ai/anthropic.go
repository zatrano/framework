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

// AnthropicDriver calls Anthropic Messages HTTP API.
type AnthropicDriver struct {
	BaseURL    string
	APIKey     string
	Model      string
	Version    string // anthropic-version header; default 2023-06-01
	HTTPClient *http.Client
	name       string
}

// Anthropic returns an Anthropic Messages driver.
func Anthropic(apiKey string) Driver {
	return &AnthropicDriver{
		BaseURL: "https://api.anthropic.com",
		APIKey:  apiKey,
		Model:   "claude-sonnet-4-20250514",
		Version: "2023-06-01",
		name:    "anthropic",
	}
}

func (d *AnthropicDriver) Name() string {
	if d != nil && d.name != "" {
		return d.name
	}
	return "anthropic"
}

func (d *AnthropicDriver) Capabilities() []Capability {
	return []Capability{CapChat, CapVision, CapTools, CapStream}
}

func (d *AnthropicDriver) Health(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	base := strings.TrimRight(d.BaseURL, "/")
	if base == "" {
		base = "https://api.anthropic.com"
	}
	client := d.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/v1/models", nil)
	if err != nil {
		return HealthError(d.Name(), err)
	}
	d.setHeaders(req)
	resp, err := client.Do(req)
	if err != nil {
		return HealthError(d.Name(), err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return HealthError(d.Name(), fmt.Errorf("status %d", resp.StatusCode))
	}
	return nil
}

func (d *AnthropicDriver) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	if d.APIKey != "" {
		req.Header.Set("x-api-key", d.APIKey)
	}
	ver := d.Version
	if ver == "" {
		ver = "2023-06-01"
	}
	req.Header.Set("anthropic-version", ver)
}

func (d *AnthropicDriver) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	base := strings.TrimRight(d.BaseURL, "/")
	if base == "" {
		base = "https://api.anthropic.com"
	}
	model := req.Model
	if model == "" || model == "zatrano-fake-1" {
		model = d.Model
	}
	if model == "" {
		model = "claude-sonnet-4-20250514"
	}
	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 1024
	}

	system, msgs := splitAnthropicMessages(req.Messages)
	body := map[string]any{
		"model":      model,
		"max_tokens": maxTokens,
		"messages":   msgs,
	}
	if system != "" {
		body["system"] = system
	}
	if req.Temperature != nil {
		body["temperature"] = *req.Temperature
	}
	applyAnthropicTools(body, req.Tools, req.ToolChoice)
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	client := d.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/v1/messages", bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	d.setHeaders(httpReq)
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, wrapTransportError(d.Name(), err)
	}
	defer resp.Body.Close()
	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, wrapTransportError(d.Name(), err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, HTTPError(d.Name(), resp.StatusCode, string(payload), parseRetryAfter(resp.Header.Get("Retry-After")))
	}
	return parseAnthropicMessage(payload, model)
}

func parseAnthropicMessage(payload []byte, fallbackModel string) (*ChatResponse, error) {
	var parsed struct {
		ID         string `json:"id"`
		Model      string `json:"model"`
		StopReason string `json:"stop_reason"`
		Content    []struct {
			Type  string          `json:"type"`
			Text  string          `json:"text"`
			ID    string          `json:"id"`
			Name  string          `json:"name"`
			Input json.RawMessage `json:"input"`
		} `json:"content"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(payload, &parsed); err != nil {
		return nil, err
	}
	var text strings.Builder
	var calls []ToolCall
	for _, c := range parsed.Content {
		switch c.Type {
		case "text":
			text.WriteString(c.Text)
		case "tool_use":
			args := "{}"
			if len(c.Input) > 0 {
				args = string(c.Input)
			}
			calls = append(calls, ToolCall{
				ID:   c.ID,
				Type: "function",
				Function: ToolCallFunction{
					Name:      c.Name,
					Arguments: args,
				},
			})
		}
	}
	finish := parsed.StopReason
	switch finish {
	case "end_turn":
		finish = "stop"
	case "tool_use":
		finish = "tool_calls"
	}
	return &ChatResponse{
		ID:    parsed.ID,
		Model: firstNonEmpty(parsed.Model, fallbackModel),
		Message: Message{
			Role:      "assistant",
			Content:   text.String(),
			ToolCalls: calls,
		},
		FinishReason: finish,
		Usage: Usage{
			PromptTokens:     parsed.Usage.InputTokens,
			CompletionTokens: parsed.Usage.OutputTokens,
			TotalTokens:      parsed.Usage.InputTokens + parsed.Usage.OutputTokens,
		},
		Created: time.Now().UTC(),
	}, nil
}

func applyAnthropicTools(body map[string]any, tools []Tool, choice *ToolChoice) {
	if body == nil || len(tools) == 0 {
		return
	}
	out := make([]map[string]any, 0, len(tools))
	for _, t := range tools {
		schema := t.Function.Parameters
		if len(schema) == 0 {
			schema = json.RawMessage(`{"type":"object","properties":{}}`)
		}
		out = append(out, map[string]any{
			"name":         t.Function.Name,
			"description":  t.Function.Description,
			"input_schema": schema,
		})
	}
	body["tools"] = out
	if choice == nil {
		return
	}
	mode := strings.ToLower(strings.TrimSpace(choice.Mode))
	switch mode {
	case "", "auto":
		body["tool_choice"] = map[string]any{"type": "auto"}
	case "none":
		body["tool_choice"] = map[string]any{"type": "none"}
	case "required":
		body["tool_choice"] = map[string]any{"type": "any"}
	case "function":
		name := strings.TrimSpace(choice.Name)
		if name == "" {
			body["tool_choice"] = map[string]any{"type": "auto"}
		} else {
			body["tool_choice"] = map[string]any{"type": "tool", "name": name}
		}
	}
}

func splitAnthropicMessages(in []Message) (system string, out []map[string]any) {
	out = make([]map[string]any, 0, len(in))
	for _, m := range in {
		role := strings.ToLower(m.Role)
		if role == "system" {
			if system != "" {
				system += "\n"
			}
			system += m.TextContent()
			continue
		}
		if role == "tool" {
			out = append(out, map[string]any{
				"role": "user",
				"content": []map[string]any{{
					"type":        "tool_result",
					"tool_use_id": m.ToolCallID,
					"content":     m.TextContent(),
				}},
			})
			continue
		}
		if role != "user" && role != "assistant" {
			role = "user"
		}
		content := anthropicContent(m)
		out = append(out, map[string]any{"role": role, "content": content})
	}
	return system, out
}

func anthropicContent(m Message) any {
	if len(m.ToolCalls) > 0 {
		parts := make([]map[string]any, 0, len(m.ToolCalls)+1)
		if txt := m.TextContent(); txt != "" {
			parts = append(parts, map[string]any{"type": "text", "text": txt})
		}
		for _, tc := range m.ToolCalls {
			var input any = map[string]any{}
			raw := strings.TrimSpace(tc.Function.Arguments)
			if raw != "" {
				_ = json.Unmarshal([]byte(raw), &input)
			}
			parts = append(parts, map[string]any{
				"type":  "tool_use",
				"id":    tc.ID,
				"name":  tc.Function.Name,
				"input": input,
			})
		}
		return parts
	}
	if len(m.Parts) == 0 {
		return m.TextContent()
	}
	parts := make([]map[string]any, 0, len(m.Parts))
	for _, p := range m.Parts {
		switch p.Type {
		case PartImageURL:
			if p.ImageURL == nil {
				continue
			}
			parts = append(parts, map[string]any{
				"type": "image",
				"source": map[string]any{
					"type": "url",
					"url":  p.ImageURL.URL,
				},
			})
		default:
			parts = append(parts, map[string]any{"type": "text", "text": p.Text})
		}
	}
	if len(parts) == 0 {
		return m.TextContent()
	}
	return parts
}
