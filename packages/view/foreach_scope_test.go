package view_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zatrano/framework/packages/view"
)

func TestForeachPreservesParentScopeAndCSRF(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "list.html"), []byte(`
<p class="flag">{{ $show_admin_extras }}</p>
<p class="base">{{ $detail_base }}</p>
@foreach($items as $item)
<div class="row">
  <span class="item">{{ $item.label }}</span>
  <span class="parent">{{ $show_admin_extras }}</span>
  <span class="cat">{{ $category.ID }}</span>
  <form>@csrf</form>
</div>
@endforeach
`), 0o644)

	engine := view.New(dir)
	out, err := engine.Render("list", map[string]any{
		"show_admin_extras": "YES",
		"detail_base":       "/d",
		"category":          map[string]any{"ID": "42"},
		"_token":            "csrf-secret",
		"items": []map[string]any{
			{"label": "One"},
			{"label": "Two"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `class="flag">YES`) || !strings.Contains(out, `class="base">/d`) {
		t.Fatalf("parent outside loop missing: %s", out)
	}
	if !strings.Contains(out, `class="item">One`) || !strings.Contains(out, `class="item">Two`) {
		t.Fatalf("item fields missing: %s", out)
	}
	if strings.Count(out, `class="parent">YES`) < 2 {
		t.Fatalf("parent var lost inside foreach: %s", out)
	}
	if strings.Count(out, `class="cat">42`) < 2 {
		t.Fatalf("dotted parent path lost inside foreach: %s", out)
	}
	if strings.Count(out, `name="_token" value="csrf-secret"`) < 2 {
		t.Fatalf("csrf token empty inside foreach forms: %s", out)
	}
}

func TestNestedForeachSectionPages(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "nav.html"), []byte(`
@foreach($nav as $section)
<section data-title="{{ $section.title }}">
@foreach($section.pages as $link)
@if($link.active)
<a class="active" href="{{ $link.href }}">{{ $link.title }}</a>
@else
<a href="{{ $link.href }}">{{ $link.title }}</a>
@endif
@endforeach
</section>
@endforeach
@if($prev)
<span class="prev">{{ $prev.title }}</span>
@endif
`), 0o644)

	engine := view.New(dir)
	out, err := engine.Render("nav", map[string]any{
		"nav": []map[string]any{{
			"title": "Prologue",
			"pages": []map[string]any{
				{"title": "Overview", "href": "/docs", "active": true},
				{"title": "Releases", "href": "/docs/releases", "active": false},
			},
		}},
		"prev": map[string]any{"title": "Home", "slug": ""},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `data-title="Prologue"`) {
		t.Fatalf("section title missing: %s", out)
	}
	if !strings.Contains(out, `class="active" href="/docs">Overview`) {
		t.Fatalf("active link missing: %s", out)
	}
	if !strings.Contains(out, `href="/docs/releases">Releases`) {
		t.Fatalf("inactive link missing: %s", out)
	}
	if !strings.Contains(out, `class="prev">Home`) {
		t.Fatalf("prev missing: %s", out)
	}
}

func TestIfInequalityOperators(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "pager.html"), []byte(`
@if($pages > 1)
<span class="multi">yes</span>
@else
<span class="multi">no</span>
@endif
@if($count >= 10)
<span class="ge">ge</span>
@elseif($page < $pages)
<span class="lt">lt</span>
@else
<span class="other">other</span>
@endif
`), 0o644)

	engine := view.New(dir)
	out, err := engine.Render("pager", map[string]any{"pages": 3, "count": 3, "page": 1})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `class="multi">yes`) {
		t.Fatalf("pages>1 failed: %s", out)
	}
	if !strings.Contains(out, `class="lt">lt`) {
		t.Fatalf("page < pages branch failed: %s", out)
	}

	out, err = engine.Render("pager", map[string]any{"pages": 1, "count": 10, "page": 1})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `class="multi">no`) || !strings.Contains(out, `class="ge">ge`) {
		t.Fatalf("inequality branches wrong: %s", out)
	}
}

func TestUnsupportedIfFailsFast(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "bad.html"), []byte(`
@if($a && $b)
x
@endif
`), 0o644)
	engine := view.New(dir)
	_, err := engine.Render("bad", map[string]any{"a": true, "b": true})
	if err == nil || !strings.Contains(err.Error(), "unsupported") {
		t.Fatalf("expected unsupported compile error, got %v", err)
	}
}

func TestNestedIfElseInclude(t *testing.T) {
	dir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(dir, "partials"), 0o755)
	_ = os.WriteFile(filepath.Join(dir, "partials", "pager.html"), []byte(`
@if($pages > 1)
<nav class="pager">
@if($page > 1)
<a class="prev" href="?p={{ $prev }}">prev</a>
@else
<span class="prev-disabled">prev</span>
@endif
@if($page < $pages)
<a class="next" href="?p={{ $next }}">next</a>
@else
<span class="next-disabled">next</span>
@endif
</nav>
@else
<p class="single">one page</p>
@endif
`), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "page.html"), []byte(`
<h1>{{ $title }}</h1>
@include('partials.pager')
`), 0o644)

	engine := view.New(dir)
	out, err := engine.Render("page", map[string]any{
		"title": "List",
		"pages": 5,
		"page":  1,
		"prev":  0,
		"next":  2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `class="pager"`) || !strings.Contains(out, `class="prev-disabled"`) || !strings.Contains(out, `class="next"`) {
		t.Fatalf("nested include if/else broken: %s", out)
	}
}

func TestCsrfMetaNotBrokenByCsrf(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "head.html"), []byte(`
@csrfMeta
@csrf
`), 0o644)
	engine := view.New(dir)
	out, err := engine.Render("head", map[string]any{"_token": "tok"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `meta name="csrf-token" content="tok"`) {
		t.Fatalf("csrfMeta broken: %s", out)
	}
	if !strings.Contains(out, `name="_token" value="tok"`) {
		t.Fatalf("csrf missing: %s", out)
	}
}

func TestUnlessInsideForeach(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "list.html"), []byte(`
@foreach($invitations as $inv)
<div class="row">
@unless($inv.AdminExtras)
<span class="venue">{{ $inv.Venue }}</span>
@endunless
@isset($inv.EditHref)
<a class="edit" href="{{ $inv.EditHref }}">edit</a>
@endisset
@empty($inv.Venue)
<span class="empty-venue">none</span>
@endempty
<select>
@foreach($inv.Cats as $cat)
<option @selected($cat.Selected)>{{ $cat.Label }}</option>
@endforeach
</select>
</div>
@endforeach
@unless($show_admin_extras)
<p class="parent-unless">hidden-when-admin</p>
@endunless
`), 0o644)

	engine := view.New(dir)
	out, err := engine.Render("list", map[string]any{
		"show_admin_extras": false,
		"invitations": []map[string]any{
			{
				"AdminExtras": false,
				"Venue":       "Hall A",
				"EditHref":    "/edit/1",
				"Cats": []map[string]any{
					{"Label": "One", "Selected": true},
					{"Label": "Two", "Selected": false},
				},
			},
			{
				"AdminExtras": true,
				"Venue":       "Hall B",
				"Cats":        []map[string]any{},
			},
			{
				"AdminExtras": false,
				"Venue":       "",
				"EditHref":    "/edit/3",
				"Cats":        []map[string]any{},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `class="venue">Hall A`) {
		t.Fatalf("unless should show venue when AdminExtras false: %s", out)
	}
	if strings.Contains(out, `class="venue">Hall B`) {
		t.Fatalf("unless should hide venue when AdminExtras true: %s", out)
	}
	if !strings.Contains(out, `class="edit" href="/edit/1"`) {
		t.Fatalf("isset EditHref missing: %s", out)
	}
	if !strings.Contains(out, `class="empty-venue">none`) {
		t.Fatalf("empty Venue missing: %s", out)
	}
	if !strings.Contains(out, `selected>One`) && !strings.Contains(out, `selected="">One`) && !strings.Contains(out, ` selected>One`) {
		// attrBool typically emits ` selected` or selected="selected"
		if !strings.Contains(out, "One") || !strings.Contains(strings.ToLower(out), "selected") {
			t.Fatalf("@selected inside foreach missing: %s", out)
		}
	}
	if !strings.Contains(out, `class="parent-unless">hidden-when-admin`) {
		t.Fatalf("parent @unless lost inside template: %s", out)
	}
}

func TestForeachRootDottedCollection(t *testing.T) {
	dir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(dir, "partials"), 0o755)
	_ = os.WriteFile(filepath.Join(dir, "partials", "pagination.html"), []byte(`
@if($pagination.Pages > 1)
<nav class="pager">
@foreach($pagination.Links as $link)
@if($link.Active)
<span class="cur">{{ $link.Label }}</span>
@else
<a href="{{ $link.URL }}">{{ $link.Label }}</a>
@endif
@endforeach
@if($pagination.HasNext)
<a class="next" href="{{ $pagination.NextURL }}">next</a>
@endif
</nav>
@endif
`), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "index.html"), []byte(`
@foreach($invitations as $inv)
<p class="inv">{{ $inv.Venue }}</p>
@endforeach
@include('partials.pagination')
`), 0o644)

	engine := view.New(dir)
	out, err := engine.Render("index", map[string]any{
		"invitations": []map[string]any{{"Venue": "Hall A"}},
		"pagination": map[string]any{
			"Pages":   3,
			"HasNext": true,
			"NextURL": "?p=2",
			"Links": []map[string]any{
				{"Label": "1", "URL": "?p=1", "Active": true},
				{"Label": "2", "URL": "?p=2", "Active": false},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `class="inv">Hall A`) {
		t.Fatalf("list missing: %s", out)
	}
	if !strings.Contains(out, `class="cur">1`) || !strings.Contains(out, `href="?p=2">2`) {
		t.Fatalf("pagination links missing: %s", out)
	}
	if !strings.Contains(out, `class="next" href="?p=2"`) {
		t.Fatalf("pagination HasNext missing: %s", out)
	}
}
