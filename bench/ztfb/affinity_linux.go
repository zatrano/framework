//go:build linux

package main

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"

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

// startServer pins the server with taskset. sched_setaffinity is the fallback.
func startServer(exe, server, addr string, mask uintptr) *exec.Cmd {
	cpus := cpuList(mask)
	if cpus != "" {
		if _, err := exec.LookPath("taskset"); err == nil {
			return exec.Command("taskset", "-c", cpus, exe, "serve", server, addr)
		}
	}
	return exec.Command(exe, "serve", server, addr)
}

func cpuList(mask uintptr) string {
	var parts []string
	for cpu := 0; cpu < 64; cpu++ {
		if mask&(1<<uintptr(cpu)) != 0 {
			parts = append(parts, strconv.Itoa(cpu))
		}
	}
	return strings.Join(parts, ",")
}
