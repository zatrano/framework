package audit_test

import (
	"testing"

	"github.com/zatrano/framework/packages/audit"
)

func TestMemoryAuditStore(t *testing.T) {
	store := audit.NewMemoryStore(10)
	m := audit.New(store)
	if err := m.Record("test.action", audit.WithActor("ada")); err != nil {
		t.Fatal(err)
	}
	events, err := m.Recent(5)
	if err != nil || len(events) != 1 || events[0].Action != "test.action" {
		t.Fatalf("unexpected %#v err=%v", events, err)
	}
}
