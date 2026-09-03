// Package alerts: 告警读取 + 已读/忽略（对齐 TS 版 alerts 模块）。
package alerts

import (
	"context"
	"net/http"
	"time"

	"github.com/1443205008/ptmanager-go/internal/db"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct{ pool *pgxpool.Pool }

func NewService(pool *pgxpool.Pool) *Service { return &Service{pool: pool} }

type AlertResponse struct {
	ID          string  `json:"id"`
	AccountID   *string `json:"accountId"`
	AccountName *string `json:"accountName"`
	Type        string  `json:"type"`
	Severity    string  `json:"severity"`
	Title       string  `json:"title"`
	Message     string  `json:"message"`
	IsRead      bool    `json:"isRead"`
	IsDismissed bool    `json:"isDismissed"`
	CreatedAt   string  `json:"createdAt"`
	UpdatedAt   string  `json:"updatedAt"`
}

type alertScan struct {
	AlertResponse
	createdAtT time.Time
	updatedAtT time.Time
}

func (a *alertScan) toResponse() AlertResponse {
	a.CreatedAt = a.createdAtT.UTC().Format(time.RFC3339Nano)
	a.UpdatedAt = a.updatedAtT.UTC().Format(time.RFC3339Nano)
	return a.AlertResponse
}

const alertSelect = `
SELECT al."id", al."accountId", a."accountName", al."type", al."severity", al."title", al."message",
       al."isRead", al."isDismissed", al."createdAt", al."updatedAt"
FROM "Alert" al
LEFT JOIN "TrackerAccount" a ON a."id" = al."accountId"`

func (s *Service) FindAll(unreadOnly bool) ([]AlertResponse, error) {
	q := alertSelect + ` WHERE al."isDismissed"=false`
	if unreadOnly {
		q += ` AND al."isRead"=false`
	}
	q += ` ORDER BY al."severity" DESC, al."createdAt" DESC LIMIT 200`
	rows, err := s.pool.Query(context.Background(), q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAlerts(rows)
}

func scanAlerts(rows pgx.Rows) ([]AlertResponse, error) {
	var out []AlertResponse
	for rows.Next() {
		var r alertScan
		if err := rows.Scan(&r.ID, &r.AccountID, &r.AccountName, &r.Type, &r.Severity, &r.Title, &r.Message,
			&r.IsRead, &r.IsDismissed, &r.createdAtT, &r.updatedAtT); err != nil {
			return nil, err
		}
		out = append(out, r.toResponse())
	}
	if out == nil {
		out = []AlertResponse{}
	}
	return out, nil
}

func (s *Service) UnreadCount() (int, error) {
	var n int
	err := s.pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM "Alert" WHERE "isRead"=false AND "isDismissed"=false`).Scan(&n)
	return n, err
}

func (s *Service) MarkRead(id string) (AlertResponse, error) {
	ctx := context.Background()
	if _, err := s.GetOne(id); err != nil {
		return AlertResponse{}, err
	}
	_, err := s.pool.Exec(ctx, `UPDATE "Alert" SET "isRead"=true, "updatedAt"=NOW() WHERE "id"=$1`, id)
	if err != nil {
		return AlertResponse{}, err
	}
	return s.GetOne(id)
}

func (s *Service) GetOne(id string) (AlertResponse, error) {
	var r alertScan
	err := s.pool.QueryRow(context.Background(), alertSelect+` WHERE al."id"=$1`, id).
		Scan(&r.ID, &r.AccountID, &r.AccountName, &r.Type, &r.Severity, &r.Title, &r.Message,
			&r.IsRead, &r.IsDismissed, &r.createdAtT, &r.updatedAtT)
	if err != nil {
		if err == pgx.ErrNoRows {
			return AlertResponse{}, &apiErr{http.StatusNotFound, "Alert " + id + " not found"}
		}
		return AlertResponse{}, err
	}
	return r.toResponse(), nil
}

func (s *Service) MarkAllRead() (int, error) {
	tag, err := s.pool.Exec(context.Background(),
		`UPDATE "Alert" SET "isRead"=true, "updatedAt"=NOW() WHERE "isRead"=false AND "isDismissed"=false`)
	if err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
}

func (s *Service) Dismiss(id string) error {
	if _, err := s.GetOne(id); err != nil {
		return err
	}
	_, err := s.pool.Exec(context.Background(),
		`UPDATE "Alert" SET "isDismissed"=true, "isRead"=true, "updatedAt"=NOW() WHERE "id"=$1`, id)
	return err
}

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

func ListHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		resp, err := svc.FindAll(c.Query("unreadOnly") == "true")
		if err != nil {
			writeErr(c, err)
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}

func UnreadCountHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		n, err := svc.UnreadCount()
		if err != nil {
			writeErr(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"count": n})
	}
}

func MarkReadHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		resp, err := svc.MarkRead(c.Param("id"))
		if err != nil {
			writeErr(c, err)
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}

func MarkAllReadHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		n, err := svc.MarkAllRead()
		if err != nil {
			writeErr(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"updated": n})
	}
}

func DismissHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := svc.Dismiss(c.Param("id")); err != nil {
			writeErr(c, err)
			return
		}
		c.Status(http.StatusNoContent)
	}
}

var _ = db.NewID
var _ = time.Now
