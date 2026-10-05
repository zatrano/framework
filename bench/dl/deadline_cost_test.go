// Real net.Conn deadline cost. rawhttp sets ~4 deadlines per keep-alive request
// (idle, header x2, write). In-memory fake connections make those calls free,
// so in-process benchmarks undercount them. Run on the publishing machine:
//   go test -run "^$" -bench . -benchtime=1s -count=10
// Linux Xeon 2.8 GHz reference: time.Now 57 ns; SetRead/WriteDeadline ~185 ns each;
// four per request ~780 ns (> the whole bare ServeConn request, ~350 ns).
package dl

import (
	"net"
	"testing"
	"time"
)

func conn(b *testing.B) net.Conn {
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	go func() { c, _ := ln.Accept(); _ = c }()
	c, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() { c.Close(); ln.Close() })
	return c
}

func BenchmarkTimeNow(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = time.Now()
	}
}
func BenchmarkSetReadDeadlineReal(b *testing.B) {
	c := conn(b)
	for i := 0; i < b.N; i++ {
		_ = c.SetReadDeadline(time.Now().Add(10 * time.Second))
	}
}
func BenchmarkSetWriteDeadlineReal(b *testing.B) {
	c := conn(b)
	for i := 0; i < b.N; i++ {
		_ = c.SetWriteDeadline(time.Now().Add(60 * time.Second))
	}
}
func BenchmarkFourDeadlinesPerRequest(b *testing.B) {
	c := conn(b)
	for i := 0; i < b.N; i++ {
		_ = c.SetReadDeadline(time.Now().Add(120 * time.Second))
		_ = c.SetReadDeadline(time.Now().Add(10 * time.Second))
		_ = c.SetReadDeadline(time.Now().Add(10 * time.Second))
		_ = c.SetWriteDeadline(time.Now().Add(60 * time.Second))
	}
}
