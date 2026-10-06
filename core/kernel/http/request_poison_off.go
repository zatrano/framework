//go:build !zatrano_poison && !rawhttp_poison && !canvas_poison

package http

// requestFreedPoison is false in production binaries. The compiler deletes
// poisonCheck and markFreed, including the panic string.
const requestFreedPoison = false
