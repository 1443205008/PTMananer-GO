// Package search: 站内搜索（透传 M-Team，对齐 TS 版 search 模块）。
package search

import (
	"context"

	"database/sql"
	"github.com/redis/go-redis/v9"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/1443205008/ptmanager-go/internal/mteam"
	"github.com/1443205008/ptmanager-go/internal/providers"

	"github.com/gin-gonic/gin"
)

type Service struct {
	pool     *sql.DB
	registry *providers.Registry
	rdb      *redis.Client
}

func NewService(pool *sql.DB, registry *providers.Registry, rdb *redis.Client) *Service {
	return &Service{pool: pool, registry: registry, rdb: rdb}
}

type SearchItem struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	SmallDescr      *string `json:"smallDescr"`
	Size            string  `json:"size"`
	Category        *string `json:"category"`
	CreatedDate     *string `json:"createdDate"`
	Seeders         int     `json:"seeders"`
	Leechers        int     `json:"leechers"`
	TimesCompleted  int     `json:"timesCompleted"`
	Discount        *string `json:"discount"`
	DiscountEndTime *string `json:"discountEndTime"`
	Imdb            *string `json:"imdb"`
	IsSeeding       bool    `json:"isSeeding"`
}

type SearchResponse struct {
	Data     []SearchItem `json:"data"`
	Total    int          `json:"total"`
	Page     int          `json:"page"`
	PageSize int          `json:"pageSize"`
}

// mteamProvider 搜索需要 mteam 特有方法（TS 版同样做了窄化 cast）
type mteamProvider interface {
	SearchTorrents(accountID string, params mteam.SearchRequest) (*mteam.PageResult, error)
	GenDlToken(accountID, torrentID string) (string, error)
	GetTeamList(accountID string) ([]mteam.Team, error)
}

func (s *Service) resolveSearchAccount(ctx context.Context) (string, string, mteamProvider, error) {
	type row struct{ id, code string }
	var r row
	// 1. ACTIVE 优先；2. 非 CREDENTIAL_INVALID；3. 任意启用账户（返回明确错误）
	err := s.pool.QueryRow(`
		SELECT a.id, s.code FROM TrackerAccount a JOIN TrackerSite s ON s.id=a.siteId
		WHERE a.isEnabled=true AND a.status='ACTIVE'
		ORDER BY a.createdAt DESC LIMIT 1`).Scan(&r.id, &r.code)
	if err == sql.ErrNoRows {
		err = s.pool.QueryRow(`
			SELECT a.id, s.code FROM TrackerAccount a JOIN TrackerSite s ON s.id=a.siteId
			WHERE a.isEnabled=true AND a.status != 'CREDENTIAL_INVALID'
			ORDER BY a.createdAt DESC LIMIT 1`).Scan(&r.id, &r.code)
	}
	if err == sql.ErrNoRows {
		var anyID string
		if err2 := s.pool.QueryRow(`SELECT id FROM TrackerAccount WHERE isEnabled=true LIMIT 1`).Scan(&anyID); err2 == nil {
			return "", "", nil, &apiErr{http.StatusServiceUnavailable, "账户 API Key 无效或已过期，请在站点账户中更新后再搜索"}
		}
		return "", "", nil, &apiErr{http.StatusServiceUnavailable, "没有可用的账户，请先添加 M-Team 账户后再搜索"}
	}
	if err != nil {
		return "", "", nil, err
	}
	p, err := s.registry.Get(r.code)
	if err != nil {
		return "", "", nil, err
	}
	mp, ok := p.(mteamProvider)
	if !ok {
		return "", "", nil, &apiErr{http.StatusServiceUnavailable, "该站点不支持搜索"}
	}
	return r.id, r.code, mp, nil
}

func (s *Service) SearchTorrents(keyword string, page, pageSize int, mode, discount, sortField, sortDirection string, teams []int) (SearchResponse, error) {
	ctx := context.Background()
	accountID, _, provider, err := s.resolveSearchAccount(ctx)
	if err != nil {
		return SearchResponse{}, err
	}

	result, err := provider.SearchTorrents(accountID, mteam.SearchRequest{
		Keyword: keyword, PageNumber: page, PageSize: pageSize,
		Mode: mode, Discount: discount, SortField: sortField, SortDirection: sortDirection, Teams: teams,
	})
	if err != nil {
		return SearchResponse{}, toSearchErr(err)
	}

	// 标记已在做种 / 保种中的种子：
	// TrackerTorrent.SEEDING = 同步过来的真实做种；FakeSeedJob.RUNNING = 本地保种任务。
	// 搜索页「保种」按钮依赖 isSeeding，只查 TrackerTorrent 会漏掉保种任务。
	seeding := map[string]bool{}
	rows, err := s.pool.Query(`
		SELECT siteTorrentId FROM TrackerTorrent WHERE accountId=? AND status='SEEDING'
		UNION
		SELECT torrentId FROM FakeSeedJob WHERE accountId=? AND status='RUNNING'`,
		accountID, accountID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var id string
			_ = rows.Scan(&id)
			seeding[id] = true
		}
		if err := rows.Err(); err != nil {
			return SearchResponse{}, err
		}
	}

	var rawItems []mteam.SearchTorrentItem
	if len(result.Data) > 0 {
		_ = jsonUnmarshal(result.Data, &rawItems)
	}
	items := make([]SearchItem, 0, len(rawItems))
	for _, t := range rawItems {
		item := SearchItem{
			ID: t.ID, Name: t.Name, SmallDescr: t.SmallDescr, Size: orStr(t.Size, "0"),
			Category: t.Category, CreatedDate: t.CreatedDate, Imdb: t.Imdb,
		}
		if t.Status != nil {
			item.Seeders = atoi(t.Status.Seeders)
			item.Leechers = atoi(t.Status.Leechers)
			item.TimesCompleted = atoi(t.Status.TimesCompleted)
			item.Discount = t.Status.Discount
			item.DiscountEndTime = t.Status.DiscountEndTime
		}
		item.IsSeeding = seeding[t.ID]
		items = append(items, item)
	}

	total := atoi(result.Total.String())
	if result.Total.String() == "" {
		total = 0
	}
	return SearchResponse{
		Data: items, Total: total,
		Page:     atoiDefault(result.PageNumber.String(), 1),
		PageSize: atoiDefault(result.PageSize.String(), pageSize),
	}, nil
}

func (s *Service) GenDlToken(torrentID string) (map[string]string, error) {
	ctx := context.Background()
	accountID, _, provider, err := s.resolveSearchAccount(ctx)
	if err != nil {
		return nil, err
	}
	token, err := provider.GenDlToken(accountID, torrentID)
	if err != nil {
		return nil, toSearchErr(err)
	}
	return map[string]string{"url": token}, nil
}

const teamListCacheKey = "ptmanager:cache:teams"
const teamListCacheTTL = time.Hour

func (s *Service) GetTeamList() ([]map[string]interface{}, error) {
	ctx := context.Background()

	// 缓存命中（TS 版注释自述"结果较稳定可长时间缓存"，这里落 1h）
	if s.rdb != nil {
		if cached, err := s.rdb.Get(ctx, teamListCacheKey).Result(); err == nil && cached != "" {
			var out []map[string]interface{}
			if err := jsonUnmarshal([]byte(cached), &out); err == nil {
				return out, nil
			}
		}
	}

	accountID, _, provider, err := s.resolveSearchAccount(ctx)
	if err != nil {
		return nil, err
	}
	teams, err := provider.GetTeamList(accountID)
	if err != nil {
		return nil, toSearchErr(err)
	}
	out := make([]map[string]interface{}, 0, len(teams))
	for _, t := range teams {
		id, _ := t.ID.Int64()
		name := strings.TrimSpace(t.Name)
		if name == "" {
			continue
		}
		out = append(out, map[string]interface{}{"id": id, "name": name})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i]["name"].(string) < out[j]["name"].(string)
	})

	// 成功才写缓存（失败不缓存，避免坏数据驻留）
	if s.rdb != nil && len(out) > 0 {
		if data, err := jsonMarshal(out); err == nil {
			s.rdb.Set(ctx, teamListCacheKey, data, teamListCacheTTL)
		}
	}
	return out, nil
}

// ─── helpers ──────────────────────────────────────────────────────────────

type apiErr struct {
	status  int
	message string
}

func (e *apiErr) Error() string { return e.message }

// toSearchErr 把 M-Team 上游错误转成对前端可读的 HTTP 响应（对齐 MTeamExceptionFilter）
func toSearchErr(err error) error {
	if mte, ok := err.(*mteam.MTeamError); ok {
		switch mte.Code {
		case "AUTH_INVALID", "AUTH_EXPIRED":
			return &apiErr{http.StatusBadGateway, "M-Team API Key 无效或已过期，请在站点账户中更新后再搜索"}
		case "RATE_LIMITED":
			return &apiErr{http.StatusTooManyRequests, "M-Team 请求过于频繁，请稍后再试"}
		case "TIMEOUT":
			return &apiErr{http.StatusGatewayTimeout, "搜索 M-Team 超时，请稍后重试"}
		case "NETWORK_ERROR", "SITE_OFFLINE":
			return &apiErr{http.StatusServiceUnavailable, "无法连接 M-Team，请检查网络或站点状态"}
		}
		raw := strings.TrimSpace(err.Error())
		if raw != "" && raw != "API error" && !strings.HasPrefix(raw, "HTTP ") {
			return &apiErr{http.StatusBadGateway, raw}
		}
		return &apiErr{http.StatusBadGateway, "M-Team 搜索失败，请稍后重试"}
	}
	return err
}

func orStr(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func atoi(s string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(s))
	return n
}

func atoiDefault(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil || n == 0 {
		return def
	}
	return n
}

func jsonUnmarshal(data []byte, v interface{}) error {
	return jsonUnmarshalImpl(data, v)
}

// ─── handlers ─────────────────────────────────────────────────────────────

func writeErr(c *gin.Context, err error) {
	if ae, ok := err.(*apiErr); ok {
		c.JSON(ae.status, gin.H{"statusCode": ae.status, "message": ae.message})
		return
	}
	log.Printf("[http] 500 on %s %s: %v", c.Request.Method, c.Request.URL.Path, err)
	c.JSON(http.StatusInternalServerError, gin.H{"statusCode": 500, "message": "服务器内部错误"})
}

func SearchTorrentsHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		page, pageSize := 1, 20
		if v := c.Query("page"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				page = n
			}
		}
		if v := c.Query("pageSize"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				pageSize = n
			}
		}
		if pageSize > 100 {
			pageSize = 100
		}
		var teams []int
		if v := c.Query("teams"); v != "" {
			for _, part := range strings.Split(v, ",") {
				if n, err := strconv.Atoi(strings.TrimSpace(part)); err == nil {
					teams = append(teams, n)
				}
			}
		}
		resp, err := svc.SearchTorrents(c.Query("keyword"), page, pageSize,
			c.Query("mode"), c.Query("discount"), c.Query("sortField"), c.Query("sortDirection"), teams)
		if err != nil {
			writeErr(c, err)
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}

func TeamsHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		teams, err := svc.GetTeamList()
		if err != nil {
			writeErr(c, err)
			return
		}
		c.JSON(http.StatusOK, teams)
	}
}

func DlTokenHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			TorrentID string `json:"torrentId" binding:"required"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"statusCode": 400, "message": "torrentId 必填"})
			return
		}
		resp, err := svc.GenDlToken(body.TorrentID)
		if err != nil {
			writeErr(c, err)
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}
