// Sentinel API server — entry point.
package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/glimjoe/sentinel-api-platform/internal/api"
	"github.com/glimjoe/sentinel-api-platform/internal/middleware"
	"github.com/glimjoe/sentinel-api-platform/internal/mock"
	"github.com/glimjoe/sentinel-api-platform/internal/model"
	"github.com/glimjoe/sentinel-api-platform/internal/pkg/config"
	"github.com/glimjoe/sentinel-api-platform/internal/pkg/logger"
	"github.com/glimjoe/sentinel-api-platform/internal/repository"
	"github.com/glimjoe/sentinel-api-platform/internal/service"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// 1. Config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// 2. Logger
	log, err := logger.New(cfg.Logging.Level, cfg.Logging.Format)
	if err != nil {
		return fmt.Errorf("init logger: %w", err)
	}
	defer log.Sync() //nolint:errcheck

	log.Info("starting sentinel",
		zap.String("env", cfg.App.Env),
		zap.Int("port", cfg.App.Port),
		zap.String("version", "0.1.0"),
	)

	// 3. MySQL
	db, err := gorm.Open(mysql.Open(cfg.MySQL.DSN()), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("connect mysql: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("get sql.DB: %w", err)
	}
	sqlDB.SetMaxOpenConns(cfg.MySQL.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MySQL.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.MySQL.ConnMaxLifetime)
	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("ping mysql: %w", err)
	}
	log.Info("mysql connected", zap.String("dsn_host", cfg.MySQL.Host), zap.Int("dsn_port", cfg.MySQL.Port))

	// 4. Redis
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr(),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	pingCtx, cancelPing := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancelPing()
	if err := rdb.Ping(pingCtx).Err(); err != nil {
		return fmt.Errorf("ping redis: %w", err)
	}
	log.Info("redis connected", zap.String("addr", cfg.Redis.Addr()))

	// 5. Gin
	if cfg.App.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.Use(middleware.RequestID())
	r.Use(middleware.AccessLog(log))
	r.Use(middleware.Recovery(log))
	r.Use(middleware.CORS(cfg.App.FrontendOrigin))

	healthH := &api.HealthHandler{DB: db, Redis: rdb}
	r.GET("/api/v1/healthz", healthH.Healthz)
	r.GET("/api/v1/readyz", healthH.Readyz)

	// Auth wiring (Phase 1).
	userRepo := repository.NewUserRepo(db)
	authSvc := service.NewAuthService(userRepo, cfg.Auth.AccessSecret, cfg.Auth.AccessTTL, cfg.Auth.BcryptCost)
	authH := api.NewAuthHandler(authSvc)
	r.POST("/api/v1/auth/register", authH.Register)
	r.POST("/api/v1/auth/login", authH.Login)
	protected := r.Group("/api/v1", middleware.AuthRequired(cfg.Auth.AccessSecret))
	protected.GET("/auth/me", authH.Me)

	// Project wiring (M2-F.A + M2-F.E).
	projectRepo := repository.NewProjectRepo(db)
	projectMemberRepo := repository.NewProjectMemberRepo(db)
	projectSvc := service.NewProjectService(projectRepo, projectMemberRepo)
	projectH := api.NewProjectHandler(projectSvc)

	protected.POST("/projects", projectH.CreateProject)
	protected.GET("/projects", projectH.ListProjects)
	protected.GET("/projects/:pid", middleware.RequireProjectRole(projectSvc,
		model.ProjectRoleAdmin, model.ProjectRoleEngineer, model.ProjectRoleViewer),
		projectH.GetProject)
	protected.PATCH("/projects/:pid", middleware.RequireProjectRole(projectSvc,
		model.ProjectRoleAdmin),
		projectH.UpdateProject)
	protected.DELETE("/projects/:pid", middleware.RequireProjectRole(projectSvc,
		model.ProjectRoleAdmin, model.ProjectRoleEngineer, model.ProjectRoleViewer),
		projectH.DeleteProject)
	protected.GET("/projects/:pid/members", middleware.RequireProjectRole(projectSvc,
		model.ProjectRoleAdmin, model.ProjectRoleEngineer, model.ProjectRoleViewer),
		projectH.ListMembers)
	protected.POST("/projects/:pid/members", middleware.RequireProjectRole(projectSvc,
		model.ProjectRoleAdmin),
		projectH.AddMember)

	// API + MockRule services (M2-F.C/D — services constructed, handlers
	// pending; the route mounting will land when those handlers exist).
	apiRepo := repository.NewAPIRepo(db)
	mockRuleRepo := repository.NewMockRuleRepo(db)
	_ = service.NewAPIService(apiRepo, projectSvc)          // handler: M2-F.C
	_ = service.NewMockRuleService(mockRuleRepo, projectSvc) // handler: M2-F.D

	// Mock engine wiring (Phase 2 M1). The public /mock/:projectSlug/*path
	// route is registered OUTSIDE the protected group — anyone with the
	// project slug can hit it (this is the whole point of a mock server).
	_ = repository.NewMockHitRepo(db) // M2: wire as HitRecorder
	varbag := mock.NewRedisVarBag(rdb)
	matcher := mock.NewMatcher()
	extractor := mock.NewExtractor()
	mockEngine := mock.NewEngine(projectRepo, apiRepo, mockRuleRepo, varbag, matcher, extractor, nil)
	r.Any("/mock/:projectSlug/*path", mockEngine.Serve)

	// 6. HTTP server
	srv := &http.Server{
		Addr:              fmt.Sprintf("%s:%d", cfg.App.Host, cfg.App.Port),
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
	}

	// 7. Graceful shutdown
	errCh := make(chan error, 1)
	go func() {
		log.Info("http server listening", zap.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	select {
	case sig := <-quit:
		log.Info("shutdown signal received", zap.String("signal", sig.String()))
	case err := <-errCh:
		return fmt.Errorf("http server: %w", err)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown: %w", err)
	}
	log.Info("shutdown complete")
	return nil
}
