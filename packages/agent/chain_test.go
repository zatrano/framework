package agent_test

import (
	"context"
	"strings"
	"testing"

	"github.com/zatrano/framework/packages/agent"
	"github.com/zatrano/framework/packages/ai"
)

func TestChainSequential(t *testing.T) {
	mgr := ai.New()
	research := &agent.Agent{Chat: agent.FromManager(mgr), System: "researcher", MaxSteps: 1}
	writer := &agent.Agent{Chat: agent.FromManager(mgr), System: "writer", MaxSteps: 1}
	chain := &agent.Chain{Steps: []agent.ChainStep{
		{Name: "research", Agent: research},
		{Name: "write", Agent: writer, Prompt: func(input, prev string) string {
			return "Rewrite clearly:\n" + prev
		}},
	}}
	res, err := chain.Run(context.Background(), "What is ZATRANO?")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Steps) != 2 || res.Output == "" {
		t.Fatalf("%+v", res)
	}
	if !strings.Contains(res.Steps[1].Input, "Rewrite") {
		t.Fatalf("%q", res.Steps[1].Input)
	}
}

func TestCatalogChain(t *testing.T) {
	mgr := ai.New()
	cat := agent.NewCatalog()
	_ = cat.Register("a", &agent.Agent{Chat: agent.FromManager(mgr), MaxSteps: 1})
	_ = cat.Register("b", &agent.Agent{Chat: agent.FromManager(mgr), MaxSteps: 1})
	chain, err := agent.CatalogChain(cat, []string{"a", "b"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	res, err := chain.Run(context.Background(), "ping")
	if err != nil || len(res.Steps) != 2 {
		t.Fatalf("%v %+v", err, res)
	}
}
