package hashid_test

import (
	"testing"

	"github.com/zatrano/framework/packages/hashid"
)

func TestHashidRoundTrip(t *testing.T) {
	h := hashid.New("zatrano-salt", 8)
	hash, err := h.Encode(42)
	if err != nil || len(hash) < 8 {
		t.Fatalf("hash=%q err=%v", hash, err)
	}
	nums, err := h.Decode(hash)
	if err != nil || len(nums) != 1 || nums[0] != 42 {
		t.Fatalf("nums=%v err=%v", nums, err)
	}
	multi, err := h.Encode(1, 2, 3)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := h.Decode(multi)
	if err != nil || len(decoded) != 3 || decoded[0] != 1 || decoded[2] != 3 {
		t.Fatalf("decoded=%v err=%v", decoded, err)
	}
}
