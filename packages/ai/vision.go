package ai

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Multimodal content part types (OpenAI-compatible).
const (
	PartText     = "text"
	PartImageURL = "image_url"
)

// ContentPart is one piece of multimodal message content.
type ContentPart struct {
	Type     string    `json:"type"` // text | image_url
	Text     string    `json:"text,omitempty"`
	ImageURL *ImageURL `json:"image_url,omitempty"`
}

// ImageURL references an image for vision models.
type ImageURL struct {
	URL    string `json:"url"`
	Detail string `json:"detail,omitempty"` // auto | low | high
}

// TextPart builds a text content part.
func TextPart(text string) ContentPart {
	return ContentPart{Type: PartText, Text: text}
}

// ImageURLPart builds an image_url content part.
func ImageURLPart(url string, detail ...string) ContentPart {
	p := ContentPart{Type: PartImageURL, ImageURL: &ImageURL{URL: url}}
	if len(detail) > 0 {
		p.ImageURL.Detail = detail[0]
	}
	return p
}

// UserVision builds a user message with text + optional images.
func UserVision(text string, imageURLs ...string) Message {
	parts := make([]ContentPart, 0, 1+len(imageURLs))
	if text != "" {
		parts = append(parts, TextPart(text))
	}
	for _, u := range imageURLs {
		parts = append(parts, ImageURLPart(u))
	}
	return Message{Role: "user", Parts: parts}
}

// TextContent returns plain text from Content or text Parts.
func (m Message) TextContent() string {
	if m.Content != "" {
		return m.Content
	}
	var b strings.Builder
	for _, p := range m.Parts {
		if p.Type == PartText || (p.Type == "" && p.Text != "") {
			if b.Len() > 0 {
				b.WriteByte(' ')
			}
			b.WriteString(p.Text)
		}
	}
	return b.String()
}

// HasImages reports whether the message includes image_url parts.
func (m Message) HasImages() bool {
	for _, p := range m.Parts {
		if p.Type == PartImageURL && p.ImageURL != nil && p.ImageURL.URL != "" {
			return true
		}
	}
	return false
}

// MarshalJSON encodes Content as a string or as a multimodal parts array.
func (m Message) MarshalJSON() ([]byte, error) {
	type alias struct {
		Role       string     `json:"role"`
		Content    any        `json:"content,omitempty"`
		ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
		ToolCallID string     `json:"tool_call_id,omitempty"`
		Name       string     `json:"name,omitempty"`
	}
	out := alias{
		Role:       m.Role,
		ToolCalls:  m.ToolCalls,
		ToolCallID: m.ToolCallID,
		Name:       m.Name,
	}
	switch {
	case len(m.Parts) > 0:
		out.Content = m.Parts
	case m.Content != "":
		out.Content = m.Content
	}
	return json.Marshal(out)
}

// UnmarshalJSON accepts string or multimodal array content.
func (m *Message) UnmarshalJSON(data []byte) error {
	type raw struct {
		Role       string          `json:"role"`
		Content    json.RawMessage `json:"content"`
		ToolCalls  []ToolCall      `json:"tool_calls"`
		ToolCallID string          `json:"tool_call_id"`
		Name       string          `json:"name"`
	}
	var r raw
	if err := json.Unmarshal(data, &r); err != nil {
		return err
	}
	m.Role = r.Role
	m.ToolCalls = r.ToolCalls
	m.ToolCallID = r.ToolCallID
	m.Name = r.Name
	m.Content = ""
	m.Parts = nil
	if len(r.Content) == 0 || string(r.Content) == "null" {
		return nil
	}
	var s string
	if err := json.Unmarshal(r.Content, &s); err == nil {
		m.Content = s
		return nil
	}
	var parts []ContentPart
	if err := json.Unmarshal(r.Content, &parts); err != nil {
		return fmt.Errorf("ai: message content: %w", err)
	}
	m.Parts = parts
	return nil
}
