package main

import (
	"os/exec"
	"strings"
	"testing"
)

func TestCliUsesCoreBoot(t *testing.T) {
	cases := []struct {
		args []string
		want bool
	}{
		{nil, false},
		{[]string{}, false},
		{[]string{"serve"}, false},
		{[]string{"migrate"}, false},
		{[]string{"make:model", "Post"}, true},
		{[]string{"make:auth"}, true},
		{[]string{"make:controller", "X", "--api"}, true},
		{[]string{"new", "myapp"}, true},
		{[]string{"describe", "--format=json"}, true},
		{[]string{"doctor"}, true},
		{[]string{"agents:generate"}, true},
		{[]string{"db:setup", "--drivers=sqlite"}, true},
		{[]string{"list"}, false},
	}
	for _, tc := range cases {
		if got := cliUsesCoreBoot(tc.args); got != tc.want {
			t.Fatalf("cliUsesCoreBoot(%v)=%v want %v", tc.args, got, tc.want)
		}
	}
}

func TestZatranoBinaryHasNoPackageModule(t *testing.T) {
	out, err := exec.Command("go", "list", "-deps", ".").CombinedOutput()
	if err != nil {
		t.Fatalf("go list -deps: %v\n%s", err, out)
	}
	if strings.Contains(string(out), "github.com/zatrano/packages") {
		t.Fatalf("cmd/zatrano must not depend on github.com/zatrano/packages\n%s", out)
	}
}
