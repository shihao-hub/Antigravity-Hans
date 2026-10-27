//go:build !darwin && !windows

package main

import "fmt"

// requestGracefulQuit 目前仅在 macOS（AppleScript）与 Windows（taskkill）上实现，
// 其它平台提示手动退出，不进行强杀。
func requestGracefulQuit(cfg AppConfig) error {
	return fmt.Errorf("当前平台不支持请求应用正常退出")
}
