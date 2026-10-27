# 普通版 2.26.0 与 IDE 2.5.5 本地更新

本次同时覆盖 Antigravity 普通版 `2.26.0` 与 Antigravity IDE `2.5.5`，并新增版本适配技能与辅助脚本。README 顶部适配版本已同步为 `2.26.0` / `2.5.5`。

## 普通版 2.26.0

词库扩充至 1338 条，整表改为按源文长度倒序。新增内容覆盖设置页说明、执行策略、沙盒与自主运行、Best of N 分支、浏览器工具、演示模式、额度与自定义预算等文案；其中含变量的句子改为动态规则，保留数字、模型名和配额信息。

配套调整：

- `port.go` 的随机调试端口注释同步到 2.26.0。
- `tools/lint-overlay.mjs` 增加“源文长度必须单调不增”校验，防止词库再次出现分区或随意追加。
- 回归测试新增 `2.26.0 normal app settings and popovers are localized` 与分段提示重排用例。

## IDE 2.5.5

补充外壳与设置页文案：工作流、规则、自定义项、AI 快捷方式、智能体对话、返回智能体、设置标题等；同时修正状态栏、辅助栏标题与弹出层的翻译范围。

行为修复：

- 主窗口白名单纳入状态栏、辅助栏标题、设置容器及挂在 body 下的 portal 层，避免异步渲染的设置页被过滤。
- 支持合并大小写变体词条后的小写回退匹配。
- 含独立数字节点的安全模式句子可正确翻译。
- MutationObserver 的多批变更合并到同一计时器，取消时丢弃待处理任务。
- 文本节点不足 3 个的元素不进入分段句子重排。

为合并重复与变体词条，IDE 适配后词库为 1328 条；随后合入设置页漏译修复，最终为 1336 条。

## 设置页漏译修复

补齐设置页剩余标签与说明，并对 React 拆分的句子按节点重写，保留行内链接及其目标、事件处理器。项目删除提示保留任意项目名与对话数量；项目权限提示只翻译固定模板，未知布局保持原文。相关回归测试覆盖重复处理、链接身份和跳过内容。

## 适配技能与工具

新增 `.github/skills/antigravity-hans-adaptation/`：

- `SKILL.md`：版本确认、文案收集、词条设计、overlay 修改、回归测试、版本同步与验证流程。
- `references/overlay-guide.md`：词库结构、匹配流水线、动态文案、权限模板、DOM 范围、指纹与两个应用的差异。
- `scripts/antigravity-resources.mjs`：安装识别、静态资源检索、词库覆盖检查、IDE 候选扫描、运行页面文本抓取。

覆盖判断直接加载 overlay 并调用真实 `translate`，与界面实际翻译结果一致，避免静态规则与动态模板脱节。`live` 只读取 DOM 文本及 `aria-label`、`placeholder`、`title`，不修改页面。

README 的“参与贡献”章节增加技能入口与提交前验证提示。

## 资源指纹

本机安装文件 SHA-256：

```
普通版 resources/app.asar
8ce27ff7c2defa10ed1ed2817288a61c546690f4cae1b1a8c91d4e69e0851e4d

IDE resources/app/out/main.js
6f1a12099ac635d462bd7092357119ec1ec7516988cacb874b7e97691272b104

IDE resources/app/out/vs/workbench/workbench.desktop.main.js
f4bd347d94be4634d2adeec8b3ef65e4d65d7dd0d72281b0e1982fe9c71fe6a7
```

未将原应用资源文件复制到仓库。

## 验证结果

- `node tools/lint-overlay.mjs antigravity-hans-overlay.js`：1336 条，0 错误，0 警告。
- `node --test tools/overlay.test.mjs`：19 项全部通过。
- `go vet ./...`、`go test -count=1 ./...`：通过。
- 辅助脚本在本机识别出普通版 `2.26.0`、IDE `2.5.5`，并通过模拟 CDP 服务验证 `live` 的抓取与过滤。

未执行三平台交叉编译，未对 Windows 实际界面做端到端验证，也未重启或改动本机应用安装资源。
