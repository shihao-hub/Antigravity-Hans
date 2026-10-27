package main

import (
	"fmt"
	"net"
	"net/http"
	"testing"
)

func TestCandidatePorts(t *testing.T) {
	ports := candidatePorts(58321)
	if len(ports) != portFallbackCount+1 {
		t.Fatalf("候选端口数量应为 %d，实际 %d", portFallbackCount+1, len(ports))
	}
	if ports[0] != 58321 || ports[len(ports)-1] != 58321+portFallbackCount {
		t.Fatalf("候选端口范围不对: %v", ports)
	}
}

func TestSelectFreePortSkipsOccupied(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("无法监听本地端口: %v", err)
	}
	defer ln.Close()
	busy := ln.Addr().(*net.TCPAddr).Port

	got, ok := selectFreePort(busy)
	if !ok {
		t.Fatal("应当能找到可用的备用端口")
	}
	if got == busy {
		t.Fatalf("不应选中已被占用的端口 %d", busy)
	}
	allowed := false
	for _, p := range candidatePorts(busy) {
		if p == got {
			allowed = true
		}
	}
	if !allowed {
		t.Fatalf("选出的端口 %d 不在候选列表内", got)
	}
}

func freeTCPPort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("无法获取空闲端口: %v", err)
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port
}

func TestFindCDPPortDetectsEndpoint(t *testing.T) {
	base := freeTCPPort(t)
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", base))
	if err != nil {
		t.Skipf("无法监听 %d: %v", base, err)
	}
	defer ln.Close()

	// 起一个假 CDP 端点：/json/list 返回一个可注入的 page target
	go func() {
		_ = http.Serve(ln, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/json/list" {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintf(w, `[{"id":"t1","type":"page","title":"主窗口","url":"https://127.0.0.1:1/app","webSocketDebuggerUrl":"ws://127.0.0.1:%d/devtools/page/t1"}]`, base)
		}))
	}()

	port, found := findCDPPortWithOwner(AppConfig{Name: AppNormal.Name, Port: base}, func(port int) bool { return port == base })
	if !found {
		t.Fatalf("应识别出 %d 上的 CDP 端点", base)
	}
	if port != base {
		t.Fatalf("应返回命中端口 %d，实际 %d", base, port)
	}

	// 端点已存在时，selectFreePort 不应再把它当作可用空闲端口
	if free, _ := selectFreePort(base); free == base {
		t.Fatalf("已有 CDP 端点占用的端口 %d 不应被当作空闲端口", base)
	}
}

func TestIsPortFreeDetectsListener(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("无法监听本地端口: %v", err)
	}
	defer ln.Close()
	busy := ln.Addr().(*net.TCPAddr).Port
	if isPortFree(busy) {
		t.Fatalf("已被占用的端口 %d 不应被判定为空闲", busy)
	}
}
