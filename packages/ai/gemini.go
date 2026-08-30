package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// GeminiDriver calls Google Gemini generateContent HTTP API.
type GeminiDriver struct {
	BaseURL    string // default https://generativelanguage.googleapis.com
	APIKey     string
	Model      string
	HTTPClient *http.Client
	name       string
}

// Gemini returns a Gemini generateContent driver.
func Gemini(apiKey string) Driver {
	return &GeminiDriver{
		BaseURL: "https://generativelanguage.googleapis.com",
		APIKey:  apiKey,
		Model:   "gemini-2.0-flash",
		name:    "gemini",
	}
}

func (d *GeminiDriver) Name() string {
	if d != nil && d.name != "" {
		return d.name
	}
	return "gemini"
}

func (d *GeminiDriver) Capabilities() []Capability {
	return []Capability{CapChat}
}

func (d *GeminiDriver) Health(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	base := strings.TrimRight(d.BaseURL, "/")
	if base == "" {
		base = "https://generativelanguage.googleapis.com"
	}
	u := base + "/v1beta/models?key=" + url.QueryEscape(d.APIKey)
	client := d.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return HealthError(d.Name(), err)
	}
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

func (d *GeminiDriver) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	base := strings.TrimRight(d.BaseURL, "/")
	if base == "" {
		base = "https://generativelanguage.googleapis.com"
	}
	model := req.Model
	if model == "" || model == "zatrano-fake-1" {
		model = d.Model
	}
	if model == "" {
		model = "gemini-2.0-flash"
	}

	contents, system := geminiContents(req.Messages)
	body := map[string]any{"contents": contents}
	if system != "" {
		body["systemInstruction"] = map[string]any{
			"parts": []map[string]string{{"text": system}},
		}
	}
	gen := map[string]any{}
	if req.Temperature != nil {
		gen["temperature"] = *req.Temperature
	}
	if req.MaxTokens > 0 {
		gen["maxOutputTokens"] = req.MaxTokens
	}
	if len(gen) > 0 {
		body["generationConfig"] = gen
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	endpoint := fmt.Sprintf("%s/v1beta/models/%s:generateContent?key=%s",
		base, url.PathEscape(model), url.QueryEscape(d.APIKey))
	client := d.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
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
		return nil, HTTPError(d.Name(), resp.StatusCode, string(payload), 0)
	}
	var parsed struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
			FinishReason string `json:"finishReason"`
		} `json:"candidates"`
		UsageMetadata struct {
			PromptTokenCount     int `json:"promptTokenCount"`
			CandidatesTokenCount int `json:"candidatesTokenCount"`
			TotalTokenCount      int `json:"totalTokenCount"`
		} `json:"usageMetadata"`
	}
	if err := json.Unmarshal(payload, &parsed); err != nil {
		return nil, err
	}
	if len(parsed.Candidates) == 0 {
		return nil, &Error{Kind: KindInvalid, Provider: d.Name(), Err: fmt.Errorf("gemini response missing candidates")}
	}
	var text strings.Builder
	for _, p := range parsed.Candidates[0].Content.Parts {
		text.WriteString(p.Text)
	}
	finish := strings.ToLower(parsed.Candidates[0].FinishReason)
	if finish == "stop" || finish == "" {
		finish = "stop"
	}
	return &ChatResponse{
		ID:           "gemini_" + model,
		Model:        model,
		Message:      Message{Role: "assistant", Content: text.String()},
		FinishReason: finish,
		Usage: Usage{
			PromptTokens:     parsed.UsageMetadata.PromptTokenCount,
			CompletionTokens: parsed.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      parsed.UsageMetadata.TotalTokenCount,
		},
		Created: time.Now().UTC(),
	}, nil
}

func geminiContents(in []Message) (contents []map[string]any, system string) {
	contents = make([]map[string]any, 0, len(in))
	for _, m := range in {
		role := strings.ToLower(m.Role)
		if role == "system" {
			if system != "" {
				system += "\n"
			}
			system += m.TextContent()
			continue
		}
		gRole := "user"
		if role == "assistant" {
			gRole = "model"
		}
		contents = append(contents, map[string]any{
			"role":  gRole,
			"parts": geminiParts(m),
		})
	}
	return contents, system
}

func geminiParts(m Message) []map[string]any {
	if len(m.Parts) == 0 {
		return []map[string]any{{"text": m.TextContent()}}
	}
	out := make([]map[string]any, 0, len(m.Parts))
	for _, p := range m.Parts {
		if p.Type == PartImageURL && p.ImageURL != nil {
			// URL images: pass as text hint; native fileData needs bytes — keep URL in text for thin driver.
			out = append(out, map[string]any{"text": "[image] " + p.ImageURL.URL})
			continue
		}
		out = append(out, map[string]any{"text": p.Text})
	}
	if len(out) == 0 {
		return []map[string]any{{"text": m.TextContent()}}
	}
	return out
}
