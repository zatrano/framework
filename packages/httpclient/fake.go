package httpclient

import (
	"bytes"
	"io"
	stdhttp "net/http"
	"sync"
)

// FakeResponse is a canned HTTP response for FakeTransport.
type FakeResponse struct {
	Status  int
	Body    string
	Headers map[string]string
}

// RecordedRequest captures an outbound request seen by FakeTransport.
type RecordedRequest struct {
	Method string
	URL    string
	Body   string
	Header stdhttp.Header
}

// FakeTransport returns scripted responses in order (last repeats).
type FakeTransport struct {
	mu        sync.Mutex
	responses []FakeResponse
	index     int
	Requests  []RecordedRequest
}

// NewFakeTransport creates a fake round tripper.
func NewFakeTransport(responses ...FakeResponse) *FakeTransport {
	return &FakeTransport{responses: responses}
}

// RoundTrip implements http.RoundTripper.
func (f *FakeTransport) RoundTrip(req *stdhttp.Request) (*stdhttp.Response, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	body := ""
	if req.Body != nil {
		raw, _ := io.ReadAll(req.Body)
		_ = req.Body.Close()
		body = string(raw)
		req.Body = io.NopCloser(bytes.NewReader(raw))
	}
	f.Requests = append(f.Requests, RecordedRequest{
		Method: req.Method,
		URL:    req.URL.String(),
		Body:   body,
		Header: req.Header.Clone(),
	})

	resp := FakeResponse{Status: 200, Body: `{"ok":true}`}
	if len(f.responses) > 0 {
		idx := f.index
		if idx >= len(f.responses) {
			idx = len(f.responses) - 1
		}
		resp = f.responses[idx]
		f.index++
	}
	if resp.Status == 0 {
		resp.Status = 200
	}
	header := make(stdhttp.Header)
	for k, v := range resp.Headers {
		header.Set(k, v)
	}
	if header.Get("Content-Type") == "" {
		header.Set("Content-Type", "application/json")
	}
	return &stdhttp.Response{
		StatusCode: resp.Status,
		Header:     header,
		Body:       io.NopCloser(bytes.NewBufferString(resp.Body)),
		Request:    req,
	}, nil
}

// WithTransport sets a custom RoundTripper on the client.
func (c *Client) WithTransport(rt stdhttp.RoundTripper) *Client {
	if c.http == nil {
		c.http = &stdhttp.Client{}
	}
	c.http.Transport = rt
	return c
}

// Fake returns a client that serves the given canned responses.
func Fake(responses ...FakeResponse) *Client {
	return New().WithTransport(NewFakeTransport(responses...))
}
