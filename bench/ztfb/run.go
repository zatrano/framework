package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

type cell struct {
	Server     string   `json:"server"`
	Route      string   `json:"route"`
	Conns      int      `json:"conns"`
	Round      int      `json:"round"`
	Agg        string   `json:"agg,omitempty"`
	RPS        float64  `json:"rps"`
	P50US      float64  `json:"p50_us"`
	P99US      float64  `json:"p99_us"`
	P999US     float64  `json:"p999_us"`
	CPUUS      float64  `json:"cpu_us_per_req"`
	RSSBytes   uint64   `json:"rss_bytes"`
	HWMBytes   uint64   `json:"hwm_bytes"`
	Errors     int      `json:"errors"`
	Requests   int      `json:"requests"`
	ElapsedS   float64  `json:"elapsed_s"`
	RSSSamples []uint64 `json:"rss_samples,omitempty"`
}

func cmdRun(args []string) error {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	out := fs.String("o", "", "jsonl output path")
	affinity := fs.String("affinity", "split", "split pins the server to CPU 0; shared uses every CPU")
	roundsFlag := fs.Int("rounds", 0, "rounds per cell; 0 uses 4, or 3 at 4096 connections")
	connsFlag := fs.String("conns", "64,256,1024,4096", "connection levels")
	routesFlag := fs.String("routes", "", "comma-separated route names; empty runs all four")
	warmup := fs.Duration("warmup", 1500*time.Millisecond, "warmup before the measured window")
	measure := fs.Duration("measure", 5*time.Second, "measured window")
	addr := fs.String("addr", "127.0.0.1:18181", "server listen address")
	if err := fs.Parse(args); err != nil {
		return err
	}
	levels, err := parseInts(*connsFlag)
	if err != nil {
		return err
	}
	selected := routes()
	if *routesFlag != "" {
		selected = nil
		want := map[string]bool{}
		for _, name := range strings.Split(*routesFlag, ",") {
			want[strings.TrimSpace(name)] = true
		}
		for _, rt := range routes() {
			if want[rt.Name] {
				selected = append(selected, rt)
			}
		}
		if len(selected) == 0 {
			return fmt.Errorf("no routes matched %q", *routesFlag)
		}
	}
	return runMatrix(*out, *affinity, *roundsFlag, levels, selected, *warmup, *measure, *addr, false)
}

func cmdLoad(args []string) error {
	if len(args) != 4 {
		return fmt.Errorf("load <addr> <route> <conns> <seconds>")
	}
	var rt routeSpec
	found := false
	for _, candidate := range routes() {
		if candidate.Name == args[1] {
			rt = candidate
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("unknown route %q", args[1])
	}
	conns, err := strconv.Atoi(args[2])
	if err != nil || conns < 1 {
		return fmt.Errorf("bad conns")
	}
	secs, err := strconv.Atoi(args[3])
	if err != nil || secs < 1 {
		return fmt.Errorf("bad seconds")
	}
	host, _, err := net.SplitHostPort(args[0])
	if err != nil {
		return err
	}
	warm, measured := session(args[0], requestBytes(rt, host), rt.Want, conns, time.Second, time.Duration(secs)*time.Second, nil)
	fmt.Printf("warmup requests=%d errors=%d measure rps=%.0f errors=%d p50=%s p99=%s\n", warm.Requests, warm.Errors, measured.RPS(), measured.Errors, percentile(measured.Samples, 0.50), percentile(append([]int64(nil), measured.Samples...), 0.99))
	return nil
}

func cmdSoak(args []string) error {
	fs := flag.NewFlagSet("soak", flag.ContinueOnError)
	out := fs.String("o", "", "jsonl output path")
	affinity := fs.String("affinity", "split", "split pins the server to CPU 0; shared uses every CPU")
	addr := fs.String("addr", "127.0.0.1:18181", "server listen address")
	if err := fs.Parse(args); err != nil {
		return err
	}
	return runMatrix(*out, *affinity, 1, []int{512}, routes(), 1500*time.Millisecond, 25*time.Second, *addr, true)
}

func parseInts(s string) ([]int, error) {
	var out []int
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		n, err := strconv.Atoi(p)
		if err != nil || n < 1 {
			return nil, fmt.Errorf("bad connection level %q", p)
		}
		out = append(out, n)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no connection levels")
	}
	return out, nil
}

func runMatrix(outPath, affinity string, roundsFlag int, levels []int, selected []routeSpec, warmup, measure time.Duration, addr string, soak bool) error {
	var w *bufio.Writer
	var f *os.File
	if outPath != "" {
		var err error
		f, err = os.Create(outPath)
		if err != nil {
			return err
		}
		defer f.Close()
		w = bufio.NewWriter(f)
		defer w.Flush()
	}
	write := func(c cell) {
		line, _ := json.Marshal(c)
		fmt.Println(string(line))
		if w != nil {
			w.Write(line)
			w.WriteByte('\n')
			w.Flush()
		}
	}

	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return err
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	serverMask, clientMask := affinityMasks(affinity)
	serverProcs := "1"
	if affinity != "split" {
		serverProcs = strconv.Itoa(runtime.NumCPU())
	} else {
		_ = setAffinity(os.Getpid(), clientMask)
		runtime.GOMAXPROCS(runtime.NumCPU() - 1)
	}

	for _, rt := range selected {
		req := requestBytes(rt, host)
		for _, conns := range levels {
			rounds := roundsFlag
			if rounds <= 0 {
				rounds = 4
				if conns >= 4096 {
					rounds = 3
				}
			}
			byServer := map[string][]cell{}
			for round := 1; round <= rounds; round++ {
				order := []string{"zatrano", "fiber"}
				if round%2 == 0 {
					order = []string{"fiber", "zatrano"}
				}
				for _, server := range order {
					cell, err := oneRound(exe, server, addr, rt, req, conns, round, warmup, measure, serverMask, affinity == "split", serverProcs, soak)
					if err != nil {
						return fmt.Errorf("%s %s c=%d round %d: %w", server, rt.Name, conns, round, err)
					}
					write(cell)
					byServer[server] = append(byServer[server], cell)
				}
			}
			for _, server := range []string{"zatrano", "fiber"} {
				write(medianCell(server, rt.Name, conns, byServer[server]))
			}
			if soak {
				if err := checkSoakFlat(byServer); err != nil {
					fmt.Fprintf(os.Stderr, "soak: %v\n", err)
				}
			}
		}
	}
	return nil
}

func oneRound(exe, server, addr string, rt routeSpec, req []byte, conns, round int, warmup, measure time.Duration, serverMask uintptr, pin bool, serverProcs string, soak bool) (cell, error) {
	cmd := exec.Command(exe, "serve", server, addr)
	cmd.Env = serverEnv(serverProcs)
	cmd.Stdout = os.Stderr
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return cell{}, err
	}
	if err := cmd.Start(); err != nil {
		return cell{}, err
	}
	done := make(chan struct{})
	go func() {
		sc := bufio.NewScanner(stderr)
		for sc.Scan() {
			fmt.Fprintln(os.Stderr, sc.Text())
		}
		close(done)
	}()
	defer func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
			_, _ = cmd.Process.Wait()
		}
		<-done
	}()
	if pin {
		_ = setAffinity(cmd.Process.Pid, serverMask)
	}
	if err := waitReady(addr, 20*time.Second); err != nil {
		return cell{}, err
	}
	var before procSample
	var beforeErr error
	var rssMu sync.Mutex
	var rssSamples []uint64
	rssStop := make(chan struct{})
	warm, measured := session(addr, req, rt.Want, conns, warmup, measure, func() {
		before, beforeErr = readProc(cmd.Process.Pid)
		if !soak || cmd.Process == nil {
			return
		}
		if s, err := readProc(cmd.Process.Pid); err == nil {
			rssSamples = append(rssSamples, s.RSS)
		}
		go func() {
			t := time.NewTicker(time.Second)
			defer t.Stop()
			for {
				select {
				case <-rssStop:
					return
				case <-t.C:
					s, err := readProc(cmd.Process.Pid)
					if err != nil {
						continue
					}
					rssMu.Lock()
					rssSamples = append(rssSamples, s.RSS)
					rssMu.Unlock()
				}
			}
		}()
	})
	var copied []uint64
	if soak {
		close(rssStop)
		rssMu.Lock()
		if cmd.Process != nil {
			if s, err := readProc(cmd.Process.Pid); err == nil {
				rssSamples = append(rssSamples, s.RSS)
			}
		}
		copied = append([]uint64(nil), rssSamples...)
		rssMu.Unlock()
	}
	if warm.Requests == 0 {
		return cell{}, fmt.Errorf("warmup produced no successful responses (errors %d)", warm.Errors)
	}
	if beforeErr != nil {
		return cell{}, beforeErr
	}
	after, err := readProc(cmd.Process.Pid)
	if err != nil {
		return cell{}, err
	}
	p50 := percentile(measured.Samples, 0.50)
	p99 := percentile(append([]int64(nil), measured.Samples...), 0.99)
	// percentile sorts in place; copy for p999 from the already sorted slice.
	p999 := percentile(measured.Samples, 0.999)
	return cell{
		Server:     server,
		Route:      rt.Name,
		Conns:      conns,
		Round:      round,
		RPS:        measured.RPS(),
		P50US:      float64(p50.Microseconds()),
		P99US:      float64(p99.Microseconds()),
		P999US:     float64(p999.Microseconds()),
		CPUUS:      cpuMicrosPerReq(before, after, measured.Requests),
		RSSBytes:   after.RSS,
		HWMBytes:   after.HWM,
		Errors:     measured.Errors,
		Requests:   measured.Requests,
		ElapsedS:   measured.Elapsed.Seconds(),
		RSSSamples: copied,
	}, nil
}

func serverEnv(serverProcs string) []string {
	env := os.Environ()
	set := map[string]string{
		"APP_ENV":      "production",
		"APP_DEBUG":    "false",
		"LOG_LEVEL":    "error",
		"CORS_ENABLED": "true",
		"APP_KEY":      "zatrano-ci-test-key-not-prod!!!!",
		"GOTOOLCHAIN":  "local",
		"GOMAXPROCS":   serverProcs,
		"GOWORK":       "off",
		"GOFLAGS":      "",
	}
	var out []string
	seen := map[string]bool{}
	for _, kv := range env {
		key, _, _ := strings.Cut(kv, "=")
		if v, ok := set[key]; ok {
			out = append(out, key+"="+v)
			seen[key] = true
			continue
		}
		out = append(out, kv)
	}
	for k, v := range set {
		if !seen[k] {
			out = append(out, k+"="+v)
		}
	}
	return out
}

func waitReady(addr string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	var last error
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 200*time.Millisecond)
		if err == nil {
			conn.Close()
			return nil
		}
		last = err
		time.Sleep(50 * time.Millisecond)
	}
	return fmt.Errorf("server did not accept connections: %w", last)
}

func medianCell(server, route string, conns int, rows []cell) cell {
	var rps []float64
	var p50, p99, p999, cpu []float64
	var rss, hwm []float64
	var errs, reqs int
	for _, row := range rows {
		rps = append(rps, row.RPS)
		p50 = append(p50, row.P50US)
		p99 = append(p99, row.P99US)
		p999 = append(p999, row.P999US)
		cpu = append(cpu, row.CPUUS)
		rss = append(rss, float64(row.RSSBytes))
		hwm = append(hwm, float64(row.HWMBytes))
		errs += row.Errors
		reqs += row.Requests
	}
	return cell{
		Server:   server,
		Route:    route,
		Conns:    conns,
		Agg:      "median",
		RPS:      medianFloat(rps),
		P50US:    medianFloat(p50),
		P99US:    medianFloat(p99),
		P999US:   medianFloat(p999),
		CPUUS:    medianFloat(cpu),
		RSSBytes: uint64(medianFloat(rss)),
		HWMBytes: uint64(medianFloat(hwm)),
		Errors:   errs,
		Requests: reqs,
	}
}

func checkSoakFlat(byServer map[string][]cell) error {
	for server, rows := range byServer {
		for _, row := range rows {
			if row.Errors != 0 {
				return fmt.Errorf("%s %s errors %d", server, row.Route, row.Errors)
			}
			samples := row.RSSSamples
			if len(samples) < 8 {
				return fmt.Errorf("%s %s recorded %d RSS samples", server, row.Route, len(samples))
			}
			// The first samples include the connection ramp. Compare a
			// sample five seconds in with the sample at the end.
			first, last := samples[5], samples[len(samples)-1]
			if last > first+first/20 && last > first+(1<<20) {
				return fmt.Errorf("%s %s RSS grew %d -> %d", server, row.Route, first, last)
			}
		}
	}
	return nil
}

func affinityMasks(mode string) (server, client uintptr) {
	n := runtime.NumCPU()
	if n > 64 {
		n = 64
	}
	var all uintptr
	for i := 0; i < n; i++ {
		all |= 1 << uintptr(i)
	}
	if mode != "split" || n < 2 {
		return all, all
	}
	return 1, all ^ 1
}
