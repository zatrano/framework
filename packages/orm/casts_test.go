package orm

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

type castDemo struct {
	Model
	Active bool           `db:"active"`
	Meta   map[string]any `db:"meta"`
	Score  int64          `db:"score"`
}

func (castDemo) Casts() map[string]string {
	return map[string]string{
		"active": "bool",
		"meta":   "json",
		"score":  "int",
	}
}

func TestCastsHydrateAndCollect(t *testing.T) {
	row := map[string]any{
		"id":     int64(1),
		"active": "1",
		"meta":   `{"role":"admin"}`,
		"score":  "42",
	}
	model, err := mapToModel[castDemo](row)
	if err != nil {
		t.Fatal(err)
	}
	if !model.Active || model.Score != 42 {
		t.Fatalf("unexpected %#v", model)
	}
	if model.Meta["role"] != "admin" {
		t.Fatalf("meta=%#v", model.Meta)
	}
	attrs := modelToMap(reflect.ValueOf(model).Elem())
	if attrs["active"] != 1 {
		t.Fatalf("expected active=1, got %#v", attrs["active"])
	}
	meta, _ := attrs["meta"].(string)
	if !strings.Contains(meta, "admin") {
		t.Fatalf("meta attr=%#v", attrs["meta"])
	}
}

func TestCustomCastRegistry(t *testing.T) {
	ClearCasts()
	defer ClearCasts()
	RegisterCast("shout", func(v any) (any, error) {
		return strings.ToUpper(fmt.Sprint(v)), nil
	}, func(v any) any {
		return strings.ToLower(fmt.Sprint(v))
	})
	got, err := CastValue("shout", "hi")
	if err != nil || got != "HI" {
		t.Fatalf("in=%v err=%v", got, err)
	}
	if out := castOutgoing("shout", "HI"); out != "hi" {
		t.Fatalf("out=%v", out)
	}
	if !HasCast("shout") {
		t.Fatal("HasCast")
	}
}
