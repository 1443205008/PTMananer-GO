// Package domain: 统一领域模型 — 所有 Provider 必须映射到这里。
// 这些类型与任何 PT 站点的 API 字段名无关（与 TS 版 packages/shared 对齐）。
package domain

import "time"

// TrackerSiteCode 站点代码
const (
	SiteMTeam = "MTEAM"
)

// AccountStatus 账户状态
type AccountStatus string

const (
	AccountActive            AccountStatus = "ACTIVE"
	AccountInactive          AccountStatus = "INACTIVE"
	AccountCredentialInvalid AccountStatus = "CREDENTIAL_INVALID"
	AccountSyncError         AccountStatus = "SYNC_ERROR"
)

// TorrentStatus 种子状态
type TorrentStatus string

const (
	TorrentSeeding   TorrentStatus = "SEEDING"
	TorrentLeeching  TorrentStatus = "LEECHING"
	TorrentCompleted TorrentStatus = "COMPLETED"
	TorrentStopped   TorrentStatus = "STOPPED"
	TorrentUnknown   TorrentStatus = "UNKNOWN"
)

// SyncJobType 同步任务类型
type SyncJobType string

const (
	SyncProfileSync     SyncJobType = "PROFILE_SYNC"
	SyncStatsSync       SyncJobType = "STATS_SYNC"
	SyncTorrentSync     SyncJobType = "TORRENT_SYNC"
	SyncHNRSync         SyncJobType = "HNR_SYNC"
	SyncBonusSync       SyncJobType = "BONUS_SYNC"
	SyncSiteHealthCheck SyncJobType = "SITE_HEALTH_CHECK"
	SyncDailySnapshot   SyncJobType = "DAILY_SNAPSHOT"
)

// SyncStatus 同步状态
type SyncStatus string

const (
	SyncPending SyncStatus = "PENDING"
	SyncRunning SyncStatus = "RUNNING"
	SyncSuccess SyncStatus = "SUCCESS"
	SyncFailed  SyncStatus = "FAILED"
	SyncPartial SyncStatus = "PARTIAL"
)

// SiteStatus 站点健康
type SiteStatus string

const (
	SiteHealthy  SiteStatus = "HEALTHY"
	SiteDegraded SiteStatus = "DEGRADED"
	SiteOffline  SiteStatus = "OFFLINE"
	SiteUnknown  SiteStatus = "UNKNOWN"
)

// HnrStatus H&R 状态
type HnrStatus string

const (
	HnrActive   HnrStatus = "ACTIVE"
	HnrResolved HnrStatus = "RESOLVED"
	HnrExpired  HnrStatus = "EXPIRED"
)

// AlertType 告警类型
type AlertType string

const (
	AlertHNDDetected AlertType = "HNR_DETECTED"
	AlertAuthFailed  AlertType = "AUTH_FAILED"
	AlertSyncFailed  AlertType = "SYNC_FAILED"
	AlertSiteOffline AlertType = "SITE_OFFLINE"
	AlertAPIChanged  AlertType = "API_CHANGED"
	AlertRateLimited AlertType = "RATE_LIMITED"
)

// AlertSeverity 告警级别
type AlertSeverity string

const (
	AlertSevInfo     AlertSeverity = "INFO"
	AlertSevWarning  AlertSeverity = "WARNING"
	AlertSevError    AlertSeverity = "ERROR"
	AlertSevCritical AlertSeverity = "CRITICAL"
)

// FakeSeedStatus 保种任务状态
type FakeSeedStatus string

const (
	FakeSeedRunning FakeSeedStatus = "RUNNING"
	FakeSeedStopped FakeSeedStatus = "STOPPED"
	FakeSeedError   FakeSeedStatus = "ERROR"
)

// TrackerErrorCode 统一错误码
type TrackerErrorCode string

const (
	ErrAuthInvalid     TrackerErrorCode = "AUTH_INVALID"
	ErrAuthExpired     TrackerErrorCode = "AUTH_EXPIRED"
	ErrRateLimited     TrackerErrorCode = "RATE_LIMITED"
	ErrSiteOffline     TrackerErrorCode = "SITE_OFFLINE"
	ErrNetworkError    TrackerErrorCode = "NETWORK_ERROR"
	ErrTimeout         TrackerErrorCode = "TIMEOUT"
	ErrAPIChanged      TrackerErrorCode = "API_CHANGED"
	ErrInvalidResponse TrackerErrorCode = "INVALID_RESPONSE"
	ErrUnknown         TrackerErrorCode = "UNKNOWN_ERROR"
)

// Coder 供跨包错误码提取（MTeamError 等实现此接口）
type Coder interface {
	error
	TrackerCode() TrackerErrorCode
}

// TrackerError 领域错误（对齐 TS 版 common/errors/tracker-error.ts）
type TrackerError struct {
	Code      TrackerErrorCode
	Message   string
	AccountID string
}

func (e *TrackerError) Error() string { return e.Message }

func (e *TrackerError) TrackerCode() TrackerErrorCode { return e.Code }

func NewTrackerError(code TrackerErrorCode, msg string) *TrackerError {
	return &TrackerError{Code: code, Message: msg}
}

// RequiresCredentialReset 是否需要停用账户（凭据失效）
func (e *TrackerError) RequiresCredentialReset() bool {
	return e.Code == ErrAuthInvalid || e.Code == ErrAuthExpired
}

// UserMessage 用户可见提示（不暴露内部细节）
func (e *TrackerError) UserMessage() string {
	switch e.Code {
	case ErrAuthInvalid:
		return "API Key 无效或已失效"
	case ErrAuthExpired:
		return "API Key 已过期，请重新生成"
	case ErrRateLimited:
		return "请求过于频繁，请稍后再试"
	case ErrSiteOffline:
		return "站点暂时无法访问"
	case ErrNetworkError:
		return "网络连接异常"
	case ErrTimeout:
		return "请求超时"
	case ErrAPIChanged:
		return "站点 API 结构变化，请联系管理员"
	case ErrInvalidResponse:
		return "收到意外的响应格式"
	default:
		return "同步时发生未知错误"
	}
}

// ─── 统一领域模型 ─────────────────────────────────────────────────────────

// ConnectionResult 连接测试结果
type ConnectionResult struct {
	Success   bool              `json:"success"`
	LatencyMs int64             `json:"latencyMs,omitempty"`
	Profile   *MiniProfile      `json:"profile,omitempty"`
	Error     *TrackerErrorInfo `json:"error,omitempty"`
}

type MiniProfile struct {
	ExternalUserID string `json:"externalUserId"`
	Username       string `json:"username"`
}

type TrackerErrorInfo struct {
	Code    TrackerErrorCode `json:"code"`
	Message string           `json:"message"`
}

// TrackerProfile 用户资料
type TrackerProfile struct {
	SiteCode       string
	ExternalUserID string
	Username       string
	Email          string
	JoinedAt       time.Time
	RoleID         int
	LevelName      string
	AvatarURL      string
	Title          string
	IsParked       bool
	IsEnabled      bool
}

// TrackerStats 当前统计
type TrackerStats struct {
	UploadBytes     int64
	DownloadBytes   int64
	Ratio           float64 // -1 = 无限
	Bonus           float64
	SeedingCount    int
	SeedingBytes    int64
	LeechingCount   int
	HitAndRunCount  int
	IsWarned        bool
	IsVIP           bool
	IsDonor         bool
	LastLoginAt     *time.Time
	LastTrackerAt   *time.Time
	SnapshotTime    time.Time
	BonusHourlyRate *float64
}

// TrackerTorrent 种子
type TrackerTorrent struct {
	SiteTorrentID   string
	Name            string
	SizeBytes       int64
	Status          TorrentStatus
	UploadedBytes   int64
	DownloadedBytes int64
	Ratio           float64
	SeedTimeSecs    int
	LeechTimeSecs   int
	CompletedAt     *time.Time
	LastActivityAt  *time.Time
}

// BonusStats 魔力值统计
type BonusStats struct {
	Current    float64
	HourlyRate *float64
	UpdatedAt  time.Time
}

// HitAndRun H&R 记录
type HitAndRun struct {
	SiteTorrentID string
	TorrentName   string
	DetectedAt    time.Time
	Status        HnrStatus
	ResolvedAt    *time.Time
}

// MessageStats 消息统计
type MessageStats struct {
	UnreadCount int
	CheckedAt   time.Time
}

// TrackerSiteStatusResult 站点健康结果
type TrackerSiteStatusResult struct {
	Status    SiteStatus
	LatencyMs int64
	CheckedAt time.Time
	Message   string
}
