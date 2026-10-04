# Antigravity 汉化工具 (Antigravity-Hans)

专为 **Google Antigravity** 及 **Antigravity IDE** 开发的中文汉化工具。

> **适配版本**： **Antigravity `2.10.0`** 与 **Antigravity IDE `2.5.5`** 。

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
   - 基于 CDP 远程调试协议无感注入，不修改任何本地二进制与文件，安全无痕。
   - 可通过 `--shortcut` 参数一键生成桌面快捷方式，日常双击即可自动启动汉化。
2. **静态补丁（仅支持 Antigravity IDE）**：
   - 直接修改本地 IDE 安装包资源并修复校验和，一劳永逸，支持随时一键还原。

---

## 源码编译

```bash
./build.sh                       # CLI（macOS/Windows 交叉编译）
uv run scripts/build-gui.py      # Windows GUI（Fyne，需 gcc）
```

## 发布（维护者）

```bash
# 1. 更新 VERSION 并提交；2. 打 tag 并推送（tag 即 release）
git tag v0.4.4 && git push origin v0.4.4
```

推送 `v*` tag 后，GitHub Actions 自动构建 CLI（三平台）与 Windows GUI（release 模式）并发布 Release。

---

## 致谢

- 汉化词库参考：[antigravity-chinese](https://github.com/kdczyz/antigravity-chinese)

