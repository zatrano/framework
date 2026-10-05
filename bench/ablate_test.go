package bench

import (
	"net"
	"testing"

	khttp "github.com/zatrano/framework/v3/core/kernel/http"
	"github.com/zatrano/rawhttp"
)

// Single-factor rows explain the free-deadline gap between the frozen old
// server and the frozen Run shape. Each row changes one field on serverOld
// (Read/Write/Idle = -1). AblateFull is the whole Run shape. The driver is
// memConn, the same free-deadline connection the gap was measured on.
//
// Concurrency is unset in production (0 → carrier default). AblateConcurrency
// sets the carrier default explicitly so a difference would show up; a match
// with AblateBase means that field is not part of the gap.
// AllowUpgrade false is the zero value. The row sets it explicitly.

func ablate(mut func(*rawhttp.Server)) serveFunc {
	return func(conn net.Conn) error {
		r, err := tier0CanonRouter()
		if err != nil {
			return err
		}
		srv := serverOld(func(ctx *rawhttp.Ctx) {
			req := khttp.NewRequest(ctx)
			resp := r.Dispatch(req)
			if resp == nil {
				ctx.SetStatusCode(404)
				ctx.SetBodyString("Not Found")
				return
			}
			_ = resp.Commit(ctx)
		})
		if mut != nil {
			mut(srv)
		}
		return srv.ServeConn(conn)
	}
}

func BenchmarkAblateBase(b *testing.B) { benchN(b, ablate(nil), request) }

func BenchmarkAblateReadHeader(b *testing.B) {
	benchN(b, ablate(func(s *rawhttp.Server) {
		s.ReadHeaderTimeout = runReadHeaderTimeout
	}), request)
}

func BenchmarkAblateRead(b *testing.B) {
	benchN(b, ablate(func(s *rawhttp.Server) { s.ReadTimeout = runReadTimeout }), request)
}

func BenchmarkAblateWrite(b *testing.B) {
	benchN(b, ablate(func(s *rawhttp.Server) { s.WriteTimeout = runWriteTimeout }), request)
}

func BenchmarkAblateIdle(b *testing.B) {
	benchN(b, ablate(func(s *rawhttp.Server) { s.IdleTimeout = runIdleTimeout }), request)
}

func BenchmarkAblateTimeouts(b *testing.B) {
	benchN(b, ablate(func(s *rawhttp.Server) {
		s.ReadHeaderTimeout = runReadHeaderTimeout
		s.ReadTimeout = runReadTimeout
		s.WriteTimeout = runWriteTimeout
		s.IdleTimeout = runIdleTimeout
	}), request)
}

func BenchmarkAblateMaxHeader(b *testing.B) {
	benchN(b, ablate(func(s *rawhttp.Server) { s.MaxHeaderBytes = runMaxHeaderBytes }), request)
}

func BenchmarkAblateReadBuffer(b *testing.B) {
	benchN(b, ablate(func(s *rawhttp.Server) { s.ReadBufferSize = runMaxHeaderBytes }), request)
}

func BenchmarkAblateHijack(b *testing.B) {
	benchN(b, ablate(func(s *rawhttp.Server) { s.KeepHijackedConns = true }), request)
}

func BenchmarkAblateConcurrency(b *testing.B) {
	benchN(b, ablate(func(s *rawhttp.Server) { s.Concurrency = 256 * 1024 }), request)
}

func BenchmarkAblateHook(b *testing.B) {
	benchN(b, ablate(func(s *rawhttp.Server) { s.HeaderReceived = headFastPath }), request)
}

func BenchmarkAblateMaxBody(b *testing.B) {
	benchN(b, ablate(func(s *rawhttp.Server) { s.MaxRequestBodySize = runMaxBodyBytes }), request)
}

func BenchmarkAblateUpgrade(b *testing.B) {
	benchN(b, ablate(func(s *rawhttp.Server) { s.AllowUpgrade = false }), request)
}

func BenchmarkAblateFull(b *testing.B) { benchN(b, serveZatranoTier0Canon, request) }
