package main

import (
	"fmt"
	"math/rand"
	"sort"
)

const (
	defaultRounds  = 10
	bootstrapDraws = 10000
	bootstrapSeed  = 20261013
)

// pairedRPSRatios returns ZATRANO/Fiber throughput for each shared round.
// The cell ratio is the median of these pairs, not the ratio of the medians.
func pairedRPSRatios(zatrano, fiber []cell) ([]float64, error) {
	if len(zatrano) == 0 || len(zatrano) != len(fiber) {
		return nil, fmt.Errorf("paired rounds %d and %d", len(zatrano), len(fiber))
	}
	byRound := make(map[int]cell, len(fiber))
	for _, row := range fiber {
		byRound[row.Round] = row
	}
	out := make([]float64, 0, len(zatrano))
	for _, row := range zatrano {
		other, ok := byRound[row.Round]
		if !ok {
			return nil, fmt.Errorf("missing fiber round %d", row.Round)
		}
		if other.RPS <= 0 {
			return nil, fmt.Errorf("fiber rps is 0 in round %d", row.Round)
		}
		out = append(out, row.RPS/other.RPS)
	}
	return out, nil
}

// bootstrapMedianCI resamples with replacement and returns the sample median
// plus the 2.5 and 97.5 percentiles of the bootstrap medians.
func bootstrapMedianCI(samples []float64, draws int, seed int64) (median, low, high float64) {
	median = medianFloat(samples)
	if len(samples) == 0 || draws < 1 {
		return median, median, median
	}
	rng := rand.New(rand.NewSource(seed))
	stats := make([]float64, draws)
	buf := make([]float64, len(samples))
	for i := 0; i < draws; i++ {
		for j := range buf {
			buf[j] = samples[rng.Intn(len(samples))]
		}
		stats[i] = medianFloat(buf)
	}
	sort.Float64s(stats)
	return median, quantileSorted(stats, 0.025), quantileSorted(stats, 0.975)
}

func quantileSorted(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	if p <= 0 {
		return sorted[0]
	}
	if p >= 1 {
		return sorted[len(sorted)-1]
	}
	pos := p * float64(len(sorted)-1)
	i := int(pos)
	frac := pos - float64(i)
	if i+1 >= len(sorted) {
		return sorted[i]
	}
	return sorted[i]*(1-frac) + sorted[i+1]*frac
}

func ratioOf(a, b float64) float64 {
	if b == 0 {
		return 0
	}
	return a / b
}

// cellVerdict applies the round-2 gate.
// "ustun" requires the throughput CI to sit entirely above 1 and the p99 and
// CPU medians to be no worse than Fiber. A CI that contains 1 is "esit".
// Peak RSS is not a gate.
func cellVerdict(ciLow, ciHigh, p99Ratio, cpuRatio float64) string {
	if ciLow <= 1 && ciHigh >= 1 {
		return "esit"
	}
	if ciLow > 1 && p99Ratio <= 1 && cpuRatio <= 1 {
		return "ustun"
	}
	if ciHigh < 1 {
		return "fiber"
	}
	return "esik-yok"
}
