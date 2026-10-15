package main

import "time"

type procSample struct {
	CPU     time.Duration
	RSS     uint64
	HWM     uint64
	Private uint64
}

// wsClass splits a resident working set. Private and Shareable do not
// overlap. Mapped is the MEM_MAPPED subset and is already inside one of those
// two.
type wsClass struct {
	Private   uint64
	Shareable uint64
	Mapped    uint64
}

func cpuMicrosPerReq(before, after procSample, requests int) float64 {
	if requests <= 0 {
		return 0
	}
	d := after.CPU - before.CPU
	if d < 0 {
		d = 0
	}
	return float64(d.Microseconds()) / float64(requests)
}
