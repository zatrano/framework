package addons_test

import (
	"testing"

	_ "github.com/zatrano/framework/bootstrap"
	"github.com/zatrano/framework/bootstrap/addons"
)

func TestSelectUnknown(t *testing.T) {
	_, err := addons.Select("not-a-real-package")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSelectEmpty(t *testing.T) {
	got, err := addons.Select()
	if err != nil || got != nil {
		t.Fatalf("got %#v err=%v", got, err)
	}
}

func TestFrameworkRegistryEmpty(t *testing.T) {
	if len(addons.Available()) != 0 {
		t.Fatalf("framework binary must not register packages, got %v", addons.Names())
	}
}
