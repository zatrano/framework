package view_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zatrano/framework/packages/view"
)

func TestLayoutsExtendsAndYield(t *testing.T) {
	dir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(dir, "layouts"), 0o755)
	_ = os.MkdirAll(filepath.Join(dir, "partials"), 0o755)

	_ = os.WriteFile(filepath.Join(dir, "layouts", "app.html"), []byte(`<html><title>@yield('title', 'Default')</title><body>@include('partials.brand')@yield('content')</body></html>`), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "partials", "brand.html"), []byte(`<div>ZATRANO</div>`), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "home.html"), []byte(`@extends('layouts.app')
@section('title', 'Home')
@section('content')
<p>{{ $message }}</p>
@endsection
`), 0o644)

	engine := view.New(dir)
	out, err := engine.Render("home", map[string]any{"message": "Hello"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "<title>Home</title>") {
		t.Fatalf("expected title, got %s", out)
	}
	if !strings.Contains(out, "<div>ZATRANO</div>") {
		t.Fatalf("expected brand include, got %s", out)
	}
	if !strings.Contains(out, "Hello") {
		t.Fatalf("expected message, got %s", out)
	}
}

func TestSectionShortVariable(t *testing.T) {
	dir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(dir, "layouts"), 0o755)
	_ = os.WriteFile(filepath.Join(dir, "layouts", "app.html"), []byte(`<html><title>@yield('title', 'Default') — App</title>@yield('content')</html>`), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "page.html"), []byte(`@extends('layouts.app')
@section('title', $pageTitle)
@section('content')
ok
@endsection
`), 0o644)

	engine := view.New(dir)
	out, err := engine.Render("page", map[string]any{"pageTitle": "Davetiyelerim"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "<title>Davetiyelerim — App</title>") {
		t.Fatalf("expected variable section title, got %s", out)
	}
}

func TestViewStacks(t *testing.T) {
	dir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(dir, "layouts"), 0o755)
	_ = os.WriteFile(filepath.Join(dir, "layouts", "app.html"), []byte(`<html>@yield('content')@stack('scripts')</html>`), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "page.html"), []byte(`@extends('layouts.app')
@section('content')
<body>ok</body>
@endsection
@prepend('scripts')
PRE
@endprepend
@push('scripts')
PUSH
@endpush
`), 0o644)

	engine := view.New(dir)
	out, err := engine.Render("page", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "PRE") || !strings.Contains(out, "PUSH") {
		t.Fatalf("stacks missing: %s", out)
	}
	if idxPre, idxPush := strings.Index(out, "PRE"), strings.Index(out, "PUSH"); idxPre < 0 || idxPush < idxPre {
		t.Fatalf("prepend should come before push: %s", out)
	}
}
