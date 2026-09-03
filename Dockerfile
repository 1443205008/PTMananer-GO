# 多阶段构建：编译 + 运行时镜像
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /ptmanager ./cmd/server

# ── 运行时 ─────────────────────────────────────────────────────────────
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata && \
    adduser -D -u 1001 ptmanager

COPY --from=builder /ptmanager /usr/local/bin/ptmanager
USER ptmanager

ENV BACKEND_PORT=4000 \
    NODE_ENV=production

EXPOSE 4000

# 启动前先跑迁移和 seed（RUN_SEED=true 需要 SEED_ADMIN_PASSWORD）
ENTRYPOINT ["ptmanager"]
