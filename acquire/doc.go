// Package acquire translates a registry.Result into a Go module requirement.
//
// Phase 7 Plan layer is frozen: FromResult is a pure mapping; Targets collapses
// shared modules into a deterministic query list. It is not enablement, not a
// lockfile, and not package:install. Pins live in go.mod / go.sum after a later
// Apply. This package does not run go get.
package acquire
