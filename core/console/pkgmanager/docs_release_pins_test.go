package pkgmanager

import (
	"bufio"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

func TestPackagesGuideReleaseBlockMatchesPins(t *testing.T) {
	root := frameworkRootForDocs(t)
	version, err := os.ReadFile(filepath.Join(root, "VERSION"))
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"github.com/zatrano/framework/v3": "v" + strings.TrimSpace(string(version)),
		"github.com/zatrano/packages":     strings.TrimPrefix(packagesModuleGetArg, "github.com/zatrano/packages@"),
		"github.com/zatrano/rawhttp":      goModRequire(t, root, "github.com/zatrano/rawhttp"),
		"github.com/zatrano/canvas":       goModRequire(t, root, "github.com/zatrano/canvas"),
	}
	got := releaseBlock(t, filepath.Join(root, "PACKAGES.md"))
	for mod, ver := range want {
		if got[mod] != ver {
			t.Errorf("PACKAGES.md current releases: %s is %q, want %q", mod, got[mod], ver)
		}
	}
	for mod := range got {
		if _, ok := want[mod]; !ok {
			t.Errorf("PACKAGES.md current releases lists %s, which no pin backs", mod)
		}
	}
}

var frameworkGetRef = regexp.MustCompile(`github\.com/zatrano/framework(/v(\d+))?@v(\d+)\.`)

func TestDocsNeverPairFrameworkPathWithAnotherMajor(t *testing.T) {
	root := frameworkRootForDocs(t)
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "node_modules", "vendor", "storage":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.EqualFold(filepath.Ext(path), ".md") {
			return nil
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		for i, line := range strings.Split(string(body), "\n") {
			for _, m := range frameworkGetRef.FindAllStringSubmatch(line, -1) {
				pathMajor := 1
				if m[2] != "" {
					pathMajor, _ = strconv.Atoi(m[2])
				}
				tagMajor, _ := strconv.Atoi(m[3])
				if tagMajor < 2 && pathMajor == 1 {
					continue
				}
				if pathMajor != tagMajor {
					t.Errorf("%s:%d: %q pairs module major %d with tag major %d", filepath.ToSlash(rel), i+1, m[0], pathMajor, tagMajor)
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func releaseBlock(t *testing.T, path string) map[string]string {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	start := strings.Index(text, "Current public releases")
	if start < 0 {
		t.Fatalf("%s has no current public releases block", path)
	}
	open := strings.Index(text[start:], "```text\n")
	if open < 0 {
		open = strings.Index(text[start:], "```text\r\n")
	}
	if open < 0 {
		t.Fatalf("%s current public releases block is not a text fence", path)
	}
	rest := text[start+open:]
	rest = rest[strings.Index(rest, "\n")+1:]
	end := strings.Index(rest, "```")
	if end < 0 {
		t.Fatalf("%s current public releases block is not closed", path)
	}
	out := map[string]string{}
	for _, line := range strings.Split(rest[:end], "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 {
			out[fields[0]] = fields[1]
		}
	}
	return out
}

func goModRequire(t *testing.T, root, module string) string {
	t.Helper()
	f, err := os.Open(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		fields := strings.Fields(strings.TrimPrefix(strings.TrimSpace(sc.Text()), "require "))
		if len(fields) >= 2 && fields[0] == module {
			return fields[1]
		}
	}
	t.Fatalf("go.mod does not require %s", module)
	return ""
}

func frameworkRootForDocs(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		mod, err := os.ReadFile(filepath.Join(dir, "go.mod"))
		if err == nil && strings.Contains(string(mod), "module github.com/zatrano/framework/v3\n") ||
			err == nil && strings.Contains(string(mod), "module github.com/zatrano/framework/v3\r\n") {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("framework module root not found")
		}
		dir = parent
	}
}
