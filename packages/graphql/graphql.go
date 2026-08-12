package graphql

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/zatrano/framework/packages/http"
	"github.com/zatrano/framework/packages/routing"
)

// Resolver resolves a GraphQL field.
type Resolver func(args map[string]any) (any, error)

// Schema is a minimal GraphQL-like field registry.
type Schema struct {
	queries   map[string]Resolver
	mutations map[string]Resolver
}

// NewSchema creates an empty schema.
func NewSchema() *Schema {
	return &Schema{
		queries:   map[string]Resolver{},
		mutations: map[string]Resolver{},
	}
}

// Query registers a query field.
func (s *Schema) Query(name string, resolver Resolver) {
	s.queries[name] = resolver
}

// Mutation registers a mutation field.
func (s *Schema) Mutation(name string, resolver Resolver) {
	s.mutations[name] = resolver
}

// Request is a GraphQL HTTP payload.
type Request struct {
	Query         string         `json:"query"`
	OperationName string         `json:"operationName"`
	Variables     map[string]any `json:"variables"`
}

// Response is a GraphQL HTTP response.
type Response struct {
	Data   map[string]any   `json:"data,omitempty"`
	Errors []map[string]any `json:"errors,omitempty"`
}

// Execute runs a very small subset of GraphQL queries/mutations.
// Supported shapes:
//
//	{ health }
//	{ echo(message: "hi") }
//	mutation { ping }
func (s *Schema) Execute(query string, variables map[string]any) *Response {
	query = strings.TrimSpace(query)
	isMutation := strings.HasPrefix(strings.ToLower(query), "mutation")
	body := extractSelection(query)
	fields := parseFields(body)

	data := map[string]any{}
	errors := make([]map[string]any, 0)
	registry := s.queries
	if isMutation {
		registry = s.mutations
	}

	for _, field := range fields {
		resolver, ok := registry[field.Name]
		if !ok {
			errors = append(errors, map[string]any{
				"message": fmt.Sprintf("Cannot query field %q", field.Name),
			})
			continue
		}
		args := field.Args
		for key, value := range variables {
			if _, exists := args[key]; !exists {
				args[key] = value
			}
		}
		result, err := resolver(args)
		if err != nil {
			errors = append(errors, map[string]any{"message": err.Error()})
			continue
		}
		data[field.Name] = result
	}

	resp := &Response{}
	if len(data) > 0 {
		resp.Data = data
	}
	if len(errors) > 0 {
		resp.Errors = errors
	}
	return resp
}

// Handler serves GraphQL over HTTP (GET query=... or JSON POST).
func (s *Schema) Handler() routing.HandlerFunc {
	return func(req *http.Request) *http.Response {
		var payload Request
		if req.Method() == "GET" {
			payload.Query = req.Query("query")
		} else {
			if err := req.JSON(&payload); err != nil {
				// fallback: raw body already consumed; try All()
				payload.Query = req.Input("query")
			}
		}
		if strings.TrimSpace(payload.Query) == "" {
			return http.JSON(Response{Errors: []map[string]any{{"message": "query is required"}}}).Status(400)
		}
		result := s.Execute(payload.Query, payload.Variables)
		status := 200
		if len(result.Errors) > 0 && result.Data == nil {
			status = 400
		}
		return http.JSON(result).Status(status)
	}
}

type fieldCall struct {
	Name string
	Args map[string]any
}

func extractSelection(query string) string {
	start := strings.Index(query, "{")
	end := strings.LastIndex(query, "}")
	if start < 0 || end <= start {
		return query
	}
	return strings.TrimSpace(query[start+1 : end])
}

func parseFields(body string) []fieldCall {
	parts := splitTopLevel(body)
	out := make([]fieldCall, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		name := part
		args := map[string]any{}
		if i := strings.Index(part, "("); i >= 0 && strings.HasSuffix(part, ")") {
			name = strings.TrimSpace(part[:i])
			rawArgs := part[i+1 : len(part)-1]
			args = parseArgs(rawArgs)
		}
		out = append(out, fieldCall{Name: name, Args: args})
	}
	return out
}

func splitTopLevel(body string) []string {
	parts := []string{}
	current := strings.Builder{}
	depth := 0
	for _, r := range body {
		switch r {
		case '(':
			depth++
			current.WriteRune(r)
		case ')':
			depth--
			current.WriteRune(r)
		case ' ', '\n', '\t', ',':
			if depth == 0 {
				if current.Len() > 0 {
					parts = append(parts, current.String())
					current.Reset()
				}
				continue
			}
			current.WriteRune(r)
		default:
			current.WriteRune(r)
		}
	}
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}
	return parts
}

func parseArgs(raw string) map[string]any {
	out := map[string]any{}
	if strings.TrimSpace(raw) == "" {
		return out
	}
	pieces := strings.Split(raw, ",")
	for _, piece := range pieces {
		piece = strings.TrimSpace(piece)
		kv := strings.SplitN(piece, ":", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.TrimSpace(kv[0])
		value := strings.TrimSpace(kv[1])
		out[key] = decodeScalar(value)
	}
	return out
}

func decodeScalar(value string) any {
	if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
		var s string
		if err := json.Unmarshal([]byte(value), &s); err == nil {
			return s
		}
		return strings.Trim(value, `"`)
	}
	switch value {
	case "true":
		return true
	case "false":
		return false
	case "null":
		return nil
	}
	var n json.Number
	if err := json.Unmarshal([]byte(value), &n); err == nil {
		if i, err := n.Int64(); err == nil {
			return i
		}
		if f, err := n.Float64(); err == nil {
			return f
		}
	}
	return value
}
