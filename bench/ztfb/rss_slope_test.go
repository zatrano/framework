package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// TestRSSSlope records RSS with no connections, then with 64 and 4096
// keep-alive connections held open, and the KB/connection slope from 0 to 4096.
// It is not part of the matrix. Run with ZTFB_RSS=1.
func TestRSSSlope(t *testing.T) {
	if os.Getenv("ZTFB_RSS") != "1" {
		t.Skip("set ZTFB_RSS=1 to measure idle and open-connection RSS")
	}
	exe := filepath.Join(t.TempDir(), "ztfb.exe")
	build := exec.Command("go", "build", "-o", exe, ".")
	build.Env = append(os.Environ(), "GOWORK=off")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	for _, server := range []string{"zatrano", "fiber-tuned", "rawhttp", "fasthttp"} {
		idle, at64, at4096 := rssPoints(t, exe, server)
		slope := float64(at4096-idle) / 4096 / 1024
		fmt.Printf("rss %s idle=%d b64=%d b4096=%d slope_kb=%.3f\n", server, idle, at64, at4096, slope)
	}
}

func rssPoints(t *testing.T, exe, server string) (idle, at64, at4096 uint64) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()

	cmd := exec.Command(exe, "serve", server, addr)
	cmd.Env = rssChildEnv()
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
			_, _ = cmd.Process.Wait()
		}
	}()
	if err := waitReady(addr, 20*time.Second); err != nil {
		t.Fatal(err)
	}
	time.Sleep(200 * time.Millisecond)
	sample, err := readProc(cmd.Process.Pid)
	if err != nil {
		t.Fatal(err)
	}
	idle = sample.RSS
	hold := openKeepAlive(t, addr, 64)
	time.Sleep(200 * time.Millisecond)
	sample, err = readProc(cmd.Process.Pid)
	if err != nil {
		t.Fatal(err)
	}
	at64 = sample.RSS
	closeAll(hold)
	hold = openKeepAlive(t, addr, 4096)
	time.Sleep(300 * time.Millisecond)
	sample, err = readProc(cmd.Process.Pid)
	if err != nil {
		t.Fatal(err)
	}
	at4096 = sample.RSS
	closeAll(hold)
	return idle, at64, at4096
}

func rssChildEnv() []string {
	drop := map[string]bool{
		"ZTFB_PROFILE_SECONDS": true,
		"ZTFB_ALLOC_PROFILE":   true,
		"ZTFB_RSS":             true,
		"APP_ENV":              true,
		"HTTP_POOL_REQUESTS":   true,
	}
	var out []string
	for _, e := range os.Environ() {
		k, _, _ := strings.Cut(e, "=")
		if drop[k] {
			continue
		}
		out = append(out, e)
	}
	return append(out,
		"APP_ENV=production",
		"APP_DEBUG=false",
		"LOG_LEVEL=error",
		"CORS_ENABLED=true",
		"APP_KEY=zatrano-ci-test-key-not-prod!!!!",
		"GOMAXPROCS=1",
		"GOWORK=off",
	)
}

func openKeepAlive(t *testing.T, addr string, n int) []net.Conn {
	t.Helper()
	req := []byte("GET /plaintext HTTP/1.1\r\nHost: localhost\r\nConnection: keep-alive\r\n\r\n")
	out := make([]net.Conn, 0, n)
	for i := 0; i < n; i++ {
		conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
		if err != nil {
			closeAll(out)
			t.Fatalf("dial %d/%d: %v", i, n, err)
		}
		_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
		if _, err := conn.Write(req); err != nil {
			conn.Close()
			closeAll(out)
			t.Fatalf("write %d: %v", i, err)
		}
		if err := readOne(conn); err != nil {
			conn.Close()
			closeAll(out)
			t.Fatalf("read %d: %v", i, err)
		}
		_ = conn.SetDeadline(time.Time{})
		out = append(out, conn)
	}
	return out
}

func readOne(conn net.Conn) error {
	br := bufio.NewReader(conn)
	var length int
	for {
		line, err := br.ReadString('\n')
		if err != nil {
			return err
		}
		if line == "\r\n" {
			break
		}
		if strings.HasPrefix(strings.ToLower(line), "content-length:") {
			n, err := strconv.Atoi(strings.TrimSpace(line[len("content-length:"):]))
			if err != nil {
				return err
			}
			length = n
		}
	}
	if length > 0 {
		_, err := io.CopyN(io.Discard, br, int64(length))
		return err
	}
	return nil
}

func closeAll(conns []net.Conn) {
	for _, c := range conns {
		_ = c.Close()
	}
}
