// Package syncer: 同步编排（service）+ Redis 队列（替代 BullMQ）+ 定时调度。
package syncer

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/1443205008/ptmanager-go/internal/db"
	"github.com/1443205008/ptmanager-go/internal/domain"
	"github.com/1443205008/ptmanager-go/internal/providers"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// Queue 用 Redis List 实现的任务队列（替代 BullMQ，语义对齐：
// 立即入队返回、异步消费、指数退避重试）。
type Queue struct {
	rdb  *redis.Client
	svc  *Service
	key  string
}

func NewQueue(rdb *redis.Client, svc *Service) *Queue {
	return &Queue{rdb: rdb, svc: svc, key: "ptmanager:sync:queue"}
}

type Job struct {
	Name       string `json:"name"` // account-stats | torrent-sync
	AccountID  string `json:"accountId"`
	Trigger    string `json:"trigger"` // manual | scheduled
	Attempts   int    `json:"attempts"`
	MaxAttempt int    `json:"maxAttempt"`
}

// Enqueue 入队（立即返回 jobId）
func (q *Queue) Enqueue(ctx context.Context, name, accountID, trigger string, maxAttempts int) (string, error) {
	jobID := fmt.Sprintf("%s:%s:%d", name, accountID, time.Now().UnixMilli())
	payload, _ := json.Marshal(Job{Name: name, AccountID: accountID, Trigger: trigger, MaxAttempt: maxAttempts})
	if err := q.rdb.RPush(ctx, q.key, payload).Err(); err != nil {
		return "", err
	}
	return jobID, nil
}

// RunWorker 消费循环
func (q *Queue) RunWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		res, err := q.rdb.BLPop(ctx, 5*time.Second, q.key).Result()
		if err != nil {
			if err != redis.Nil && ctx.Err() == nil {
				log.Printf("[queue] BLPop error: %v", err)
				time.Sleep(2 * time.Second)
			}
			continue
		}
		if len(res) < 2 {
			continue
		}
		var job Job
		if err := json.Unmarshal([]byte(res[1]), &job); err != nil {
			log.Printf("[queue] bad payload: %v", err)
			continue
		}
		q.process(ctx, job)
	}
}

func (q *Queue) process(ctx context.Context, job Job) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[queue] panic in job %s: %v", job.Name, r)
		}
	}()
	var err error
	switch job.Name {
	case "account-stats":
		err = q.svc.SyncAccountStats(job.AccountID)
	case "torrent-sync":
		err = q.svc.SyncTorrents(job.AccountID)
	default:
		log.Printf("[queue] unknown job name: %s", job.Name)
		return
	}
	if err != nil {
		job.Attempts++
		if job.Attempts < job.MaxAttempt {
			// 指数退避：5s * 2^(n-1)
			delay := time.Duration(5*(1<<(job.Attempts-1))) * time.Second
			log.Printf("[queue] job %s:%s failed (attempt %d/%d), retry in %v: %v",
				job.Name, job.AccountID, job.Attempts, job.MaxAttempt, delay, err)
			payload, _ := json.Marshal(job)
			q.rdb.LPush(ctx, q.key+"retry", payload)
			go func() {
				time.Sleep(delay)
				// 从 retry 队列取回（可能已有多个，逐个搬回主队列）
				for {
					item, err := q.rdb.LPop(context.Background(), q.key+"retry").Result()
					if err != nil {
						return
					}
					// 简化：睡够再回主队列
					q.rdb.RPush(context.Background(), q.key, item)
				}
			}()
			return
		}
		log.Printf("[queue] job %s:%s failed permanently: %v", job.Name, job.AccountID, err)
	}
}

// ─── Scheduler：定时入队 ──────────────────────────────────────────────────

type Scheduler struct {
	pool     *pgxpool.Pool
	queue    *Queue
	settings SettingsService
	stop     chan struct{}
}

func NewScheduler(pool *pgxpool.Pool, queue *Queue, settings SettingsService) *Scheduler {
	return &Scheduler{pool: pool, queue: queue, settings: settings, stop: make(chan struct{})}
}

func (s *Scheduler) Run(ctx context.Context) {
	time.Sleep(5 * time.Second)
	for {
		enabled, err := s.settings.SyncEnabled(ctx)
		if err != nil {
			log.Printf("[scheduler] cannot read settings, using defaults: %v", err)
			enabled = true
		}
		if !enabled {
			log.Println("[scheduler] scheduled account sync disabled")
			select {
			case <-ctx.Done():
				return
			case <-s.stop:
				return
			case <-time.After(time.Minute):
				continue
			}
		}
		interval, err := s.settings.SyncIntervalMinutes(ctx)
		if err != nil || interval <= 0 {
			interval = 30
		}
		log.Printf("[scheduler] sync interval: %d minutes", interval)
		select {
		case <-ctx.Done():
			return
		case <-s.stop:
			return
		case <-time.After(time.Duration(interval) * time.Minute):
			s.scheduleAccountSyncs(ctx)
		}
	}
}

func (s *Scheduler) scheduleAccountSyncs(ctx context.Context) {
	rows, err := s.pool.Query(ctx, `
		SELECT "id" FROM "TrackerAccount"
		WHERE "isEnabled"=true AND "syncEnabled"=true AND "status" != 'CREDENTIAL_INVALID'`)
	if err != nil {
		log.Printf("[scheduler] query accounts: %v", err)
		return
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			continue
		}
		if _, err := s.queue.Enqueue(ctx, "account-stats", id, "scheduled", 3); err == nil {
			if _, err := s.queue.Enqueue(ctx, "torrent-sync", id, "scheduled", 2); err == nil {
				count++
			}
		}
	}
	if count > 0 {
		log.Printf("[scheduler] enqueued %d scheduled account stats + torrent syncs", count)
	}
}

// ─── 同步业务 ──────────────────────────────────────────────────────────────

type Service struct {
	pool     *pgxpool.Pool
	registry *providers.Registry
	settings SettingsService
}

func NewService(pool *pgxpool.Pool, registry *providers.Registry, settings SettingsService) *Service {
	return &Service{pool: pool, registry: registry, settings: settings}
}

// SettingsService 同步需要的设置子集（由 settings.Service 实现）
type SettingsService interface {
	SyncEnabled(ctx context.Context) (bool, error)
	SyncIntervalMinutes(ctx context.Context) (int, error)
	AlertEnabled(ctx context.Context) (bool, error)
}

// SyncAccountStats 同步单账户 profile + stats + 当日快照。
func (s *Service) SyncAccountStats(accountID string) error {
	ctx := context.Background()

	var siteCode string
	err := s.pool.QueryRow(ctx, `SELECT s."code" FROM "TrackerAccount" a JOIN "TrackerSite" s ON s."id"=a."siteId" WHERE a."id"=$1`, accountID).Scan(&siteCode)
	if err != nil {
		log.Printf("[sync] account %s not found, skipping", accountID)
		return nil
	}
	provider, err := s.registry.Get(siteCode)
	if err != nil {
		return err
	}

	jobID := db.NewID()
	start := time.Now()
	_, err = s.pool.Exec(ctx, `
		INSERT INTO "SyncJob"("id","accountId","type","status","startedAt","createdAt","updatedAt")
		VALUES($1,$2,'STATS_SYNC','RUNNING',$3,$3,$3)`, jobID, accountID, start)
	if err != nil {
		return err
	}

	fail := func(err error) error {
		s.handleSyncFailure(ctx, accountID, jobID, time.Since(start).Milliseconds(), err)
		return err
	}

	// 1. profile
	profile, err := provider.GetProfile(accountID)
	if err != nil {
		return fail(err)
	}
	if _, err := s.pool.Exec(ctx, `
		UPDATE "TrackerAccount" SET "externalUserId"=$1, "username"=$2, "avatarUrl"=$3, "joinedAt"=$4, "updatedAt"=NOW()
		WHERE "id"=$5`,
		profile.ExternalUserID, profile.Username, nilIfEmpty(profile.AvatarURL), profile.JoinedAt, accountID); err != nil {
		return fail(err)
	}

	// 2. stats upsert
	stats, err := provider.GetStats(accountID)
	if err != nil {
		return fail(err)
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO "TrackerStats"("id","accountId","uploadBytes","downloadBytes","ratio","bonus",
			"seedingCount","seedingBytes","leechingCount","hitAndRunCount","roleId","levelName",
			"isWarned","isVip","isDonor","lastLoginAt","lastTrackerAt","siteStatus","syncedAt","bonusHourlyRate","createdAt","updatedAt")
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,'HEALTHY',$18,$19,NOW(),NOW())
		ON CONFLICT ("accountId") DO UPDATE SET
			"uploadBytes"=$3,"downloadBytes"=$4,"ratio"=$5,"bonus"=$6,"seedingCount"=$7,"seedingBytes"=$8,
			"leechingCount"=$9,"hitAndRunCount"=$10,"roleId"=$11,"levelName"=$12,"isWarned"=$13,"isVip"=$14,
			"isDonor"=$15,"lastLoginAt"=$16,"lastTrackerAt"=$17,"siteStatus"='HEALTHY',"syncedAt"=$18,
			"bonusHourlyRate"=$19,"updatedAt"=NOW()`,
		db.NewID(), accountID, stats.UploadBytes, stats.DownloadBytes, stats.Ratio, stats.Bonus,
		stats.SeedingCount, stats.SeedingBytes, stats.LeechingCount, stats.HitAndRunCount,
		roleIDOrNull(profile.RoleID), nilIfEmptyStr(profile.LevelName),
		stats.IsWarned, stats.IsVIP, stats.IsDonor, stats.LastLoginAt, stats.LastTrackerAt,
		time.Now(), stats.BonusHourlyRate)
	if err != nil {
		return fail(err)
	}

	// 3. 当日快照（UTC 日期）
	if err := s.upsertDailySnapshot(ctx, accountID, stats); err != nil {
		return fail(err)
	}

	// 4. 标记 ACTIVE
	if _, err := s.pool.Exec(ctx, `
		UPDATE "TrackerAccount" SET "status"='ACTIVE', "lastSyncAt"=NOW(), "updatedAt"=NOW() WHERE "id"=$1`, accountID); err != nil {
		return fail(err)
	}

	s.finishJob(ctx, jobID, "SUCCESS", time.Since(start).Milliseconds(), 1)
	log.Printf("[sync] synced account %s in %dms", accountID, time.Since(start).Milliseconds())
	return nil
}

func (s *Service) upsertDailySnapshot(ctx context.Context, accountID string, stats domain.TrackerStats) error {
	now := time.Now().UTC()
	snapshotDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	_, err := s.pool.Exec(ctx, `
		INSERT INTO "TrackerDailySnapshot"("id","accountId","snapshotDate","uploadBytes","downloadBytes","ratio","bonus",
			"seedingCount","seedingBytes","leechingCount","hitAndRunCount","createdAt")
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,NOW())
		ON CONFLICT ("accountId","snapshotDate") DO UPDATE SET
			"uploadBytes"=$4,"downloadBytes"=$5,"ratio"=$6,"bonus"=$7,"seedingCount"=$8,
			"seedingBytes"=$9,"leechingCount"=$10,"hitAndRunCount"=$11`,
		db.NewID(), accountID, snapshotDate, stats.UploadBytes, stats.DownloadBytes, stats.Ratio, stats.Bonus,
		stats.SeedingCount, stats.SeedingBytes, stats.LeechingCount, stats.HitAndRunCount)
	return err
}

// SyncTorrents 同步种子列表（seeding + leeching）。
func (s *Service) SyncTorrents(accountID string) error {
	ctx := context.Background()
	var siteCode string
	err := s.pool.QueryRow(ctx, `SELECT s."code" FROM "TrackerAccount" a JOIN "TrackerSite" s ON s."id"=a."siteId" WHERE a."id"=$1`, accountID).Scan(&siteCode)
	if err != nil {
		log.Printf("[sync] account %s not found, skipping", accountID)
		return nil
	}
	provider, err := s.registry.Get(siteCode)
	if err != nil {
		return err
	}

	jobID := db.NewID()
	start := time.Now()
	_, err = s.pool.Exec(ctx, `
		INSERT INTO "SyncJob"("id","accountId","type","status","startedAt","createdAt","updatedAt")
		VALUES($1,$2,'TORRENT_SYNC','RUNNING',$3,$3,$3)`, jobID, accountID, start)
	if err != nil {
		return err
	}

	fail := func(err error) error {
		s.handleSyncFailure(ctx, accountID, jobID, time.Since(start).Milliseconds(), err)
		return err
	}

	// M-Team getUserTorrentList 严格限速，冷却 8s
	time.Sleep(8 * time.Second)
	torrents, err := provider.GetTorrents(accountID)
	if err != nil {
		return fail(err)
	}

	for _, t := range torrents {
		if _, err := s.pool.Exec(ctx, `
			INSERT INTO "TrackerTorrent"("id","accountId","siteTorrentId","name","sizeBytes","status",
				"uploadedBytes","downloadedBytes","ratio","seedTimeSecs","leechTimeSecs","completedAt","lastActivityAt","syncedAt","createdAt","updatedAt")
			VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,NOW(),NOW(),NOW())
			ON CONFLICT ("accountId","siteTorrentId") DO UPDATE SET
				"name"=$4,"sizeBytes"=$5,"status"=$6,"uploadedBytes"=$7,"downloadedBytes"=$8,"ratio"=$9,
				"seedTimeSecs"=$10,"leechTimeSecs"=$11,"completedAt"=$12,"lastActivityAt"=$13,"syncedAt"=NOW(),"updatedAt"=NOW()`,
			db.NewID(), accountID, t.SiteTorrentID, t.Name, t.SizeBytes, string(t.Status),
			t.UploadedBytes, t.DownloadedBytes, t.Ratio, t.SeedTimeSecs, t.LeechTimeSecs,
			t.CompletedAt, t.LastActivityAt); err != nil {
			return fail(err)
		}
	}

	// 更新 seedingBytes：做种种子大小之和
	var seedingBytes int64
	for _, t := range torrents {
		if t.Status == domain.TorrentSeeding {
			seedingBytes += t.SizeBytes
		}
	}
	if _, err := s.pool.Exec(ctx, `
		UPDATE "TrackerStats" SET "seedingBytes"=$1, "updatedAt"=NOW() WHERE "accountId"=$2`, seedingBytes, accountID); err != nil {
		return fail(err)
	}

	s.finishJob(ctx, jobID, "SUCCESS", time.Since(start).Milliseconds(), len(torrents))
	log.Printf("[sync] torrent-sync %s: %d torrents in %dms", accountID, len(torrents), time.Since(start).Milliseconds())
	return nil
}

func (s *Service) finishJob(ctx context.Context, jobID string, status string, durationMs int64, records int) {
	_, _ = s.pool.Exec(ctx, `
		UPDATE "SyncJob" SET "status"=$1, "finishedAt"=NOW(), "durationMs"=$2, "records"=$3, "updatedAt"=NOW()
		WHERE "id"=$4`, status, durationMs, records, jobID)
}

// handleSyncFailure 设置账户状态 + 记录 Job + 建告警（绝不含凭据）
func (s *Service) handleSyncFailure(ctx context.Context, accountID, jobID string, durationMs int64, err error) {
	isAuthErr := false
	code := domain.ErrUnknown
	if te, ok := err.(*domain.TrackerError); ok {
		code = te.Code
		isAuthErr = te.RequiresCredentialReset()
	} else if mte, ok := err.(*mteamErrWithCode); ok {
		code = mte.code
		isAuthErr = code == domain.ErrAuthInvalid || code == domain.ErrAuthExpired
	}

	accountStatus := "SYNC_ERROR"
	alertType := "SYNC_FAILED"
	severity := "WARNING"
	if isAuthErr {
		accountStatus = "CREDENTIAL_INVALID"
		alertType = "AUTH_FAILED"
		severity = "ERROR"
	}

	safeMessage := err.Error()

	alertEnabled := true
	if s.settings != nil {
		if v, err := s.settings.AlertEnabled(ctx); err == nil {
			alertEnabled = v
		}
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		log.Printf("[sync] failure tx: %v", err)
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	_, _ = tx.Exec(ctx, `UPDATE "TrackerAccount" SET "status"=$1, "updatedAt"=NOW() WHERE "id"=$2`, accountStatus, accountID)
	_, _ = tx.Exec(ctx, `
		UPDATE "SyncJob" SET "status"='FAILED', "finishedAt"=NOW(), "durationMs"=$1, "errorCode"=$2, "errorMessage"=$3, "updatedAt"=NOW()
		WHERE "id"=$4`, durationMs, string(code), safeMessage, jobID)
	if alertEnabled {
		title := "同步失败"
		if isAuthErr {
			title = "API Key 失效"
		}
		_, _ = tx.Exec(ctx, `
			INSERT INTO "Alert"("id","accountId","type","title","message","severity","createdAt","updatedAt")
			VALUES($1,$2,$3,$4,$5,$6,NOW(),NOW())`,
			db.NewID(), accountID, alertType, title, safeMessage, severity)
	}
	if err := tx.Commit(ctx); err != nil {
		log.Printf("[sync] failure commit: %v", err)
	}
	log.Printf("[sync] failed for account %s: [%s] %s", accountID, code, safeMessage)
}

// mteamErrWithCode 供错误码提取（避免依赖 mteam 包）
type mteamErrWithCode struct{ code domain.TrackerErrorCode }

func (e *mteamErrWithCode) Error() string { return string(e.code) }

func roleIDOrNull(id int) interface{} {
	if id == 0 {
		return nil
	}
	return id
}

func nilIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func nilIfEmptyStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

var _ = strconv.Itoa
var _ = fmt.Sprintf
