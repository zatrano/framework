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

// ChatStream opens an OpenAI-compatible SSE chat stream.
// Setup errors (HTTP/status) are returned directly; mid-stream errors are sent on the channel.
func (d *OpenAIDriver) ChatStream(ctx context.Context, req ChatRequest) (<-chan StreamChunk, error) {
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
		"stream":   true,
		"stream_options": map[string]any{
			"include_usage": true,
		},
	}
	if req.Temperature != nil {
		body["temperature"] = *req.Temperature
	}
	if req.MaxTokens > 0 {
		body["max_tokens"] = req.MaxTokens
	}
	applyResponseFormat(body, req.ResponseFormat)
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
	httpReq.Header.Set("Accept", "text/event-stream")
	if d.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+d.APIKey)
	}

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
	go readOpenAISSE(ctx, resp.Body, model, out)
	return out, nil
}

func readOpenAISSE(ctx context.Context, body io.ReadCloser, fallbackModel string, out chan<- StreamChunk) {
	defer close(out)
	defer body.Close()

	scanner := bufio.NewScanner(body)
	// Some providers send large SSE lines.
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	var lastID, lastModel string
	lastModel = fallbackModel
	sentDone := false

	send := func(c StreamChunk) bool {
		select {
		case <-ctx.Done():
			out <- StreamChunk{Done: true, Err: ctx.Err(), ID: lastID, Model: lastModel}
			return false
		case out <- c:
			return true
		}
	}

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			if !sentDone {
				_ = send(StreamChunk{Done: true, ID: lastID, Model: lastModel})
				sentDone = true
			}
			return
		}

		var frame openAIStreamFrame
		if err := json.Unmarshal([]byte(data), &frame); err != nil {
			_ = send(StreamChunk{Done: true, Err: err, ID: lastID, Model: lastModel})
			return
		}
		if frame.ID != "" {
			lastID = frame.ID
		}
		if frame.Model != "" {
			lastModel = frame.Model
		}

		var delta string
		for _, ch := range frame.Choices {
			delta += ch.Delta.Content
		}
		chunk := StreamChunk{Delta: delta, ID: lastID, Model: lastModel}
		if frame.Usage != nil {
			u := *frame.Usage
			chunk.Usage = &u
		}
		if !send(chunk) {
			return
		}
	}
	if err := scanner.Err(); err != nil {
		_ = send(StreamChunk{Done: true, Err: wrapTransportError("openai", err), ID: lastID, Model: lastModel})
		return
	}
	if !sentDone {
		_ = send(StreamChunk{Done: true, ID: lastID, Model: lastModel})
	}
}

type openAIStreamFrame struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
	Usage *Usage `json:"usage"`
}
