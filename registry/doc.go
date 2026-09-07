// Package registry is the distribution index data model and resolution rules.
//
// It is not addons.Register, not a boot path, and not a marketplace.
// CLI, marketplace, and IDE tooling consume Search, Lookup, and Resolve.
// They must not copy the version-selection algorithm. Channel "main" is a
// source stream, not a published release.
package registry
