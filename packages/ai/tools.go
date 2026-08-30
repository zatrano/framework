package ai

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Tool describes a callable function the model may invoke.
type Tool struct {
	Type     string      `json:"type"` // "function"
	Function FunctionDef `json:"function"`
}

// FunctionDef is an OpenAI-compatible function tool definition.
type FunctionDef struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
}

// FunctionTool builds a function tool.
func FunctionTool(name, description string, parameters json.RawMessage) Tool {
	if len(parameters) == 0 {
		parameters = json.RawMessage(`{"type":"object","properties":{}}`)
	}
	return Tool{
		Type: "function",
		Function: FunctionDef{
			Name:        name,
			Description: description,
			Parameters:  parameters,
		},
	}
}

// ToolCall is a model-requested function invocation.
type ToolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"` // "function"
	Function ToolCallFunction `json:"function"`
}

// ToolCallFunction holds the function name and JSON argument string.
type ToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// UnmarshalArguments decodes Function.Arguments into dest.
func (c ToolCall) UnmarshalArguments(dest any) error {
	raw := strings.TrimSpace(c.Function.Arguments)
	if raw == "" {
		raw = "{}"
	}
	if err := json.Unmarshal([]byte(raw), dest); err != nil {
		return &Error{Kind: KindInvalid, Err: fmt.Errorf("tool call arguments: %w", err)}
	}
	return nil
}

// ToolChoice selects how tools may be used (auto / none / required / force function).
type ToolChoice struct {
	Mode string // auto | none | required | function
	Name string // function name when Mode == "function"
}

// ToolChoiceAuto lets the model decide.
func ToolChoiceAuto() *ToolChoice { return &ToolChoice{Mode: "auto"} }

// ToolChoiceNone disables tools for this turn.
func ToolChoiceNone() *ToolChoice { return &ToolChoice{Mode: "none"} }

// ToolChoiceRequired forces at least one tool call.
func ToolChoiceRequired() *ToolChoice { return &ToolChoice{Mode: "required"} }

// ToolChoiceFunction forces a specific function name.
func ToolChoiceFunction(name string) *ToolChoice {
	return &ToolChoice{Mode: "function", Name: name}
}

func (c *ToolChoice) toAPI() any {
	if c == nil {
		return nil
	}
	mode := strings.ToLower(strings.TrimSpace(c.Mode))
	switch mode {
	case "", "auto":
		return "auto"
	case "none", "required":
		return mode
	case "function":
		name := strings.TrimSpace(c.Name)
		if name == "" {
			return "auto"
		}
		return map[string]any{
			"type": "function",
			"function": map[string]string{
				"name": name,
			},
		}
	default:
		return mode
	}
}

// ToolResultMessage builds a role=tool message for a prior tool call id.
func ToolResultMessage(toolCallID, content string) Message {
	return Message{Role: "tool", Content: content, ToolCallID: toolCallID}
}

// AssistantToolCalls builds an assistant message that only contains tool calls.
func AssistantToolCalls(calls ...ToolCall) Message {
	return Message{Role: "assistant", ToolCalls: calls}
}

// HasToolCalls reports whether the assistant requested tools.
func (r *ChatResponse) HasToolCalls() bool {
	return r != nil && len(r.Message.ToolCalls) > 0
}

func applyTools(body map[string]any, tools []Tool, choice *ToolChoice) {
	if body == nil || len(tools) == 0 {
		return
	}
	body["tools"] = tools
	if choice != nil {
		body["tool_choice"] = choice.toAPI()
	}
}
