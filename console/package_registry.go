// Registry CLI commands consume Index.Search / Lookup / Resolve.
// They do not select versions themselves and do not mutate go.mod.
// Phase 6 freeze: do not grow these into an installer; do not copy
// compareSemver / latestCompatible into this package.
package console

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/zatrano/framework/v2/distribution/manifest"
	"github.com/zatrano/framework/v2/distribution/registry"
	"github.com/zatrano/framework/v2/kernel"
)

func registerPackageRegistryCommands(console *Application, app *kernel.Application) {
	console.Register(
		&PackageSearchCommand{},
		&PackageInfoCommand{},
		&PackageResolveCommand{app: app},
		&PackageAcquireCommand{app: app},
	)
}

// catalogRegistryIndex is the official in-memory index from the CLI catalog.
// Resolution stays in package registry; this only supplies documents.
// Invariant: these commands must call Index.Search / Lookup / Resolve and
// must not copy version-selection (semver compare, latestCompatible, main fallback).
func catalogRegistryIndex() (registry.Index, error) {
	docs := make([]manifest.Document, 0, len(ecosystemCatalog))
	for _, p := range ecosystemCatalog {
		docs = append(docs, manifest.Derive(manifest.Input{
			Name:        p.Name,
			Kind:        string(p.EffectiveKind()),
			Layer:       string(p.Layer),
			Description: p.Description,
			Heavy:       p.Heavy,
		}))
	}
	return registry.FromDocuments(docs)
}

// PackageSearchCommand discovers package identities. It does not pick a version.
type PackageSearchCommand struct {
	out   io.Writer
	index *registry.Index
}

func (c *PackageSearchCommand) Name() string { return "package:search" }
func (c *PackageSearchCommand) Description() string {
	return "Discover packages in the registry index (does not select a version)"
}
func (c *PackageSearchCommand) writer() io.Writer {
	if c.out != nil {
		return c.out
	}
	return os.Stdout
}

func (c *PackageSearchCommand) Handle(args []string) error {
	if hasFlag(args, "--help", "-h") {
		fmt.Fprintln(c.writer(), "Usage: package:search [query] [--kind=] [--layer=] [--heavy] [--format=json|text]")
		return nil
	}
	format, err := formatFromArgs(args)
	if err != nil {
		return err
	}
	filter := registry.Filter{
		Query: firstPositional(args),
		Kind:  optionValue(args, "--kind"),
		Layer: optionValue(args, "--layer"),
	}
	if hasFlag(args, "--heavy") {
		heavy := true
		filter.Heavy = &heavy
	}
	idx, err := c.loadIndex()
	if err != nil {
		return err
	}
	hits := idx.Search(filter)
	if format == "json" {
		views := make([]searchHit, 0, len(hits))
		for _, p := range hits {
			views = append(views, searchHitFrom(p))
		}
		return writeJSON(c.writer(), views)
	}
	w := tabwriter.NewWriter(c.writer(), 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tKIND\tLAYER\tHEAVY\tMODULE\tDESCRIPTION")
	for _, p := range hits {
		heavy := ""
		if p.Heavy {
			heavy = "yes"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n", p.Name, p.Kind, p.Layer, heavy, p.Module, p.Description)
	}
	return w.Flush()
}

func (c *PackageSearchCommand) loadIndex() (registry.Index, error) {
	if c.index != nil {
		return *c.index, nil
	}
	return catalogRegistryIndex()
}

// PackageInfoCommand prints one package identity and its known releases.
// It does not select a version (use package:resolve).
type PackageInfoCommand struct {
	out   io.Writer
	index *registry.Index
}

func (c *PackageInfoCommand) Name() string { return "package:info" }
func (c *PackageInfoCommand) Description() string {
	return "Show registry identity and known releases for a package (no version selection)"
}
func (c *PackageInfoCommand) writer() io.Writer {
	if c.out != nil {
		return c.out
	}
	return os.Stdout
}

func (c *PackageInfoCommand) Handle(args []string) error {
	if hasFlag(args, "--help", "-h") {
		fmt.Fprintln(c.writer(), "Usage: package:info <name> [--format=json|text]")
		return nil
	}
	format, err := formatFromArgs(args)
	if err != nil {
		return err
	}
	name := firstPositional(args)
	if name == "" {
		return fmt.Errorf("usage: package:info <name>")
	}
	idx, err := c.loadIndex()
	if err != nil {
		return err
	}
	p, ok := idx.Lookup(name)
	if !ok {
		return fmt.Errorf("unknown package %q", name)
	}
	if format == "json" {
		return writeJSON(c.writer(), p)
	}
	fmt.Fprintf(c.writer(), "name: %s\n", p.Name)
	fmt.Fprintf(c.writer(), "import: %s\n", p.Import)
	fmt.Fprintf(c.writer(), "module: %s\n", p.Module)
	fmt.Fprintf(c.writer(), "kind: %s\n", p.Kind)
	fmt.Fprintf(c.writer(), "layer: %s\n", p.Layer)
	fmt.Fprintf(c.writer(), "heavy: %t\n", p.Heavy)
	fmt.Fprintf(c.writer(), "description: %s\n", p.Description)
	fmt.Fprintln(c.writer(), "releases:")
	for _, r := range p.Releases {
		fmt.Fprintf(c.writer(), "  - %s\n", formatRelease(r))
	}
	return nil
}

func (c *PackageInfoCommand) loadIndex() (registry.Index, error) {
	if c.index != nil {
		return *c.index, nil
	}
	return catalogRegistryIndex()
}

// PackageResolveCommand selects one compatible release via registry.Resolve.
// It does not enable the package or write go.mod.
type PackageResolveCommand struct {
	app   *kernel.Application
	out   io.Writer
	index *registry.Index
}

func (c *PackageResolveCommand) Name() string { return "package:resolve" }
func (c *PackageResolveCommand) Description() string {
	return "Select a compatible package release (does not install or enable)"
}
func (c *PackageResolveCommand) writer() io.Writer {
	if c.out != nil {
		return c.out
	}
	return os.Stdout
}

func (c *PackageResolveCommand) Handle(args []string) error {
	if hasFlag(args, "--help", "-h") {
		fmt.Fprintln(c.writer(), "Usage: package:resolve <name>[@version] [version] [--framework=] [--kind=] [--format=json|text]")
		return nil
	}
	format, err := formatFromArgs(args)
	if err != nil {
		return err
	}
	pos := positionalArgs(args)
	if len(pos) < 1 {
		return fmt.Errorf("usage: package:resolve <name>[@version] [version]")
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
	view := resolveView{
		Name:     got.Package.Name,
		Import:   got.Package.Import,
		Module:   got.Package.Module,
		Kind:     got.Package.Kind,
		Layer:    got.Package.Layer,
		Heavy:    got.Package.Heavy,
		Selected: formatRelease(got.Release),
		Release:  got.Release,
	}
	if format == "json" {
		return writeJSON(c.writer(), view)
	}
	fmt.Fprintf(c.writer(), "name: %s\n", view.Name)
	fmt.Fprintf(c.writer(), "selected: %s\n", view.Selected)
	fmt.Fprintf(c.writer(), "module: %s\n", view.Module)
	fmt.Fprintf(c.writer(), "import: %s\n", view.Import)
	fmt.Fprintf(c.writer(), "kind: %s\n", view.Kind)
	return nil
}

func (c *PackageResolveCommand) loadIndex() (registry.Index, error) {
	if c.index != nil {
		return *c.index, nil
	}
	return catalogRegistryIndex()
}

type searchHit struct {
	Name        string `json:"name"`
	Import      string `json:"import"`
	Module      string `json:"module"`
	Kind        string `json:"kind"`
	Layer       string `json:"layer"`
	Heavy       bool   `json:"heavy,omitempty"`
	Description string `json:"description"`
}

type resolveView struct {
	Name     string           `json:"name"`
	Import   string           `json:"import"`
	Module   string           `json:"module"`
	Kind     string           `json:"kind"`
	Layer    string           `json:"layer"`
	Heavy    bool             `json:"heavy,omitempty"`
	Selected string           `json:"selected"`
	Release  registry.Release `json:"release"`
}

func searchHitFrom(p registry.Package) searchHit {
	return searchHit{
		Name:        p.Name,
		Import:      p.Import,
		Module:      p.Module,
		Kind:        p.Kind,
		Layer:       p.Layer,
		Heavy:       p.Heavy,
		Description: p.Description,
	}
}

func formatRelease(r registry.Release) string {
	if v := strings.TrimSpace(r.Version); v != "" {
		return v
	}
	if ch := strings.TrimSpace(r.Channel); ch != "" {
		return ch
	}
	return "-"
}

func writeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func optionValue(args []string, name string) string {
	prefix := name + "="
	for i, a := range args {
		if a == name && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
			return args[i+1]
		}
		if strings.HasPrefix(a, prefix) {
			return strings.TrimPrefix(a, prefix)
		}
	}
	return ""
}

func firstPositional(args []string) string {
	pos := positionalArgs(args)
	if len(pos) == 0 {
		return ""
	}
	return pos[0]
}

func positionalArgs(args []string) []string {
	out := make([]string, 0)
	skipNext := false
	for i, a := range args {
		if skipNext {
			skipNext = false
			continue
		}
		switch {
		case a == "--help", a == "-h", a == "--heavy", a == "--json", a == "--dry-run", a == "--no-recover":
			continue
		case strings.HasPrefix(a, "--format="), strings.HasPrefix(a, "--kind="),
			strings.HasPrefix(a, "--layer="), strings.HasPrefix(a, "--framework="),
			strings.HasPrefix(a, "--version="), strings.HasPrefix(a, "--root="):
			continue
		case a == "--format", a == "--kind", a == "--layer", a == "--framework", a == "--version", a == "--root":
			if i+1 < len(args) {
				skipNext = true
			}
			continue
		case strings.HasPrefix(a, "-"):
			continue
		default:
			out = append(out, a)
		}
	}
	return out
}

func splitNameVersion(s string) (name, version string) {
	s = strings.TrimSpace(s)
	if i := strings.Index(s, "@"); i >= 0 {
		return s[:i], s[i+1:]
	}
	return s, ""
}
