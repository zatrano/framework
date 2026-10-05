package bench

import (
	"bytes"
	"testing"
)

// Canonical rows use the deadline-faithful connection. Free* rows are the
// same servers on memConn, whose SetDeadline returns without arming a timer.
func BenchmarkCanonT0(b *testing.B)     { benchFaith(b, serveZatranoTier0Canon, request) }
func BenchmarkCanonT1(b *testing.B)     { benchFaith(b, serveZatranoTier1Run, requestNoID) }
func BenchmarkCanonT1Echo(b *testing.B) { benchFaith(b, serveZatranoTier1Run, request) }

func BenchmarkCanonT0Fiber(b *testing.B)       { benchFaith(b, serveFiberTier0, request) }
func BenchmarkCanonT0FiberEq(b *testing.B)     { benchFaith(b, serveFiberTier0Eq, request) }
func BenchmarkCanonT1FiberEcho(b *testing.B)   { benchFaith(b, serveFiberTier1, request) }
func BenchmarkCanonT1FiberEqEcho(b *testing.B) { benchFaith(b, serveFiberTier1Eq, request) }
func BenchmarkCanonT1FiberGen(b *testing.B)    { benchFaith(b, serveFiberTier1Gen, requestNoID) }
func BenchmarkCanonT1FiberEqGen(b *testing.B)  { benchFaith(b, serveFiberTier1GenEq, requestNoID) }

func BenchmarkFreeCanonT0(b *testing.B)       { benchN(b, serveZatranoTier0Canon, request) }
func BenchmarkFreeCanonT1(b *testing.B)       { benchN(b, serveZatranoTier1Run, requestNoID) }
func BenchmarkFreeCanonT1Echo(b *testing.B)   { benchN(b, serveZatranoTier1Run, request) }
func BenchmarkFreeT0FiberEq(b *testing.B)     { benchN(b, serveFiberTier0Eq, request) }
func BenchmarkFreeT1FiberEqEcho(b *testing.B) { benchN(b, serveFiberTier1Eq, request) }
func BenchmarkFreeT1FiberEqGen(b *testing.B)  { benchN(b, serveFiberTier1GenEq, requestNoID) }

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

func benchFaith(b *testing.B, serve serveFunc, req []byte) {
	b.Helper()
	warm, wbuf, err := captureFaith(req)
	if err != nil {
		b.Fatal(err)
	}
	if err := serve(warm); wbuf.Len() == 0 {
		b.Fatalf("warmup: %v", err)
	}
	payload := bytes.Repeat(req, b.N)
	conn, err := newFaith(bytes.NewReader(payload), nil)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	_ = serve(conn)
}
