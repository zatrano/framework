//go:build windows

package main

import "os/exec"

// setAffinity is implemented in proc_windows.go next to the other process calls.

func startServer(exe, server, addr string, _ uintptr) *exec.Cmd {
	return exec.Command(exe, "serve", server, addr)
}
