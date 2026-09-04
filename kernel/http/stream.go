package http

import (
	"fmt"
	"io"
	stdhttp "net/http"
	"strings"
	"time"
)

// StreamWriter writes a streaming response body.
type StreamWriter func(w stdhttp.ResponseWriter, flusher stdhttp.Flusher) error

// IsStream reports whether the response uses a stream writer.
func (r *Response) IsStream() bool {
	return r != nil && r.stream != nil
}

// Stream creates a streaming response.
func Stream(contentType string, writer StreamWriter) *Response {
	return &Response{
		status:      stdhttp.StatusOK,
		contentType: contentType,
		headers:     make(stdhttp.Header),
		stream:      writer,
	}
}

// SSEEvent is a single server-sent event.
type SSEEvent struct {
	Event string
	Data  string
	ID    string
	Retry int
}

// SSE creates a Server-Sent Events response.
func SSE(handler func(send func(SSEEvent) error) error) *Response {
	resp := Stream("text/event-stream", func(w stdhttp.ResponseWriter, flusher stdhttp.Flusher) error {
		send := func(event SSEEvent) error {
			var b strings.Builder
			if event.ID != "" {
				b.WriteString("id: ")
				b.WriteString(event.ID)
				b.WriteString("\n")
			}
			if event.Event != "" {
				b.WriteString("event: ")
				b.WriteString(event.Event)
				b.WriteString("\n")
			}
			if event.Retry > 0 {
				b.WriteString(fmt.Sprintf("retry: %d\n", event.Retry))
			}
			for _, line := range strings.Split(event.Data, "\n") {
				b.WriteString("data: ")
				b.WriteString(line)
				b.WriteString("\n")
			}
			b.WriteString("\n")
			if _, err := io.WriteString(w, b.String()); err != nil {
				return err
			}
			flusher.Flush()
			return nil
		}
		return handler(send)
	})
	resp.Header("Cache-Control", "no-cache")
	resp.Header("Connection", "keep-alive")
	resp.Header("X-Accel-Buffering", "no")
	return resp
}

// SSETick is a helper that emits numbered tick events.
func SSETick(count int, interval time.Duration) *Response {
	if count <= 0 {
		count = 5
	}
	if interval <= 0 {
		interval = time.Second
	}
	return SSE(func(send func(SSEEvent) error) error {
		for i := 1; i <= count; i++ {
			if err := send(SSEEvent{
				Event: "tick",
				ID:    fmt.Sprintf("%d", i),
				Data:  fmt.Sprintf(`{"n":%d,"at":"%s"}`, i, time.Now().Format(time.RFC3339)),
			}); err != nil {
				return err
			}
			if i < count {
				time.Sleep(interval)
			}
		}
		return send(SSEEvent{Event: "done", Data: `{"ok":true}`})
	})
}
