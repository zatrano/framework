package uuid_test

import (
	"testing"

	"github.com/zatrano/framework/packages/support/uuid"
)

func TestUUID(t *testing.T) {
	id := uuid.New()
	if !uuid.IsValid(id) {
		t.Fatalf("invalid: %s", id)
	}
	if !uuid.IsV4(id) {
		t.Fatalf("not v4: %s", id)
	}
	parsed, err := uuid.Parse(id)
	if err != nil {
		t.Fatal(err)
	}
	if uuid.Format(parsed) != id {
		t.Fatalf("roundtrip %s != %s", uuid.Format(parsed), id)
	}
	if uuid.IsValid("not-a-uuid") {
		t.Fatal("expected invalid")
	}
}
