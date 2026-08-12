package pagination_test

import (
	"testing"

	"github.com/zatrano/framework/packages/pagination"
)

func TestPaginator(t *testing.T) {
	items := []string{"a", "b"}
	p := pagination.New(items, 20, 2, 2, "/api/users")
	if p.CurrentPage != 2 || p.LastPage != 10 || p.From != 3 || p.To != 4 {
		t.Fatalf("unexpected paginator: %#v", p)
	}
	if p.NextPageURL == "" || p.PrevPageURL == "" {
		t.Fatal("expected next/prev urls")
	}
}
