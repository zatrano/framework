//go:build zatrano_poison || rawhttp_poison || canvas_poison

package http

// requestFreedPoison is true for -tags zatrano_poison and also when
// rawhttp_poison or canvas_poison is set.
const requestFreedPoison = true
