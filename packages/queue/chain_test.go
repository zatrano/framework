package queue_test

import (
	"fmt"
	"testing"

	"github.com/zatrano/framework/packages/queue"
)

func TestQueueChainAndFail(t *testing.T) {
	syncQ := queue.NewSyncQueue()
	m := queue.NewManager("sync", map[string]queue.Queue{"sync": syncQ})
	steps := []string{}
	m.Register("step.a", func(payload map[string]any) error {
		steps = append(steps, "a")
		return nil
	})
	m.Register("step.b", func(payload map[string]any) error {
		steps = append(steps, "b")
		return nil
	})
	m.Register("step.fail", func(payload map[string]any) error {
		return fmt.Errorf("boom")
	})

	if err := m.Chain(
		queue.NamedJob{Name: "step.a"},
		queue.NamedJob{Name: "step.b"},
	); err != nil {
		t.Fatal(err)
	}
	if len(steps) != 2 || steps[0] != "a" || steps[1] != "b" {
		t.Fatalf("steps=%v", steps)
	}

	err := m.Chain(queue.NamedJob{Name: "step.fail"})
	if err == nil {
		t.Fatal("expected failure")
	}
	failed := m.Failed()
	if len(failed) == 0 || failed[len(failed)-1].Error != "boom" {
		t.Fatalf("failed=%#v", failed)
	}
}
