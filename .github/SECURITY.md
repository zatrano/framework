# Security Policy

## Supported Versions

| Version | Supported |
| ------- | --------- |
| 3.2.x   | ✅ |
| < 3.2   | ❌ |

## Build toolchain

The module declares `go 1.25.0`. That is the language floor. It does not select the standard library that ends up in a binary: the toolchain that compiles the binary does.

Build production binaries and images with Go >= `go1.26.9` (`RecommendedMinimum` in `core/kernel/goversion/goversion.go`). The 2026-10-08 fixes in `crypto/tls`, `html/template`, `net/http`, `net/textproto`, and `os` are in go1.26.9 and go1.27.2. Go 1.25's last release is go1.25.14 (2026-08-19). [Go's release policy](https://go.dev/doc/devel/release) supports a major until two newer majors exist, so Go 1.25 left support when Go 1.27.0 shipped that day.

Distribution packages often lag. `apt install golang-go` on Ubuntu usually installs a release below go1.26.9. Build with the official toolchain from [go.dev/dl](https://go.dev/dl/), or with `GOTOOLCHAIN=go1.26.9` (needs Go 1.21 or newer on `PATH`). Check `go version` in the build environment, including CI runners.

`zatrano doctor` reports this as APP-GO-001 (warning) when `go env GOVERSION`, run in the project directory, is older than `RecommendedMinimum`. The rule is skipped when `go` is not on `PATH`.

The built binary checks itself too. With `APP_ENV=production`, boot logs a warning when `runtime.Version()` is older than `RecommendedMinimum`. A binary built from apt's Go therefore reports the problem on the server, where `doctor` does not run.

Images use `golang:1.26-alpine` (that tag follows the 1.26 patch releases) and `alpine:3.24`. Alpine 3.20 support ended on 2026-04-01 ([Alpine release branches](https://alpinelinux.org/releases/)).

## Release checklist

Review `RecommendedMinimum` on every release. The release workflow reads that constant and [go.dev/dl/?mode=json](https://go.dev/dl/?mode=json). It fails when the constant cannot be read. It warns, and does not fail the release, when the constant is more than two patches behind the newest stable release on the same minor, or when that minor is no longer published.

## Reporting a Vulnerability

Email **Serhan KARAKOÇ** at [serhankarakoc@zatrano.com](mailto:serhankarakoc@zatrano.com).

Do **not** open a public GitHub issue for security vulnerabilities.

Please include:

* ZATRANO version / commit
* Affected package and API
* Reproduction steps (PoC)
* Impact assessment

We aim to acknowledge reports within a few business days and ship fixes promptly.

## Documentation

Full security guides live on the docs site (not in this repository):

* [Security overview](https://zatrano.com/docs/security)
* [Security testing](https://zatrano.com/docs/security-testing)
* [Security audit](https://zatrano.com/docs/security-audit)
* [Security report](https://zatrano.com/docs/security-report)
