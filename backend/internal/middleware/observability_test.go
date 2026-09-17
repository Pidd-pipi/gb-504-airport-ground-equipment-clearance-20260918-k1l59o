package middleware

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestRequestIDRejectsUnboundedClientValue(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(RequestID())
	engine.GET("/", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set(requestIDHeader, "invalid request id with spaces and an unsafe length that must never reach audit storage")
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	requestID := response.Header().Get(requestIDHeader)
	if !requestIDPattern.MatchString(requestID) || requestID == request.Header.Get(requestIDHeader) {
		t.Fatalf("unsafe request id was not replaced: %q", requestID)
	}
}

func TestLocalRateLimiterRefillsFractionalMinutes(t *testing.T) {
	limiter := NewRateLimiter(60, nil)
	limiter.buckets["127.0.0.1"] = &bucket{tokens: 0, lastFill: time.Now().Add(-1100 * time.Millisecond)}
	if !limiter.allowLocal("127.0.0.1") {
		t.Fatal("local fallback did not refill one token after one second")
	}
	if limiter.allowLocal("127.0.0.1") {
		t.Fatal("local fallback granted a second token without enough elapsed time")
	}
}

func TestRecoveryUsesStableErrorEnvelope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(Recovery(slog.New(slog.NewTextHandler(io.Discard, nil))))
	engine.GET("/panic", func(c *gin.Context) { panic("sensitive internal detail") })
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/panic", nil))
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("unexpected status: %d", response.Code)
	}
	if body := response.Body.String(); body != `{"code":50000,"data":null,"message":"服务器内部错误"}` {
		t.Fatalf("unexpected recovery body: %s", body)
	}
}
