package middleware

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"groundclearance/internal/config"
	"groundclearance/internal/constants"
	"groundclearance/internal/util"
)

func Recovery(logger *slog.Logger) gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered any) {
		logger.Error("panic recovered", "error", fmt.Sprint(recovered), "path", c.FullPath())
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"code": constants.CodeInternalError, "message": constants.MsgInternalError, "data": nil,
		})
	})
}

// CORS 跨域配置，来源从环境变量 APP_CORS_ORIGINS 读取，生产默认不允许 *。
func CORS(cfg *config.Config) gin.HandlerFunc {
	origins := cfg.CORSOrigins
	if len(origins) == 0 {
		origins = []string{"http://localhost:18504"}
	}
	return cors.New(cors.Config{
		AllowOrigins:     origins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-Request-Id"},
		ExposeHeaders:    []string{"X-Request-Id"},
		AllowCredentials: false,
	})
}

// ErrorHandler 统一错误响应格式。
func ErrorHandler(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		if len(c.Errors) == 0 {
			return
		}
		err := c.Errors.Last().Err
		var appErr *util.AppError
		if errors.As(err, &appErr) {
			status := appErrorStatus(appErr.Code)
			c.JSON(status, gin.H{"code": appErr.Code, "message": appErr.Message, "data": nil})
			return
		}
		logger.Error("unhandled error", "error", err.Error(), "path", c.FullPath())
		c.JSON(http.StatusInternalServerError, gin.H{"code": constants.CodeInternalError, "message": constants.MsgInternalError, "data": nil})
	}
}

func appErrorStatus(code int) int {
	switch code {
	case constants.CodeUnauthorized, constants.CodeInvalidCredentials:
		return http.StatusUnauthorized
	case constants.CodeForbidden:
		return http.StatusForbidden
	case constants.CodeNotFound:
		return http.StatusNotFound
	case constants.CodeConflict, constants.CodeStateConflict:
		return http.StatusConflict
	case constants.CodeValidationFailed:
		return http.StatusUnprocessableEntity
	case constants.CodeTooManyRequests:
		return http.StatusTooManyRequests
	default:
		return http.StatusBadRequest
	}
}

type bucket struct {
	tokens   float64
	lastFill time.Time
}

// RateLimiter uses Redis for a cross-instance fixed window and keeps a local
// token bucket as a degraded fallback when Redis is temporarily unavailable.
type RateLimiter struct {
	mu       sync.Mutex
	perMin   int
	buckets  map[string]*bucket
	capacity int
	redis    *redis.Client
}

func NewRateLimiter(perMin int, client *redis.Client) *RateLimiter {
	if perMin <= 0 {
		perMin = 120
	}
	return &RateLimiter{perMin: perMin, buckets: make(map[string]*bucket), capacity: perMin, redis: client}
}

func (rl *RateLimiter) Limit() gin.HandlerFunc {
	return func(c *gin.Context) {
		allowed, err := rl.allowDistributed(c)
		if err != nil {
			allowed = rl.allowLocal(c.ClientIP())
		}
		if !allowed {
			rejectRateLimit(c)
			return
		}
		c.Next()
	}
}

var rateLimitScript = redis.NewScript(`
local current = redis.call("INCR", KEYS[1])
if current == 1 then
  redis.call("EXPIRE", KEYS[1], ARGV[1])
end
return current
`)

func (rl *RateLimiter) allowDistributed(c *gin.Context) (bool, error) {
	if rl.redis == nil {
		return false, errors.New("redis client unavailable")
	}
	window := strconv.FormatInt(time.Now().Unix()/60, 10)
	key := "ground-clearance:rate:" + c.ClientIP() + ":" + window
	count, err := rateLimitScript.Run(c.Request.Context(), rl.redis, []string{key}, 90).Int64()
	if err != nil {
		return false, err
	}
	c.Header("X-RateLimit-Limit", strconv.Itoa(rl.perMin))
	remaining := int64(rl.perMin) - count
	if remaining < 0 {
		remaining = 0
	}
	c.Header("X-RateLimit-Remaining", strconv.FormatInt(remaining, 10))
	return count <= int64(rl.perMin), nil
}

func (rl *RateLimiter) allowLocal(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	now := time.Now()
	b, ok := rl.buckets[ip]
	if !ok {
		b = &bucket{tokens: float64(rl.capacity), lastFill: now}
		rl.buckets[ip] = b
	}
	elapsed := now.Sub(b.lastFill)
	b.tokens += elapsed.Seconds() * float64(rl.perMin) / 60
	if b.tokens > float64(rl.capacity) {
		b.tokens = float64(rl.capacity)
	}
	b.lastFill = now
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

func rejectRateLimit(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
		"code": constants.CodeTooManyRequests, "message": constants.MsgTooManyRequests, "data": nil,
	})
}
