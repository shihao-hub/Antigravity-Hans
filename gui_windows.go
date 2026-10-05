//go:build gui && windows

// gui_windows.go：GUI 壳入口（Windows）。
// 以 `go build -tags gui -ldflags "-H windowsgui ..."` 构建时，本文件提供 main()，
// 产物为无控制台子系统的窗口程序；默认构建（无 gui 标签）不包含本文件，仍由
// main.go 提供原控制台 CLI 入口，原有逻辑保持不变。
package main

import (
	_ "embed"
	"os"
	"sync"
	"unsafe"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"golang.org/x/sys/windows"
)

// appIconPNG 窗口与应用图标（256×256 PNG，构建期嵌入）。
//
//go:embed gui_assets/appicon.png
var appIconPNG []byte

// Version 与 ReleaseMode 由构建脚本通过
// -ldflags "-X main.Version=... -X main.ReleaseMode=..." 注入；
// 同名变量在 main.go（//go:build !gui）中另有声明，两者互斥不会冲突。
var (
	Version     = "dev"
	ReleaseMode = "dev"
)

// isReleaseBuild 报告当前构建是否为 release（构建提交带 git tag）。
func isReleaseBuild() bool { return ReleaseMode == "release" }

// guiOptions GUI 启动参数（由快捷方式/开机自启传入）。
type guiOptions struct {
	Hidden bool // --hidden：静默启动，仅托盘不弹主窗（开机自启使用）
	Start  bool // --start：启动后立即开始汉化监视（一键汉化快捷方式使用）
}

func parseGUIOptions(args []string) guiOptions {
	opts := guiOptions{}
	for _, a := range args {
		switch a {
		case "--hidden":
			opts.Hidden = true
		case "--start":
			opts.Start = true
		}
	}
	return opts
}

// guiApp GUI 组装状态：应用、主窗与页面控件。
type guiApp struct {
	app    fyne.App
	win    fyne.Window
	status *widget.Label
	btnApp *widget.Button

	mu         sync.Mutex
	monitoring bool
}

func main() {
	opts := parseGUIOptions(os.Args[1:])

	// 启动时抹除上次重命名置换遗留的 .old 文件（旧实例已退出，锁已解开）
	cleanOldVersions()

	if !acquireSingleInstance() {
		showMessageBox("Antigravity 汉化工具", "程序已在运行，请查看系统托盘。")
		return
	}

	g := &guiApp{}
	g.app = app.NewWithID("antigravity-hans-gui")

	// 设置应用与窗口图标（消除无图标问题，展示专属品牌标识）
	iconRes := fyne.NewStaticResource("appicon.png", appIconPNG)
	g.app.SetIcon(iconRes)

	g.win = g.app.NewWindow("Antigravity 汉化工具")
	g.win.SetIcon(iconRes)
	g.win.Resize(fyne.NewSize(500, 320))
	g.win.SetCloseIntercept(func() { g.win.Hide() }) // 关窗仅隐藏到托盘，退出走托盘菜单
	g.win.SetContent(g.buildPage())

	buildTray(g)

	g.app.Lifecycle().SetOnStarted(func() {
		if opts.Start {
			g.startMonitor()
		}
	})

	if !opts.Hidden {
		g.win.Show()
	}
	g.app.Run()
}

// buildPage 构建主页面：版本信息、启动按钮与状态行。
func (g *guiApp) buildPage() fyne.CanvasObject {
	title := widget.NewLabelWithStyle("Antigravity 汉化工具", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	version := widget.NewLabelWithStyle("版本 "+Version, fyne.TextAlignCenter, fyne.TextStyle{})
	hint := widget.NewLabel("点击下方按钮启动 Antigravity 动态汉化（CDP 注入，不修改本地文件）。\n关闭窗口后程序驻留系统托盘继续运行。")
	hint.Wrapping = fyne.TextWrapWord

	g.btnApp = widget.NewButton("启动 Antigravity 汉化", func() { g.startMonitor() })

	g.status = widget.NewLabel("就绪")
	g.status.Wrapping = fyne.TextWrapWord

	return container.NewPadded(container.NewVBox(
		title,
		version,
		widget.NewSeparator(),
		hint,
		g.btnApp,
		widget.NewSeparator(),
		g.status,
	))
}

// startMonitor 在后台启动汉化监视；同一时间只允许一个监视任务。
func (g *guiApp) startMonitor() {
	g.mu.Lock()
	if g.monitoring {
		g.mu.Unlock()
		return
	}
	g.monitoring = true
	g.mu.Unlock()

	g.setBusy(true, "正在启动 Antigravity 动态汉化...")

	overlaySource := LoadOverlaySource()
	go func() {
		Run(AppNormal, overlaySource) // 阻塞至监视结束（目标退出或启动失败）
		g.mu.Lock()
		g.monitoring = false
		g.mu.Unlock()
		g.setBusy(false, "监视已结束（目标应用已退出或启动失败）")
	}()
}

// setBusy 更新按钮可用状态与状态行（可安全地从任意 goroutine 调用）。
func (g *guiApp) setBusy(busy bool, status string) {
	fyne.Do(func() {
		if g.btnApp == nil {
			return
		}
		if busy {
			g.btnApp.Disable()
		} else {
			g.btnApp.Enable()
		}
		if status != "" && g.status != nil {
			g.status.SetText(status)
		}
	})
}

// showMainWindow 显示并聚焦主窗口（托盘菜单与左键唤窗共用）。
func (g *guiApp) showMainWindow() {
	g.win.Show()
	g.win.RequestFocus()
}

// showMessageBox 弹出原生消息框；不依赖 Fyne 启动，可在启动早期使用。
func showMessageBox(title, text string) {
	user32 := windows.NewLazySystemDLL("user32.dll")
	proc := user32.NewProc("MessageBoxW")

	tPtr, err1 := windows.UTF16PtrFromString(text)
	cPtr, err2 := windows.UTF16PtrFromString(title)
	if err1 != nil || err2 != nil {
		return
	}
	const mbOK, mbIconInfo = 0x00000000, 0x00000040
	proc.Call(0, uintptr(unsafe.Pointer(tPtr)), uintptr(unsafe.Pointer(cPtr)), mbOK|mbIconInfo)
}
