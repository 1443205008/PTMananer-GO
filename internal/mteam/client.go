package mteam

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/1443205008/ptmanager-go/internal/config"
	"github.com/1443205008/ptmanager-go/internal/domain"
)

// MTeamError M-Team 特有错误
type MTeamError struct {
	Code         domain.TrackerErrorCode
	Message      string
	StatusCode   int
	ResponseBody string
}

func (e *MTeamError) Error() string { return e.Message }

// TrackerCode 实现 domain.Coder，供 syncer 提取错误码
func (e *MTeamError) TrackerCode() domain.TrackerErrorCode { return e.Code }

// httpStatusToTrackerError HTTP 状态码 → TrackerErrorCode
func httpStatusToTrackerError(status int) domain.TrackerErrorCode {
	switch {
	case status == 401:
		return domain.ErrAuthInvalid
	case status == 403:
		return domain.ErrAuthExpired
	case status == 429:
		return domain.ErrRateLimited
	case status >= 500 && status < 600:
		return domain.ErrSiteOffline
	default:
		return domain.ErrUnknown
	}
}

// isSuccessCode 成功码可能是 number 0 或 string "0"——两者都认。
func isSuccessCode(codeStr string) bool {
	return codeStr == "0"
}

// businessCodeToTrackerError M-Team 业务码 → TrackerErrorCode
// 真实响应确认：0="0"成功；1="無許可權"；401=鉴权失败
func businessCodeToTrackerError(codeStr string) domain.TrackerErrorCode {
	switch codeStr {
	case "1":
		return domain.ErrAPIChanged // 无权限：Key 有效但端点需更高权限
	case "401":
		return domain.ErrAuthInvalid
	case "403":
		return domain.ErrAuthExpired
	case "429":
		return domain.ErrRateLimited
	default:
		return domain.ErrUnknown
	}
}

// Client M-Team HTTP API 客户端（统一 base URL / x-api-key / timeout / 重试 / 限速）。
type Client struct {
	baseURL    string
	timeout    time.Duration
	maxRetries int
	httpClient *http.Client

	mu         sync.Mutex
	rateWindow []time.Time // 滑动窗口时间戳
	rateLimit  int         // 每分钟
}

func NewClient(cfg *config.Config) *Client {
	return &Client{
		baseURL:    cfg.MTeamBaseURL,
		timeout:    time.Duration(cfg.MTeamTimeoutMs) * time.Millisecond,
		maxRetries: cfg.MTeamMaxRetries,
		httpClient: &http.Client{Timeout: time.Duration(cfg.MTeamTimeoutMs) * time.Millisecond},
		rateLimit:  cfg.MTeamRateLimitPerMin,
	}
}

// Request 发起 JSON POST 请求
func (c *Client) Request(endpoint, apiKey string, body interface{}, out interface{}, maxRetries int) error {
	if maxRetries <= 0 {
		maxRetries = c.maxRetries
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return err
	}
	return c.doWithRetry(endpoint, apiKey, "application/json", jsonBody, out, 0, maxRetries)
}

// RequestForm 以 application/x-www-form-urlencoded 发送（genDlToken 需要）
func (c *Client) RequestForm(endpoint, apiKey string, form map[string]string, out interface{}) error {
	buf := &bytes.Buffer{}
	first := true
	for k, v := range form {
		if !first {
			buf.WriteByte('&')
		}
		first = false
		buf.WriteString(escape(k) + "=" + escape(v))
	}
	return c.doWithRetry(endpoint, apiKey, "application/x-www-form-urlencoded", buf.Bytes(), out, 0, c.maxRetries)
}

func escape(s string) string {
	var b bytes.Buffer
	for _, r := range []byte(s) {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' || r == '~' {
			b.WriteByte(r)
		} else {
			fmt.Fprintf(&b, "%%%02X", r)
		}
	}
	return b.String()
}

func (c *Client) doWithRetry(endpoint, apiKey, contentType string, body []byte, out interface{}, attempt, maxRetries int) error {
	c.enforceRateLimit()

	url := c.baseURL + endpoint
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set(AuthHeader, apiKey)
	req.Header.Set("User-Agent", UserAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		// 网络错误 — 指数退避重试
		if attempt < maxRetries {
			time.Sleep(c.backoffDelay(attempt))
			return c.doWithRetry(endpoint, apiKey, contentType, body, out, attempt+1, maxRetries)
		}
		return &MTeamError{Code: domain.ErrNetworkError, Message: err.Error()}
	}
	defer resp.Body.Close()

	// 5xx — 指数退避重试（对齐 TS 版：5xx / 网络错误都重试）
	if resp.StatusCode >= 500 && attempt < maxRetries {
		time.Sleep(c.backoffDelay(attempt))
		return c.doWithRetry(endpoint, apiKey, contentType, body, out, attempt+1, maxRetries)
	}

	raw, _ := io.ReadAll(resp.Body)

	// 业务层错误处理（HTTP 200 但 code 非成功值）
	var result Result
	if err := json.Unmarshal(raw, &result); err != nil || result.Code.String() == "" {
		// Cloudflare / 非 JSON 响应：不能当成成功
		if resp.StatusCode == 429 {
			return &MTeamError{Code: domain.ErrRateLimited, Message: "M-Team 请求过于频繁，请稍后再试", StatusCode: 429}
		}
		return &MTeamError{Code: domain.ErrInvalidResponse, Message: "M-Team 返回了无法解析的响应", StatusCode: resp.StatusCode, ResponseBody: string(raw)}
	}

	codeStr := result.Code.String()
	if !isSuccessCode(codeStr) {
		code := businessCodeToTrackerError(codeStr)
		msg := result.Message
		if msg == "" {
			msg = "API error"
		}
		return &MTeamError{Code: code, Message: msg, StatusCode: resp.StatusCode, ResponseBody: string(raw)}
	}

	if out != nil && len(result.Data) > 0 {
		if err := json.Unmarshal(result.Data, out); err != nil {
			return &MTeamError{Code: domain.ErrInvalidResponse, Message: "M-Team 响应 data 解析失败: " + err.Error(), StatusCode: resp.StatusCode}
		}
	}
	return nil
}

// 429 与 4xx 不重试；5xx / 网络错误重试（在 doWithRetry 内处理）
func (c *Client) backoffDelay(attempt int) time.Duration {
	base := time.Second * time.Duration(math.Pow(2, float64(attempt)))
	if base > 60*time.Second {
		base = 60 * time.Second
	}
	return base
}

// enforceRateLimit 滑动窗口 Rate Limiter（每分钟 N 次）
func (c *Client) enforceRateLimit() {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	window := time.Minute

	// 清除窗口外旧记录
	i := 0
	for ; i < len(c.rateWindow); i++ {
		if c.rateWindow[i].After(now.Add(-window)) {
			break
		}
	}
	c.rateWindow = c.rateWindow[i:]

	if len(c.rateWindow) >= c.rateLimit {
		oldest := c.rateWindow[0]
		wait := window - now.Sub(oldest) + 50*time.Millisecond
		if wait > 0 {
			time.Sleep(wait)
		}
		// 重新清理
		now = time.Now()
		i = 0
		for ; i < len(c.rateWindow); i++ {
			if c.rateWindow[i].After(now.Add(-window)) {
				break
			}
		}
		c.rateWindow = c.rateWindow[i:]
	}
	c.rateWindow = append(c.rateWindow, time.Now())
}
