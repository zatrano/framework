package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

type cell struct {
	Server            string   `json:"server"`
	Route             string   `json:"route"`
	Conns             int      `json:"conns"`
	Round             int      `json:"round"`
	Agg               string   `json:"agg,omitempty"`
	RPS               float64  `json:"rps"`
	P50US             float64  `json:"p50_us"`
	P99US             float64  `json:"p99_us"`
	P999US            float64  `json:"p999_us"`
	CPUUS             float64  `json:"cpu_us_per_req"`
	RSSBytes          uint64   `json:"rss_bytes"`
	HWMBytes          uint64   `json:"hwm_bytes"`
	Errors            int      `json:"errors"`
	Requests          int      `json:"requests"`
	ElapsedS          float64  `json:"elapsed_s"`
	GCCycles          int      `json:"gc_cycles,omitempty"`
	RSSSamples        []uint64 `json:"rss_samples,omitempty"`
	Config            string   `json:"config,omitempty"`
	Client            string   `json:"client,omitempty"`
	Codec             string   `json:"codec,omitempty"`
	Ratio             float64  `json:"ratio,omitempty"`
	CILow             float64  `json:"ci_low,omitempty"`
	CIHigh            float64  `json:"ci_high,omitempty"`
	P50Ratio          float64  `json:"p50_ratio,omitempty"`
	P99Ratio          float64  `json:"p99_ratio,omitempty"`
	CPURatio          float64  `json:"cpu_ratio,omitempty"`
	RSSRatio          float64  `json:"rss_ratio,omitempty"`
	Verdict           string   `json:"verdict,omitempty"`
	AllocsPerReq      *float64 `json:"allocs_per_req,omitempty"`
	BytesPerReq       *float64 `json:"bytes_per_req,omitempty"`
	AllocsPerReqSetup *float64 `json:"allocs_per_req_with_setup,omitempty"`
	BytesPerReqSetup  *float64 `json:"bytes_per_req_with_setup,omitempty"`
}

type matrixOpts struct {
	outPath  string
	affinity string
	rounds   int
	levels   []int
	routes   []routeSpec
	warmup   time.Duration
	measure  time.Duration
	addr     string
	soak     bool
	client   string
	config   string
	codec    string
	pair     string
	gcMin    int
}

func cmdRun(args []string) error {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	out := fs.String("o", "", "jsonl output path")
	affinity := fs.String("affinity", "split", "split pins the server to CPU 0; shared uses every CPU")
	roundsFlag := fs.Int("rounds", 0, "rounds per cell; 0 uses 10")
	connsFlag := fs.String("conns", "64,256,1024,4096", "connection levels")
	routesFlag := fs.String("routes", "", "comma-separated route names; empty runs all four")
	warmup := fs.Duration("warmup", 1500*time.Millisecond, "warmup before the measured window")
	measure := fs.Duration("measure", 5*time.Second, "measured window")
	addr := fs.String("addr", "127.0.0.1:18181", "server listen address")
	client := fs.String("client", "go", "load generator: go or wrk")
	config := fs.String("config", "default", "label recorded on every row; pooled sets HTTP_POOL_REQUESTS=true")
	codec := fs.String("codec", "encodingjson", "encodingjson or goccy; both servers use it")
	pair := fs.String("pair", "fiber", "fiber compares zatrano with fiber and fiber-tuned; rawhttp compares rawhttp with fasthttp")
	gcMin := fs.Int("gc-min", 0, "before scoring, set one window long enough for this many GC cycles on every server in the pair; 0 keeps -measure")
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
	if *client != "go" && *client != "wrk" {
		return fmt.Errorf("client must be go or wrk")
	}
	if *codec != "encodingjson" && *codec != "goccy" {
		return fmt.Errorf("codec must be encodingjson or goccy")
	}
	if *pair != "fiber" && *pair != "rawhttp" {
		return fmt.Errorf("pair must be fiber or rawhttp")
	}
	return runMatrix(matrixOpts{
		outPath: *out, affinity: *affinity, rounds: *roundsFlag, levels: levels, routes: selected,
		warmup: *warmup, measure: *measure, addr: *addr, client: *client, config: *config, codec: *codec,
		pair: *pair, gcMin: *gcMin,
	})
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
	warm, measured := session(args[0], requestBytes(rt, host), rt.Want, conns, time.Second, time.Duration(secs)*time.Second, nil, nil)
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
	return runMatrix(matrixOpts{
		outPath: *out, affinity: *affinity, rounds: 1, levels: []int{512}, routes: routes(),
		warmup: 1500 * time.Millisecond, measure: 25 * time.Second, addr: *addr, soak: true,
		client: "go", config: "default", codec: "encodingjson",
	})
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

func runMatrix(opt matrixOpts) error {
	var w *bufio.Writer
	var f *os.File
	if opt.outPath != "" {
		var err error
		f, err = os.Create(opt.outPath)
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

	host, _, err := net.SplitHostPort(opt.addr)
	if err != nil {
		return err
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	serverMask, clientMask := affinityMasks(opt.affinity)
	// GOMAXPROCS matches the cores the server is pinned to, for both servers.
	serverProcs := strconv.Itoa(serverProcCount(serverMask))
	if opt.affinity == "split" {
		_ = setAffinity(os.Getpid(), clientMask)
		clientN := runtime.NumCPU() - serverProcCount(serverMask)
		if clientN < 1 {
			clientN = 1
		}
		runtime.GOMAXPROCS(clientN)
	}
	if opt.client == "" {
		opt.client = "go"
	}
	if opt.config == "" {
		opt.config = "default"
	}
	if opt.gcMin > 0 {
		if opt.client != "go" {
			return fmt.Errorf("gc-min requires the go client")
		}
		_ = os.Setenv("ZTFB_GCTRACE", "1")
	}
	if !opt.soak && opt.rounds > 0 && opt.rounds < defaultRounds {
		fmt.Fprintf(os.Stderr, "warning: %d rounds is below the %d-round cell size used for the CI\n", opt.rounds, defaultRounds)
	}

	for _, rt := range opt.routes {
		req := requestBytes(rt, host)
		for _, conns := range opt.levels {
			rounds := opt.rounds
			if rounds <= 0 {
				rounds = defaultRounds
			}
			order := pairOrder(opt.pair)
			left := order[0]
			measure := opt.measure
			if opt.gcMin > 0 {
				var err error
				measure, err = calibrateWindow(exe, opt.addr, rt, req, conns, opt.warmup, opt.measure, serverMask, opt.affinity == "split", serverProcs, opt.client, opt.config, opt.codec, order, opt.gcMin)
				if err != nil {
					return err
				}
				fmt.Fprintf(os.Stderr, "gc window %s for %s c=%d (every server, %d cycles)\n", measure, rt.Name, conns, opt.gcMin)
			}
			byServer := map[string][]cell{}
			for round := 1; round <= rounds; round++ {
				for _, server := range rotate(order, round) {
					row, err := oneRound(exe, server, opt.addr, rt, req, conns, round, opt.warmup, measure, serverMask, opt.affinity == "split", serverProcs, opt.soak, opt.client, opt.config, opt.codec, 0)
					if err != nil {
						return fmt.Errorf("%s %s c=%d round %d: %w", server, rt.Name, conns, round, err)
					}
					write(row)
					byServer[server] = append(byServer[server], row)
				}
			}
			medians := map[string]cell{}
			for _, server := range order {
				med := medianCell(server, rt.Name, conns, byServer[server])
				med.Config = opt.config
				med.Client = opt.client
				med.Codec = opt.codec
				write(med)
				medians[server] = med
			}
			if !opt.soak {
				// The last name is the binding opponent. For the fiber pair
				// that is fiber-tuned. fiber stays a second paired row.
				binding := order[len(order)-1]
				paired, err := pairedCell(left+"/"+binding, rt.Name, conns, opt.config, opt.client, byServer[left], byServer[binding], medians[left], medians[binding])
				if err != nil {
					return err
				}
				write(paired)
				for _, extra := range order[1 : len(order)-1] {
					paired, err = pairedCell(left+"/"+extra, rt.Name, conns, opt.config, opt.client, byServer[left], byServer[extra], medians[left], medians[extra])
					if err != nil {
						return err
					}
					write(paired)
				}
			}
			if opt.soak {
				if err := checkSoakFlat(byServer); err != nil {
					fmt.Fprintf(os.Stderr, "soak: %v\n", err)
				}
			}
		}
	}
	return nil
}

func oneRound(exe, server, addr string, rt routeSpec, req []byte, conns, round int, warmup, measure time.Duration, serverMask uintptr, pin bool, serverProcs string, soak bool, client, config, codec string, gcStop int) (cell, error) {
	cmd := startServer(exe, server, addr, serverMask)
	cmd.Env = serverEnv(serverProcs, config, codec)
	snapLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return cell{}, err
	}
	defer snapLn.Close()
	cmd.Env = append(cmd.Env, "ZTFB_MEMSNAP="+snapLn.Addr().String())
	cmd.Stdout = os.Stderr
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return cell{}, err
	}
	procStart := time.Now()
	if err := cmd.Start(); err != nil {
		return cell{}, err
	}
	var gcMu sync.Mutex
	var marks []gcMark
	done := make(chan struct{})
	go func() {
		sc := bufio.NewScanner(stderr)
		for sc.Scan() {
			line := sc.Text()
			fmt.Fprintln(os.Stderr, line)
			if at, forced, ok := parseGCLine(line); ok {
				gcMu.Lock()
				marks = append(marks, gcMark{at: at, forced: forced})
				gcMu.Unlock()
			}
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
	if tcp, ok := snapLn.(*net.TCPListener); ok {
		_ = tcp.SetDeadline(time.Now().Add(15 * time.Second))
	}
	snapConn, err := snapLn.Accept()
	if err != nil {
		return cell{}, fmt.Errorf("memsnap: %w", err)
	}
	defer snapConn.Close()
	if err := waitReady(addr, 20*time.Second); err != nil {
		return cell{}, err
	}
	var before procSample
	var beforeErr error
	var rssMu sync.Mutex
	var rssSamples []uint64
	rssStop := make(chan struct{})
	var winStart time.Time
	var winMu sync.Mutex
	stop := make(chan struct{})
	var stopOnce sync.Once
	halt := func() { stopOnce.Do(func() { close(stop) }) }
	if gcStop > 0 {
		watchDone := make(chan struct{})
		defer close(watchDone)
		go func() {
			t := time.NewTicker(100 * time.Millisecond)
			defer t.Stop()
			for {
				select {
				case <-watchDone:
					return
				case <-t.C:
					winMu.Lock()
					ws := winStart
					winMu.Unlock()
					if ws.IsZero() {
						continue
					}
					gcMu.Lock()
					n := gcCyclesInWindow(marks, ws.Sub(procStart).Seconds(), time.Since(procStart).Seconds())
					gcMu.Unlock()
					if n >= gcStop {
						halt()
						return
					}
				}
			}
		}()
	}
	var warm, measured loadResult
	var bootMallocs, bootTotal uint64
	var bootErr error
	bootMallocs, bootTotal, bootErr = readMemSnap(snapConn)
	if bootErr != nil {
		return cell{}, bootErr
	}
	var snapMallocs, snapTotal uint64
	var snapErr error
	snapped := false
	takeSnap := func() {
		snapMallocs, snapTotal, snapErr = readMemSnap(snapConn)
		snapped = true
	}
	if client == "wrk" {
		var err error
		warm, err = runWrk(addr, rt, conns, warmup)
		if err != nil {
			return cell{}, err
		}
		before, beforeErr = readProc(cmd.Process.Pid)
		takeSnap()
		winMu.Lock()
		winStart = time.Now()
		winMu.Unlock()
		measured, err = runWrk(addr, rt, conns, measure)
		if err != nil {
			return cell{}, err
		}
	} else {
		var sessionStop <-chan struct{}
		if gcStop > 0 {
			sessionStop = stop
		}
		warm, measured = session(addr, req, rt.Want, conns, warmup, measure, func() {
			winMu.Lock()
			winStart = time.Now()
			winMu.Unlock()
			before, beforeErr = readProc(cmd.Process.Pid)
			takeSnap()
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
		}, sessionStop)
	}
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
	if warm.Requests == 0 && warm.RPS() <= 0 {
		return cell{}, fmt.Errorf("warmup produced no successful responses (errors %d)", warm.Errors)
	}
	if beforeErr != nil {
		return cell{}, beforeErr
	}
	if snapErr != nil {
		return cell{}, snapErr
	}
	if measured.Requests > 0 && !snapped {
		return cell{}, fmt.Errorf("memsnap: measured window started without a snapshot")
	}
	endMallocs, endTotal, err := readMemSnap(snapConn)
	if err != nil {
		return cell{}, err
	}
	var allocsPerReq, bytesPerReq *float64
	var allocsSetup, bytesSetup *float64
	if measured.Requests > 0 && endMallocs >= snapMallocs && endTotal >= snapTotal {
		a := float64(endMallocs-snapMallocs) / float64(measured.Requests)
		b := float64(endTotal-snapTotal) / float64(measured.Requests)
		allocsPerReq = &a
		bytesPerReq = &b
	}
	allReqs := warm.Requests + measured.Requests
	if allReqs > 0 && endMallocs >= bootMallocs && endTotal >= bootTotal {
		a := float64(endMallocs-bootMallocs) / float64(allReqs)
		b := float64(endTotal-bootTotal) / float64(allReqs)
		allocsSetup = &a
		bytesSetup = &b
	}
	after, err := readProc(cmd.Process.Pid)
	if err != nil {
		return cell{}, err
	}
	var p50, p99, p999 time.Duration
	if measured.FixedPct {
		p50 = measured.P50
		p99 = measured.P99
	} else {
		p50 = percentile(measured.Samples, 0.50)
		p99 = percentile(append([]int64(nil), measured.Samples...), 0.99)
		// percentile sorts in place; copy for p999 from the already sorted slice.
		p999 = percentile(measured.Samples, 0.999)
	}
	cycles := 0
	winMu.Lock()
	ws := winStart
	winMu.Unlock()
	if !ws.IsZero() {
		gcMu.Lock()
		cycles = gcCyclesInWindow(marks, ws.Sub(procStart).Seconds(), ws.Sub(procStart).Seconds()+measured.Elapsed.Seconds())
		gcMu.Unlock()
	}
	if os.Getenv("ZTFB_GCTRACE") == "1" {
		fmt.Fprintf(os.Stderr, "gc-window %s %s c=%d round %d cycles=%d elapsed=%.1fs\n", server, rt.Name, conns, round, cycles, measured.Elapsed.Seconds())
	}
	return cell{
		Server:            server,
		Route:             rt.Name,
		Conns:             conns,
		Round:             round,
		Config:            config,
		Client:            client,
		Codec:             codec,
		GCCycles:          cycles,
		RPS:               measured.RPS(),
		P50US:             float64(p50.Microseconds()),
		P99US:             float64(p99.Microseconds()),
		P999US:            float64(p999.Microseconds()),
		CPUUS:             cpuMicrosPerReq(before, after, measured.Requests),
		RSSBytes:          after.RSS,
		HWMBytes:          after.HWM,
		Errors:            measured.Errors,
		Requests:          measured.Requests,
		ElapsedS:          measured.Elapsed.Seconds(),
		RSSSamples:        copied,
		AllocsPerReq:      allocsPerReq,
		BytesPerReq:       bytesPerReq,
		AllocsPerReqSetup: allocsSetup,
		BytesPerReqSetup:  bytesSetup,
	}, nil
}

func pairedCell(label, route string, conns int, config, client string, zatrano, fiber []cell, zMed, fMed cell) (cell, error) {
	ratios, err := pairedRPSRatios(zatrano, fiber)
	if err != nil {
		return cell{}, fmt.Errorf("%s c=%d: %w", route, conns, err)
	}
	median, low, high := bootstrapMedianCI(ratios, bootstrapDraws, bootstrapSeed)
	p99Ratio := ratioOf(zMed.P99US, fMed.P99US)
	cpuRatio := ratioOf(zMed.CPUUS, fMed.CPUUS)
	return cell{
		Server:   label,
		Route:    route,
		Conns:    conns,
		Agg:      "paired",
		Config:   config,
		Client:   client,
		Codec:    zMed.Codec,
		Ratio:    median,
		CILow:    low,
		CIHigh:   high,
		P50Ratio: ratioOf(zMed.P50US, fMed.P50US),
		P99Ratio: p99Ratio,
		CPURatio: cpuRatio,
		RSSRatio: ratioOf(float64(zMed.HWMBytes), float64(fMed.HWMBytes)),
		Verdict:  cellVerdict(low, high, p99Ratio, cpuRatio),
		RPS:      median,
		P50US:    zMed.P50US,
		P99US:    zMed.P99US,
		CPUUS:    zMed.CPUUS,
		HWMBytes: zMed.HWMBytes,
		Errors:   zMed.Errors + fMed.Errors,
	}, nil
}

func serverEnv(serverProcs, config, codec string) []string {
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
		"ZTFB_CODEC":   codec,
	}
	if os.Getenv("ZTFB_GCTRACE") == "1" {
		prev := os.Getenv("GODEBUG")
		if !strings.Contains(prev, "gctrace=") {
			if prev == "" {
				set["GODEBUG"] = "gctrace=1"
			} else {
				set["GODEBUG"] = prev + ",gctrace=1"
			}
		}
	}
	if config == "pooled" {
		set["HTTP_POOL_REQUESTS"] = "true"
	}
	// default leaves the variable unset, which is the product default (pool on).
	dropInherited := config != "pooled"
	var out []string
	seen := map[string]bool{}
	for _, kv := range env {
		key, _, _ := strings.Cut(kv, "=")
		if dropInherited && key == "HTTP_POOL_REQUESTS" {
			continue
		}
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

type gcMark struct {
	at     float64
	forced bool
}

func calibrateWindow(exe, addr string, rt routeSpec, req []byte, conns int, warmup, cap time.Duration, serverMask uintptr, pin bool, serverProcs, client, config, codec string, servers []string, gcMin int) (time.Duration, error) {
	if cap < 5*time.Second {
		cap = 180 * time.Second
	}
	var longest time.Duration
	for _, server := range servers {
		row, err := oneRound(exe, server, addr, rt, req, conns, 0, warmup, cap, serverMask, pin, serverProcs, false, client, config, codec, gcMin)
		if err != nil {
			return 0, err
		}
		if row.GCCycles < gcMin {
			return 0, fmt.Errorf("%s %s c=%d: %d GC cycles in %.1fs, need %d", server, rt.Name, conns, row.GCCycles, row.ElapsedS, gcMin)
		}
		d := time.Duration(row.ElapsedS * float64(time.Second))
		if d > longest {
			longest = d
		}
	}
	longest += longest / 10
	if longest < 5*time.Second {
		longest = 5 * time.Second
	}
	return longest, nil
}

func parseGCLine(line string) (at float64, forced, ok bool) {
	// gc 5 @5.146s 0%: ...
	if !strings.HasPrefix(line, "gc ") {
		return 0, false, false
	}
	atPos := strings.Index(line, " @")
	if atPos < 0 {
		return 0, false, false
	}
	rest := line[atPos+2:]
	end := strings.IndexByte(rest, 's')
	if end <= 0 {
		return 0, false, false
	}
	v, err := strconv.ParseFloat(rest[:end], 64)
	if err != nil {
		return 0, false, false
	}
	return v, strings.Contains(line, "(forced)"), true
}

func gcCyclesInWindow(marks []gcMark, start, end float64) int {
	n := 0
	for _, m := range marks {
		if m.at >= start && m.at <= end {
			n++
		}
	}
	return n
}

func medianCell(server, route string, conns int, rows []cell) cell {
	var rps []float64
	var p50, p99, p999, cpu []float64
	var rss, hwm, gcs, allocs, bytesPer, allocsSetup, bytesSetup []float64
	var errs, reqs int
	for _, row := range rows {
		rps = append(rps, row.RPS)
		p50 = append(p50, row.P50US)
		p99 = append(p99, row.P99US)
		p999 = append(p999, row.P999US)
		cpu = append(cpu, row.CPUUS)
		rss = append(rss, float64(row.RSSBytes))
		hwm = append(hwm, float64(row.HWMBytes))
		gcs = append(gcs, float64(row.GCCycles))
		if row.AllocsPerReq != nil {
			allocs = append(allocs, *row.AllocsPerReq)
		}
		if row.BytesPerReq != nil {
			bytesPer = append(bytesPer, *row.BytesPerReq)
		}
		if row.AllocsPerReqSetup != nil {
			allocsSetup = append(allocsSetup, *row.AllocsPerReqSetup)
		}
		if row.BytesPerReqSetup != nil {
			bytesSetup = append(bytesSetup, *row.BytesPerReqSetup)
		}
		errs += row.Errors
		reqs += row.Requests
	}
	return cell{
		Server:            server,
		Route:             route,
		Conns:             conns,
		Agg:               "median",
		GCCycles:          int(medianFloat(gcs) + 0.5),
		RPS:               medianFloat(rps),
		P50US:             medianFloat(p50),
		P99US:             medianFloat(p99),
		P999US:            medianFloat(p999),
		CPUUS:             medianFloat(cpu),
		RSSBytes:          uint64(medianFloat(rss)),
		HWMBytes:          uint64(medianFloat(hwm)),
		Errors:            errs,
		Requests:          reqs,
		AllocsPerReq:      medianPtr(allocs),
		BytesPerReq:       medianPtr(bytesPer),
		AllocsPerReqSetup: medianPtr(allocsSetup),
		BytesPerReqSetup:  medianPtr(bytesSetup),
	}
}

func pairOrder(pair string) []string {
	if pair == "rawhttp" {
		return []string{"rawhttp", "fasthttp"}
	}
	return []string{"zatrano", "fiber", "fiber-tuned"}
}

func rotate(servers []string, round int) []string {
	if len(servers) == 0 {
		return nil
	}
	k := (round - 1) % len(servers)
	if k < 0 {
		k = 0
	}
	out := make([]string, len(servers))
	for i := range servers {
		out[i] = servers[(i+k)%len(servers)]
	}
	return out
}

func medianPtr(xs []float64) *float64 {
	if len(xs) == 0 {
		return nil
	}
	v := medianFloat(xs)
	return &v
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

func serverProcCount(mask uintptr) int {
	n := 0
	for cpu := 0; cpu < 64; cpu++ {
		if mask&(1<<uintptr(cpu)) != 0 {
			n++
		}
	}
	if n < 1 {
		return 1
	}
	return n
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
