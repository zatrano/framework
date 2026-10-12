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
