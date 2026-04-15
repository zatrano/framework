package gen

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Markers include a leading tab so the region between start/end is empty (gofmt-safe).
const (
	wireImportsStart = "\t// zatrano:wire:imports:start\n"
	wireImportsEnd   = "\t// zatrano:wire:imports:end\n"
	wireRegStart     = "\t// zatrano:wire:register:start\n"
	wireRegEnd       = "\t// zatrano:wire:register:end\n"
)

// ResolveWireFile picks the routes file to patch: app layout first, then framework layout.
func ResolveWireFile(moduleRoot string) (string, error) {
	moduleRoot = filepath.Clean(moduleRoot)
	candidates := []string{
		filepath.Join(moduleRoot, "internal", "routes", "register.go"),
		filepath.Join(moduleRoot, "pkg", "server", "register_modules.go"),
	}
	for _, c := range candidates {
		st, err := os.Stat(c)
		if err != nil {
			continue
		}
		if st.IsDir() {
			continue
		}
		b, err := os.ReadFile(c)
		if err != nil {
			return "", err
		}
		if !bytesHasMarkers(b) {
			continue
		}
		return c, nil
	}
	return "", fmt.Errorf("no wire target with zatrano:wire markers — add internal/routes/register.go (scaffolded apps) or pkg/server/register_modules.go (framework checkout)")
}

func bytesHasMarkers(b []byte) bool {
	s := string(b)
	return strings.Contains(s, wireImportsStart) &&
		strings.Contains(s, wireImportsEnd) &&
		strings.Contains(s, wireRegStart) &&
		strings.Contains(s, wireRegEnd)
}

// WireTargetsFromModuleDir decides Register vs RegisterCRUD from files under modules/<pkg>/.
func WireTargetsFromModuleDir(moduleRoot, out, rawName string) (addRegister, addCRUD bool, moduleDir string, err error) {
	pkg := PackageName(rawName)
	if pkg == "" {
		return false, false, "", fmt.Errorf("invalid module name %q", rawName)
	}
	moduleDir = filepath.Join(moduleRoot, out, pkg)
	reg := filepath.Join(moduleDir, "register.go")
	crud := filepath.Join(moduleDir, "crud_register.go")
	if _, e := os.Stat(reg); e == nil {
		addRegister = true
	}
	if _, e := os.Stat(crud); e == nil {
		addCRUD = true
	}
	if !addRegister && !addCRUD {
		return false, false, moduleDir, fmt.Errorf("expected register.go and/or crud_register.go under %s (run gen module / gen crud first)", filepath.ToSlash(moduleDir))
	}
	return addRegister, addCRUD, moduleDir, nil
}

// WirePatch updates import and register markers for one module package (snake_case name).
func WirePatch(wireFile, moduleImport, relPkg string, pkgName string, addRegister, addCRUD bool) error {
	if !addRegister && !addCRUD {
		return nil
	}
	relPkg = strings.Trim(relPkg, "/")
	fullImport := strings.TrimSuffix(moduleImport, "/") + "/" + relPkg

	b, err := os.ReadFile(wireFile)
	if err != nil {
		return err
	}
	s := string(b)
	s, err = patchWireImports(s, fullImport)
	if err != nil {
		return err
	}
	s, err = patchWireRegisters(s, pkgName, addRegister, addCRUD)
	if err != nil {
		return err
	}
	return os.WriteFile(wireFile, []byte(s), 0o644)
}

func patchWireImports(s, importPath string) (string, error) {
	i := strings.Index(s, wireImportsStart)
	if i < 0 {
		return "", fmt.Errorf("missing %q in %s", strings.TrimSpace(wireImportsStart), "wire file")
	}
	j := strings.Index(s[i+len(wireImportsStart):], wireImportsEnd)
	if j < 0 {
		return "", fmt.Errorf("missing %q", strings.TrimSpace(wireImportsEnd))
	}
	absEnd := i + len(wireImportsStart) + j
	inner := s[i+len(wireImportsStart) : absEnd]
	line := fmt.Sprintf("\t%q\n", importPath)
	if strings.Contains(inner, importPath) {
		return s, nil
	}
	newInner := inner + line
	return s[:i+len(wireImportsStart)] + newInner + s[absEnd:], nil
}

func patchWireRegisters(s string, pkgName string, addRegister, addCRUD bool) (string, error) {
	i := strings.Index(s, wireRegStart)
	if i < 0 {
		return "", fmt.Errorf("missing %q", strings.TrimSpace(wireRegStart))
	}
	j := strings.Index(s[i+len(wireRegStart):], wireRegEnd)
	if j < 0 {
		return "", fmt.Errorf("missing %q", strings.TrimSpace(wireRegEnd))
	}
	absEnd := i + len(wireRegStart) + j
	inner := s[i+len(wireRegStart) : absEnd]

	out := inner
	if addRegister && !strings.Contains(inner, pkgName+".Register(") {
		out = fmt.Sprintf("\t%s.Register(a, app)\n", pkgName) + out
	}
	if addCRUD && !strings.Contains(inner, pkgName+".RegisterCRUD(") {
		out = out + fmt.Sprintf("\t%s.RegisterCRUD(a, app)\n", pkgName)
	}
	if out == inner {
		return s, nil
	}
	return s[:i+len(wireRegStart)] + out + s[absEnd:], nil
}

