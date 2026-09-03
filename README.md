# PT Manager (Go)

私人 PT 账号聚合管理平台 —— [PTMananer](https://github.com/1443205008/PTMananer) 的 Go 重写版。

**单二进制**：前端（Next.js 静态导出）通过 `go:embed` 打进 Go 程序，MySQL 做存储，Redis 做同步队列。一个可执行文件 + MySQL + Redis 即完整部署。

## 架构

| 原 TS 版 | Go 版 |
|---|---|
| NestJS + Express | Gin |
| Prisma (PostgreSQL) | database/sql + go-sql-driver (**MySQL 8**) |
| BullMQ (Redis) | Redis List 队列 + 指数退避重试 |
| Next.js 前端（独立进程） | **go:embed 静态导出，同源服务** |
| @nestjs/jwt / class-validator | golang-jwt/v5 / gin binding |

表结构与 Prisma 版一一对应（列名/索引/枚举值），内置幂等迁移，启动即建表。

## 快速开始

```bash
# 1. 环境变量
cat > .env <<'EOF'
MYSQL_PASSWORD=change-me
MYSQL_ROOT_PASSWORD=change-me-root
JWT_SECRET=                  # openssl rand -hex 32
CREDENTIAL_ENCRYPTION_KEY=    # openssl rand -hex 32
SEED_ADMIN_PASSWORD=change-me-admin
EOF

# 2. 一键起（MySQL + Redis + 单二进制应用）
docker compose up -d --build

# 3. 打开
# http://localhost:4000  → 登录页（admin@pt-manager.local / SEED_ADMIN_PASSWORD）
```

## 从源码构建

```bash
# 前端：静态导出 + 嵌入（需要 pnpm）
./scripts/embed-frontend.sh ../PTMananer/apps/frontend

# 后端：编译（产物已含前端）
go build -o ptmanager ./cmd/server

# 测试
go test ./...
```

前端改动后重跑 `embed-frontend.sh` 再编译即可。前端构建参数 `NEXT_PUBLIC_API_BASE_URL=/api`（同源）。

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
| `CORS_ORIGIN` | 跨源（前端已同源，基本用不到） | http://localhost:3000 |

## 功能

管理员登录（JWT httpOnly cookie）· M-Team 账户管理（API Key AES-256-GCM 加密）· 数据同步（profile/stats/每日快照/种子，队列异步+退避重试+失败告警）· 定时同步（30 分钟可调）· 仪表盘（跨账户汇总+趋势）· 种子列表 · 站内搜索/制作组/下载 Token · 告警中心 · 模拟保种（bencode 解析 + tracker 周期上报，重启恢复）· 系统设置/清理

## 项目结构

```
cmd/server/            入口
internal/
  web/                 go:embed 前端 + 静态路由（frontend/ 为构建产物）
  config/ domain/ db/ cryptoutil/ auth/
  providers/ mteam/    TrackerProvider 接口 + M-Team 实现
  accounts/ syncer/ dashboard/ torrents/ search/ alerts/ fakeseed/ settings/
scripts/embed-frontend.sh  前端构建+嵌入脚本
```

## 安全约束

- API Key 加密存储（AES-256-GCM），绝不回显/记录
- 凭据字段与响应 DTO 物理隔离
- 生产缺 JWT_SECRET 拒绝启动
