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

// TargetStatus is SPEC §15: success, failed, or unattempted.
type TargetStatus string

const (
	StatusSuccess     TargetStatus = "success"
	StatusFailed      TargetStatus = "failed"
	StatusUnattempted TargetStatus = "unattempted"
)

// TargetReport is one acquisition target in an ApplyResult.
type TargetReport struct {
	GoGetArg string
	Status   TargetStatus
	Result   InvocationResult
	Err      error
}

// ApplyResult is SPEC §15 partial-apply reporting. Recovery (SPEC §16) is
// unavailable unless the caller attaches a RecoverFiles outcome. It is not
// enablement and not a claim that all targets were acquired.
type ApplyResult struct {
	Root     string
	Reports  []TargetReport
	Recovery Recovery
}

// WithRecovery keeps target reports and records a file-restore outcome.
// It does not rewrite success, failed, or unattempted.
func (r ApplyResult) WithRecovery(rec Recovery) ApplyResult {
	r.Recovery = rec
	return r
}

// Successful returns GoGetArg values reported as success, in input order.
func (r ApplyResult) Successful() []string {
	return r.argsWith(StatusSuccess)
}

// Failed returns GoGetArg values reported as failed, in input order.
func (r ApplyResult) Failed() []string {
	return r.argsWith(StatusFailed)
}

// Unattempted returns GoGetArg values that fail-fast never started.
func (r ApplyResult) Unattempted() []string {
	return r.argsWith(StatusUnattempted)
}

func (r ApplyResult) argsWith(status TargetStatus) []string {
	var out []string
	for _, rep := range r.Reports {
		if rep.Status == status {
			out = append(out, rep.GoGetArg)
		}
	}
	return out
}

// Execute runs `go get` through ExecRunner under a per-module-root mutation
// lock. Same root is serialized; other roots proceed independently. Inspect
// is not locked. It does not run tidy, resolve latest, or report partial apply.
func Execute(ctx context.Context, req Request) (InvocationResult, error) {
	return execute(ctx, ExecRunner{}, req)
}

func execute(ctx context.Context, runner Runner, req Request) (InvocationResult, error) {
	unlock, err := lockMutation(ctx, strings.TrimSpace(req.Root))
	if err != nil {
		return InvocationResult{}, err
	}
	defer unlock()
	return Invoke(ctx, runner, req)
}

// ExecuteTargets applies each GoGetArg in order under one per-root mutation
// lock. Fail-fast: the first failure stops the sequence; later targets stay
// unattempted. Earlier successes are not rolled back and files are not
// restored. The result is never “all targets acquired”.
func ExecuteTargets(ctx context.Context, root string, args []string) (ApplyResult, error) {
	return executeTargets(ctx, ExecRunner{}, Request{Root: root}, args)
}

func executeTargets(ctx context.Context, runner Runner, req Request, args []string) (ApplyResult, error) {
	root := strings.TrimSpace(req.Root)
	out := ApplyResult{
		Root:     root,
		Reports:  make([]TargetReport, 0, len(args)),
		Recovery: Recovery{Kind: RecoveryUnavailable},
	}
	for _, arg := range args {
		out.Reports = append(out.Reports, TargetReport{
			GoGetArg: strings.TrimSpace(arg),
			Status:   StatusUnattempted,
		})
	}
	if len(args) == 0 {
		return out, nil
	}
	unlock, err := lockMutation(ctx, root)
	if err != nil {
		return out, err
	}
	defer unlock()
	for i, arg := range args {
		res, err := Invoke(ctx, runner, Request{Root: root, GoGetArg: arg, Go: req.Go})
		if err == nil && res.ExitCode != 0 {
			err = fmt.Errorf("acquire: go get %s: exit %d", strings.TrimSpace(arg), res.ExitCode)
		}
		if err != nil {
			out.Reports[i] = TargetReport{
				GoGetArg: strings.TrimSpace(arg),
				Status:   StatusFailed,
				Result:   res,
				Err:      err,
			}
			return out, err
		}
		out.Reports[i] = TargetReport{
			GoGetArg: strings.TrimSpace(arg),
			Status:   StatusSuccess,
			Result:   res,
		}
	}
	return out, nil
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
