// Command rotate builds one test binary per variant and samples them in
// A B C A B C order, with a 3s gap, so a warm core does not stay on one binary.
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
	"time"
)

type built struct {
	name  string
	exe   string
	bench string
}

func main() {
	benchDir, err := os.Getwd()
	if err != nil {
		fatal(err)
	}
	if filepath.Base(benchDir) == "rotate" {
		benchDir = filepath.Dir(filepath.Dir(benchDir))
	}
	fmt.Println(powerPlan())
	root := filepath.Dir(benchDir)
	out := filepath.Join(os.TempDir(), "zat-rotate")
	_ = os.RemoveAll(out)
	if err := os.MkdirAll(out, 0o755); err != nil {
		fatal(err)
	}

	headExe := filepath.Join(out, "head.exe")
	fmt.Fprintln(os.Stderr, "compile HEAD")
	if err := compileAt(benchDir, headExe); err != nil {
		fatal(err)
	}

	revs := []struct{ name, rev string }{
		{"v3.0.1", "e4c09693fe9359e5883f3943124125239b05a61e"},
		{"2fb0316", "2fb0316ab4ffd921ae9e1ef956d901da98d034c0"},
		{"2546ae4", "2546ae49bb3e2ced0126bb1e01fe497ec20cc1ea"},
		{"03d734c", "03d734cf40e63646118c8bead8863338a013ee50"},
		{"dc47e3b", "dc47e3ba2bf4eb367d3652bf0293b9c16c619df2"},
		{"2e5d22f", "2e5d22fa0157bffa88ad9a92d5601dc532fd3123"},
		{"1fb5597", "1fb55972c5e11cf00f70dcd588a9c5033bd5af80"},
	}
	var trees []string
	defer func() {
		for _, wt := range trees {
			_ = exec.Command("git", "-C", root, "worktree", "remove", "--force", wt).Run()
		}
	}()
	bins := map[string]string{}
	for _, rev := range revs {
		wt := filepath.Join(out, "wt-"+rev.name)
		_ = exec.Command("git", "-C", root, "worktree", "remove", "--force", wt).Run()
		cmd := exec.Command("git", "-C", root, "worktree", "add", "--detach", wt, rev.rev)
		cmd.Stdout = os.Stderr
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fatal(err)
		}
		trees = append(trees, wt)
		dir := filepath.Join(out, "src-"+rev.name)
		if err := copyBench(benchDir, dir, wt); err != nil {
			fatal(err)
		}
		exe := filepath.Join(out, rev.name+".exe")
		fmt.Fprintln(os.Stderr, "compile", rev.name)
		if err := compileAt(dir, exe); err != nil {
			fatal(err)
		}
		bins[rev.name] = exe
	}
	for _, name := range []string{"v3.0.1", "1fb5597"} {
		fmt.Fprintln(os.Stderr, "deadlines", name)
		cmd := exec.Command(bins[name], "-test.run", "^TestDeadlineCalls$", "-test.v", "-test.count", "1")
		cmd.Env = append(os.Environ(), "GOWORK=off")
		out, err := cmd.CombinedOutput()
		fmt.Println(string(out))
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}

	// HEAD process variants. Each name is its own already-compiled binary
	// invocation; revision binaries are separate go test -c outputs.
	var variants []built
	for _, name := range []string{
		"BenchmarkT0Old", "BenchmarkT0Run", "BenchmarkT0RunV301",
		"BenchmarkT1OldEcho", "BenchmarkT1RunEcho", "BenchmarkT1RunGen",
		"BenchmarkT0Fiber", "BenchmarkT1FiberEcho", "BenchmarkT1FiberGen",
		"BenchmarkT0Gin", "BenchmarkT1Gin", "BenchmarkT0EchoFrm", "BenchmarkT1EchoFrm",
	} {
		variants = append(variants, built{name: "HEAD/" + name, exe: headExe, bench: "^" + name + "$"})
	}
	for _, rev := range revs {
		variants = append(variants,
			built{name: rev.name + "/T0Old", exe: bins[rev.name], bench: "^BenchmarkT0Old$"},
			built{name: rev.name + "/T1OldEcho", exe: bins[rev.name], bench: "^BenchmarkT1OldEcho$"},
		)
	}

	const rounds = 10
	samples := map[string][]float64{}
	for round := 0; round < rounds; round++ {
		for _, v := range variants {
			ns, err := runOnce(v)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s round %d: %v\n", v.name, round+1, err)
				continue
			}
			samples[v.name] = append(samples[v.name], ns)
			fmt.Printf("SAMPLE %s round %d %.1f\n", v.name, round+1, ns)
			time.Sleep(3 * time.Second)
		}
	}
	printTable(samples)
	fmt.Fprintln(os.Stderr, "profiles")
	profile(bins["v3.0.1"], filepath.Join(out, "v301.prof"))
	profile(headExe, filepath.Join(out, "head.prof"))
	diff := exec.Command("go", "tool", "pprof", "-top", "-nodecount=40",
		"-diff_base="+filepath.Join(out, "v301.prof"), filepath.Join(out, "head.prof"))
	diff.Stdout = os.Stdout
	diff.Stderr = os.Stderr
	if err := diff.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "pprof", err)
	}
}

func compileAt(dir, exe string) error {
	cmd := exec.Command("go", "test", "-c", "-o", exe, ".")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOWORK=off")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %w\n%s", dir, err, out)
	}
	return nil
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
		name := e.Name()
		if !strings.HasSuffix(name, ".go") && name != "go.mod" && name != "go.sum" {
			continue
		}
		b, err := os.ReadFile(filepath.Join(src, name))
		if err != nil {
			return err
		}
		if name == "go.mod" {
			fw := filepath.ToSlash(framework)
			b = regexp.MustCompile(`(?m)^replace github.com/zatrano/framework/v3 => .*$`).ReplaceAll(b,
				[]byte("replace github.com/zatrano/framework/v3 => "+fw))
		}
		if err := os.WriteFile(filepath.Join(dst, name), b, 0o644); err != nil {
			return err
		}
	}
	return nil
}

var nsRe = regexp.MustCompile(`\s([\d.]+) ns/op`)

func runOnce(v built) (float64, error) {
	cmd := exec.Command(v.exe, "-test.bench", v.bench, "-test.run", "^$", "-test.benchtime", "1s", "-test.count", "1", "-test.benchmem")
	cmd.Env = append(os.Environ(), "GOWORK=off")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("%w\n%s", err, out)
	}
	var last float64
	found := false
	for _, line := range strings.Split(string(out), "\n") {
		if !strings.Contains(line, "ns/op") || strings.Contains(line, "Benchmark") == false {
			continue
		}
		m := nsRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		last, _ = strconv.ParseFloat(m[1], 64)
		found = true
	}
	if !found {
		return 0, fmt.Errorf("no ns/op in\n%s", out)
	}
	return last, nil
}

func profile(exe, path string) {
	cmd := exec.Command(exe, "-test.bench", "^BenchmarkT0Old$", "-test.run", "^$", "-test.benchtime", "3s", "-test.count", "1", "-test.cpuprofile", path)
	cmd.Env = append(os.Environ(), "GOWORK=off")
	out, err := cmd.CombinedOutput()
	fmt.Fprintln(os.Stderr, string(out))
	if err != nil {
		fmt.Fprintln(os.Stderr, "profile", err)
	}
}

func printTable(samples map[string][]float64) {
	names := make([]string, 0, len(samples))
	for name := range samples {
		names = append(names, name)
	}
	sort.Strings(names)
	fmt.Printf("%-40s %8s %10s %10s %10s\n", "variant", "n", "median", "min", "max")
	for _, name := range names {
		s := append([]float64(nil), samples[name]...)
		sort.Float64s(s)
		fmt.Printf("%-40s %8d %10.1f %10.1f %10.1f\n", name, len(s), median(s), s[0], s[len(s)-1])
	}
}

func median(s []float64) float64 {
	n := len(s)
	if n == 0 {
		return 0
	}
	if n%2 == 1 {
		return s[n/2]
	}
	return (s[n/2-1] + s[n/2]) / 2
}

func powerPlan() string {
	out, err := exec.Command("powercfg", "/getactivescheme").CombinedOutput()
	if err != nil {
		return "power plan: " + err.Error()
	}
	return "power plan: " + strings.TrimSpace(string(out))
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
