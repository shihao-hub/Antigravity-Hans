package main

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestAcquireMonitorLockIsExclusive(t *testing.T) {
	monitorStateDirOverride = t.TempDir()
	defer func() { monitorStateDirOverride = "" }()

	cfg := AppConfig{Name: "Antigravity", Port: 58321}

	release, ok := acquireMonitorLock(cfg)
	if !ok {
		t.Fatal("首次获取锁应当成功")
	}
	if _, ok := acquireMonitorLock(cfg); ok {
		t.Fatal("重复获取锁应当失败，避免重复监视器")
	}
	release()

	if release, ok := acquireMonitorLock(cfg); !ok {
		t.Fatal("释放后应当可以再次获取锁")
	} else {
		t.Cleanup(release)
	}
}

func TestAcquireMonitorLockOverridesStalePID(t *testing.T) {
	monitorStateDirOverride = t.TempDir()
	defer func() { monitorStateDirOverride = "" }()

	cfg := AppConfig{Name: "Antigravity IDE", Port: 58322}
	path := monitorLockPath(cfg)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	// 一个几乎不可能存在的 PID，模拟异常退出留下的残留锁
	if err := os.WriteFile(path, []byte(strconv.Itoa(1<<30)), 0o600); err != nil {
		t.Fatal(err)
	}
	if release, ok := acquireMonitorLock(cfg); !ok {
		t.Fatal("残留锁文件不应阻断启动")
	} else {
		t.Cleanup(release)
	}
}

func TestMonitorLockPathIsPerApp(t *testing.T) {
	monitorStateDirOverride = t.TempDir()
	defer func() { monitorStateDirOverride = "" }()

	if monitorLockPath(AppNormal) == monitorLockPath(AppIDE) {
		t.Fatal("两个应用应使用不同的锁文件，以便同时汉化")
	}
	if monitorLockPath(AppNormal) != monitorLockPath(AppConfig{Name: "Antigravity", Port: 58999}) {
		t.Fatal("锁文件不应随端口变化（运行期可能改用备用端口）")
	}
	want := filepath.Join(monitorStateDirOverride, "monitor-antigravity-ide.pid")
	if got := monitorLockPath(AppIDE); got != want {
		t.Fatalf("锁文件命名不符合预期: %s", got)
	}
}
