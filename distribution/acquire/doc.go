// Package acquire translates a registry.Result into a Go module requirement.
//
// Phase 7 is frozen: FromResult, Targets, Plan.GoGetArg. Phase 8 is frozen:
// Execute runs go get through ExecRunner, serialized per module root. Inspect
// reads go.mod / go.sum and does not take that lock. ExecuteTargets reports
// successful, failed, and unattempted targets (fail-fast). RecoverFiles
// restores a go.mod / go.sum snapshot (not transactional; cache not undone).
// No func Apply. Phase 9 Contract A is complete / frozen: DryRun reports the
// same GoGetArg without mutation. Contract B is implemented: package:acquire
// (console) orchestrates these APIs. Contract C is open: package:acquire
// --enable reuses enablePackage after successful acquisition; default acquire
// does not enable. There is no implicit transaction. package:install is not acquisition.
package acquire
