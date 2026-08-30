package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"
)

// SpeechDriver optionally converts text↔speech.
type SpeechDriver interface {
	TextToSpeech(ctx context.Context, req TTSRequest) (*TTSResponse, error)
	SpeechToText(ctx context.Context, req STTRequest) (*STTResponse, error)
}

// TTSRequest is a text-to-speech request.
type TTSRequest struct {
	Model  string // e.g. tts-1, tts-1-hd
	Input  string
	Voice  string // alloy, echo, fable, onyx, nova, shimmer
	Format string // mp3, opus, aac, flac, wav, pcm (default mp3)
}

// TTSResponse holds synthesized audio bytes.
type TTSResponse struct {
	Model       string
	ContentType string
	Audio       []byte
}

// STTRequest is a speech-to-text request.
type STTRequest struct {
	Model    string // e.g. whisper-1
	Audio    []byte
	Filename string // e.g. audio.mp3 (helps multipart content-type)
	Language string // optional ISO-639-1
}

// STTResponse holds a transcript.
type STTResponse struct {
	Model    string
	Text     string
	Language string
}

// TextToSpeech runs TTS with Using/Profile scoping and fallback.
func (c *Client) TextToSpeech(ctx context.Context, req TTSRequest) (*TTSResponse, error) {
	if c == nil || c.mgr == nil {
		return nil, fmt.Errorf("ai: client is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if strings.TrimSpace(req.Input) == "" {
		return nil, &Error{Kind: KindInvalid, Err: fmt.Errorf("tts input is required")}
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
	base := RequestInfo{Provider: c.provider, Profile: c.profile, Model: req.Model, Op: "tts"}
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
		sd, ok := d.(SpeechDriver)
		if !ok {
			lastErr = fmt.Errorf("ai: driver [%s] does not support speech", name)
			continue
		}
		resp, n, err := callWithRetry(ctx, retry, func(ctx context.Context) (*TTSResponse, error) {
			return sd.TextToSpeech(ctx, req)
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
		lastErr = fmt.Errorf("ai: no speech providers available")
	}
	c.mgr.notifyResult(ctx, resultFrom(base, lastProv, start, attempts, fallbacks, Usage{}, lastErr))
	return nil, lastErr
}

// SpeechToText runs STT with Using/Profile scoping and fallback.
func (c *Client) SpeechToText(ctx context.Context, req STTRequest) (*STTResponse, error) {
	if c == nil || c.mgr == nil {
		return nil, fmt.Errorf("ai: client is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if len(req.Audio) == 0 {
		return nil, &Error{Kind: KindInvalid, Err: fmt.Errorf("stt audio is required")}
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
	base := RequestInfo{Provider: c.provider, Profile: c.profile, Model: req.Model, Op: "stt"}
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
		sd, ok := d.(SpeechDriver)
		if !ok {
			lastErr = fmt.Errorf("ai: driver [%s] does not support speech", name)
			continue
		}
		resp, n, err := callWithRetry(ctx, retry, func(ctx context.Context) (*STTResponse, error) {
			return sd.SpeechToText(ctx, req)
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
		lastErr = fmt.Errorf("ai: no speech providers available")
	}
	c.mgr.notifyResult(ctx, resultFrom(base, lastProv, start, attempts, fallbacks, Usage{}, lastErr))
	return nil, lastErr
}

// TextToSpeech on the default (or named) provider.
func (m *Manager) TextToSpeech(ctx context.Context, req TTSRequest, driver ...string) (*TTSResponse, error) {
	if len(driver) > 0 && strings.TrimSpace(driver[0]) != "" {
		return m.Using(driver[0]).TextToSpeech(ctx, req)
	}
	return (&Client{mgr: m}).TextToSpeech(ctx, req)
}

// SpeechToText on the default (or named) provider.
func (m *Manager) SpeechToText(ctx context.Context, req STTRequest, driver ...string) (*STTResponse, error) {
	if len(driver) > 0 && strings.TrimSpace(driver[0]) != "" {
		return m.Using(driver[0]).SpeechToText(ctx, req)
	}
	return (&Client{mgr: m}).SpeechToText(ctx, req)
}

// TextToSpeech implements SpeechDriver via OpenAI Audio Speech API.
func (d *OpenAIDriver) TextToSpeech(ctx context.Context, req TTSRequest) (*TTSResponse, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	base := strings.TrimRight(d.BaseURL, "/")
	if base == "" {
		base = "https://api.openai.com/v1"
	}
	model := req.Model
	if model == "" || model == "zatrano-fake-1" {
		model = "tts-1"
	}
	voice := req.Voice
	if voice == "" {
		voice = "alloy"
	}
	format := req.Format
	if format == "" {
		format = "mp3"
	}
	raw, err := json.Marshal(map[string]any{
		"model":           model,
		"input":           req.Input,
		"voice":           voice,
		"response_format": format,
	})
	if err != nil {
		return nil, err
	}
	client := d.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/audio/speech", bytes.NewReader(raw))
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
	ct := resp.Header.Get("Content-Type")
	if ct == "" {
		ct = "audio/" + format
	}
	return &TTSResponse{Model: model, ContentType: ct, Audio: payload}, nil
}

// SpeechToText implements SpeechDriver via OpenAI Audio Transcriptions API.
func (d *OpenAIDriver) SpeechToText(ctx context.Context, req STTRequest) (*STTResponse, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	base := strings.TrimRight(d.BaseURL, "/")
	if base == "" {
		base = "https://api.openai.com/v1"
	}
	model := req.Model
	if model == "" || model == "zatrano-fake-1" {
		model = "whisper-1"
	}
	filename := req.Filename
	if filename == "" {
		filename = "audio.mp3"
	}
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.WriteField("model", model)
	if req.Language != "" {
		_ = w.WriteField("language", req.Language)
	}
	part, err := w.CreateFormFile("file", filename)
	if err != nil {
		return nil, err
	}
	if _, err := part.Write(req.Audio); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	client := d.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/audio/transcriptions", &buf)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", w.FormDataContentType())
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
		Text     string `json:"text"`
		Language string `json:"language"`
	}
	if err := json.Unmarshal(payload, &parsed); err != nil {
		return nil, err
	}
	return &STTResponse{Model: model, Text: parsed.Text, Language: firstNonEmpty(parsed.Language, req.Language)}, nil
}

// TextToSpeech implements SpeechDriver for FakeDriver.
func (FakeDriver) TextToSpeech(ctx context.Context, req TTSRequest) (*TTSResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	model := req.Model
	if model == "" {
		model = "zatrano-fake-tts"
	}
	format := req.Format
	if format == "" {
		format = "mp3"
	}
	return &TTSResponse{
		Model:       model,
		ContentType: "audio/" + format,
		Audio:       []byte("FAKE-AUDIO:" + req.Input),
	}, nil
}

// SpeechToText implements SpeechDriver for FakeDriver.
func (FakeDriver) SpeechToText(ctx context.Context, req STTRequest) (*STTResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	model := req.Model
	if model == "" {
		model = "zatrano-fake-stt"
	}
	return &STTResponse{
		Model:    model,
		Text:     "ZATRANO AI stub transcript (" + fmt.Sprintf("%d bytes", len(req.Audio)) + ")",
		Language: req.Language,
	}, nil
}
