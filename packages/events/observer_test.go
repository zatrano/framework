package events_test

import (
	"testing"

	"github.com/zatrano/framework/packages/events"
)

type demoObserver struct {
	created int
	updated int
	deleted int
}

func (o *demoObserver) Created(event any) error {
	o.created++
	return nil
}
func (o *demoObserver) Updated(event any) error {
	o.updated++
	return nil
}
func (o *demoObserver) Deleted(event any) error {
	o.deleted++
	return nil
}

func TestObserveModel(t *testing.T) {
	d := events.New()
	obs := &demoObserver{}
	d.ObserveModel("user", obs)

	if err := d.Dispatch("user.created", map[string]any{"id": 1}); err != nil {
		t.Fatal(err)
	}
	if err := d.Dispatch("user.updated", map[string]any{"id": 1}); err != nil {
		t.Fatal(err)
	}
	if err := d.Dispatch("user.deleted", map[string]any{"id": 1}); err != nil {
		t.Fatal(err)
	}
	if obs.created != 1 || obs.updated != 1 || obs.deleted != 1 {
		t.Fatalf("unexpected counts %#v", obs)
	}
}

func TestObserveMap(t *testing.T) {
	d := events.New()
	var seen string
	d.Observe("order", map[string]events.Listener{
		"paid": func(event any) error {
			seen = "paid"
			return nil
		},
	})
	if err := d.Dispatch("order.paid", nil); err != nil {
		t.Fatal(err)
	}
	if seen != "paid" {
		t.Fatalf("expected paid, got %q", seen)
	}
}
