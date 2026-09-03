#!/usr/bin/env bash
# 构建前端并填充 embed 目录。
# 用法：scripts/embed-frontend.sh [前端仓库路径]
#   默认前端路径 ../PTMananer（TS monorepo）
set -euo pipefail

FRONTEND_DIR="${1:-$(dirname "$0")/../../PTMananer/apps/frontend}"
TARGET="$(cd "$(dirname "$0")" && pwd)/../internal/web/frontend"

command -v pnpm >/dev/null || { echo "需要 pnpm"; exit 1; }

cd "$FRONTEND_DIR"
echo "==> 安装依赖"
pnpm install --frozen-lockfile 2>/dev/null || pnpm install

echo "==> 构建（静态导出）"
NEXT_PUBLIC_API_BASE_URL=/api pnpm build

echo "==> 拷贝到 $TARGET"
rm -rf "$TARGET"
mkdir -p "$TARGET"
cp -r out/. "$TARGET/"

echo "==> 完成：$(find "$TARGET" -type f | wc -l) 个文件"
