package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/zatrano/framework/packages/ai"
)

// Agent runs a chat↔tool loop with optional memory and RAG retrieval.
type Agent struct {
	Chat     Chatter
	Tools    *Registry
	Memory   Memory // optional; defaults to ephemeral BufferMemory per Run
	Retrieve Retriever
	System   string // optional system prompt prepended when memory is empty
	MaxSteps int    // default 6
	// AllowToolsOnFinal keeps tools enabled on the last step (default false:
	// last step uses ToolChoiceNone so the model must produce a text answer).
	AllowToolsOnFinal bool
}

// Run appends the user message and loops until a text reply or MaxSteps.
func (a *Agent) Run(ctx context.Context, userMessage string) (*Result, error) {
	if a == nil || a.Chat == nil {
		return nil, fmt.Errorf("agent: Chat is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	userMessage = strings.TrimSpace(userMessage)
	if userMessage == "" {
		return nil, fmt.Errorf("agent: user message is required")
	}

	mem := a.Memory
	if mem == nil {
		mem = &BufferMemory{}
	}
	maxSteps := a.MaxSteps
	if maxSteps <= 0 {
		maxSteps = 6
	}

	if len(mem.Messages()) == 0 && strings.TrimSpace(a.System) != "" {
		mem.Append(ai.Message{Role: "system", Content: strings.TrimSpace(a.System)})
	}

	content := userMessage
	if a.Retrieve != nil {
		block, err := a.Retrieve.Retrieve(ctx, userMessage)
		if err != nil {
			return nil, err
		}
		block = strings.TrimSpace(block)
		if block != "" {
			content = "Context:\n" + block + "\n\nQuestion: " + userMessage
		}
	}
	mem.Append(ai.Message{Role: "user", Content: content})

	var tools []ai.Tool
	if a.Tools != nil {
		tools = a.Tools.Tools()
	}

	var last *ai.ChatResponse
	steps := 0
	for step := 1; step <= maxSteps; step++ {
		steps = step
		req := ai.ChatRequest{
			Messages: mem.Messages(),
			Tools:    tools,
		}
		if len(tools) > 0 {
			if step == maxSteps && !a.AllowToolsOnFinal {
				req.ToolChoice = ai.ToolChoiceNone()
			} else {
				req.ToolChoice = ai.ToolChoiceAuto()
			}
		}
		resp, err := a.Chat.Chat(ctx, req)
		if err != nil {
			return nil, err
		}
		last = resp
		if !resp.HasToolCalls() {
			mem.Append(resp.Message)
			return &Result{Response: resp, Steps: steps, Messages: mem.Messages()}, nil
		}
		mem.Append(ai.AssistantToolCalls(resp.Message.ToolCalls...))
		if a.Tools == nil {
			return nil, fmt.Errorf("agent: model requested tools but Tools registry is nil")
		}
		for _, call := range resp.Message.ToolCalls {
			out, err := a.Tools.Execute(ctx, call)
			if err != nil {
				out = "error: " + err.Error()
			}
			mem.Append(ai.ToolResultMessage(call.ID, out))
		}
	}
	if last == nil {
		return nil, fmt.Errorf("agent: no response")
	}
	return &Result{Response: last, Steps: steps, Messages: mem.Messages()}, fmt.Errorf("agent: max steps (%d) reached", maxSteps)
}
