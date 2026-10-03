//go:build gui && windows

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

// autostartValueName 开机自启在 HKCU Run 键下的值名。
const autostartValueName = "Antigravity-Hans"

// autostartRunKey 当前用户开机自启动注册表键。
const autostartRunKey = `Software\Microsoft\Windows\CurrentVersion\Run`

// autostartEnabled 查询开机自启是否已注册（查询失败按未启用处理）。
func autostartEnabled() bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, autostartRunKey, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()
	v, _, err := k.GetStringValue(autostartValueName)
	return err == nil && v != ""
}

// setAutostart 注册/取消开机自启。
// 仅 release 构建允许注册（构建提交带 git tag），dev 构建直接拒绝，
// 避免开发调试时污染系统注册表与正式版踩踏。
func setAutostart(enabled bool) error {
	if !enabled {
		return removeAutostart()
	}
	if !isReleaseBuild() {
		return fmt.Errorf("dev_mode: 开发版禁止注册开机自启，防止污染系统注册表")
	}
	if err := autostartExecutableGuard(); err != nil {
		return err
	}
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("获取可执行文件路径失败: %w", err)
	}
	k, err := registry.OpenKey(registry.CURRENT_USER, autostartRunKey, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("打开注册表键失败: %w", err)
	}
	defer k.Close()
	// 登录后静默启动到系统托盘（--hidden：不弹主窗）
	value := fmt.Sprintf("\"%s\" --hidden", exe)
	if err := k.SetStringValue(autostartValueName, value); err != nil {
		return fmt.Errorf("写入注册表失败: %w", err)
	}
	return nil
}

// removeAutostart 删除开机自启注册项（不存在时视为成功）。
func removeAutostart() error {
	k, err := registry.OpenKey(registry.CURRENT_USER, autostartRunKey, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("打开注册表键失败: %w", err)
	}
	defer k.Close()
	if err := k.DeleteValue(autostartValueName); err != nil && err != registry.ErrNotExist {
		return fmt.Errorf("删除注册表值失败: %w", err)
	}
	return nil
}

// autostartExecutableGuard 拒绝注册临时构建产物（go run / go-build 缓存目录），
// 避免重启后留下指向失效路径的自启项。
func autostartExecutableGuard() error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("获取可执行文件路径失败: %w", err)
	}
	resolved := filepath.Clean(exe)
	tempDir := filepath.Clean(os.TempDir())
	if strings.Contains(resolved, "go-build") ||
		strings.HasPrefix(strings.ToLower(resolved), strings.ToLower(tempDir)) {
		return fmt.Errorf("dev_executable: 当前 exe 为临时构建产物（%s），不可注册开机自启；请使用构建脚本产出的 exe", resolved)
	}
	return nil
}
