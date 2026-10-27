# /// script
# requires-python = ">=3.10"
# dependencies = []
# ///
"""
mock_update_server.py - 本地 GitHub Release 仿真服务

用于无需上传 GitHub、无网络延迟与限制、完全离线闭环测试 Antigravity-Hans 的 Windows 自动更新机制。
原理：
1. 拦截 release API 请求，返回与 GitHub Release API 结构完全一致的 JSON；
2. 托管目标新版本 exe，提供带 Content-Length 的流式下载端点（驱动客户端下载百分比进度条）；
3. 客户端设置环境变量 ANTIGRAVITY_UPDATE_API_URL 即可无感接入。

用法示例：
  # 1. 模拟向客户端推送 v0.4.6 新版本（目标 exe 为 dist/new_app.exe）
  uv run scripts/mock_update_server.py --port 8999 --target-exe dist/antigravity-hans-gui-windows-amd64.exe --target-tag v0.4.6

  # 2. 模拟弱网环境（限速 5MB/s，方便看清 GUI 进度条跳动）
  uv run scripts/mock_update_server.py --port 8999 --target-exe dist/antigravity-hans-gui-windows-amd64.exe --target-tag v0.4.6 --speed-mb 5
"""

import argparse
import json
import os
import sys
import time
from http import HTTPStatus
from http.server import HTTPServer, BaseHTTPRequestHandler
from pathlib import Path
from socketserver import ThreadingMixIn

UPDATE_ASSET_NAME = "antigravity-hans-gui-windows-amd64.exe"


class ThreadedHTTPServer(ThreadingMixIn, HTTPServer):
    daemon_threads = True


class MockUpdateHandler(BaseHTTPRequestHandler):
    target_exe: Path = Path()
    target_tag: str = "v0.4.6"
    target_body: str = ""
    speed_mb: float = 0.0  # 0 为不限速

    def log_message(self, format, *args):
        # 格式化输出日志，增强可读性
        sys.stdout.write(f"[{time.strftime('%H:%M:%S')}] [MockServer] {format % args}\n")
        sys.stdout.flush()

    def do_GET(self):
        # 统一处理路径（去掉查询参数）
        clean_path = self.path.split("?")[0].rstrip("/")

        # 1. 下载接口
        if clean_path.endswith(f"/{UPDATE_ASSET_NAME}") or clean_path == "/download":
            self.handle_download()
            return

        # 2. Release 元数据接口（支持根路径、/api/release 或任何 GitHub API 兼容路径）
        self.handle_release_info()

    def handle_release_info(self):
        if not self.target_exe.exists():
            self.send_error(HTTPStatus.NOT_FOUND, f"目标文件不存在: {self.target_exe}")
            return

        file_size = self.target_exe.stat().st_size
        host = self.headers.get("Host", f"127.0.0.1:{self.server.server_port}")
        download_url = f"http://{host}/download/{UPDATE_ASSET_NAME}"

        payload = {
            "tag_name": self.target_tag,
            "name": f"Antigravity-Hans {self.target_tag} (本地仿真发布)",
            "body": self.target_body or (
                f"### 本地自更新仿真测试 ({self.target_tag})\n\n"
                f"- **目标产物**：`{UPDATE_ASSET_NAME}`\n"
                f"- **文件大小**：{file_size / (1024 * 1024):.2f} MB\n"
                f"- **测试重点**：NTFS 重命名置换法、互斥体主动交接、无缝拉起新进程与旧文件自动清理"
            ),
            "assets": [
                {
                    "name": UPDATE_ASSET_NAME,
                    "browser_download_url": download_url,
                    "size": file_size,
                }
            ],
        }

        body_bytes = json.dumps(payload, ensure_ascii=False, indent=2).encode("utf-8")
        self.send_response(HTTPStatus.OK)
        self.send_header("Content-Type", "application/json; charset=utf-8")
        self.send_header("Content-Length", str(len(body_bytes)))
        self.end_headers()
        self.wfile.write(body_bytes)
        self.log_message("已响应版本检查请求: 返回版本 %s，大小 %.2f MB", self.target_tag, file_size / (1024 * 1024))

    def handle_download(self):
        if not self.target_exe.exists():
            self.send_error(HTTPStatus.NOT_FOUND, f"目标文件不存在: {self.target_exe}")
            return

        file_size = self.target_exe.stat().st_size
        self.send_response(HTTPStatus.OK)
        self.send_header("Content-Type", "application/octet-stream")
        self.send_header("Content-Length", str(file_size))
        self.send_header("Content-Disposition", f'attachment; filename="{UPDATE_ASSET_NAME}"')
        self.end_headers()

        self.log_message("客户端开始下载更新包: %s (%.2f MB)...", UPDATE_ASSET_NAME, file_size / (1024 * 1024))
        chunk_size = 64 * 1024  # 64 KB
        sent_bytes = 0

        # 限速延迟计算（每个 chunk 的目标耗时）
        sleep_per_chunk = 0.0
        if self.speed_mb > 0:
            bytes_per_sec = self.speed_mb * 1024 * 1024
            sleep_per_chunk = chunk_size / bytes_per_sec

        with open(self.target_exe, "rb") as f:
            while True:
                chunk = f.read(chunk_size)
                if not chunk:
                    break
                try:
                    self.wfile.write(chunk)
                    sent_bytes += len(chunk)
                    if sleep_per_chunk > 0:
                        time.sleep(sleep_per_chunk)
                except (ConnectionResetError, BrokenPipeError):
                    self.log_message("客户端中断了下载连接")
                    return

        self.log_message("更新包传输完成: 共发送 %.2f MB", sent_bytes / (1024 * 1024))


def main() -> int:
    parser = argparse.ArgumentParser(description="Antigravity-Hans 自更新机制本地 Mock 仿真服务")
    parser.add_argument("--port", type=int, default=8999, help="服务监听端口（默认 8999）")
    parser.add_argument("--host", type=str, default="127.0.0.1", help="服务监听地址（默认 127.0.0.1）")
    parser.add_argument("--target-exe", type=str, required=True, help="准备推送给客户端的新版本 exe 文件路径")
    parser.add_argument("--target-tag", type=str, default="v0.4.6", help="模拟的新版本号（默认 v0.4.6）")
    parser.add_argument("--speed-mb", type=float, default=10.0, help="模拟下载限速 (MB/s)，默认 10MB/s 方便观察进度条")
    args = parser.parse_args()

    exe_path = Path(args.target_exe).resolve()
    if not exe_path.exists():
        print(f"[错误] 目标 exe 文件不存在: {exe_path}", file=sys.stderr)
        return 1

    MockUpdateHandler.target_exe = exe_path
    MockUpdateHandler.target_tag = args.target_tag
    MockUpdateHandler.speed_mb = args.speed_mb

    server = ThreadedHTTPServer((args.host, args.port), MockUpdateHandler)
    api_url = f"http://{args.host}:{args.port}/api/release"

    print("=" * 68)
    print(f" Antigravity-Hans 本地自更新仿真服务已就绪")
    print("=" * 68)
    print(f" 监听地址:      http://{args.host}:{args.port}")
    print(f" API 接口:      {api_url}")
    print(f" 模拟推送版本:  {args.target_tag}")
    print(f" 托管文件:      {exe_path} ({exe_path.stat().st_size / (1024 * 1024):.2f} MB)")
    if args.speed_mb > 0:
        print(f" 模拟限速:      {args.speed_mb:.1f} MB/s（可清晰观察客户端下载进度百分比）")
    print("-" * 68)
    print(f" 【使用说明】")
    print(f" 在 PowerShell 中执行以下命令启动待测试的旧版本客户端：")
    print(f'   $env:ANTIGRAVITY_UPDATE_API_URL = "{api_url}"')
    print(f'   & "<待测程序路径>.exe"')
    print(f" 然后在右下角系统托盘菜单中点击「检查更新」即可触发仿真更新。")
    print("=" * 68)
    print("按 Ctrl+C 可停止服务。\n")

    try:
        server.serve_forever()
    except KeyboardInterrupt:
        print("\n[MockServer] 服务已安全停止。")
    return 0


if __name__ == "__main__":
    sys.exit(main())
