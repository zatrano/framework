package console

import "testing"

func TestCurrentReleaseMatchesVERSION(t *testing.T) {
	root, err := frameworkModuleRoot()
	if err != nil {
		t.Fatal(err)
	}
	got := productVersionAt(root)
	if got != currentRelease {
		t.Fatalf("VERSION=%q currentRelease=%q — keep console/version.go in sync", got, currentRelease)
	}
}
