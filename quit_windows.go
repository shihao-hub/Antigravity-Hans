//go:build windows

package main

import (
	"context"
	"os/exec"
	"time"
)

func init() {
	// Windows 上应用常被配置为“关闭窗口后继续在后台运行”，
	// 优雅关闭的成功率低于 macOS，等待时间取短一些，避免双击后干等。
	gracefulQuitTimeout = 5 * time.Second
}

// requestGracefulQuit 在 Windows 上通过 taskkill（不带 /F）向应用主窗口发送关闭请求，
// 让 Electron / VS Code 走正常的退出与保存流程，避免直接 TerminateProcess 丢失用户状态。
// 任何一步失败都中止重启。
func requestGracefulQuit(cfg AppConfig) error {
	if _, err := exec.LookPath("taskkill"); err != nil {
		return err
	}
	// 不带 /F：只请求关闭；不带 /T：不动子进程树。残留的辅助进程由强制结束流程兜底。
	ctx, cancel := context.WithTimeout(context.Background(), gracefulQuitTimeout)
	defer cancel()
	return exec.CommandContext(ctx, "taskkill", "/IM", cfg.Exe).Run()
}
