// Package acquire translates a registry.Result into a Go module requirement.
//
// It is not enablement, not a lockfile, and not package:install.
// Pins live in go.mod / go.sum after a later Apply. This package only
// builds the plan (zatrano.acquire/v1).
package acquire
