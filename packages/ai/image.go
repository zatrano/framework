package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// ImageDriver optionally generates images from a text prompt.
type ImageDriver interface {
	GenerateImage(ctx context.Context, req ImageRequest) (*ImageResponse, error)
}

// ImageRequest is an image generation request.
type ImageRequest struct {
	Model  string `json:"model,omitempty"`
	Prompt string `json:"prompt"`
	N      int    `json:"n,omitempty"`    // default 1
	Size   string `json:"size,omitempty"` // e.g. 1024x1024
}

// ImageData is one generated image (URL and/or base64).
type ImageData struct {
	URL     string `json:"url,omitempty"`
	B64JSON string `json:"b64_json,omitempty"`
}

// ImageResponse holds generated images.
type ImageResponse struct {
	Model   string      `json:"model"`
	Created time.Time   `json:"created_at"`
	Data    []ImageData `json:"data"`
}

// GenerateImage runs image generation with Using/Profile scoping and fallback.
func (c *Client) GenerateImage(ctx context.Context, req ImageRequest) (*ImageResponse, error) {
	if c == nil || c.mgr == nil {
		return nil, fmt.Errorf("ai: client is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	c.mgr.mu.RLock()
	defs := c.mgr.defaults
	names, err := c.resolveNamesLocked()
	timeout := defs.Timeout
	retry := defs.Retry.normalized()
	fallbackOnTimeout := defs.FallbackOnTimeout
	modelDefault := defs.Model
	if c.profile != "" {
		if p, ok := c.mgr.profileLocked(c.profile); ok && p.Model != "" {
			modelDefault = p.Model
		}
	}
	c.mgr.mu.RUnlock()
	if err != nil {
		return nil, err
	}
	if req.Model == "" {
		req.Model = modelDefault
	}
	if strings.TrimSpace(req.Prompt) == "" {
		return nil, &Error{Kind: KindInvalid, Err: fmt.Errorf("image prompt is required")}
	}

	start := time.Now()
	base := RequestInfo{Provider: c.provider, Profile: c.profile, Model: req.Model, Op: "image"}
	c.mgr.notifyRequest(ctx, base)

	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	var lastErr error
	var lastProv string
	attempts, fallbacks, tried := 0, 0, 0
	for _, name := range names {
		if tried > 0 {
			fallbacks++
		}
		tried++
		lastProv = name
		d, err := c.mgr.Driver(name)
		if err != nil {
			lastErr = err
			if !Fallbackable(err, fallbackOnTimeout) {
				c.mgr.notifyResult(ctx, resultFrom(base, name, start, attempts, fallbacks, Usage{}, err))
				return nil, err
			}
			continue
		}
		id, ok := d.(ImageDriver)
		if !ok {
			lastErr = fmt.Errorf("ai: driver [%s] does not support image generation", name)
			continue
		}
		resp, n, err := callWithRetry(ctx, retry, func(ctx context.Context) (*ImageResponse, error) {
			return id.GenerateImage(ctx, req)
		})
		attempts += n
		if err == nil {
			c.mgr.notifyResult(ctx, resultFrom(base, name, start, attempts, fallbacks, Usage{}, nil))
			return resp, nil
		}
		lastErr = err
		if !Fallbackable(err, fallbackOnTimeout) {
			c.mgr.notifyResult(ctx, resultFrom(base, name, start, attempts, fallbacks, Usage{}, err))
			return nil, err
		}
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("ai: no image providers available")
	}
	c.mgr.notifyResult(ctx, resultFrom(base, lastProv, start, attempts, fallbacks, Usage{}, lastErr))
	return nil, lastErr
}

// GenerateImage on the default (or named) provider.
func (m *Manager) GenerateImage(ctx context.Context, req ImageRequest, driver ...string) (*ImageResponse, error) {
	if len(driver) > 0 && strings.TrimSpace(driver[0]) != "" {
		return m.Using(driver[0]).GenerateImage(ctx, req)
	}
	return (&Client{mgr: m}).GenerateImage(ctx, req)
}

// GenerateImage implements ImageDriver via OpenAI Images API.
func (d *OpenAIDriver) GenerateImage(ctx context.Context, req ImageRequest) (*ImageResponse, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	base := strings.TrimRight(d.BaseURL, "/")
	if base == "" {
		base = "https://api.openai.com/v1"
	}
	model := req.Model
	if model == "" || model == "zatrano-fake-1" {
		model = "dall-e-3"
	}
	n := req.N
	if n <= 0 {
		n = 1
	}
	body := map[string]any{
		"model":  model,
		"prompt": req.Prompt,
		"n":      n,
	}
	if req.Size != "" {
		body["size"] = req.Size
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	client := d.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/images/generations", bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if d.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+d.APIKey)
	}
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
	var parsed struct {
		Created int64       `json:"created"`
		Data    []ImageData `json:"data"`
	}
	if err := json.Unmarshal(payload, &parsed); err != nil {
		return nil, err
	}
	created := time.Now().UTC()
	if parsed.Created > 0 {
		created = time.Unix(parsed.Created, 0).UTC()
	}
	return &ImageResponse{Model: model, Created: created, Data: parsed.Data}, nil
}

// GenerateImage implements ImageDriver for FakeDriver.
func (FakeDriver) GenerateImage(ctx context.Context, req ImageRequest) (*ImageResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	model := req.Model
	if model == "" {
		model = "zatrano-fake-image"
	}
	return &ImageResponse{
		Model:   model,
		Created: time.Now().UTC(),
		Data: []ImageData{{
			URL: "https://example.invalid/fake-image.png?q=" + strings.ReplaceAll(req.Prompt, " ", "+"),
		}},
	}, nil
}

// ImageEditor optionally edits or varies images.
type ImageEditor interface {
	EditImage(ctx context.Context, req ImageEditRequest) (*ImageResponse, error)
	VaryImage(ctx context.Context, req ImageVaryRequest) (*ImageResponse, error)
}

// ImageEditRequest is an OpenAI-style image edit.
type ImageEditRequest struct {
	Model  string
	Prompt string
	Image  []byte // PNG
	Mask   []byte // optional PNG
	N      int
	Size   string
}

// ImageVaryRequest is an OpenAI-style image variation.
type ImageVaryRequest struct {
	Model string
	Image []byte // PNG
	N     int
	Size  string
}

// EditImage runs image edit with Using/Profile scoping and fallback.
func (c *Client) EditImage(ctx context.Context, req ImageEditRequest) (*ImageResponse, error) {
	return c.runImageEditor(ctx, "image_edit", req.Model, func(ed ImageEditor, ctx context.Context) (*ImageResponse, error) {
		return ed.EditImage(ctx, req)
	}, func() error {
		if strings.TrimSpace(req.Prompt) == "" {
			return &Error{Kind: KindInvalid, Err: fmt.Errorf("image edit prompt is required")}
		}
		if len(req.Image) == 0 {
			return &Error{Kind: KindInvalid, Err: fmt.Errorf("image edit bytes are required")}
		}
		return nil
	})
}

// VaryImage runs image variation with Using/Profile scoping and fallback.
func (c *Client) VaryImage(ctx context.Context, req ImageVaryRequest) (*ImageResponse, error) {
	return c.runImageEditor(ctx, "image_vary", req.Model, func(ed ImageEditor, ctx context.Context) (*ImageResponse, error) {
		return ed.VaryImage(ctx, req)
	}, func() error {
		if len(req.Image) == 0 {
			return &Error{Kind: KindInvalid, Err: fmt.Errorf("image variation bytes are required")}
		}
		return nil
	})
}

func (c *Client) runImageEditor(ctx context.Context, op, model string, call func(ImageEditor, context.Context) (*ImageResponse, error), validate func() error) (*ImageResponse, error) {
	if c == nil || c.mgr == nil {
		return nil, fmt.Errorf("ai: client is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := validate(); err != nil {
		return nil, err
	}
	c.mgr.mu.RLock()
	defs := c.mgr.defaults
	names, err := c.resolveNamesLocked()
	timeout := defs.Timeout
	retry := defs.Retry.normalized()
	fallbackOnTimeout := defs.FallbackOnTimeout
	c.mgr.mu.RUnlock()
	if err != nil {
		return nil, err
	}
	start := time.Now()
	base := RequestInfo{Provider: c.provider, Profile: c.profile, Model: model, Op: op}
	c.mgr.notifyRequest(ctx, base)
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	var lastErr error
	var lastProv string
	attempts, fallbacks, tried := 0, 0, 0
	for _, name := range names {
		if tried > 0 {
			fallbacks++
		}
		tried++
		lastProv = name
		d, err := c.mgr.Driver(name)
		if err != nil {
			lastErr = err
			if !Fallbackable(err, fallbackOnTimeout) {
				c.mgr.notifyResult(ctx, resultFrom(base, name, start, attempts, fallbacks, Usage{}, err))
				return nil, err
			}
			continue
		}
		ed, ok := d.(ImageEditor)
		if !ok {
			lastErr = fmt.Errorf("ai: driver [%s] does not support image edit/variation", name)
			continue
		}
		resp, n, err := callWithRetry(ctx, retry, func(ctx context.Context) (*ImageResponse, error) {
			return call(ed, ctx)
		})
		attempts += n
		if err == nil {
			c.mgr.notifyResult(ctx, resultFrom(base, name, start, attempts, fallbacks, Usage{}, nil))
			return resp, nil
		}
		lastErr = err
		if !Fallbackable(err, fallbackOnTimeout) {
			c.mgr.notifyResult(ctx, resultFrom(base, name, start, attempts, fallbacks, Usage{}, err))
			return nil, err
		}
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("ai: no image editor providers available")
	}
	c.mgr.notifyResult(ctx, resultFrom(base, lastProv, start, attempts, fallbacks, Usage{}, lastErr))
	return nil, lastErr
}

// EditImage on the default (or named) provider.
func (m *Manager) EditImage(ctx context.Context, req ImageEditRequest, driver ...string) (*ImageResponse, error) {
	if len(driver) > 0 && strings.TrimSpace(driver[0]) != "" {
		return m.Using(driver[0]).EditImage(ctx, req)
	}
	return (&Client{mgr: m}).EditImage(ctx, req)
}

// VaryImage on the default (or named) provider.
func (m *Manager) VaryImage(ctx context.Context, req ImageVaryRequest, driver ...string) (*ImageResponse, error) {
	if len(driver) > 0 && strings.TrimSpace(driver[0]) != "" {
		return m.Using(driver[0]).VaryImage(ctx, req)
	}
	return (&Client{mgr: m}).VaryImage(ctx, req)
}

// EditImage implements ImageEditor via OpenAI Images Edits API.
func (d *OpenAIDriver) EditImage(ctx context.Context, req ImageEditRequest) (*ImageResponse, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	base := strings.TrimRight(d.BaseURL, "/")
	if base == "" {
		base = "https://api.openai.com/v1"
	}
	model := req.Model
	if model == "" || model == "zatrano-fake-1" {
		model = "dall-e-2"
	}
	n := req.N
	if n <= 0 {
		n = 1
	}
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.WriteField("model", model)
	_ = w.WriteField("prompt", req.Prompt)
	_ = w.WriteField("n", strconv.Itoa(n))
	if req.Size != "" {
		_ = w.WriteField("size", req.Size)
	}
	part, err := w.CreateFormFile("image", "image.png")
	if err != nil {
		return nil, err
	}
	if _, err := part.Write(req.Image); err != nil {
		return nil, err
	}
	if len(req.Mask) > 0 {
		mp, err := w.CreateFormFile("mask", "mask.png")
		if err != nil {
			return nil, err
		}
		if _, err := mp.Write(req.Mask); err != nil {
			return nil, err
		}
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return d.postImageMultipart(ctx, base+"/images/edits", w.FormDataContentType(), &buf, model)
}

// VaryImage implements ImageEditor via OpenAI Images Variations API.
func (d *OpenAIDriver) VaryImage(ctx context.Context, req ImageVaryRequest) (*ImageResponse, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	base := strings.TrimRight(d.BaseURL, "/")
	if base == "" {
		base = "https://api.openai.com/v1"
	}
	model := req.Model
	if model == "" || model == "zatrano-fake-1" {
		model = "dall-e-2"
	}
	n := req.N
	if n <= 0 {
		n = 1
	}
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.WriteField("model", model)
	_ = w.WriteField("n", strconv.Itoa(n))
	if req.Size != "" {
		_ = w.WriteField("size", req.Size)
	}
	part, err := w.CreateFormFile("image", "image.png")
	if err != nil {
		return nil, err
	}
	if _, err := part.Write(req.Image); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return d.postImageMultipart(ctx, base+"/images/variations", w.FormDataContentType(), &buf, model)
}

func (d *OpenAIDriver) postImageMultipart(ctx context.Context, endpoint, contentType string, body io.Reader, model string) (*ImageResponse, error) {
	client := d.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, body)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", contentType)
	if d.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+d.APIKey)
	}
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
	var parsed struct {
		Created int64       `json:"created"`
		Data    []ImageData `json:"data"`
	}
	if err := json.Unmarshal(payload, &parsed); err != nil {
		return nil, err
	}
	created := time.Now().UTC()
	if parsed.Created > 0 {
		created = time.Unix(parsed.Created, 0).UTC()
	}
	return &ImageResponse{Model: model, Created: created, Data: parsed.Data}, nil
}

// EditImage implements ImageEditor for FakeDriver.
func (FakeDriver) EditImage(ctx context.Context, req ImageEditRequest) (*ImageResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	model := req.Model
	if model == "" {
		model = "zatrano-fake-image-edit"
	}
	return &ImageResponse{
		Model:   model,
		Created: time.Now().UTC(),
		Data:    []ImageData{{URL: "https://example.invalid/fake-edit.png?q=" + strings.ReplaceAll(req.Prompt, " ", "+")}},
	}, nil
}

// VaryImage implements ImageEditor for FakeDriver.
func (FakeDriver) VaryImage(ctx context.Context, req ImageVaryRequest) (*ImageResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	model := req.Model
	if model == "" {
		model = "zatrano-fake-image-vary"
	}
	return &ImageResponse{
		Model:   model,
		Created: time.Now().UTC(),
		Data:    []ImageData{{URL: "https://example.invalid/fake-vary.png"}},
	}, nil
}
