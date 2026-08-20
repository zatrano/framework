//go:build windows

package session

import (
	"os"

	"golang.org/x/sys/windows"
)

func withFlock(lockPath string, fn func() error) error {
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()

	h := windows.Handle(f.Fd())
	var ol windows.Overlapped
	const lockfileExclusiveLock = 2
	if err := windows.LockFileEx(h, lockfileExclusiveLock, 0, 1, 0, &ol); err != nil {
		return err
	}
	defer func() {
		var uol windows.Overlapped
		_ = windows.UnlockFileEx(h, 0, 1, 0, &uol)
	}()
	return fn()
}
