# PT Manager (Go)

私人 PT 账号聚合管理平台 —— [PTMananer](https://github.com/1443205008/PTMananer) 的 Go 重写版。

**单二进制**：前端（Next.js 静态导出）经 `go:embed` 打进 Go 程序，MySQL 存储，Redis 同步队列。一个可执行文件 + MySQL + Redis 即完整部署。

## 架构

| 原 TS 版 | Go 版 |
|---|---|
| NestJS + Express | Gin |
| Prisma (PostgreSQL) | database/sql + go-sql-driver (**MySQL 8**) |
| BullMQ (Redis) | Redis List 队列 + 指数退避重试 |
| Next.js 前端（独立进程） | **go:embed 静态导出，同源服务** |
| @nestjs/jwt / class-validator | golang-jwt/v5 / gin binding |

表结构与 Prisma 版一一对应（列名/索引/枚举值），内置幂等迁移，启动即建表。

## 部署

**方式一：拉镜像（推荐，零构建）**——main 分支每次推送，GitHub Actions 自动构建并发布到 GHCR：

```bash
docker run -d --name ptmanager \
  -e DATABASE_URL="mysql://user:pass@host:3306/ptmanager" \
  -e REDIS_URL="redis://host:6379" \
  -e JWT_SECRET=... -e CREDENTIAL_ENCRYPTION_KEY=... \
  -e RUN_SEED=true -e SEED_ADMIN_PASSWORD=... \
  -p 4000:4000 ghcr.io/1443205008/ptmananer-go:latest
```

（首次使用需在 GitHub 仓库 Packages 设置里把 package 设为 public，或先 docker login ghcr.io）

**方式二：本地构建**（改了代码想立即验证）：

```bash
docker compose up -d --build
```

依赖安装和 Go 模块下载走 BuildKit 缓存挂载，二次构建只有编译耗时。

## 快速开始（Docker）

```bash
cat > .env <<'EOF'
MYSQL_PASSWORD=change-me
MYSQL_ROOT_PASSWORD=change-me-root
JWT_SECRET=                  # openssl rand -hex 32
CREDENTIAL_ENCRYPTION_KEY=    # openssl rand -hex 32
SEED_ADMIN_PASSWORD=change-me-admin
EOF

docker compose up -d --build

# 打开 http://localhost:4000
# 登录：admin@pt-manager.local / SEED_ADMIN_PASSWORD
```

Docker 多阶段构建：Node 阶段编译前端 → Go 阶段 embed 进二进制。无需本地 Node 环境。

## 仓库结构（monorepo）

```
cmd/server/            Go 入口
internal/              Go 后端（auth/accounts/syncer/mteam/fakeseed/...）
apps/frontend/         前端源码（Next.js 15，output: 'export'）
packages/shared/       前后端共享类型/枚举
scripts/embed-frontend.sh   前端构建 + 填充 embed 目录
```

## 从源码构建

```bash
# 方式一：两步（本地有 Node + Go）
./scripts/embed-frontend.sh        # 构建前端 → internal/web/frontend/
go build -o ptmanager ./cmd/server

# 方式二：一条 Docker 命令（推荐，无需本地环境）
docker compose up -d --build

# 测试
go test ./...
```

改前端：编辑 `apps/frontend/src/` → 重跑上面任一构建。

## 环境变量

| 变量 | 说明 | 默认 |
|---|---|---|
| `DATABASE_URL` | `mysql://user:pass@host:3306/dbname` | 必填 |
| `REDIS_URL` | `redis://host:6379` | localhost |
| `JWT_SECRET` | 生产必填（缺省启动失败） | - |
| `CREDENTIAL_ENCRYPTION_KEY` | API Key 加密 key（64 hex） | 必填 |
| `BACKEND_PORT` | 监听端口 | 4000 |
| `RUN_SEED` / `SEED_ADMIN_EMAIL` / `SEED_ADMIN_PASSWORD` | 初始管理员 | false |
| `MTEAM_BASE_URL` | M-Team API | https://api.m-team.cc |

## 功能

管理员登录（JWT httpOnly cookie，5次/分/IP 登录限流）· M-Team 账户管理（API Key AES-256-GCM 加密）· 数据同步（profile/stats/每日快照/种子，队列异步+退避重试+失败告警）· 定时同步（启动即首跑，间隔可调）· 仪表盘 · 种子列表 · 站内搜索/制作组/下载 Token · 告警中心 · 模拟保种（bencode 解析 + tracker 上报，重启恢复）· 系统设置/每日自动清理 · gzip 压缩 · /healthz 健康检查

## 安全约束

- API Key AES-256-GCM 加密存储，绝不回显/记录
- scrypt 密码哈希（与 TS 版互通）
- JWT 锁定 HS256
- 生产缺 JWT_SECRET 拒绝启动
