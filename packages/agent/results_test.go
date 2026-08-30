package agent_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/zatrano/framework/packages/agent"
	"github.com/zatrano/framework/packages/ai"
	"github.com/zatrano/framework/packages/queue"
)

func TestResultStoreMemoryAndFile(t *testing.T) {
	mem := agent.NewMemoryResultStore()
	run := agent.StoredRun{ID: "1", Agent: "support", Message: "hi", Content: "hello"}
	if err := mem.Put(context.Background(), run); err != nil {
		t.Fatal(err)
	}
	got, ok, err := mem.Get(context.Background(), "1")
	if err != nil || !ok || got.Content != "hello" {
		t.Fatalf("%v %v %+v", err, ok, got)
	}

	path := filepath.Join(t.TempDir(), "results.json")
	file := agent.NewJSONFileResultStore(path)
	if err := file.Put(context.Background(), run); err != nil {
		t.Fatal(err)
	}
	file2 := agent.NewJSONFileResultStore(path)
	got, ok, err = file2.Get(context.Background(), "1")
	if err != nil || !ok || got.Agent != "support" {
		t.Fatalf("%v %v %+v", err, ok, got)
	}
}

func TestRunnerPersistsResults(t *testing.T) {
	mgr := ai.New()
	cat := agent.NewCatalog()
	_ = cat.Register("support", &agent.Agent{Chat: agent.FromManager(mgr), MaxSteps: 1})
	store := agent.NewMemoryResultStore()
	runner := &agent.Runner{Catalog: cat, Results: store}
	syncQ := queue.NewSyncQueue()
	qm := queue.NewManager("sync", map[string]queue.Queue{"sync": syncQ})
	_ = runner.RegisterQueue(qm)
	if err := runner.PushRun(qm, agent.RunJob{Agent: "support", Message: "ping", ID: "r1"}); err != nil {
		t.Fatal(err)
	}
	got, ok, err := store.Get(context.Background(), "r1")
	if err != nil || !ok || got.Content == "" {
		t.Fatalf("%v %v %+v", err, ok, got)
	}
}
