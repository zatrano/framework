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

func TestSelectKnown(t *testing.T) {
	got, err := addons.Select("ai", "ai")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 provider, got %d", len(got))
	}
}

func TestRegistryCoversInTreeServices(t *testing.T) {
	if len(addons.Available()) < 1 {
		t.Fatalf("registry too small: %d", len(addons.Available()))
	}
}
