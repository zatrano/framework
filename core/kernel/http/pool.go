package http

import (
	"context"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/zatrano/rawhttp"
)

// Request pooling is on unless HTTP_POOL_REQUESTS=false. A pooled request is
// reset and reused after the handler returns. Do not keep the pointer.
// With debug on, a method call after release panics and reports the generation.
var (
	pooling   atomic.Bool
	poolDebug atomic.Bool
	reqPool   = sync.Pool{New: func() any { return &Request{pooled: true} }}
	respPool  = sync.Pool{New: func() any { return &Response{pooled: true} }}
)

func init() {
	pooling.Store(true)
}

// SetRequestPooling turns the process-wide request and response pool on or off.
// debug makes a use-after-release call panic. The default is on.
func SetRequestPooling(on, debug bool) {
	pooling.Store(on)
	poolDebug.Store(debug)
}

// RequestPooling reports whether AcquireRequest draws from the pool.
func RequestPooling() bool { return pooling.Load() }

// AcquireRequest returns a request for ctx. With pooling off this is NewRequest.
func AcquireRequest(ctx *rawhttp.Ctx) *Request {
	if !pooling.Load() {
		return NewRequest(ctx)
	}
	r := reqPool.Get().(*Request)
	r.Reset(ctx)
	return r
}

// ReleaseRequest returns a pooled request. A request that was not acquired
// from the pool is left alone.
func ReleaseRequest(r *Request) {
	if r == nil || !r.pooled {
		return
	}
	gen := r.gen + 1
	*r = Request{pooled: true, gen: gen, released: poolDebug.Load()}
	reqPool.Put(r)
}

// Reset prepares a pooled request for another connection buffer.
func (r *Request) Reset(ctx *rawhttp.Ctx) {
	if r == nil {
		return
	}
	gen := r.gen + 1
	*r = Request{
		ctx:      ctx,
		stdCtx:   context.Background(),
		pooled:   true,
		gen:      gen,
		released: false,
	}
}

// Generation is the reuse counter. It changes when the request is released.
func (r *Request) Generation() uint64 {
	if r == nil {
		return 0
	}
	return r.gen
}

func (r *Request) touch() {
	if r != nil && r.released {
		panic("http: request used after release (generation " + strconv.FormatUint(r.gen, 10) + ")")
	}
}

func newResponse() *Response {
	if !pooling.Load() {
		return &Response{}
	}
	r := respPool.Get().(*Response)
	*r = Response{pooled: true}
	return r
}

// ReleaseResponse returns a pooled response after Commit. The JSON bytes are
// not reused: rawhttp still aliases them until it writes, which happens after
// Handle returns. Debug mode panics if a method runs after this release.
func ReleaseResponse(r *Response) {
	if r == nil || !r.pooled {
		return
	}
	*r = Response{pooled: true, released: poolDebug.Load()}
	respPool.Put(r)
}
