package main

import "time"

type procSample struct {
	CPU time.Duration
	RSS uint64
	HWM uint64
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
