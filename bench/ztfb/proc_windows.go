//go:build windows

package main

import (
	"fmt"
	"reflect"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modKernel32              = windows.NewLazySystemDLL("kernel32.dll")
	modPSAPI                 = windows.NewLazySystemDLL("psapi.dll")
	procSetAffinity          = modKernel32.NewProc("SetProcessAffinityMask")
	procGetProcessMemoryInfo = modPSAPI.NewProc("GetProcessMemoryInfo")
	procQueryWorkingSet      = modPSAPI.NewProc("QueryWorkingSet")
	procVirtualQueryEx       = modKernel32.NewProc("VirtualQueryEx")
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
	mem.CB = uint32(reflect.TypeOf(mem).Size())
	r, _, callErr := procGetProcessMemoryInfo.Call(uintptr(h), reflect.ValueOf(&mem).Pointer(), uintptr(mem.CB))
	if r == 0 {
		return procSample{}, fmt.Errorf("GetProcessMemoryInfo: %w", callErr)
	}
	return procSample{
		CPU:     cpu,
		RSS:     uint64(mem.WorkingSetSize),
		HWM:     uint64(mem.PeakWorkingSetSize),
		Private: uint64(mem.PagefileUsage),
	}, nil
}

const (
	memCommit = 0x1000
	memMapped = 0x40000
	pageSize  = 4096
)

// memoryBasicInformation matches the amd64 MEMORY_BASIC_INFORMATION layout.
type memoryBasicInformation struct {
	BaseAddress       uintptr
	AllocationBase    uintptr
	AllocationProtect uint32
	PartitionId       uint16
	_                 uint16
	RegionSize        uintptr
	State             uint32
	Protect           uint32
	Type              uint32
	_                 uint32
}

// readWorkingSetClass walks the resident pages with QueryWorkingSet and
// classifies each with VirtualQueryEx. GetProcessMemoryInfo supplies the
// private commit separately, via procSample.Private.
func readWorkingSetClass(pid int) (wsClass, error) {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_INFORMATION|windows.PROCESS_VM_READ, false, uint32(pid))
	if err != nil {
		return wsClass{}, err
	}
	defer windows.CloseHandle(h)

	buf := make([]uintptr, 1<<16)
	for {
		r, _, callErr := procQueryWorkingSet.Call(uintptr(h), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)*int(unsafe.Sizeof(buf[0]))))
		if r != 0 {
			break
		}
		errno, _ := callErr.(windows.Errno)
		if errno != windows.ERROR_BAD_LENGTH && errno != windows.ERROR_INSUFFICIENT_BUFFER {
			return wsClass{}, fmt.Errorf("QueryWorkingSet: %w", callErr)
		}
		need := int(buf[0]) + 1
		if need < len(buf)*2 {
			need = len(buf) * 2
		}
		if need > 1<<24 {
			return wsClass{}, fmt.Errorf("QueryWorkingSet: %d entries", need)
		}
		buf = make([]uintptr, need)
	}
	n := int(buf[0])
	if n < 0 || n+1 > len(buf) {
		return wsClass{}, fmt.Errorf("QueryWorkingSet: count %d", n)
	}
	var out wsClass
	var region memoryBasicInformation
	for _, e := range buf[1 : 1+n] {
		addr := e &^ (pageSize - 1)
		shared := (e>>8)&1 == 1
		if addr < region.BaseAddress || addr >= region.BaseAddress+region.RegionSize || region.State != memCommit {
			r, _, callErr := procVirtualQueryEx.Call(uintptr(h), addr, uintptr(unsafe.Pointer(&region)), unsafe.Sizeof(region))
			if r == 0 {
				return wsClass{}, fmt.Errorf("VirtualQueryEx %x: %w", addr, callErr)
			}
		}
		if region.Type == memMapped {
			out.Mapped += pageSize
		}
		if shared {
			out.Shareable += pageSize
		} else {
			out.Private += pageSize
		}
	}
	return out, nil
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
