package view_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zatrano/framework/packages/view"
)

func TestIfNumericComparison(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "num.html"), []byte(`
@if($limit == 100)
<span class="eq">100</span>
@elseif($limit == 50)
<span class="eq">50</span>
@else
<span class="eq">other</span>
@endif
@if($limit != 0)
<span class="nz">yes</span>
@endif
`), 0o644)

	engine := view.New(dir)
	out, err := engine.Render("num", map[string]any{"limit": 100})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `class="eq">100`) {
		t.Fatalf("expected limit==100 branch, got %s", out)
	}
	if !strings.Contains(out, `class="nz">yes`) {
		t.Fatalf("expected limit!=0 branch, got %s", out)
	}

	out, err = engine.Render("num", map[string]any{"limit": 50})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `class="eq">50`) {
		t.Fatalf("expected limit==50 branch, got %s", out)
	}
}
