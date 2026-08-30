package agent_test

import (
	"context"
	"sync"
	"testing"

	"github.com/zatrano/framework/packages/agent"
	"github.com/zatrano/framework/packages/ai"
	"github.com/zatrano/framework/packages/queue"
)

func TestQueueRunnerSync(t *testing.T) {
	mgr := ai.New()
	cat := agent.NewCatalog()
	if err := cat.Register("support", &agent.Agent{
		Chat:     agent.FromManager(mgr),
		MaxSteps: 1,
	}); err != nil {
		t.Fatal(err)
	}

	var mu sync.Mutex
	var outcomes []agent.RunOutcome
	runner := &agent.Runner{
		Catalog: cat,
		OnResult: func(o agent.RunOutcome) {
			mu.Lock()
			outcomes = append(outcomes, o)
			mu.Unlock()
		},
	}

	syncQ := queue.NewSyncQueue()
	qm := queue.NewManager("sync", map[string]queue.Queue{"sync": syncQ})
	if err := runner.RegisterQueue(qm); err != nil {
		t.Fatal(err)
	}

	err := runner.PushRun(qm, agent.RunJob{
		Agent:   "support",
		Message: "hello queue",
		ID:      "job-1",
	})
	if err != nil {
		t.Fatal(err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(outcomes) != 1 || outcomes[0].ID != "job-1" || outcomes[0].Err != nil {
		t.Fatalf("%+v", outcomes)
	}
	if outcomes[0].Result == nil || outcomes[0].Result.Response == nil {
		t.Fatal("missing result")
	}
}

func TestPushRunUnknownAgent(t *testing.T) {
	cat := agent.NewCatalog()
	syncQ := queue.NewSyncQueue()
	qm := queue.NewManager("sync", map[string]queue.Queue{"sync": syncQ})
	err := agent.PushRun(qm, cat, agent.RunJob{Agent: "nope", Message: "x"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCatalogGet(t *testing.T) {
	cat := agent.NewCatalog()
	_ = cat.Register("A", &agent.Agent{Chat: agent.FromManager(ai.New()), MaxSteps: 1})
	if _, ok := cat.Get("a"); !ok {
		t.Fatal("case fold")
	}
	_ = context.Background()
}
