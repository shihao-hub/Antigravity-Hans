package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsExpectedExecutable(t *testing.T) {
	dir := t.TempDir()
	cfg := AppConfig{Name: "Antigravity", Exe: "Antigravity"}

	good := filepath.Join(dir, "Antigravity")
	if err := os.WriteFile(good, []byte("bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	other := filepath.Join(dir, "evil")
	if err := os.WriteFile(other, []byte("bin"), 0o755); err != nil {
		t.Fatal(err)
	}

	if !isExpectedExecutable(good, cfg) {
		t.Fatal("同名普通文件应通过校验")
	}
	if isExpectedExecutable(other, cfg) {
		t.Fatal("文件名与目标不一致的路径应被拒绝")
	}
	if isExpectedExecutable(dir, cfg) {
		t.Fatal("目录应被拒绝")
	}
	if isExpectedExecutable("", cfg) {
		t.Fatal("空路径应被拒绝")
	}
	if isExpectedExecutable(filepath.Join(dir, "missing"), cfg) {
		t.Fatal("不存在的路径应被拒绝")
	}

	// 符号链接即使名字正确也必须拒绝（注册表可能指向别处）
	sub := filepath.Join(dir, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(sub, "Antigravity")
	if err := os.Symlink(good, link); err != nil {
		t.Skipf("当前环境不支持符号链接: %v", err)
	}
	if isExpectedExecutable(link, cfg) {
		t.Fatal("符号链接应被拒绝")
	}

	// Windows 形态：大小写不同也应接受
	winCfg := AppConfig{Name: "Antigravity IDE", Exe: "Antigravity IDE.exe"}
	winExe := filepath.Join(dir, "Antigravity IDE.exe")
	if err := os.WriteFile(winExe, []byte("bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if !isExpectedExecutable(winExe, winCfg) {
		t.Fatal("Windows 可执行文件名应通过校验")
	}
}
