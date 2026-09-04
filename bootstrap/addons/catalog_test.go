package addons_test

import (
	"testing"

	_ "github.com/zatrano/framework/bootstrap"
	"github.com/zatrano/framework/bootstrap/addons"
)

func TestFrameworkBinaryRegistersNoPackages(t *testing.T) {
	if got := addons.Available(); len(got) != 0 {
		t.Fatalf("framework must not blank-import packages, got %v", addons.Names())
	}
}
