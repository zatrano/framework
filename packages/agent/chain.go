package agent

import (
	"context"
	"fmt"
	"strings"
)

// ChainStep is one sequential agent invocation in a Chain.
type ChainStep struct {
	Name  string // optional label for traces
	Agent *Agent
	// Prompt builds the user message for this step.
	// If nil, uses the previous step's assistant content (or the chain input on step 0).
	Prompt func(input, previous string) string
}

// ChainResult is the outcome of Chain.Run.
type ChainResult struct {
	Output string
	Steps  []StepTrace
}

// StepTrace records one chain hop.
type StepTrace struct {
	Name   string
	Input  string
	Output string
	Result *Result
	Err    error
}

// Chain runs agents sequentially, feeding each step the previous output.
type Chain struct {
	Steps []ChainStep
}

// Run executes all steps. Stops on the first error.
func (c *Chain) Run(ctx context.Context, input string) (*ChainResult, error) {
	if c == nil || len(c.Steps) == 0 {
		return nil, fmt.Errorf("agent: chain has no steps")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, fmt.Errorf("agent: chain input is required")
	}
	prev := input
	out := &ChainResult{}
	for i, step := range c.Steps {
		if step.Agent == nil {
			return out, fmt.Errorf("agent: chain step %d missing Agent", i)
		}
		name := strings.TrimSpace(step.Name)
		if name == "" {
			name = fmt.Sprintf("step-%d", i+1)
		}
		msg := prev
		if step.Prompt != nil {
			msg = strings.TrimSpace(step.Prompt(input, prev))
		}
		if msg == "" {
			return out, fmt.Errorf("agent: chain step %q empty prompt", name)
		}
		// Fresh memory per step so prior tool transcripts do not leak unless the Agent shares Memory.
		res, err := step.Agent.Run(ctx, msg)
		trace := StepTrace{Name: name, Input: msg, Result: res, Err: err}
		if res != nil && res.Response != nil {
			trace.Output = res.Response.Message.Content
			prev = trace.Output
		}
		out.Steps = append(out.Steps, trace)
		if err != nil {
			return out, err
		}
		out.Output = prev
	}
	return out, nil
}

// CatalogChain builds a Chain from catalog agent names (same Prompt for all optional).
func CatalogChain(cat *Catalog, names []string, prompt func(input, previous string) string) (*Chain, error) {
	if cat == nil {
		return nil, fmt.Errorf("agent: catalog is nil")
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("agent: no agent names")
	}
	steps := make([]ChainStep, 0, len(names))
	for _, n := range names {
		a, ok := cat.Get(n)
		if !ok {
			return nil, fmt.Errorf("agent: unknown agent %q", n)
		}
		steps = append(steps, ChainStep{Name: n, Agent: a, Prompt: prompt})
	}
	return &Chain{Steps: steps}, nil
}
