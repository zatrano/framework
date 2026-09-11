package console

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/zatrano/framework/v2/kernel"
)

func registerDoctorCommand(console *Application, app *kernel.Application) {
	_ = app
	console.Register(&DoctorCommand{})
}

// DoctorCommand reports architecture convention warnings (zatrano doctor).
type DoctorCommand struct {
	out io.Writer
}

func (c *DoctorCommand) Name() string { return "doctor" }
func (c *DoctorCommand) Description() string {
	return "Check a ZATRANO app for STANDARD architecture drift (errors fail CI)"
}

func (c *DoctorCommand) writer() io.Writer {
	if c.out != nil {
		return c.out
	}
	return os.Stdout
}

func (c *DoctorCommand) Handle(args []string) error {
	if hasFlag(args, "--help", "-h") {
		fmt.Fprintln(c.writer(), "Usage: zatrano doctor [path] [--json] [--strict]")
		fmt.Fprintln(c.writer(), "Architecture errors exit 1. Warnings do not, unless --strict.")
		fmt.Fprintln(c.writer(), "No --fix. See https://zatrano.com/docs/application-engineering/rules")
		return nil
	}
	root, err := os.Getwd()
	if err != nil {
		return err
	}
	for _, a := range args {
		if strings.HasPrefix(a, "-") {
			continue
		}
		root = a
		break
	}
	findings, err := RunDoctor(root)
	if err != nil {
		return err
	}
	if hasFlag(args, "--json") {
		enc := json.NewEncoder(c.writer())
		enc.SetIndent("", "  ")
		if err := enc.Encode(doctorReport(root, findings)); err != nil {
			return err
		}
	} else {
		fmt.Fprint(c.writer(), FormatDoctorText(root, findings))
	}
	nErr, nWarn := countDoctorSeverities(findings)
	if nErr > 0 {
		return cliErr(ExitGeneral, fmt.Errorf("zatrano doctor found %d error(s)\nNext: fix ERROR findings (rule IDs in output), then rerun zatrano doctor", nErr))
	}
	if hasFlag(args, "--strict") && nWarn > 0 {
		return cliErr(ExitGeneral, fmt.Errorf("zatrano doctor --strict: %d warning(s) treated as errors", nWarn))
	}
	return nil
}

// Finding is one architecture finding. Checks are separate functions so a later Fix() can attach.
type Finding struct {
	Rule     string `json:"rule"`
	Check    string `json:"check"`
	Severity string `json:"severity"`
	File     string `json:"file"`
	Line     int    `json:"line"`
	Found    string `json:"found"`
	Why      string `json:"why"`
	How      string `json:"how"`
	See      string `json:"see,omitempty"`
}

func (f Finding) id() string {
	if f.Rule != "" {
		return f.Rule
	}
	return f.Check
}

// DoctorCheck is one inspectable rule (future: add Fix).
type DoctorCheck struct {
	Name string
	Run  func(root string) ([]Finding, error)
}

// DoctorChecks returns architecture checks in stable order.
func DoctorChecks() []DoctorCheck {
	return []DoctorCheck{
		{Name: "routes", Run: checkRouteLocation},
		{Name: "concrete", Run: checkConcreteLeak},
		{Name: "layout", Run: checkAppLayout},
		{Name: "providers", Run: checkProviders},
		{Name: "layers", Run: checkForbiddenLayers},
		{Name: "controllers", Run: checkControllers},
		{Name: "requests", Run: checkRequests},
		{Name: "orm", Run: checkORMArchitecture},
		{Name: "validation", Run: checkValidationArchitecture},
	}
}

// RunDoctor runs every check against a consumer app root.
func RunDoctor(root string) ([]Finding, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	appDir := filepath.Join(abs, "app")
	st, err := os.Stat(appDir)
	if err != nil || !st.IsDir() {
		return []Finding{{
			Check:    "layout",
			Severity: "warning",
			File:     "app",
			Found:    "no app/ directory at " + abs,
			Why:      "zatrano doctor inspects consumer applications created by zatrano new.",
			How:      "Run this command from a project that has an app/ folder, or pass that path as the argument.",
		}}, nil
	}
	var all []Finding
	for _, check := range DoctorChecks() {
		found, err := check.Run(abs)
		if err != nil {
			return nil, err
		}
		all = append(all, found...)
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].Check != all[j].Check {
			return all[i].Check < all[j].Check
		}
		if all[i].File != all[j].File {
			return all[i].File < all[j].File
		}
		return all[i].Line < all[j].Line
	})
	return all, nil
}

// FormatDoctorText renders findings with found / why / how.
func FormatDoctorText(root string, findings []Finding) string {
	var b strings.Builder
	nErr, nWarn := countDoctorSeverities(findings)
	status := "PASS"
	if nErr > 0 {
		status = "FAIL"
	}
	fmt.Fprintf(&b, "zatrano doctor\nroot: %s\nstatus: %s\nerrors: %d\nwarnings: %d\n", root, status, nErr, nWarn)
	if len(findings) == 0 {
		return b.String()
	}
	b.WriteString("\n")
	for _, f := range findings {
		loc := f.File
		if f.Line > 0 {
			loc = fmt.Sprintf("%s:%d", f.File, f.Line)
		}
		fmt.Fprintf(&b, "[%s] %s  %s  %s\n", f.id(), f.Severity, f.Check, loc)
		fmt.Fprintf(&b, "  found: %s\n", f.Found)
		fmt.Fprintf(&b, "  why:   %s\n", f.Why)
		fmt.Fprintf(&b, "  how:   %s\n", f.How)
		if f.See != "" {
			fmt.Fprintf(&b, "  see:   %s\n", f.See)
		}
		b.WriteString("\n")
	}
	return b.String()
}

type doctorReportPayload struct {
	Root     string    `json:"root"`
	Status   string    `json:"status"`
	Errors   int       `json:"errors"`
	Warnings int       `json:"warnings"`
	Findings []Finding `json:"findings"`
}

func doctorReport(root string, findings []Finding) doctorReportPayload {
	nErr, nWarn := countDoctorSeverities(findings)
	status := "PASS"
	if nErr > 0 {
		status = "FAIL"
	}
	if findings == nil {
		findings = []Finding{}
	}
	return doctorReportPayload{Root: root, Status: status, Errors: nErr, Warnings: nWarn, Findings: findings}
}

func countDoctorSeverities(findings []Finding) (errors, warnings int) {
	for _, f := range findings {
		switch f.Severity {
		case "error":
			errors++
		default:
			warnings++
		}
	}
	return errors, warnings
}
