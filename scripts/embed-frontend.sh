#!/usr/bin/env bash
# 构建前端并填充 internal/web/frontend（go:embed 目录）。
# 用法：scripts/embed-frontend.sh   （在仓库根目录执行，需要 pnpm + Node 20+）
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
FRONTEND_DIR="$ROOT/apps/frontend"
TARGET="$ROOT/internal/web/frontend"

command -v pnpm >/dev/null || { echo "需要 pnpm（corepack enable）"; exit 1; }

cd "$FRONTEND_DIR"
echo "==> 安装依赖"
pnpm install --frozen-lockfile 2>/dev/null || pnpm install

echo "==> 构建前端（静态导出）"
NEXT_PUBLIC_API_BASE_URL=/api pnpm build

echo "==> 拷贝到 $TARGET"
rm -rf "$TARGET"
mkdir -p "$TARGET"
cp -r out/. "$TARGET/"

echo "==> 完成：$(find "$TARGET" -type f | wc -l) 个文件"
echo "    下一步：go build ./cmd/server"
