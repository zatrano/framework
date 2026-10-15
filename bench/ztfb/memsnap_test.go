package main

import (
	"net"
	"testing"
	"time"
)

func TestMemSnapReportsMallocs(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	t.Setenv("ZTFB_MEMSNAP", ln.Addr().String())
	startMemSnap()
	if tcp, ok := ln.(*net.TCPListener); ok {
		_ = tcp.SetDeadline(time.Now().Add(5 * time.Second))
	}
	conn, err := ln.Accept()
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	m0, t0, err := readMemSnap(conn)
	if err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 4096)
	_ = buf
	m1, t1, err := readMemSnap(conn)
	if err != nil {
		t.Fatal(err)
	}
	if m1 < m0 || t1 < t0 {
		t.Fatalf("mallocs %d -> %d total %d -> %d", m0, m1, t0, t1)
	}
}
