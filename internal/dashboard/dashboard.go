// Package dashboard: 仪表盘聚合（对齐 TS 版 dashboard 模块）。
package dashboard

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"github.com/1443205008/ptmanager-go/internal/accounts"

	"github.com/gin-gonic/gin"
)

type Service struct {
	pool     *sql.DB
	accounts *accounts.Service
}

func NewService(pool *sql.DB, acc *accounts.Service) *Service {
	return &Service{pool: pool, accounts: acc}
}

type Summary struct {
	AccountCount    int     `json:"accountCount"`
	SiteCount        int     `json:"siteCount"`
	ActiveCount      int     `json:"activeCount"`
	ProblemCount     int     `json:"problemCount"`
	TotalUploadBytes string  `json:"totalUploadBytes"`
	TotalDownloadBytes string `json:"totalDownloadBytes"`
	OverallRatio     float64 `json:"overallRatio"`
	TotalBonus       float64 `json:"totalBonus"`
	TotalSeedingCount int    `json:"totalSeedingCount"`
	TotalSeedingBytes string  `json:"totalSeedingBytes"`
	TotalHitAndRun   int     `json:"totalHitAndRun"`
	LastSyncAt       *string `json:"lastSyncAt"`
}

type TrendPoint struct {
	Date          string `json:"date"`
	UploadBytes   string `json:"uploadBytes"`
	DownloadBytes string `json:"downloadBytes"`
	Bonus         float64 `json:"bonus"`
	SeedingCount  int    `json:"seedingCount"`
}

type Dashboard struct {
	Summary  Summary               `json:"summary"`
	Accounts []accounts.AccountResponse `json:"accounts"`
	Trend    []TrendPoint          `json:"trend"`
}

func (s *Service) Get(trendDays int) (Dashboard, error) {
	if trendDays < 1 {
		trendDays = 1
	}
	if trendDays > 365 {
		trendDays = 365
	}

	accs, err := s.accounts.ListAll()
	if err != nil {
		return Dashboard{}, err
	}

	var summary Summary
	summary.AccountCount = len(accs)
	siteCodes := map[string]bool{}
	var lastSync string
	for _, a := range accs {
		siteCodes[a.SiteCode] = true
		if a.Status == "ACTIVE" {
			summary.ActiveCount++
		}
		if a.Status == "CREDENTIAL_INVALID" || a.Status == "SYNC_ERROR" {
			summary.ProblemCount++
		}
		if a.LastSyncAt != nil && (*a.LastSyncAt > lastSync) {
			lastSync = *a.LastSyncAt
		}
		if a.Stats == nil {
			continue
		}
		st := a.Stats
		summary.TotalUploadBytes = addBigStr(summary.TotalUploadBytes, st.UploadBytes)
		summary.TotalDownloadBytes = addBigStr(summary.TotalDownloadBytes, st.DownloadBytes)
		summary.TotalSeedingBytes = addBigStr(summary.TotalSeedingBytes, st.SeedingBytes)
		summary.TotalBonus += st.Bonus
		summary.TotalSeedingCount += st.SeedingCount
		summary.TotalHitAndRun += st.HitAndRunCount
	}
	summary.SiteCount = len(siteCodes)
	if lastSync != "" {
		summary.LastSyncAt = &lastSync
	}

	up, _ := strconv.ParseInt(orZero(summary.TotalUploadBytes), 10, 64)
	down, _ := strconv.ParseInt(orZero(summary.TotalDownloadBytes), 10, 64)
	if down == 0 {
		if up == 0 {
			summary.OverallRatio = 0
		} else {
			summary.OverallRatio = -1
		}
	} else {
		summary.OverallRatio = round4(float64(up) / float64(down))
	}
	summary.TotalBonus = round2(summary.TotalBonus)

	trend, err := s.buildTrend(trendDays)
	if err != nil {
		return Dashboard{}, err
	}
	return Dashboard{Summary: summary, Accounts: accs, Trend: trend}, nil
}

func orZero(s string) string {
	if s == "" {
		return "0"
	}
	return s
}

func addBigStr(a, b string) string {
	x, _ := strconv.ParseInt(orZero(a), 10, 64)
	y, _ := strconv.ParseInt(orZero(b), 10, 64)
	return strconv.FormatInt(x+y, 10)
}

func round4(f float64) float64 { return float64(int64(f*10000+0.5)) / 10000 }
func round2(f float64) float64 { return float64(int64(f*100+0.5)) / 100 }

func (s *Service) buildTrend(days int) ([]TrendPoint, error) {
	since := time.Now().UTC().AddDate(0, 0, -days)
	since = time.Date(since.Year(), since.Month(), since.Day(), 0, 0, 0, 0, time.UTC)
	rows, err := s.pool.Query(`
		SELECT snapshotDate,
		       COALESCE(SUM(uploadBytes),0), COALESCE(SUM(downloadBytes),0),
		       COALESCE(SUM(bonus),0), COALESCE(SUM(seedingCount),0)
		FROM TrackerDailySnapshot
		WHERE snapshotDate >= ?
		GROUP BY snapshotDate
		ORDER BY snapshotDate ASC`, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []TrendPoint
	for rows.Next() {
		var d time.Time
		var up, down int64
		var bonus float64
		var seeding int
		if err := rows.Scan(&d, &up, &down, &bonus, &seeding); err != nil {
			return nil, err
		}
		out = append(out, TrendPoint{
			Date: d.Format("2006-01-02"),
			UploadBytes: strconv.FormatInt(up, 10),
			DownloadBytes: strconv.FormatInt(down, 10),
			Bonus: bonus,
			SeedingCount: seeding,
		})
	}
	if out == nil {
		out = []TrendPoint{}
	}
	return out, nil
}

func Handler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		days := 30
		if v := c.Query("trendDays"); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				days = n
			}
		}
		d, err := svc.Get(days)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"statusCode": 500, "message": "查询失败"})
			return
		}
		c.JSON(http.StatusOK, d)
	}
}
