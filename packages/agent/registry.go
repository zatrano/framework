package agent

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/zatrano/framework/packages/ai"
)

// Registry maps tool names to handlers and builds ai.Tool definitions.
type Registry struct {
	mu       sync.RWMutex
	defs     []ai.Tool
	handlers map[string]Handler
}

// NewRegistry creates an empty tool registry.
func NewRegistry() *Registry {
	return &Registry{handlers: make(map[string]Handler)}
}

// Register adds a tool definition and handler (name from tool.Function.Name).
func (r *Registry) Register(tool ai.Tool, h Handler) error {
	if r == nil {
		return fmt.Errorf("agent: registry is nil")
	}
	if h == nil {
		return fmt.Errorf("agent: handler is nil")
	}
	name := strings.TrimSpace(tool.Function.Name)
	if name == "" {
		return fmt.Errorf("agent: tool name is required")
	}
	if tool.Type == "" {
		tool.Type = "function"
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.handlers == nil {
		r.handlers = make(map[string]Handler)
	}
	if _, exists := r.handlers[name]; exists {
		return fmt.Errorf("agent: tool %q already registered", name)
	}
	r.handlers[name] = h
	r.defs = append(r.defs, tool)
	return nil
}

// Tools returns a copy of registered tool definitions.
func (r *Registry) Tools() []ai.Tool {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return append([]ai.Tool(nil), r.defs...)
}

// Execute runs the handler for call.Function.Name.
func (r *Registry) Execute(ctx context.Context, call ai.ToolCall) (string, error) {
	if r == nil {
		return "", fmt.Errorf("agent: registry is nil")
	}
	name := strings.TrimSpace(call.Function.Name)
	r.mu.RLock()
	h := r.handlers[name]
	r.mu.RUnlock()
	if h == nil {
		return "", fmt.Errorf("agent: unknown tool %q", name)
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return h(ctx, call)
}
