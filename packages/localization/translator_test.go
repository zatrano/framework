package localization_test

import (
	"testing"

	"github.com/zatrano/framework/packages/localization"
)

func TestEmbeddedDefaults(t *testing.T) {
	tr := localization.New("", "en", "en")
	if err := tr.Load("en"); err != nil {
		t.Fatal(err)
	}
	if got := tr.Get("messages.welcome_title"); got != "ZATRANO" {
		t.Fatalf("welcome_title=%q", got)
	}
	trTR := localization.New("", "tr", "en")
	if err := trTR.Load("tr"); err != nil {
		t.Fatal(err)
	}
	if got := trTR.Get("messages.built_with"); got == "" || got == "messages.built_with" {
		t.Fatalf("tr built_with missing: %q", got)
	}
}
