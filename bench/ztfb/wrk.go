package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func runWrk(addr string, rt routeSpec, conns int, d time.Duration) (loadResult, error) {
	if _, err := exec.LookPath("wrk"); err != nil {
		return loadResult{}, fmt.Errorf("wrk not found in PATH")
	}
	threads := 2
	args := []string{
		"-t", strconv.Itoa(threads),
		"-c", strconv.Itoa(conns),
		"-d", d.String(),
		"--latency",
	}
	var script string
	if rt.Body != "" {
		f, err := os.CreateTemp("", "ztfb-wrk-*.lua")
		if err != nil {
			return loadResult{}, err
		}
		script = f.Name()
		_, err = fmt.Fprintf(f, "wrk.method = %q\nwrk.body = %q\nwrk.headers[\"Content-Type\"] = \"application/json\"\n", rt.Method, rt.Body)
		_ = f.Close()
		if err != nil {
			return loadResult{}, err
		}
		defer os.Remove(script)
		args = append(args, "-s", script)
	}
	args = append(args, "http://"+addr+rt.Path)
	cmd := exec.Command("wrk", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return loadResult{}, fmt.Errorf("wrk: %w\n%s", err, out)
	}
	return parseWrk(string(out))
}

func parseWrk(out string) (loadResult, error) {
	var res loadResult
	res.FixedPct = true
	inDist := false
	var requests int
	var non2xx int
	var sockErr int
	for _, line := range strings.Split(out, "\n") {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "Latency Distribution") {
			inDist = true
			continue
		}
		if inDist {
			fields := strings.Fields(trim)
			if len(fields) >= 2 && (fields[0] == "50%" || fields[0] == "99%") {
				dur, err := parseWrkDuration(fields[1])
				if err != nil {
					return loadResult{}, err
				}
				if fields[0] == "50%" {
					res.P50 = dur
				} else {
					res.P99 = dur
				}
			}
			if trim == "" || strings.HasPrefix(trim, "Requests/sec") {
				inDist = false
			}
		}
		if strings.HasPrefix(trim, "Requests/sec:") {
			rps, err := strconv.ParseFloat(strings.TrimSpace(strings.TrimPrefix(trim, "Requests/sec:")), 64)
			if err != nil {
				return loadResult{}, fmt.Errorf("wrk rps: %w", err)
			}
			res.RPSFixed = rps
		}
		if strings.Contains(trim, "requests in ") {
			fields := strings.Fields(trim)
			if len(fields) > 0 {
				n, err := strconv.Atoi(fields[0])
				if err == nil {
					requests = n
				}
			}
			if i := strings.Index(trim, "requests in "); i >= 0 {
				rest := strings.TrimSpace(trim[i+len("requests in "):])
				fields = strings.Fields(rest)
				if len(fields) > 0 {
					if dur, err := parseWrkDuration(fields[0]); err == nil {
						res.Elapsed = dur
					}
				}
			}
		}
		if strings.HasPrefix(trim, "Non-2xx or 3xx responses:") {
			fields := strings.Fields(trim)
			if len(fields) > 0 {
				n, err := strconv.Atoi(fields[len(fields)-1])
				if err == nil {
					non2xx = n
				}
			}
		}
		if strings.HasPrefix(trim, "Socket errors:") {
			for _, field := range strings.Fields(trim) {
				n, err := strconv.Atoi(strings.Trim(field, ","))
				if err == nil {
					sockErr += n
				}
			}
		}
	}
	if res.RPSFixed <= 0 && requests == 0 {
		return loadResult{}, fmt.Errorf("wrk produced no requests\n%s", out)
	}
	res.Requests = requests
	res.Errors = non2xx + sockErr
	if res.Elapsed <= 0 {
		res.Elapsed = time.Second
	}
	if res.RPSFixed <= 0 {
		res.RPSFixed = float64(requests) / res.Elapsed.Seconds()
	}
	return res, nil
}

func parseWrkDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	switch {
	case strings.HasSuffix(s, "us"):
		n, err := strconv.ParseFloat(strings.TrimSuffix(s, "us"), 64)
		return time.Duration(n * float64(time.Microsecond)), err
	case strings.HasSuffix(s, "ms"):
		n, err := strconv.ParseFloat(strings.TrimSuffix(s, "ms"), 64)
		return time.Duration(n * float64(time.Millisecond)), err
	case strings.HasSuffix(s, "s"):
		n, err := strconv.ParseFloat(strings.TrimSuffix(s, "s"), 64)
		return time.Duration(n * float64(time.Second)), err
	default:
		return time.ParseDuration(s)
	}
}
