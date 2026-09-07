// Package acquire translates a registry.Result into a Go module requirement.
//
// Phase 7 is frozen: FromResult maps a Result; Targets deduplicates modules.
// It is not enablement, not a lockfile, not package:install, and not Apply.
// Pins live in go.mod / go.sum after a later Phase 8 Apply. This package
// does not run go get.
package acquire
