package jsonschema

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
)

// Schema is a JSON Schema subset for request validation.
type Schema struct {
	Type                 string            `json:"type,omitempty"`
	Properties           map[string]Schema `json:"properties,omitempty"`
	Required             []string          `json:"required,omitempty"`
	Items                *Schema           `json:"items,omitempty"`
	Enum                 []any             `json:"enum,omitempty"`
	Minimum              *float64          `json:"minimum,omitempty"`
	Maximum              *float64          `json:"maximum,omitempty"`
	MinLength            *int              `json:"minLength,omitempty"`
	MaxLength            *int              `json:"maxLength,omitempty"`
	Pattern              string            `json:"pattern,omitempty"`
	AdditionalProperties *bool             `json:"additionalProperties,omitempty"`
}

// ValidationError describes a single schema violation.
type ValidationError struct {
	Path    string
	Message string
}

func (e ValidationError) Error() string {
	if e.Path == "" {
		return e.Message
	}
	return e.Path + ": " + e.Message
}

// Validate checks data against schema and returns all errors.
func Validate(schema Schema, data any) []error {
	var errs []error
	validate("", schema, data, &errs)
	return errs
}

// Valid reports whether data matches schema.
func Valid(schema Schema, data any) bool {
	return len(Validate(schema, data)) == 0
}

func validate(path string, schema Schema, data any, errs *[]error) {
	if schema.Type != "" && !typeMatches(schema.Type, data) {
		*errs = append(*errs, ValidationError{Path: path, Message: "expected type " + schema.Type})
		return
	}

	if len(schema.Enum) > 0 && !enumMatches(schema.Enum, data) {
		*errs = append(*errs, ValidationError{Path: path, Message: "value not in enum"})
	}

	switch strings.ToLower(schema.Type) {
	case "object", "":
		obj, ok := asMap(data)
		if schema.Type == "object" && !ok {
			return
		}
		if ok {
			for _, key := range schema.Required {
				if _, exists := obj[key]; !exists {
					*errs = append(*errs, ValidationError{
						Path:    join(path, key),
						Message: "required",
					})
				}
			}
			for key, prop := range schema.Properties {
				if v, exists := obj[key]; exists {
					validate(join(path, key), prop, v, errs)
				}
			}
			if schema.AdditionalProperties != nil && !*schema.AdditionalProperties {
				for key := range obj {
					if _, known := schema.Properties[key]; !known {
						*errs = append(*errs, ValidationError{
							Path:    join(path, key),
							Message: "additional property not allowed",
						})
					}
				}
			}
		}
	case "array":
		rv := reflect.ValueOf(data)
		if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
			return
		}
		if schema.Items != nil {
			for i := 0; i < rv.Len(); i++ {
				validate(fmt.Sprintf("%s[%d]", path, i), *schema.Items, rv.Index(i).Interface(), errs)
			}
		}
	case "string":
		s, ok := data.(string)
		if !ok {
			return
		}
		if schema.MinLength != nil && len(s) < *schema.MinLength {
			*errs = append(*errs, ValidationError{Path: path, Message: fmt.Sprintf("minLength %d", *schema.MinLength)})
		}
		if schema.MaxLength != nil && len(s) > *schema.MaxLength {
			*errs = append(*errs, ValidationError{Path: path, Message: fmt.Sprintf("maxLength %d", *schema.MaxLength)})
		}
		if schema.Pattern != "" {
			re, err := regexp.Compile(schema.Pattern)
			if err != nil {
				*errs = append(*errs, ValidationError{Path: path, Message: "invalid pattern"})
			} else if !re.MatchString(s) {
				*errs = append(*errs, ValidationError{Path: path, Message: "pattern mismatch"})
			}
		}
	case "number", "integer":
		n, ok := asFloat(data)
		if !ok {
			return
		}
		if schema.Minimum != nil && n < *schema.Minimum {
			*errs = append(*errs, ValidationError{Path: path, Message: fmt.Sprintf("minimum %v", *schema.Minimum)})
		}
		if schema.Maximum != nil && n > *schema.Maximum {
			*errs = append(*errs, ValidationError{Path: path, Message: fmt.Sprintf("maximum %v", *schema.Maximum)})
		}
	}

	// Constraints may apply even when type is omitted.
	if schema.Type == "" {
		if s, ok := data.(string); ok {
			if schema.MinLength != nil && len(s) < *schema.MinLength {
				*errs = append(*errs, ValidationError{Path: path, Message: fmt.Sprintf("minLength %d", *schema.MinLength)})
			}
			if schema.MaxLength != nil && len(s) > *schema.MaxLength {
				*errs = append(*errs, ValidationError{Path: path, Message: fmt.Sprintf("maxLength %d", *schema.MaxLength)})
			}
			if schema.Pattern != "" {
				if re, err := regexp.Compile(schema.Pattern); err == nil && !re.MatchString(s) {
					*errs = append(*errs, ValidationError{Path: path, Message: "pattern mismatch"})
				}
			}
		}
		if n, ok := asFloat(data); ok {
			if schema.Minimum != nil && n < *schema.Minimum {
				*errs = append(*errs, ValidationError{Path: path, Message: fmt.Sprintf("minimum %v", *schema.Minimum)})
			}
			if schema.Maximum != nil && n > *schema.Maximum {
				*errs = append(*errs, ValidationError{Path: path, Message: fmt.Sprintf("maximum %v", *schema.Maximum)})
			}
		}
	}
}

func enumMatches(enum []any, data any) bool {
	for _, want := range enum {
		if reflect.DeepEqual(want, data) {
			return true
		}
		// JSON numbers often decode as float64; allow int/float equality.
		wf, wok := asFloat(want)
		df, dok := asFloat(data)
		if wok && dok && wf == df {
			return true
		}
		if fmt.Sprint(want) == fmt.Sprint(data) && reflect.TypeOf(want) == reflect.TypeOf(data) {
			return true
		}
	}
	return false
}

func asFloat(data any) (float64, bool) {
	switch v := data.(type) {
	case int:
		return float64(v), true
	case int8:
		return float64(v), true
	case int16:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	case uint:
		return float64(v), true
	case uint8:
		return float64(v), true
	case uint16:
		return float64(v), true
	case uint32:
		return float64(v), true
	case uint64:
		return float64(v), true
	case float32:
		return float64(v), true
	case float64:
		return v, true
	default:
		return 0, false
	}
}

func typeMatches(want string, data any) bool {
	if data == nil {
		return strings.EqualFold(want, "null")
	}
	switch strings.ToLower(want) {
	case "object":
		_, ok := asMap(data)
		return ok
	case "array":
		k := reflect.ValueOf(data).Kind()
		return k == reflect.Slice || k == reflect.Array
	case "string":
		_, ok := data.(string)
		return ok
	case "number", "integer":
		switch data.(type) {
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
			if strings.EqualFold(want, "integer") {
				switch v := data.(type) {
				case float64:
					return v == float64(int64(v))
				case float32:
					return v == float32(int64(v))
				}
			}
			return true
		default:
			return false
		}
	case "boolean":
		_, ok := data.(bool)
		return ok
	case "null":
		return data == nil
	default:
		return true
	}
}

func asMap(data any) (map[string]any, bool) {
	switch v := data.(type) {
	case map[string]any:
		return v, true
	default:
		return nil, false
	}
}

func join(path, key string) string {
	if path == "" {
		return key
	}
	return path + "." + key
}
