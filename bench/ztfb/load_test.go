package main

import (
	"bufio"
	"bytes"
	"testing"
	"time"
)

func TestReadCheckedAcceptsContentLengthAndBody(t *testing.T) {
	raw := []byte("HTTP/1.1 200 OK\r\nContent-Type: text/plain; charset=utf-8\r\nContent-Length: 13\r\n\r\nHello, World!")
	if err := readChecked(bufio.NewReader(bytes.NewReader(raw)), []byte("Hello, World!")); err != nil {
		t.Fatal(err)
	}
}

func TestReadCheckedRejectsWrongLength(t *testing.T) {
	raw := []byte("HTTP/1.1 200 OK\r\nContent-Length: 2\r\n\r\nHi")
	if err := readChecked(bufio.NewReader(bytes.NewReader(raw)), []byte("Hello, World!")); err == nil {
		t.Fatal("expected length mismatch")
	}
}

func TestPercentileIsOrdered(t *testing.T) {
	samples := []int64{30, 10, 20, 40}
	if got := percentile(append([]int64(nil), samples...), 0.50); got != 20*time.Nanosecond && got != 30*time.Nanosecond {
		t.Fatalf("p50 %s", got)
	}
	if got := percentile(append([]int64(nil), samples...), 0.99); got != 40*time.Nanosecond {
		t.Fatalf("p99 %s", got)
	}
}

func TestMedianEven(t *testing.T) {
	if got := medianFloat([]float64{1, 4, 2, 9}); got != 3 {
		t.Fatalf("median %v", got)
	}
}

func TestDefaultRoundsIsTen(t *testing.T) {
	if defaultRounds < 10 || bootstrapDraws < 10000 {
		t.Fatalf("rounds %d draws %d", defaultRounds, bootstrapDraws)
	}
}

func TestPairedRatioUsesRoundsNotMedianOfMedians(t *testing.T) {
	z := []cell{{Round: 1, RPS: 100}, {Round: 2, RPS: 200}}
	f := []cell{{Round: 2, RPS: 100}, {Round: 1, RPS: 200}}
	got, err := pairedRPSRatios(z, f)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != 0.5 || got[1] != 2 {
		t.Fatalf("ratios %v", got)
	}
	median, low, high := bootstrapMedianCI(got, bootstrapDraws, bootstrapSeed)
	if median != 1.25 {
		t.Fatalf("median %v", median)
	}
	if low > 1 || high < 1 {
		t.Fatalf("CI that must contain 1: %v %v", low, high)
	}
	if cellVerdict(low, high, 0.9, 0.9) != "esit" {
		t.Fatal("a CI that contains 1 is equal, not ahead")
	}
}

func TestVerdictRequiresCIEntirelyAboveOne(t *testing.T) {
	if cellVerdict(1.01, 1.08, 0.99, 1) != "ustun" {
		t.Fatal("expected ustun")
	}
	if cellVerdict(1.01, 1.08, 1.01, 0.9) != "esik-yok" {
		t.Fatal("p99 miss is not a win")
	}
	if cellVerdict(1, 1.08, 0.9, 0.9) != "esit" {
		t.Fatal("lower bound of 1 still contains 1")
	}
	if cellVerdict(0.8, 0.95, 0.9, 0.9) != "fiber" {
		t.Fatal("expected fiber")
	}
}

func TestBootstrapCICollapsesOnIdenticalSamples(t *testing.T) {
	samples := []float64{1.2, 1.2, 1.2, 1.2, 1.2, 1.2, 1.2, 1.2, 1.2, 1.2}
	median, low, high := bootstrapMedianCI(samples, bootstrapDraws, bootstrapSeed)
	if median != 1.2 || low != 1.2 || high != 1.2 {
		t.Fatalf("median %v ci %v %v", median, low, high)
	}
}

func TestParseWrk(t *testing.T) {
	out := `
  Latency Distribution
     50%    1.10ms
     99%   10.00ms
  50000 requests in 5.00s, 1.00MB read
Requests/sec:  10000.00
Non-2xx or 3xx responses: 3
Socket errors: connect 0, read 1, write 0, timeout 0
`
	got, err := parseWrk(out)
	if err != nil {
		t.Fatal(err)
	}
	if got.RPS() != 10000 || got.Requests != 50000 || got.Errors != 4 {
		t.Fatalf("%+v", got)
	}
	if got.P50 != 1100*time.Microsecond || got.P99 != 10*time.Millisecond {
		t.Fatalf("p50 %s p99 %s", got.P50, got.P99)
	}
}
