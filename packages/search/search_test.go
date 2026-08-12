package search_test

import (
	"testing"

	"github.com/zatrano/framework/packages/search"
)

func TestSearchRanksTitleHigher(t *testing.T) {
	eng := search.NewMemoryEngine()
	_ = eng.Index(search.Document{ID: "1", Type: "post", Title: "Queues", Body: "background work"})
	_ = eng.Index(search.Document{ID: "2", Type: "post", Title: "Hello", Body: "queues are useful"})
	hits, err := eng.Search("queues", 10)
	if err != nil || len(hits) < 2 {
		t.Fatalf("hits=%v err=%v", hits, err)
	}
	if hits[0].Document.ID != "1" {
		t.Fatalf("expected title match first, got %#v", hits[0])
	}
}
