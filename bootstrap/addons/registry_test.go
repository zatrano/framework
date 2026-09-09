package addons_test

import (
	"testing"

	"github.com/zatrano/framework/v2/bootstrap/addons"
	"github.com/zatrano/framework/v2/contracts"
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

func TestRegisterDuplicatePanics(t *testing.T) {
	addons.ClearRegistry()
	t.Cleanup(addons.ClearRegistry)
	addons.Register(addons.Meta{Name: "dup-test", Factory: func() contracts.Provider { return nil }})
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic on duplicate name")
		}
	}()
	addons.Register(addons.Meta{Name: "dup-test"})
}

func TestRegisterEmptyNameIsIgnored(t *testing.T) {
	addons.ClearRegistry()
	t.Cleanup(addons.ClearRegistry)
	addons.Register(addons.Meta{Name: "   "})
	if len(addons.Available()) != 0 {
		t.Fatalf("blank name must not register, got %v", addons.Names())
	}
}

func TestConfigFileNames(t *testing.T) {
	got := addons.ConfigFileNames(addons.Meta{ConfigFiles: map[string]string{
		"mongo.go": "package config\n",
		"oauth.go": "package config\n",
	}})
	if len(got) != 2 || got[0] != "mongo.go" || got[1] != "oauth.go" {
		t.Fatalf("got %#v", got)
	}
	if addons.ConfigFileNames(addons.Meta{}) != nil {
		t.Fatal("empty ConfigFiles must be nil")
	}
}
