package turnstile

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

const siteverifyURL = "https://challenges.cloudflare.com/turnstile/v0/siteverify"

// SiteverifyResponse Cloudflare Siteverify API yanıtı
type SiteverifyResponse struct {
	Success     bool     `json:"success"`
	ChallengeTS string   `json:"challenge_ts,omitempty"`
	Hostname    string   `json:"hostname,omitempty"`
	ErrorCodes  []string `json:"error-codes,omitempty"`
	Action      string   `json:"action,omitempty"`
	CData       string   `json:"cdata,omitempty"`
}

// Verify token'ı Cloudflare Siteverify API ile doğrular.
// secretKey: TURNSTILE_SECRET_KEY, token: form'dan gelen cf-turnstile-response, remoteIP: istemci IP.
func Verify(secretKey, token, remoteIP string) (*SiteverifyResponse, error) {
	if secretKey == "" {
		return &SiteverifyResponse{Success: false, ErrorCodes: []string{"missing-secret"}}, nil
	}
	if token == "" {
		return &SiteverifyResponse{Success: false, ErrorCodes: []string{"missing-input-response"}}, nil
	}

	form := url.Values{}
	form.Set("secret", secretKey)
	form.Set("response", token)
	if remoteIP != "" {
		form.Set("remoteip", remoteIP)
	}

	req, err := http.NewRequest(http.MethodPost, siteverifyURL, bytes.NewBufferString(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("turnstile request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("turnstile siteverify: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("turnstile read body: %w", err)
	}

	var result SiteverifyResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("turnstile decode: %w", err)
	}
	return &result, nil
}
