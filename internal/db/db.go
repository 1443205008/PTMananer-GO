// Package db: PostgreSQL 连接、迁移（幂等）与 seed。
package db

import (
	"context"
	"strings"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/1443205008/ptmanager-go/internal/config"
	"github.com/1443205008/ptmanager-go/internal/cryptoutil"
)

func Connect(databaseURL string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, err
	}
	cfg.MaxConns = 20
	cfg.MinConns = 2
	cfg.MaxConnLifetime = time.Hour
	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := pool.Ping(ctx); err != nil {
		return nil, err
	}
	log.Println("Database connected")
	return pool, nil
}

func NewRedis(redisURL string) *redis.Client {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		opt = &redis.Options{Addr: "127.0.0.1:6379"}
	}
	return redis.NewClient(opt)
}

// migrations 与 TS 版 Prisma 迁移保持同一结构（表名/列名/枚举完全一致，
// 便于直接复用已有数据库）。
var migrations = []string{
	// 20260807065801_init
	`
DO $$ BEGIN CREATE TYPE "AccountStatus" AS ENUM ('ACTIVE', 'INACTIVE', 'CREDENTIAL_INVALID', 'SYNC_ERROR'); EXCEPTION WHEN duplicate_object THEN NULL; END $$;
DO $$ BEGIN CREATE TYPE "SiteStatus" AS ENUM ('HEALTHY', 'DEGRADED', 'OFFLINE', 'UNKNOWN'); EXCEPTION WHEN duplicate_object THEN NULL; END $$;
DO $$ BEGIN CREATE TYPE "TorrentStatus" AS ENUM ('SEEDING', 'LEECHING', 'COMPLETED', 'STOPPED', 'UNKNOWN'); EXCEPTION WHEN duplicate_object THEN NULL; END $$;
DO $$ BEGIN CREATE TYPE "HnrStatus" AS ENUM ('ACTIVE', 'RESOLVED', 'EXPIRED'); EXCEPTION WHEN duplicate_object THEN NULL; END $$;
DO $$ BEGIN CREATE TYPE "SyncJobType" AS ENUM ('PROFILE_SYNC', 'STATS_SYNC', 'TORRENT_SYNC', 'HNR_SYNC', 'BONUS_SYNC', 'SITE_HEALTH_CHECK', 'DAILY_SNAPSHOT'); EXCEPTION WHEN duplicate_object THEN NULL; END $$;
DO $$ BEGIN CREATE TYPE "SyncStatus" AS ENUM ('PENDING', 'RUNNING', 'SUCCESS', 'FAILED', 'PARTIAL'); EXCEPTION WHEN duplicate_object THEN NULL; END $$;
DO $$ BEGIN CREATE TYPE "LogLevel" AS ENUM ('DEBUG', 'INFO', 'WARN', 'ERROR'); EXCEPTION WHEN duplicate_object THEN NULL; END $$;
DO $$ BEGIN CREATE TYPE "AlertType" AS ENUM ('HNR_DETECTED', 'AUTH_FAILED', 'SYNC_FAILED', 'SITE_OFFLINE', 'API_CHANGED', 'RATE_LIMITED'); EXCEPTION WHEN duplicate_object THEN NULL; END $$;
DO $$ BEGIN CREATE TYPE "AlertSeverity" AS ENUM ('INFO', 'WARNING', 'ERROR', 'CRITICAL'); EXCEPTION WHEN duplicate_object THEN NULL; END $$;

CREATE TABLE IF NOT EXISTS "User" (
    "id" TEXT NOT NULL,
    "email" TEXT NOT NULL,
    "name" TEXT,
    "password" TEXT NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL,
    CONSTRAINT "User_pkey" PRIMARY KEY ("id")
);
CREATE UNIQUE INDEX IF NOT EXISTS "User_email_key" ON "User"("email");

CREATE TABLE IF NOT EXISTS "TrackerSite" (
    "id" TEXT NOT NULL,
    "code" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "apiBaseUrl" TEXT NOT NULL,
    "isEnabled" BOOLEAN NOT NULL DEFAULT true,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL,
    CONSTRAINT "TrackerSite_pkey" PRIMARY KEY ("id")
);
CREATE UNIQUE INDEX IF NOT EXISTS "TrackerSite_code_key" ON "TrackerSite"("code");

CREATE TABLE IF NOT EXISTS "TrackerAccount" (
    "id" TEXT NOT NULL,
    "userId" TEXT NOT NULL,
    "siteId" TEXT NOT NULL,
    "accountName" TEXT NOT NULL,
    "remark" TEXT,
    "externalUserId" TEXT,
    "username" TEXT,
    "avatarUrl" TEXT,
    "joinedAt" TIMESTAMP(3),
    "isEnabled" BOOLEAN NOT NULL DEFAULT true,
    "syncEnabled" BOOLEAN NOT NULL DEFAULT true,
    "status" "AccountStatus" NOT NULL DEFAULT 'ACTIVE',
    "lastSyncAt" TIMESTAMP(3),
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL,
    CONSTRAINT "TrackerAccount_pkey" PRIMARY KEY ("id")
);
CREATE INDEX IF NOT EXISTS "TrackerAccount_userId_idx" ON "TrackerAccount"("userId");
CREATE INDEX IF NOT EXISTS "TrackerAccount_siteId_idx" ON "TrackerAccount"("siteId");

CREATE TABLE IF NOT EXISTS "TrackerCredential" (
    "id" TEXT NOT NULL,
    "accountId" TEXT NOT NULL,
    "encryptedApiKey" TEXT NOT NULL,
    "iv" TEXT NOT NULL,
    "authTag" TEXT NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL,
    CONSTRAINT "TrackerCredential_pkey" PRIMARY KEY ("id")
);
CREATE UNIQUE INDEX IF NOT EXISTS "TrackerCredential_accountId_key" ON "TrackerCredential"("accountId");

CREATE TABLE IF NOT EXISTS "TrackerStats" (
    "id" TEXT NOT NULL,
    "accountId" TEXT NOT NULL,
    "uploadBytes" BIGINT NOT NULL DEFAULT 0,
    "downloadBytes" BIGINT NOT NULL DEFAULT 0,
    "ratio" DECIMAL(10,4) NOT NULL DEFAULT 0,
    "bonus" DECIMAL(15,2) NOT NULL DEFAULT 0,
    "seedingCount" INTEGER NOT NULL DEFAULT 0,
    "seedingBytes" BIGINT NOT NULL DEFAULT 0,
    "leechingCount" INTEGER NOT NULL DEFAULT 0,
    "hitAndRunCount" INTEGER NOT NULL DEFAULT 0,
    "roleId" INTEGER,
    "levelName" TEXT,
    "isWarned" BOOLEAN NOT NULL DEFAULT false,
    "isLeechWarn" BOOLEAN NOT NULL DEFAULT false,
    "isVip" BOOLEAN NOT NULL DEFAULT false,
    "isDonor" BOOLEAN NOT NULL DEFAULT false,
    "lastLoginAt" TIMESTAMP(3),
    "lastTrackerAt" TIMESTAMP(3),
    "siteStatus" "SiteStatus" NOT NULL DEFAULT 'UNKNOWN',
    "syncedAt" TIMESTAMP(3),
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL,
    CONSTRAINT "TrackerStats_pkey" PRIMARY KEY ("id")
);
CREATE UNIQUE INDEX IF NOT EXISTS "TrackerStats_accountId_key" ON "TrackerStats"("accountId");

CREATE TABLE IF NOT EXISTS "TrackerDailySnapshot" (
    "id" TEXT NOT NULL,
    "accountId" TEXT NOT NULL,
    "snapshotDate" TIMESTAMP(3) NOT NULL,
    "uploadBytes" BIGINT NOT NULL DEFAULT 0,
    "downloadBytes" BIGINT NOT NULL DEFAULT 0,
    "ratio" DECIMAL(10,4) NOT NULL DEFAULT 0,
    "bonus" DECIMAL(15,2) NOT NULL DEFAULT 0,
    "seedingCount" INTEGER NOT NULL DEFAULT 0,
    "seedingBytes" BIGINT NOT NULL DEFAULT 0,
    "leechingCount" INTEGER NOT NULL DEFAULT 0,
    "hitAndRunCount" INTEGER NOT NULL DEFAULT 0,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT "TrackerDailySnapshot_pkey" PRIMARY KEY ("id")
);
CREATE UNIQUE INDEX IF NOT EXISTS "TrackerDailySnapshot_accountId_snapshotDate_key" ON "TrackerDailySnapshot"("accountId", "snapshotDate");
CREATE INDEX IF NOT EXISTS "TrackerDailySnapshot_accountId_snapshotDate_idx" ON "TrackerDailySnapshot"("accountId", "snapshotDate");

CREATE TABLE IF NOT EXISTS "TrackerTorrent" (
    "id" TEXT NOT NULL,
    "accountId" TEXT NOT NULL,
    "siteTorrentId" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "sizeBytes" BIGINT NOT NULL DEFAULT 0,
    "status" "TorrentStatus" NOT NULL DEFAULT 'UNKNOWN',
    "uploadedBytes" BIGINT NOT NULL DEFAULT 0,
    "downloadedBytes" BIGINT NOT NULL DEFAULT 0,
    "ratio" DECIMAL(10,4) NOT NULL DEFAULT 0,
    "seedTimeSecs" INTEGER NOT NULL DEFAULT 0,
    "leechTimeSecs" INTEGER NOT NULL DEFAULT 0,
    "completedAt" TIMESTAMP(3),
    "lastActivityAt" TIMESTAMP(3),
    "syncedAt" TIMESTAMP(3),
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL,
    CONSTRAINT "TrackerTorrent_pkey" PRIMARY KEY ("id")
);
CREATE UNIQUE INDEX IF NOT EXISTS "TrackerTorrent_accountId_siteTorrentId_key" ON "TrackerTorrent"("accountId", "siteTorrentId");
CREATE INDEX IF NOT EXISTS "TrackerTorrent_accountId_status_idx" ON "TrackerTorrent"("accountId", "status");
CREATE INDEX IF NOT EXISTS "TrackerTorrent_accountId_lastActivityAt_idx" ON "TrackerTorrent"("accountId", "lastActivityAt");

CREATE TABLE IF NOT EXISTS "HitAndRun" (
    "id" TEXT NOT NULL,
    "accountId" TEXT NOT NULL,
    "torrentId" TEXT,
    "siteTorrentId" TEXT,
    "torrentName" TEXT,
    "detectedAt" TIMESTAMP(3) NOT NULL,
    "status" "HnrStatus" NOT NULL DEFAULT 'ACTIVE',
    "resolvedAt" TIMESTAMP(3),
    "rawData" JSONB,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL,
    CONSTRAINT "HitAndRun_pkey" PRIMARY KEY ("id")
);
CREATE INDEX IF NOT EXISTS "HitAndRun_accountId_status_idx" ON "HitAndRun"("accountId", "status");

CREATE TABLE IF NOT EXISTS "SyncJob" (
    "id" TEXT NOT NULL,
    "accountId" TEXT NOT NULL,
    "type" "SyncJobType" NOT NULL,
    "status" "SyncStatus" NOT NULL DEFAULT 'PENDING',
    "startedAt" TIMESTAMP(3),
    "finishedAt" TIMESTAMP(3),
    "durationMs" INTEGER,
    "records" INTEGER,
    "errorCode" TEXT,
    "errorMessage" TEXT,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL,
    CONSTRAINT "SyncJob_pkey" PRIMARY KEY ("id")
);
CREATE INDEX IF NOT EXISTS "SyncJob_accountId_type_createdAt_idx" ON "SyncJob"("accountId", "type", "createdAt" DESC);
CREATE INDEX IF NOT EXISTS "SyncJob_status_createdAt_idx" ON "SyncJob"("status", "createdAt" DESC);

CREATE TABLE IF NOT EXISTS "SyncLog" (
    "id" TEXT NOT NULL,
    "jobId" TEXT NOT NULL,
    "level" "LogLevel" NOT NULL DEFAULT 'INFO',
    "message" TEXT NOT NULL,
    "metadata" JSONB,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT "SyncLog_pkey" PRIMARY KEY ("id")
);
CREATE INDEX IF NOT EXISTS "SyncLog_jobId_createdAt_idx" ON "SyncLog"("jobId", "createdAt");

CREATE TABLE IF NOT EXISTS "Alert" (
    "id" TEXT NOT NULL,
    "accountId" TEXT,
    "type" "AlertType" NOT NULL,
    "title" TEXT NOT NULL,
    "message" TEXT NOT NULL,
    "severity" "AlertSeverity" NOT NULL DEFAULT 'INFO',
    "isRead" BOOLEAN NOT NULL DEFAULT false,
    "isDismissed" BOOLEAN NOT NULL DEFAULT false,
    "metadata" JSONB,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL,
    CONSTRAINT "Alert_pkey" PRIMARY KEY ("id")
);
CREATE INDEX IF NOT EXISTS "Alert_isRead_severity_createdAt_idx" ON "Alert"("isRead", "severity", "createdAt" DESC);

CREATE TABLE IF NOT EXISTS "SystemSetting" (
    "id" TEXT NOT NULL,
    "userId" TEXT,
    "key" TEXT NOT NULL,
    "value" TEXT NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL,
    CONSTRAINT "SystemSetting_pkey" PRIMARY KEY ("id")
);
CREATE UNIQUE INDEX IF NOT EXISTS "SystemSetting_userId_key_key" ON "SystemSetting"("userId", "key");

ALTER TABLE "TrackerAccount" ADD CONSTRAINT "TrackerAccount_userId_fkey" FOREIGN KEY ("userId") REFERENCES "User"("id") ON DELETE RESTRICT ON UPDATE CASCADE;
ALTER TABLE "TrackerAccount" ADD CONSTRAINT "TrackerAccount_siteId_fkey" FOREIGN KEY ("siteId") REFERENCES "TrackerSite"("id") ON DELETE RESTRICT ON UPDATE CASCADE;
ALTER TABLE "TrackerCredential" ADD CONSTRAINT "TrackerCredential_accountId_fkey" FOREIGN KEY ("accountId") REFERENCES "TrackerAccount"("id") ON DELETE CASCADE ON UPDATE CASCADE;
ALTER TABLE "TrackerStats" ADD CONSTRAINT "TrackerStats_accountId_fkey" FOREIGN KEY ("accountId") REFERENCES "TrackerAccount"("id") ON DELETE CASCADE ON UPDATE CASCADE;
ALTER TABLE "TrackerDailySnapshot" ADD CONSTRAINT "TrackerDailySnapshot_accountId_fkey" FOREIGN KEY ("accountId") REFERENCES "TrackerAccount"("id") ON DELETE CASCADE ON UPDATE CASCADE;
ALTER TABLE "TrackerTorrent" ADD CONSTRAINT "TrackerTorrent_accountId_fkey" FOREIGN KEY ("accountId") REFERENCES "TrackerAccount"("id") ON DELETE CASCADE ON UPDATE CASCADE;
ALTER TABLE "HitAndRun" ADD CONSTRAINT "HitAndRun_torrentId_fkey" FOREIGN KEY ("torrentId") REFERENCES "TrackerTorrent"("id") ON DELETE SET NULL ON UPDATE CASCADE;
ALTER TABLE "SyncJob" ADD CONSTRAINT "SyncJob_accountId_fkey" FOREIGN KEY ("accountId") REFERENCES "TrackerAccount"("id") ON DELETE CASCADE ON UPDATE CASCADE;
ALTER TABLE "SyncLog" ADD CONSTRAINT "SyncLog_jobId_fkey" FOREIGN KEY ("jobId") REFERENCES "SyncJob"("id") ON DELETE CASCADE ON UPDATE CASCADE;
ALTER TABLE "Alert" ADD CONSTRAINT "Alert_accountId_fkey" FOREIGN KEY ("accountId") REFERENCES "TrackerAccount"("id") ON DELETE SET NULL ON UPDATE CASCADE;
ALTER TABLE "SystemSetting" ADD CONSTRAINT "SystemSetting_userId_fkey" FOREIGN KEY ("userId") REFERENCES "User"("id") ON DELETE CASCADE ON UPDATE CASCADE;
`,
	// 20260808055557_add_bonus_hourly_rate
	`ALTER TABLE "TrackerStats" ADD COLUMN IF NOT EXISTS "bonusHourlyRate" DECIMAL(10,4);`,
	// 20260808115854_add_joined_at
	`ALTER TABLE "TrackerAccount" ADD COLUMN IF NOT EXISTS "joinedAt" TIMESTAMP(3);`,
	// 20260808131359_add_fake_seed_job
	`
DO $$ BEGIN CREATE TYPE "FakeSeedStatus" AS ENUM ('RUNNING', 'STOPPED', 'ERROR'); EXCEPTION WHEN duplicate_object THEN NULL; END $$;
CREATE TABLE IF NOT EXISTS "FakeSeedJob" (
    "id" TEXT NOT NULL,
    "accountId" TEXT NOT NULL,
    "torrentId" TEXT NOT NULL,
    "torrentName" TEXT NOT NULL,
    "infoHash" TEXT NOT NULL,
    "trackerUrl" TEXT NOT NULL,
    "totalSize" BIGINT NOT NULL DEFAULT 0,
    "peerId" TEXT NOT NULL,
    "peerKey" TEXT NOT NULL,
    "port" INTEGER NOT NULL DEFAULT 34567,
    "status" "FakeSeedStatus" NOT NULL DEFAULT 'STOPPED',
    "interval" INTEGER NOT NULL DEFAULT 300,
    "lastReportAt" TIMESTAMP(3),
    "nextReportAt" TIMESTAMP(3),
    "errorMessage" TEXT,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL,
    CONSTRAINT "FakeSeedJob_pkey" PRIMARY KEY ("id")
);
CREATE UNIQUE INDEX IF NOT EXISTS "FakeSeedJob_infoHash_key" ON "FakeSeedJob"("infoHash");
ALTER TABLE "FakeSeedJob" ADD CONSTRAINT "FakeSeedJob_accountId_fkey" FOREIGN KEY ("accountId") REFERENCES "TrackerAccount"("id") ON DELETE CASCADE ON UPDATE CASCADE;
`,
}

// Migrate 逐条执行幂等 DDL。对已存在数据库（Prisma 建的）零影响：
// CREATE ... IF NOT EXISTS 全部跳过；ALTER ... ADD COLUMN IF NOT EXISTS 跳过；
// ADD CONSTRAINT 若已存在会报错，捕获并忽略 "already exists"。
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	// 先建迁移记录表
	if _, err := pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS "_go_migration" (
		"id" SERIAL PRIMARY KEY,
		"name" TEXT UNIQUE NOT NULL,
		"appliedAt" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`); err != nil {
		return fmt.Errorf("create migration table: %w", err)
	}

	names := []string{
		"20260807065801_init",
		"20260808055557_add_bonus_hourly_rate",
		"20260808115854_add_joined_at",
		"20260808131359_add_fake_seed_job",
	}

	for i, sqlText := range migrations {
		name := names[i]
		var exists bool
		if err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM "_go_migration" WHERE name=$1)`, name).Scan(&exists); err != nil {
			return err
		}
		if exists {
			continue
		}
		// 逐条执行（DO 块内含分号，不能简单 split；跳过 $$..$$ 内部的分号）
		stmts := splitSQL(sqlText)
		for _, stmt := range stmts {
			stmt = strings.TrimSpace(stmt)
			if stmt == "" {
				continue
			}
			if _, err := pool.Exec(ctx, stmt); err != nil {
				msg := err.Error()
				// 幂等：已存在（复用 Prisma 建的库 / 索引重复建）时跳过
				if strings.Contains(msg, "already exists") {
					continue
				}
				return fmt.Errorf("migration %s: %w", name, err)
			}
		}
		if _, err := pool.Exec(ctx, `INSERT INTO "_go_migration"(name) VALUES($1) ON CONFLICT DO NOTHING`, name); err != nil {
			return err
		}
		log.Printf("applied migration %s", name)
	}
	return nil
}

// splitSQL 按 ";" 切分，但跳过 $$...$$ 内部的分号（DO 块）
func splitSQL(sqlText string) []string {
	var out []string
	var cur strings.Builder
	inDollar := false
	for i := 0; i < len(sqlText); i++ {
		c := sqlText[i]
		if !inDollar && c == '$' && i+1 < len(sqlText) && sqlText[i+1] == '$' {
			inDollar = true
			cur.WriteString("$$")
			i++
			continue
		}
		if inDollar && c == '$' && i+1 < len(sqlText) && sqlText[i+1] == '$' {
			inDollar = false
			cur.WriteString("$$")
			i++
			continue
		}
		if !inDollar && c == ';' {
			out = append(out, cur.String())
			cur.Reset()
			continue
		}
		cur.WriteByte(c)
	}
	if cur.Len() > 0 {
		out = append(out, cur.String())
	}
	return out
}

// NewID 生成 25 位 cuid 风格 ID（时间戳+随机），与 Prisma 默认长度近似。
func NewID() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return fmt.Sprintf("c%s%s", fmt.Sprintf("%08x", time.Now().UnixMilli()), hex.EncodeToString(b))[:25]
}

// Seed 初始化默认管理员与 M-Team 站点（幂等，对齐 prisma/seed.ts）。
func Seed(ctx context.Context, pool *pgxpool.Pool, cfg *config.Config) error {
	defaultEmail := os.Getenv("SEED_ADMIN_EMAIL")
	if defaultEmail == "" {
		defaultEmail = "admin@pt-manager.local"
	}
	defaultPassword := os.Getenv("SEED_ADMIN_PASSWORD")
	if defaultPassword == "" {
		return fmt.Errorf("SEED_ADMIN_PASSWORD is required to initialize the admin user")
	}

	now := time.Now()
	hashed, err := cryptoutil.HashPassword(defaultPassword)
	if err != nil {
		return err
	}

	var userID string
	err = pool.QueryRow(ctx, `SELECT "id" FROM "User" WHERE "email"=$1`, defaultEmail).Scan(&userID)
	switch {
	case err == nil:
		// 已存在：若密码还是占位符则更新
		var pwd string
		if err := pool.QueryRow(ctx, `SELECT "password" FROM "User" WHERE "id"=$1`, userID).Scan(&pwd); err != nil {
			return err
		}
		if pwd == "PENDING_AUTH_PHASE_5" {
			_, err = pool.Exec(ctx, `UPDATE "User" SET "password"=$1, "updatedAt"=$2 WHERE "id"=$3`, hashed, now, userID)
			if err != nil {
				return err
			}
		}
	case err == pgx.ErrNoRows:
		userID = NewID()
		_, err = pool.Exec(ctx,
			`INSERT INTO "User"("id","email","name","password","createdAt","updatedAt") VALUES($1,$2,$3,$4,$5,$5)`,
			userID, defaultEmail, "Administrator", hashed, now)
		if err != nil {
			return err
		}
	default:
		return err
	}
	log.Printf("✓ Default user ready: %s (%s)", defaultEmail, userID)

	mteamBase := cfg.MTeamBaseURL
	_, err = pool.Exec(ctx, `
		INSERT INTO "TrackerSite"("id","code","name","apiBaseUrl","isEnabled","createdAt","updatedAt")
		VALUES($1,'MTEAM','M-Team',$2,true,$3,$3)
		ON CONFLICT ("code") DO UPDATE SET "apiBaseUrl"=$2, "updatedAt"=$3`,
		NewID(), mteamBase, now)
	if err != nil {
		return err
	}
	log.Printf("✓ Tracker site ready: MTEAM → %s", mteamBase)
	log.Println("Seed completed. You can now create tracker accounts via POST /api/v1/accounts")
	return nil
}
