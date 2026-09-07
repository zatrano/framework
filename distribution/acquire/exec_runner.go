package acquire

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
)

// ExecRunner runs an Invocation with CommandContext. It is the only acquire
// file allowed to call exec. Invoke tests stay on a fake Runner; Execute
// is the acquisition path that uses this type.
type ExecRunner struct{}

func (ExecRunner) Run(ctx context.Context, inv Invocation) (InvocationResult, error) {
	if ctx == nil {
		return InvocationResult{}, errors.New("acquire: nil context")
	}
	cmd := exec.CommandContext(ctx, inv.Path, append([]string{}, inv.Args...)...)
	cmd.Dir = inv.Dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	res := InvocationResult{
		Invocation: inv,
		Stdout:     stdout.String(),
		Stderr:     stderr.String(),
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		res.ExitCode = ee.ExitCode()
		return res, err
	}
	if err != nil {
		return res, err
	}
	return res, nil
}
