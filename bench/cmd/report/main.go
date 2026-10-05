// Command report runs the same harness against framework v3.0.1 and the
// module replaced in this directory (perf/v3 HEAD), then prints median/min/max.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

func main() {
	wd, err := os.Getwd()
	if err != nil {
		fatal(err)
	}
	// Allow `go run ./cmd/report` from the bench module.
	benchDir := wd
	if filepath.Base(wd) == "report" {
		benchDir = filepath.Dir(filepath.Dir(wd))
	}
	fmt.Println("HEAD (replace => ../)")
	head, err := runBench(benchDir, "BenchmarkTier")
	if err != nil {
		fatal(err)
	}
	printTables(head)

	fw := filepath.Dir(benchDir)
	wt, err := filepath.Abs(filepath.Join(os.TempDir(), "zat-fw-v301"))
	if err != nil {
		fatal(err)
	}
	_ = exec.Command("git", "-C", fw, "worktree", "remove", "--force", wt).Run()
	add := exec.Command("git", "-C", fw, "worktree", "add", "--detach", wt, "v3.0.1")
	add.Stdout = os.Stderr
	add.Stderr = os.Stderr
	if err := add.Run(); err != nil {
		fatal(err)
	}
	defer exec.Command("git", "-C", fw, "worktree", "remove", "--force", wt).Run()

	child := filepath.Join(os.TempDir(), "zat-bench-v301")
	_ = os.RemoveAll(child)
	if err := copyBench(benchDir, child, wt); err != nil {
		fatal(err)
	}
	fmt.Println("v3.0.1 (same harness, rawhttp v0.2.2)")
	base, err := runBench(child, "BenchmarkTier.*/Zatrano")
	if err != nil {
		fatal(err)
	}
	printTables(base)
}

func copyBench(src, dst, framework string) error {
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if !strings.HasSuffix(e.Name(), ".go") && e.Name() != "go.sum" && e.Name() != "go.mod" {
			continue
		}
		b, err := os.ReadFile(filepath.Join(src, e.Name()))
		if err != nil {
			return err
		}
		if e.Name() == "go.mod" {
			fw := filepath.ToSlash(framework)
			b = regexp.MustCompile(`(?m)^replace github.com/zatrano/framework/v3 => .*$`).ReplaceAll(b,
				[]byte("replace github.com/zatrano/framework/v3 => "+fw))
		}
		if err := os.WriteFile(filepath.Join(dst, e.Name()), b, 0o644); err != nil {
			return err
		}
	}
	return nil
}

type sample struct {
	ns, bytes, allocs float64
}

func runBench(dir, pattern string) (map[string][]sample, error) {
	args := []string{"test", "-bench", pattern, "-benchmem", "-count=10", "-run", "^$", "-timeout", "30m"}
	cmd := exec.Command("go", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOWORK=off")
	out, err := cmd.CombinedOutput()
	fmt.Fprint(os.Stderr, string(out))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", dir, err)
	}
	return parseBench(string(out)), nil
}

var benchLine = regexp.MustCompile(`^(Benchmark\S+)\s+\d+\s+([\d.]+) ns/op(?:\s+([\d.]+) B/op\s+(\d+) allocs/op)?`)

func parseBench(out string) map[string][]sample {
	got := map[string][]sample{}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		m := benchLine.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		name := stripCPU(m[1])
		ns, _ := strconv.ParseFloat(m[2], 64)
		s := sample{ns: ns}
		if m[3] != "" {
			s.bytes, _ = strconv.ParseFloat(m[3], 64)
			s.allocs, _ = strconv.ParseFloat(m[4], 64)
		}
		got[name] = append(got[name], s)
	}
	return got
}

func stripCPU(name string) string {
	i := strings.LastIndex(name, "-")
	if i < 0 {
		return name
	}
	if _, err := strconv.Atoi(name[i+1:]); err == nil {
		return name[:i]
	}
	return name
}

func printTables(rows map[string][]sample) {
	fmt.Printf("%-28s %12s %12s %12s %10s %10s %10s %10s\n",
		"benchmark", "ns med", "ns min", "ns max", "B med", "B min", "alloc med", "alloc max")
	names := make([]string, 0, len(rows))
	for name := range rows {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		s := rows[name]
		fmt.Printf("%-28s %12.1f %12.1f %12.1f %10.1f %10.1f %10.1f %10.1f\n",
			name, median(s, func(x sample) float64 { return x.ns }),
			min(s, func(x sample) float64 { return x.ns }),
			max(s, func(x sample) float64 { return x.ns }),
			median(s, func(x sample) float64 { return x.bytes }),
			min(s, func(x sample) float64 { return x.bytes }),
			median(s, func(x sample) float64 { return x.allocs }),
			max(s, func(x sample) float64 { return x.allocs }),
		)
	}
}

func median(s []sample, f func(sample) float64) float64 {
	if len(s) == 0 {
		return 0
	}
	v := make([]float64, len(s))
	for i := range s {
		v[i] = f(s[i])
	}
	sort.Float64s(v)
	n := len(v)
	if n%2 == 1 {
		return v[n/2]
	}
	return (v[n/2-1] + v[n/2]) / 2
}

func min(s []sample, f func(sample) float64) float64 {
	m := f(s[0])
	for _, x := range s[1:] {
		if f(x) < m {
			m = f(x)
		}
	}
	return m
}

func max(s []sample, f func(sample) float64) float64 {
	m := f(s[0])
	for _, x := range s[1:] {
		if f(x) > m {
			m = f(x)
		}
	}
	return m
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
