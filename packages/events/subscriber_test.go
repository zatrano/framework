package events_test

import (
	"testing"

	"github.com/zatrano/framework/packages/events"
)

type demoSubscriber struct {
	hits int
}

func (s *demoSubscriber) Subscribe(d *events.Dispatcher) {
	d.Listen("order.placed", s.onPlaced)
	d.Listen("order.shipped", s.onShipped)
}

func (s *demoSubscriber) onPlaced(event any) error {
	s.hits++
	return nil
}

func (s *demoSubscriber) onShipped(event any) error {
	s.hits += 10
	return nil
}

func TestRegisterSubscriber(t *testing.T) {
	d := events.New()
	sub := &demoSubscriber{}
	d.Register(sub)
	d.RegisterSubscribers(&demoSubscriber{}) // second instance also listens

	if err := d.Dispatch("order.placed", map[string]any{"id": 1}); err != nil {
		t.Fatal(err)
	}
	if err := d.Dispatch("order.shipped", nil); err != nil {
		t.Fatal(err)
	}
	if sub.hits != 11 {
		t.Fatalf("hits=%d", sub.hits)
	}
	if !d.HasListeners("order.placed") || !d.HasListeners("order.shipped") {
		t.Fatal("expected listeners")
	}
}
