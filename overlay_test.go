package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVerifyExternalOverlay(t *testing.T) {
	dir := t.TempDir()
	content := []byte("window." + overlaySentinel + " = true;\n")

	write := func(name string, data []byte, perm os.FileMode) string {
		t.Helper()
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, data, perm); err != nil {
			t.Fatalf("写入测试文件失败: %v", err)
		}
		// umask 会把 0666 之类削弱成 0644，这里显式 chmod 才能测到权限分支
		if err := os.Chmod(p, perm); err != nil {
			t.Fatalf("设置测试文件权限失败: %v", err)
		}
		return p
	}

	good := write("good.js", content, 0o600)
	if err := verifyExternalOverlay(good, content); !IsWindows && err != nil {
		t.Fatalf("合规脚本不应被拒绝: %v", err)
	} else if IsWindows && err == nil {
		t.Fatal("Windows 必须显式信任外部脚本，不能依赖 Unix 权限位")
	}

	worldWritable := write("world.js", content, 0o666)
	if err := verifyExternalOverlay(worldWritable, content); err == nil {
		t.Fatal("同组/其他用户可写的脚本应被拒绝")
	}

	noSentinel := write("other.js", []byte("console.log(1);"), 0o600)
	if err := verifyExternalOverlay(noSentinel, []byte("console.log(1);")); err == nil {
		t.Fatal("缺少汉化标记的文件应被拒绝")
	}

	linkTarget := write("target.js", content, 0o600)
	link := filepath.Join(dir, "link.js")
	if err := os.Symlink(linkTarget, link); err != nil {
		t.Skipf("当前环境不支持符号链接: %v", err)
	}
	if err := verifyExternalOverlay(link, content); err == nil {
		t.Fatal("符号链接应被拒绝")
	}

	big := filepath.Join(dir, "big.js")
	if err := os.WriteFile(big, append(content, make([]byte, maxExternalOverlaySize)...), 0o600); err != nil {
		t.Fatalf("写入大文件失败: %v", err)
	}
	if err := verifyExternalOverlay(big, content); err == nil {
		t.Fatal("超出体积上限的文件应被拒绝")
	}
}

func TestOverlayFingerprint(t *testing.T) {
	got := overlayFingerprint("hello")
	if len(got) != 16 {
		t.Fatalf("指纹长度应为 16，实际 %d", len(got))
	}
	if got != overlayFingerprint("hello") {
		t.Fatal("相同内容应得到相同指纹")
	}
	if got == overlayFingerprint("hello!") {
		t.Fatal("不同内容不应得到相同指纹")
	}
	if !strings.Contains(string(embeddedOverlayJS), overlaySentinel) {
		t.Fatal("内嵌脚本应当包含汉化标记")
	}
}
