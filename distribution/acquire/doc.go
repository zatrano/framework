// Package acquire translates a registry.Result into a Go module requirement.
//
// Phase 7 is frozen: FromResult, Targets, Plan.GoGetArg. Phase 8 Apply is
// SPEC-only (APPLY.md); this package does not run go get. package:install
// is not acquisition.
package acquire
