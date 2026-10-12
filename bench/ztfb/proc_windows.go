//go:build windows

package main

import (
	"fmt"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modKernel32              = windows.NewLazySystemDLL("kernel32.dll")
	modPSAPI                 = windows.NewLazySystemDLL("psapi.dll")
	procSetAffinity          = modKernel32.NewProc("SetProcessAffinityMask")
	procGetProcessMemoryInfo = modPSAPI.NewProc("GetProcessMemoryInfo")
)

type processMemoryCounters struct {
	CB                         uint32
	PageFaultCount             uint32
	PeakWorkingSetSize         uintptr
	WorkingSetSize             uintptr
	QuotaPeakPagedPoolUsage    uintptr
	QuotaPagedPoolUsage        uintptr
	QuotaPeakNonPagedPoolUsage uintptr
	QuotaNonPagedPoolUsage     uintptr
	PagefileUsage              uintptr
	PeakPagefileUsage          uintptr
}

func readProc(pid int) (procSample, error) {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_INFORMATION|windows.PROCESS_VM_READ, false, uint32(pid))
	if err != nil {
		return procSample{}, err
	}
	defer windows.CloseHandle(h)

	var creation, exit, kernel, user windows.Filetime
	if err := windows.GetProcessTimes(h, &creation, &exit, &kernel, &user); err != nil {
		return procSample{}, err
	}
	cpu := time.Duration(filetime(kernel)+filetime(user)) * 100

	var mem processMemoryCounters
	mem.CB = uint32(unsafe.Sizeof(mem))
	r, _, callErr := procGetProcessMemoryInfo.Call(uintptr(h), uintptr(unsafe.Pointer(&mem)), uintptr(mem.CB))
	if r == 0 {
		return procSample{}, fmt.Errorf("GetProcessMemoryInfo: %w", callErr)
	}
	return procSample{
		CPU: cpu,
		RSS: uint64(mem.WorkingSetSize),
		HWM: uint64(mem.PeakWorkingSetSize),
	}, nil
}

func filetime(ft windows.Filetime) int64 {
	return int64(ft.HighDateTime)<<32 | int64(ft.LowDateTime)
}

func setAffinity(pid int, mask uintptr) error {
	h, err := windows.OpenProcess(windows.PROCESS_SET_INFORMATION|windows.PROCESS_QUERY_INFORMATION, false, uint32(pid))
	if err != nil {
		return err
	}
	defer windows.CloseHandle(h)
	r, _, callErr := procSetAffinity.Call(uintptr(h), mask)
	if r == 0 {
		return fmt.Errorf("affinity pid %d: %w", pid, callErr)
	}
	return nil
}
