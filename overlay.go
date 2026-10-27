package main

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

//go:embed antigravity-hans-overlay.js
var embeddedOverlayJS []byte

// externalOverlayName 是允许放在可执行文件同目录、用于免编译更新词库的外部脚本名。
const externalOverlayName = "antigravity-hans-overlay.js"

// overlaySentinel 是汉化脚本必须包含的标记，用于挡掉同目录下同名的无关文件。
const overlaySentinel = "__antigravityZhPatchInstalled"

// maxExternalOverlaySize 是外部脚本体积上限（正常约 80KB）。
const maxExternalOverlaySize = 2 << 20

// overlayFingerprint 返回脚本内容的 SHA-256 前缀，便于用户核对实际被注入的内容。
func overlayFingerprint(js string) string {
	sum := sha256.Sum256([]byte(js))
	return hex.EncodeToString(sum[:])[:16]
}

// verifyExternalOverlay 校验外部脚本是否可以安全注入。
//
// 注入目标是持有账号凭据、文件与终端权限的应用进程，因此外部脚本只有在
// “普通文件（非符号链接）+ 不对同组或其他用户开放写权限 + 体积合理 + 含汉化标记”
// 全部满足时才会被采用；否则退回编译期内嵌版本。
func verifyExternalOverlay(path string, data []byte) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	return verifyExternalOverlayInfo(info, data)
}

func verifyExternalOverlayInfo(info os.FileInfo, data []byte) error {
	if !info.Mode().IsRegular() {
		return fmt.Errorf("不是普通文件（符号链接或特殊文件）")
	}
	if info.Mode().Perm()&0o022 != 0 {
		return fmt.Errorf("同组或其他用户可写 (%#o)", info.Mode().Perm())
	}
	if err := verifyOverlayOwner(info); err != nil {
		return err
	}
	if info.Size() > maxExternalOverlaySize || len(data) > maxExternalOverlaySize {
		return fmt.Errorf("体积超出上限 (%d 字节)", info.Size())
	}
	if !strings.Contains(string(data), overlaySentinel) {
		return fmt.Errorf("内容缺少汉化脚本标记 %q", overlaySentinel)
	}
	return nil
}

func readExternalOverlay(path string, allowExternal bool) (string, error) {
	// The trust flag skips provenance checks, never size/type/read safety.
	data, info, err := readRegularFile(path, maxExternalOverlaySize)
	if err != nil {
		return "", err
	}
	if !allowExternal {
		if err := verifyExternalOverlayInfo(info, data); err != nil {
			return "", err
		}
		parent, err := os.Stat(filepath.Dir(path))
		if err != nil {
			return "", err
		}
		if err := verifyOverlayOwner(parent); err != nil {
			return "", err
		}
	}
	return string(data), nil
}

// LoadOverlaySource 优先从可执行文件同目录加载外部 JS，若不存在或校验不通过则使用编译期内嵌版本。
// allowExternal 为 true 时跳过安全校验（对应 --trust-external-overlay，仅建议在本机调试词库时使用）。
func LoadOverlaySource(allowExternal bool) string {
	exePath, err := os.Executable()
	if err == nil {
		exeDir := filepath.Dir(exePath)
		extPath := filepath.Join(exeDir, externalOverlayName)
		if source, err := readExternalOverlay(extPath, allowExternal); err == nil {
			if allowExternal {
				fmt.Printf("[警告] 已显式信任外部汉化脚本: %s (SHA-256 %s)\n", extPath, overlayFingerprint(source))
			} else {
				fmt.Printf("[提示] 使用外部汉化脚本: %s (SHA-256 %s)\n", extPath, overlayFingerprint(source))
			}
			return source
		} else if !os.IsNotExist(err) {
			fmt.Printf("[警告] 已忽略外部汉化脚本: %v\n       路径: %s\n", err, extPath)
		}
	}
	// 兜底：使用内嵌版本
	fmt.Printf("[提示] 使用内嵌汉化脚本 (SHA-256 %s)\n", overlayFingerprint(string(embeddedOverlayJS)))
	return string(embeddedOverlayJS)
}
