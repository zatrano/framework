package http

import "fmt"

// AbortIf returns an abort response when condition is true; otherwise nil.
func AbortIf(condition bool, status int, message ...string) *Response {
	if !condition {
		return nil
	}
	return Abort(status, message...)
}

// AbortUnless returns an abort response when condition is false; otherwise nil.
func AbortUnless(condition bool, status int, message ...string) *Response {
	return AbortIf(!condition, status, message...)
}

// Rescue runs fn and converts panics into a 500 abort response.
func Rescue(fn func() *Response) (resp *Response) {
	defer func() {
		if recovered := recover(); recovered != nil {
			resp = Abort(500, fmt.Sprintf("Server Error: %v", recovered))
		}
	}()
	return fn()
}
