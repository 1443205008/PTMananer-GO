# 多阶段构建：Node 构建前端 → Go 编译（前端经 go:embed 进二进制）
FROM node:20-alpine AS frontend

WORKDIR /repo
# packageManager 字段锁 pnpm 版本（corepack 依此下载）
COPY package.json ./
RUN corepack enable && corepack prepare pnpm@9.15.0 --activate

# 先装依赖（利用层缓存）
COPY pnpm-workspace.yaml pnpm-lock.yaml tsconfig.base.json ./
COPY packages/shared/package.json packages/shared/
COPY apps/frontend/package.json apps/frontend/
RUN pnpm install --frozen-lockfile

# 构建（packages/shared 会被 frontend 的 tsc 一起编译）
COPY packages/shared packages/shared
COPY apps/frontend apps/frontend
RUN cd apps/frontend && NEXT_PUBLIC_API_BASE_URL=/api pnpm build

# ── Go 编译 ─────────────────────────────────────────────────
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# 从 frontend 阶段拷构建产物到 embed 目录（go:embed 会把它打进二进制）
COPY --from=frontend /repo/apps/frontend/out internal/web/frontend

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /ptmanager ./cmd/server

# ── 运行时 ──────────────────────────────────────────────────
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata mysql-client && \
    adduser -D -u 1001 ptmanager

COPY --from=builder /ptmanager /usr/local/bin/ptmanager
USER ptmanager

ENV BACKEND_PORT=4000 \
    NODE_ENV=production

EXPOSE 4000

HEALTHCHECK --interval=30s --timeout=3s --retries=3 \
    CMD wget -qO- http://127.0.0.1:4000/healthz || exit 1

ENTRYPOINT ["ptmanager"]
