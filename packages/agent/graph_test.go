package agent_test

import (
	"context"
	"strings"
	"testing"

	"github.com/zatrano/framework/packages/agent"
	"github.com/zatrano/framework/packages/ai"
)

func TestGraphBranching(t *testing.T) {
	mgr := ai.New()
	classify := &agent.Agent{Chat: agent.FromManager(mgr), MaxSteps: 1}
	tech := &agent.Agent{Chat: agent.FromManager(mgr), MaxSteps: 1}
	general := &agent.Agent{Chat: agent.FromManager(mgr), MaxSteps: 1}

	g := &agent.Graph{
		Start: "classify",
		Nodes: map[string]agent.GraphNode{
			"classify": {
				Agent: classify,
				Prompt: func(s agent.GraphState) string {
					return "Classify: " + s.Input
				},
				Route: agent.RouteContains("stub", "tech", "general"),
			},
			"tech": {
				Agent: tech,
				Prompt: func(s agent.GraphState) string {
					return "Tech answer: " + s.Input
				},
			},
			"general": {
				Agent: general,
			},
		},
		MaxHops: 4,
	}
	res, err := g.Run(context.Background(), "How do profiles work?")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Path) < 2 || res.Path[0] != "classify" {
		t.Fatalf("%+v", res.Path)
	}
	// Fake reply contains "stub" → RouteContains → tech
	if res.Path[1] != "tech" {
		t.Fatalf("path=%v", res.Path)
	}
	if !strings.Contains(res.Steps[1].Input, "Tech answer") {
		t.Fatalf("%q", res.Steps[1].Input)
	}
}

func TestGraphMaxHops(t *testing.T) {
	mgr := ai.New()
	a := &agent.Agent{Chat: agent.FromManager(mgr), MaxSteps: 1}
	g := &agent.Graph{
		Start: "loop",
		Nodes: map[string]agent.GraphNode{
			"loop": {
				Agent: a,
				Route: func(ctx context.Context, s agent.GraphState) (string, error) {
					return "loop", nil
				},
			},
		},
		MaxHops: 2,
	}
	_, err := g.Run(context.Background(), "x")
	if err == nil || !strings.Contains(err.Error(), "max hops") {
		t.Fatalf("%v", err)
	}
}
