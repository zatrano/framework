package http

import (
	"github.com/zatrano/rawhttp"
)

// Request and Response objects are never pooled. A *Request or *Response kept
// after the handler returns stays valid and keeps describing its own request.
// Only rawhttp's connection buffers are reused, and no public value points
// into them. The functions below remain so v3 callers keep compiling.

// SetRequestPooling has no effect.
//
// Deprecated: request and response pooling was removed. HTTP_POOL_REQUESTS is
// ignored.
func SetRequestPooling(on, debug bool) {}

// RequestPooling reports false.
//
// Deprecated: request and response pooling was removed.
func RequestPooling() bool { return false }

// AcquireRequest is NewRequest.
//
// Deprecated: use NewRequest.
func AcquireRequest(ctx *rawhttp.Ctx) *Request { return NewRequest(ctx) }

// ReleaseRequest has no effect. The request stays valid.
//
// Deprecated: requests are not pooled.
func ReleaseRequest(r *Request) {}

// ReleaseResponse has no effect. The response stays valid.
//
// Deprecated: responses are not pooled.
func ReleaseResponse(r *Response) {}

// ReleaseAfterCommit detaches req from its rawhttp context. resp is left
// unchanged.
//
// Deprecated: call Request.Detach.
func ReleaseAfterCommit(req *Request, resp *Response) { req.Detach() }

// Generation reports 0.
//
// Deprecated: requests are not pooled, so there is no reuse counter.
func (r *Request) Generation() uint64 { return 0 }

func newResponse() *Response { return &Response{} }

func (r *Response) outlivesCommit() bool {
	if r == nil || r.cold == nil {
		return false
	}
	c := r.cold
	return c.stream != nil || c.streamReader != nil || c.hijack != nil
}
