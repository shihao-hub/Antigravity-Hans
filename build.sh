#!/bin/bash
# Antigravity-Hans — 一键交叉编译脚本
# 在 macOS 上运行，生成 Windows / macOS 双平台可执行文件
# 用法: ./build.sh

set -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
DIST_DIR="$SCRIPT_DIR/dist"

echo "========================================="
echo "  Antigravity-Hans Go 交叉编译脚本"
echo "========================================="

# 检查 Go 是否安装
if ! command -v go &>/dev/null; then
    echo "[ERROR] 未找到 Go 编译器。请先安装 Go: https://go.dev/dl/"
    exit 1
fi

GO_VERSION=$(go version)
echo "[提示] 使用 $GO_VERSION"

# 读取版本号
VERSION=$(cat "$SCRIPT_DIR/VERSION" | tr -d '[:space:]')
echo "[提示] 版本号: $VERSION"

# 创建输出目录
mkdir -p "$DIST_DIR"

# 进入项目根目录
cd "$SCRIPT_DIR"

echo ""
echo "开始编译..."
echo ""

# macOS ARM64 (Apple Silicon)
echo ">>> [1/3] macOS ARM64 (Apple Silicon)..."
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w -X main.Version=$VERSION" -o "$DIST_DIR/antigravity-hans-macos-arm64" .
echo "    ✅ $DIST_DIR/antigravity-hans-macos-arm64"

# macOS AMD64 (Intel)
echo ">>> [2/3] macOS AMD64 (Intel Mac)..."
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w -X main.Version=$VERSION" -o "$DIST_DIR/antigravity-hans-macos-amd64" .
echo "    ✅ $DIST_DIR/antigravity-hans-macos-amd64"

# Windows AMD64
echo ">>> [3/3] Windows AMD64..."
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="-s -w -X main.Version=$VERSION" -o "$DIST_DIR/antigravity-hans-windows-amd64.exe" .
echo "    ✅ $DIST_DIR/antigravity-hans-windows-amd64.exe"

echo ""
echo "========================================="
echo "  编译完成！输出文件："
echo "========================================="
ls -lh "$DIST_DIR"
echo ""
echo "[提示] Windows 用户直接运行 .exe 文件"
echo "[提示] macOS 用户首次运行需授权: chmod +x antigravity-hans-macos-arm64 && ./antigravity-hans-macos-arm64"
