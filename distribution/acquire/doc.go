// Package acquire translates a registry.Result into a Go module requirement.
//
// The acquisition plan is frozen: FromResult, Targets, Plan.GoGetArg. The Apply contract is frozen:
// Execute runs go get through ExecRunner, serialized per module root. Inspect
// reads go.mod / go.sum and does not take that lock. ExecuteTargets reports
// successful, failed, and unattempted targets (fail-fast). RecoverFiles
// restores a go.mod / go.sum snapshot (not transactional; cache not undone).
// No func Apply. Contract A is complete / frozen: DryRun reports the
// same GoGetArg without mutation. Contract B is implemented: package:acquire
// (console) orchestrates these APIs. Contract C is complete / frozen at v2.0.27:
// package:acquire --enable reuses enablePackage after successful acquisition;
// default acquire does not enable. There is no implicit transaction.
// package:install is not acquisition.
// Acquisition hardening SPEC (HARDENING.md) is accepted; implementation is COMPLETE.
package acquire
