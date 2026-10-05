package bench

import (
	"bytes"
	"testing"
)

func BenchmarkCanonT0(b *testing.B)     { benchN(b, serveZatranoTier0Canon, request) }
func BenchmarkCanonT1(b *testing.B)     { benchN(b, serveZatranoTier1Run, requestNoID) }
func BenchmarkCanonT1Echo(b *testing.B) { benchN(b, serveZatranoTier1Run, request) }

func BenchmarkT0Old(b *testing.B)       { benchN(b, serveZatranoTier0, request) }
func BenchmarkT0Run(b *testing.B)       { benchN(b, serveZatranoTier0Run, request) }
func BenchmarkT0RunV301(b *testing.B)   { benchN(b, serveZatranoTier0RunV301, request) }
func BenchmarkT1OldEcho(b *testing.B)   { benchN(b, serveZatranoTier1, request) }
func BenchmarkT1RunEcho(b *testing.B)   { benchN(b, serveZatranoTier1Run, request) }
func BenchmarkT1RunGen(b *testing.B)    { benchN(b, serveZatranoTier1Run, requestNoID) }
func BenchmarkT0Fiber(b *testing.B)     { benchN(b, serveFiberTier0, request) }
func BenchmarkT1FiberEcho(b *testing.B) { benchN(b, serveFiberTier1, request) }
func BenchmarkT1FiberGen(b *testing.B)  { benchN(b, serveFiberTier1Gen, requestNoID) }
func BenchmarkT0Gin(b *testing.B)       { benchN(b, serveGinTier0, request) }
func BenchmarkT1Gin(b *testing.B)       { benchN(b, serveGinTier1, request) }
func BenchmarkT0EchoFrm(b *testing.B)   { benchN(b, serveEchoTier0, request) }
func BenchmarkT1EchoFrm(b *testing.B)   { benchN(b, serveEchoTier1, request) }

func benchN(b *testing.B, serve serveFunc, req []byte) {
	b.Helper()
	warm, wbuf := captureRequest(req)
	if err := serve(warm); wbuf.Len() == 0 {
		b.Fatalf("warmup: %v", err)
	}
	payload := bytes.Repeat(req, b.N)
	conn := &memConn{r: bytes.NewReader(payload)}
	b.ReportAllocs()
	b.ResetTimer()
	_ = serve(conn)
}
