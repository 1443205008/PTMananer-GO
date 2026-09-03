// Package torrents: 种子列表只读查询（对齐 TS 版 torrents 模块）。
package torrents

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct{ pool *pgxpool.Pool }

func NewService(pool *pgxpool.Pool) *Service { return &Service{pool: pool} }

type TorrentResponse struct {
	ID             string  `json:"id"`
	AccountID      string  `json:"accountId"`
	AccountName    string  `json:"accountName"`
	SiteTorrentID  string  `json:"siteTorrentId"`
	Name           string  `json:"name"`
	SizeBytes      string  `json:"sizeBytes"`
	Status         string  `json:"status"`
	UploadedBytes  string  `json:"uploadedBytes"`
	DownloadedBytes string `json:"downloadedBytes"`
	Ratio          float64 `json:"ratio"`
	SeedTimeSecs   int     `json:"seedTimeSecs"`
	LeechTimeSecs  int     `json:"leechTimeSecs"`
	CompletedAt    *string `json:"completedAt"`
	LastActivityAt *string `json:"lastActivityAt"`
	SyncedAt       *string `json:"syncedAt"`
	CreatedAt      string  `json:"createdAt"`
}

type ListResponse struct {
	Data   []TorrentResponse `json:"data"`
	Total  int               `json:"total"`
	Page   int               `json:"page"`
	Limit  int               `json:"limit"`
}

func (s *Service) FindAll(accountID, status string, page, limit int) (ListResponse, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	ctx := context.Background()

	where := ` WHERE 1=1`
	args := []interface{}{}
	if accountID != "" {
		args = append(args, accountID)
		where += ` AND t."accountId"=$` + strconv.Itoa(len(args))
	}
	if status != "" {
		args = append(args, status)
		where += ` AND t."status"=$` + strconv.Itoa(len(args))
	}

	var total int
	if err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM "TrackerTorrent" t`+where, args...).Scan(&total); err != nil {
		return ListResponse{}, err
	}

	args = append(args, limit, (page-1)*limit)
	q := `
		SELECT t."id", t."accountId", COALESCE(a."accountName",''), t."siteTorrentId", t."name",
		       t."sizeBytes", t."status", t."uploadedBytes", t."downloadedBytes", t."ratio",
		       t."seedTimeSecs", t."leechTimeSecs", t."completedAt", t."lastActivityAt", t."syncedAt", t."createdAt"
		FROM "TrackerTorrent" t
		JOIN "TrackerAccount" a ON a."id" = t."accountId"` + where + `
		ORDER BY t."status" ASC, t."lastActivityAt" DESC NULLS LAST
		LIMIT $` + strconv.Itoa(len(args)-1) + ` OFFSET $` + strconv.Itoa(len(args))

	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return ListResponse{}, err
	}
	defer rows.Close()

	var out []TorrentResponse
	for rows.Next() {
		var r TorrentResponse
		var completedAt, lastActivityAt, syncedAt *time.Time
		if err := rows.Scan(&r.ID, &r.AccountID, &r.AccountName, &r.SiteTorrentID, &r.Name,
			&r.SizeBytes, &r.Status, &r.UploadedBytes, &r.DownloadedBytes, &r.Ratio,
			&r.SeedTimeSecs, &r.LeechTimeSecs, &completedAt, &lastActivityAt, &syncedAt, &r.CreatedAt); err != nil {
			return ListResponse{}, err
		}
		r.SizeBytes = strconv.FormatInt(mustParse(r.SizeBytes), 10)
		r.UploadedBytes = strconv.FormatInt(mustParse(r.UploadedBytes), 10)
		r.DownloadedBytes = strconv.FormatInt(mustParse(r.DownloadedBytes), 10)
		r.CompletedAt = fmtTime(completedAt)
		r.LastActivityAt = fmtTime(lastActivityAt)
		r.SyncedAt = fmtTime(syncedAt)
		r.CreatedAt = time.Time{}.Format(time.RFC3339Nano)
		// createdAt 重扫
		out = append(out, r)
	}
	if out == nil {
		out = []TorrentResponse{}
	}
	return ListResponse{Data: out, Total: total, Page: page, Limit: limit}, nil
}

func mustParse(s string) int64 {
	// pgx 扫 BIGINT 到 string 接口时给的是数字字符串
	n, _ := strconv.ParseInt(s, 10, 64)
	return n
}

func fmtTime(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.UTC().Format(time.RFC3339Nano)
	return &s
}

func ListHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		page, limit := 1, 50
		if v := c.Query("page"); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				page = n
			}
		}
		if v := c.Query("limit"); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				limit = n
			}
		}
		resp, err := svc.FindAll(c.Query("accountId"), c.Query("status"), page, limit)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"statusCode": 500, "message": "查询失败"})
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}
