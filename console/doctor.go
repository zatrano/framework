package console

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/zatrano/framework/kernel"
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
	return "Check a ZATRANO app for routing, contract, layout, and provider convention drift"
}

func (c *DoctorCommand) writer() io.Writer {
	if c.out != nil {
		return c.out
	}
	return os.Stdout
}

func (c *DoctorCommand) Handle(args []string) error {
	if hasFlag(args, "--help", "-h") {
		fmt.Fprintln(c.writer(), "Usage: zatrano doctor [path]")
		fmt.Fprintln(c.writer(), "Reports warnings only; no --fix (checks are Fix-shaped for a later pass).")
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
	fmt.Fprint(c.writer(), FormatDoctorText(root, findings))
	return nil
}

// Finding is one architecture warning. Checks are separate functions so a later Fix() can attach.
type Finding struct {
	Check    string `json:"check"`
	Severity string `json:"severity"`
	File     string `json:"file"`
	Line     int    `json:"line"`
	Found    string `json:"found"`
	Why      string `json:"why"`
	How      string `json:"how"`
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
	fmt.Fprintf(&b, "zatrano doctor\nroot: %s\n", root)
	if len(findings) == 0 {
		b.WriteString("findings: 0\n")
		return b.String()
	}
	fmt.Fprintf(&b, "findings: %d (warnings)\n\n", len(findings))
	for _, f := range findings {
		loc := f.File
		if f.Line > 0 {
			loc = fmt.Sprintf("%s:%d", f.File, f.Line)
		}
		fmt.Fprintf(&b, "[%s] %s  %s\n", f.Check, f.Severity, loc)
		fmt.Fprintf(&b, "  found: %s\n", f.Found)
		fmt.Fprintf(&b, "  why:   %s\n", f.Why)
		fmt.Fprintf(&b, "  how:   %s\n\n", f.How)
	}
	return b.String()
}
