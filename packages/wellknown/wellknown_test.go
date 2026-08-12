package wellknown_test

import (
	"strings"
	"testing"
	"time"

	"github.com/zatrano/framework/packages/wellknown"
)

func TestSecurityTxt(t *testing.T) {
	repo := wellknown.New(wellknown.Config{
		ContactEmail: "security@zatrano.test",
		Expires:      time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC),
		Canonical:    "https://zatrano.test/.well-known/security.txt",
	})
	txt := repo.SecurityTxt()
	if !strings.Contains(txt, "Contact: mailto:security@zatrano.test") {
		t.Fatal(txt)
	}
	if !strings.Contains(txt, "Expires:") {
		t.Fatal(txt)
	}
}
