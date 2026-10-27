//go:build !windows

package main

import "golang.org/x/sys/unix"

func safeReadFlags() int { return unix.O_NOFOLLOW | unix.O_NONBLOCK }
