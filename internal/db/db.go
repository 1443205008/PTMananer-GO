// Package db: MySQL 连接、迁移（幂等）与 seed。
// 表结构与 Prisma(PostgreSQL) 版对齐：列名/枚举值/索引一一对应。
package db

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/1443205008/ptmanager-go/internal/config"
	"github.com/1443205008/ptmanager-go/internal/cryptoutil"

	_ "github.com/go-sql-driver/mysql"
	"github.com/redis/go-redis/v9"
)

// Connect 解析 MySQL DSN（支持 mysql://user:pass@host:port/dbname 或 go-sql-driver 原生格式）
func Connect(databaseURL string) (*sql.DB, error) {
	dsn := databaseURL
	if strings.HasPrefix(dsn, "mysql://") {
		dsn = strings.TrimPrefix(dsn, "mysql://")
		atIdx := strings.LastIndex(dsn, "@")
		if atIdx < 0 {
			return nil, fmt.Errorf("invalid DATABASE_URL")
		}
		userPart := dsn[:atIdx]
		hostPart := dsn[atIdx+1:]
		slashIdx := strings.Index(hostPart, "/")
		if slashIdx < 0 {
			return nil, fmt.Errorf("invalid DATABASE_URL: missing dbname")
		}
		host := hostPart[:slashIdx]
		rest := hostPart[slashIdx+1:]
		params := "parseTime=true&loc=UTC"
		if q := strings.Index(rest, "?"); q >= 0 {
			dbName := rest[:q]
			extra := rest[q+1:]
			if extra != "" {
				params += "&" + extra
			}
			rest = dbName
		}
		if strings.Contains(host, ":") {
			host = "tcp(" + host + ")"
		}
		dsn = fmt.Sprintf("%s@%s/%s?%s", userPart, host, rest, params)
	} else if !strings.Contains(dsn, "parseTime") {
		dsn += "?parseTime=true&loc=UTC"
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return nil, err
	}
	log.Println("Database connected (MySQL)")
	return db, nil
}

func NewRedis(redisURL string) *redis.Client {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		opt = &redis.Options{Addr: "127.0.0.1:6379"}
	}
	return redis.NewClient(opt)
}

// migrations：列名/索引与 Prisma 版一致（枚举用 VARCHAR，值域由应用层保证）。
// SQL 用普通双引号字符串（MySQL 反引号与 Go raw string 冲突）。
var migrations = []string{
	// 20260807065801_init
	"CREATE TABLE IF NOT EXISTS `User` (" +
		"`id` VARCHAR(64) NOT NULL PRIMARY KEY," +
		"`email` VARCHAR(255) NOT NULL," +
		"`name` VARCHAR(255)," +
		"`password` VARCHAR(255) NOT NULL," +
		"`createdAt` TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)," +
		"`updatedAt` TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3)," +
		"UNIQUE KEY `User_email_key` (`email`)" +
		") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;" +
		"CREATE TABLE IF NOT EXISTS `TrackerSite` (" +
		"`id` VARCHAR(64) NOT NULL PRIMARY KEY," +
		"`code` VARCHAR(32) NOT NULL," +
		"`name` VARCHAR(255) NOT NULL," +
		"`apiBaseUrl` VARCHAR(512) NOT NULL," +
		"`isEnabled` BOOLEAN NOT NULL DEFAULT TRUE," +
		"`createdAt` TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)," +
		"`updatedAt` TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3)," +
		"UNIQUE KEY `TrackerSite_code_key` (`code`)" +
		") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;" +
		"CREATE TABLE IF NOT EXISTS `TrackerAccount` (" +
		"`id` VARCHAR(64) NOT NULL PRIMARY KEY," +
		"`userId` VARCHAR(64) NOT NULL," +
		"`siteId` VARCHAR(64) NOT NULL," +
		"`accountName` VARCHAR(64) NOT NULL," +
		"`remark` VARCHAR(256)," +
		"`externalUserId` VARCHAR(64)," +
		"`username` VARCHAR(255)," +
		"`avatarUrl` VARCHAR(1024)," +
		"`joinedAt` TIMESTAMP(3) NULL," +
		"`isEnabled` BOOLEAN NOT NULL DEFAULT TRUE," +
		"`syncEnabled` BOOLEAN NOT NULL DEFAULT TRUE," +
		"`status` VARCHAR(32) NOT NULL DEFAULT 'ACTIVE'," +
		"`lastSyncAt` TIMESTAMP(3) NULL," +
		"`createdAt` TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)," +
		"`updatedAt` TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3)," +
		"INDEX `TrackerAccount_userId_idx` (`userId`)," +
		"INDEX `TrackerAccount_siteId_idx` (`siteId`)," +
		"CONSTRAINT `TrackerAccount_userId_fkey` FOREIGN KEY (`userId`) REFERENCES `User`(`id`)," +
		"CONSTRAINT `TrackerAccount_siteId_fkey` FOREIGN KEY (`siteId`) REFERENCES `TrackerSite`(`id`)" +
		") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;" +
		"CREATE TABLE IF NOT EXISTS `TrackerCredential` (" +
		"`id` VARCHAR(64) NOT NULL PRIMARY KEY," +
		"`accountId` VARCHAR(64) NOT NULL," +
		"`encryptedApiKey` TEXT NOT NULL," +
		"`iv` VARCHAR(64) NOT NULL," +
		"`authTag` VARCHAR(64) NOT NULL," +
		"`createdAt` TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)," +
		"`updatedAt` TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3)," +
		"UNIQUE KEY `TrackerCredential_accountId_key` (`accountId`)," +
		"CONSTRAINT `TrackerCredential_accountId_fkey` FOREIGN KEY (`accountId`) REFERENCES `TrackerAccount`(`id`) ON DELETE CASCADE" +
		") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;" +
		"CREATE TABLE IF NOT EXISTS `TrackerStats` (" +
		"`id` VARCHAR(64) NOT NULL PRIMARY KEY," +
		"`accountId` VARCHAR(64) NOT NULL," +
		"`uploadBytes` BIGINT NOT NULL DEFAULT 0," +
		"`downloadBytes` BIGINT NOT NULL DEFAULT 0," +
		"`ratio` DECIMAL(10,4) NOT NULL DEFAULT 0," +
		"`bonus` DECIMAL(15,2) NOT NULL DEFAULT 0," +
		"`seedingCount` INT NOT NULL DEFAULT 0," +
		"`seedingBytes` BIGINT NOT NULL DEFAULT 0," +
		"`leechingCount` INT NOT NULL DEFAULT 0," +
		"`hitAndRunCount` INT NOT NULL DEFAULT 0," +
		"`roleId` INT," +
		"`levelName` VARCHAR(64)," +
		"`isWarned` BOOLEAN NOT NULL DEFAULT FALSE," +
		"`isLeechWarn` BOOLEAN NOT NULL DEFAULT FALSE," +
		"`isVip` BOOLEAN NOT NULL DEFAULT FALSE," +
		"`isDonor` BOOLEAN NOT NULL DEFAULT FALSE," +
		"`lastLoginAt` TIMESTAMP(3) NULL," +
		"`lastTrackerAt` TIMESTAMP(3) NULL," +
		"`siteStatus` VARCHAR(16) NOT NULL DEFAULT 'UNKNOWN'," +
		"`syncedAt` TIMESTAMP(3) NULL," +
		"`bonusHourlyRate` DECIMAL(10,4) NULL," +
		"`createdAt` TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)," +
		"`updatedAt` TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3)," +
		"UNIQUE KEY `TrackerStats_accountId_key` (`accountId`)," +
		"CONSTRAINT `TrackerStats_accountId_fkey` FOREIGN KEY (`accountId`) REFERENCES `TrackerAccount`(`id`) ON DELETE CASCADE" +
		") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;" +
		"CREATE TABLE IF NOT EXISTS `TrackerDailySnapshot` (" +
		"`id` VARCHAR(64) NOT NULL PRIMARY KEY," +
		"`accountId` VARCHAR(64) NOT NULL," +
		"`snapshotDate` DATE NOT NULL," +
		"`uploadBytes` BIGINT NOT NULL DEFAULT 0," +
		"`downloadBytes` BIGINT NOT NULL DEFAULT 0," +
		"`ratio` DECIMAL(10,4) NOT NULL DEFAULT 0," +
		"`bonus` DECIMAL(15,2) NOT NULL DEFAULT 0," +
		"`seedingCount` INT NOT NULL DEFAULT 0," +
		"`seedingBytes` BIGINT NOT NULL DEFAULT 0," +
		"`leechingCount` INT NOT NULL DEFAULT 0," +
		"`hitAndRunCount` INT NOT NULL DEFAULT 0," +
		"`createdAt` TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)," +
		"UNIQUE KEY `TrackerDailySnapshot_accountId_snapshotDate_key` (`accountId`, `snapshotDate`)," +
		"CONSTRAINT `TrackerDailySnapshot_accountId_fkey` FOREIGN KEY (`accountId`) REFERENCES `TrackerAccount`(`id`) ON DELETE CASCADE" +
		") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;" +
		"CREATE TABLE IF NOT EXISTS `TrackerTorrent` (" +
		"`id` VARCHAR(64) NOT NULL PRIMARY KEY," +
		"`accountId` VARCHAR(64) NOT NULL," +
		"`siteTorrentId` VARCHAR(64) NOT NULL," +
		"`name` VARCHAR(512) NOT NULL," +
		"`sizeBytes` BIGINT NOT NULL DEFAULT 0," +
		"`status` VARCHAR(16) NOT NULL DEFAULT 'UNKNOWN'," +
		"`uploadedBytes` BIGINT NOT NULL DEFAULT 0," +
		"`downloadedBytes` BIGINT NOT NULL DEFAULT 0," +
		"`ratio` DECIMAL(10,4) NOT NULL DEFAULT 0," +
		"`seedTimeSecs` INT NOT NULL DEFAULT 0," +
		"`leechTimeSecs` INT NOT NULL DEFAULT 0," +
		"`completedAt` TIMESTAMP(3) NULL," +
		"`lastActivityAt` TIMESTAMP(3) NULL," +
		"`syncedAt` TIMESTAMP(3) NULL," +
		"`createdAt` TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)," +
		"`updatedAt` TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3)," +
		"UNIQUE KEY `TrackerTorrent_accountId_siteTorrentId_key` (`accountId`, `siteTorrentId`)," +
		"INDEX `TrackerTorrent_accountId_status_idx` (`accountId`, `status`)," +
		"INDEX `TrackerTorrent_accountId_lastActivityAt_idx` (`accountId`, `lastActivityAt`)," +
		"CONSTRAINT `TrackerTorrent_accountId_fkey` FOREIGN KEY (`accountId`) REFERENCES `TrackerAccount`(`id`) ON DELETE CASCADE" +
		") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;" +
		"CREATE TABLE IF NOT EXISTS `HitAndRun` (" +
		"`id` VARCHAR(64) NOT NULL PRIMARY KEY," +
		"`accountId` VARCHAR(64) NOT NULL," +
		"`torrentId` VARCHAR(64)," +
		"`siteTorrentId` VARCHAR(64)," +
		"`torrentName` VARCHAR(512)," +
		"`detectedAt` TIMESTAMP(3) NOT NULL," +
		"`status` VARCHAR(16) NOT NULL DEFAULT 'ACTIVE'," +
		"`resolvedAt` TIMESTAMP(3) NULL," +
		"`rawData` JSON," +
		"`createdAt` TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)," +
		"`updatedAt` TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3)," +
		"INDEX `HitAndRun_accountId_status_idx` (`accountId`, `status`)," +
		"CONSTRAINT `HitAndRun_torrentId_fkey` FOREIGN KEY (`torrentId`) REFERENCES `TrackerTorrent`(`id`) ON DELETE SET NULL" +
		") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;" +
		"CREATE TABLE IF NOT EXISTS `SyncJob` (" +
		"`id` VARCHAR(64) NOT NULL PRIMARY KEY," +
		"`accountId` VARCHAR(64) NOT NULL," +
		"`type` VARCHAR(32) NOT NULL," +
		"`status` VARCHAR(16) NOT NULL DEFAULT 'PENDING'," +
		"`startedAt` TIMESTAMP(3) NULL," +
		"`finishedAt` TIMESTAMP(3) NULL," +
		"`durationMs` INT," +
		"`records` INT," +
		"`errorCode` VARCHAR(64)," +
		"`errorMessage` TEXT," +
		"`createdAt` TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)," +
		"`updatedAt` TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3)," +
		"INDEX `SyncJob_accountId_type_createdAt_idx` (`accountId`, `type`, `createdAt`)," +
		"INDEX `SyncJob_status_createdAt_idx` (`status`, `createdAt`)," +
		"CONSTRAINT `SyncJob_accountId_fkey` FOREIGN KEY (`accountId`) REFERENCES `TrackerAccount`(`id`) ON DELETE CASCADE" +
		") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;" +
		"CREATE TABLE IF NOT EXISTS `SyncLog` (" +
		"`id` VARCHAR(64) NOT NULL PRIMARY KEY," +
		"`jobId` VARCHAR(64) NOT NULL," +
		"`level` VARCHAR(16) NOT NULL DEFAULT 'INFO'," +
		"`message` TEXT NOT NULL," +
		"`metadata` JSON," +
		"`createdAt` TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)," +
		"INDEX `SyncLog_jobId_createdAt_idx` (`jobId`, `createdAt`)," +
		"CONSTRAINT `SyncLog_jobId_fkey` FOREIGN KEY (`jobId`) REFERENCES `SyncJob`(`id`) ON DELETE CASCADE" +
		") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;" +
		"CREATE TABLE IF NOT EXISTS `Alert` (" +
		"`id` VARCHAR(64) NOT NULL PRIMARY KEY," +
		"`accountId` VARCHAR(64)," +
		"`type` VARCHAR(32) NOT NULL," +
		"`title` VARCHAR(255) NOT NULL," +
		"`message` TEXT NOT NULL," +
		"`severity` VARCHAR(16) NOT NULL DEFAULT 'INFO'," +
		"`isRead` BOOLEAN NOT NULL DEFAULT FALSE," +
		"`isDismissed` BOOLEAN NOT NULL DEFAULT FALSE," +
		"`metadata` JSON," +
		"`createdAt` TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)," +
		"`updatedAt` TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3)," +
		"INDEX `Alert_isRead_severity_createdAt_idx` (`isRead`, `severity`, `createdAt`)," +
		"CONSTRAINT `Alert_accountId_fkey` FOREIGN KEY (`accountId`) REFERENCES `TrackerAccount`(`id`) ON DELETE SET NULL" +
		") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;" +
		"CREATE TABLE IF NOT EXISTS `SystemSetting` (" +
		"`id` VARCHAR(64) NOT NULL PRIMARY KEY," +
		"`userId` VARCHAR(64)," +
		"`key` VARCHAR(128) NOT NULL," +
		"`value` TEXT NOT NULL," +
		"`createdAt` TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)," +
		"`updatedAt` TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3)," +
		"UNIQUE KEY `SystemSetting_userId_key_key` (`userId`, `key`)," +
		"CONSTRAINT `SystemSetting_userId_fkey` FOREIGN KEY (`userId`) REFERENCES `User`(`id`) ON DELETE CASCADE" +
		") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;",
	// 20260808055557 / 20260808115854：init DDL 已含这两列（MySQL 版合并），占位保持版本号
	"SELECT 1;",
	"SELECT 1;",
	// 20260808131359_add_fake_seed_job
	"CREATE TABLE IF NOT EXISTS `FakeSeedJob` (" +
		"`id` VARCHAR(64) NOT NULL PRIMARY KEY," +
		"`accountId` VARCHAR(64) NOT NULL," +
		"`torrentId` VARCHAR(64) NOT NULL," +
		"`torrentName` VARCHAR(512) NOT NULL," +
		"`infoHash` VARCHAR(64) NOT NULL," +
		"`trackerUrl` VARCHAR(1024) NOT NULL," +
		"`totalSize` BIGINT NOT NULL DEFAULT 0," +
		"`peerId` VARCHAR(32) NOT NULL," +
		"`peerKey` VARCHAR(32) NOT NULL," +
		"`port` INT NOT NULL DEFAULT 34567," +
		"`status` VARCHAR(16) NOT NULL DEFAULT 'STOPPED'," +
		"`interval` INT NOT NULL DEFAULT 300," +
		"`lastReportAt` TIMESTAMP(3) NULL," +
		"`nextReportAt` TIMESTAMP(3) NULL," +
		"`errorMessage` TEXT," +
		"`createdAt` TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)," +
		"`updatedAt` TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3)," +
		"UNIQUE KEY `FakeSeedJob_infoHash_key` (`infoHash`)," +
		"CONSTRAINT `FakeSeedJob_accountId_fkey` FOREIGN KEY (`accountId`) REFERENCES `TrackerAccount`(`id`) ON DELETE CASCADE" +
		") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;",
}

var migrationNames = []string{
	"20260807065801_init",
	"20260808055557_add_bonus_hourly_rate",
	"20260808115854_add_joined_at",
	"20260808131359_add_fake_seed_job",
}

// Migrate 逐条执行幂等 DDL。
func Migrate(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, "CREATE TABLE IF NOT EXISTS `_go_migration` ("+
		"id INT AUTO_INCREMENT PRIMARY KEY,"+
		"name VARCHAR(255) NOT NULL UNIQUE,"+
		"appliedAt TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP"+
		") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4"); err != nil {
		return fmt.Errorf("create migration table: %w", err)
	}

	for i, sqlText := range migrations {
		name := migrationNames[i]
		var exists bool
		if err := db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM `_go_migration` WHERE name=?)", name).Scan(&exists); err != nil {
			return err
		}
		if exists {
			continue
		}
		for _, stmt := range splitSQL(sqlText) {
			stmt = strings.TrimSpace(stmt)
			if stmt == "" || stmt == "SELECT 1" {
				continue
			}
			if _, err := db.ExecContext(ctx, stmt); err != nil {
				return fmt.Errorf("migration %s: %w", name, err)
			}
		}
		if _, err := db.ExecContext(ctx, "INSERT IGNORE INTO `_go_migration`(name) VALUES(?)", name); err != nil {
			return err
		}
		log.Printf("applied migration %s", name)
	}
	return nil
}

// splitSQL 按 ";" 切分（跳过引号内的分号）
func splitSQL(sqlText string) []string {
	var out []string
	var cur strings.Builder
	inSQuote, inBQuote := false, false
	for i := 0; i < len(sqlText); i++ {
		c := sqlText[i]
		switch {
		case c == '\'' && !inBQuote:
			inSQuote = !inSQuote
		case c == '`' && !inSQuote:
			inBQuote = !inBQuote
		}
		if c == ';' && !inSQuote && !inBQuote {
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

// NewID 生成 25 位 cuid 风格 ID
func NewID() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return fmt.Sprintf("c%s%s", fmt.Sprintf("%08x", time.Now().UnixMilli()), hex.EncodeToString(b))[:25]
}

// Seed 初始化默认管理员与 M-Team 站点（幂等）
func Seed(ctx context.Context, db *sql.DB, cfg *config.Config) error {
	defaultEmail := os.Getenv("SEED_ADMIN_EMAIL")
	if defaultEmail == "" {
		defaultEmail = "admin@pt-manager.local"
	}
	defaultPassword := os.Getenv("SEED_ADMIN_PASSWORD")
	if defaultPassword == "" {
		return fmt.Errorf("SEED_ADMIN_PASSWORD is required to initialize the admin user")
	}

	hashed, err := cryptoutil.HashPassword(defaultPassword)
	if err != nil {
		return err
	}

	var userID string
	err = db.QueryRowContext(ctx, "SELECT `id` FROM `User` WHERE `email`=?", defaultEmail).Scan(&userID)
	switch {
	case err == nil:
		var pwd string
		if err := db.QueryRowContext(ctx, "SELECT `password` FROM `User` WHERE `id`=?", userID).Scan(&pwd); err != nil {
			return err
		}
		if pwd == "PENDING_AUTH_PHASE_5" {
			_, err = db.ExecContext(ctx, "UPDATE `User` SET `password`=?, `updatedAt`=NOW(3) WHERE `id`=?", hashed, userID)
			if err != nil {
				return err
			}
		}
	case err == sql.ErrNoRows:
		userID = NewID()
		_, err = db.ExecContext(ctx,
			"INSERT INTO `User`(`id`,`email`,`name`,`password`,`createdAt`,`updatedAt`) VALUES(?,?,?,?,"+sqlNow+","+sqlNow+")",
			userID, defaultEmail, "Administrator", hashed)
		if err != nil {
			return err
		}
	default:
		return err
	}
	log.Printf("✓ Default user ready: %s (%s)", defaultEmail, userID)

	_, err = db.ExecContext(ctx,
		"INSERT INTO `TrackerSite`(`id`,`code`,`name`,`apiBaseUrl`,`isEnabled`,`createdAt`,`updatedAt`) "+
			"VALUES(?,'MTEAM','M-Team',?,TRUE,"+sqlNow+","+sqlNow+") "+
			"ON DUPLICATE KEY UPDATE `apiBaseUrl`=VALUES(`apiBaseUrl`), `updatedAt`=NOW(3)",
		NewID(), cfg.MTeamBaseURL)
	if err != nil {
		return err
	}
	log.Printf("✓ Tracker site ready: MTEAM → %s", cfg.MTeamBaseURL)
	log.Println("Seed completed. You can now create tracker accounts via POST /api/v1/accounts")
	return nil
}

// sqlNow 兼容写法：MySQL NOW(3)
const sqlNow = "NOW(3)"
