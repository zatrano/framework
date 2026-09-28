package http

import "github.com/zatrano/rawhttp"

func (r *Request) methodBytes() []byte {
	if r == nil || r.ctx == nil {
		return nil
	}
	return r.ctx.Method
}

func (r *Request) pathBytes() []byte {
	if r == nil || r.ctx == nil {
		return nil
	}
	return r.ctx.Path
}

// Ensure rawhttp import used when helpers expand.
var _ = rawhttp.ErrServerClosed
