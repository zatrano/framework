package ai

import (
	"strings"

	"github.com/zatrano/framework/packages/http"
)

type demoChatBody struct {
	Message string `json:"message"`
	Model   string `json:"model"`
	Driver  string `json:"driver"`
}

// DemoChatHandler returns POST /demo/ai/chat — JSON body: message (required), optional model/driver.
func DemoChatHandler(mgr *Manager) func(*http.Request) *http.Response {
	return func(req *http.Request) *http.Response {
		if mgr == nil {
			return http.JSON(map[string]any{"message": "ai manager not configured"}).Status(503)
		}
		var body demoChatBody
		if err := req.JSON(&body); err != nil {
			return http.JSON(map[string]any{"message": "invalid json body"}).Status(400)
		}
		msg := strings.TrimSpace(body.Message)
		if msg == "" {
			return http.JSON(map[string]any{"message": "message is required"}).Status(422)
		}
		ctx := req.Raw().Context()
		chatReq := ChatRequest{
			Model: strings.TrimSpace(body.Model),
			Messages: []Message{
				{Role: "user", Content: msg},
			},
		}
		var (
			resp *ChatResponse
			err  error
		)
		if strings.TrimSpace(body.Driver) != "" {
			resp, err = mgr.Chat(ctx, chatReq, body.Driver)
		} else {
			resp, err = mgr.Chat(ctx, chatReq)
		}
		if err != nil {
			return http.JSON(map[string]any{"message": err.Error()}).Status(502)
		}
		return http.JSON(map[string]any{
			"id":      resp.ID,
			"model":   resp.Model,
			"message": resp.Message,
			"usage":   resp.Usage,
		})
	}
}
