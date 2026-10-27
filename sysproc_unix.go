//go:build !windows

package main

import (
	"os/exec"
	"syscall"
)

func HideConsole() {
	// Unix 平台无需且无法通过 Windows API 隐藏控制台，由后台守护程序或 wrapper 自身处理
}

func setSysProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true, // 创建新的会话，与父进程分离
	}
}
