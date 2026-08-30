package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// Response format types (OpenAI-compatible).
const (
	ResponseText       = "text"
	ResponseJSONObject = "json_object"
	ResponseJSONSchema = "json_schema"
)

// ResponseFormat asks the model to return structured output.
type ResponseFormat struct {
	Type   string          `json:"type"`             // text | json_object | json_schema
	Name   string          `json:"name,omitempty"`   // required for json_schema
	Schema json.RawMessage `json:"schema,omitempty"` // JSON Schema document
	Strict *bool           `json:"strict,omitempty"` // json_schema strict mode (default true when schema set)
}

// JSONObject requests any valid JSON object (provider json_object mode).
func JSONObject() *ResponseFormat {
	return &ResponseFormat{Type: ResponseJSONObject}
}

// JSONSchema requests output matching a JSON Schema (OpenAI json_schema mode).
func JSONSchema(name string, schema json.RawMessage) *ResponseFormat {
	return &ResponseFormat{Type: ResponseJSONSchema, Name: name, Schema: schema}
}

func (f *ResponseFormat) wantsJSON() bool {
	if f == nil {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(f.Type)) {
	case ResponseJSONObject, ResponseJSONSchema:
		return true
	default:
		return false
	}
}

// applyResponseFormat adds OpenAI-compatible response_format to a request body map.
func applyResponseFormat(body map[string]any, f *ResponseFormat) {
	if f == nil || body == nil {
		return
	}
	typ := strings.ToLower(strings.TrimSpace(f.Type))
	switch typ {
	case "", ResponseText:
		return
	case ResponseJSONObject:
		body["response_format"] = map[string]any{"type": ResponseJSONObject}
	case ResponseJSONSchema:
		name := strings.TrimSpace(f.Name)
		if name == "" {
			name = "response"
		}
		js := map[string]any{
			"name":   name,
			"schema": json.RawMessage(`{}`),
		}
		if len(f.Schema) > 0 {
			js["schema"] = f.Schema
		}
		strict := true
		if f.Strict != nil {
			strict = *f.Strict
		}
		js["strict"] = strict
		body["response_format"] = map[string]any{
			"type":        ResponseJSONSchema,
			"json_schema": js,
		}
	default:
		body["response_format"] = map[string]any{"type": typ}
	}
}

// DecodeJSON decodes the assistant message content into dest.
// Strips optional markdown code fences (```json … ```).
func (r *ChatResponse) DecodeJSON(dest any) error {
	if r == nil {
		return fmt.Errorf("ai: nil ChatResponse")
	}
	raw := strings.TrimSpace(r.Message.Content)
	if raw == "" {
		return &Error{Kind: KindInvalid, Err: fmt.Errorf("empty assistant content")}
	}
	raw = stripJSONFence(raw)
	if err := json.Unmarshal([]byte(raw), dest); err != nil {
		return &Error{Kind: KindInvalid, Err: fmt.Errorf("decode structured output: %w", err)}
	}
	return nil
}

func stripJSONFence(s string) string {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "```") {
		return s
	}
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSpace(s)
	if strings.HasPrefix(strings.ToLower(s), "json") {
		s = strings.TrimSpace(s[4:])
	}
	if i := strings.LastIndex(s, "```"); i >= 0 {
		s = s[:i]
	}
	return strings.TrimSpace(s)
}

// ChatJSON runs Chat and unmarshals the assistant JSON into dest.
// If ResponseFormat is unset, json_object mode is applied.
func (c *Client) ChatJSON(ctx context.Context, req ChatRequest, dest any) (*ChatResponse, error) {
	if req.ResponseFormat == nil {
		req.ResponseFormat = JSONObject()
	}
	resp, err := c.Chat(ctx, req)
	if err != nil {
		return nil, err
	}
	if err := resp.DecodeJSON(dest); err != nil {
		return resp, err
	}
	return resp, nil
}

// ChatJSON runs Chat on the default (or named) provider and decodes JSON into dest.
func (m *Manager) ChatJSON(ctx context.Context, req ChatRequest, dest any, driver ...string) (*ChatResponse, error) {
	if len(driver) > 0 && strings.TrimSpace(driver[0]) != "" {
		return m.Using(driver[0]).ChatJSON(ctx, req, dest)
	}
	return (&Client{mgr: m}).ChatJSON(ctx, req, dest)
}
