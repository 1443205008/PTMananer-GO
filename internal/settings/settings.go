// Package settings: 系统设置（对齐 TS 版 settings 模块，含运行状态与清理）。
package settings

import (
	"context"
	"net/http"
	"time"

	"github.com/1443205008/ptmanager-go/internal/config"
	"github.com/1443205008/ptmanager-go/internal/db"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// Defaults 与 TS 版 SETTINGS_DEFAULTS 一致
var Defaults = map[string]string{
	"systemName":          "PT Manager",
	"timezone":            "Asia/Shanghai",
	"defaultPageSize":     "20",
	"syncEnabled":         "true",
	"syncIntervalMinutes": "30",
	"alertEnabled":        "true",
	"alertRetentionDays":  "90",
	"syncLogRetentionDays": "30",
	"snapshotRetentionDays": "365",
}

// key → 类型（int 的需要数值校验）
var intKeys = map[string]bool{
	"defaultPageSize": true, "syncIntervalMinutes": true, "alertRetentionDays": true,
	"syncLogRetentionDays": true, "snapshotRetentionDays": true,
}
var boolKeys = map[string]bool{"syncEnabled": true, "alertEnabled": true}

// 允许值（对齐 UpdateSettingsDto 的 @IsIn）
var allowedInts = map[string][]int{
	"defaultPageSize":     {20, 50, 100},
	"syncIntervalMinutes": {5, 15, 30, 60},
}

type SystemSettings struct {
	SystemName          string  `json:"systemName"`
	Timezone            string  `json:"timezone"`
	DefaultPageSize     int     `json:"defaultPageSize"`
	SyncEnabled         bool    `json:"syncEnabled"`
	SyncIntervalMinutes int     `json:"syncIntervalMinutes"`
	AlertEnabled        bool    `json:"alertEnabled"`
	AlertRetentionDays  int     `json:"alertRetentionDays"`
	SyncLogRetentionDays int    `json:"syncLogRetentionDays"`
	SnapshotRetentionDays int   `json:"snapshotRetentionDays"`
	UpdatedAt           *string `json:"updatedAt"`
}

type Service struct {
	pool *pgxpool.Pool
	rdb  *redis.Client
	cfg  *config.Config
}

func NewService(pool *pgxpool.Pool, rdb *redis.Client, cfg *config.Config) *Service {
	return &Service{pool: pool, rdb: rdb, cfg: cfg}
}

func (s *Service) defaultUserID(ctx context.Context) (string, error) {
	var id string
	err := s.pool.QueryRow(ctx, `SELECT "id" FROM "User" ORDER BY "createdAt" ASC LIMIT 1`).Scan(&id)
	if err != nil {
		return "", &apiErr{http.StatusNotFound, "系统用户不存在，请先执行数据库初始化"}
	}
	return id, nil
}

func (s *Service) Get(ctx context.Context) (SystemSettings, error) {
	userID, err := s.defaultUserID(ctx)
	if err != nil {
		return SystemSettings{}, err
	}
	rows, err := s.pool.Query(ctx, `SELECT "key","value","updatedAt" FROM "SystemSetting" WHERE "userId"=$1`, userID)
	if err != nil {
		return SystemSettings{}, err
	}
	defer rows.Close()

	vals := map[string]string{}
	var updatedAt *time.Time
	for rows.Next() {
		var k, v string
		var u time.Time
		if err := rows.Scan(&k, &v, &u); err != nil {
			continue
		}
		if _, isDefault := Defaults[k]; isDefault {
			vals[k] = v
			if updatedAt == nil || u.After(*updatedAt) {
				updatedAt = &u
			}
		}
	}

	for k, dv := range Defaults {
		if _, ok := vals[k]; !ok {
			vals[k] = dv
		}
	}

	var updatedStr *string
	if updatedAt != nil {
		fs := updatedAt.UTC().Format(time.RFC3339Nano)
		updatedStr = &fs
	}

	return SystemSettings{
		SystemName:          vals["systemName"],
		Timezone:            vals["timezone"],
		DefaultPageSize:     atoi(vals["defaultPageSize"]),
		SyncEnabled:         vals["syncEnabled"] == "true",
		SyncIntervalMinutes: atoi(vals["syncIntervalMinutes"]),
		AlertEnabled:        vals["alertEnabled"] == "true",
		AlertRetentionDays:  atoi(vals["alertRetentionDays"]),
		SyncLogRetentionDays: atoi(vals["syncLogRetentionDays"]),
		SnapshotRetentionDays: atoi(vals["snapshotRetentionDays"]),
		UpdatedAt:           updatedStr,
	}, nil
}

func (s *Service) Update(ctx context.Context, dto map[string]interface{}) (SystemSettings, error) {
	userID, err := s.defaultUserID(ctx)
	if err != nil {
		return SystemSettings{}, err
	}

	for k, v := range dto {
		if _, ok := Defaults[k]; !ok {
			return SystemSettings{}, &apiErr{http.StatusBadRequest, "不支持的系统设置：" + k}
		}
		var strVal string
		switch t := v.(type) {
		case bool:
			if !boolKeys[k] {
				return SystemSettings{}, &apiErr{http.StatusBadRequest, "不支持的系统设置：" + k}
			}
			strVal = boolToStr(t)
		case float64:
			if !intKeys[k] {
				return SystemSettings{}, &apiErr{http.StatusBadRequest, "不支持的系统设置：" + k}
			}
			strVal = itoa(int(t))
			if allowed, ok := allowedInts[k]; ok && !intIn(int(t), allowed) {
				return SystemSettings{}, &apiErr{http.StatusBadRequest, "不支持的取值：" + k}
			}
		case string:
			strVal = t
		default:
			return SystemSettings{}, &apiErr{http.StatusBadRequest, "不支持的系统设置：" + k}
		}
		if _, err := s.pool.Exec(ctx, `
			INSERT INTO "SystemSetting"("id","userId","key","value","createdAt","updatedAt")
			VALUES($1,$2,$3,$4,NOW(),NOW())
			ON CONFLICT ("userId","key") DO UPDATE SET "value"=$4, "updatedAt"=NOW()`,
			db.NewID(), userID, k, strVal); err != nil {
			return SystemSettings{}, err
		}
	}
	return s.Get(ctx)
}

func (s *Service) Reset(ctx context.Context) (SystemSettings, error) {
	userID, err := s.defaultUserID(ctx)
	if err != nil {
		return SystemSettings{}, err
	}
	if _, err := s.pool.Exec(ctx, `DELETE FROM "SystemSetting" WHERE "userId"=$1`, userID); err != nil {
		return SystemSettings{}, err
	}
	return s.Get(ctx)
}

type Status struct {
	Version     string `json:"version"`
	Environment string `json:"environment"`
	Database    string `json:"database"`
	Redis       string `json:"redis"`
	CheckedAt   string `json:"checkedAt"`
}

func (s *Service) StatusOf(ctx context.Context) Status {
	st := Status{Version: "0.1.0", Environment: s.cfg.NodeEnv, Database: "connected", Redis: "connected", CheckedAt: time.Now().UTC().Format(time.RFC3339Nano)}
	if _, err := s.pool.Exec(ctx, `SELECT 1`); err != nil {
		st.Database = "error"
	}
	if err := s.rdb.Ping(ctx).Err(); err != nil {
		st.Redis = "error"
	}
	return st
}

type CleanupResult struct {
	SyncJobs   int    `json:"syncJobs"`
	Snapshots  int    `json:"snapshots"`
	Alerts     int    `json:"alerts"`
	CleanedAt  string `json:"cleanedAt"`
}

func (s *Service) Cleanup(ctx context.Context) (CleanupResult, error) {
	settings, err := s.Get(ctx)
	if err != nil {
		return CleanupResult{}, err
	}
	now := time.Now()
	syncBefore := now.AddDate(0, 0, -settings.SyncLogRetentionDays)
	snapBefore := now.AddDate(0, 0, -settings.SnapshotRetentionDays)
	alertBefore := now.AddDate(0, 0, -settings.AlertRetentionDays)

	res := CleanupResult{CleanedAt: now.UTC().Format(time.RFC3339Nano)}

	tag, err := s.pool.Exec(ctx, `
		DELETE FROM "SyncJob" WHERE "createdAt" < $1 AND "status" != 'RUNNING'`, syncBefore)
	if err == nil {
		res.SyncJobs = int(tag.RowsAffected())
	}
	tag, err = s.pool.Exec(ctx, `DELETE FROM "TrackerDailySnapshot" WHERE "snapshotDate" < $1`, snapBefore)
	if err == nil {
		res.Snapshots = int(tag.RowsAffected())
	}
	tag, err = s.pool.Exec(ctx, `DELETE FROM "Alert" WHERE "createdAt" < $1 AND "isDismissed"=true`, alertBefore)
	if err == nil {
		res.Alerts = int(tag.RowsAffected())
	}
	return res, nil
}

// SyncEnabled / SyncIntervalMinutes 供 syncer 调度读取
func (s *Service) SyncEnabled(ctx context.Context) (bool, error) {
	settings, err := s.Get(ctx)
	if err != nil {
		return true, err
	}
	return settings.SyncEnabled, nil
}

func (s *Service) AlertEnabled(ctx context.Context) (bool, error) {
	settings, err := s.Get(ctx)
	if err != nil {
		return true, err
	}
	return settings.AlertEnabled, nil
}

func (s *Service) SyncIntervalMinutes(ctx context.Context) (int, error) {
	settings, err := s.Get(ctx)
	if err != nil {
		return 30, err
	}
	return settings.SyncIntervalMinutes, nil
}

// ─── helpers + handlers ───────────────────────────────────────────────────

type apiErr struct {
	status  int
	message string
}

func (e *apiErr) Error() string { return e.message }

func writeErr(c *gin.Context, err error) {
	if ae, ok := err.(*apiErr); ok {
		c.JSON(ae.status, gin.H{"statusCode": ae.status, "message": ae.message})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"statusCode": 500, "message": "服务器内部错误"})
}

func atoi(s string) int {
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return n
		}
		n = n*10 + int(r-'0')
	}
	return n
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [24]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func boolToStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func intIn(n int, list []int) bool {
	for _, v := range list {
		if v == n {
			return true
		}
	}
	return false
}

func GetHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		s, err := svc.Get(c.Request.Context())
		if err != nil {
			writeErr(c, err)
			return
		}
		c.JSON(http.StatusOK, s)
	}
}

func UpdateHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var dto map[string]interface{}
		if err := c.ShouldBindJSON(&dto); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"statusCode": 400, "message": "参数不合法"})
			return
		}
		s, err := svc.Update(c.Request.Context(), dto)
		if err != nil {
			writeErr(c, err)
			return
		}
		c.JSON(http.StatusOK, s)
	}
}

func ResetHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		s, err := svc.Reset(c.Request.Context())
		if err != nil {
			writeErr(c, err)
			return
		}
		c.JSON(http.StatusOK, s)
	}
}

func StatusHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, svc.StatusOf(c.Request.Context()))
	}
}

func CleanupHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		r, err := svc.Cleanup(c.Request.Context())
		if err != nil {
			writeErr(c, err)
			return
		}
		c.JSON(http.StatusOK, r)
	}
}
