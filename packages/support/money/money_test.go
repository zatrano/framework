package money_test

import (
	"testing"

	"github.com/zatrano/framework/packages/support/money"
)

func TestMoney(t *testing.T) {
	a := money.FromMajor(10.50, "USD")
	b := money.Of(250, "USD")
	sum, err := a.Add(b)
	if err != nil || sum.Amount != 1300 {
		t.Fatalf("%+v err=%v", sum, err)
	}
	parts, err := money.Of(100, "USD").Allocate(3)
	if err != nil || len(parts) != 3 {
		t.Fatal(err)
	}
	total := int64(0)
	for _, p := range parts {
		total += p.Amount
	}
	if total != 100 {
		t.Fatalf("total=%d parts=%v", total, parts)
	}
	if a.Format("$") != "$10.50" {
		t.Fatal(a.Format("$"))
	}
}
