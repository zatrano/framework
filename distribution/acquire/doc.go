// Package acquire translates a registry.Result into a Go module requirement.
//
// Phase 7 is frozen: FromResult, Targets, Plan.GoGetArg. Phase 8 has an
// invocation boundary (Invoke + Runner). It does not run go get unless a
// Runner does. package:install is not acquisition.
package acquire
