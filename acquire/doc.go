// Package acquire translates a registry.Result into a Go module requirement.
//
// Phase 7 is frozen: FromResult, Targets, Plan.GoGetArg. Not enablement,
// not a lockfile, not package:install, and not Apply. This package does
// not run go get. Phase 8 starts with an Apply contract, not code.
package acquire
