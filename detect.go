package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/shirou/gopsutil/v4/process"
)

// DetectedApp 进程检测结果
type DetectedApp struct {
	Name     string
	Exe      string
	Port     int
	Running  bool
	Path     string   // 正在运行实例的可执行文件路径（如有）
	PIDs     []int32  // 所有匹配进程的 PID
	AllPaths []string // 所有已找到的安装路径
}

// isProcessMatch 判断进程是否与目标 App 匹配
func isProcessMatch(pName, pExe string, cfg AppConfig) bool {
	pName = strings.ToLower(pName)
	pExe = strings.ToLower(pExe)
	exeLower := strings.ToLower(cfg.Exe)

	if IsMac {
		appNameLower := strings.ToLower(cfg.Name)
		if strings.Contains(appNameLower, "ide") {
			return strings.Contains(pExe, "antigravity ide.app") || pName == "antigravity ide"
		}
		return strings.Contains(pExe, "antigravity.app") ||
			pName == "antigravity" ||
			filepath.Base(pExe) == exeLower
	}
	return pName == exeLower || filepath.Base(pExe) == exeLower
}

// DetectApp 检测目标 App 的运行状态与安装路径
func DetectApp(cfg AppConfig) DetectedApp {
	result := DetectedApp{
		Name:     cfg.Name,
		Exe:      cfg.Exe,
		Port:     cfg.Port,
		AllPaths: []string{},
		PIDs:     []int32{},
	}

	// 遍历进程列表
	procs, err := process.Processes()
	if err == nil {
		for _, p := range procs {
			pName, _ := p.Name()
			pExe, _ := p.Exe()

			if isProcessMatch(pName, pExe, cfg) {
				result.Running = true
				result.PIDs = append(result.PIDs, p.Pid)

				if pExe != "" {
					if _, err := os.Stat(pExe); err == nil {
						if IsMac {
							if isMacMainExe(pExe, cfg) {
								result.Path = pExe
							}
						} else {
							result.Path = pExe
						}
					}
				}
			}
		}
	}

	// 从预设路径探测安装位置
	allFound := []string{}
	for _, p := range cfg.PossiblePaths {
		if _, err := os.Stat(p); err == nil {
			allFound = append(allFound, p)
		}
	}

	// 若预设路径无结果，查询注册表（Windows）。
	// 注册表内容不受本工具控制，因此只接受文件名与目标 exe 一致、且确实是普通文件的路径。
	if len(allFound) == 0 {
		regPaths := GetPathsFromRegistry(cfg.Name, cfg.Exe)
		for _, p := range regPaths {
			if !containsStr(allFound, p) && isExpectedExecutable(p, cfg) {
				allFound = append(allFound, p)
			}
		}
	}
	result.AllPaths = allFound

	if result.Path == "" && len(allFound) == 1 {
		result.Path = allFound[0]
	}

	return result
}

// isExpectedExecutable 校验待启动路径确实是目标应用的可执行文件。
// 路径可能来自注册表等外部来源，因此要求：存在、是普通文件（非目录或符号链接）、
// 且文件名与配置中的可执行文件名一致。
func isExpectedExecutable(path string, cfg AppConfig) bool {
	if path == "" || cfg.Exe == "" {
		return false
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() {
		return false
	}
	return strings.EqualFold(filepath.Base(path), cfg.Exe)
}

// isMacMainExe 判断 macOS 上是否是主进程可执行文件路径
func isMacMainExe(pExe string, cfg AppConfig) bool {
	appNameLower := strings.ToLower(cfg.Name)
	if strings.Contains(appNameLower, "ide") {
		return strings.HasSuffix(pExe, "/Contents/MacOS/Electron")
	}
	return strings.HasSuffix(pExe, "/Contents/MacOS/Antigravity")
}

// KillApp 强制结束所有匹配的进程
func KillApp(cfg AppConfig) {
	procs, err := process.Processes()
	if err != nil {
		return
	}
	for _, p := range procs {
		pName, _ := p.Name()
		pExe, _ := p.Exe()
		if isProcessMatch(pName, pExe, cfg) {
			_ = p.Kill()
		}
	}
}

// CheckProcessRunning 返回与应用匹配的所有 PID
func CheckProcessRunning(cfg AppConfig, exePath string) []int32 {
	var pids []int32
	procs, err := process.Processes()
	if err != nil {
		return pids
	}
	exePathLower := strings.ToLower(exePath)

	for _, p := range procs {
		pName, _ := p.Name()
		pExe, _ := p.Exe()

		// 绝对路径精确匹配
		if exePath != "" && pExe != "" {
			if strings.EqualFold(filepath.Clean(pExe), filepath.Clean(exePathLower)) {
				pids = append(pids, p.Pid)
				continue
			}
		}

		// macOS bundle 路径检测
		if IsMac {
			appNameLower := strings.ToLower(cfg.Name)
			pExeLower := strings.ToLower(pExe)
			pNameLower := strings.ToLower(pName)
			if strings.Contains(appNameLower, "ide") {
				if strings.Contains(pExeLower, "antigravity ide.app") || pNameLower == "antigravity ide" {
					pids = append(pids, p.Pid)
					continue
				}
			} else {
				if strings.Contains(pExeLower, "antigravity.app") || pNameLower == "antigravity" {
					pids = append(pids, p.Pid)
					continue
				}
			}
			// 避免 macOS 误匹配其他 Electron 实例
			if strings.ToLower(cfg.Exe) == "electron" {
				continue
			}
		}

		// 通用文件名匹配（兜底）
		if isProcessMatch(pName, pExe, cfg) {
			pids = append(pids, p.Pid)
		}
	}
	return pids
}

func containsStr(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
