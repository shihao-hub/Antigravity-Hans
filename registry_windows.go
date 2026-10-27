//go:build windows

package main

import (
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

// GetPathsFromRegistry 从 Windows 注册表查询可执行文件的所有可能路径
func GetPathsFromRegistry(appName, exeName string) []string {
	found := make(map[string]struct{})

	roots := []registry.Key{registry.CURRENT_USER, registry.LOCAL_MACHINE}

	// 1. 优先从 App Paths 检索（效率高）
	subKey := `Software\Microsoft\Windows\CurrentVersion\App Paths\` + exeName
	for _, root := range roots {
		k, err := registry.OpenKey(root, subKey, registry.QUERY_VALUE)
		if err != nil {
			continue
		}
		val, _, err := k.GetStringValue("")
		k.Close()
		if err != nil || val == "" {
			continue
		}
		val = strings.Trim(val, `'"`)
		// 只接受与目标可执行文件同名的路径：注册表内容不受本工具控制
		if !strings.EqualFold(filepath.Base(val), exeName) {
			continue
		}
		if _, err := os.Stat(val); err == nil {
			found[val] = struct{}{}
		}
	}

	// 2. 兜底扫描 Uninstall 列表
	uninstallKey := `Software\Microsoft\Windows\CurrentVersion\Uninstall`
	for _, root := range roots {
		k, err := registry.OpenKey(root, uninstallKey, registry.ENUMERATE_SUB_KEYS|registry.QUERY_VALUE)
		if err != nil {
			continue
		}
		subKeys, err := k.ReadSubKeyNames(-1)
		k.Close()
		if err != nil {
			continue
		}

		for _, subKeyName := range subKeys {
			fullSubKey := uninstallKey + `\` + subKeyName
			sk, err := registry.OpenKey(root, fullSubKey, registry.QUERY_VALUE)
			if err != nil {
				continue
			}

			displayName, _, _ := sk.GetStringValue("DisplayName")
			if displayName == "" {
				displayName = subKeyName
			}

			appNameLower := strings.ToLower(appName)
			if !strings.Contains(strings.ToLower(displayName), appNameLower) &&
				!strings.Contains(strings.ToLower(subKeyName), appNameLower) {
				sk.Close()
				continue
			}

			// 尝试 InstallLocation
			if loc, _, err := sk.GetStringValue("InstallLocation"); err == nil && loc != "" {
				loc = strings.Trim(loc, `'"`)
				fullPath := filepath.Join(loc, exeName)
				if _, err := os.Stat(fullPath); err == nil {
					found[fullPath] = struct{}{}
				}
			}

			// 尝试 DisplayIcon
			if icon, _, err := sk.GetStringValue("DisplayIcon"); err == nil && icon != "" {
				parts := strings.SplitN(icon, ",", 2)
				iconPath := strings.Trim(parts[0], `'"`)
				if strings.HasSuffix(strings.ToLower(iconPath), ".exe") &&
					strings.EqualFold(filepath.Base(iconPath), exeName) {
					if _, err := os.Stat(iconPath); err == nil {
						found[iconPath] = struct{}{}
					}
				}
			}

			sk.Close()
		}
	}

	result := make([]string, 0, len(found))
	for p := range found {
		result = append(result, p)
	}
	return result
}
