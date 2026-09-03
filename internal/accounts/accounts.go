// Package accounts: 站点账户 CRUD + 连接测试（对齐 TS 版 accounts 模块）。
package accounts

import (
	"log"
	"database/sql"
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/1443205008/ptmanager-go/internal/cryptoutil"
	"github.com/1443205008/ptmanager-go/internal/db"
	"github.com/1443205008/ptmanager-go/internal/domain"
	"github.com/1443205008/ptmanager-go/internal/providers"

	"github.com/gin-gonic/gin"
)

type Service struct {
	pool    *sql.DB
	crypto  *cryptoutil.CredentialCrypto
	registry *providers.Registry
}

func NewService(pool *sql.DB, crypto *cryptoutil.CredentialCrypto, registry *providers.Registry) *Service {
	return &Service{pool: pool, crypto: crypto, registry: registry}
}

// AccountResponse ⚠️ 绝不包含 encryptedApiKey / iv / authTag / 明文 API Key
type AccountResponse struct {
	ID              string        `json:"id"`
	SiteCode        string        `json:"siteCode"`
	SiteName        string        `json:"siteName"`
	AccountName     string        `json:"accountName"`
	Remark          *string       `json:"remark"`
	ExternalUserID  *string       `json:"externalUserId"`
	Username        *string       `json:"username"`
	AvatarURL       *string       `json:"avatarUrl"`
	JoinedAt        *string       `json:"joinedAt"`
	IsEnabled       bool          `json:"isEnabled"`
	SyncEnabled     bool          `json:"syncEnabled"`
	Status          string        `json:"status"`
	LastSyncAt      *string       `json:"lastSyncAt"`
	CreatedAt       string        `json:"createdAt"`
	UpdatedAt       string        `json:"updatedAt"`
	Stats           *AccountStats `json:"stats"`
}

type AccountStats struct {
	UploadBytes    string  `json:"uploadBytes"`
	DownloadBytes  string  `json:"downloadBytes"`
	Ratio          float64 `json:"ratio"`
	Bonus          float64 `json:"bonus"`
	SeedingCount   int     `json:"seedingCount"`
	SeedingBytes   string  `json:"seedingBytes"`
	LeechingCount  int     `json:"leechingCount"`
	HitAndRunCount int     `json:"hitAndRunCount"`
	RoleID         *int    `json:"roleId"`
	LevelName      *string `json:"levelName"`
	IsWarned       bool    `json:"isWarned"`
	IsVIP          bool    `json:"isVip"`
	IsDonor        bool    `json:"isDonor"`
	LastLoginAt    *string `json:"lastLoginAt"`
	LastTrackerAt  *string `json:"lastTrackerAt"`
	SiteStatus     string  `json:"siteStatus"`
	SyncedAt       *string `json:"syncedAt"`
	BonusHourlyRate *float64 `json:"bonusHourlyRate"`
}

type createDTO struct {
	SiteCode    string  `json:"siteCode" binding:"required"`
	AccountName string  `json:"accountName" binding:"required,max=64"`
	APIKey      string  `json:"apiKey" binding:"required,min=8,max=256"`
	Remark      *string `json:"remark" binding:"omitempty,max=256"`
}

type updateDTO struct {
	AccountName *string `json:"accountName" binding:"omitempty,required,max=64"`
	Remark      *string `json:"remark" binding:"omitempty,max=256"`
	IsEnabled   *bool   `json:"isEnabled"`
	SyncEnabled *bool   `json:"syncEnabled"`
	APIKey      *string `json:"apiKey" binding:"omitempty,min=8,max=256"`
}

type accountRow struct {
	id, siteCode, siteName, accountName string
	remark, externalUserID, username, avatarURL *string
	joinedAt, lastSyncAt *time.Time
	isEnabled, syncEnabled bool
	status string
	createdAt, updatedAt time.Time
	hasStats bool
	// stats
	uploadBytes, downloadBytes, seedingBytes int64
	ratio, bonus float64
	seedingCount, leechingCount, hitAndRunCount int
	roleID *int
	levelName *string
	isWarned, isVIP, isDonor bool
	lastLoginAt, lastTrackerAt, syncedAt *time.Time
	siteStatus string
	bonusHourlyRate *float64
}

const accountSelect = `
SELECT a.id, s.code, s.name, a.accountName, a.remark, a.externalUserId,
       a.username, a.avatarUrl, a.joinedAt, a.isEnabled, a.syncEnabled,
       a.status, a.lastSyncAt, a.createdAt, a.updatedAt,
       (t.id IS NOT NULL) AS has_stats,
       COALESCE(t.uploadBytes,0), COALESCE(t.downloadBytes,0), COALESCE(t.seedingBytes,0),
       COALESCE(t.ratio,0), COALESCE(t.bonus,0),
       COALESCE(t.seedingCount,0), COALESCE(t.leechingCount,0), COALESCE(t.hitAndRunCount,0),
       t.roleId, t.levelName, COALESCE(t.isWarned,false), COALESCE(t.isVip,false), COALESCE(t.isDonor,false),
       t.lastLoginAt, t.lastTrackerAt, t.syncedAt,
       COALESCE(t.siteStatus,'UNKNOWN'), t.bonusHourlyRate
FROM TrackerAccount a
JOIN TrackerSite s ON s.id = a.siteId
LEFT JOIN TrackerStats t ON t.accountId = a.id`

func scanAccount(row interface{ Scan(dest ...interface{}) error }) (*accountRow, error) {
	var r accountRow
	err := row.Scan(&r.id, &r.siteCode, &r.siteName, &r.accountName, &r.remark, &r.externalUserID,
		&r.username, &r.avatarURL, &r.joinedAt, &r.isEnabled, &r.syncEnabled,
		&r.status, &r.lastSyncAt, &r.createdAt, &r.updatedAt,
		&r.hasStats,
		&r.uploadBytes, &r.downloadBytes, &r.seedingBytes,
		&r.ratio, &r.bonus,
		&r.seedingCount, &r.leechingCount, &r.hitAndRunCount,
		&r.roleID, &r.levelName, &r.isWarned, &r.isVIP, &r.isDonor,
		&r.lastLoginAt, &r.lastTrackerAt, &r.syncedAt,
		&r.siteStatus, &r.bonusHourlyRate)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func fmtTime(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.UTC().Format(time.RFC3339Nano)
	return &s
}

func (r *accountRow) toResponse() AccountResponse {
	stats := (*AccountStats)(nil)
	if r.hasStats {
		stats = &AccountStats{
			UploadBytes:    int64Str(r.uploadBytes),
			DownloadBytes:  int64Str(r.downloadBytes),
			Ratio:          round4(r.ratio),
			Bonus:          round2(r.bonus),
			SeedingCount:   r.seedingCount,
			SeedingBytes:   int64Str(r.seedingBytes),
			LeechingCount:  r.leechingCount,
			HitAndRunCount: r.hitAndRunCount,
			RoleID:         r.roleID,
			LevelName:      r.levelName,
			IsWarned:       r.isWarned,
			IsVIP:          r.isVIP,
			IsDonor:        r.isDonor,
			LastLoginAt:    fmtTime(r.lastLoginAt),
			LastTrackerAt:  fmtTime(r.lastTrackerAt),
			SiteStatus:     r.siteStatus,
			SyncedAt:       fmtTime(r.syncedAt),
			BonusHourlyRate: r.bonusHourlyRate,
		}
	}
	return AccountResponse{
		ID: r.id, SiteCode: r.siteCode, SiteName: r.siteName, AccountName: r.accountName,
		Remark: r.remark, ExternalUserID: r.externalUserID, Username: r.username,
		AvatarURL: r.avatarURL, JoinedAt: fmtTime(r.joinedAt), IsEnabled: r.isEnabled,
		SyncEnabled: r.syncEnabled, Status: r.status, LastSyncAt: fmtTime(r.lastSyncAt),
		CreatedAt: r.createdAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt: r.updatedAt.UTC().Format(time.RFC3339Nano),
		Stats: stats,
	}
}

func int64Str(n int64) string {
	return strings.TrimSpace(fmtInt(n))
}

func fmtInt(n int64) string {
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

func round4(f float64) float64 { return float64(int64(f*10000+0.5)) / 10000 }
func round2(f float64) float64 { return float64(int64(f*100+0.5)) / 100 }

// ─── Service 方法 ─────────────────────────────────────────────────────────

func (s *Service) Create(dto createDTO) (AccountResponse, error) {
	ctx := context.Background()
	siteID, siteEnabled, err := s.resolveSite(ctx, dto.SiteCode)
	if err != nil {
		return AccountResponse{}, err
	}
	_ = siteEnabled
	userID, err := s.resolveDefaultUser(ctx)
	if err != nil {
		return AccountResponse{}, err
	}

	enc, err := s.crypto.Encrypt(dto.APIKey)
	if err != nil {
		return AccountResponse{}, err
	}

	accountID := db.NewID()
	tx, err := s.pool.Begin()
	if err != nil {
		return AccountResponse{}, err
	}
	defer func() { _ = tx.Rollback() }()

	now := time.Now()
	_, err = tx.Exec(`
		INSERT INTO TrackerAccount(id,userId,siteId,accountName,remark,isEnabled,syncEnabled,status,createdAt,updatedAt)
		VALUES(?,?,?,?,?,true,true,'ACTIVE',?,?)`,
		accountID, userID, siteID, dto.AccountName, dto.Remark, now, now)
	if err != nil {
		return AccountResponse{}, err
	}
	_, err = tx.Exec(`
		INSERT INTO TrackerCredential(id,accountId,encryptedApiKey,iv,authTag,createdAt,updatedAt)
		VALUES(?,?,?,?,?,?,?)`,
		db.NewID(), accountID, enc.EncryptedAPIKey, enc.IV, enc.AuthTag, now, now)
	if err != nil {
		return AccountResponse{}, err
	}
	if err := tx.Commit(); err != nil {
		return AccountResponse{}, err
	}
	return s.GetOne(accountID)
}

func (s *Service) resolveSite(ctx context.Context, code string) (string, bool, error) {
	var id string
	var enabled bool
	err := s.pool.QueryRow(`SELECT id,isEnabled FROM TrackerSite WHERE code=?`, code).Scan(&id, &enabled)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", false, &apiErr{http.StatusBadRequest, "Site " + code + " is not registered. Run seed to initialize sites."}
		}
		return "", false, err
	}
	if !enabled {
		return "", false, &apiErr{http.StatusBadRequest, "Site " + code + " is currently disabled"}
	}
	if !s.registry.Has(code) {
		return "", false, &apiErr{http.StatusBadRequest, "No provider implementation registered for site " + code}
	}
	return id, enabled, nil
}

func (s *Service) resolveDefaultUser(ctx context.Context) (string, error) {
	var id string
	err := s.pool.QueryRow(`SELECT id FROM User ORDER BY createdAt ASC LIMIT 1`).Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", &apiErr{http.StatusBadRequest, "No system user found. Run seed to initialize the default user."}
		}
		return "", err
	}
	return id, nil
}

func (s *Service) ListAll() ([]AccountResponse, error) {
	rows, err := s.pool.Query(accountSelect + " ORDER BY a.createdAt DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AccountResponse
	for rows.Next() {
		r, err := scanAccount(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r.toResponse())
	}
	return out, nil
}

func (s *Service) GetOne(id string) (AccountResponse, error) {
	r, err := scanAccount(s.pool.QueryRow(accountSelect+" WHERE a.id=?", id))
	if err != nil {
		if err == sql.ErrNoRows {
			return AccountResponse{}, &apiErr{http.StatusNotFound, "Tracker account " + id + " not found"}
		}
		return AccountResponse{}, err
	}
	return r.toResponse(), nil
}

func (s *Service) Update(id string, dto updateDTO) (AccountResponse, error) {
	if _, err := s.GetOne(id); err != nil {
		return AccountResponse{}, err
	}

	// 轮换 API Key（可选）—— 重新加密覆盖旧凭据
	if dto.APIKey != nil && *dto.APIKey != "" {
		enc, err := s.crypto.Encrypt(*dto.APIKey)
		if err != nil {
			return AccountResponse{}, err
		}
		_, err = s.pool.Exec(`
			INSERT INTO TrackerCredential(id,accountId,encryptedApiKey,iv,authTag,createdAt,updatedAt)
			VALUES(?,?,?,?,?,?,?)
			ON DUPLICATE KEY UPDATE encryptedApiKey=VALUES(encryptedApiKey),iv=VALUES(iv),authTag=VALUES(authTag),updatedAt=NOW(3)`,
			db.NewID(), id, enc.EncryptedAPIKey, enc.IV, enc.AuthTag, time.Now())
		if err != nil {
			return AccountResponse{}, err
		}
	}

	sets := []string{"updatedAt=NOW(3)"}
	args := []interface{}{}
	if dto.AccountName != nil && *dto.AccountName != "" {
		args = append(args, *dto.AccountName)
		sets = append(sets, "accountName=?")
	}
	if dto.Remark != nil {
		args = append(args, *dto.Remark)
		sets = append(sets, "remark=?")
	}
	if dto.IsEnabled != nil {
		args = append(args, *dto.IsEnabled)
		sets = append(sets, "isEnabled=?")
	}
	if dto.SyncEnabled != nil {
		args = append(args, *dto.SyncEnabled)
		sets = append(sets, "syncEnabled=?")
	}
	args = append(args, id)
	q := "UPDATE TrackerAccount SET " + joinStrings(sets, ", ") + " WHERE id=?"
	_, err := s.pool.Exec(q, args...)
	if err != nil {
		return AccountResponse{}, err
	}
	return s.GetOne(id)
}

func (s *Service) Remove(id string) (map[string]string, error) {
	if _, err := s.GetOne(id); err != nil {
		return nil, err
	}
	_, err := s.pool.Exec(`DELETE FROM TrackerAccount WHERE id=?`, id)
	if err != nil {
		return nil, err
	}
	return map[string]string{"id": id}, nil
}

func (s *Service) TestConnection(id string) (domain.ConnectionResult, error) {
	var siteCode string
	err := s.pool.QueryRow(`SELECT s.code FROM TrackerAccount a JOIN TrackerSite s ON s.id=a.siteId WHERE a.id=?`, id).Scan(&siteCode)
	if err != nil {
		if err == sql.ErrNoRows {
			return domain.ConnectionResult{}, &apiErr{http.StatusNotFound, "Tracker account " + id + " not found"}
		}
		return domain.ConnectionResult{}, err
	}
	provider, err := s.registry.Get(siteCode)
	if err != nil {
		return domain.ConnectionResult{}, err
	}
	return provider.TestConnection(id)
}

// ─── handlers ─────────────────────────────────────────────────────────────

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
	log.Printf("[http] 500 on %s %s: %v", c.Request.Method, c.Request.URL.Path, err)
	c.JSON(http.StatusInternalServerError, gin.H{"statusCode": 500, "message": "服务器内部错误"})
}

func joinStrings(parts []string, sep string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += sep
		}
		out += p
	}
	return out
}

func CreateHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var dto createDTO
		if err := c.ShouldBindJSON(&dto); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"statusCode": 400, "message": "参数不合法：" + err.Error()})
			return
		}
		resp, err := svc.Create(dto)
		if err != nil {
			writeErr(c, err)
			return
		}
		c.JSON(http.StatusCreated, resp)
	}
}

func ListHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		resp, err := svc.ListAll()
		if err != nil {
			writeErr(c, err)
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}

func GetHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		resp, err := svc.GetOne(c.Param("id"))
		if err != nil {
			writeErr(c, err)
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}

func UpdateHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var dto updateDTO
		if err := c.ShouldBindJSON(&dto); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"statusCode": 400, "message": "参数不合法：" + err.Error()})
			return
		}
		resp, err := svc.Update(c.Param("id"), dto)
		if err != nil {
			writeErr(c, err)
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}

func DeleteHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		resp, err := svc.Remove(c.Param("id"))
		if err != nil {
			writeErr(c, err)
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}

func TestConnectionHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		resp, err := svc.TestConnection(c.Param("id"))
		if err != nil {
			writeErr(c, err)
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}
