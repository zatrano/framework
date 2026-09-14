package console

import (
	"context"
	"errors"

	"github.com/zatrano/framework/v2/kernel"
)

// Classified CLI exit codes. Interpretation stays at the CLI boundary;
// the acquire package does not own presentation codes.
const (
	ExitSuccess     = 0
	ExitGeneral     = 1
	ExitUsage       = 2
	ExitResolution  = 3
	ExitPlanning    = 4
	ExitAcquisition = 5
	ExitEnablement  = 6
	ExitCanceled    = 7

	// Runtime serve/Run classifications. Acquisition keeps 2–7.
	ExitRuntimeBoot     = 20
	ExitRuntimeShutdown = 21
	ExitRuntimeCanceled = 22
	ExitRuntimeTimeout  = 23
)

// CLIError is a command failure with a deterministic exit code.
type CLIError struct {
	Code int
	Err  error
}

func (e *CLIError) Error() string {
	if e == nil || e.Err == nil {
		return ""
	}
	return e.Err.Error()
}

func (e *CLIError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func (e *CLIError) ExitCode() int {
	if e == nil || e.Code == 0 {
		return ExitGeneral
	}
	return e.Code
}

// CodeFromError maps a command error to a process exit code.
func CodeFromError(err error) int {
	if err == nil {
		return ExitSuccess
	}
	var ce *CLIError
	if errors.As(err, &ce) {
		return ce.ExitCode()
	}
	return ExitGeneral
}

func cliErr(code int, err error) error {
	if err == nil {
		return nil
	}
	var ce *CLIError
	if errors.As(err, &ce) {
		return ce
	}
	return &CLIError{Code: code, Err: err}
}

func classifyContextError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return cliErr(ExitCanceled, err)
	}
	return nil
}

// classifyRuntimeError maps serve/Run failures. It must not reuse
// acquisition codes 2–7 (including ExitCanceled).
func classifyRuntimeError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) {
		return cliErr(ExitRuntimeCanceled, err)
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return cliErr(ExitRuntimeTimeout, err)
	}
	if errors.Is(err, kernel.ErrRuntimeBoot) {
		return cliErr(ExitRuntimeBoot, err)
	}
	if errors.Is(err, kernel.ErrRuntimeShutdown) {
		return cliErr(ExitRuntimeShutdown, err)
	}
	return cliErr(ExitGeneral, err)
}
