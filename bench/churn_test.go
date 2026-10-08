package bench

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/zatrano/rawhttp"
)

func BenchmarkConnectionChurn(b *testing.B) {
	for _, size := range []int{1 << 20, 64 << 10, 16 << 10, 8 << 10} {
		b.Run(fmt.Sprintf("%d", size), func(b *testing.B) {
			ns, err := churnNanos(size, 64, b.N)
			if err != nil {
				b.Fatal(err)
			}
			if ns > 0 {
				b.ReportMetric(1e9/ns, "rps")
			}
		})
	}
}

// TestConnectionChurnRotating measures a new TCP connection per request.
// It skips unless ZATRANO_CHURN=1. Linux CI sets that. Windows numbers are
// informational.
func TestConnectionChurnRotating(t *testing.T) {
	if os.Getenv("ZATRANO_CHURN") == "" {
		t.Skip("set ZATRANO_CHURN=1")
	}
	sizes := []int{1 << 20, 64 << 10, 16 << 10, 8 << 10}
	const rounds = 10
	const requests = 256
	samples := map[int][]float64{}
	for round := 0; round < rounds; round++ {
		for _, size := range sizes {
			ns, err := churnNanos(size, 64, requests)
			if err != nil {
				t.Fatal(err)
			}
			samples[size] = append(samples[size], ns)
		}
	}
	t.Logf("connection churn, rotating n=%d, 64 clients, %d requests/sample", rounds, requests)
	for _, size := range sizes {
		ns := append([]float64(nil), samples[size]...)
		sort.Float64s(ns)
		med := (ns[len(ns)/2-1] + ns[len(ns)/2]) / 2
		t.Logf("MaxHeaderBytes %d  median %.0f ns/req  min %.0f  max %.0f  rps %.0f", size, med, ns[0], ns[len(ns)-1], 1e9/med)
	}
}

func churnNanos(headerBytes, workers, requests int) (float64, error) {
	if requests < workers {
		requests = workers
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	srv := serverRunHead(func(ctx *rawhttp.Ctx) {
		ctx.SetStatusCode(200)
		ctx.SetBodyString("ok")
	})
	srv.MaxHeaderBytes = headerBytes
	go func() { _ = srv.Serve(ln) }()
	defer func() { _ = srv.Close() }()

	addr := ln.Addr().String()
	per := requests / workers
	var wg sync.WaitGroup
	errCh := make(chan error, workers)
	start := time.Now()
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < per; i++ {
				if err := oneCloseRequest(addr); err != nil {
					errCh <- err
					return
				}
			}
		}()
	}
	wg.Wait()
	elapsed := time.Since(start)
	close(errCh)
	for err := range errCh {
		if err != nil {
			return 0, err
		}
	}
	total := workers * per
	return float64(elapsed.Nanoseconds()) / float64(total), nil
}

func oneCloseRequest(addr string) error {
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(2 * time.Second))
	if _, err := io.WriteString(conn, "GET / HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\n\r\n"); err != nil {
		return err
	}
	br := bufio.NewReader(conn)
	line, err := br.ReadString('\n')
	if err != nil {
		return err
	}
	var status int
	if _, err := fmt.Sscanf(line, "HTTP/1.1 %d", &status); err != nil {
		return err
	}
	if status != 200 {
		return fmt.Errorf("status %d", status)
	}
	_, _ = io.Copy(io.Discard, br)
	return nil
}
