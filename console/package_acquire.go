package console

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

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
		fmt.Fprintln(c.writer(), "Usage: package:acquire <name>[@version] [version] [--dry-run] [--enable] [--root=] [--framework=] [--no-recover] [--format=json|text]")
		return nil
	}
	format, err := formatFromArgs(args)
	if err != nil {
		return err
	}
	pos := positionalArgs(args)
	if len(pos) < 1 {
		return fmt.Errorf("usage: package:acquire <name>[@version] [version]")
	}
	name, ver := splitNameVersion(pos[0])
	if len(pos) > 1 {
		if ver != "" && pos[1] != ver {
			return fmt.Errorf("conflicting version %q and %q", ver, pos[1])
		}
		ver = pos[1]
	}
	if opt := optionValue(args, "--version"); opt != "" {
		if ver != "" && opt != ver {
			return fmt.Errorf("conflicting version %q and %q", ver, opt)
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
		return err
	}
	got, err := idx.Resolve(q)
	if err != nil {
		return err
	}
	plan, err := acquire.FromResult(got)
	if err != nil {
		return err
	}
	getArgs, err := acquire.Targets([]acquire.Plan{plan})
	if err != nil {
		return err
	}
	root, err := c.resolveRoot(args)
	if err != nil {
		return err
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
			return err
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
	ctx := context.Background()
	var snap acquire.FileSnapshot
	var snapErr error
	if !hasFlag(args, "--no-recover") {
		snap, snapErr = c.runSnapshot(root)
	} else {
		snapErr = errSkipRecover
	}
	result, execErr := c.runExecute(ctx, root, getArgs)
	if execErr != nil && snapErr == nil {
		rec, _ := c.runRecover(ctx, snap)
		result = result.WithRecovery(rec)
	}
	view.Successful = result.Successful()
	view.Failed = result.Failed()
	view.Unattempted = result.Unattempted()
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
			if _, ierr := c.runInspect(root); ierr != nil {
				view.InspectErr = ierr.Error()
			}
			if werr := c.writeView(format, view); werr != nil {
				return werr
			}
			return err
		}
		view.Enablement = enablementSuccess
		view.Enabled = true
	}
	if _, err := c.runInspect(root); err != nil {
		view.InspectErr = err.Error()
	}
	if err := c.writeView(format, view); err != nil {
		return err
	}
	return execErr
}

var errSkipRecover = fmt.Errorf("acquire-cli: skip recover")

func (c *PackageAcquireCommand) writeView(format string, view acquireCLIView) error {
	if format == "json" {
		return writeJSON(c.writer(), view)
	}
	w := c.writer()
	fmt.Fprintf(w, "mode: %s\n", view.Mode)
	fmt.Fprintf(w, "acquisition: %s\n", view.Acquisition)
	fmt.Fprintf(w, "enablement: %s\n", view.Enablement)
	fmt.Fprintf(w, "enabled: %t\n", view.Enabled)
	fmt.Fprintf(w, "root: %s\n", view.Root)
	for _, arg := range view.GoGetArgs {
		fmt.Fprintf(w, "go_get_arg: %s\n", arg)
	}
	if view.Mode == "dry-run" {
		for _, rep := range view.DryRun {
			fmt.Fprintf(w, "module: %s\n", rep.Module)
			fmt.Fprintf(w, "selected: %s\n", rep.Selected)
			fmt.Fprintf(w, "command: %s\n", strings.Join(rep.Command, " "))
		}
		return nil
	}
	for _, arg := range view.Successful {
		fmt.Fprintf(w, "successful: %s\n", arg)
	}
	for _, arg := range view.Failed {
		fmt.Fprintf(w, "failed: %s\n", arg)
	}
	for _, arg := range view.Unattempted {
		fmt.Fprintf(w, "unattempted: %s\n", arg)
	}
	fmt.Fprintf(w, "recovery: %s\n", view.Recovery)
	return nil
}

func (c *PackageAcquireCommand) resolveRoot(args []string) (string, error) {
	if v := optionValue(args, "--root"); v != "" {
		return v, nil
	}
	if strings.TrimSpace(c.root) != "" {
		return c.root, nil
	}
	if c.app != nil {
		if p := strings.TrimSpace(c.app.BasePath()); p != "" {
			return p, nil
		}
	}
	return os.Getwd()
}

func (c *PackageAcquireCommand) loadIndex() (registry.Index, error) {
	if c.index != nil {
		return *c.index, nil
	}
	return catalogRegistryIndex()
}

func (c *PackageAcquireCommand) runDryRun(root string, args []string) ([]acquire.DryRunReport, error) {
	if c.dryRunTargets != nil {
		return c.dryRunTargets(root, args)
	}
	return acquire.DryRunTargets(root, args)
}

func (c *PackageAcquireCommand) runExecute(ctx context.Context, root string, args []string) (acquire.ApplyResult, error) {
	if c.executeTargets != nil {
		return c.executeTargets(ctx, root, args)
	}
	return acquire.ExecuteTargets(ctx, root, args)
}

func (c *PackageAcquireCommand) runInspect(root string) (acquire.Inspection, error) {
	if c.inspect != nil {
		return c.inspect(root)
	}
	return acquire.Inspect(root)
}

func (c *PackageAcquireCommand) runSnapshot(root string) (acquire.FileSnapshot, error) {
	if c.snapshotFiles != nil {
		return c.snapshotFiles(root)
	}
	return acquire.SnapshotFiles(root)
}

func (c *PackageAcquireCommand) runRecover(ctx context.Context, snap acquire.FileSnapshot) (acquire.Recovery, error) {
	if c.recoverFiles != nil {
		return c.recoverFiles(ctx, snap)
	}
	return acquire.RecoverFiles(ctx, snap)
}

func (c *PackageAcquireCommand) runEnable(name string) error {
	if c.enableFn != nil {
		_, err := c.enableFn(name)
		return err
	}
	_, err := enablePackage(c.app, name)
	if err != nil {
		return err
	}
	if err := wireEnabledAddon(c.app, name); err != nil {
		return err
	}
	_ = applyPackageEnvList(c.app, []string{name})
	return nil
}

type acquireCLIView struct {
	Mode        string          `json:"mode"`
	Root        string          `json:"root"`
	GoGetArgs   []string        `json:"go_get_args"`
	Successful  []string        `json:"successful,omitempty"`
	Failed      []string        `json:"failed,omitempty"`
	Unattempted []string        `json:"unattempted,omitempty"`
	Recovery    string          `json:"recovery,omitempty"`
	Acquisition string          `json:"acquisition"`
	Enablement  string          `json:"enablement"`
	Enabled     bool            `json:"enabled"`
	DryRun      []dryRunCLIView `json:"dry_run,omitempty"`
	InspectErr  string          `json:"inspect_error,omitempty"`
}

type dryRunCLIView struct {
	Module   string   `json:"module"`
	Selected string   `json:"selected"`
	GoGetArg string   `json:"go_get_arg"`
	Command  []string `json:"command"`
}
