# PT Manager (Go)

私人 PT 账号聚合管理平台 —— [PTMananer](https://github.com/1443205008/PTMananer) 的 **Go 后端重写版**。前端（Next.js）不变，REST 接口完全兼容，可直接替换 NestJS 后端。

## 这是什么

原项目是 TypeScript monorepo（NestJS + Prisma + BullMQ 后端 / Next.js 前端）。本仓库将其后端完整转换为 Go：

| TS 版 | Go 版 |
|---|---|
| NestJS + Express | Gin |
| Prisma + migrations | pgx/v5 + 内置幂等迁移（**表结构与 Prisma 完全一致，可直连已有数据库**） |
| BullMQ (Redis) | Redis List 队列 + 指数退避重试（语义对齐） |
| @nestjs/schedule | 内置 Scheduler（读取 settings 表的间隔，运行时可改） |
| @nestjs/jwt | golang-jwt/v5 |
| class-validator | gin binding |
| axios | net/http |

## 功能（与 TS 版对齐）

- 管理员邮箱/密码登录（JWT，httpOnly cookie，scrypt 哈希——**与 TS 版密码格式互通**）
- M-Team 账户 CRUD，API Key **AES-256-GCM 加密存储**（与 TS 版密文格式互通）
- 数据同步：profile + stats + 每日快照 + 种子列表（队列异步、失败告警、指数退避）
- 定时同步（默认 30 分钟，可在设置页改）
- 仪表盘（跨账户汇总 + 趋势）
- 种子列表、站内搜索、制作组筛选、下载 Token
- 告警（已读/忽略/清理）
- 模拟保种（bencode 解析 + tracker 周期上报，重启自动恢复）
- 系统设置 + 运行状态 + 历史数据清理

## 运行

```bash
# 1. 准备环境变量
cat > .env <<'EOF'
POSTGRES_PASSWORD=change-me-pg
JWT_SECRET=            # openssl rand -hex 32
CREDENTIAL_ENCRYPTION_KEY=   # openssl rand -hex 32
SEED_ADMIN_PASSWORD=change-me-admin
EOF

# 2. 起整套（PG + Redis + 后端）
docker compose up -d --build

# 3. 验证
curl http://localhost:4000/api/v1/settings   # 401 = 正常（未登录）
```

登录：`POST /api/v1/auth/login`，邮箱 `admin@pt-manager.local`（或 `SEED_ADMIN_EMAIL`），密码 `SEED_ADMIN_PASSWORD`。

前端：`NEXT_PUBLIC_API_BASE_URL=http://localhost:4000/api` 照常指向本后端。

## 从已有 TS 版数据库迁移

无需迁移。Go 版使用与 Prisma 相同的表名/列名/枚举，启动时自动跑幂等迁移（已存在的对象跳过）。两点注意：

1. **`CREDENTIAL_ENCRYPTION_KEY` 必须与 TS 版相同**，否则已存的 API Key 解不开
2. TS 版建的 `_prisma_migrations` 与 Go 版的 `_go_migration` 互不干扰

## 本地开发

```bash
go build ./cmd/server
go test ./...
go vet ./...
```

环境变量：`DATABASE_URL`、`REDIS_URL`、`JWT_SECRET`、`CREDENTIAL_ENCRYPTION_KEY`（64 位 hex）、`MTEAM_BASE_URL`（默认 https://api.m-team.cc）等，见 `internal/config/config.go`。

## 项目结构

```
cmd/server/          入口（路由注册、依赖组装）
internal/
  config/            环境变量
  domain/            统一领域模型（站点无关）
  db/                PG 连接、幂等迁移、seed
  cryptoutil/        scrypt 密码 + AES-256-GCM 凭据
  auth/              JWT 登录 + 中间件
  providers/         TrackerProvider 接口 + 注册表
  mteam/             M-Team 实现（client/mapper/provider/types）
  accounts/          账户 CRUD
  syncer/            同步编排 + Redis 队列 + 调度器
  dashboard/ torrents/ search/ alerts/ fakeseed/ settings/
```

## 安全约束（与 TS 版一致）

- 绝不返回/记录 encryptedApiKey / iv / authTag / 明文 API Key
- 响应 DTO 物理隔离凭据字段
- 生产缺 JWT_SECRET 直接拒绝启动
