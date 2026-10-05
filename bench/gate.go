package bench

// Performance gates for the published harness: a frozen router and the
// Run-shaped server. TestGateDesign only logs them. Phase 6 turns them into
// a Linux CI failure.
const (
	// GateTier0VsFiber is H3: tier-0 ns must be at most 1.00× Fiber.
	GateTier0VsFiber = 1.00
	// GateRegression is the maximum slowdown of tier-0 and tier-1 against
	// the previous tag (5%).
	GateRegression = 1.05
)
