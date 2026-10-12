//go:build linux

package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

func readProc(pid int) (procSample, error) {
	stat, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return procSample{}, err
	}
	s := string(stat)
	end := strings.LastIndex(s, ")")
	if end < 0 || end+2 >= len(s) {
		return procSample{}, fmt.Errorf("proc stat: no comm")
	}
	fields := strings.Fields(s[end+2:])
	// utime and stime are fields 14 and 15 of stat (1-based), which are
	// indexes 11 and 12 of the slice that starts at field 3.
	if len(fields) < 13 {
		return procSample{}, fmt.Errorf("proc stat: short")
	}
	utime, err := strconv.ParseInt(fields[11], 10, 64)
	if err != nil {
		return procSample{}, err
	}
	stime, err := strconv.ParseInt(fields[12], 10, 64)
	if err != nil {
		return procSample{}, err
	}
	ticks := unix.Sysconf(unix.SC_CLK_TCK)
	if ticks <= 0 {
		ticks = 100
	}
	cpu := time.Duration(utime+stime) * time.Second / time.Duration(ticks)

	status, err := os.ReadFile(fmt.Sprintf("/proc/%d/status", pid))
	if err != nil {
		return procSample{}, err
	}
	var rss, hwm uint64
	for _, line := range strings.Split(string(status), "\n") {
		if strings.HasPrefix(line, "VmRSS:") {
			rss = kbField(line)
		}
		if strings.HasPrefix(line, "VmHWM:") {
			hwm = kbField(line)
		}
	}
	return procSample{CPU: cpu, RSS: rss, HWM: hwm}, nil
}

func kbField(line string) uint64 {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return 0
	}
	n, _ := strconv.ParseUint(fields[1], 10, 64)
	return n * 1024
}
