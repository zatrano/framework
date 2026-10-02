package console

import "testing"

func TestCurrentReleaseMatchesVERSION(t *testing.T) {
	root, err := frameworkModuleRoot()
	if err != nil {
		t.Fatal(err)
	}
	got := productVersionAt(root)
	base := got
	if i := len(base); i > 0 {
		for j := 0; j < len(base); j++ {
			if base[j] == '-' {
				base = base[:j]
				break
			}
		}
	}
	if base != currentRelease {
		t.Fatalf("VERSION=%q (base %q) currentRelease=%q — keep consolecore.CurrentRelease in sync with VERSION major.minor.patch", got, base, currentRelease)
	}
}
