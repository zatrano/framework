package ai

import "context"

// LogFn is a printf-style logger (compatible with packages/log Infof).
type LogFn func(format string, args ...any)

// LogDriver wraps another driver and logs prompt/reply (or error).
type LogDriver struct {
	Log   LogFn
	Inner Driver
}

func (LogDriver) Name() string { return "log" }

func (d LogDriver) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	inner := d.Inner
	if inner == nil {
		inner = FakeDriver{}
	}
	resp, err := inner.Chat(ctx, req)
	if d.Log != nil {
		prompt := truncate(lastUser(req.Messages), 120)
		if err != nil {
			d.Log("ai: driver=log model=%s prompt=%q err=%v", req.Model, prompt, err)
		} else if resp != nil {
			d.Log("ai: driver=log model=%s prompt=%q reply=%q tokens=%d",
				req.Model, prompt, truncate(resp.Message.Content, 120), resp.Usage.TotalTokens)
		}
	}
	return resp, err
}
