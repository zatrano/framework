package compatibility_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/zatrano/framework/v2/console"
)

// TestGoldenAppCompilesAgainstCurrentFramework scaffolds `zatrano new`
// and verifies the generated application builds and tests against this checkout
// without changing application source.
func TestGoldenAppCompilesAgainstCurrentFramework(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(t.TempDir(), "golden")
	cmd := &console.NewCommand{}
	if err := cmd.Handle([]string{dest, "--module", "example.com/golden", "--replace", root}); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"vet", "./..."},
		{"test", "./..."},
	} {
		c := exec.Command("go", args...)
		c.Dir = dest
		out, err := c.CombinedOutput()
		if err != nil {
			t.Fatalf("go %s: %v\n%s", args[0], err, out)
		}
	}
	if _, err := os.Stat(filepath.Join(dest, "cmd", "app", "main.go")); err != nil {
		t.Fatal(err)
	}
}
