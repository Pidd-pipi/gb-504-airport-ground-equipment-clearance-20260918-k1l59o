package handler

import (
	"log/slog"
	"net/http"
	"strconv"

	"groundclearance/internal/constants"
	"groundclearance/internal/dto"
	"groundclearance/internal/middleware"
	"groundclearance/internal/model"
	"groundclearance/internal/service"

	"github.com/gin-gonic/gin"
)

type SafetyCheckHandler struct {
	svc    *service.SafetyCheckService
	logger *slog.Logger
}

func NewSafetyCheckHandler(svc *service.SafetyCheckService, logger *slog.Logger) *SafetyCheckHandler {
	return &SafetyCheckHandler{svc: svc, logger: logger}
}

func (h *SafetyCheckHandler) List(c *gin.Context) {
	var query dto.PageQuery
	if !bindPageQuery(c, &query) {
		return
	}
	var turnaroundID uint64
	if rawID := c.Query("turnaround_id"); rawID != "" {
		parsed, err := strconv.ParseUint(rawID, 10, 64)
		if err != nil || parsed == 0 {
			Fail(c, http.StatusBadRequest, constants.CodeBadRequest, "invalid turnaround_id")
			return
		}
		turnaroundID = parsed
	}
	rows, total, err := h.svc.List(query.Page, query.PageSize, turnaroundID, c.Query("result"))
	if err != nil {
		handleServiceError(c, h.logger, err, "safety check list")
		return
	}
	OK(c, pageResponse(rows, total, query.Page, query.PageSize))
}

func (h *SafetyCheckHandler) Summary(c *gin.Context) {
	result, err := h.svc.Summary()
	if err != nil {
		handleServiceError(c, h.logger, err, "safety check summary")
		return
	}
	OK(c, result)
}

func (h *SafetyCheckHandler) Create(c *gin.Context) {
	var request struct {
		TurnaroundID uint64 `json:"turnaround_id" binding:"required"`
		dto.SafetyCheckRequest
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		Fail(c, http.StatusBadRequest, constants.CodeBadRequest, err.Error())
		return
	}
	check := &model.SafetyCheck{TurnaroundID: request.TurnaroundID, GroundUnitID: request.GroundUnitID,
		CheckCode: request.CheckCode, ItemName: request.ItemName, RiskLevel: request.RiskLevel, Evidence: model.JSONList(request.Evidence)}
	created, err := h.svc.Create(check, requestAuditContext(c))
	if err != nil {
		handleServiceError(c, h.logger, err, "safety check create")
		return
	}
	c.Set("audit_persisted", true)
	OK(c, created)
}

func (h *SafetyCheckHandler) Review(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var request dto.SafetyCheckReviewRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		Fail(c, http.StatusBadRequest, constants.CodeBadRequest, err.Error())
		return
	}
	check, err := h.svc.Review(id, middleware.GetUserID(c), request.Result, request.Evidence, request.Remark, requestAuditContext(c))
	if err != nil {
		handleServiceError(c, h.logger, err, "safety check review")
		return
	}
	c.Set("audit_persisted", true)
	OK(c, check)
}
