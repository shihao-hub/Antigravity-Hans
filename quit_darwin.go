//go:build darwin

package main

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// requestGracefulQuit 在 macOS 上通过 AppleScript 请求目标应用正常退出。
// 正常退出会走 Electron / VS Code 的保存与热退出（hot exit）流程，
// 避免直接 SIGKILL 丢失用户正在进行的会话与未落盘内容。
// 任何一步取消或失败都中止重启。
func requestGracefulQuit(cfg AppConfig) error {
	bundle := macBundlePath(cfg)
	if bundle == "" {
		return fmt.Errorf("未找到应用 bundle")
	}
	if _, err := exec.LookPath("osascript"); err != nil {
		return err
	}

	fmt.Println("       (macOS 可能弹出“想要控制 Antigravity”的授权提示，允许一次即可)")
	script := fmt.Sprintf("tell application %q to quit", bundle)
	ctx, cancel := context.WithTimeout(context.Background(), gracefulQuitTimeout)
	defer cancel()
	return exec.CommandContext(ctx, "osascript", "-e", script).Run()
}

// macBundlePath 返回正在运行（或已安装）实例对应的 .app 路径
func macBundlePath(cfg AppConfig) string {
	exe := DetectApp(cfg).Path
	if exe == "" {
		for _, p := range cfg.PossiblePaths {
			if fileExists(p) {
				exe = p
				break
			}
		}
	}
	if exe == "" {
		return ""
	}
	// .../Foo.app/Contents/MacOS/Foo -> .../Foo.app
	app := filepath.Dir(filepath.Dir(filepath.Dir(exe)))
	if strings.HasSuffix(app, ".app") {
		return app
	}
	return ""
}
