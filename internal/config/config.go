package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

const devFallbackSecret = "dev-only-insecure-secret"

type Config struct {
	Port                    int
	NodeEnv                 string
	DatabaseURL             string
	RedisURL                string
	CredentialEncryptionKey string
	JWTSecret               string
	JWTExpiresIn            string // e.g. "7d"
	MTeamBaseURL            string
	MTeamTimeoutMs          int
	MTeamMaxRetries         int
	MTeamRateLimitPerMin    int
	CorsOrigin              string
	CookieSecure            bool
}

func env(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

// requireJWTSecret mirrors the TS behaviour: fail hard in production.
func requireJWTSecret(nodeEnv string) (string, error) {
	if s := strings.TrimSpace(os.Getenv("JWT_SECRET")); s != "" {
		return s, nil
	}
	if nodeEnv == "production" {
		return "", errors.New("JWT_SECRET is required in production. Generate one with: openssl rand -hex 32")
	}
	fmt.Fprintln(os.Stderr, "⚠ JWT_SECRET 未设置，正在使用开发占位 secret。请在 .env 中配置：openssl rand -hex 32")
	return devFallbackSecret, nil
}

func Load() (*Config, error) {
	nodeEnv := env("NODE_ENV", "development")
	jwtSecret, err := requireJWTSecret(nodeEnv)
	if err != nil {
		return nil, err
	}

	port := 4000
	if p := os.Getenv("BACKEND_PORT"); p != "" {
		if n, err := strconv.Atoi(p); err == nil {
			port = n
		}
	}

	timeoutMs := 30000
	if t := os.Getenv("MTEAM_TIMEOUT_MS"); t != "" {
		if n, err := strconv.Atoi(t); err == nil {
			timeoutMs = n
		}
	}
	maxRetries := 3
	if m := os.Getenv("MTEAM_MAX_RETRIES"); m != "" {
		if n, err := strconv.Atoi(m); err == nil {
			maxRetries = n
		}
	}
	rateLimit := 30
	if r := os.Getenv("MTEAM_RATE_LIMIT_PER_MINUTE"); r != "" {
		if n, err := strconv.Atoi(r); err == nil {
			rateLimit = n
		}
	}

	cfg := &Config{
		Port:                    port,
		NodeEnv:                 nodeEnv,
		DatabaseURL:             env("DATABASE_URL", ""),
		RedisURL:                env("REDIS_URL", "redis://localhost:6379"),
		CredentialEncryptionKey: env("CREDENTIAL_ENCRYPTION_KEY", ""),
		JWTSecret:               jwtSecret,
		JWTExpiresIn:            env("JWT_EXPIRES_IN", "7d"),
		MTeamBaseURL:            env("MTEAM_BASE_URL", "https://api.m-team.cc"),
		MTeamTimeoutMs:          timeoutMs,
		MTeamMaxRetries:         maxRetries,
		MTeamRateLimitPerMin:    rateLimit,
		CorsOrigin:              env("CORS_ORIGIN", "http://localhost:3000"),
		CookieSecure:            env("COOKIE_SECURE", "false") == "true",
	}
	if cfg.DatabaseURL == "" {
		return nil, errors.New("DATABASE_URL is required")
	}
	return cfg, nil
}
