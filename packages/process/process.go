package process

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Result holds process output.
type Result struct {
	Command  string
	Args     []string
	ExitCode int
	Stdout   string
	Stderr   string
	Err      error
}

// Successful reports whether the process exited with 0.
func (r Result) Successful() bool {
	return r.Err == nil && r.ExitCode == 0
}

// Failed reports whether the process failed.
func (r Result) Failed() bool {
	return !r.Successful()
}

// Output returns combined stdout.
func (r Result) Output() string {
	return r.Stdout
}

// ErrorOutput returns stderr.
func (r Result) ErrorOutput() string {
	return r.Stderr
}

// PendingProcess builds a process invocation.
type PendingProcess struct {
	command string
	args    []string
	dir     string
	env     []string
	timeout time.Duration
	input   string
}

// Command creates a pending process.
func Command(name string, args ...string) *PendingProcess {
	return &PendingProcess{command: name, args: args}
}

// Path sets the working directory.
func (p *PendingProcess) Path(dir string) *PendingProcess {
	p.dir = dir
	return p
}

// Env appends environment variables (KEY=VALUE).
func (p *PendingProcess) Env(vars ...string) *PendingProcess {
	p.env = append(p.env, vars...)
	return p
}

// Timeout sets a process timeout.
func (p *PendingProcess) Timeout(d time.Duration) *PendingProcess {
	p.timeout = d
	return p
}

// Input sets stdin content.
func (p *PendingProcess) Input(input string) *PendingProcess {
	p.input = input
	return p
}

// Run executes the process.
func (p *PendingProcess) Run() Result {
	ctx := context.Background()
	cancel := func() {}
	if p.timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, p.timeout)
	}
	defer cancel()

	cmd := exec.CommandContext(ctx, p.command, p.args...)
	cmd.Dir = p.dir
	if len(p.env) > 0 {
		cmd.Env = append(cmd.Environ(), p.env...)
	}
	if p.input != "" {
		cmd.Stdin = strings.NewReader(p.input)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}

	return Result{
		Command:  p.command,
		Args:     p.args,
		ExitCode: exitCode,
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		Err:      err,
	}
}

// MustRun runs and panics on failure.
func (p *PendingProcess) MustRun() Result {
	result := p.Run()
	if result.Failed() {
		panic(fmt.Sprintf("process failed: %s %v: %v (%s)", p.command, p.args, result.Err, result.Stderr))
	}
	return result
}

// Pipe runs commands sequentially, piping stdout of each into the next stdin.
func Pipe(commands ...*PendingProcess) Result {
	if len(commands) == 0 {
		return Result{Err: fmt.Errorf("no commands provided")}
	}
	input := ""
	var last Result
	for _, command := range commands {
		if input != "" {
			command.Input(input)
		}
		last = command.Run()
		if last.Failed() {
			return last
		}
		input = last.Stdout
	}
	return last
}
