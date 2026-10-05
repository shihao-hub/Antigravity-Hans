//go:build gui && windows

package main

import (
	"golang.org/x/sys/windows"
)

// singleInstanceHandle 持有命名互斥体句柄，故意不释放（随进程结束由系统回收），
// 保证互斥体在整个进程生命周期内有效，第二个实例才能检测到已存在的实例。
var singleInstanceHandle windows.Handle

// acquireSingleInstance 创建命名互斥体判定单实例：
// 返回 false 表示已有同名实例在运行（新实例应提示后退出）。
// dev 与 release 使用不同的互斥体名，开发调试可与正式实例并行。
func acquireSingleInstance() bool {
	name := "Antigravity-Hans-GUI"
	if !isReleaseBuild() {
		name += ".dev"
	}
	namePtr, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return true // 命名失败视为无法判重，不阻塞主功能
	}
	h, err := windows.CreateMutex(nil, false, namePtr)
	if err != nil {
		if err == windows.ERROR_ALREADY_EXISTS {
			return false
		}
		return true // 其他创建失败：不阻塞主功能
	}
	singleInstanceHandle = h
	return true
}

// releaseSingleInstance 主动释放互斥体（在自更新准备启动新实例并退出前调用），
// 避免新进程在旧进程完全退出前因互斥体碰撞而误判退出。
func releaseSingleInstance() {
	if singleInstanceHandle != 0 {
		_ = windows.CloseHandle(singleInstanceHandle)
		singleInstanceHandle = 0
	}
}
