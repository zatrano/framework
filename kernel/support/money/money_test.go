package money_test

import (
	"testing"

	"github.com/zatrano/framework/v2/kernel/support/money"
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

func TestMoneySubMulFormat(t *testing.T) {
	a := money.Of(1000, "USD")
	b := money.Of(250, "USD")
	diff, err := a.Sub(b)
	if err != nil || diff.Amount != 750 {
		t.Fatalf("%+v %v", diff, err)
	}
	if _, err := a.Sub(money.Of(1, "EUR")); err == nil {
		t.Fatal("currency mismatch")
	}
	if a.Mul(2).Amount != 2000 {
		t.Fatal("mul")
	}
	if a.Format("") == "" {
		t.Fatal("format symbol empty")
	}
}
