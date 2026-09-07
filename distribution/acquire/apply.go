package acquire

import (
	"context"
	"fmt"
	"strings"
)

// Request is one `go get <GoGetArg>` invocation against an application module root.
// GoGetArg must be a frozen Phase 7 token (Plan.GoGetArg). This is not enablement.
type Request struct {
	Root     string
	GoGetArg string
	Go       string
}

// Execute runs `go get` through ExecRunner. It is the acquisition execution
// entry: frozen GoGetArg as argv, explicit module root as Dir, streams and
// exit status on InvocationResult. It does not inspect go.mod / go.sum,
// run tidy, resolve latest, or invent ApplyResult.
func Execute(ctx context.Context, req Request) (InvocationResult, error) {
	return Invoke(ctx, ExecRunner{}, req)
}

// Invoke asks Runner to run `go get` with the concrete Phase 7 argument as-is.
// It does not resolve packages, rewrite versions, run tidy, or edit go.mod as text.
func Invoke(ctx context.Context, runner Runner, req Request) (InvocationResult, error) {
	if ctx == nil {
		return InvocationResult{}, fmt.Errorf("acquire: nil context")
	}
	if runner == nil {
		return InvocationResult{}, fmt.Errorf("acquire: nil runner")
	}
	root := strings.TrimSpace(req.Root)
	if root == "" {
		return InvocationResult{}, fmt.Errorf("acquire: module root required")
	}
	arg := strings.TrimSpace(req.GoGetArg)
	if arg == "" {
		return InvocationResult{}, fmt.Errorf("acquire: empty GoGetArg")
	}
	if strings.EqualFold(arg, "latest") || strings.HasSuffix(strings.ToLower(arg), "@latest") {
		return InvocationResult{}, fmt.Errorf("acquire: latest must not reach Apply")
	}
	goBin := strings.TrimSpace(req.Go)
	if goBin == "" {
		goBin = "go"
	}
	return runner.Run(ctx, Invocation{
		Path: goBin,
		Args: []string{"get", arg},
		Dir:  root,
	})
}
