package acquire

import "context"

// Invocation is one structured process call. Path, args, and dir are explicit.
// It is not a shell string and not a go.mod editor.
type Invocation struct {
	Path string
	Args []string
	Dir  string
}

// InvocationResult is what the process boundary can observe.
// Inspect owns go.mod / go.sum observation; this type is process-only.
type InvocationResult struct {
	Invocation Invocation
	ExitCode   int
	Stdout     string
	Stderr     string
}

// Runner executes an Invocation. Tests inject a fake; production uses ExecRunner.
type Runner interface {
	Run(ctx context.Context, inv Invocation) (InvocationResult, error)
}
