package process_test

import (
	"runtime"
	"testing"

	"github.com/zatrano/framework/packages/process"
)

func TestCommand(t *testing.T) {
	var result process.Result
	if runtime.GOOS == "windows" {
		result = process.Command("cmd", "/C", "echo hello").Run()
	} else {
		result = process.Command("echo", "hello").Run()
	}
	if !result.Successful() {
		t.Fatalf("process failed: %v %s", result.Err, result.Stderr)
	}
	if result.Output() == "" {
		t.Fatal("expected output")
	}
}
