package validation

// ValidationError holds structured validation failures returned by Validate[T].
type ValidationError struct {
	Errors []FieldError `json:"errors"`
}

// Error implements the error interface.
func (ve *ValidationError) Error() string {
	if len(ve.Errors) == 0 {
		return "validation failed"
	}
	return "validation failed: " + ve.Errors[0].Field + " " + ve.Errors[0].Message
}

// FieldError describes a single field validation failure.
type FieldError struct {
	Field   string `json:"field"`
	Tag     string `json:"tag"`
	Value   any    `json:"value,omitempty"`
	Message string `json:"message"`
}
