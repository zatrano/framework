package bench

// Performance gates for the published harness: a frozen router and the
// Run-shaped server. TestGateDesign only logs them. Phase 6 turns them into
// a Linux CI failure.
const (
	// GateTier0VsFiber is H3: tier-0 ns must be at most 1.00× Fiber.
	// Report it against both Fiber rows. The equal-timeout row is the
	// comparison row. Windows numbers are informational; Phase 6 enforces
	// the gate on Linux CI.
	GateTier0VsFiber = 1.00
	// GateRegression is the maximum slowdown of tier-0 and tier-1 against
	// the previous tag (5%).
	GateRegression = 1.05
	// GateRunVsOld is the header-hook budget. Run-shaped tier-0 must stay
	// within 1.10× the unlimited-timeout shape. allocs/op must match that
	// shape. B/op matches aside from the amortized +1 of the header
	// buffer (MaxHeaderBytes), which is not the hook. Informative until Phase 6.
	GateRunVsOld = 1.10
)
