package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/1443205008/ptmanager-go/internal/accounts"
	"github.com/1443205008/ptmanager-go/internal/alerts"
	"github.com/1443205008/ptmanager-go/internal/auth"
	"github.com/1443205008/ptmanager-go/internal/config"
	"github.com/1443205008/ptmanager-go/internal/cryptoutil"
	"github.com/1443205008/ptmanager-go/internal/dashboard"
	"github.com/1443205008/ptmanager-go/internal/db"
	"github.com/1443205008/ptmanager-go/internal/fakeseed"
	"github.com/1443205008/ptmanager-go/internal/mteam"
	"github.com/1443205008/ptmanager-go/internal/providers"
	"github.com/1443205008/ptmanager-go/internal/search"
	"github.com/1443205008/ptmanager-go/internal/settings"
	"github.com/1443205008/ptmanager-go/internal/syncer"
	"github.com/1443205008/ptmanager-go/internal/torrents"
	"github.com/1443205008/ptmanager-go/internal/web"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	if cfg.NodeEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	pool, err := db.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer pool.Close()

	rdb := db.NewRedis(cfg.RedisURL)

	// 迁移（幂等：已存在的表跳过）
	if err := db.Migrate(context.Background(), pool); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	cryptoSvc, err := cryptoutil.NewCredentialCrypto(cfg.CredentialEncryptionKey)
	if err != nil {
		log.Fatalf("crypto: %v", err)
	}

	registry := providers.NewRegistry()
	mteamProv := mteam.NewProvider(cfg, pool, cryptoSvc)
	registry.Register("MTEAM", mteamProv)

	authSvc := auth.NewService(pool, cfg)
	settingsSvc := settings.NewService(pool, rdb, cfg)
	syncSvc := syncer.NewService(pool, registry, settingsSvc)
	accountsSvc := accounts.NewService(pool, cryptoSvc, registry)
	dashSvc := dashboard.NewService(pool, accountsSvc)
	torrentSvc := torrents.NewService(pool)
	searchSvc := search.NewService(pool, registry)
	alertsSvc := alerts.NewService(pool)
	fakeSvc := fakeseed.NewService(pool, registry, cfg)

	// 队列 worker + 定时调度
	queue := syncer.NewQueue(rdb, syncSvc)
	go queue.RunWorker(context.Background())
	scheduler := syncer.NewScheduler(pool, queue, settingsSvc)
	go scheduler.Run(context.Background())
	go fakeseed.RestoreRunning(pool, fakeSvc)

	// seed（幂等）
	if os.Getenv("RUN_SEED") == "true" {
		if err := db.Seed(context.Background(), pool, cfg); err != nil {
			log.Printf("seed skipped: %v", err)
		}
	}

	r := gin.New()
	r.Use(gin.LoggerWithWriter(gin.DefaultWriter), gin.Recovery())
	r.Use(corsMiddleware(cfg.CorsOrigin))

	// 全局鉴权中间件
	authMW := auth.Middleware(cfg)

	api := r.Group("/api/v1")
	public := api.Group("", authMW.Public())
	{
		pubAuth := public.Group("/auth")
		pubAuth.POST("/login", auth.LoginHandler(authSvc, cfg))
		pubAuth.POST("/logout", auth.LogoutHandler())
	}

	secured := api.Group("", authMW.Handle)
	{
		secured.GET("/auth/me", auth.MeHandler(authSvc))

		acc := secured.Group("/accounts")
		acc.POST("", accounts.CreateHandler(accountsSvc))
		acc.GET("", accounts.ListHandler(accountsSvc))
		acc.GET("/:id", accounts.GetHandler(accountsSvc))
		acc.PATCH("/:id", accounts.UpdateHandler(accountsSvc))
		acc.DELETE("/:id", accounts.DeleteHandler(accountsSvc))
		acc.POST("/:id/test-connection", accounts.TestConnectionHandler(accountsSvc))

		sy := secured.Group("/sync")
		sy.POST("/accounts/:id", syncer.TriggerAccountHandler(pool, queue))
		sy.POST("/all", syncer.TriggerAllHandler(pool, queue))
		sy.POST("/accounts/:id/torrents", syncer.TriggerTorrentHandler(queue))

		secured.GET("/dashboard", dashboard.Handler(dashSvc))

		tor := secured.Group("/torrents")
		tor.GET("", torrents.ListHandler(torrentSvc))

		se := secured.Group("/search")
		se.GET("/torrents", search.SearchTorrentsHandler(searchSvc))
		se.GET("/teams", search.TeamsHandler(searchSvc))
		se.POST("/dl-token", search.DlTokenHandler(searchSvc))

		al := secured.Group("/alerts")
		al.GET("", alerts.ListHandler(alertsSvc))
		al.GET("/unread-count", alerts.UnreadCountHandler(alertsSvc))
		al.PATCH("/read-all", alerts.MarkAllReadHandler(alertsSvc))
		al.PATCH("/:id/read", alerts.MarkReadHandler(alertsSvc))
		al.DELETE("/:id", alerts.DismissHandler(alertsSvc))

		fs := secured.Group("/fake-seed")
		fs.GET("", fakeseed.ListHandler(fakeSvc))
		fs.POST("", fakeseed.StartHandler(fakeSvc))
		fs.DELETE("/:id/stop", fakeseed.StopHandler(fakeSvc))
		fs.DELETE("/:id", fakeseed.DeleteHandler(fakeSvc))

		st := secured.Group("/settings")
		st.GET("", settings.GetHandler(settingsSvc))
		st.PATCH("", settings.UpdateHandler(settingsSvc))
		st.POST("/reset", settings.ResetHandler(settingsSvc))
		st.GET("/status", settings.StatusHandler(settingsSvc))
		st.POST("/cleanup", settings.CleanupHandler(settingsSvc))
	}

	// 静态前端（go:embed）
	web.Register(r, "/api")

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Port),
		Handler:           r,
		ReadHeaderTimeout: 15 * time.Second,
	}

	go func() {
		log.Printf("PT Manager (Go) backend running on http://localhost:%d/api/v1", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}

func corsMiddleware(origin string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if origin == "" {
			origin = "http://localhost:3000"
		}
		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

var _ = strings.TrimSpace // keep import if unused later
