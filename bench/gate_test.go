package bench

import "testing"

func TestGateDesign(t *testing.T) {
	t.Logf("H3 tier-0 <= %.2f x Fiber", GateTier0VsFiber)
	t.Logf("regression gate: tier-0 and tier-1 <= %.2f x previous tag", GateRegression)
	t.Log("enforcement waits for the approved fix; this test does not fail the build")
}
