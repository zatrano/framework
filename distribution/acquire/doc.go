// Package acquire translates a registry.Result into a Go module requirement.
//
// Phase 7 is frozen: FromResult, Targets, Plan.GoGetArg. Phase 8 Execute
// runs go get through ExecRunner, serialized per module root. Inspect
// reads go.mod / go.sum and does not take that lock. package:install is
// not acquisition.
package acquire
