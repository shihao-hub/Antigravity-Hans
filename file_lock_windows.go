//go:build windows

package main

import (
	"os"

	"golang.org/x/sys/windows"
)

func verifyMonitorStorage(_ os.FileInfo) error { return nil }

// Lock beyond the metadata bytes so --stop can still read the PID record.
func lockMonitorFile(f *os.File) error {
	return windows.LockFileEx(windows.Handle(f.Fd()), windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY, 0, 1, 0, &windows.Overlapped{OffsetHigh: 1})
}
func unlockMonitorFile(f *os.File) error {
	return windows.UnlockFileEx(windows.Handle(f.Fd()), 0, 1, 0, &windows.Overlapped{OffsetHigh: 1})
}
