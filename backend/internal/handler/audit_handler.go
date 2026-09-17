package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"groundclearance/internal/constants"
	"groundclearance/internal/dto"
	"groundclearance/internal/middleware"
	"groundclearance/internal/model"
	"groundclearance/internal/service"
	"groundclearance/internal/util"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AuditLogHandler struct {
	db     *gorm.DB
	logger *slog.Logger
}

func NewAuditLogHandler(db *gorm.DB, logger *slog.Logger) *AuditLogHandler {
	return &AuditLogHandler{db: db, logger: logger}
}

func (h *AuditLogHandler) List(c *gin.Context) {
	var query dto.PageQuery
	if !bindPageQuery(c, &query) {
		return
	}
	var rows []model.AuditLog
	var total int64
	dbQuery := h.db.Model(&model.AuditLog{})
	if entity := c.Query("entity_type"); entity != "" {
		dbQuery = dbQuery.Where("entity_type = ?", entity)
	}
	if err := dbQuery.Count(&total).Error; err != nil {
		handleServiceError(c, h.logger, err, "audit list")
		return
	}
	if err := dbQuery.Order("created_at DESC").Offset((query.Page - 1) * query.PageSize).Limit(query.PageSize).Find(&rows).Error; err != nil {
		handleServiceError(c, h.logger, err, "audit list")
		return
	}
	OK(c, pageResponse(rows, total, query.Page, query.PageSize))
}

func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, gin.H{"code": constants.CodeOK, "message": constants.MsgOK, "data": data})
}

func OKWithMessage(c *gin.Context, message string, data any) {
	c.JSON(http.StatusOK, gin.H{"code": constants.CodeOK, "message": message, "data": data})
}

func Fail(c *gin.Context, status, code int, message string) {
	c.AbortWithStatusJSON(status, gin.H{"code": code, "message": message, "data": nil})
}

func pageResponse(list any, total int64, page, pageSize int) gin.H {
	return gin.H{"list": list, "total": total, "page": page, "page_size": pageSize}
}

func bindPageQuery(c *gin.Context, query *dto.PageQuery) bool {
	if err := c.ShouldBindQuery(query); err != nil {
		Fail(c, http.StatusBadRequest, constants.CodeBadRequest, "invalid pagination parameters")
		return false
	}
	query.Normalize()
	return true
}

func handleServiceError(c *gin.Context, logger *slog.Logger, err error, context string) {
	var appErr *util.AppError
	if errors.As(err, &appErr) {
		logger.Warn("request rejected", "context", context, "error", appErr.Error())
		Fail(c, appErrorStatus(appErr.Code), appErr.Code, appErr.Message)
		return
	}
	logger.Error("request failed", "context", context, "error", err.Error())
	Fail(c, http.StatusInternalServerError, constants.CodeInternalError, constants.MsgInternalError)
}

func requestAuditContext(c *gin.Context) service.AuditContext {
	requestID, _ := c.Get("request_id")
	return service.AuditContext{
		OperatorID: middleware.GetUserID(c), OperatorName: middleware.GetPhone(c),
		RequestID: stringValue(requestID), IP: c.ClientIP(),
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
