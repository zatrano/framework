package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/zatrano/framework/packages/support/uuid"
)

// Message is a chat message.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest is a chat completion request.
type ChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

// ChatResponse is a chat completion response.
type ChatResponse struct {
	ID      string    `json:"id"`
	Model   string    `json:"model"`
	Message Message   `json:"message"`
	Usage   Usage     `json:"usage"`
	Created time.Time `json:"created_at"`
}

// Usage tracks token usage.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Driver generates completions.
type Driver interface {
	Name() string
	Chat(req ChatRequest) (*ChatResponse, error)
}

// Manager resolves AI drivers.
type Manager struct {
	mu            sync.RWMutex
	defaultDriver string
	drivers       map[string]Driver
}

// New creates an AI manager with fake and log drivers.
func New() *Manager {
	m := &Manager{drivers: make(map[string]Driver), defaultDriver: "fake"}
	m.Extend("fake", FakeDriver{})
	m.Extend("log", LogDriver{})
	return m
}

// Extend registers a driver.
func (m *Manager) Extend(name string, driver Driver) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.drivers[strings.ToLower(name)] = driver
}

// Use sets the default driver.
func (m *Manager) Use(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.defaultDriver = strings.ToLower(name)
}

// Driver returns a named driver.
func (m *Manager) Driver(name ...string) (Driver, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	n := m.defaultDriver
	if len(name) > 0 && name[0] != "" {
		n = strings.ToLower(name[0])
	}
	d, ok := m.drivers[n]
	if !ok {
		return nil, fmt.Errorf("ai: driver [%s] not configured", n)
	}
	return d, nil
}

// Chat runs a chat completion on the default (or named) driver.
func (m *Manager) Chat(req ChatRequest, driver ...string) (*ChatResponse, error) {
	d, err := m.Driver(driver...)
	if err != nil {
		return nil, err
	}
	if req.Model == "" {
		req.Model = "zatrano-fake-1"
	}
	return d.Chat(req)
}

// FakeDriver returns deterministic stub replies for tests and local use without API keys.
type FakeDriver struct{}

func (FakeDriver) Name() string { return "fake" }

func (FakeDriver) Chat(req ChatRequest) (*ChatResponse, error) {
	prompt := lastUser(req.Messages)
	reply := "ZATRANO AI stub: " + prompt
	if prompt == "" {
		reply = "ZATRANO AI stub: hello"
	}
	promptTokens := len(strings.Fields(prompt)) + 1
	completionTokens := len(strings.Fields(reply))
	return &ChatResponse{
		ID:      "chat_" + uuid.New()[:8],
		Model:   req.Model,
		Message: Message{Role: "assistant", Content: reply},
		Usage: Usage{
			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
			TotalTokens:      promptTokens + completionTokens,
		},
		Created: time.Now().UTC(),
	}, nil
}

// LogDriver mirrors FakeDriver (placeholder for logging integrations).
type LogDriver struct{}

func (LogDriver) Name() string { return "log" }

func (d LogDriver) Chat(req ChatRequest) (*ChatResponse, error) {
	return FakeDriver{}.Chat(req)
}

// OpenAIDriver calls an OpenAI-compatible chat completions HTTP API.
type OpenAIDriver struct {
	BaseURL    string
	APIKey     string
	Model      string
	HTTPClient *http.Client
}

// OpenAI returns an OpenAIDriver with the default API base URL.
func OpenAI(apiKey string) Driver {
	return &OpenAIDriver{
		BaseURL: "https://api.openai.com/v1",
		APIKey:  apiKey,
		Model:   "gpt-4o-mini",
	}
}

func (d *OpenAIDriver) Name() string { return "openai" }

func (d *OpenAIDriver) Chat(req ChatRequest) (*ChatResponse, error) {
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
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	client := d.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	httpReq, err := http.NewRequest(http.MethodPost, base+"/chat/completions", bytes.NewReader(raw))
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

func lastUser(messages []Message) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			return messages[i].Content
		}
	}
	if len(messages) > 0 {
		return messages[len(messages)-1].Content
	}
	return ""
}
