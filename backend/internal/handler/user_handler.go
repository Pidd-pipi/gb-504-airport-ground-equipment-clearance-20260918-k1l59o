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

type UserHandler struct {
	svc    *service.UserService
	logger *slog.Logger
}

func NewUserHandler(svc *service.UserService, logger *slog.Logger) *UserHandler {
	return &UserHandler{svc: svc, logger: logger}
}

func (h *UserHandler) Register(c *gin.Context) {
	var request dto.RegisterRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		Fail(c, http.StatusBadRequest, constants.CodeBadRequest, err.Error())
		return
	}
	user, err := h.svc.Register(request.Phone, request.Password, request.Name, request.Role)
	if err != nil {
		handleServiceError(c, h.logger, err, "register")
		return
	}
	OK(c, user)
}

func (h *UserHandler) Login(c *gin.Context) {
	var request dto.LoginRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		Fail(c, http.StatusBadRequest, constants.CodeBadRequest, err.Error())
		return
	}
	token, user, err := h.svc.Login(c.GetString("jwt_secret"), c.GetInt("jwt_expire_hours"), request.Phone, request.Password)
	if err != nil {
		handleServiceError(c, h.logger, err, "login")
		return
	}
	OKWithMessage(c, constants.MsgLoginSuccess, dto.LoginResponse{Token: token, User: user})
}

func (h *UserHandler) Me(c *gin.Context) {
	user, err := h.svc.GetByID(middleware.GetUserID(c))
	if err != nil {
		handleServiceError(c, h.logger, err, "current user")
		return
	}
	OK(c, user)
}

func (h *UserHandler) UpdateProfile(c *gin.Context) {
	var request dto.UpdateProfileRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		Fail(c, http.StatusBadRequest, constants.CodeBadRequest, err.Error())
		return
	}
	user, err := h.svc.UpdateProfile(middleware.GetUserID(c), request.Name)
	if err != nil {
		handleServiceError(c, h.logger, err, "update profile")
		return
	}
	OK(c, user)
}

func (h *UserHandler) List(c *gin.Context) {
	var query dto.PageQuery
	if !bindPageQuery(c, &query) {
		return
	}
	rows, total, err := h.svc.List(query.Page, query.PageSize)
	if err != nil {
		handleServiceError(c, h.logger, err, "user list")
		return
	}
	OK(c, pageResponse(rows, total, query.Page, query.PageSize))
}
