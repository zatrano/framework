package bus_test

import (
	"testing"

	"github.com/zatrano/framework/packages/bus"
)

type greetCommand struct {
	Name string
}

func TestBusDispatch(t *testing.T) {
	b := bus.New()
	b.Map(greetCommand{}, func(command any) (any, error) {
		cmd := command.(greetCommand)
		return "hello " + cmd.Name, nil
	})
	b.MapNamed("math.add", func(command any) (any, error) {
		payload := command.(map[string]any)
		return int(payload["a"].(int)) + int(payload["b"].(int)), nil
	})

	got, err := b.Dispatch(greetCommand{Name: "ada"})
	if err != nil || got != "hello ada" {
		t.Fatalf("got=%v err=%v", got, err)
	}
	sum, err := b.DispatchNamed("math.add", map[string]any{"a": 2, "b": 3})
	if err != nil || sum != 5 {
		t.Fatalf("sum=%v err=%v", sum, err)
	}
}
