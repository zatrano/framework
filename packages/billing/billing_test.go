package billing_test

import (
	"testing"

	"github.com/zatrano/framework/packages/billing"
)

func TestBillingFlow(t *testing.T) {
	m := billing.New("http://localhost:8080")
	cus, err := m.CreateCustomer("buyer@zatrano.test", "Buyer")
	if err != nil {
		t.Fatal(err)
	}
	sub, err := m.Subscribe(cus.ID, "default", "price_pro", 7)
	if err != nil {
		t.Fatal(err)
	}
	if !m.Subscribed(cus.ID, "default") {
		t.Fatal("expected subscribed")
	}
	if !m.OnTrial(cus.ID, "default") {
		t.Fatal("expected trial")
	}
	session, err := m.Checkout(cus.ID, "price_pro")
	if err != nil || session.URL == "" {
		t.Fatalf("checkout=%v err=%v", session, err)
	}
	pay, err := m.CheckoutPayment(cus.ID, "usd", []billing.PaymentLineItem{
		{Name: "Setup fee", Amount: 2500, Quantity: 1},
	})
	if err != nil || pay.URL == "" {
		t.Fatalf("checkout payment=%v err=%v", pay, err)
	}
	inv, err := m.Invoice(cus.ID, 1999, "usd")
	if err != nil || inv.Status != "paid" {
		t.Fatalf("invoice=%v err=%v", inv, err)
	}
	canceled, err := m.Cancel(sub.ID, true)
	if err != nil || canceled.Status != "canceled" {
		t.Fatalf("cancel=%v err=%v", canceled, err)
	}
}
