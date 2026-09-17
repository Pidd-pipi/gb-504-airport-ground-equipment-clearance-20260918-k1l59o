package handler

import (
	"log/slog"
	"net/http"

	"groundclearance/internal/constants"
	"groundclearance/internal/dto"
	"groundclearance/internal/middleware"
	"groundclearance/internal/service"

	"github.com/gin-gonic/gin"
)

type ClearanceDecisionHandler struct {
	svc    *service.ClearanceDecisionService
	logger *slog.Logger
}

func NewClearanceDecisionHandler(svc *service.ClearanceDecisionService, logger *slog.Logger) *ClearanceDecisionHandler {
	return &ClearanceDecisionHandler{svc: svc, logger: logger}
}

func (h *ClearanceDecisionHandler) List(c *gin.Context) {
	var query dto.PageQuery
	if !bindPageQuery(c, &query) {
		return
	}
	rows, total, err := h.svc.List(query.Page, query.PageSize, c.Query("state"))
	if err != nil {
		handleServiceError(c, h.logger, err, "clearance list")
		return
	}
	OK(c, pageResponse(rows, total, query.Page, query.PageSize))
}

func (h *ClearanceDecisionHandler) Summary(c *gin.Context) {
	result, err := h.svc.Summary()
	if err != nil {
		handleServiceError(c, h.logger, err, "clearance summary")
		return
	}
	OK(c, result)
}

func (h *ClearanceDecisionHandler) Get(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	row, err := h.svc.Get(id)
	if err != nil {
		handleServiceError(c, h.logger, err, "clearance get")
		return
	}
	OK(c, row)
}

func (h *ClearanceDecisionHandler) Decide(c *gin.Context) {
	var request dto.ClearanceDecisionRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		Fail(c, http.StatusBadRequest, constants.CodeBadRequest, err.Error())
		return
	}
	requestID, _ := c.Get("request_id")
	row, err := h.svc.Decide(request.TurnaroundID, middleware.GetUserID(c), request.State,
		request.Restrictions, request.Reason, request.Evidence, stringValue(requestID), middleware.GetPhone(c), c.ClientIP())
	if err != nil {
		handleServiceError(c, h.logger, err, "clearance decision")
		return
	}
	c.Set("audit_persisted", true)
	OKWithMessage(c, constants.MsgDecisionRecorded, row)
}

func stringValue(value any) string {
	result, _ := value.(string)
	return result
}
