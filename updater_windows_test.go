//go:build gui && windows

package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseSemVer(t *testing.T) {
	tests := []struct {
		input    string
		expected []int
		ok       bool
	}{
		{"v0.4.4", []int{0, 4, 4}, true},
		{"0.4.5", []int{0, 4, 5}, true},
		{"v1.0.0-dev", []int{1, 0, 0}, true},
		{"V2.15.1+build123", []int{2, 15, 1}, true},
		{"invalid", nil, false},
	}

	for _, tt := range tests {
		nums, ok := parseSemVer(tt.input)
		if ok != tt.ok {
			t.Errorf("parseSemVer(%q) ok = %v, expected %v", tt.input, ok, tt.ok)
			continue
		}
		if ok {
			if len(nums) != len(tt.expected) {
				t.Errorf("parseSemVer(%q) len = %d, expected %d", tt.input, len(nums), len(tt.expected))
				continue
			}
			for i := range nums {
				if nums[i] != tt.expected[i] {
					t.Errorf("parseSemVer(%q)[%d] = %d, expected %d", tt.input, i, nums[i], tt.expected[i])
				}
			}
		}
	}
}

func TestIsNewerVersion(t *testing.T) {
	tests := []struct {
		current string
		latest  string
		newer   bool
	}{
		{"v0.4.4", "v0.4.5", true},
		{"0.4.4", "v0.4.5", true},
		{"v0.4.5", "v0.4.5", false},
		{"v0.4.5", "v0.4.4", false},
		{"v0.4.4", "v1.0.0", true},
		{"v1.2.0", "v1.1.9", false},
		{"dev", "v0.4.5", true},
		{"v0.4.4-dev", "v0.4.4", true},
	}

	for _, tt := range tests {
		got := isNewerVersion(tt.current, tt.latest)
		if got != tt.newer {
			t.Errorf("isNewerVersion(%q, %q) = %v, expected %v", tt.current, tt.latest, got, tt.newer)
		}
	}
}

func TestRenameReplacePattern(t *testing.T) {
	// 测试重命名置换法的核心文件系统生命周期：
	// 1. 创建模拟运行的 app.exe
	// 2. 将其改名为 app.exe.old
	// 3. 将模拟新下载的 app.exe.download 改名为 app.exe
	// 4. 清理 app.exe.old
	tmpDir, err := os.MkdirTemp("", "updater_test_*")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	mockExe := filepath.Join(tmpDir, "app.exe")
	mockOld := filepath.Join(tmpDir, "app.exe.old")
	mockDownload := filepath.Join(tmpDir, "app.exe.download")

	if err := os.WriteFile(mockExe, []byte("v1"), 0755); err != nil {
		t.Fatalf("写入模拟原文件失败: %v", err)
	}
	if err := os.WriteFile(mockDownload, []byte("v2"), 0755); err != nil {
		t.Fatalf("写入模拟下载文件失败: %v", err)
	}

	// 步骤 1：原文件重命名为 .old
	if err := os.Rename(mockExe, mockOld); err != nil {
		t.Fatalf("重命名 mockExe -> mockOld 失败: %v", err)
	}

	// 步骤 2：下载文件重命名为原文件名
	if err := os.Rename(mockDownload, mockExe); err != nil {
		t.Fatalf("重命名 mockDownload -> mockExe 失败: %v", err)
	}

	// 验证内容已更新为 v2
	content, err := os.ReadFile(mockExe)
	if err != nil || string(content) != "v2" {
		t.Fatalf("置换后内容验证失败: %v, 内容: %s", err, string(content))
	}

	// 步骤 3：清理 .old
	if err := os.Remove(mockOld); err != nil {
		t.Fatalf("清理 mockOld 失败: %v", err)
	}
}
