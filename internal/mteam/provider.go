package mteam

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/1443205008/ptmanager-go/internal/config"
	"github.com/1443205008/ptmanager-go/internal/cryptoutil"
	"github.com/1443205008/ptmanager-go/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Provider M-Team TrackerProvider 实现。
// 真实 API 行为（2026-08 验证）：
// - /member/profile 不传 uid 即返回当前 API Key 对应用户资料
// - seedingStats / getCrimeRecords 当前 API Key 无权 → 做种数用 getUserTorrentList total
type Provider struct {
	client *Client
	pool   *pgxpool.Pool
	crypto *cryptoutil.CredentialCrypto
}

func NewProvider(cfg *config.Config, pool *pgxpool.Pool, crypto *cryptoutil.CredentialCrypto) *Provider {
	return &Provider{client: NewClient(cfg), pool: pool, crypto: crypto}
}

func (p *Provider) TestConnection(accountID string) (domain.ConnectionResult, error) {
	start := time.Now()
	apiKey, err := p.getDecryptedAPIKey(accountID)
	if err != nil {
		return p.connErr(err, time.Since(start).Milliseconds()), nil
	}
	member, err := p.fetchSelfProfile(apiKey)
	if err != nil {
		return p.connErr(err, time.Since(start).Milliseconds()), nil
	}
	_, _ = p.syncExternalUserID(accountID, member.ID)
	return domain.ConnectionResult{
		Success:   true,
		LatencyMs: time.Since(start).Milliseconds(),
		Profile: &domain.MiniProfile{
			ExternalUserID: member.ID,
			Username:       member.Username,
		},
	}, nil
}

func (p *Provider) connErr(err error, latencyMs int64) domain.ConnectionResult {
	code := domain.ErrUnknown
	var msg string
	if mte, ok := err.(*MTeamError); ok {
		code = mte.Code
		msg = mte.Message
	} else {
		msg = err.Error()
	}
	return domain.ConnectionResult{
		Success:   false,
		LatencyMs: latencyMs,
		Error: &domain.TrackerErrorInfo{Code: code, Message: msg},
	}
}

func (p *Provider) GetProfile(accountID string) (domain.TrackerProfile, error) {
	apiKey, err := p.getDecryptedAPIKey(accountID)
	if err != nil {
		return domain.TrackerProfile{}, err
	}
	member, err := p.fetchSelfProfile(apiKey)
	if err != nil {
		return domain.TrackerProfile{}, err
	}
	_, _ = p.syncExternalUserID(accountID, member.ID)
	return ToProfile(member), nil
}

func (p *Provider) GetStats(accountID string) (domain.TrackerStats, error) {
	apiKey, err := p.getDecryptedAPIKey(accountID)
	if err != nil {
		return domain.TrackerStats{}, err
	}
	member, err := p.fetchSelfProfile(apiKey)
	if err != nil {
		return domain.TrackerStats{}, err
	}
	uid, err := p.syncExternalUserID(accountID, member.ID)
	if err != nil {
		return domain.TrackerStats{}, err
	}

	seedingCount := p.fetchTorrentCount(apiKey, uid, "SEEDING")
	time.Sleep(2 * time.Second) // 限速冷却
	leechingCount := p.fetchTorrentCount(apiKey, uid, "LEECHING")

	bonusHourlyRate := p.fetchBonusHourlyRate(apiKey, uid)

	return ToStats(member, seedingCount, leechingCount, 0, bonusHourlyRate), nil
}

func (p *Provider) GetTorrents(accountID string) ([]domain.TrackerTorrent, error) {
	seeding, err := p.GetSeedingTorrents(accountID)
	if err != nil {
		return nil, err
	}
	time.Sleep(3 * time.Second) // 连续请求冷却
	leeching, err := p.GetLeechingTorrents(accountID)
	if err != nil {
		return nil, err
	}
	return append(seeding, leeching...), nil
}

func (p *Provider) GetSeedingTorrents(accountID string) ([]domain.TrackerTorrent, error) {
	return p.fetchAllTorrents(accountID, "SEEDING")
}

func (p *Provider) GetLeechingTorrents(accountID string) ([]domain.TrackerTorrent, error) {
	return p.fetchAllTorrents(accountID, "LEECHING")
}

func (p *Provider) GetBonus(accountID string) (domain.BonusStats, error) {
	apiKey, err := p.getDecryptedAPIKey(accountID)
	if err != nil {
		return domain.BonusStats{}, err
	}
	member, err := p.fetchSelfProfile(apiKey)
	if err != nil {
		return domain.BonusStats{}, err
	}
	return domain.BonusStats{
		Current:   toFloat(member.MemberCount.Bonus),
		UpdatedAt: time.Now(),
	}, nil
}

func (p *Provider) GetHitAndRuns(accountID string) ([]domain.HitAndRun, error) {
	// getCrimeRecords 当前 API Key 无权（code 1 "無許可權"）→ 降级为空
	return []domain.HitAndRun{}, nil
}

func (p *Provider) GetMessages(accountID string) (domain.MessageStats, error) {
	// M-Team 消息 API 待补充；占位
	return domain.MessageStats{UnreadCount: 0, CheckedAt: time.Now()}, nil
}

func (p *Provider) GetSiteStatus(accountID string) (domain.TrackerSiteStatusResult, error) {
	start := time.Now()
	apiKey, err := p.getDecryptedAPIKey(accountID)
	if err != nil {
		return domain.TrackerSiteStatusResult{Status: domain.SiteOffline, CheckedAt: start, Message: err.Error()}, nil
	}
	if _, err := p.fetchSelfProfile(apiKey); err != nil {
		status := domain.SiteOffline
		if mte, ok := err.(*MTeamError); ok && mte.Code == domain.ErrRateLimited {
			status = domain.SiteDegraded
		}
		return domain.TrackerSiteStatusResult{Status: status, LatencyMs: time.Since(start).Milliseconds(), CheckedAt: time.Now(), Message: err.Error()}, nil
	}
	return domain.TrackerSiteStatusResult{Status: domain.SiteHealthy, LatencyMs: time.Since(start).Milliseconds(), CheckedAt: time.Now()}, nil
}

// SearchTorrents 站内种子搜索
func (p *Provider) SearchTorrents(accountID string, params SearchRequest) (*PageResult, error) {
	apiKey, err := p.getDecryptedAPIKey(accountID)
	if err != nil {
		return nil, err
	}
	var result PageResult
	if err := p.client.Request(EndpointTorrentSearch, apiKey, buildSearchRequest(params), &result, 1); err != nil {
		return nil, err
	}
	return &result, nil
}

// GenDlToken 生成种子下载 Token（要求 form-urlencoded）
func (p *Provider) GenDlToken(accountID, torrentID string) (string, error) {
	apiKey, err := p.getDecryptedAPIKey(accountID)
	if err != nil {
		return "", err
	}
	id, err := strconv.Atoi(torrentID)
	if err != nil {
		return "", fmt.Errorf("invalid torrentId: %s", torrentID)
	}
	var token string
	err = p.client.RequestForm(EndpointGenDlToken, apiKey, map[string]string{"id": strconv.Itoa(id)}, &token)
	if err != nil {
		return "", err
	}
	return token, nil
}

// GetTeamList 制作组列表
func (p *Provider) GetTeamList(accountID string) ([]Team, error) {
	apiKey, err := p.getDecryptedAPIKey(accountID)
	if err != nil {
		return nil, err
	}
	var teams []Team
	if err := p.client.Request(EndpointTeamList, apiKey, map[string]interface{}{}, &teams, 1); err != nil {
		return nil, err
	}
	if teams == nil {
		teams = []Team{}
	}
	return teams, nil
}

// ─── 私有辅助 ─────────────────────────────────────────────────────────────

// buildSearchRequest 只发送允许的字段，避免空值/非法枚举触发「請求參數錯誤」
func buildSearchRequest(params SearchRequest) map[string]interface{} {
	modes := map[string]bool{"normal": true, "adult": true, "movie": true, "music": true, "tvshow": true, "anime": true, "waterfall": true, "rss": true, "rankings": true, "all": true}
	discounts := map[string]bool{"NORMAL": true, "PERCENT_70": true, "PERCENT_50": true, "FREE": true, "_2X_FREE": true, "_2X": true, "_2X_PERCENT_50": true}
	sortFields := map[string]bool{"CREATED_DATE": true, "SIZE": true, "SEEDERS": true, "LEECHERS": true, "TIMES_COMPLETED": true, "NAME": true}

	pageNumber := params.PageNumber
	if pageNumber < 1 {
		pageNumber = 1
	}
	if pageNumber > 1000 {
		pageNumber = 1000
	}
	pageSize := params.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}

	body := map[string]interface{}{
		"pageNumber": pageNumber,
		"pageSize":   pageSize,
		"mode":       "normal",
	}
	if params.Mode != "" && modes[params.Mode] {
		body["mode"] = params.Mode
	}
	if params.Keyword != "" {
		kw := trimSpaceMax(params.Keyword, 100)
		if kw != "" {
			body["keyword"] = kw
		}
	}
	if params.Discount != "" && discounts[params.Discount] {
		body["discount"] = params.Discount
	}
	if params.SortField != "" && sortFields[params.SortField] {
		body["sortField"] = params.SortField
	}
	if params.SortDirection == "ASC" || params.SortDirection == "DESC" {
		body["sortDirection"] = params.SortDirection
	}
	if len(params.Teams) > 0 {
		body["teams"] = params.Teams
	}
	return body
}

func trimSpaceMax(s string, max int) string {
	// trim 空格并截断
	out := ""
	for _, r := range s {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			continue
		}
		out += string(r)
		if len(out) >= max {
			break
		}
	}
	return out
}

func (p *Provider) fetchSelfProfile(apiKey string) (*Member, error) {
	var member Member
	// ⚠️ M-Team 所有端点均为 POST，传 {} 空 body 以避免 "請求參數錯誤"
	if err := p.client.Request(EndpointMemberProfile, apiKey, map[string]interface{}{}, &member, p.client.maxRetries); err != nil {
		return nil, err
	}
	return &member, nil
}

// fetchBonusHourlyRate 从 tracker/mybonus 取 finalBs（时魔值 bonus/h），失败时返回 nil
func (p *Provider) fetchBonusHourlyRate(apiKey string, uid int) *float64 {
	var resp MyBonusResponse
	err := p.client.Request(EndpointMyBonus, apiKey, map[string]interface{}{"uid": uid}, &resp, p.client.maxRetries)
	if err != nil {
		log.Printf("[mteam] fetchBonusHourlyRate failed: %v → nil", err)
		return nil
	}
	if resp.FormulaParams == nil || resp.FormulaParams.FinalBs == nil {
		return nil
	}
	rate := toFloat(*resp.FormulaParams.FinalBs)
	if rate <= 0 {
		return nil
	}
	return &rate
}

// fetchTorrentCount 获取指定查询类型的种子总数（pageSize=1，只读 total）
func (p *Provider) fetchTorrentCount(apiKey string, uid int, queryType string) int {
	var result PageResult
	body := map[string]interface{}{
		"userid":    uid,
		"type":      queryType,
		"pageNumber": 1,
		"pageSize":  1,
	}
	if err := p.client.Request(EndpointMemberTorrentList, apiKey, body, &result, p.client.maxRetries); err != nil {
		log.Printf("[mteam] fetchTorrentCount(%s) failed: %v → 0", queryType, err)
		return 0
	}
	return toInt(result.Total.String())
}

// fetchAllTorrents 分页拉取全部种子（最大 100 页）
func (p *Provider) fetchAllTorrents(accountID, queryType string) ([]domain.TrackerTorrent, error) {
	apiKey, err := p.getDecryptedAPIKey(accountID)
	if err != nil {
		return nil, err
	}
	uid, err := p.resolveUID(accountID, apiKey)
	if err != nil {
		return nil, err
	}

	var all []domain.TrackerTorrent
	pageSize := 100
	pageNumber := 1
	for pageNumber <= 100 {
		var result PageResult
		body := map[string]interface{}{
			"userid":     uid,
			"type":       queryType,
			"pageNumber": pageNumber,
			"pageSize":   pageSize,
		}
		if err := p.client.Request(EndpointMemberTorrentList, apiKey, body, &result, p.client.maxRetries); err != nil {
			return nil, err
		}
		var items []TorrentItem
		if len(result.Data) > 0 {
			if err := json.Unmarshal(result.Data, &items); err != nil {
				return nil, fmt.Errorf("parse torrent list: %w", err)
			}
		}
		for i := range items {
			if t := ToTorrent(&items[i], queryType); t != nil {
				all = append(all, *t)
			}
		}
		totalPages := toInt(result.TotalPages.String())
		if totalPages < 1 {
			totalPages = 1
		}
		if pageNumber >= totalPages || len(items) == 0 {
			break
		}
		pageNumber++
	}
	return all, nil
}

func (p *Provider) getDecryptedAPIKey(accountID string) (string, error) {
	var enc, iv, tag string
	err := p.pool.QueryRow(context.Background(),
		`SELECT "encryptedApiKey", "iv", "authTag" FROM "TrackerCredential" WHERE "accountId"=$1`, accountID,
	).Scan(&enc, &iv, &tag)
	if err != nil {
		return "", fmt.Errorf("credential not found for account %s: %w", accountID, err)
	}
	return p.crypto.Decrypt(enc, iv, tag)
}

// resolveUID DB 已有则直接用；否则拉 profile 引导获取并回写
func (p *Provider) resolveUID(accountID, apiKey string) (int, error) {
	var externalID *string
	err := p.pool.QueryRow(context.Background(),
		`SELECT "externalUserId" FROM "TrackerAccount" WHERE "id"=$1`, accountID).Scan(&externalID)
	if err != nil {
		return 0, err
	}
	if externalID != nil && *externalID != "" {
		if uid, err := strconv.Atoi(*externalID); err == nil {
			return uid, nil
		}
	}
	member, err := p.fetchSelfProfile(apiKey)
	if err != nil {
		return 0, err
	}
	return p.syncExternalUserID(accountID, member.ID)
}

// syncExternalUserID 将 profile 的 member.id 回写 DB（若变化），返回数字 uid
func (p *Provider) syncExternalUserID(accountID, externalID string) (int, error) {
	var current *string
	err := p.pool.QueryRow(context.Background(),
		`SELECT "externalUserId" FROM "TrackerAccount" WHERE "id"=$1`, accountID).Scan(&current)
	if err != nil {
		return 0, err
	}
	if current == nil || *current != externalID {
		_, err = p.pool.Exec(context.Background(),
			`UPDATE "TrackerAccount" SET "externalUserId"=$1, "updatedAt"=NOW() WHERE "id"=$2`, externalID, accountID)
		if err != nil {
			return 0, err
		}
	}
	uid, err := strconv.Atoi(externalID)
	if err != nil {
		return 0, fmt.Errorf("non-numeric external id %q: %w", externalID, err)
	}
	return uid, nil
}
