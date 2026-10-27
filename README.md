# Antigravity 汉化工具 (Antigravity-Hans)

专为 **Google Antigravity** 及 **Antigravity IDE** 开发的中文汉化工具。

> **适配版本**：**Antigravity `2.26.0`** 与 **Antigravity IDE `2.5.5`**。

---

## 效果预览

<img width="2560" height="1600" alt="Antigravity 汉化预览 1" src="https://github.com/user-attachments/assets/4afcc033-34e4-47f7-bca1-53c341780b36" />
<img width="2560" height="1600" alt="Antigravity 汉化预览 2" src="https://github.com/user-attachments/assets/db2173c3-6c92-493c-bcb0-d3235133a381" />

---

## 快速使用

前往 [Releases](../../releases) 下载对应平台的单文件即可运行：

- **Windows（GUI 版，推荐）**：双击 `antigravity-hans-gui-windows-amd64.exe`，无终端窗口；主页面点击按钮即可启动汉化，托盘常驻。
- **Windows（CLI 版）**：双击 `antigravity-hans-windows-amd64.exe`，在交互菜单中选择对应模式。
- **macOS**：
  ```bash
  chmod +x antigravity-hans-macos-*
  ./antigravity-hans-macos-arm64  # Apple Silicon
  ./antigravity-hans-macos-amd64  # Intel
  ```

```text
Antigravity-Hans 启动器
----------------------------------------
1. 启动 Antigravity (动态汉化)
2. 启动 Antigravity IDE (动态汉化)
3. 静态补丁管理 (仅支持 IDE)
4. 生成桌面快捷启动方式
0. 退出
----------------------------------------
```

> **提示**：首次使用 IDE 时，工具会自动检测并提示安装[官方中文语言包 (Chinese (Simplified) Language Pack)](https://marketplace.visualstudio.com/items?itemName=MS-CEINTL.vscode-language-pack-zh-hans)，确保菜单栏与底层编辑器完整汉化。

---

## 命令行参数

```bash
# 直接启动汉化
antigravity-hans --app         # 动态汉化 Antigravity
antigravity-hans --ide         # 动态汉化 Antigravity IDE
antigravity-hans --patch       # 静态补丁管理 (仅支持 IDE)

# 生成一键汉化快捷方式
antigravity-hans --shortcut        # 为两者生成快捷方式
antigravity-hans --shortcut --app  # 仅 Antigravity
antigravity-hans --shortcut --ide  # 仅 Antigravity IDE

# 维护
antigravity-hans --stop                    # 结束由本工具启动的汉化监视器
antigravity-hans --trust-external-overlay  # 显式信任同目录外部汉化脚本（类型与大小限制仍生效）
```

---

## GUI 版（Windows）

Fyne 实现的无终端窗口程序（`-H windowsgui` 子系统，不创建控制台）：

- **主页面**：一键启动 Antigravity 动态汉化（与 CLI 同源的 CDP 注入逻辑）。
- **系统托盘**：显示主窗口、打开 exe 所在目录、开机自启动、退出。
- **关闭主窗口仅隐藏到托盘**，后台监视持续；退出请使用托盘菜单。
- **开机自启动仅正式版可用**：以 git tag 构建的 release 版本才能注册（写入 `HKCU\...\Run` 的 `"<exe>" --hidden`）；开发构建（无 tag / 工作区有改动）菜单置灰并拒绝注册，双实例互不冲突。
- **快捷方式**：同目录存在 GUI exe 时，CLI 为 Antigravity 生成的桌面快捷方式会自动指向 GUI（参数 `--start --hidden`），双击不再闪现终端，直接静默启动汉化。启动参数支持手工传入：`--start` 启动即开始汉化，`--hidden` 仅托盘不弹窗。

GUI 构建与图标生成（本机需 gcc，MSYS2 MinGW64 加入 PATH）：

```bash
uv run scripts/render_icon.py              # 渲染图标并编译 Windows syso 资源对象（支持 --variant 1..4）
uv run scripts/build-gui.py                # 开发构建 -> dist/antigravity-hans-gui-windows-amd64.exe
uv run scripts/build-gui.py --force-release  # 强制 release（本地验证开机自启；勿用于正式分发）
```

## 汉化模式

1. **动态汉化（推荐）**：
   - 支持 Antigravity 及 Antigravity IDE。
   - 通过 CDP 动态翻译界面，保留应用安装资源；注入状态随应用页面关闭而消失。
   - 可通过 `--shortcut` 参数一键生成桌面快捷方式，日常双击即可自动启动汉化。
2. **静态补丁（仅支持 Antigravity IDE）**：
   - 直接修改本地 IDE 安装包资源并修复校验和，一劳永逸，支持随时一键还原。

---

## 常见问题

- **macOS 双击「中文」图标后没有汉化**：监视进程日志位于 `~/Library/Logs/antigravity-hans.log`，先看该文件。
- **提示汉化监视器已在运行**：说明上一次的监视器仍在工作，可用 `antigravity-hans --stop` 结束它。
- **提示退出取消或超时**：仅在没有可复用调试端口时请求正常退出（macOS 用 AppleScript，Windows 用不带 `/F` 的 `taskkill`）。取消、拒绝或超时会中止重启，不会强杀应用；请保存内容并手动退出后重试。
- **更新后仍使用旧词库**：先结束旧监视器；如果日志提示旧脚本没有清理入口，重新加载当前应用窗口后再启动新版监视器。

---

## 源码编译

需要 **Go 1.27.3 或更高版本**（`go.mod` 已声明，旧工具链会自动下载对应版本）。

```bash
./build.sh                       # CLI（macOS/Windows 交叉编译）
uv run scripts/build-gui.py      # Windows GUI（Fyne，需 gcc）
```

## 发布（维护者）

```bash
# 1. 更新 VERSION 并提交；2. 打 tag 并推送（tag 即 release）
git tag v0.4.8 && git push origin v0.4.8
```

推送 `v*` tag 后，GitHub Actions 自动构建 CLI（三平台）与 Windows GUI（release 模式）并发布 Release。

---

## 参与贡献

由于学业繁忙，更新可能不够及时，敬请谅解。欢迎提交 PR 补充词条、适配新版本或修复问题。

新版本适配与词条维护流程见 [适配技能](.github/skills/antigravity-hans-adaptation/SKILL.md)，提交前请运行词库 lint 与回归测试。

---

## 致谢

- 感谢所有为本项目提交代码、文档、测试、Issue 的贡献者。

<a href="https://github.com/yuexps/Antigravity-Hans/graphs/contributors">
  <img src="https://contrib.rocks/image?repo=yuexps/Antigravity-Hans" alt="contributors" />
</a>

---

- 汉化词库参考：[antigravity-chinese](https://github.com/kdczyz/antigravity-chinese)
