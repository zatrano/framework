//go:build !linux && !windows

package main

import "fmt"

func readProc(pid int) (procSample, error) {
	return procSample{}, fmt.Errorf("process counters are not implemented for this OS (pid %d)", pid)
}

func readWorkingSetClass(pid int) (wsClass, error) {
	return wsClass{}, fmt.Errorf("working set class is not implemented for this OS (pid %d)", pid)
}
