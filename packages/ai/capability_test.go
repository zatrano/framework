package ai_test

import (
	"testing"

	"github.com/zatrano/framework/packages/ai"
)

func TestCapabilitiesFake(t *testing.T) {
	m := ai.New()
	caps, err := m.Capabilities("fake")
	if err != nil {
		t.Fatal(err)
	}
	want := map[ai.Capability]bool{
		ai.CapChat: true, ai.CapEmbed: true, ai.CapStream: true,
		ai.CapTools: true, ai.CapJSON: true, ai.CapVision: true, ai.CapImage: true, ai.CapSpeech: true,
	}
	if len(caps) != len(want) {
		t.Fatalf("%v", caps)
	}
	for _, c := range caps {
		if !want[c] {
			t.Fatalf("unexpected %q in %v", c, caps)
		}
	}
	// sorted
	for i := 1; i < len(caps); i++ {
		if caps[i-1] > caps[i] {
			t.Fatalf("unsorted %v", caps)
		}
	}
	if !m.Supports(ai.CapEmbed, "fake") {
		t.Fatal("embed")
	}
}

func TestDescribeAndModels(t *testing.T) {
	m := ai.New()
	m.Extend("openai", ai.OpenAI("sk-test"))
	m.SetModels("openai",
		ai.ModelInfo{ID: "gpt-4o-mini", ContextWindow: 128000},
		ai.ModelInfo{ID: "gpt-4o", ContextWindow: 128000, Caps: []ai.Capability{ai.CapChat, ai.CapTools}},
	)
	info, err := m.Describe("openai")
	if err != nil {
		t.Fatal(err)
	}
	if info.Name != "openai" || info.Driver != "openai" || info.DefaultModel != "gpt-4o-mini" {
		t.Fatalf("%+v", info)
	}
	if len(info.Models) != 2 || info.Models[0].ContextWindow != 128000 {
		t.Fatalf("%+v", info.Models)
	}
	if !m.Supports(ai.CapJSON, "openai") {
		t.Fatal("json")
	}
	all := m.DescribeAll()
	if len(all) < 3 { // fake, log, openai
		t.Fatalf("%d", len(all))
	}
	for i := 1; i < len(all); i++ {
		if all[i-1].Name > all[i].Name {
			t.Fatalf("unsorted %#v", all)
		}
	}
}

func TestInferCapabilitiesUnknownDriver(t *testing.T) {
	d := failDriver{err: nil}
	caps := ai.InferCapabilities(d)
	if len(caps) != 1 || caps[0] != ai.CapChat {
		t.Fatalf("%v", caps)
	}
}
