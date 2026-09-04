// Package auth: JWT 登录 + cookie 中间件（对齐 TS 版 auth 模块行为）。
package auth

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/1443205008/ptmanager-go/internal/config"
	"github.com/1443205008/ptmanager-go/internal/cryptoutil"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const CookieName = "pt_manager_session"

type AuthUser struct {
	ID        string  `json:"id"`
	Email     string  `json:"email"`
	Name      *string `json:"name"`
	CreatedAt string  `json:"createdAt"`
}

type Service struct {
	pool *sql.DB
	cfg  *config.Config
}

func NewService(pool *sql.DB, cfg *config.Config) *Service {
	return &Service{pool: pool, cfg: cfg}
}

// parseDuration 支持 "7d" 形式（JWT_EXPIRES_IN）
func parseDuration(s string) time.Duration {
	if s == "" {
		return 7 * 24 * time.Hour
	}
	if strings.HasSuffix(s, "d") && isAllDigits(strings.TrimSuffix(s, "d")) {
		days, _ := strconv.Atoi(strings.TrimSuffix(s, "d"))
		return time.Duration(days) * 24 * time.Hour
	}
	if d, err := time.ParseDuration(s); err == nil {
		return d
	}
	return 7 * 24 * time.Hour
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func (s *Service) Login(email, password string) (string, AuthUser, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	var id, hash, name *string
	var createdAt time.Time
	err := s.pool.QueryRow(`SELECT id, password, name, createdAt FROM User WHERE email=?`, email).Scan(&id, &hash, &name, &createdAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", AuthUser{}, &APIError{Status: http.StatusUnauthorized, Message: "邮箱或密码错误"}
		}
		return "", AuthUser{}, err
	}
	if !cryptoutil.VerifyPassword(password, deref(hash)) {
		return "", AuthUser{}, &APIError{Status: http.StatusUnauthorized, Message: "邮箱或密码错误"}
	}

	claims := jwt.MapClaims{
		"sub":   deref(id),
		"email": email,
		"iat":   time.Now().Unix(),
		"exp":   time.Now().Add(parseDuration(s.cfg.JWTExpiresIn)).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return "", AuthUser{}, err
	}

	createdAtStr := createdAt.UTC().Format(time.RFC3339Nano)
	user := AuthUser{ID: deref(id), Email: email, Name: name, CreatedAt: createdAtStr}
	return signed, user, nil
}

func (s *Service) GetUser(userID string) (AuthUser, error) {
	var email string
	var name *string
	var createdAt time.Time
	err := s.pool.QueryRow(`SELECT email, name, createdAt FROM User WHERE id=?`, userID).Scan(&email, &name, &createdAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return AuthUser{}, &APIError{Status: http.StatusUnauthorized, Message: "登录用户不存在"}
		}
		return AuthUser{}, err
	}
	return AuthUser{ID: userID, Email: email, Name: name, CreatedAt: createdAt.UTC().Format(time.RFC3339Nano)}, nil
}

// UpdateAccount 修改当前登录用户的邮箱和/或密码。必须提供当前密码。
func (s *Service) UpdateAccount(userID, currentPassword, newEmail, newPassword string) (AuthUser, error) {
	newEmail = strings.ToLower(strings.TrimSpace(newEmail))
	newPassword = strings.TrimSpace(newPassword)
	if currentPassword == "" {
		return AuthUser{}, &APIError{Status: http.StatusBadRequest, Message: "请输入当前密码"}
	}
	if newEmail == "" && newPassword == "" {
		return AuthUser{}, &APIError{Status: http.StatusBadRequest, Message: "请填写新邮箱或新密码"}
	}
	if newPassword != "" && len(newPassword) < 8 {
		return AuthUser{}, &APIError{Status: http.StatusBadRequest, Message: "新密码至少 8 位"}
	}
	if newPassword != "" && len(newPassword) > 128 {
		return AuthUser{}, &APIError{Status: http.StatusBadRequest, Message: "新密码过长"}
	}
	if newEmail != "" {
		if !strings.Contains(newEmail, "@") || strings.Contains(newEmail, " ") {
			return AuthUser{}, &APIError{Status: http.StatusBadRequest, Message: "请输入有效的邮箱地址"}
		}
	}

	var hash, email string
	var name *string
	var createdAt time.Time
	err := s.pool.QueryRow(`SELECT password, email, name, createdAt FROM User WHERE id=?`, userID).
		Scan(&hash, &email, &name, &createdAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return AuthUser{}, &APIError{Status: http.StatusUnauthorized, Message: "登录用户不存在"}
		}
		return AuthUser{}, err
	}
	if !cryptoutil.VerifyPassword(currentPassword, hash) {
		return AuthUser{}, &APIError{Status: http.StatusUnauthorized, Message: "当前密码错误"}
	}

	if newEmail != "" && newEmail != email {
		var dummy string
		dup := s.pool.QueryRow(`SELECT id FROM User WHERE email=? AND id<>?`, newEmail, userID).Scan(&dummy)
		if dup == nil {
			return AuthUser{}, &APIError{Status: http.StatusConflict, Message: "该邮箱已被使用"}
		}
		if dup != sql.ErrNoRows {
			return AuthUser{}, dup
		}
		email = newEmail
	}

	if newPassword != "" {
		hashed, err := cryptoutil.HashPassword(newPassword)
		if err != nil {
			return AuthUser{}, err
		}
		hash = hashed
	}

	if _, err := s.pool.Exec(`UPDATE User SET email=?, password=?, updatedAt=NOW(3) WHERE id=?`, email, hash, userID); err != nil {
		return AuthUser{}, err
	}
	return AuthUser{ID: userID, Email: email, Name: name, CreatedAt: createdAt.UTC().Format(time.RFC3339Nano)}, nil
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// ─── HTTP handlers ─────────────────────────────────────────────────────────

type APIError struct {
	Status  int
	Message string
}

func (e *APIError) Error() string { return e.Message }

func WriteError(c *gin.Context, err error) {
	if apiErr, ok := err.(*APIError); ok {
		c.JSON(apiErr.Status, gin.H{"statusCode": apiErr.Status, "message": apiErr.Message})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"statusCode": 500, "message": "服务器内部错误"})
}

// loginRateLimit 每 IP 每分钟 5 次登录尝试（对齐 TS 版 @Throttle）
var loginRateLimit = newIPRateLimiter(5, time.Minute)

func LoginHandler(svc *Service, authCfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !loginRateLimit.Allow(c.ClientIP()) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"statusCode": 429, "message": "尝试次数过多，请稍后再试"})
			return
		}
		var dto struct {
			Email    string `json:"email" binding:"required,email"`
			Password string `json:"password" binding:"required,min=1"`
		}
		if err := c.ShouldBindJSON(&dto); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"statusCode": 400, "message": "请输入有效的邮箱地址和密码"})
			return
		}
		token, user, err := svc.Login(dto.Email, dto.Password)
		if err != nil {
			WriteError(c, err)
			return
		}
		c.SetCookie(CookieName, token, 7*24*3600, "/", "", authCfg.CookieSecure, true)
		c.JSON(http.StatusOK, gin.H{"user": user})
	}
}

func LogoutHandler(authCfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.SetCookie(CookieName, "", -1, "/", "", authCfg.CookieSecure, true)
		c.JSON(http.StatusOK, gin.H{"success": true})
	}
}

func MeHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString("userID")
		user, err := svc.GetUser(userID)
		if err != nil {
			WriteError(c, err)
			return
		}
		c.JSON(http.StatusOK, user)
	}
}

func UpdateAccountHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var dto struct {
			CurrentPassword string `json:"currentPassword" binding:"required"`
			Email           string `json:"email"`
			NewPassword     string `json:"newPassword"`
		}
		if err := c.ShouldBindJSON(&dto); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"statusCode": 400, "message": "请输入当前密码"})
			return
		}
		user, err := svc.UpdateAccount(c.GetString("userID"), dto.CurrentPassword, dto.Email, dto.NewPassword)
		if err != nil {
			WriteError(c, err)
			return
		}
		c.JSON(http.StatusOK, user)
	}
}

// ─── 中间件 ────────────────────────────────────────────────────────────────

type Claims struct {
	Sub   string `json:"sub"`
	Email string `json:"email"`
	jwt.RegisteredClaims
}

func tokenFromRequest(c *gin.Context) string {
	// cookie 优先（对齐 TS 版 JwtAuthGuard）
	if ck, err := c.Cookie(CookieName); err == nil && ck != "" {
		return ck
	}
	h := c.GetHeader("Authorization")
	if strings.HasPrefix(h, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(h, "Bearer "))
	}
	return ""
}

// Middleware 返回一个带 Public() 子路由组的鉴权器。
func Middleware(cfg *config.Config) *AuthMiddleware {
	return &AuthMiddleware{cfg: cfg}
}

type AuthMiddleware struct{ cfg *config.Config }

// Public 返回免登录组（占位中间件）。
func (m *AuthMiddleware) Public() gin.HandlerFunc {
	return func(c *gin.Context) { c.Next() }
}

func (m *AuthMiddleware) Handle(c *gin.Context) {
	token := tokenFromRequest(c)
	if token == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"statusCode": 401, "message": "请先登录"})
		return
	}
	claims := &Claims{}
	tok, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(m.cfg.JWTSecret), nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil || !tok.Valid {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"statusCode": 401, "message": "登录已失效，请重新登录"})
		return
	}
	c.Set("userID", claims.Sub)
	c.Set("email", claims.Email)
	c.Next()
}

// ─── 简单 IP 限流器（滑动窗口）────────────────────────────────────────

type ipRateLimiter struct {
	mu     sync.Mutex
	visits map[string][]time.Time
	limit  int
	window time.Duration
}

func newIPRateLimiter(limit int, window time.Duration) *ipRateLimiter {
	return &ipRateLimiter{visits: map[string][]time.Time{}, limit: limit, window: window}
}

func (l *ipRateLimiter) Allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	// 清理过期
	valid := l.visits[ip][:0]
	for _, t := range l.visits[ip] {
		if now.Sub(t) < l.window {
			valid = append(valid, t)
		}
	}
	if len(valid) >= l.limit {
		l.visits[ip] = valid
		return false
	}
	l.visits[ip] = append(valid, now)
	return true
}
