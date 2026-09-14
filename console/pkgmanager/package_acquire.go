package pkgmanager

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/zatrano/framework/v2/distribution/acquire"
	"github.com/zatrano/framework/v2/distribution/registry"
	"github.com/zatrano/framework/v2/kernel"
)

const (
	acquireStatusSuccess     = "success"
	acquireStatusFailed      = "failed"
	acquireStatusNotExecuted = "not_executed"
	enablementNotRequested   = "not_requested"
	enablementSuccess        = "success"
	enablementFailed         = "failed"
)

// PackageAcquireCommand orchestrates existing acquisition APIs.
// It is not a second resolver or process runner.
// Enablement is a separate transition: only an explicit --enable after
// successful acquisition reuses enablePackage. Default acquire does not enable.
type PackageAcquireCommand struct {
	app            *kernel.Application
	out            io.Writer
	index          *registry.Index
	root           string
	dryRunTargets  func(root string, args []string) ([]acquire.DryRunReport, error)
	executeTargets func(ctx context.Context, root string, args []string) (acquire.ApplyResult, error)
	inspect        func(root string) (acquire.Inspection, error)
	snapshotFiles  func(root string) (acquire.FileSnapshot, error)
	recoverFiles   func(ctx context.Context, snap acquire.FileSnapshot) (acquire.Recovery, error)
	enableFn       func(name string) (bool, error)
	ctx            context.Context
}

func (c *PackageAcquireCommand) Name() string { return "package:acquire" }
func (c *PackageAcquireCommand) Description() string {
	return "Acquire a package module via existing acquire APIs (does not enable unless --enable)"
}
func (c *PackageAcquireCommand) writer() io.Writer {
	if c.out != nil {
		return c.out
	}
	return os.Stdout
}

func (c *PackageAcquireCommand) Handle(args []string) error {
	if hasFlag(args, "--help", "-h") {
		fmt.Fprintln(c.writer(), "Usage: package:acquire <name>[@version] [version] [--dry-run] [--enable] [--root=] [--framework=] [--timeout=] [--no-recover] [--format=json|text]")
		return nil
	}
	format, err := formatFromArgs(args)
	if err != nil {
		return cliErr(ExitUsage, err)
	}
	pos := positionalArgs(args)
	if len(pos) < 1 {
		return cliErr(ExitUsage, fmt.Errorf("usage: package:acquire <name>[@version] [version]"))
	}
	name, ver := splitNameVersion(pos[0])
	if len(pos) > 1 {
		if ver != "" && pos[1] != ver {
			return cliErr(ExitUsage, fmt.Errorf("conflicting version %q and %q", ver, pos[1]))
		}
		ver = pos[1]
	}
	if opt := optionValue(args, "--version"); opt != "" {
		if ver != "" && opt != ver {
			return cliErr(ExitUsage, fmt.Errorf("conflicting version %q and %q", ver, opt))
		}
		ver = opt
	}
	q := registry.Query{
		Name:      name,
		Version:   ver,
		Kind:      optionValue(args, "--kind"),
		Framework: optionValue(args, "--framework"),
	}
	if q.Framework == "" && c.app != nil {
		q.Framework = c.app.Version()
	}
	idx, err := c.loadIndex()
	if err != nil {
		return cliFailed(ExitResolution, "package:acquire", name, err, "the registry index could not be loaded from the CLI catalog")
	}
	got, err := idx.Resolve(q)
	if err != nil {
		return cliFailed(ExitResolution, "package:acquire", name, err, "run package:info "+name+" or package:resolve "+name)
	}
	plan, err := acquire.FromResult(got)
	if err != nil {
		return cliFailed(ExitPlanning, "package:acquire", name, err, "the selected release could not be turned into a go get argument")
	}
	getArgs, err := acquire.Targets([]acquire.Plan{plan})
	if err != nil {
		return cliFailed(ExitPlanning, "package:acquire", name, err, "the acquisition plan has no go get targets")
	}
	root, err := c.resolveRoot(args)
	if err != nil {
		return cliFailed(ExitUsage, "package:acquire", name, err, "pass --root to a module directory that contains go.mod")
	}
	view := acquireCLIView{
		Root:        root,
		GoGetArgs:   getArgs,
		Enabled:     false,
		Acquisition: acquireStatusNotExecuted,
		Enablement:  enablementNotRequested,
	}
	if hasFlag(args, "--dry-run") {
		reps, err := c.runDryRun(root, getArgs)
		if err != nil {
			return cliFailed(ExitPlanning, "package:acquire", name, err, "dry-run could not inspect the planned go get; check --root and go.mod")
		}
		view.Mode = "dry-run"
		view.DryRun = make([]dryRunCLIView, 0, len(reps))
		for _, rep := range reps {
			view.DryRun = append(view.DryRun, dryRunCLIView{
				Module:   rep.Module,
				Selected: rep.Selected,
				GoGetArg: rep.GoGetArg,
				Command:  append([]string(nil), rep.Command...),
			})
		}
		return c.writeView(format, view)
	}

	view.Mode = "execute"
	ctx, cancel, err := c.commandContext(args)
	if err != nil {
		return err
	}
	defer cancel()
	var snap acquire.FileSnapshot
	var snapErr error
	if !hasFlag(args, "--no-recover") {
		snap, snapErr = c.runSnapshot(root)
		if snapErr != nil {
			view.SnapshotError = snapErr.Error()
			view.appendError(snapErr)
		}
	} else {
		snapErr = errSkipRecover
	}
	result, execErr := c.runExecute(ctx, root, getArgs)
	if execErr != nil && snapErr == nil {
		rec, recErr := c.runRecover(ctx, snap)
		result = result.WithRecovery(rec)
		if recErr != nil {
			view.RecoveryError = recErr.Error()
			view.appendError(recErr)
		}
	}
	view.Successful = result.Successful()
	view.Failed = result.Failed()
	view.Unattempted = result.Unattempted()
	view.Targets = targetViews(result)
	view.Recovery = string(result.Recovery.Kind)
	if view.Recovery == "" {
		view.Recovery = string(acquire.RecoveryUnavailable)
	}
	view.Acquisition = acquireStatusFailed
	if execErr == nil && len(view.Failed) == 0 {
		view.Acquisition = acquireStatusSuccess
	}
	view.Enablement = enablementNotRequested
	view.Enabled = false
	if hasFlag(args, "--enable") && view.Acquisition == acquireStatusSuccess {
		if err := c.runEnable(name); err != nil {
			view.Enablement = enablementFailed
			view.appendError(err)
			c.attachInspect(root, &view)
			if werr := c.writeView(format, view); werr != nil {
				return werr
			}
			return cliFailed(ExitEnablement, "package:acquire --enable", name, err, "acquisition succeeded; enablement is separate — fix Requires/import then package:enable "+name+" (acquisition is not rolled back)")
		}
		view.Enablement = enablementSuccess
		view.Enabled = true
	}
	c.attachInspect(root, &view)
	if err := c.writeView(format, view); err != nil {
		return err
	}
	if execErr == nil {
		return nil
	}
	wrapped := fmt.Errorf("package:acquire %q failed: %w", name, execErr)
	if cerr := classifyContextError(wrapped); cerr != nil {
		return cliFailed(ExitCanceled, "package:acquire", name, execErr, "the command was canceled or timed out; retry, or raise --timeout")
	}
	return cliFailed(ExitAcquisition, "package:acquire", name, execErr, "inspect go.mod / go.sum, retry, or run package:acquire --dry-run")
}

var errSkipRecover = fmt.Errorf("acquire-cli: skip recover")
