package handler

import (
	"log/slog"
	"net/http"

	"groundclearance/internal/constants"
	"groundclearance/internal/dto"
	"groundclearance/internal/model"
	"groundclearance/internal/service"

	"github.com/gin-gonic/gin"
)

type TurnaroundHandler struct {
	svc    *service.TurnaroundService
	logger *slog.Logger
}

func NewTurnaroundHandler(svc *service.TurnaroundService, logger *slog.Logger) *TurnaroundHandler {
	return &TurnaroundHandler{svc: svc, logger: logger}
}

func (h *TurnaroundHandler) List(c *gin.Context) {
	var query dto.PageQuery
	if !bindPageQuery(c, &query) {
		return
	}
	rows, total, err := h.svc.List(query.Page, query.PageSize, c.Query("status"), c.Query("risk_level"), c.Query("search"))
	if err != nil {
		handleServiceError(c, h.logger, err, "turnaround list")
		return
	}
	OK(c, pageResponse(rows, total, query.Page, query.PageSize))
}

func (h *TurnaroundHandler) Summary(c *gin.Context) {
	result, err := h.svc.Summary()
	if err != nil {
		handleServiceError(c, h.logger, err, "turnaround summary")
		return
	}
	OK(c, result)
}

func (h *TurnaroundHandler) Get(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	row, checks, err := h.svc.Get(id)
	if err != nil {
		handleServiceError(c, h.logger, err, "turnaround get")
		return
	}
	OK(c, gin.H{"turnaround": row, "checks": checks})
}

func (h *TurnaroundHandler) Readiness(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	result, err := h.svc.Readiness(id)
	if err != nil {
		handleServiceError(c, h.logger, err, "turnaround readiness")
		return
	}
	OK(c, result)
}

func (h *TurnaroundHandler) Create(c *gin.Context) {
	var request dto.TurnaroundCreateRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		Fail(c, http.StatusBadRequest, constants.CodeBadRequest, err.Error())
		return
	}
	row := &model.Turnaround{FlightNo: request.FlightNo, Stand: request.Stand, Phase: request.Phase,
		ScheduledAt: request.ScheduledAt, RiskLevel: request.RiskLevel, GroundUnitIDs: model.JSONList(request.GroundUnitIDs), CoordinatorID: request.CoordinatorID}
	checks := make([]model.SafetyCheck, 0, len(request.Checks))
	for _, item := range request.Checks {
		checks = append(checks, model.SafetyCheck{GroundUnitID: item.GroundUnitID, CheckCode: item.CheckCode,
			ItemName: item.ItemName, RiskLevel: item.RiskLevel, Evidence: model.JSONList(item.Evidence)})
	}
	created, err := h.svc.Create(row, checks, requestAuditContext(c))
	if err != nil {
		handleServiceError(c, h.logger, err, "turnaround create")
		return
	}
	c.Set("audit_persisted", true)
	OK(c, created)
}

func (h *TurnaroundHandler) ChangeStatus(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var request dto.TurnaroundStatusRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		Fail(c, http.StatusBadRequest, constants.CodeBadRequest, err.Error())
		return
	}
	row, err := h.svc.ChangeStatus(id, request.Status, request.Version, requestAuditContext(c))
	if err != nil {
		handleServiceError(c, h.logger, err, "turnaround status change")
		return
	}
	c.Set("audit_persisted", true)
	OK(c, row)
}
