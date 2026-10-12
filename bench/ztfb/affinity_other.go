//go:build !linux && !windows

package main

import "fmt"

func setAffinity(pid int, mask uintptr) error {
	return fmt.Errorf("affinity is not implemented (pid %d mask %d)", pid, mask)
}
