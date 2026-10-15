package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

// TestRSSSlope records, for each server, RSS with no connections, the
// high-water mark while 64 and 4096 connections are held, and RSS five
// seconds after those connections close. ZATRANO and Fiber (tuned) also
// report the MemStats delta from 0 to 64 connections. It is not part of
// the matrix. Run with ZTFB_RSS=1.
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
	servers := []string{"zatrano", "fiber-tuned", "rawhttp", "fasthttp"}
	if v := os.Getenv("ZTFB_RSS_SERVERS"); v != "" {
		servers = strings.Split(v, ",")
	}
	for _, server := range servers {
		row := rssPoints(t, exe, server)
		fmt.Printf("rss %s idle=%d open64=%d hwm64=%d priv64=%d rss64_after5s=%d priv64_after=%d open4096=%d hwm4096=%d rss4096_after5s=%d\n",
			server, row.idle, row.rssOpen64, row.hwm64, row.privOpen64, row.rss64, row.priv64, row.rssOpen4096, row.hwm4096, row.rss4096)
		fmt.Printf("ws %s open64 private=%d shareable=%d mapped=%d after5s private=%d shareable=%d mapped=%d\n",
			server, row.wsOpen.Private, row.wsOpen.Shareable, row.wsOpen.Mapped, row.wsAfter.Private, row.wsAfter.Shareable, row.wsAfter.Mapped)
		if row.memOK {
			d := row.mem64
			z := row.mem0
			fmt.Printf("mem %s heap=%d stack=%d gcsys=%d othersys=%d\n",
				server,
				int64(d.HeapInuse-z.HeapInuse),
				int64(d.StackInuse-z.StackInuse),
				int64(d.GCSys-z.GCSys),
				int64(d.OtherSys-z.OtherSys))
		}
	}
}

type rssRow struct {
	idle, hwm64, rssOpen64, privOpen64, rss64, priv64, hwm4096, rssOpen4096, rss4096 uint64
	mem0, mem64                                                                      memClasses
	memOK                                                                            bool
	wsOpen, wsAfter                                                                  wsClass
}

func rssPoints(t *testing.T, exe, server string) rssRow {
	t.Helper()
	snapLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer snapLn.Close()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()

	cmd := exec.Command(exe, "serve", server, addr)
	cmd.Env = append(rssChildEnv(), "ZTFB_MEMSNAP="+snapLn.Addr().String())
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
	if tcp, ok := snapLn.(*net.TCPListener); ok {
		_ = tcp.SetDeadline(time.Now().Add(15 * time.Second))
	}
	snapConn, err := snapLn.Accept()
	if err != nil {
		t.Fatal(err)
	}
	defer snapConn.Close()
	if err := waitReady(addr, 20*time.Second); err != nil {
		t.Fatal(err)
	}
	time.Sleep(200 * time.Millisecond)
	var row rssRow
	sample, err := readProc(cmd.Process.Pid)
	if err != nil {
		t.Fatal(err)
	}
	row.idle = sample.RSS
	if classes, err := readMemClasses(snapConn); err == nil {
		row.mem0 = classes
		row.memOK = true
	}

	hold := openKeepAlive(t, addr, 64)
	time.Sleep(200 * time.Millisecond)
	sample, err = readProc(cmd.Process.Pid)
	if err != nil {
		t.Fatal(err)
	}
	row.hwm64 = sample.HWM
	row.rssOpen64 = sample.RSS
	row.privOpen64 = sample.Private
	if ws, err := readWorkingSetClass(cmd.Process.Pid); err != nil {
		if runtime.GOOS == "windows" {
			t.Fatal(err)
		}
	} else {
		row.wsOpen = ws
	}
	if row.memOK {
		if classes, err := readMemClasses(snapConn); err == nil {
			row.mem64 = classes
		}
	}
	closeAll(hold)
	time.Sleep(5 * time.Second)
	sample, err = readProc(cmd.Process.Pid)
	if err != nil {
		t.Fatal(err)
	}
	row.rss64 = sample.RSS
	row.priv64 = sample.Private
	if ws, err := readWorkingSetClass(cmd.Process.Pid); err != nil {
		if runtime.GOOS == "windows" {
			t.Fatal(err)
		}
	} else {
		row.wsAfter = ws
	}
	if os.Getenv("ZTFB_RSS_SKIP4096") == "1" {
		return row
	}

	hold = openKeepAlive(t, addr, 4096)
	time.Sleep(300 * time.Millisecond)
	sample, err = readProc(cmd.Process.Pid)
	if err != nil {
		t.Fatal(err)
	}
	row.hwm4096 = sample.HWM
	row.rssOpen4096 = sample.RSS
	closeAll(hold)
	time.Sleep(5 * time.Second)
	sample, err = readProc(cmd.Process.Pid)
	if err != nil {
		t.Fatal(err)
	}
	row.rss4096 = sample.RSS
	return row
}

func rssChildEnv() []string {
	drop := map[string]bool{
		"ZTFB_PROFILE_SECONDS": true,
		"ZTFB_ALLOC_PROFILE":   true,
		"ZTFB_RSS":             true,
		"ZTFB_MEMSNAP":         true,
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
