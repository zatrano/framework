//go:build linux

package main

import (
	"fmt"

	"golang.org/x/sys/unix"
)

func setAffinity(pid int, mask uintptr) error {
	var set unix.CPUSet
	for cpu := 0; cpu < 64; cpu++ {
		if mask&(1<<uintptr(cpu)) != 0 {
			set.Set(cpu)
		}
	}
	if err := unix.SchedSetaffinity(pid, &set); err != nil {
		return fmt.Errorf("affinity pid %d: %w", pid, err)
	}
	return nil
}
