//go:build gui && windows

package main

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
)

// trayIconPNG 托盘图标（由 icon.ico 提取的 64×64 PNG，构建期嵌入）。
//
//go:embed gui_assets/tray.png
var trayIconPNG []byte

// buildTray 创建系统托盘图标与菜单。
func buildTray(g *guiApp) {
	desk, ok := g.app.(desktop.App)
	if !ok {
		return // Windows 桌面驱动必然支持；防御性兜底
	}
	desk.SetSystemTrayIcon(fyne.NewStaticResource("tray.png", trayIconPNG))
	desk.SetSystemTrayWindow(g.win) // 左键点击托盘图标显示主窗口

	showItem := fyne.NewMenuItem("显示主窗口", func() { g.showMainWindow() })
	openDirItem := fyne.NewMenuItem("打开 exe 所在目录", func() {
		if err := openExeDir(); err != nil {
			dialog.ShowError(fmt.Errorf("打开目录失败: %w", err), g.win)
		}
	})

	// 开机自启动：仅 release 构建可勾选；dev 构建置灰（逻辑侧同样拒绝注册）
	autostartItem := fyne.NewMenuItem("开机自启动", nil)
	autostartItem.Checked = autostartEnabled()
	if !isReleaseBuild() {
		autostartItem.Disabled = true
		autostartItem.Label = "开机自启动（开发版不可用）"
	}

	quitItem := fyne.NewMenuItem("退出", func() { g.app.Quit() })

	menu := fyne.NewMenu("Antigravity-Hans",
		showItem,
		openDirItem,
		autostartItem,
		fyne.NewMenuItemSeparator(),
		quitItem,
	)

	autostartItem.Action = func() {
		target := !autostartItem.Checked
		if err := setAutostart(target); err != nil {
			dialog.ShowError(fmt.Errorf("设置开机自启失败: %w", err), g.win)
			target = autostartEnabled() // 失败时回滚为注册表实际状态
		}
		autostartItem.Checked = target
		menu.Refresh()
	}

	desk.SetSystemTrayMenu(menu)
}

// openExeDir 用资源管理器打开当前可执行文件所在目录。
func openExeDir() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	return exec.Command("explorer.exe", filepath.Dir(exe)).Start()
}
