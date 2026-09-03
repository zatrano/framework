package main

import "testing"

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
		{[]string{"db:setup", "--drivers=sqlite"}, true},
		{[]string{"list"}, false},
	}
	for _, tc := range cases {
		if got := cliUsesCoreBoot(tc.args); got != tc.want {
			t.Fatalf("cliUsesCoreBoot(%v)=%v want %v", tc.args, got, tc.want)
		}
	}
}
