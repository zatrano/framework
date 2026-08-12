package circuit_test

import (
	"errors"
	"testing"
	"time"

	"github.com/zatrano/framework/packages/circuit"
)

func TestCircuitOpens(t *testing.T) {
	m := circuit.New(circuit.Settings{FailureThreshold: 2, SuccessThreshold: 1, Timeout: 50 * time.Millisecond})
	b := m.Breaker("demo")
	_ = b.Execute(func() error { return errors.New("fail") })
	_ = b.Execute(func() error { return errors.New("fail") })
	if b.State() != circuit.Open {
		t.Fatalf("state=%s", b.State())
	}
	if err := b.Execute(func() error { return nil }); !errors.Is(err, circuit.ErrOpen) {
		t.Fatalf("err=%v", err)
	}
	time.Sleep(60 * time.Millisecond)
	if err := b.Execute(func() error { return nil }); err != nil {
		t.Fatal(err)
	}
	if b.State() != circuit.Closed {
		t.Fatalf("state=%s", b.State())
	}
}
