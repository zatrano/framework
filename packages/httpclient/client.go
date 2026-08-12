package httpclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	stdhttp "net/http"
	"net/url"
	"strings"
	"time"
)

// Response wraps an HTTP response.
type Response struct {
	StatusCode int
	Headers    stdhttp.Header
	Body       []byte
}

// OK reports whether the status is 2xx.
func (r *Response) OK() bool {
	return r.StatusCode >= 200 && r.StatusCode < 300
}

// JSON decodes the body into dest.
func (r *Response) JSON(dest any) error {
	return json.Unmarshal(r.Body, dest)
}

// String returns the body as string.
func (r *Response) String() string {
	return string(r.Body)
}

// PendingRequest builds an outbound HTTP request.
type PendingRequest struct {
	client  *stdhttp.Client
	headers map[string]string
	query   url.Values
	baseURL string
	retry   *RetryPolicy
}

// Client is the ZATRANO HTTP client.
type Client struct {
	http    *stdhttp.Client
	baseURL string
}

// New creates an HTTP client.
func New(timeout ...time.Duration) *Client {
	wait := 30 * time.Second
	if len(timeout) > 0 && timeout[0] > 0 {
		wait = timeout[0]
	}
	return &Client{
		http: &stdhttp.Client{Timeout: wait},
	}
}

// BaseURL sets a base URL for relative requests.
func (c *Client) BaseURL(base string) *Client {
	c.baseURL = strings.TrimRight(base, "/")
	return c
}

// WithHeaders starts a pending request with headers.
func (c *Client) WithHeaders(headers map[string]string) *PendingRequest {
	return c.newPending().WithHeaders(headers)
}

// AsJSON sets JSON accept/content headers.
func (c *Client) AsJSON() *PendingRequest {
	return c.newPending().AsJSON()
}

// WithToken sets a bearer token.
func (c *Client) WithToken(token string) *PendingRequest {
	return c.newPending().WithToken(token)
}

// Get sends a GET request.
func (c *Client) Get(uri string, query ...map[string]string) (*Response, error) {
	return c.newPending().Get(uri, query...)
}

// Post sends a POST request with JSON body.
func (c *Client) Post(uri string, body any) (*Response, error) {
	return c.newPending().Post(uri, body)
}

// Put sends a PUT request with JSON body.
func (c *Client) Put(uri string, body any) (*Response, error) {
	return c.newPending().Put(uri, body)
}

// Patch sends a PATCH request with JSON body.
func (c *Client) Patch(uri string, body any) (*Response, error) {
	return c.newPending().Patch(uri, body)
}

// Delete sends a DELETE request.
func (c *Client) Delete(uri string) (*Response, error) {
	return c.newPending().Delete(uri)
}

func (c *Client) newPending() *PendingRequest {
	return &PendingRequest{
		client:  c.http,
		headers: map[string]string{},
		query:   url.Values{},
		baseURL: c.baseURL,
	}
}

// WithHeaders merges headers.
func (p *PendingRequest) WithHeaders(headers map[string]string) *PendingRequest {
	for key, value := range headers {
		p.headers[key] = value
	}
	return p
}

// AsJSON sets JSON headers.
func (p *PendingRequest) AsJSON() *PendingRequest {
	p.headers["Accept"] = "application/json"
	p.headers["Content-Type"] = "application/json"
	return p
}

// WithToken sets Authorization bearer token.
func (p *PendingRequest) WithToken(token string) *PendingRequest {
	p.headers["Authorization"] = "Bearer " + token
	return p
}

// WithQuery sets query parameters.
func (p *PendingRequest) WithQuery(query map[string]string) *PendingRequest {
	for key, value := range query {
		p.query.Set(key, value)
	}
	return p
}

// Get sends a GET request.
func (p *PendingRequest) Get(uri string, query ...map[string]string) (*Response, error) {
	if len(query) > 0 {
		p.WithQuery(query[0])
	}
	return p.send(stdhttp.MethodGet, uri, nil)
}

// Post sends a POST request.
func (p *PendingRequest) Post(uri string, body any) (*Response, error) {
	return p.send(stdhttp.MethodPost, uri, body)
}

// Put sends a PUT request.
func (p *PendingRequest) Put(uri string, body any) (*Response, error) {
	return p.send(stdhttp.MethodPut, uri, body)
}

// Patch sends a PATCH request.
func (p *PendingRequest) Patch(uri string, body any) (*Response, error) {
	return p.send(stdhttp.MethodPatch, uri, body)
}

// Delete sends a DELETE request.
func (p *PendingRequest) Delete(uri string) (*Response, error) {
	return p.send(stdhttp.MethodDelete, uri, nil)
}

func (p *PendingRequest) send(method, uri string, body any) (*Response, error) {
	target := uri
	if p.baseURL != "" && !strings.HasPrefix(uri, "http://") && !strings.HasPrefix(uri, "https://") {
		if !strings.HasPrefix(uri, "/") {
			uri = "/" + uri
		}
		target = p.baseURL + uri
	}

	parsed, err := url.Parse(target)
	if err != nil {
		return nil, err
	}
	q := parsed.Query()
	for key, values := range p.query {
		for _, value := range values {
			q.Set(key, value)
		}
	}
	parsed.RawQuery = q.Encode()

	var reader io.Reader
	if body != nil {
		switch v := body.(type) {
		case []byte:
			reader = bytes.NewReader(v)
		case string:
			reader = strings.NewReader(v)
		case io.Reader:
			reader = v
		default:
			raw, err := json.Marshal(v)
			if err != nil {
				return nil, err
			}
			reader = bytes.NewReader(raw)
			if _, ok := p.headers["Content-Type"]; !ok {
				p.headers["Content-Type"] = "application/json"
			}
		}
	}

	req, err := stdhttp.NewRequest(method, parsed.String(), reader)
	if err != nil {
		return nil, err
	}
	for key, value := range p.headers {
		req.Header.Set(key, value)
	}

	attempts := 1
	if p.retry != nil && p.retry.MaxAttempts > 1 {
		attempts = p.retry.MaxAttempts
	}

	var lastResp *Response
	var lastErr error
	for attempt := 0; attempt < attempts; attempt++ {
		if attempt > 0 {
			// Rebuild body reader for retries when possible.
			if body != nil {
				switch v := body.(type) {
				case []byte:
					reader = bytes.NewReader(v)
				case string:
					reader = strings.NewReader(v)
				default:
					if raw, err := json.Marshal(v); err == nil {
						reader = bytes.NewReader(raw)
					}
				}
				req, err = stdhttp.NewRequest(method, parsed.String(), reader)
				if err != nil {
					return nil, err
				}
				for key, value := range p.headers {
					req.Header.Set(key, value)
				}
			}
			time.Sleep(p.backoff(attempt - 1))
		}

		resp, err := p.client.Do(req)
		if err != nil {
			lastErr = err
			lastResp = nil
			if p.retry == nil || p.retry.RetryOn == nil || !p.retry.RetryOn(0, err) || attempt == attempts-1 {
				return nil, err
			}
			continue
		}
		raw, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if readErr != nil {
			lastErr = readErr
			if attempt == attempts-1 {
				return nil, readErr
			}
			continue
		}
		lastResp = &Response{
			StatusCode: resp.StatusCode,
			Headers:    resp.Header.Clone(),
			Body:       raw,
		}
		lastErr = nil
		shouldRetry := p.retry != nil && p.retry.RetryOn != nil && p.retry.RetryOn(resp.StatusCode, nil)
		if !shouldRetry || attempt == attempts-1 {
			return lastResp, nil
		}
	}
	if lastResp != nil {
		return lastResp, nil
	}
	return nil, lastErr
}

// MustOK returns an error when the response is not successful.
func (r *Response) MustOK() error {
	if r.OK() {
		return nil
	}
	return fmt.Errorf("http client: unexpected status %d: %s", r.StatusCode, r.String())
}
