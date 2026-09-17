package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"groundclearance/internal/constants"
	"groundclearance/internal/model"
	"log/slog"
	"regexp"
	"strings"
	"time"
)

// AuditLog 操作审计日志中间件。
func AuditLog(db *gorm.DB, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == "GET" || c.Request.Method == "OPTIONS" {
			c.Next()
			return
		}
		c.Next()
		if c.Writer.Status() >= 400 {
			return
		}
		if persisted, _ := c.Get("audit_persisted"); persisted == true {
			return
		}
		path := c.FullPath()
		entityType := "unknown"
		seg := strings.Split(strings.TrimPrefix(path, "/api/v1/"), "/")
		if len(seg) > 0 && seg[0] != "" {
			entityType = seg[0]
		}
		detail := map[string]any{"method": c.Request.Method, "path": path}
		if b, ok := c.Get("audit_detail"); ok {
			detail["body"] = b
		}
		raw, _ := json.Marshal(detail)
		entry := &model.AuditLog{
			OperatorID: GetUserID(c), OperatorName: GetPhone(c),
			Action: c.Request.Method, EntityType: entityType,
			EntityID: c.Param("id"), Detail: string(raw), IP: c.ClientIP(), CreatedAt: time.Now(),
		}
		if err := db.Create(entry).Error; err != nil {
			logger.Error(constants.LogAuditWriteFailed, "error", err.Error())
		}
	}
}

const requestIDHeader = "X-Request-Id"

var requestIDPattern = regexp.MustCompile(`^[A-Za-z0-9._:-]{1,64}$`)

// RequestID 为每个请求生成 request_id，响应头与日志统一携带。
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(requestIDHeader)
		if !requestIDPattern.MatchString(id) {
			id = newRequestID()
		}
		c.Header(requestIDHeader, id)
		c.Set("request_id", id)
		c.Next()
	}
}

func newRequestID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "req-0000000000000001"
	}
	return hex.EncodeToString(b[:])
}

// RequestLogger 请求日志。
func RequestLogger(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		rid := c.Writer.Header().Get("X-Request-Id")
		if v, ok := c.Get("request_id"); ok {
			if s, ok := v.(string); ok && s != "" {
				rid = s
			}
		}
		logger.Info("http request", "request_id", rid,
			"method", c.Request.Method, "path", c.FullPath(),
			"status", c.Writer.Status(), "latency_ms", time.Since(start).Milliseconds())
	}
}
