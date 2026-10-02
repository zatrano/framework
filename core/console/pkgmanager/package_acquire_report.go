package pkgmanager

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/zatrano/framework/v3/core/distribution/acquire"
	"github.com/zatrano/framework/v3/core/distribution/registry"
)

func (c *PackageAcquireCommand) writeReport(format string, report acquireCLIReport) error {
	if format == "json" {
		return writeJSON(c.writer(), report)
	}
	w := c.writer()
	fmt.Fprintf(w, "mode: %s\n", report.Mode)
	fmt.Fprintf(w, "acquisition: %s\n", report.Acquisition)
	fmt.Fprintf(w, "enablement: %s\n", report.Enablement)
	fmt.Fprintf(w, "enabled: %t\n", report.Enabled)
	fmt.Fprintf(w, "root: %s\n", report.Root)
	for _, arg := range report.GoGetArgs {
		fmt.Fprintf(w, "go_get_arg: %s\n", arg)
	}
	if report.Mode == "dry-run" {
		for _, rep := range report.DryRun {
			fmt.Fprintf(w, "module: %s\n", rep.Module)
			fmt.Fprintf(w, "selected: %s\n", rep.Selected)
			fmt.Fprintf(w, "command: %s\n", strings.Join(rep.Command, " "))
		}
		return nil
	}
	for _, arg := range report.Successful {
		fmt.Fprintf(w, "successful: %s\n", arg)
	}
	for _, arg := range report.Failed {
		fmt.Fprintf(w, "failed: %s\n", arg)
	}
	for _, arg := range report.Unattempted {
		fmt.Fprintf(w, "unattempted: %s\n", arg)
	}
	fmt.Fprintf(w, "recovery: %s\n", report.Recovery)
	if report.SnapshotError != "" {
		fmt.Fprintf(w, "snapshot_error: %s\n", report.SnapshotError)
	}
	if report.RecoveryError != "" {
		fmt.Fprintf(w, "recovery_error: %s\n", report.RecoveryError)
	}
	if report.Inspection != nil && report.Inspection.Module != "" {
		fmt.Fprintf(w, "inspect_module: %s\n", report.Inspection.Module)
		for _, req := range report.Inspection.Requirements {
			fmt.Fprintf(w, "inspect_require: %s %s\n", req.Path, req.Version)
		}
	}
	for _, msg := range report.Errors {
		fmt.Fprintf(w, "error: %s\n", msg)
	}
	return nil
}

func (c *PackageAcquireCommand) commandContext(args []string) (context.Context, context.CancelFunc, error) {
	parent := c.ctx
	if parent == nil {
		parent = context.Background()
	}
	raw := strings.TrimSpace(optionValue(args, "--timeout"))
	if raw == "" {
		return parent, func() {}, nil
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		return nil, nil, cliFailed(ExitUsage, "package:acquire", "", fmt.Errorf("invalid --timeout %q", raw), "pass a positive duration such as 30s or 2m")
	}
	ctx, cancel := context.WithTimeout(parent, d)
	return ctx, cancel, nil
}

func (c *PackageAcquireCommand) attachInspect(root string, report *acquireCLIReport) {
	in, err := c.runInspect(root)
	if err != nil {
		report.InspectErr = err.Error()
		report.appendError(err)
		return
	}
	report.Inspection = inspectionReport(in)
}

func inspectionReport(in acquire.Inspection) *inspectCLIReport {
	reqs := make([]inspectReqCLIReport, 0, len(in.Requirements))
	for _, r := range in.Requirements {
		reqs = append(reqs, inspectReqCLIReport{Path: r.Path, Version: r.Version, Indirect: r.Indirect})
	}
	sums := make([]inspectSumCLIReport, 0, len(in.Checksums))
	for _, s := range in.Checksums {
		sums = append(sums, inspectSumCLIReport{Module: s.Module, Version: s.Version, Hash: s.Hash})
	}
	return &inspectCLIReport{
		Module:       in.Module,
		Go:           in.Go,
		Requirements: reqs,
		Checksums:    sums,
		GoModMissing: in.GoModMissing,
		GoSumMissing: in.GoSumMissing,
	}
}

func targetReports(result acquire.ApplyResult) []targetCLIReport {
	if len(result.Reports) == 0 {
		return nil
	}
	out := make([]targetCLIReport, 0, len(result.Reports))
	for _, rep := range result.Reports {
		out = append(out, targetCLIReport{GoGetArg: rep.GoGetArg, Status: string(rep.Status)})
	}
	return out
}

func (v *acquireCLIReport) appendError(err error) {
	if v == nil || err == nil {
		return
	}
	msg := err.Error()
	if msg == "" {
		return
	}
	v.Errors = append(v.Errors, msg)
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
	if err := wireEnablement(c.app, name); err != nil {
		return err
	}
	names, _ := enableRequiresNames(name)
	if len(names) == 0 {
		names = []string{name}
	}
	_ = applyPackageEnvList(c.app, names)
	return nil
}

type acquireCLIReport struct {
	Mode          string            `json:"mode"`
	Root          string            `json:"root"`
	GoGetArgs     []string          `json:"go_get_args"`
	Successful    []string          `json:"successful,omitempty"`
	Failed        []string          `json:"failed,omitempty"`
	Unattempted   []string          `json:"unattempted,omitempty"`
	Targets       []targetCLIReport `json:"targets,omitempty"`
	Recovery      string            `json:"recovery,omitempty"`
	RecoveryError string            `json:"recovery_error,omitempty"`
	SnapshotError string            `json:"snapshot_error,omitempty"`
	Acquisition   string            `json:"acquisition"`
	Enablement    string            `json:"enablement"`
	Enabled       bool              `json:"enabled"`
	DryRun        []dryRunCLIReport `json:"dry_run,omitempty"`
	Inspection    *inspectCLIReport `json:"inspection,omitempty"`
	InspectErr    string            `json:"inspect_error,omitempty"`
	Errors        []string          `json:"errors,omitempty"`
}

type targetCLIReport struct {
	GoGetArg string `json:"go_get_arg"`
	Status   string `json:"status"`
}

type inspectCLIReport struct {
	Module       string                `json:"module,omitempty"`
	Go           string                `json:"go,omitempty"`
	Requirements []inspectReqCLIReport `json:"requirements,omitempty"`
	Checksums    []inspectSumCLIReport `json:"checksums,omitempty"`
	GoModMissing bool                  `json:"go_mod_missing,omitempty"`
	GoSumMissing bool                  `json:"go_sum_missing,omitempty"`
}

type inspectReqCLIReport struct {
	Path     string `json:"path"`
	Version  string `json:"version"`
	Indirect bool   `json:"indirect,omitempty"`
}

type inspectSumCLIReport struct {
	Module  string `json:"module"`
	Version string `json:"version"`
	Hash    string `json:"hash"`
}

type dryRunCLIReport struct {
	Module   string   `json:"module"`
	Selected string   `json:"selected"`
	GoGetArg string   `json:"go_get_arg"`
	Command  []string `json:"command"`
}
