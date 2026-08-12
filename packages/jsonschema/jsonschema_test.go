package jsonschema_test

import (
	"testing"

	"github.com/zatrano/framework/packages/jsonschema"
)

func ptrFloat(v float64) *float64 { return &v }
func ptrInt(v int) *int           { return &v }
func ptrBool(v bool) *bool        { return &v }

func TestValidateObject(t *testing.T) {
	schema := jsonschema.Schema{
		Type:     "object",
		Required: []string{"email"},
		Properties: map[string]jsonschema.Schema{
			"email": {Type: "string"},
			"age":   {Type: "integer"},
		},
	}
	errs := jsonschema.Validate(schema, map[string]any{"age": 20})
	if len(errs) == 0 {
		t.Fatal("expected required error")
	}
	if !jsonschema.Valid(schema, map[string]any{"email": "a@b.c", "age": 20}) {
		t.Fatal("expected valid payload")
	}
}

func TestValidateArray(t *testing.T) {
	schema := jsonschema.Schema{
		Type:  "array",
		Items: &jsonschema.Schema{Type: "string"},
	}
	if jsonschema.Valid(schema, []any{"a", 1}) {
		t.Fatal("expected type error on item")
	}
	if !jsonschema.Valid(schema, []any{"a", "b"}) {
		t.Fatal("expected valid array")
	}
}

func TestValidateEnumMinMaxLengthPatternAdditional(t *testing.T) {
	schema := jsonschema.Schema{
		Type: "object",
		Properties: map[string]jsonschema.Schema{
			"role": {
				Type: "string",
				Enum: []any{"admin", "user"},
			},
			"age": {
				Type:    "integer",
				Minimum: ptrFloat(18),
				Maximum: ptrFloat(120),
			},
			"code": {
				Type:      "string",
				MinLength: ptrInt(2),
				MaxLength: ptrInt(8),
				Pattern:   `^[A-Z]+$`,
			},
		},
		AdditionalProperties: ptrBool(false),
	}

	if !jsonschema.Valid(schema, map[string]any{
		"role": "admin",
		"age":  30,
		"code": "AB",
	}) {
		t.Fatal("expected valid")
	}

	errs := jsonschema.Validate(schema, map[string]any{
		"role":  "guest",
		"age":   10,
		"code":  "a",
		"extra": true,
	})
	if len(errs) < 4 {
		t.Fatalf("expected multiple errors, got %v", errs)
	}
}
