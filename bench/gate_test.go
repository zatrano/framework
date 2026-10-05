package bench

import "testing"

func TestGateDesign(t *testing.T) {
	t.Logf("H3 tier-0 <= %.2f x Fiber, both Fiber rows; equal-timeout is the comparison row", GateTier0VsFiber)
	t.Logf("regression gate: tier-0 and tier-1 <= %.2f x previous tag", GateRegression)
	t.Log("informative only; Phase 6 enforces this on Linux CI. Windows numbers are informational")
}
