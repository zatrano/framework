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

func registerDescribeCommand(console *Application, app *kernel.Application) {
	ver := "2.0.0"
	if app != nil {
		if v := app.Version(); v != "" {
			ver = v
		}
	}
	console.Register(&DescribeCommand{version: ver})
}

// DescribeCommand prints the framework surface derived from source (zatrano describe).
type DescribeCommand struct {
	out     io.Writer
	version string
}

func (c *DescribeCommand) Name() string { return "describe" }
func (c *DescribeCommand) Description() string {
	return "Print the framework machine-readable surface (contracts, catalog, routing, providers)"
}

func (c *DescribeCommand) writer() io.Writer {
	if c.out != nil {
		return c.out
	}
	return os.Stdout
}

func (c *DescribeCommand) Handle(args []string) error {
	if hasFlag(args, "--help", "-h") {
		fmt.Fprintln(c.writer(), "Usage: zatrano describe [--format=json|text]")
		return nil
	}
	format, err := formatFromArgs(args)
	if err != nil {
		return err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	doc, err := BuildDescribeDocument(cwd)
	if err != nil {
		return err
	}
	if format == "json" {
		enc := json.NewEncoder(c.writer())
		enc.SetEscapeHTML(false)
		enc.SetIndent("", "  ")
		return enc.Encode(doc)
	}
	fmt.Fprint(c.writer(), FormatDescribeText(doc))
	return nil
}

// DescribeDocument is the machine-readable framework surface.
type DescribeDocument struct {
	Version   string                  `json:"version"`
	Contracts map[string]ContractType `json:"contracts"`
	Catalog   CatalogReport           `json:"catalog"`
	Routing   RoutingReport           `json:"routing"`
	Providers ProvidersReport         `json:"providers"`
}

// ContractType is one interface parsed from source.
type ContractType struct {
	Name    string           `json:"name"`
	File    string           `json:"file"`
	Methods []ContractMethod `json:"methods"`
}

// ContractMethod is a method or function signature.
type ContractMethod struct {
	Name      string `json:"name"`
	Signature string `json:"signature"`
}

// CatalogReport groups catalog packages by layer.
type CatalogReport struct {
	Layers []CatalogLayerReport `json:"layers"`
}

// CatalogLayerReport is one kernel catalog layer.
type CatalogLayerReport struct {
	Constant string                 `json:"constant"`
	Name     string                 `json:"name"`
	Role     string                 `json:"role"`
	Packages []CatalogPackageReport `json:"packages"`
}

// CatalogPackageReport is one catalog entry (kernel primitives plus CLI ecosystem).
type CatalogPackageReport struct {
	Name        string `json:"name"`
	Kind        string `json:"kind"`
	Heavy       bool   `json:"heavy"`
	Description string `json:"description"`
}

// RoutingReport holds discovery primitives and optional in-app sample routes.
type RoutingReport struct {
	Primitives   []ContractMethod `json:"primitives"`
	SampleRoutes []SampleRoute    `json:"sample_routes"`
}

// SampleRoute is a route-registration call found under app/routes.
type SampleRoute struct {
	File  string `json:"file"`
	Line  int    `json:"line"`
	Group string `json:"group,omitempty"`
	Call  string `json:"call"`
	Path  string `json:"path,omitempty"`
}

// ProvidersReport describes kernel.Provider and addon self-registration.
type ProvidersReport struct {
	Interface        ContractType         `json:"interface"`
	SelfRegistration SelfRegistrationInfo `json:"self_registration"`
}

// SelfRegistrationInfo is structured addon registration metadata (not prose).
type SelfRegistrationInfo struct {
	Package               string   `json:"package"`
	Register              string   `json:"register"`
	Select                string   `json:"select"`
	Lookup                string   `json:"lookup"`
	Available             string   `json:"available"`
	MetaType              string   `json:"meta_type"`
	MetaFields            []string `json:"meta_fields"`
	FactoryField          string   `json:"factory_field"`
	FactoryReturns        string   `json:"factory_returns"`
	RegisterCalledFrom    string   `json:"register_called_from"`
	RegistryImportsAddons bool     `json:"registry_imports_addons"`
	ConsumerBlankImport   bool     `json:"consumer_blank_import"`
}

// BuildDescribeDocument derives the describe document from framework source.
// scanRoot is searched for sample routes (typically the consumer app cwd).
func BuildDescribeDocument(scanRoot string) (*DescribeDocument, error) {
	root, err := frameworkModuleRoot()
	if err != nil {
		return nil, err
	}
	modPath, err := modulePath(root)
	if err != nil {
		return nil, err
	}
	contracts, err := parseContractInterfaces(filepath.Join(root, "contracts"))
	if err != nil {
		return nil, err
	}
	layers, err := parseCatalogLayers(filepath.Join(root, "kernel", "catalog.go"))
	if err != nil {
		return nil, err
	}
	catalog := assembleCatalog(layers)
	primitives, err := parseNamedFuncs(filepath.Join(root, "kernel", "routing", "discovery.go"), []string{
		"RegisterWeb", "RegisterAPI", "ApplyWeb", "ApplyAPI",
	})
	if err != nil {
		return nil, err
	}
	samples, err := parseSampleRoutes(scanRoot)
	if err != nil {
		return nil, err
	}
	provider, err := parseNamedInterface(filepath.Join(root, "contracts"), "Provider")
	if err != nil {
		return nil, err
	}
	selfReg, err := parseSelfRegistration(root, modPath)
	if err != nil {
		return nil, err
	}
	return &DescribeDocument{
		Version:   "2.0.0",
		Contracts: contracts,
		Catalog:   CatalogReport{Layers: catalog},
		Routing: RoutingReport{
			Primitives:   primitives,
			SampleRoutes: samples,
		},
		Providers: ProvidersReport{
			Interface:        provider,
			SelfRegistration: selfReg,
		},
	}, nil
}

func assembleCatalog(layers []CatalogLayerReport) []CatalogLayerReport {
	byName := map[kernel.Layer]int{}
	for i := range layers {
		byName[kernel.Layer(layers[i].Name)] = i
	}
	for _, p := range catalogAll() {
		idx, ok := byName[p.Layer]
		if !ok {
			layers = append(layers, CatalogLayerReport{
				Constant: "",
				Name:     string(p.Layer),
			})
			byName[p.Layer] = len(layers) - 1
			idx = len(layers) - 1
		}
		kind := string(p.EffectiveKind())
		layers[idx].Packages = append(layers[idx].Packages, CatalogPackageReport{
			Name:        p.Name,
			Kind:        kind,
			Heavy:       p.Heavy,
			Description: p.Description,
		})
	}
	return layers
}

// FormatDescribeText renders a human-readable summary of doc.
func FormatDescribeText(doc *DescribeDocument) string {
	if doc == nil {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "ZATRANO describe %s\n", doc.Version)
	b.WriteString("\n== contracts ==\n")
	names := make([]string, 0, len(doc.Contracts))
	for name := range doc.Contracts {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		ct := doc.Contracts[name]
		fmt.Fprintf(&b, "%s  (%s)\n", ct.Name, ct.File)
		for _, m := range ct.Methods {
			fmt.Fprintf(&b, "  %s\n", m.Signature)
		}
	}
	b.WriteString("\n== catalog ==\n")
	for _, layer := range doc.Catalog.Layers {
		fmt.Fprintf(&b, "%s (%s)\n", layer.Constant, layer.Name)
		if layer.Role != "" {
			fmt.Fprintf(&b, "  role: %s\n", layer.Role)
		}
		for _, p := range layer.Packages {
			fmt.Fprintf(&b, "  - %s [%s] %s\n", p.Name, p.Kind, p.Description)
		}
	}
	b.WriteString("\n== routing ==\n")
	b.WriteString("primitives:\n")
	for _, m := range doc.Routing.Primitives {
		fmt.Fprintf(&b, "  %s\n", m.Signature)
	}
	b.WriteString("sample_routes:\n")
	if len(doc.Routing.SampleRoutes) == 0 {
		b.WriteString("  (none)\n")
	}
	for _, r := range doc.Routing.SampleRoutes {
		fmt.Fprintf(&b, "  %s:%d %s %s %s\n", r.File, r.Line, r.Group, r.Call, r.Path)
	}
	b.WriteString("\n== providers ==\n")
	fmt.Fprintf(&b, "%s  (%s)\n", doc.Providers.Interface.Name, doc.Providers.Interface.File)
	for _, m := range doc.Providers.Interface.Methods {
		fmt.Fprintf(&b, "  %s\n", m.Signature)
	}
	sr := doc.Providers.SelfRegistration
	b.WriteString("self_registration:\n")
	fmt.Fprintf(&b, "  package: %s\n", sr.Package)
	fmt.Fprintf(&b, "  Register: %s\n", sr.Register)
	fmt.Fprintf(&b, "  Select: %s\n", sr.Select)
	fmt.Fprintf(&b, "  Lookup: %s\n", sr.Lookup)
	fmt.Fprintf(&b, "  Available: %s\n", sr.Available)
	fmt.Fprintf(&b, "  Meta: %s\n", sr.MetaType)
	fmt.Fprintf(&b, "  Meta.fields: %s\n", strings.Join(sr.MetaFields, ", "))
	fmt.Fprintf(&b, "  Meta.%s: %s\n", sr.FactoryField, sr.FactoryReturns)
	fmt.Fprintf(&b, "  register_called_from: %s\n", sr.RegisterCalledFrom)
	fmt.Fprintf(&b, "  registry_imports_addons: %t\n", sr.RegistryImportsAddons)
	fmt.Fprintf(&b, "  consumer_blank_import: %t\n", sr.ConsumerBlankImport)
	return b.String()
}

func formatFromArgs(args []string) (string, error) {
	format := "text"
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--format" && i+1 < len(args):
			format = args[i+1]
			i++
		case strings.HasPrefix(a, "--format="):
			format = strings.TrimPrefix(a, "--format=")
		}
	}
	format = strings.ToLower(strings.TrimSpace(format))
	switch format {
	case "", "text", "pretty":
		return "text", nil
	case "json":
		return "json", nil
	default:
		return "", fmt.Errorf("unknown --format %q (want json or text)", format)
	}
}
