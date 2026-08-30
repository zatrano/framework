package agent

import (
	"context"
	"fmt"
	"strings"
)

// GraphState is the mutable context passed between graph nodes.
type GraphState struct {
	Input    string
	Previous string // last assistant (or input before first hop)
	Output   string
	Data     map[string]string // optional scratch keys for routers
	Visits   map[string]int
}

// GraphNode is one named hop in a branching Graph.
type GraphNode struct {
	Agent *Agent
	// Prompt builds the user message; nil → use state.Previous.
	Prompt func(state GraphState) string
	// Route chooses the next node name. Empty string ends the graph.
	// nil → end after this node.
	Route func(ctx context.Context, state GraphState) (next string, err error)
}

// Graph runs agents with conditional edges (branching).
type Graph struct {
	Start   string
	Nodes   map[string]GraphNode
	MaxHops int // default 8; safety against cycles
}

// GraphResult is the outcome of Graph.Run.
type GraphResult struct {
	Output string
	Steps  []StepTrace
	Path   []string // visited node names in order
}

// Run starts at Start and follows Route decisions until end or MaxHops.
func (g *Graph) Run(ctx context.Context, input string) (*GraphResult, error) {
	if g == nil || len(g.Nodes) == 0 {
		return nil, fmt.Errorf("agent: graph has no nodes")
	}
	start := strings.TrimSpace(g.Start)
	if start == "" {
		return nil, fmt.Errorf("agent: graph Start is required")
	}
	if _, ok := g.Nodes[start]; !ok {
		return nil, fmt.Errorf("agent: unknown start node %q", start)
	}
	if ctx == nil {
		ctx = context.Background()
	}
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, fmt.Errorf("agent: graph input is required")
	}
	maxHops := g.MaxHops
	if maxHops <= 0 {
		maxHops = 8
	}
	state := GraphState{
		Input:    input,
		Previous: input,
		Data:     map[string]string{},
		Visits:   map[string]int{},
	}
	out := &GraphResult{}
	cur := start
	for hop := 0; hop < maxHops; hop++ {
		node, ok := g.Nodes[cur]
		if !ok {
			return out, fmt.Errorf("agent: unknown graph node %q", cur)
		}
		if node.Agent == nil {
			return out, fmt.Errorf("agent: graph node %q missing Agent", cur)
		}
		state.Visits[cur]++
		msg := state.Previous
		if node.Prompt != nil {
			msg = strings.TrimSpace(node.Prompt(state))
		}
		if msg == "" {
			return out, fmt.Errorf("agent: graph node %q empty prompt", cur)
		}
		res, err := node.Agent.Run(ctx, msg)
		trace := StepTrace{Name: cur, Input: msg, Result: res, Err: err}
		if res != nil && res.Response != nil {
			trace.Output = res.Response.Message.Content
			state.Previous = trace.Output
			state.Output = trace.Output
		}
		out.Steps = append(out.Steps, trace)
		out.Path = append(out.Path, cur)
		out.Output = state.Output
		if err != nil {
			return out, err
		}
		if node.Route == nil {
			return out, nil
		}
		next, err := node.Route(ctx, state)
		if err != nil {
			return out, err
		}
		next = strings.TrimSpace(next)
		if next == "" {
			return out, nil
		}
		cur = next
	}
	return out, fmt.Errorf("agent: graph max hops (%d) reached", maxHops)
}

// RouteIf returns a Route that goes to yes when pred is true, otherwise no (empty no ends).
func RouteIf(pred func(GraphState) bool, yes, no string) func(context.Context, GraphState) (string, error) {
	return func(ctx context.Context, state GraphState) (string, error) {
		if pred != nil && pred(state) {
			return yes, nil
		}
		return no, nil
	}
}

// RouteContains goes to yes when last output contains substr (case-insensitive).
func RouteContains(substr, yes, no string) func(context.Context, GraphState) (string, error) {
	sub := strings.ToLower(substr)
	return RouteIf(func(s GraphState) bool {
		return strings.Contains(strings.ToLower(s.Output), sub)
	}, yes, no)
}
