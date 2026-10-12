package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"math"
	"net"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

type loadResult struct {
	Requests int
	Errors   int
	Elapsed  time.Duration
	Samples  []int64 // nanoseconds
}

func (r loadResult) RPS() float64 {
	if r.Elapsed <= 0 {
		return 0
	}
	return float64(r.Requests) / r.Elapsed.Seconds()
}

type routeSpec struct {
	Name   string
	Method string
	Path   string
	Body   string
	Want   []byte
}

func routes() []routeSpec {
	jsonBody := []byte(`{"message":"Hello, World!"}`)
	userBody := []byte(`{"id":"42"}`)
	return []routeSpec{
		{Name: "GET /plaintext", Method: "GET", Path: "/plaintext", Want: []byte(helloBody)},
		{Name: "GET /json", Method: "GET", Path: "/json", Want: jsonBody},
		{Name: "GET /users/{id}", Method: "GET", Path: "/users/42", Want: userBody},
		{Name: "POST /echo", Method: "POST", Path: "/echo", Body: echoBody, Want: jsonBody},
	}
}

func requestBytes(rt routeSpec, host string) []byte {
	var b bytes.Buffer
	fmt.Fprintf(&b, "%s %s HTTP/1.1\r\nHost: %s\r\nConnection: keep-alive\r\n", rt.Method, rt.Path, host)
	if rt.Body != "" {
		fmt.Fprintf(&b, "Content-Type: application/json\r\nContent-Length: %d\r\n", len(rt.Body))
		b.WriteString("\r\n")
		b.WriteString(rt.Body)
		return b.Bytes()
	}
	b.WriteString("\r\n")
	return b.Bytes()
}

// session keeps one connection per worker. Warmup and the measured window
// share that connection so the concurrency level is reached before timing starts.
func session(addr string, req, want []byte, workers int, warmup, measure time.Duration, atMeasure func()) (loadResult, loadResult) {
	var warmReqs, warmErrs, measReqs, measErrs atomic.Int64
	var mu sync.Mutex
	var samples []int64
	var gate sync.Once
	var begin time.Time
	mark := func() time.Time {
		gate.Do(func() {
			begin = time.Now()
			if atMeasure != nil {
				atMeasure()
			}
		})
		return begin
	}
	slots := make(chan struct{}, 128)
	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			local := sessionConn(addr, req, want, warmup, measure, slots, mark, &warmReqs, &warmErrs, &measReqs, &measErrs)
			if len(local) > 0 {
				mu.Lock()
				samples = append(samples, local...)
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	elapsed := time.Duration(0)
	if !begin.IsZero() {
		elapsed = time.Since(begin)
	}
	return loadResult{Requests: int(warmReqs.Load()), Errors: int(warmErrs.Load())},
		loadResult{
			Requests: int(measReqs.Load()),
			Errors:   int(measErrs.Load()),
			Elapsed:  elapsed,
			Samples:  samples,
		}
}

func sessionConn(addr string, req, want []byte, warmup, measure time.Duration, slots chan struct{}, mark func() time.Time, warmReqs, warmErrs, measReqs, measErrs *atomic.Int64) []int64 {
	warmDeadline := time.Now().Add(warmup)
	conn, br := dialRetry(addr, warmDeadline, slots)
	if conn == nil {
		warmErrs.Add(1)
		return nil
	}
	defer func() { _ = conn.Close() }()
	for time.Now().Before(warmDeadline) {
		if !exchange(conn, br, req, want, warmDeadline, warmReqs, warmErrs, nil) {
			_ = conn.Close()
			conn, br = dialRetry(addr, warmDeadline, slots)
			if conn == nil {
				return nil
			}
		}
	}
	deadline := mark().Add(measure)
	var samples []int64
	for time.Now().Before(deadline) {
		if !exchange(conn, br, req, want, deadline, measReqs, measErrs, &samples) {
			if time.Now().After(deadline) {
				return samples
			}
			_ = conn.Close()
			conn, br = dialRetry(addr, deadline, slots)
			if conn == nil {
				return samples
			}
		}
	}
	return samples
}

func exchange(conn net.Conn, br *bufio.Reader, req, want []byte, deadline time.Time, reqs, errs *atomic.Int64, samples *[]int64) bool {
	_ = conn.SetDeadline(time.Now().Add(30 * time.Second))
	t0 := time.Now()
	if _, err := conn.Write(req); err != nil {
		if time.Now().Before(deadline) {
			errs.Add(1)
		}
		return false
	}
	if err := readChecked(br, want); err != nil {
		if time.Now().Before(deadline) {
			errs.Add(1)
		}
		return false
	}
	reqs.Add(1)
	if samples != nil {
		*samples = append(*samples, time.Since(t0).Nanoseconds())
	}
	return true
}

func dial(addr string) (net.Conn, *bufio.Reader) {
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		return nil, nil
	}
	return conn, bufio.NewReaderSize(conn, 2048)
}

func dialRetry(addr string, deadline time.Time, slots chan struct{}) (net.Conn, *bufio.Reader) {
	for time.Now().Before(deadline) {
		select {
		case slots <- struct{}{}:
		case <-time.After(time.Until(deadline)):
			return nil, nil
		}
		conn, br := dial(addr)
		<-slots
		if conn != nil {
			return conn, br
		}
		time.Sleep(5 * time.Millisecond)
	}
	return nil, nil
}

func readChecked(br *bufio.Reader, want []byte) error {
	status, err := br.ReadSlice('\n')
	if err != nil {
		return err
	}
	if !bytes.Contains(status, []byte(" 200 ")) {
		return fmt.Errorf("status %q", trimNL(status))
	}
	cl := -1
	for {
		line, err := br.ReadSlice('\n')
		if err != nil {
			return err
		}
		if len(line) == 2 && line[0] == '\r' {
			break
		}
		if n, ok := contentLength(line); ok {
			if cl != -1 {
				return fmt.Errorf("duplicate content-length")
			}
			cl = n
		}
	}
	if cl != len(want) {
		return fmt.Errorf("content-length %d want %d", cl, len(want))
	}
	buf := bodyBuf(cl)
	defer releaseBody(buf)
	if _, err := io.ReadFull(br, buf); err != nil {
		return err
	}
	if !bytes.Equal(buf, want) {
		return fmt.Errorf("body %q", buf)
	}
	return nil
}

var bodyPool = sync.Pool{New: func() any { return make([]byte, 64) }}

func bodyBuf(n int) []byte {
	b := bodyPool.Get().([]byte)
	if cap(b) < n {
		b = make([]byte, n)
	}
	return b[:n]
}

func releaseBody(b []byte) {
	bodyPool.Put(b[:0])
}

func contentLength(line []byte) (int, bool) {
	// "Content-Length: N\r\n"
	const key = "content-length:"
	if len(line) < len(key)+1 {
		return 0, false
	}
	for i := 0; i < len(key); i++ {
		c := line[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		if c != key[i] {
			return 0, false
		}
	}
	rest := line[len(key):]
	for len(rest) > 0 && (rest[0] == ' ' || rest[0] == '\t') {
		rest = rest[1:]
	}
	rest = bytes.TrimRight(rest, "\r\n")
	n, err := strconv.Atoi(string(rest))
	if err != nil || n < 0 {
		return 0, false
	}
	return n, true
}

func trimNL(b []byte) []byte {
	return bytes.TrimRight(b, "\r\n")
}

func percentile(samples []int64, p float64) time.Duration {
	if len(samples) == 0 {
		return 0
	}
	sort.Slice(samples, func(i, j int) bool { return samples[i] < samples[j] })
	idx := int(math.Ceil(p*float64(len(samples)))) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(samples) {
		idx = len(samples) - 1
	}
	return time.Duration(samples[idx])
}

func medianFloat(v []float64) float64 {
	if len(v) == 0 {
		return 0
	}
	cp := append([]float64(nil), v...)
	sort.Float64s(cp)
	n := len(cp)
	if n%2 == 1 {
		return cp[n/2]
	}
	return (cp[n/2-1] + cp[n/2]) / 2
}

func medianDur(v []time.Duration) time.Duration {
	if len(v) == 0 {
		return 0
	}
	ns := make([]int64, len(v))
	for i, d := range v {
		ns[i] = int64(d)
	}
	sort.Slice(ns, func(i, j int) bool { return ns[i] < ns[j] })
	n := len(ns)
	if n%2 == 1 {
		return time.Duration(ns[n/2])
	}
	return time.Duration((ns[n/2-1] + ns[n/2]) / 2)
}
