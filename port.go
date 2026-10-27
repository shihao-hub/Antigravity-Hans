package main

import (
	"context"
	"fmt"
	"net"
	"sort"
	"time"

	procnet "github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/process"
)

// portFallbackCount 是首选端口被占用时继续尝试的备用端口数量
const portFallbackCount = 4

// candidatePorts 返回首选端口及其备用端口。
// 端口落在系统临时端口区间内（macOS / Windows 默认 49152-65535），
// 任何进程的瞬时出站连接都可能占用，因此必须准备备用端口。
func candidatePorts(preferred int) []int {
	ports := make([]int, 0, portFallbackCount+1)
	for i := 0; i <= portFallbackCount; i++ {
		ports = append(ports, preferred+i)
	}
	return ports
}

// isPortFree 判断本机 127.0.0.1 上该端口当前是否可绑定
func isPortFree(port int) bool {
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return false
	}
	_ = ln.Close()
	return true
}

// findCDPPort 在候选端口里寻找已经提供 CDP 服务的端口（通常是上一次由本工具启动的实例）。
// 找到时返回 (端口, true)：此时应当直接复用监视，而不是把应用杀掉重启。
func findCDPPort(cfg AppConfig) (int, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	connections, err := procnet.ConnectionsWithContext(ctx, "tcp")
	if err != nil {
		return 0, false
	}
	owners := make(map[int]bool)
	for _, connection := range connections {
		port := int(connection.Laddr.Port)
		if connection.Status != "LISTEN" || !isLoopbackHost(connection.Laddr.IP) {
			continue
		}
		p, err := process.NewProcess(connection.Pid)
		if err != nil {
			continue
		}
		name, _ := p.Name()
		exe, _ := p.Exe()
		if isProcessMatch(name, exe, cfg) && isExpectedExecutable(exe, cfg) && (!IsMac || isMacMainExe(exe, cfg)) {
			owners[port] = true
		}
	}
	var additional []int
	// 2.26.0 enables a random CDP port by default; reuse only listeners owned
	// by the normal app, so attaching does not require restarting its windows.
	if cfg.Name == AppNormal.Name {
		for port := range owners {
			additional = append(additional, port)
		}
		sort.Ints(additional)
	}
	return findCDPPortWithOwner(cfg, func(port int) bool { return owners[port] }, additional...)
}

func findCDPPortWithOwner(cfg AppConfig, belongsToApp func(int) bool, additional ...int) (int, bool) {
	ports := append(candidatePorts(cfg.Port), additional...)
	seen := make(map[int]bool)
	for _, port := range ports {
		if seen[port] {
			continue
		}
		seen[port] = true
		if !belongsToApp(port) {
			continue
		}
		for _, target := range CheckPort(port) {
			if isTargetPage(target.URL, cfg.Name) {
				return port, true
			}
		}
		// A normal app can keep running with no windows (or only a sign-in
		// window). Its verified browser endpoint can watch the next app page.
		if cfg.Name == AppNormal.Name {
			if _, err := getBrowserWSURL(port); err == nil {
				return port, true
			}
		}
	}
	return 0, false
}

// selectFreePort 返回第一个可绑定的候选端口；全部被占用时返回 (首选端口, false)
func selectFreePort(preferred int) (int, bool) {
	for _, port := range candidatePorts(preferred) {
		if isPortFree(port) {
			return port, true
		}
	}
	return preferred, false
}
