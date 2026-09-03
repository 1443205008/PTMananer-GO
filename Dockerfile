# 阶段 0：前端（可选——本仓库已提交构建产物 internal/web/frontend/，无前端源码时可跳过）
# 如需重建前端：把 TS 仓库放到 ../PTMananer，取消下面注释
# FROM node:20-alpine AS frontend
# RUN corepack enable
# COPY PTMananer/apps/frontend /app
# WORKDIR /app
# RUN pnpm install && NEXT_PUBLIC_API_BASE_URL=/api pnpm build

# ── 阶段 1：Go 编译 ─────────────────────────────────────────
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /ptmanager ./cmd/server

# ── 阶段 2：运行时 ──────────────────────────────────────────
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata && \
    adduser -D -u 1001 ptmanager

COPY --from=builder /ptmanager /usr/local/bin/ptmanager
USER ptmanager

ENV BACKEND_PORT=4000 \
    NODE_ENV=production

EXPOSE 4000

ENTRYPOINT ["ptmanager"]
