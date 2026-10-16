// Command summarize writes SUMMARY.md for a dedicated paired run and,
// for a full performance-governor run, replaces the README bench-env note.
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type row struct {
	Server  string  `json:"server"`
	Route   string  `json:"route"`
	Conns   int     `json:"conns"`
	Agg     string  `json:"agg"`
	Config  string  `json:"config"`
	Client  string  `json:"client"`
	Verdict string  `json:"verdict"`
	Ratio   float64 `json:"ratio"`
	CILow   float64 `json:"ci_low"`
	CIHigh  float64 `json:"ci_high"`
	P50     float64 `json:"p50_ratio"`
	P99     float64 `json:"p99_ratio"`
	CPU     float64 `json:"cpu_ratio"`
	RSS     float64 `json:"rss_ratio"`
}

type cell struct {
	client, config, route string
	conns                 int
	verdicts              []string
	ratio, lo, hi         []float64
	p50, p99, cpu, rss    []float64
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: summarize <dedicated-dir>")
		os.Exit(2)
	}
	dir := os.Args[1]
	env := readEnv(filepath.Join(dir, "env.txt"))
	cells := map[string]*cell{}
	matches, err := filepath.Glob(filepath.Join(dir, "*.jsonl"))
	if err != nil {
		fatal(err)
	}
	if len(matches) == 0 {
		fatal(fmt.Errorf("no jsonl in %s", dir))
	}
	for _, path := range matches {
		f, err := os.Open(path)
		if err != nil {
			fatal(err)
		}
		sc := bufio.NewScanner(f)
		sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
		for sc.Scan() {
			var r row
			if err := json.Unmarshal(sc.Bytes(), &r); err != nil {
				f.Close()
				fatal(err)
			}
			if r.Agg != "paired" || r.Server != "zatrano/fiber-tuned" {
				continue
			}
			key := fmt.Sprintf("%s|%s|%s|%d", r.Client, r.Config, r.Route, r.Conns)
			c := cells[key]
			if c == nil {
				c = &cell{client: r.Client, config: r.Config, route: r.Route, conns: r.Conns}
				cells[key] = c
			}
			c.verdicts = append(c.verdicts, r.Verdict)
			c.ratio = append(c.ratio, r.Ratio)
			c.lo = append(c.lo, r.CILow)
			c.hi = append(c.hi, r.CIHigh)
			c.p50 = append(c.p50, r.P50)
			c.p99 = append(c.p99, r.P99)
			c.cpu = append(c.cpu, r.CPU)
			c.rss = append(c.rss, r.RSS)
		}
		if err := sc.Err(); err != nil {
			f.Close()
			fatal(err)
		}
		f.Close()
	}

	var keys []string
	for k := range cells {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		a, b := cells[keys[i]], cells[keys[j]]
		if a.config != b.config {
			return a.config < b.config
		}
		if a.client != b.client {
			return a.client < b.client
		}
		if routeOrder(a.route) != routeOrder(b.route) {
			return routeOrder(a.route) < routeOrder(b.route)
		}
		return a.conns < b.conns
	})

	var b strings.Builder
	fmt.Fprintf(&b, "# Dedicated run %s\n\n", env["date"])
	fmt.Fprintf(&b, "CPU %s, %s cores, kernel %s, governor %s, %s, wrk %s, commit %s. Rounds %s, runs %s. Binding row `zatrano/fiber-tuned`. A class is stable when at least two runs agree. Numbers are the median of the run summaries.\n\n",
		env["cpu_model"], env["cores"], env["kernel"], env["governor"], env["go_version"], env["wrk_version"], env["commit"], env["rounds"], env["runs"])
	for _, title := range []struct{ config, heading string }{
		{"default", "default (pool off)"},
		{"pooled", "HTTP_POOL_REQUESTS=true"},
	} {
		fmt.Fprintf(&b, "## %s\n\n", title.heading)
		b.WriteString("| Client | Route | Conns | Ratio | CI low | CI high | p50 | p99 | CPU/req | Peak RSS | Verdict |\n")
		b.WriteString("|---|---|---:|---:|---:|---:|---:|---:|---:|---:|---|\n")
		for _, k := range keys {
			c := cells[k]
			if c.config != title.config {
				continue
			}
			fmt.Fprintf(&b, "| %s | %s | %d | %.3f | %.3f | %.3f | %.3f | %.3f | %.3f | %.3f | %s |\n",
				c.client, c.route, c.conns, median(c.ratio), median(c.lo), median(c.hi), median(c.p50), median(c.p99), median(c.cpu), median(c.rss), majority(c.verdicts))
		}
		b.WriteString("\n")
		for _, client := range []string{"go", "wrk"} {
			counts := map[string]int{}
			for _, k := range keys {
				c := cells[k]
				if c.config == title.config && c.client == client {
					counts[majority(c.verdicts)]++
				}
			}
			fmt.Fprintf(&b, "%s: ustun %d, esit %d, esik-yok %d, fiber %d, kararsiz %d.\n",
				client, counts["ustun"], counts["esit"], counts["esik-yok"], counts["fiber"], counts["kararsiz"])
		}
		b.WriteString("\n")
	}
	if err := os.WriteFile(filepath.Join(dir, "SUMMARY.md"), []byte(b.String()), 0o644); err != nil {
		fatal(err)
	}

	full := env["rounds"] == "20" && env["runs"] == "3" && env["quick"] != "1"
	if !full || env["governor"] != "performance" || env["allow_any_governor"] == "1" {
		return
	}
	note, changed := readmeNote(env, cells)
	if err := replaceBenchEnv(note); err != nil {
		fatal(err)
	}
	if err := commitReadme(env["date"]); err != nil {
		fatal(err)
	}
	if len(changed) > 0 {
		fmt.Println(strings.Join(changed, "\n"))
	}
}

type tally struct{ ustun, esit, miss, fiber int }

func readmeNote(env map[string]string, cells map[string]*cell) (string, []string) {
	var onGo, onWrk, offGo, offWrk tally
	var changed []string
	for _, c := range cells {
		got := majority(c.verdicts)
		switch {
		case c.config == "pooled" && c.client == "go":
			add(&onGo, got)
		case c.config == "pooled" && c.client == "wrk":
			add(&onWrk, got)
		case c.config == "default" && c.client == "go":
			add(&offGo, got)
		case c.config == "default" && c.client == "wrk":
			add(&offWrk, got)
		}
		want, ok := published[fmt.Sprintf("%s|%s|%s|%d", c.client, c.config, c.route, c.conns)]
		if ok && got != want {
			changed = append(changed, fmt.Sprintf("%s %s %s %d: %s -> %s", c.client, c.config, c.route, c.conns, want, got))
		}
	}
	sort.Strings(changed)
	if onGo.ustun == 16 && onWrk.ustun == 16 && offGo.fiber == 0 && offWrk.fiber == 0 {
		note := fmt.Sprintf("Reproduced on dedicated hardware (%s, %s cores, governor performance, Go %s, %s). Pool on: go %d ustun; wrk %d ustun. Pool off: go %d ustun, %d esit, %d esik-yok, %d fiber; wrk %d ustun, %d esit, %d esik-yok, %d fiber.",
			env["cpu_model"], env["cores"], env["go_version"], env["date"],
			onGo.ustun, onWrk.ustun,
			offGo.ustun, offGo.esit, offGo.miss, offGo.fiber,
			offWrk.ustun, offWrk.esit, offWrk.miss, offWrk.fiber)
		return note, changed
	}
	var b strings.Builder
	fmt.Fprintf(&b, "In the dedicated run ZATRANO led in %d of 16 cells with pooling on and trailed Fiber in %d cells with pooling off (Go client). ", onGo.ustun, offGo.fiber)
	fmt.Fprintf(&b, "In the dedicated run ZATRANO led in %d of 16 cells with pooling on and trailed Fiber in %d cells with pooling off (wrk).", onWrk.ustun, offWrk.fiber)
	if len(changed) > 0 {
		b.WriteString(" Changed cells: ")
		b.WriteString(strings.Join(changed, "; "))
		b.WriteString(".")
	}
	return b.String(), changed
}

func add(t *tally, verdict string) {
	switch verdict {
	case "ustun":
		t.ustun++
	case "esit":
		t.esit++
	case "esik-yok":
		t.miss++
	case "fiber":
		t.fiber++
	}
}

func replaceBenchEnv(note string) error {
	root, err := repoRoot()
	if err != nil {
		return err
	}
	path := filepath.Join(root, "README.md")
	body, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	start := []byte("<!-- bench-env:start -->")
	end := []byte("<!-- bench-env:end -->")
	i := bytes.Index(body, start)
	j := bytes.Index(body, end)
	if i < 0 || j < 0 || j < i {
		return fmt.Errorf("bench-env markers missing in %s", path)
	}
	var buf bytes.Buffer
	buf.Write(body[:i+len(start)])
	buf.WriteString("\n")
	buf.WriteString(note)
	buf.WriteString("\n")
	buf.Write(body[j:])
	return os.WriteFile(path, buf.Bytes(), 0o644)
}

func commitReadme(day string) error {
	root, err := repoRoot()
	if err != nil {
		return err
	}
	diff := exec.Command("git", "-C", root, "diff", "--quiet", "--", "README.md")
	if err := diff.Run(); err == nil {
		return nil
	}
	msg := "docs(bench): record dedicated-hardware run " + day
	add := exec.Command("git", "-C", root, "add", "--", "README.md")
	add.Stdout, add.Stderr = os.Stdout, os.Stderr
	if err := add.Run(); err != nil {
		return err
	}
	commit := exec.Command("git", "-C", root, "commit", "-m", msg, "--", "README.md")
	commit.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=serhankarakoc",
		"GIT_AUTHOR_EMAIL=serhankarakoc@gmail.com",
		"GIT_COMMITTER_NAME=serhankarakoc",
		"GIT_COMMITTER_EMAIL=serhankarakoc@gmail.com",
	)
	commit.Stdout, commit.Stderr = os.Stdout, os.Stderr
	return commit.Run()
}

func repoRoot() (string, error) {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func readEnv(path string) map[string]string {
	f, err := os.Open(path)
	if err != nil {
		fatal(err)
	}
	defer f.Close()
	out := map[string]string{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		k, v, ok := strings.Cut(sc.Text(), "=")
		if ok {
			out[k] = v
		}
	}
	if err := sc.Err(); err != nil {
		fatal(err)
	}
	return out
}

func majority(values []string) string {
	counts := map[string]int{}
	best, n := "", 0
	for _, v := range values {
		counts[v]++
		if counts[v] > n {
			best, n = v, counts[v]
		}
	}
	if n >= 2 {
		return best
	}
	return "kararsiz"
}

func median(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	s := append([]float64(nil), xs...)
	sort.Float64s(s)
	return s[len(s)/2]
}

func routeOrder(route string) int {
	switch route {
	case "GET /plaintext":
		return 0
	case "GET /json":
		return 1
	case "GET /users/{id}":
		return 2
	case "POST /echo":
		return 3
	default:
		return 9
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

// published is the shared-runner class at b966476 for the cells this script runs.
// A dedicated run lists a cell when its stable class differs.
var published = map[string]string{}

func init() {
	offGo := map[string]string{
		"GET /plaintext|64": "ustun", "GET /plaintext|256": "esik-yok", "GET /plaintext|1024": "esik-yok", "GET /plaintext|4096": "esik-yok",
		"GET /json|64": "ustun", "GET /json|256": "esik-yok", "GET /json|1024": "ustun", "GET /json|4096": "ustun",
		"GET /users/{id}|64": "ustun", "GET /users/{id}|256": "esik-yok", "GET /users/{id}|1024": "ustun", "GET /users/{id}|4096": "ustun",
		"POST /echo|64": "ustun", "POST /echo|256": "ustun", "POST /echo|1024": "ustun", "POST /echo|4096": "ustun",
	}
	offWrk := map[string]string{
		"GET /plaintext|64": "esit", "GET /plaintext|256": "esit", "GET /plaintext|1024": "esit", "GET /plaintext|4096": "esik-yok",
		"GET /json|64": "ustun", "GET /json|256": "ustun", "GET /json|1024": "ustun", "GET /json|4096": "ustun",
		"GET /users/{id}|64": "ustun", "GET /users/{id}|256": "ustun", "GET /users/{id}|1024": "ustun", "GET /users/{id}|4096": "ustun",
		"POST /echo|64": "ustun", "POST /echo|256": "ustun", "POST /echo|1024": "ustun", "POST /echo|4096": "ustun",
	}
	for k, v := range offGo {
		route, conns, _ := strings.Cut(k, "|")
		n, _ := strconv.Atoi(conns)
		published[fmt.Sprintf("go|default|%s|%d", route, n)] = v
	}
	for k, v := range offWrk {
		route, conns, _ := strings.Cut(k, "|")
		n, _ := strconv.Atoi(conns)
		published[fmt.Sprintf("wrk|default|%s|%d", route, n)] = v
	}
	for _, client := range []string{"go", "wrk"} {
		for _, route := range []string{"GET /plaintext", "GET /json", "GET /users/{id}", "POST /echo"} {
			for _, conns := range []int{64, 256, 1024, 4096} {
				published[fmt.Sprintf("%s|pooled|%s|%d", client, route, conns)] = "ustun"
			}
		}
	}
}
