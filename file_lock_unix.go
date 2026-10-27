//go:build !windows

package main

import (
	"fmt"
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

func verifyMonitorStorage(info os.FileInfo) error {
	if err := verifyOverlayOwner(info); err != nil {
		return fmt.Errorf("监视器状态文件/目录权限不安全: %w", err)
	}
	if stat, ok := info.Sys().(*syscall.Stat_t); ok && info.Mode().IsRegular() && stat.Nlink > 1 {
		return fmt.Errorf("监视器状态文件不能是硬链接")
	}
	return nil
}

func lockMonitorFile(f *os.File) error   { return unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB) }
func unlockMonitorFile(f *os.File) error { return unix.Flock(int(f.Fd()), unix.LOCK_UN) }
