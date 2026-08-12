package enums

import (
	"fmt"
	"strings"
)

// String is a backed string enumeration.
type String struct {
	name   string
	values map[string]string // value -> label
}

// NewString creates a string enum with optional value=>label pairs.
// values may be provided as "draft", "published" or "draft:Draft".
func NewString(name string, values ...string) *String {
	e := &String{name: name, values: make(map[string]string)}
	for _, raw := range values {
		value, label := splitEnum(raw)
		e.values[value] = label
	}
	return e
}

// Name returns the enum name.
func (e *String) Name() string { return e.name }

// Contains reports whether value is a valid case.
func (e *String) Contains(value string) bool {
	_, ok := e.values[value]
	return ok
}

// Label returns the label for a value.
func (e *String) Label(value string) string {
	if label, ok := e.values[value]; ok {
		return label
	}
	return value
}

// Values returns all valid values.
func (e *String) Values() []string {
	out := make([]string, 0, len(e.values))
	for value := range e.values {
		out = append(out, value)
	}
	return out
}

// Cases returns value/label pairs.
func (e *String) Cases() map[string]string {
	out := make(map[string]string, len(e.values))
	for k, v := range e.values {
		out[k] = v
	}
	return out
}

// Assert panics if value is invalid.
func (e *String) Assert(value string) string {
	if !e.Contains(value) {
		panic(fmt.Sprintf("invalid %s value [%s]", e.name, value))
	}
	return value
}

// Try returns value or error.
func (e *String) Try(value string) (string, error) {
	if !e.Contains(value) {
		return "", fmt.Errorf("invalid %s value [%s]", e.name, value)
	}
	return value, nil
}

// Registry holds named enums.
type Registry struct {
	items map[string]*String
}

// NewRegistry creates an enum registry.
func NewRegistry() *Registry {
	return &Registry{items: make(map[string]*String)}
}

// Register stores an enum.
func (r *Registry) Register(enum *String) *String {
	r.items[enum.Name()] = enum
	return enum
}

// Get returns a registered enum.
func (r *Registry) Get(name string) (*String, bool) {
	e, ok := r.items[name]
	return e, ok
}

// All returns all enums.
func (r *Registry) All() map[string]*String {
	out := make(map[string]*String, len(r.items))
	for k, v := range r.items {
		out[k] = v
	}
	return out
}

func splitEnum(raw string) (value, label string) {
	parts := strings.SplitN(raw, ":", 2)
	value = strings.TrimSpace(parts[0])
	if len(parts) == 2 {
		label = strings.TrimSpace(parts[1])
	} else {
		label = value
	}
	return value, label
}
