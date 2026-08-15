package view_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zatrano/framework/packages/view"
)

func TestAdvancedViewDirectives(t *testing.T) {
	dir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(dir, "partials"), 0o755)
	_ = os.WriteFile(filepath.Join(dir, "partials", "row.html"), []byte(`<li>{{ $item.title }}</li>`), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "partials", "empty.html"), []byte(`<p>none</p>`), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "partials", "optional.html"), []byte(`<span>optional</span>`), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "page.html"), []byte(`
@includeIf('partials.optional')
@includeIf('partials.missing')
@isset($name)
<p>hi {{ $name }}</p>
@endisset
@empty($missing)
<p>missing-empty</p>
@endempty
@once
<script>once()</script>
@endonce
@once
<script>once()</script>
@endonce
@each('partials.row', $items, 'item', 'partials.empty')
@verbatim
{{ $raw }}
@endverbatim
`), 0o644)

	engine := view.New(dir)
	out, err := engine.Render("page", map[string]any{
		"name": "Ada",
		"items": []map[string]any{
			{"title": "One"},
			{"title": "Two"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "optional") {
		t.Fatalf("includeIf missing: %s", out)
	}
	if strings.Contains(out, "includeIf error") || strings.Contains(out, "partials.missing") {
		t.Fatalf("missing includeIf should be silent: %s", out)
	}
	if !strings.Contains(out, "hi Ada") || !strings.Contains(out, "missing-empty") {
		t.Fatalf("isset/empty missing: %s", out)
	}
	if strings.Count(out, "once()") != 1 {
		t.Fatalf("once should render once: %s", out)
	}
	if !strings.Contains(out, "<li>One</li>") || !strings.Contains(out, "<li>Two</li>") {
		t.Fatalf("each missing rows: %s", out)
	}
	if !strings.Contains(out, "{{ $raw }}") {
		t.Fatalf("verbatim should keep raw braces: %s", out)
	}

	emptyOut, err := engine.Render("page", map[string]any{
		"name":  "Ada",
		"items": []map[string]any{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(emptyOut, "none") {
		t.Fatalf("each empty partial missing: %s", emptyOut)
	}
}

func TestExtendedViewDirectives(t *testing.T) {
	dir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(dir, "partials"), 0o755)
	_ = os.MkdirAll(filepath.Join(dir, "components"), 0o755)
	_ = os.MkdirAll(filepath.Join(dir, "layouts"), 0o755)

	_ = os.WriteFile(filepath.Join(dir, "partials", "card.html"), []byte(`<div class="card">{{ $title }}</div>`), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "partials", "note.html"), []byte(`<em>{{ $text }}</em>`), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "partials", "a.html"), []byte(`<span>A</span>`), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "partials", "row.html"), []byte(`<li>{{ $item.label }}</li>`), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "components", "box.html"), []byte(`<div class="box {{ $type }}"><strong>{{ $title }}</strong>{{ $slot }}</div>`), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "layouts", "base.html"), []byte(`
<html>
@hasSection('sidebar')
<aside>@yield('sidebar')</aside>
@endif
@sectionMissing('sidebar')
<p class="no-side">none</p>
@endif
<main>@yield('content')</main>
@stack('scripts', '/*default*/')
</html>
`), 0o644)

	_ = os.WriteFile(filepath.Join(dir, "page.html"), []byte(`
@extends('layouts.base')
@section('content')
<p class="nested">{{ $user.profile.name }}</p>
{!! $html.raw !!}
<script type="application/json">@json($payload)</script>
<div @class($classes) @style($styles)></div>
<input type="checkbox" @checked($active)>
<option @selected($picked)>x</option>
<button @disabled($locked) @readonly($locked) @required($needed)>go</button>
@if($status == 'ok')
<span class="ok">yes</span>
@elseif($status == 'pending')
<span class="pending">wait</span>
@else
<span class="other">no</span>
@endif
@unless($hidden)
<span class="visible">show</span>
@endunless
<ul>
@foreach($users as $user)
<li class="foreach">{{ $user.name }}</li>
@endforeach
</ul>
<ul class="forelse">
@forelse($tags as $tag)
<li>{{ $tag.label }}</li>
@empty
<li class="empty-tags">none</li>
@endforelse
</ul>
@switch($mode)
@case('a')
<span class="sw-a">A</span>
@break
@case('b')
<span class="sw-b">B</span>
@break
@default
<span class="sw-d">D</span>
@endswitch
@include('partials.card', ['title' => 'Included'])
@includeWhen($showNote, 'partials.note', ['text' => 'Note'])
@includeUnless($showNote, 'partials.a')
@includeFirst(['partials.missing', 'partials.a'])
@each('partials.row', $items, 'item')
@component('box', ['type' => 'info'])
@slot('title') Hello @endslot
Body
@endcomponent
@lang('messages.welcome')
@pushOnce('scripts')
<script>oncePush()</script>
@endPushOnce
@pushOnce('scripts')
<script>oncePush()</script>
@endPushOnce
@endsection
@section('sidebar')
<nav>side</nav>
@endsection
`), 0o644)

	engine := view.New(dir)
	engine.AddFunc("trans", func(key string) string {
		if key == "messages.welcome" {
			return "Welcome"
		}
		return key
	})

	out, err := engine.Render("page", map[string]any{
		"user":     map[string]any{"profile": map[string]any{"name": "Ada"}},
		"html":     map[string]any{"raw": "<b>raw</b>"},
		"payload":  map[string]any{"n": 1},
		"classes":  map[string]any{"active": true, "hidden": false},
		"styles":   map[string]any{"color:red": true, "display:none": false},
		"active":   true,
		"picked":   true,
		"locked":   true,
		"needed":   true,
		"status":   "ok",
		"hidden":   false,
		"users":    []map[string]any{{"name": "Ann"}, {"name": "Bob"}},
		"tags":     []map[string]any{},
		"mode":     "b",
		"showNote": true,
		"items": []map[string]any{
			{"label": "One"},
			{"label": "Two"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	checks := []string{
		`class="nested">Ada`,
		`<b>raw</b>`,
		`{"n":1}`,
		`class="active"`,
		`style="color:red"`,
		`checked`,
		`selected`,
		`disabled`,
		`readonly`,
		`required`,
		`class="ok">yes`,
		`class="visible">show`,
		`class="foreach">Ann`,
		`class="foreach">Bob`,
		`class="empty-tags">none`,
		`class="sw-b">B`,
		`class="card">Included`,
		`<em>Note</em>`,
		`<li>One</li>`,
		`<li>Two</li>`,
		`class="box info"`,
		`<strong>Hello</strong>`,
		`Body`,
		`Welcome`,
		`<aside>`,
		`<nav>side</nav>`,
	}
	for _, want := range checks {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
	if strings.Count(out, "oncePush()") != 1 {
		t.Fatalf("pushOnce count: %s", out)
	}
	if strings.Contains(out, "no-side") {
		t.Fatalf("sectionMissing should be hidden: %s", out)
	}
	if !strings.Contains(out, "<span>A</span>") {
		t.Fatalf("includeFirst should pick partials.a: %s", out)
	}
	if strings.Count(out, "<span>A</span>") != 1 {
		t.Fatalf("includeUnless should not add extra A: %s", out)
	}
}

func TestSectionMissingAndIncludeFirst(t *testing.T) {
	dir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(dir, "layouts"), 0o755)
	_ = os.MkdirAll(filepath.Join(dir, "partials"), 0o755)
	_ = os.WriteFile(filepath.Join(dir, "partials", "a.html"), []byte(`<span>A</span>`), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "layouts", "base.html"), []byte(`
@sectionMissing('sidebar')
<p class="no-side">none</p>
@endif
@hasSection('sidebar')
<aside>@yield('sidebar')</aside>
@endif
@yield('content')
`), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "page.html"), []byte(`
@extends('layouts.base')
@section('content')
@includeFirst(['partials.missing', 'partials.a'])
@endsection
`), 0o644)

	engine := view.New(dir)
	out, err := engine.Render("page", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "no-side") || !strings.Contains(out, "<span>A</span>") {
		t.Fatalf("out=%s", out)
	}
	if strings.Contains(out, "<aside>") {
		t.Fatalf("hasSection should be empty: %s", out)
	}
}

func TestParentDirective(t *testing.T) {
	dir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(dir, "layouts"), 0o755)

	_ = os.WriteFile(filepath.Join(dir, "layouts", "base.html"), []byte(`
@yield('content', 'Base')
`), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "layouts", "middle.html"), []byte(`
@extends('layouts.base')
@section('content')
Middle @parent
@endsection
`), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "page.html"), []byte(`
@extends('layouts.middle')
@section('content')
Page @parent
@endsection
`), 0o644)

	engine := view.New(dir)
	out, err := engine.Render("page", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Page Middle Base") {
		t.Fatalf("expected nested @parent chain, got: %s", out)
	}
}

func TestParentWithSectionShow(t *testing.T) {
	dir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(dir, "layouts"), 0o755)

	_ = os.WriteFile(filepath.Join(dir, "layouts", "app.html"), []byte(`
<title>@yield('title')</title>
@section('title')
App Title
@show
<main>@yield('content')</main>
`), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "page.html"), []byte(`
@extends('layouts.app')
@section('title')
Page @parent
@endsection
@section('content')
<body>ok</body>
@endsection
`), 0o644)

	engine := view.New(dir)
	out, err := engine.Render("page", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Page App Title") {
		t.Fatalf("expected @parent from @section/@show, got: %s", out)
	}
	if !strings.Contains(out, "<body>ok</body>") {
		t.Fatalf("content section missing: %s", out)
	}
}

func TestPropsDirectiveOnComponent(t *testing.T) {
	dir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(dir, "components"), 0o755)
	_ = os.WriteFile(filepath.Join(dir, "components", "alert.html"), []byte(`
@props(['type' => 'info'])
<div class="alert {{ $type }}">{{ $message }}</div>
`), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "page.html"), []byte(`
@component('alert', ['message' => 'Hello'])
@endcomponent
`), 0o644)

	engine := view.New(dir)
	out, err := engine.Render("page", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `class="alert info"`) || !strings.Contains(out, "Hello") {
		t.Fatalf("props defaults missing: %s", out)
	}
}

func TestCanDirective(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "page.html"), []byte(`
@can('edit')
<span class="can-edit">yes</span>
@endcan
@cannot('delete')
<span class="cannot-delete">no-delete</span>
@endcannot
@can('edit', $user)
<span class="can-user">user-edit</span>
@endcan
`), 0o644)

	engine := view.New(dir)
	out, err := engine.Render("page", map[string]any{"user": map[string]any{"id": 1}})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "can-edit") || strings.Contains(out, "user-edit") {
		t.Fatalf("default can should be false: %s", out)
	}
	if !strings.Contains(out, "cannot-delete") {
		t.Fatalf("cannot should render when can is false: %s", out)
	}

	engine.ClearCache()
	out, err = engine.Render("page", map[string]any{
		"user": map[string]any{"id": 1},
		"__can": func(ability string, _ ...any) bool {
			return ability == "edit"
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "can-edit") || !strings.Contains(out, "user-edit") {
		t.Fatalf("request-scoped __can missing: %s", out)
	}

	engine.AddFunc("can", func(data map[string]any, ability string, args ...any) bool {
		return ability == "edit"
	})
	engine.ClearCache()
	out, err = engine.Render("page", map[string]any{"user": map[string]any{"id": 1}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "can-edit") || !strings.Contains(out, "user-edit") {
		t.Fatalf("custom can missing: %s", out)
	}
}

func TestEnvAndProductionDirectives(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "page.html"), []byte(`
@env('local')
<span class="local">local</span>
@endenv
@env('production')
<span class="prod-env">prod-env</span>
@endenv
@production
<span class="production">production</span>
@endproduction
`), 0o644)

	engine := view.New(dir)
	out, err := engine.Render("page", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "local") {
		t.Fatalf("default env local missing: %s", out)
	}
	if strings.Contains(out, "prod-env") || strings.Contains(out, `class="production"`) {
		t.Fatalf("production blocks should be hidden on local: %s", out)
	}

	engine.SetEnvironment("production")
	engine.ClearCache()
	out, err = engine.Render("page", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "prod-env") || !strings.Contains(out, `class="production"`) {
		t.Fatalf("production env missing: %s", out)
	}
	if strings.Contains(out, `class="local"`) {
		t.Fatalf("local block should be hidden on production: %s", out)
	}
}

func TestCustomDirective(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "page.html"), []byte(`
@datetime('2026-01-01')
@badge
`), 0o644)

	engine := view.New(dir)
	engine.Directive("datetime", func(args string) string {
		return "<time>" + strings.Trim(args, `'"`) + "</time>"
	})
	engine.Directive("badge", func(args string) string {
		return `<span class="badge">` + args + `</span>`
	})

	out, err := engine.Render("page", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "<time>2026-01-01</time>") {
		t.Fatalf("datetime directive missing: %s", out)
	}
	if !strings.Contains(out, `<span class="badge"></span>`) {
		t.Fatalf("bare directive missing: %s", out)
	}
}

func TestPhpDirectiveStripped(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "page.html"), []byte(`
@php echo 'secret'; @endphp
<p>ok</p>
`), 0o644)

	engine := view.New(dir)
	out, err := engine.Render("page", nil)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "secret") {
		t.Fatalf("php should not execute: %s", out)
	}
	if !strings.Contains(out, "ok") {
		t.Fatalf("content after php missing: %s", out)
	}
}

func TestClearCache(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "x.html")
	_ = os.WriteFile(path, []byte(`v1`), 0o644)
	engine := view.New(dir)
	engine.EnableCache(true)
	out, err := engine.Render("x", nil)
	if err != nil || out != "v1" {
		t.Fatalf("first=%q err=%v", out, err)
	}
	_ = os.WriteFile(path, []byte(`v2`), 0o644)
	out, _ = engine.Render("x", nil)
	if out != "v1" {
		t.Fatalf("cached expected v1 got %q", out)
	}
	engine.ClearCache()
	out, err = engine.Render("x", nil)
	if err != nil || out != "v2" {
		t.Fatalf("after clear=%q err=%v", out, err)
	}
}

func TestXTagsAttributesAndAware(t *testing.T) {
	dir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(dir, "components"), 0o755)
	_ = os.WriteFile(filepath.Join(dir, "components", "alert.html"), []byte(`
@props(['type' => 'info'])
<div {{ $attributes }} class="alert-{{ $type }}">{{ $slot }}</div>
`), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "components", "badge.html"), []byte(`
@aware(['type'])
<span class="badge-{{ $type }}">{{ $slot }}</span>
`), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "components", "panel.html"), []byte(`
@props(['type' => 'info'])
<div class="panel">
<x-badge>inner</x-badge>
</div>
`), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "page.html"), []byte(`
<x-alert type="danger" class="flash" id="a1">Hello</x-alert>
<x-panel type="success" />
`), 0o644)

	engine := view.New(dir)
	out, err := engine.Render("page", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `class="flash"`) || !strings.Contains(out, `id="a1"`) {
		t.Fatalf("attributes missing: %s", out)
	}
	if !strings.Contains(out, "alert-danger") || !strings.Contains(out, "Hello") {
		t.Fatalf("x-alert missing: %s", out)
	}
	if !strings.Contains(out, "badge-success") {
		t.Fatalf("@aware type not inherited: %s", out)
	}
}

func TestForeachAliasAndOnceKey(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "page.html"), []byte(`
@foreach($items as $item)
<li>{{ $item.label }}</li>
@endforeach
@foreach($items as $i => $item)
<li data-i="{{ $i }}">{{ $item.label }}</li>
@endforeach
@once('k')
<span class="once-a">A</span>
@endonce
@once('k')
<span class="once-b">B</span>
@endonce
`), 0o644)

	engine := view.New(dir)
	out, err := engine.Render("page", map[string]any{
		"items": []map[string]any{{"label": "One"}, {"label": "Two"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "One") || !strings.Contains(out, "Two") {
		t.Fatalf("foreach alias missing: %s", out)
	}
	if !strings.Contains(out, `data-i="0"`) || !strings.Contains(out, `data-i="1"`) {
		t.Fatalf("foreach key alias missing: %s", out)
	}
	if !strings.Contains(out, "once-a") || strings.Contains(out, "once-b") {
		t.Fatalf("named once failed: %s", out)
	}
}

func TestErrorNamedBag(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "page.html"), []byte(`
@error('email', 'login')
<span class="login-err">bad</span>
@enderror
@error('email')
<span class="default-err">also</span>
@enderror
`), 0o644)

	engine := view.New(dir)
	out, err := engine.Render("page", map[string]any{
		"errors": map[string][]string{},
		"errorBags": map[string]any{
			"login": map[string][]string{"email": {"x"}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "login-err") {
		t.Fatalf("named bag missing: %s", out)
	}
	if strings.Contains(out, "default-err") {
		t.Fatalf("default bag should be empty: %s", out)
	}
}
