// Package fakeseed: 模拟保种（对齐 TS 版 fake-seed 模块）。
// 下载 .torrent → 解析 bencode → 周期性向 tracker 上报做种。
package fakeseed

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha1"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/1443205008/ptmanager-go/internal/config"
	"github.com/1443205008/ptmanager-go/internal/db"
	"github.com/1443205008/ptmanager-go/internal/providers"

	"github.com/gin-gonic/gin"
)

type Service struct {
	pool     *sql.DB
	registry *providers.Registry
	cfg      *config.Config

	mu       sync.Mutex
	sessions map[string]*session // jobID → session
}

func NewService(pool *sql.DB, registry *providers.Registry, cfg *config.Config) *Service {
	return &Service{pool: pool, registry: registry, cfg: cfg, sessions: map[string]*session{}}
}

type session struct {
	stop     chan struct{}
	interval int
}

// FakeSeedJobResponse 对齐前端 FakeSeedJob 类型
type FakeSeedJobResponse struct {
	ID           string  `json:"id"`
	AccountID    string  `json:"accountId"`
	AccountName  *string `json:"accountName"`
	SiteCode     *string `json:"siteCode"`
	SiteName     *string `json:"siteName"`
	TorrentID    string  `json:"torrentId"`
	TorrentName  string  `json:"torrentName"`
	InfoHash     string  `json:"infoHash"`
	TotalSize    string  `json:"totalSize"`
	Status       string  `json:"status"`
	Interval     int     `json:"interval"`
	LastReportAt *string `json:"lastReportAt"`
	NextReportAt *string `json:"nextReportAt"`
	ErrorMessage *string `json:"errorMessage"`
	CreatedAt    string  `json:"createdAt"`
}

type jobScan struct {
	jobRow
	createdAtT time.Time
}

type jobRow struct {
	id, accountID, torrentID, torrentName, infoHash, trackerURL string
	totalSize                                                   int64
	peerID, peerKey                                             string
	port, interval                                              int
	status                                                      string
	lastReportAt, nextReportAt                                  *time.Time
	errorMessage                                                *string
	createdAt                                                   *time.Time
	accountName, siteCode, siteName                             *string
}

const jobSelect = `
SELECT f.id, f.accountId, f.torrentId, f.torrentName, f.infoHash, f.trackerUrl,
       f.totalSize, f.peerId, f.peerKey, f.port, f.interval, f.status,
       f.lastReportAt, f.nextReportAt, f.errorMessage, f.createdAt,
       a.accountName, s.code, s.name
FROM FakeSeedJob f
LEFT JOIN TrackerAccount a ON a.id = f.accountId
LEFT JOIN TrackerSite s ON s.id = a.siteId`

func (s *Service) getJob(ctx context.Context, id string) (*jobRow, error) {
	var r jobRow
	err := s.pool.QueryRow(jobSelect+" WHERE f.id=?", id).Scan(
		&r.id, &r.accountID, &r.torrentID, &r.torrentName, &r.infoHash, &r.trackerURL,
		&r.totalSize, &r.peerID, &r.peerKey, &r.port, &r.interval, &r.status,
		&r.lastReportAt, &r.nextReportAt, &r.errorMessage, &r.createdAt,
		&r.accountName, &r.siteCode, &r.siteName)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, &apiErr{http.StatusNotFound, "保种任务不存在"}
		}
		return nil, err
	}
	return &r, nil
}

func (r *jobRow) toResponse() FakeSeedJobResponse {
	return FakeSeedJobResponse{
		ID: r.id, AccountID: r.accountID, AccountName: r.accountName, SiteCode: r.siteCode, SiteName: r.siteName,
		TorrentID: r.torrentID, TorrentName: r.torrentName, InfoHash: r.infoHash,
		TotalSize: strconv.FormatInt(r.totalSize, 10), Status: r.status, Interval: r.interval,
		LastReportAt: fmtTime(r.lastReportAt), NextReportAt: fmtTime(r.nextReportAt),
		ErrorMessage: r.errorMessage, CreatedAt: fmtTimeT(r.createdAt),
	}
}

func fmtTimeT(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}

func fmtTime(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.UTC().Format(time.RFC3339Nano)
	return &s
}

// StartByTorrentID 启动保种
func (s *Service) StartByTorrentID(accountID, torrentID, torrentName string) (FakeSeedJobResponse, error) {
	ctx := context.Background()

	// 已存在则复启
	var existingID string
	err := s.pool.QueryRow(`SELECT id FROM FakeSeedJob WHERE torrentId=? AND accountId=?`, torrentID, accountID).Scan(&existingID)
	if err == nil {
		var status string
		_ = s.pool.QueryRow(`SELECT status FROM FakeSeedJob WHERE id=?`, existingID).Scan(&status)
		if status == "RUNNING" {
			return FakeSeedJobResponse{}, &apiErr{http.StatusConflict, "该种子已在保种中"}
		}
		job, err := s.getJob(ctx, existingID)
		if err != nil {
			return FakeSeedJobResponse{}, err
		}
		if err := s.startSession(job); err != nil {
			return FakeSeedJobResponse{}, err
		}
		return job.toResponse(), nil
	}
	if err != sql.ErrNoRows {
		return FakeSeedJobResponse{}, err
	}

	// 下载 torrent 文件并解析
	torrentData, err := s.downloadTorrentFile(ctx, accountID, torrentID)
	if err != nil {
		// 上游错误（M-Team Key 无效 / 下载失败）→ 503，对齐 TS 版 ServiceUnavailableException
		if ae, ok := err.(*apiErr); ok {
			return FakeSeedJobResponse{}, ae
		}
		return FakeSeedJobResponse{}, &apiErr{http.StatusServiceUnavailable, "获取种子文件失败：" + err.Error()}
	}
	parsed, err := parseTorrent(torrentData)
	if err != nil {
		return FakeSeedJobResponse{}, err
	}

	if torrentName == "" {
		torrentName = parsed.name
		if torrentName == "" {
			torrentName = torrentID
		}
	}

	jobID := db.NewID()
	_, err = s.pool.Exec(`
		INSERT INTO FakeSeedJob(id,accountId,torrentId,torrentName,infoHash,trackerUrl,
			totalSize,peerId,peerKey,port,status,interval,createdAt,updatedAt)
		VALUES(?,?,?,?,?,?,?,?,?,34567,'STOPPED',300,NOW(),NOW())`,
		jobID, accountID, torrentID, torrentName, parsed.infoHash, parsed.trackerURL,
		parsed.totalSize, genPeerID(), genKey())
	if err != nil {
		return FakeSeedJobResponse{}, err
	}
	job, err := s.getJob(ctx, jobID)
	if err != nil {
		return FakeSeedJobResponse{}, err
	}
	if err := s.startSession(job); err != nil {
		return FakeSeedJobResponse{}, err
	}
	return job.toResponse(), nil
}

// downloadTorrentFile 用 provider 拿下载 token 再下载 .torrent
func (s *Service) downloadTorrentFile(ctx context.Context, accountID, torrentID string) ([]byte, error) {
	var siteCode string
	err := s.pool.QueryRow(`
		SELECT s.code FROM TrackerAccount a JOIN TrackerSite s ON s.id=a.siteId
		WHERE a.id=? AND a.isEnabled=true`, accountID).Scan(&siteCode)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, &apiErr{http.StatusNotFound, "账户不存在或未启用"}
		}
		return nil, err
	}
	p, err := s.registry.Get(siteCode)
	if err != nil {
		return nil, err
	}
	type dlTokenProvider interface {
		GenDlToken(accountID, torrentID string) (string, error)
	}
	tp, ok := p.(dlTokenProvider)
	if !ok {
		return nil, &apiErr{http.StatusServiceUnavailable, "该站点不支持保种"}
	}
	downloadURL, err := tp.GenDlToken(accountID, torrentID)
	if err != nil {
		return nil, err
	}

	req, _ := http.NewRequest(http.MethodGet, downloadURL, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0")
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("下载种子失败：HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20)) // 防超大文件
	if err != nil {
		return nil, err
	}
	// bencode 快速预检：合法 torrent 必以 'd' 开头
	if len(body) == 0 || body[0] != 'd' {
		return nil, fmt.Errorf("响应不是有效的 torrent 文件")
	}
	return body, nil
}

// ─── bencode 解析（对齐 TS 版手写解析器语义）────────────────────────────

type parsedTorrent struct {
	infoHash   string // 40 hex
	trackerURL string
	totalSize  int64
	name       string
}

// ParseTorrentPublic 导出供外部调用/测试
func ParseTorrentPublic(data []byte) (*ParsedTorrent, error) {
	return parseTorrent(data)
}

type ParsedTorrent = parsedTorrent

func parseTorrent(data []byte) (*parsedTorrent, error) {
	// announce
	announce, err := bencodeStringForKey(data, []byte("8:announce"))
	if err != nil || announce == "" {
		return nil, fmt.Errorf("torrent 缺少 announce 字段")
	}

	// info 段（计算 infoHash）
	infoKey := []byte("4:info")
	infoPos := bytes.Index(data, infoKey)
	if infoPos == -1 {
		return nil, fmt.Errorf("torrent 缺少 info 字段")
	}
	infoStart := infoPos + len(infoKey)
	infoEnd, err := bencodeSkip(data, infoStart)
	if err != nil {
		return nil, fmt.Errorf("解析 info 段失败: %w", err)
	}
	if infoEnd <= infoStart || infoEnd > len(data) {
		return nil, fmt.Errorf("info 段边界异常")
	}
	sum := sha1.Sum(data[infoStart:infoEnd])

	// name（info 段内）
	name, _ := bencodeStringForKey(data[infoStart:infoEnd], []byte("4:name"))

	// totalSize：单文件 length 或 多文件 length 之和
	totalSize := sumLengths(data[infoStart:infoEnd])

	return &parsedTorrent{
		infoHash:   hex.EncodeToString(sum[:]),
		trackerURL: announce,
		totalSize:  totalSize,
		name:       name,
	}, nil
}

// bencodeStringForKey 在 data 中找 "N:key" 后面的字符串值
func bencodeStringForKey(data, key []byte) (string, error) {
	pos := bytes.Index(data, key)
	if pos == -1 {
		return "", fmt.Errorf("key not found")
	}
	p := pos + len(key)
	if p >= len(data) {
		return "", fmt.Errorf("bad string")
	}
	colon := bytes.IndexByte(data[p:], ':')
	if colon == -1 {
		return "", fmt.Errorf("bad string")
	}
	length, err := strconv.Atoi(string(data[p : p+colon]))
	if err != nil || length < 0 {
		return "", fmt.Errorf("bad string length")
	}
	start := p + colon + 1
	if start+length > len(data) {
		return "", fmt.Errorf("string out of range")
	}
	return string(data[start : start+length]), nil
}

// bencodeSkip 跳过一个 bencode 值，返回结束位置（不含）
func bencodeSkip(buf []byte, pos int) (int, error) {
	if pos < 0 || pos >= len(buf) {
		return 0, fmt.Errorf("eof")
	}
	switch buf[pos] {
	case 'i': // 整数 i123e
		e := bytes.IndexByte(buf[pos:], 'e')
		if e == -1 {
			return 0, fmt.Errorf("bad int")
		}
		return pos + e + 1, nil
	case 'l': // 列表 l...e
		p := pos + 1
		for p < len(buf) && buf[p] != 'e' {
			np, err := bencodeSkip(buf, p)
			if err != nil {
				return 0, err
			}
			if np <= p { // 防御：无进展即损坏数据
				return 0, fmt.Errorf("malformed bencode")
			}
			p = np
		}
		if p >= len(buf) { // 缺少闭合 'e'
			return 0, fmt.Errorf("unterminated list")
		}
		return p + 1, nil
	case 'd': // 字典 d...e（key、value 交替）
		p := pos + 1
		for p < len(buf) && buf[p] != 'e' {
			np, err := bencodeSkip(buf, p) // key
			if err != nil {
				return 0, err
			}
			if np <= p {
				return 0, fmt.Errorf("malformed bencode")
			}
			np2, err := bencodeSkip(buf, np) // value
			if err != nil {
				return 0, err
			}
			if np2 <= np {
				return 0, fmt.Errorf("malformed bencode")
			}
			p = np2
		}
		if p >= len(buf) {
			return 0, fmt.Errorf("unterminated dict")
		}
		return p + 1, nil
	default: // 字符串 N:...
		colon := bytes.IndexByte(buf[pos:], ':')
		if colon == -1 {
			return 0, fmt.Errorf("bad string")
		}
		length, err := strconv.Atoi(string(buf[pos : pos+colon]))
		if err != nil || length < 0 {
			return 0, fmt.Errorf("bad string length")
		}
		end := pos + colon + 1 + length
		if end > len(buf) {
			return 0, fmt.Errorf("string out of range")
		}
		return end, nil
	}
}

// sumLengths 累加所有 6:length 整数值（多文件）或取第一个（单文件）
func sumLengths(info []byte) int64 {
	var total int64
	pos := 0
	lengthKey := []byte("6:length")
	for pos <= len(info) {
		idx := bytes.Index(info[pos:], lengthKey)
		if idx == -1 {
			break
		}
		abs := pos + idx
		vStart := abs + len(lengthKey)
		if vStart < len(info) && info[vStart] == 'i' {
			e := bytes.IndexByte(info[vStart:], 'e')
			if e > 1 { // 至少 1 位数字
				if n, err := strconv.ParseInt(string(info[vStart+1:vStart+e]), 10, 64); err == nil && n > 0 {
					total += n
				}
			}
		}
		pos = abs + len(lengthKey)
	}
	return total
}

// parseTrackerInterval 从 tracker 响应解析 interval（60 < v < 86400）
func parseTrackerInterval(buf []byte) (int, bool) {
	key := []byte("8:interval")
	idx := bytes.Index(buf, key)
	if idx == -1 {
		return 0, false
	}
	vStart := idx + len(key)
	if vStart >= len(buf) || buf[vStart] != 'i' {
		return 0, false
	}
	e := bytes.IndexByte(buf[vStart:], 'e')
	if e == -1 {
		return 0, false
	}
	v, err := strconv.Atoi(string(buf[vStart+1 : vStart+e]))
	if err != nil || v <= 60 || v >= 86400 {
		return 0, false
	}
	return v, true
}

// ─── tracker announce ─────────────────────────────────────────────────────

func (s *Service) sendAnnounce(job *jobRow, event string) error {
	infoHashBytes, err := hex.DecodeString(job.infoHash)
	if err != nil {
		return err
	}
	// info_hash percent-encode（每个字节都编码）
	var sb strings.Builder
	for _, b := range infoHashBytes {
		fmt.Fprintf(&sb, "%%%02X", b)
	}

	u := job.trackerURL
	sep := "?"
	if strings.Contains(u, "?") {
		sep = "&"
	}
	announceURL := u + sep + "info_hash=" + sb.String() +
		"&peer_id=" + url.QueryEscape(job.peerID) +
		"&port=" + strconv.Itoa(job.port) +
		"&uploaded=0&downloaded=0&left=0&compact=1&numwant=0&key=" + url.QueryEscape(job.peerKey)
	if event != "" {
		announceURL += "&event=" + event
	}

	req, _ := http.NewRequest(http.MethodGet, announceURL, nil)
	req.Header.Set("User-Agent", "Transmission/2.92")
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

	// tracker 拒绝（bencode failure reason）：b"d14:failure reason..."
	if idx := bytes.Index(body, []byte("failure reason")); idx != -1 {
		reasonEnd := bytes.IndexByte(body[idx:], 'e')
		msg := string(body[idx : idx+reasonEnd])
		return fmt.Errorf("tracker 拒绝：%s", msg)
	}

	if newInterval, ok := parseTrackerInterval(body); ok {
		if _, err := s.pool.Exec(`UPDATE FakeSeedJob SET interval=?, updatedAt=NOW() WHERE id=?`, newInterval, job.id); err == nil {
			s.mu.Lock()
			if sess, ok := s.sessions[job.id]; ok && sess.interval != newInterval {
				sess.interval = newInterval
			}
			s.mu.Unlock()
		}
	}
	return nil
}

// startSession 标记 RUNNING、发 started 事件、起定时器
func (s *Service) startSession(job *jobRow) error {
	s.stopSession(job.id)
	now := time.Now()
	_, err := s.pool.Exec(`
		UPDATE FakeSeedJob SET status='RUNNING', lastReportAt=?, nextReportAt=?, errorMessage=NULL, updatedAt=NOW()
		WHERE id=?`, now, now.Add(time.Duration(job.interval)*time.Second), job.id)
	if err != nil {
		return err
	}
	if err := s.sendAnnounce(job, "started"); err != nil {
		_, _ = s.pool.Exec(`UPDATE FakeSeedJob SET status='ERROR', errorMessage=?, updatedAt=NOW() WHERE id=?`, err.Error(), job.id)
		return &apiErr{http.StatusServiceUnavailable, "保种启动失败：" + err.Error()}
	}
	s.scheduleTimer(job.id, job.interval)
	return nil
}

// scheduleTimer 周期上报
func (s *Service) scheduleTimer(jobID string, interval int) {
	s.mu.Lock()
	if old, ok := s.sessions[jobID]; ok {
		close(old.stop)
	}
	stop := make(chan struct{})
	s.sessions[jobID] = &session{stop: stop, interval: interval}
	s.mu.Unlock()

	go func() {
		timer := time.NewTimer(time.Duration(interval) * time.Second)
		defer timer.Stop()
		for {
			select {
			case <-stop:
				return
			case <-timer.C:
			}
			ctx := context.Background()
			job, err := s.getJob(ctx, jobID)
			if err != nil || job.status != "RUNNING" {
				s.stopSession(jobID)
				return
			}
			now := time.Now()
			_, _ = s.pool.Exec(`
				UPDATE FakeSeedJob SET lastReportAt=?, nextReportAt=?, updatedAt=NOW() WHERE id=?`,
				now, now.Add(time.Duration(job.interval)*time.Second), jobID)
			if err := s.sendAnnounce(job, ""); err != nil {
				_, _ = s.pool.Exec(`UPDATE FakeSeedJob SET status='ERROR', errorMessage=?, updatedAt=NOW() WHERE id=?`, err.Error(), jobID)
				s.stopSession(jobID)
				return
			}
			// 用最新 interval 重启定时
			s.mu.Lock()
			cur := s.sessions[jobID]
			s.mu.Unlock()
			if cur == nil {
				return
			}
			timer.Reset(time.Duration(cur.interval) * time.Second)
		}
	}()
}

func (s *Service) stopSession(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if sess, ok := s.sessions[id]; ok {
		close(sess.stop)
		delete(s.sessions, id)
	}
}

func (s *Service) Stop(id string) (map[string]string, error) {
	ctx := context.Background()
	job, err := s.getJob(ctx, id)
	if err != nil {
		return nil, err
	}
	s.stopSession(id)
	_ = s.sendAnnounce(job, "stopped")
	_, err = s.pool.Exec(`UPDATE FakeSeedJob SET status='STOPPED', nextReportAt=NULL, updatedAt=NOW() WHERE id=?`, id)
	if err != nil {
		return nil, err
	}
	return map[string]string{"id": id, "status": "STOPPED"}, nil
}

func (s *Service) Remove(id string) (map[string]string, error) {
	s.stopSession(id)
	_, err := s.pool.Exec(`DELETE FROM FakeSeedJob WHERE id=?`, id)
	if err != nil {
		return nil, err
	}
	return map[string]string{"id": id}, nil
}

func (s *Service) List(accountID string) ([]FakeSeedJobResponse, error) {
	q := jobSelect
	var args []interface{}
	if accountID != "" {
		args = append(args, accountID)
		q += " WHERE f.accountId=?"
	}
	q += " ORDER BY f.createdAt DESC"
	rows, err := s.pool.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []FakeSeedJobResponse
	for rows.Next() {
		var r jobRow
		if err := rows.Scan(&r.id, &r.accountID, &r.torrentID, &r.torrentName, &r.infoHash, &r.trackerURL,
			&r.totalSize, &r.peerID, &r.peerKey, &r.port, &r.interval, &r.status,
			&r.lastReportAt, &r.nextReportAt, &r.errorMessage, &r.createdAt,
			&r.accountName, &r.siteCode, &r.siteName); err != nil {
			return nil, err
		}
		out = append(out, r.toResponse())
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if out == nil {
		out = []FakeSeedJobResponse{}
	}
	return out, nil
}

// RestoreRunning 服务启动时恢复 RUNNING 任务（避免重启后丢失）
func RestoreRunning(pool *sql.DB, svc *Service) {
	ctx := context.Background()
	rows, err := pool.Query(`SELECT id FROM FakeSeedJob WHERE status='RUNNING'`)
	if err != nil {
		return
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		_ = rows.Scan(&id)
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return
	}
	if len(ids) == 0 {
		return
	}
	fmt.Printf("[保种] 恢复 %d 个中断的保种任务…\n", len(ids))
	restored := 0
	for _, id := range ids {
		_, _ = pool.Exec(`UPDATE FakeSeedJob SET status='STOPPED' WHERE id=?`, id)
		job, err := svc.getJob(ctx, id)
		if err != nil {
			continue
		}
		if err := svc.startSession(job); err != nil {
			fmt.Printf("[保种] ⚠️ 恢复失败：%s — %v\n", job.torrentName, err)
			continue
		}
		restored++
	}
	fmt.Printf("[保种] 共恢复 %d/%d 个任务\n", restored, len(ids))
}

// ─── helpers + handlers ────────────────────────────────────────────────────

func genPeerID() string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	out := "-TR2920-"
	for _, c := range b {
		out += string(chars[int(c)%len(chars)])
	}
	return out
}

func genKey() string {
	const chars = "ABCDEF0123456789"
	out := ""
	for i := 0; i < 8; i++ {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		out += string(chars[n.Int64()])
	}
	return out
}

func bytesIndexImpl(b []byte, sub string) int {
	return bytes.Index(b, []byte(sub))
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
	log.Printf("[http] 500 on %s %s: %v", c.Request.Method, c.Request.URL.Path, err)
	c.JSON(http.StatusInternalServerError, gin.H{"statusCode": 500, "message": "服务器内部错误"})
}

func ListHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		resp, err := svc.List(c.Query("accountId"))
		if err != nil {
			writeErr(c, err)
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}

func StartHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			AccountID   string `json:"accountId" binding:"required"`
			TorrentID   string `json:"torrentId" binding:"required"`
			TorrentName string `json:"torrentName"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"statusCode": 400, "message": "参数不合法"})
			return
		}
		resp, err := svc.StartByTorrentID(body.AccountID, body.TorrentID, body.TorrentName)
		if err != nil {
			writeErr(c, err)
			return
		}
		c.JSON(http.StatusCreated, resp)
	}
}

func StopHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		resp, err := svc.Stop(c.Param("id"))
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
