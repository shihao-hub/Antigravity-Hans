# /// script
# requires-python = ">=3.10"
# dependencies = []
# ///
# 编译 Antigravity-Hans GUI（Windows 壳，Fyne）：
#
# 产物命名规则（遵循仓库 dev/release 隔离规范）：
#   正式版（release）：dist/antigravity-hans-gui-windows-amd64.exe
#   开发版（dev）：    dist/antigravity-hans-gui-dev-windows-amd64.exe
#
# 构建模式判定（决定开机自启是否可用及产物命名）：
#   当前提交带 git tag（git describe --tags --exact-match HEAD）且工作区干净 -> release；
#   否则 dev。
#   可用 --dev 强制按开发版构建，或用 --force-release 强制按正式版构建。
#
# 用法：
#   uv run scripts/build-gui.py                # 自动判定（有 tag 且干净工作区为 release，否则为 dev）
#   uv run scripts/build-gui.py --dev          # 显式构建开发版（产物自动带 -dev 后缀）
#   uv run scripts/build-gui.py --force-release # 强制构建正式版（本地验证开机自启；勿用于正式分发）
#   uv run scripts/build-gui.py --show-console # 保留控制台窗口（调试排查用）

import argparse
import os
import shutil
import subprocess
import sys
from datetime import datetime
from pathlib import Path

PROJECT_ROOT = Path(__file__).resolve().parent.parent
DIST_DIR = PROJECT_ROOT / "dist"

if sys.stdout.encoding and sys.stdout.encoding.lower() not in ("utf-8", "utf8"):
    sys.stdout.reconfigure(encoding="utf-8", errors="replace", line_buffering=True)
if sys.stderr.encoding and sys.stderr.encoding.lower() not in ("utf-8", "utf8"):
    sys.stderr.reconfigure(encoding="utf-8", errors="replace")


def run(cmd: list[str], cwd: Path, env: dict | None = None) -> None:
    executable = shutil.which(cmd[0])
    if executable is None:
        raise RuntimeError(f"未找到命令：{cmd[0]}（请确认已安装并在 PATH 上）")
    result = subprocess.run([executable, *cmd[1:]], cwd=str(cwd), env=env)
    if result.returncode != 0:
        raise RuntimeError(f"命令失败（exit {result.returncode}）：{' '.join(cmd)}")


def git_output(args: list[str]) -> str | None:
    try:
        r = subprocess.run(["git", *args], cwd=str(PROJECT_ROOT), capture_output=True, text=True)
    except OSError:
        return None
    if r.returncode != 0:
        return None
    return r.stdout.strip()


def detect_release(force_release: bool, force_dev: bool = False) -> tuple[bool, str]:
    if force_dev:
        return False, "强制开发版（--dev）"
    if force_release:
        return True, "强制 release（--force-release）"
    tag = git_output(["describe", "--tags", "--exact-match", "HEAD"])
    if not tag:
        return False, "当前提交没有 git tag"
    if git_output(["status", "--porcelain"]):
        return False, "工作区有未提交改动（tag 提交需干净）"
    return True, f"命中 tag {tag}"


def ensure_icon() -> None:
    syso = PROJECT_ROOT / "rsrc_windows_amd64.syso"
    ico = PROJECT_ROOT / "icon.ico"
    if syso.exists() and ico.exists() and syso.stat().st_mtime >= ico.stat().st_mtime:
        return
    if not ico.exists():
        print("[警告] 未找到 icon.ico 与 rsrc_windows_amd64.syso，产物将不带图标")
        return
    rsrc = shutil.which("rsrc") or str(Path.home() / "go" / "bin" / "rsrc.exe")
    if not Path(rsrc).exists():
        print("[警告] 未找到 rsrc 工具，无法生成 syso，产物将不带图标")
        return
    print(f"[提示] 生成 Windows 资源对象：{syso.name}")
    run([rsrc, "-ico", str(ico), "-o", str(syso)], PROJECT_ROOT)


def main() -> int:
    parser = argparse.ArgumentParser(description="编译 Antigravity-Hans GUI（Windows 壳）")
    parser.add_argument("--dev", action="store_true", help="强制按 dev 开发版构建（产物自动带 -dev 后缀）")
    parser.add_argument("--force-release", action="store_true", help="强制按 release 构建（本地验证开机自启用）")
    parser.add_argument("--show-console", action="store_true", help="保留控制台窗口（调试用，默认 -H windowsgui）")
    args = parser.parse_args()

    try:
        env = os.environ.copy()
        env.update({"CGO_ENABLED": "1", "GOOS": "windows", "GOARCH": "amd64"})
        if sys.platform == "win32":
            if shutil.which("gcc") is None:
                raise RuntimeError("未找到 gcc：Fyne Windows 构建需要 C 编译器（推荐 MSYS2 MinGW64 并加入 PATH）")
        else:
            cc = shutil.which("x86_64-w64-mingw32-gcc")
            if cc is None:
                raise RuntimeError("非 Windows 主机构建需 mingw-w64（提供 x86_64-w64-mingw32-gcc）")
            env["CC"] = cc

        release, reason = detect_release(args.force_release, args.dev)
        mode = "release" if release else "dev"

        version = ""
        version_file = PROJECT_ROOT / "VERSION"
        if version_file.exists():
            version = version_file.read_text(encoding="utf-8").strip()
        if release:
            tag = git_output(["describe", "--tags", "--exact-match", "HEAD"])
            if tag:
                version = tag
        if not version:
            version = "dev"
        if mode == "dev" and not version.endswith("-dev"):
            version += "-dev"

        ensure_icon()

        DIST_DIR.mkdir(parents=True, exist_ok=True)
        out_name = "antigravity-hans-gui-windows-amd64.exe" if release else "antigravity-hans-gui-dev-windows-amd64.exe"
        out_exe = DIST_DIR / out_name

        gui_flag = "" if args.show_console else "-H windowsgui"
        ldflags = f"{gui_flag} -s -w -X main.Version={version}".strip()
        if release:
            ldflags += " -X main.ReleaseMode=release"

        print(f"[提示] 构建模式：{mode}（{reason}）")
        print(f"[提示] 版本号：{version}")
        print(f"[提示] 开始编译 -> {out_exe}")

        run(
            ["go", "build", "-tags", "gui", "-trimpath", "-buildvcs=false",
             "-ldflags", ldflags, "-o", str(out_exe), "."],
            PROJECT_ROOT,
            env=env,
        )

        stat = out_exe.stat()
        print(f"[成功] GUI 构建完成：{out_exe}")
        print(f"        大小 {stat.st_size / 1024 / 1024:.1f} MB，时间 {datetime.fromtimestamp(stat.st_mtime):%Y-%m-%d %H:%M:%S}")
        return 0
    except RuntimeError as exc:
        print(f"[错误] {exc}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    sys.exit(main())
