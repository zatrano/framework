package bench

// Performance gates for the next approved fix. They are not enforced while
// perf/v3 is slower than v3.0.1: H3 held on v3.0.1 and the branch broke it.
const (
	// GateTier0VsFiber is H3: tier-0 ns must be at most 1.00× Fiber.
	GateTier0VsFiber = 1.00
	// GateRegression is the maximum slowdown of tier-0 and tier-1 against
	// the previous tag (5%).
	GateRegression = 1.05
)
