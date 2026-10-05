# TODO 待办清单

本项目用于跟踪待排期的工程优化、重构建议与功能演进任务。

---

## 待排期任务

### 1. 解耦 CLI 与 GUI 入口架构（消除 Build Tag 互斥与 gopls 告警）

- **实施计划文档**：[`plans/01-decouple-gui-cli-entrypoints.md`](plans/01-decouple-gui-cli-entrypoints.md)
- **当前状态**：待排期 / 暂不实施
- **当前临时方案**：
  在 `.vscode/settings.json`（或 `.zed/settings.json`）中配置 `gopls` 构建标签垫片，避免当前开发 GUI 模块时报 `No packages found for open file`：
  ```json
  {
    "go.buildFlags": ["-tags=gui"],
    "gopls": {
      "build.buildFlags": ["-tags=gui"]
    }
  }
  ```
- **核心问题与背景**：
  - 根目录下 `main.go`（`//go:build !gui`）与 `gui_windows.go`（`//go:build gui && windows`）共处同一目录 `package main`，通过构建标签互斥隔离。
  - 语言服务器 `gopls` 在单一标签上下文中无法同时加载双端入口，必然导致一侧提示被 build tags 排除。
- **长远优化目标**：
  - 将公共逻辑（CDP、Launcher、Patch、Shortcut 等）下沉至 `internal/core/`。
  - 将双端入口拆分到独立的 main 包目录：
    - `cmd/antigravity-hans/`：CLI 控制台入口（移除 `!gui` 标签）
    - `cmd/antigravity-hans-gui/`：Windows GUI 托盘入口（移除 `gui` 标签，保留系统平台标签）
  - 更新 `build.sh`、`scripts/build-gui.py` 与 CI 流水线，实现 IDE 零配置下双端完整补全与无冲突构建。
- **任务概览**：
  - [ ] Task 1: 创建 `internal/core` 共享核心包并迁移公共模块
  - [ ] Task 2: 抽取 CLI 入口至 `cmd/antigravity-hans`
  - [ ] Task 3: 抽取 GUI 入口至 `cmd/antigravity-hans-gui`
  - [ ] Task 4: 更新编译脚本与 CI/CD 发布流水线
  - [ ] Task 5: 清理临时 LSP 配置与接线收尾
