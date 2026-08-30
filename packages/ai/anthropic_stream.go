package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

// ChatStream implements StreamDriver for Anthropic Messages SSE.
func (d *AnthropicDriver) ChatStream(ctx context.Context, req ChatRequest) (<-chan StreamChunk, error) {
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
		"stream":     true,
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
	httpReq.Header.Set("Accept", "text/event-stream")
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, wrapTransportError(d.Name(), err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		payload, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		return nil, HTTPError(d.Name(), resp.StatusCode, string(payload), parseRetryAfter(resp.Header.Get("Retry-After")))
	}
	out := make(chan StreamChunk, 16)
	go readAnthropicSSE(ctx, resp.Body, model, out)
	return out, nil
}

func readAnthropicSSE(ctx context.Context, body io.ReadCloser, model string, out chan<- StreamChunk) {
	defer close(out)
	defer body.Close()
	sc := bufio.NewScanner(body)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var usage *Usage
	var toolCalls []ToolCall
	var currentTool *ToolCall
	var argBuf strings.Builder
	send := func(c StreamChunk) {
		select {
		case <-ctx.Done():
		case out <- c:
		}
	}
	for sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" || data == "[DONE]" {
			continue
		}
		var ev struct {
			Type  string `json:"type"`
			Index int    `json:"index"`
			Delta struct {
				Type        string `json:"type"`
				Text        string `json:"text"`
				PartialJSON string `json:"partial_json"`
			} `json:"delta"`
			ContentBlock struct {
				Type  string          `json:"type"`
				ID    string          `json:"id"`
				Name  string          `json:"name"`
				Input json.RawMessage `json:"input"`
			} `json:"content_block"`
			Message struct {
				Usage struct {
					InputTokens  int `json:"input_tokens"`
					OutputTokens int `json:"output_tokens"`
				} `json:"usage"`
			} `json:"message"`
			Usage struct {
				OutputTokens int `json:"output_tokens"`
			} `json:"usage"`
		}
		if err := json.Unmarshal([]byte(data), &ev); err != nil {
			continue
		}
		switch ev.Type {
		case "content_block_start":
			if ev.ContentBlock.Type == "tool_use" {
				currentTool = &ToolCall{
					ID:   ev.ContentBlock.ID,
					Type: "function",
					Function: ToolCallFunction{
						Name: ev.ContentBlock.Name,
					},
				}
				argBuf.Reset()
			}
		case "content_block_delta":
			if ev.Delta.Type == "text_delta" && ev.Delta.Text != "" {
				send(StreamChunk{Delta: ev.Delta.Text, Model: model})
			}
			if ev.Delta.Type == "input_json_delta" && ev.Delta.PartialJSON != "" {
				argBuf.WriteString(ev.Delta.PartialJSON)
				if currentTool != nil {
					send(StreamChunk{
						Model: model,
						ToolCallDeltas: []StreamToolCallDelta{{
							Index:     ev.Index,
							ID:        currentTool.ID,
							Type:      "function",
							Name:      currentTool.Function.Name,
							Arguments: ev.Delta.PartialJSON,
						}},
					})
				}
			}
		case "content_block_stop":
			if currentTool != nil {
				currentTool.Function.Arguments = argBuf.String()
				if currentTool.Function.Arguments == "" {
					currentTool.Function.Arguments = "{}"
				}
				toolCalls = append(toolCalls, *currentTool)
				currentTool = nil
			}
		case "message_start":
			u := Usage{
				PromptTokens:     ev.Message.Usage.InputTokens,
				CompletionTokens: ev.Message.Usage.OutputTokens,
				TotalTokens:      ev.Message.Usage.InputTokens + ev.Message.Usage.OutputTokens,
			}
			usage = &u
		case "message_delta":
			if usage == nil {
				usage = &Usage{}
			}
			if ev.Usage.OutputTokens > 0 {
				usage.CompletionTokens = ev.Usage.OutputTokens
				usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
			}
		case "message_stop":
			send(StreamChunk{Done: true, Model: model, Usage: usage, ToolCalls: toolCalls})
			return
		}
	}
	if err := sc.Err(); err != nil {
		send(StreamChunk{Err: err, Done: true})
		return
	}
	send(StreamChunk{Done: true, Model: model, Usage: usage, ToolCalls: toolCalls})
}
