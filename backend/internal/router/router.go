package router

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"groundclearance/internal/config"
	"groundclearance/internal/handler"
	"groundclearance/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type Router struct {
	cfg        *config.Config
	db         *gorm.DB
	logger     *slog.Logger
	limiter    *middleware.RateLimiter
	redis      *redis.Client
	user       *handler.UserHandler
	turnaround *handler.TurnaroundHandler
	groundUnit *handler.GroundUnitHandler
	check      *handler.SafetyCheckHandler
	clearance  *handler.ClearanceDecisionHandler
	audit      *handler.AuditLogHandler
}

func New(cfg *config.Config, db *gorm.DB, redisClient *redis.Client, logger *slog.Logger, user *handler.UserHandler,
	turnaround *handler.TurnaroundHandler, groundUnit *handler.GroundUnitHandler,
	check *handler.SafetyCheckHandler, clearance *handler.ClearanceDecisionHandler,
	audit *handler.AuditLogHandler) *Router {
	return &Router{cfg: cfg, db: db, logger: logger, redis: redisClient, limiter: middleware.NewRateLimiter(cfg.RateLimitPerMinute, redisClient),
		user: user, turnaround: turnaround, groundUnit: groundUnit, check: check, clearance: clearance, audit: audit}
}

func (r *Router) Setup() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	// Do not trust client-supplied forwarding headers on the directly exposed API.
	// Deployments behind a trusted load balancer should terminate access there.
	_ = engine.SetTrustedProxies(nil)
	engine.Use(middleware.Recovery(r.logger), middleware.RequestID(), middleware.RequestLogger(r.logger),
		middleware.ErrorHandler(r.logger), middleware.CORS(r.cfg), middleware.JWTConfig(r.cfg), middleware.AuditLog(r.db, r.logger))
	engine.GET("/healthz", r.health)
	engine.GET("/api/healthz", r.health)
	v1 := engine.Group("/api/v1")
	v1.Use(r.limiter.Limit())
	r.registerAuthRoutes(v1)
	r.registerUserRoutes(v1)
	r.registerTurnaroundRoutes(v1)
	r.registerGroundUnitRoutes(v1)
	r.registerCheckRoutes(v1)
	r.registerClearanceRoutes(v1)
	r.registerAuditRoutes(v1)
	return engine
}

func (r *Router) health(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()
	dependencies := gin.H{"postgres": "ok", "redis": "ok"}
	status := http.StatusOK
	statusText := "ok"
	sqlDB, err := r.db.DB()
	if err != nil || sqlDB.PingContext(ctx) != nil {
		dependencies["postgres"] = "error"
		status = http.StatusServiceUnavailable
		statusText = "degraded"
	}
	if r.redis == nil || r.redis.Ping(ctx).Err() != nil {
		dependencies["redis"] = "error"
		status = http.StatusServiceUnavailable
		statusText = "degraded"
	}
	c.JSON(status, gin.H{"status": statusText, "service": "airport-ground-equipment-clearance", "dependencies": dependencies})
}

func (r *Router) registerAuthRoutes(group *gin.RouterGroup) {
	auth := group.Group("/auth")
	auth.POST("/login", r.user.Login)
}
