package http

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
)

func TestUseAfterReturnCheckIsEliminated(t *testing.T) {
	if requestFreedPoison {
		t.Skip("poison builds keep the panic string")
	}
	dir := t.TempDir()
	framework, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	rawhttpDir := filepath.Clean(filepath.Join(framework, "..", "rawhttp"))
	if _, err := os.Stat(filepath.Join(framework, "go.mod")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(
		"module dceproof\n\ngo 1.25.0\n\nrequire github.com/zatrano/framework/v3 v3.0.2\nrequire github.com/zatrano/rawhttp v0.2.3\n\n"+
			"replace github.com/zatrano/framework/v3 => "+strconv.Quote(filepath.ToSlash(framework))+"\n"+
			"replace github.com/zatrano/rawhttp => "+strconv.Quote(filepath.ToSlash(rawhttpDir))+"\n",
	), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(`package main

import "github.com/zatrano/framework/v3/core/kernel/http"

func main() {
	_ = http.NewRequest(nil).Path()
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(dir, "app.exe")
	cmd := exec.Command("go", "build", "-o", bin, ".")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOWORK=off", "CGO_ENABLED=0")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	blob, err := os.ReadFile(bin)
	if err != nil {
		t.Fatal(err)
	}
	if bytesContains(blob, []byte("request used after handler returned")) {
		t.Fatal("production binary still contains the use-after-return panic")
	}
}

func bytesContains(haystack, needle []byte) bool {
	if len(needle) == 0 || len(haystack) < len(needle) {
		return false
	}
	for i := 0; i+len(needle) <= len(haystack); i++ {
		match := true
		for j := range needle {
			if haystack[i+j] != needle[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
