//go:build gui && windows

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
)

const (
	// defaultUpdateRepo 默认自更新检查的 GitHub 仓库
	defaultUpdateRepo = "shihao-hub/Antigravity-Hans"
	// updateAssetName Windows GUI 发布文件名
	updateAssetName = "antigravity-hans-gui-windows-amd64.exe"
)

// githubAsset 单个发布附件元数据
type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

// githubRelease GitHub Release API 返回结构
type githubRelease struct {
	TagName string        `json:"tag_name"`
	Name    string        `json:"name"`
	Body    string        `json:"body"`
	Assets  []githubAsset `json:"assets"`
}

// cleanOldVersions 在启动时清理上次重命名置换遗留的 .old 和未完成的 .download 文件。
// 此时旧进程已完全终止，NTFS 文件锁已解开，可安全删除。
func cleanOldVersions() {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	cleanExe := filepath.Clean(exe)
	_ = os.Remove(cleanExe + ".old")
	_ = os.Remove(cleanExe + ".download")
}

// parseSemVer 解析形如 "v1.2.3" 或 "1.2.3" 或 "v1.2.3-dev" 的语义版本为数字切片。
func parseSemVer(v string) ([]int, bool) {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "v")
	v = strings.TrimPrefix(v, "V")
	if idx := strings.IndexAny(v, "-+"); idx != -1 {
		v = v[:idx]
	}
	parts := strings.Split(v, ".")
	nums := make([]int, len(parts))
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return nil, false
		}
		nums[i] = n
	}
	return nums, true
}

// isNewerVersion 判定最新 release tag 是否新于当前程序版本。
func isNewerVersion(currentVer, latestTag string) bool {
	if strings.EqualFold(currentVer, "dev") || strings.Contains(currentVer, "-dev") {
		// 开发版允许升级到任何合法发布的版本，方便调试
		return strings.TrimSpace(latestTag) != ""
	}
	curNums, okCur := parseSemVer(currentVer)
	latNums, okLat := parseSemVer(latestTag)
	if !okCur || !okLat {
		return currentVer != latestTag
	}
	maxLen := len(curNums)
	if len(latNums) > maxLen {
		maxLen = len(latNums)
	}
	for i := 0; i < maxLen; i++ {
		cur := 0
		if i < len(curNums) {
			cur = curNums[i]
		}
		lat := 0
		if i < len(latNums) {
			lat = latNums[i]
		}
		if lat > cur {
			return true
		}
		if lat < cur {
			return false
		}
	}
	return false
}

// fetchLatestRelease 请求 GitHub API 获取指定仓库的最新发布信息。
func fetchLatestRelease(repo string) (*githubRelease, error) {
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)
	if customURL := os.Getenv("ANTIGRAVITY_UPDATE_API_URL"); customURL != "" {
		apiURL = customURL
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("User-Agent", "Antigravity-Hans-Updater")
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("连接 GitHub 失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API 返回状态码 %d", resp.StatusCode)
	}

	var rel githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, fmt.Errorf("解析发布信息失败: %w", err)
	}
	return &rel, nil
}

// downloadAsset 将文件下载到指定路径，支持进度回调。
func downloadAsset(ctx context.Context, downloadURL, destPath string, progressCb func(downloaded, total int64)) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Antigravity-Hans-Updater")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("下载返回状态码: %d", resp.StatusCode)
	}

	total := resp.ContentLength
	out, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer out.Close()

	buf := make([]byte, 64*1024)
	var downloaded int64
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := out.Write(buf[:n]); werr != nil {
				return werr
			}
			downloaded += int64(n)
			if progressCb != nil {
				progressCb(downloaded, total)
			}
		}
		if rerr != nil {
			if rerr == io.EOF {
				break
			}
			return rerr
		}
	}
	return nil
}

// inPlaceReplaceAndRestart 核心黑魔法：重命名置换法 (Rename-then-Replace)。
// 1. 将运行中的自身 exe 改名为 .old（NTFS 允许运行中改名，旧进程不受影响）；
// 2. 将已下载的新文件改名为原本的 exe 名字；
// 3. 释放自身单实例互斥体，拉起新版 exe；
// 4. 当前旧进程退出，新进程启动后自动清理 .old。
func inPlaceReplaceAndRestart(downloadedPath string) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("获取当前程序路径失败: %w", err)
	}
	cleanExe := filepath.Clean(exe)
	oldExe := cleanExe + ".old"

	// 如果残留上次 .old，先尝试清除
	_ = os.Remove(oldExe)

	// 步骤 1：重命名运行中的自身
	if err := os.Rename(cleanExe, oldExe); err != nil {
		return fmt.Errorf("重命名旧版本失败: %w", err)
	}

	// 步骤 2：将新版就位
	if err := os.Rename(downloadedPath, cleanExe); err != nil {
		// 置换失败，紧急回滚
		_ = os.Rename(oldExe, cleanExe)
		return fmt.Errorf("新版本就位失败: %w", err)
	}

	// 步骤 3：主动提前释放互斥体，避免新进程启动时遭遇互斥体碰撞
	releaseSingleInstance()

	// 步骤 4：启动新版本进程
	cmd := exec.Command(cleanExe, os.Args[1:]...)
	cmd.Dir = filepath.Dir(cleanExe)
	if err := cmd.Start(); err != nil {
		// 启动失败，尽量回滚保护
		_ = os.Rename(cleanExe, downloadedPath)
		_ = os.Rename(oldExe, cleanExe)
		return fmt.Errorf("拉起新版本失败: %w", err)
	}

	// 步骤 5：旧进程优雅自尽
	os.Exit(0)
	return nil
}

// checkUpdateAsync 异步检查更新并弹窗提示
func (g *guiApp) checkUpdateAsync(silent bool) {
	go func() {
		if !silent {
			g.setBusy(true, "正在检查更新...")
		}
		// sh-ai-todo[261006][n] 由于国内访问 github 仓库容易出现网络问题，所以首先是提示用户，其次是我觉得需要考虑镜像网站
		rel, err := fetchLatestRelease(defaultUpdateRepo)
		if !silent {
			g.setBusy(false, "就绪")
		}
		if err != nil {
			if !silent {
				g.showErrorDialog("检查更新失败", err)
			}
			return
		}

		var targetAsset *githubAsset
		for i := range rel.Assets {
			if strings.EqualFold(rel.Assets[i].Name, updateAssetName) {
				targetAsset = &rel.Assets[i]
				break
			}
		}

		if targetAsset == nil {
			if !silent {
				g.showInfoDialog("检查更新", fmt.Sprintf("发现新发布 %s，但未找到对应的 GUI 安装包 (%s)。", rel.TagName, updateAssetName))
			}
			return
		}

		if !isNewerVersion(Version, rel.TagName) {
			if !silent {
				g.showInfoDialog("检查更新", fmt.Sprintf("当前已是最新版本 (%s)。", Version))
			}
			return
		}

		sizeMB := float64(targetAsset.Size) / (1024 * 1024)
		msg := fmt.Sprintf("发现新版本：%s（当前版本：%s）\n大小：%.1f MB\n\n更新内容：\n%s\n\n是否立即下载并自动升级？",
			rel.TagName, Version, sizeMB, rel.Body)

		g.showConfirmDialog("发现新版本", msg, func(ok bool) {
			if ok {
				go g.performUpdate(rel, targetAsset)
			}
		})
	}()
}

// performUpdate 执行下载、重命名置换与无缝接管升级
func (g *guiApp) performUpdate(rel *githubRelease, asset *githubAsset) {
	g.setBusy(true, "正在准备下载新版本...")
	exe, err := os.Executable()
	if err != nil {
		g.setBusy(false, "就绪")
		g.showErrorDialog("升级失败", fmt.Errorf("获取程序路径失败: %w", err))
		return
	}
	cleanExe := filepath.Clean(exe)
	downloadPath := cleanExe + ".download"
	_ = os.Remove(downloadPath)

	err = downloadAsset(context.Background(), asset.BrowserDownloadURL, downloadPath, func(downloaded, total int64) {
		if total > 0 {
			percent := float64(downloaded) / float64(total) * 100
			g.setBusy(true, fmt.Sprintf("正在下载更新 (%.1f%%)...", percent))
		}
	})
	if err != nil {
		_ = os.Remove(downloadPath)
		g.setBusy(false, "就绪")
		g.showErrorDialog("下载失败", fmt.Errorf("下载新版本失败: %w", err))
		return
	}

	g.setBusy(true, "正在应用新版本并无缝重启...")
	if err := inPlaceReplaceAndRestart(downloadPath); err != nil {
		_ = os.Remove(downloadPath)
		g.setBusy(false, "就绪")
		g.showErrorDialog("升级失败", err)
	}
}

// showInfoDialog 在主线程展示提示框，并保证窗口唤起
func (g *guiApp) showInfoDialog(title, message string) {
	fyne.Do(func() {
		g.showMainWindow()
		dialog.ShowInformation(title, message, g.win)
	})
}

// showErrorDialog 在主线程展示错误框，并保证窗口唤起
func (g *guiApp) showErrorDialog(title string, err error) {
	fyne.Do(func() {
		g.showMainWindow()
		dialog.ShowError(err, g.win)
	})
}

// showConfirmDialog 在主线程展示确认框，并保证窗口唤起
func (g *guiApp) showConfirmDialog(title, message string, callback func(bool)) {
	fyne.Do(func() {
		g.showMainWindow()
		dialog.ShowConfirm(title, message, callback, g.win)
	})
}
