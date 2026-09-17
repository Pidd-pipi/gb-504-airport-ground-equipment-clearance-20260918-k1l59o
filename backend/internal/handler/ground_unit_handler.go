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

type GroundUnitHandler struct {
	svc    *service.GroundUnitService
	logger *slog.Logger
}

func NewGroundUnitHandler(svc *service.GroundUnitService, logger *slog.Logger) *GroundUnitHandler {
	return &GroundUnitHandler{svc: svc, logger: logger}
}

func (h *GroundUnitHandler) List(c *gin.Context) {
	var query dto.PageQuery
	if !bindPageQuery(c, &query) {
		return
	}
	rows, total, err := h.svc.List(query.Page, query.PageSize, c.Query("state"), c.Query("unit_type"), c.Query("search"))
	if err != nil {
		handleServiceError(c, h.logger, err, "ground unit list")
		return
	}
	OK(c, pageResponse(rows, total, query.Page, query.PageSize))
}

func (h *GroundUnitHandler) Summary(c *gin.Context) {
	result, err := h.svc.Summary()
	if err != nil {
		handleServiceError(c, h.logger, err, "ground unit summary")
		return
	}
	OK(c, result)
}

func (h *GroundUnitHandler) Get(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	unit, err := h.svc.Get(id)
	if err != nil {
		handleServiceError(c, h.logger, err, "ground unit get")
		return
	}
	OK(c, unit)
}

func (h *GroundUnitHandler) Create(c *gin.Context) {
	var request dto.GroundUnitCreateRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		Fail(c, http.StatusBadRequest, constants.CodeBadRequest, err.Error())
		return
	}
	unit := &model.GroundUnit{UnitCode: request.UnitCode, Name: request.Name, UnitType: request.UnitType,
		Stand: request.Stand, State: request.State, LastInspectionAt: request.LastInspectionAt, Notes: request.Notes}
	created, err := h.svc.CreateUnit(unit, requestAuditContext(c))
	if err != nil {
		handleServiceError(c, h.logger, err, "ground unit create")
		return
	}
	c.Set("audit_persisted", true)
	OK(c, created)
}

func (h *GroundUnitHandler) ChangeState(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var request dto.GroundUnitStateRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		Fail(c, http.StatusBadRequest, constants.CodeBadRequest, err.Error())
		return
	}
	unit, err := h.svc.ChangeState(id, request.State, request.Notes, request.Version,
		middleware.GetRole(c), requestAuditContext(c))
	if err != nil {
		handleServiceError(c, h.logger, err, "ground unit state change")
		return
	}
	c.Set("audit_persisted", true)
	OK(c, unit)
}

func parseID(c *gin.Context) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		Fail(c, http.StatusBadRequest, constants.CodeBadRequest, "invalid id")
		return 0, false
	}
	return id, true
}
