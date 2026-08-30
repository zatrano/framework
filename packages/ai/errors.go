package ai

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
)

// Kind classifies AI errors for retry and fallback decisions.
type Kind int

const (
	// KindUnknown is an unclassified error (no retry; fallback allowed).
	KindUnknown Kind = iota
	// KindAuth is authentication/authorization failure (no retry, no fallback).
	KindAuth
	// KindInvalid is a bad request / validation error (no retry, no fallback).
	KindInvalid
	// KindRateLimit is HTTP 429 (retry with backoff, then fallback).
	KindRateLimit
	// KindUnavailable is transient failure: 5xx, timeout, network (retry, then fallback).
	KindUnavailable
	// KindContext is context cancel/deadline (no retry; fallback only when deadline + FallbackOnTimeout).
	KindContext
)

func (k Kind) String() string {
	switch k {
	case KindAuth:
		return "auth"
	case KindInvalid:
		return "invalid"
	case KindRateLimit:
		return "rate_limit"
	case KindUnavailable:
		return "unavailable"
	case KindContext:
		return "context"
	default:
		return "unknown"
	}
}

// Error is a typed AI error used for retry/fallback policy.
type Error struct {
	Kind       Kind
	Provider   string
	Status     int
	RetryAfter time.Duration
	Err        error
}

func (e *Error) Error() string {
	if e == nil {
		return "ai: error"
	}
	var b strings.Builder
	b.WriteString("ai: ")
	b.WriteString(e.Kind.String())
	if e.Provider != "" {
		b.WriteString(" provider=")
		b.WriteString(e.Provider)
	}
	if e.Status > 0 {
		fmt.Fprintf(&b, " status=%d", e.Status)
	}
	if e.Err != nil {
		b.WriteString(": ")
		b.WriteString(e.Err.Error())
	}
	return b.String()
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// HTTPError builds a typed Error from an HTTP status code.
func HTTPError(provider string, status int, body string, retryAfter time.Duration) *Error {
	return &Error{
		Kind:       KindFromStatus(status),
		Provider:   provider,
		Status:     status,
		RetryAfter: retryAfter,
		Err:        fmt.Errorf("status %d: %s", status, strings.TrimSpace(body)),
	}
}

// KindFromStatus maps HTTP status codes to Kind.
func KindFromStatus(status int) Kind {
	switch {
	case status == 401 || status == 403:
		return KindAuth
	case status == 429:
		return KindRateLimit
	case status == 408 || status == 409:
		return KindUnavailable
	case status >= 500 && status <= 599:
		return KindUnavailable
	case status == 400 || status == 404 || status == 413 || status == 415 || status == 422:
		return KindInvalid
	case status >= 400 && status < 500:
		return KindInvalid
	default:
		return KindUnknown
	}
}

// Classify returns the Kind for err (unwraps *Error and common stdlib cases).
func Classify(err error) Kind {
	if err == nil {
		return KindUnknown
	}
	var ae *Error
	if errors.As(err, &ae) && ae != nil {
		return ae.Kind
	}
	if errors.Is(err, context.Canceled) {
		return KindContext
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return KindContext
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return KindUnavailable
	}
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return KindUnavailable
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "connection reset") ||
		strings.Contains(msg, "i/o timeout") ||
		strings.Contains(msg, "tls handshake") {
		return KindUnavailable
	}
	return KindUnknown
}

// Retryable reports whether the same provider should be retried.
func Retryable(err error) bool {
	switch Classify(err) {
	case KindRateLimit, KindUnavailable:
		return true
	default:
		return false
	}
}

// Fallbackable reports whether the next provider in a profile chain may be tried.
// deadlineExceededFallback enables fallback on context deadline (manager timeout).
func Fallbackable(err error, deadlineExceededFallback bool) bool {
	switch Classify(err) {
	case KindAuth, KindInvalid:
		return false
	case KindContext:
		if errors.Is(err, context.Canceled) {
			return false
		}
		if errors.Is(err, context.DeadlineExceeded) {
			return deadlineExceededFallback
		}
		var ae *Error
		if errors.As(err, &ae) && ae != nil && ae.Err != nil {
			if errors.Is(ae.Err, context.Canceled) {
				return false
			}
			if errors.Is(ae.Err, context.DeadlineExceeded) {
				return deadlineExceededFallback
			}
		}
		return false
	default:
		return true
	}
}

// retryAfterOf extracts Retry-After from a typed Error when set.
func retryAfterOf(err error) time.Duration {
	var ae *Error
	if errors.As(err, &ae) && ae != nil && ae.RetryAfter > 0 {
		return ae.RetryAfter
	}
	return 0
}
