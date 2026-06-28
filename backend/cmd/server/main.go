// Sentinel API server — entry point.
package main

import (
	"context"
	"encoding/json"
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

	"github.com/glimjoe/sentinel-api-platform/internal/ai"
	"github.com/glimjoe/sentinel-api-platform/internal/api"
	"github.com/glimjoe/sentinel-api-platform/internal/middleware"
	"github.com/glimjoe/sentinel-api-platform/internal/mock"
	"github.com/glimjoe/sentinel-api-platform/internal/model"
	"github.com/glimjoe/sentinel-api-platform/internal/runner"
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
	// ADR-0008: Secure cookies only in production.
	middleware.SetCookieConfig(cfg.App.Env == "production", "")

	r := gin.New()
	r.Use(middleware.RequestID())
	r.Use(middleware.AccessLog(log))
	r.Use(middleware.Recovery(log))
	r.Use(middleware.CORS(cfg.App.FrontendOrigin))

	// Audit logging (Phase 2 M2).
	auditRepo := repository.NewAuditLogRepo(db)
	r.Use(middleware.Audit(auditRepo))

	healthH := &api.HealthHandler{DB: db, Redis: rdb}
	r.GET("/api/v1/healthz", healthH.Healthz)
	r.GET("/api/v1/readyz", healthH.Readyz)

	// Auth wiring (Phase 1, cookie-based per ADR-0008).
	userRepo := repository.NewUserRepo(db)
	refreshTokenRepo := repository.NewRefreshTokenRepo(db)
	authSvc := service.NewAuthService(userRepo, refreshTokenRepo, cfg.Auth.AccessSecret, cfg.Auth.AccessTTL, cfg.Auth.BcryptCost)
	authH := api.NewAuthHandler(authSvc)

	// Public auth endpoints (no CSRF).
	r.POST("/api/v1/auth/register", authH.Register)
	r.POST("/api/v1/auth/login", authH.Login)
	r.POST("/api/v1/auth/refresh", authH.Refresh)

	// Protected group — CSRF → cookie auth (ADR-0008).
	protected := r.Group("/api/v1")
	protected.Use(middleware.CSRFRequired())
	protected.Use(middleware.CookieAuthRequired(cfg.Auth.AccessSecret))
	protected.GET("/auth/me", authH.Me)
	protected.POST("/auth/logout", authH.Logout)

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

	// API + MockRule construction (M2-F.C/D). RBAC for API routes is enforced
	// through RequireProjectRole middleware; MockRule routes delegate RBAC to
	// the service layer because the rule ID alone doesn't carry project context.
	apiRepo := repository.NewAPIRepo(db)
	apiSvc := service.NewAPIService(apiRepo, projectSvc)
	apiH := api.NewAPIHandler(apiSvc)
	mockRuleRepo := repository.NewMockRuleRepo(db)
	mockRuleSvc := service.NewMockRuleService(mockRuleRepo, projectSvc)
	mockRuleH := api.NewMockRuleHandler(mockRuleSvc)

	// API routes (M2-F.C) — scoped under projects/:pid with RBAC.
	protected.POST("/projects/:pid/apis", middleware.RequireProjectRole(projectSvc,
		model.ProjectRoleAdmin, model.ProjectRoleEngineer),
		apiH.CreateAPI)
	protected.GET("/projects/:pid/apis", middleware.RequireProjectRole(projectSvc,
		model.ProjectRoleAdmin, model.ProjectRoleEngineer, model.ProjectRoleViewer),
		apiH.ListAPIs)
	protected.GET("/projects/:pid/apis/:apiId", middleware.RequireProjectRole(projectSvc,
		model.ProjectRoleAdmin, model.ProjectRoleEngineer, model.ProjectRoleViewer),
		apiH.GetAPI)
	protected.PATCH("/projects/:pid/apis/:apiId", middleware.RequireProjectRole(projectSvc,
		model.ProjectRoleAdmin, model.ProjectRoleEngineer),
		apiH.UpdateAPI)
	protected.DELETE("/projects/:pid/apis/:apiId", middleware.RequireProjectRole(projectSvc,
		model.ProjectRoleAdmin),
		apiH.DeleteAPI)
	protected.POST("/projects/:pid/apis/:apiId/tags", middleware.RequireProjectRole(projectSvc,
		model.ProjectRoleAdmin, model.ProjectRoleEngineer),
		apiH.AddTag)
	protected.DELETE("/projects/:pid/apis/:apiId/tags/:tag", middleware.RequireProjectRole(projectSvc,
		model.ProjectRoleAdmin, model.ProjectRoleEngineer),
		apiH.RemoveTag)
		protected.POST("/projects/:pid/apis/import-openapi", middleware.RequireProjectRole(projectSvc,
			model.ProjectRoleAdmin, model.ProjectRoleEngineer),
			apiH.ImportOpenAPI)

	// MockRule routes (M2-F.D) — RBAC enforced at service layer.
	protected.POST("/rules", mockRuleH.CreateRule)
	protected.GET("/rules/:rid", mockRuleH.GetRule)
	protected.PATCH("/rules/:rid", mockRuleH.UpdateRule)
	protected.DELETE("/rules/:rid", mockRuleH.DeleteRule)
	protected.GET("/apis/:apiId/rules", mockRuleH.ListRules)
	protected.POST("/rules/:rid/hits", mockRuleH.RecordHit)
	protected.GET("/rules/:rid/hits", mockRuleH.ListHits)

	// TestCase + TestRun wiring (Phase 3).
	testCaseRepo := repository.NewTestCaseRepo(db)
	testResultRepo := repository.NewTestResultRepo(db)
	testRunRepo := repository.NewTestRunRepo(db)
	broker := runner.NewRedisEventBroker(rdb)
	testCaseSvc := service.NewTestCaseService(testCaseRepo, projectSvc)
	testRunSvc := service.NewTestRunService(testRunRepo, testCaseRepo, testResultRepo, projectSvc, broker)
	testCaseH := api.NewTestCaseHandler(testCaseSvc)
	testRunH := api.NewTestRunHandler(testRunSvc, broker)

	protected.POST("/projects/:pid/cases", middleware.RequireProjectRole(projectSvc,
		model.ProjectRoleAdmin, model.ProjectRoleEngineer),
		testCaseH.Create)
	protected.GET("/projects/:pid/cases", middleware.RequireProjectRole(projectSvc,
		model.ProjectRoleAdmin, model.ProjectRoleEngineer, model.ProjectRoleViewer),
		testCaseH.List)
	protected.GET("/projects/:pid/cases/:caseId", middleware.RequireProjectRole(projectSvc,
		model.ProjectRoleAdmin, model.ProjectRoleEngineer, model.ProjectRoleViewer),
		testCaseH.Get)
	protected.PATCH("/projects/:pid/cases/:caseId", middleware.RequireProjectRole(projectSvc,
		model.ProjectRoleAdmin, model.ProjectRoleEngineer),
		testCaseH.Update)
	protected.DELETE("/projects/:pid/cases/:caseId", middleware.RequireProjectRole(projectSvc,
		model.ProjectRoleAdmin, model.ProjectRoleEngineer),
		testCaseH.Delete)
	protected.POST("/projects/:pid/runs", middleware.RequireProjectRole(projectSvc,
		model.ProjectRoleAdmin, model.ProjectRoleEngineer),
		testRunH.Create)
	protected.POST("/projects/:pid/runs/:runId/start", middleware.RequireProjectRole(projectSvc,
		model.ProjectRoleAdmin, model.ProjectRoleEngineer),
		testRunH.Start)
	protected.GET("/projects/:pid/runs", middleware.RequireProjectRole(projectSvc,
		model.ProjectRoleAdmin, model.ProjectRoleEngineer, model.ProjectRoleViewer),
		testRunH.List)
	protected.GET("/projects/:pid/runs/:runId", middleware.RequireProjectRole(projectSvc,
		model.ProjectRoleAdmin, model.ProjectRoleEngineer, model.ProjectRoleViewer),
		testRunH.Get)
		protected.GET("/projects/:pid/runs/:runId/results", middleware.RequireProjectRole(projectSvc,
			model.ProjectRoleAdmin, model.ProjectRoleEngineer, model.ProjectRoleViewer),
			testRunH.Export)
		protected.POST("/projects/:pid/runs/:runId/cancel", middleware.RequireProjectRole(projectSvc,
			model.ProjectRoleAdmin, model.ProjectRoleEngineer),
			testRunH.Cancel)
	// SSE stream uses URL query-param auth so EventSource can connect.
	// EventSource does not support custom headers; JWT is passed as ?token=.
	r.GET("/api/v1/projects/:pid/runs/:runId/stream",
		middleware.TokenQueryAuth(cfg.Auth.AccessSecret),
		middleware.RequireProjectRole(projectSvc,
			model.ProjectRoleAdmin, model.ProjectRoleEngineer, model.ProjectRoleViewer),
		testRunH.Stream)

	// AI module wiring (Phase 4).
	aiRepo := repository.NewAiRepo(db)
	aiProvider := ai.NewProvider(cfg.AI.Provider, cfg.AI.AnthropicKey, cfg.AI.OpenAIKey)
	aiCache := ai.NewCache(rdb, time.Duration(cfg.AI.CacheTTLSeconds)*time.Second)
	aiGuard := ai.NewGuard(aiRepo, cfg.AI.DailyLimitUSD, cfg.AI.MonthlyLimitUSD)
	temp := cfg.AI.Temperature
if temp == 0 { temp = 0.3 }
aiEngine := ai.NewEngine(aiProvider, aiCache, aiGuard, cfg.AI.MaxTokens, temp)
	aiSvc := service.NewAIService(projectSvc, ai.NewAttributor(aiEngine), ai.NewCompleter(aiEngine), ai.NewPrioritizer(aiEngine), apiRepo, testCaseRepo, aiGuard)

	// Attribution hook: runs AI attribution on failed/errored test results
	// after each test case executes, before the result is persisted.
	attributionHook := func(ctx context.Context, run *model.TestRun, result *model.TestResult) {
		if result.Status != "fail" && result.Status != "error" {
			return
		}
		resultJSON, err := json.Marshal(result)
		if err != nil {
			log.Error("attribution hook: marshal result", zap.Error(err))
			return
		}
		callerID := "system"
		if run.TriggeredBy != nil {
			callerID = *run.TriggeredBy
		}
		attribution, err := aiSvc.Attribute(ctx, callerID, run.ProjectID, string(resultJSON))
		if err != nil {
			log.Warn("attribution hook: attribute", zap.Error(err))
			return
		}
		attrJSON, err := json.Marshal(attribution)
		if err != nil {
			log.Error("attribution hook: marshal attribution", zap.Error(err))
			return
		}
		result.AIAttributionJSON = attrJSON
	}
	testRunSvc.SetPostExecuteHook(attributionHook)

	aiH := api.NewAIHandler(aiSvc)

	if cfg.AI.Enabled {
		protected.POST("/projects/:pid/ai/attribution", middleware.RequireProjectRole(projectSvc,
			model.ProjectRoleAdmin, model.ProjectRoleEngineer),
			aiH.Attribute)
		protected.POST("/projects/:pid/ai/complete", middleware.RequireProjectRole(projectSvc,
			model.ProjectRoleAdmin, model.ProjectRoleEngineer),
			aiH.Complete)
		protected.POST("/projects/:pid/ai/prioritize", middleware.RequireProjectRole(projectSvc,
			model.ProjectRoleAdmin, model.ProjectRoleEngineer),
			aiH.Prioritize)
	}
	protected.GET("/ai/budget", aiH.Budget)

	// Mock engine wiring (Phase 2 M1). The public /mock/:projectSlug/*path
	// route is registered OUTSIDE the protected group — anyone with the
	// project slug can hit it (this is the whole point of a mock server).
	mockHitRepo := repository.NewMockHitRepo(db)
	varbag := mock.NewRedisVarBag(rdb)
	matcher := mock.NewMatcher()
	extractor := mock.NewExtractor()
	mockEngine := mock.NewEngine(projectRepo, apiRepo, mockRuleRepo, varbag, matcher, extractor, mockHitRepo)
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
