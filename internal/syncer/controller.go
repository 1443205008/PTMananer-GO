package syncer

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
)

// TriggerAccountHandler POST /sync/accounts/:id — 手动触发单账户同步（异步入队）
func TriggerAccountHandler(pool *sql.DB, queue *Queue) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		jobID, err := queue.Enqueue(c.Request.Context(), "account-stats", id, "manual", 3)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"statusCode": 500, "message": "入队失败"})
			return
		}
		c.JSON(http.StatusAccepted, gin.H{"queued": true, "jobId": jobID})
	}
}

// TriggerAllHandler POST /sync/all — 所有账户批量入队
func TriggerAllHandler(pool *sql.DB, queue *Queue) gin.HandlerFunc {
	return func(c *gin.Context) {
		rows, err := pool.Query(`
			SELECT id FROM TrackerAccount
			WHERE isEnabled=true AND syncEnabled=true AND status != 'CREDENTIAL_INVALID'`)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"statusCode": 500, "message": "查询失败"})
			return
		}
		defer rows.Close()
		var jobIDs []string
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				continue
			}
			if jid, err := queue.Enqueue(c.Request.Context(), "account-stats", id, "manual", 3); err == nil {
				jobIDs = append(jobIDs, jid)
			}
		}
		if jobIDs == nil {
			jobIDs = []string{}
		}
		c.JSON(http.StatusAccepted, gin.H{"queued": len(jobIDs), "jobIds": jobIDs})
	}
}

// TriggerTorrentHandler POST /sync/accounts/:id/torrents
func TriggerTorrentHandler(queue *Queue) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		jobID, err := queue.Enqueue(c.Request.Context(), "torrent-sync", id, "manual", 2)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"statusCode": 500, "message": "入队失败"})
			return
		}
		c.JSON(http.StatusAccepted, gin.H{"queued": true, "jobId": jobID})
	}
}
