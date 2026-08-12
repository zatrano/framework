# Browser HTTP feature tests

`core/browser` is a fluent wrapper around `core/testing.TestCase`. It issues real in-process HTTP requests against the application — **not** a headless browser or JavaScript DOM engine.

## Usage

```go
b, err := browser.New(app)
b.Visit("/login").
  AssertOK().
  Type("email", "a@b.c").
  Type("password", "secret").
  Press("Log in").
  AssertPathIs("/dashboard")
```

## Form submit (`Press`)

1. Locates the `<form>` that contains a matching submit control (`button` text / `name` / `value`, or `input type=submit`).
2. If none match, uses the first `<form>` on the page.
3. Reads `method` / `action` and default field values from HTML.
4. Merges values from `Type(...)` (overrides HTML defaults).
5. Submits with `TestCase.Post` (or GET when the form method is GET).

Optional second argument overrides the form action URL.

## Assertions

`AssertOK`, `AssertStatus`, `AssertSee`, `AssertDontSee`, `AssertPathIs` operate on the last HTTP response body and tracked path.
