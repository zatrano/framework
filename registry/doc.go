// Package registry is the distribution index data model and resolution rules.
//
// It is not addons.Register, not a boot path, and not a marketplace.
// CLI, marketplace, and IDE tooling consume Search and Resolve; they must
// not own the algorithm. Channel "main" is a source stream, not a release.
package registry
