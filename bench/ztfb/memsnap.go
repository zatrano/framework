package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
	"runtime"
	"time"
)

// startMemSnap answers two runtime.MemStats reads on the TCP address in
// ZTFB_MEMSNAP. The parent asks once at the start of the measured window
// and once at the end. The goroutine stays blocked on that socket; it does
// not run on the request path.
func startMemSnap() {
	addr := os.Getenv("ZTFB_MEMSNAP")
	if addr == "" {
		return
	}
	go func() {
		var conn net.Conn
		var err error
		deadline := time.Now().Add(15 * time.Second)
		for time.Now().Before(deadline) {
			conn, err = net.DialTimeout("tcp", addr, 200*time.Millisecond)
			if err == nil {
				break
			}
			time.Sleep(20 * time.Millisecond)
		}
		if conn == nil {
			fmt.Fprintf(os.Stderr, "memsnap dial: %v\n", err)
			return
		}
		defer conn.Close()
		var cmd [1]byte
		var out [16]byte
		for {
			_ = conn.SetReadDeadline(time.Now().Add(30 * time.Minute))
			if _, err := io.ReadFull(conn, cmd[:]); err != nil {
				return
			}
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			binary.BigEndian.PutUint64(out[0:8], m.Mallocs)
			binary.BigEndian.PutUint64(out[8:16], m.TotalAlloc)
			if _, err := conn.Write(out[:]); err != nil {
				return
			}
		}
	}()
}

// readMemSnap asks the server process for Mallocs and TotalAlloc.
func readMemSnap(conn net.Conn) (mallocs, totalAlloc uint64, err error) {
	if conn == nil {
		return 0, 0, fmt.Errorf("memsnap: nil conn")
	}
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
	if _, err = conn.Write([]byte{1}); err != nil {
		return 0, 0, err
	}
	var buf [16]byte
	if _, err = io.ReadFull(conn, buf[:]); err != nil {
		return 0, 0, err
	}
	return binary.BigEndian.Uint64(buf[0:8]), binary.BigEndian.Uint64(buf[8:16]), nil
}
