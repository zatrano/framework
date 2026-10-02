//go:build windows

package safepath

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestWindowsVolumeNameSplitAbs(t *testing.T) {
	base, parts := splitAbs(`C:\Users\demo\file.txt`)
	if !strings.HasPrefix(base, `C:`) {
		t.Fatalf("volume base=%q", base)
	}
	if len(parts) < 3 {
		t.Fatalf("parts=%v", parts)
	}
	if filepath.VolumeName(`C:\Users\demo`) != `C:` {
		t.Fatal("expected volume name C:")
	}
}

func TestWindowsUNCVolumeName(t *testing.T) {
	unc := `\\server\share\dir\file.txt`
	vol := filepath.VolumeName(unc)
	if vol == "" {
		t.Fatalf("expected UNC volume for %q", unc)
	}
	base, parts := splitAbs(unc)
	if !strings.Contains(base, vol) && base == "" {
		t.Fatalf("base=%q vol=%q parts=%v", base, vol, parts)
	}
	if len(parts) == 0 {
		t.Fatalf("expected UNC parts vol=%q", vol)
	}
}
