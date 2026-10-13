//go:build !linux && !windows

package main

import (
	"fmt"
	"os/exec"
)

func setAffinity(pid int, mask uintptr) error {
	return fmt.Errorf("affinity is not implemented (pid %d mask %d)", pid, mask)
}

func startServer(exe, server, addr string, _ uintptr) *exec.Cmd {
	return exec.Command(exe, "serve", server, addr)
}
