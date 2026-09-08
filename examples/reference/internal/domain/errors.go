package domain

import "errors"

// Sentinel application errors. HTTP mapping uses errors.Is against these
// values so infrastructure wrapping (%w) stays intact.
var (
	ErrNotFound       = errors.New("item not found")
	ErrInvalidRequest = errors.New("invalid request")
	ErrUnavailable    = errors.New("dependency unavailable")
	ErrConfiguration  = errors.New("configuration failure")
)
