//go:build !windows

package main

import (
	"golang.org/x/sys/unix"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMonitorLockRejectsHardlinkedOrWritableState(t *testing.T) {
	monitorStateDirOverride = t.TempDir()
	defer func() { monitorStateDirOverride = "" }()
	victim := filepath.Join(t.TempDir(), "victim")
	if err := os.WriteFile(victim, []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	lock := monitorLockPath(AppNormal)
	if err := os.Link(victim, lock); err != nil {
		t.Fatal(err)
	}
	if release, ok := acquireMonitorLock(AppNormal); ok {
		release()
		t.Fatal("hardlinked PID file accepted")
	}
	data, _ := os.ReadFile(victim)
	if string(data) != "original" {
		t.Fatal("hardlinked file overwritten")
	}
	if err := os.Remove(lock); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(lock, []byte("0"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(lock, 0666); err != nil {
		t.Fatal(err)
	}
	if release, ok := acquireMonitorLock(AppNormal); ok {
		release()
		t.Fatal("writable PID file accepted")
	}
}

func TestExternalOverlayRejectsFIFOBeforeRead(t *testing.T) {
	p := filepath.Join(t.TempDir(), "antigravity-hans-overlay.js")
	if err := unix.Mkfifo(p, 0600); err != nil {
		t.Fatal(err)
	}
	result := make(chan error, 1)
	go func() { _, err := readExternalOverlay(p, true); result <- err }()
	select {
	case err := <-result:
		if err == nil {
			t.Fatal("FIFO accepted")
		}
	case <-time.After(time.Second):
		t.Fatal("FIFO blocked startup")
	}
}
