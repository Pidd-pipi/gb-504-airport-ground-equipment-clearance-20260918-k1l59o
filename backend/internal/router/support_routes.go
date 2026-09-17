package router

import (
	"groundclearance/internal/constants"
	"groundclearance/internal/middleware"

	"github.com/gin-gonic/gin"
)

func (r *Router) registerUserRoutes(group *gin.RouterGroup) {
	routes := group.Group("/users")
	routes.Use(middleware.AuthRequired(r.cfg))
	routes.GET("/me", r.user.Me)
	routes.PUT("/me", r.user.UpdateProfile)
	routes.GET("", middleware.RequireRole(constants.RoleAdmin, constants.RoleSafetyManager), r.user.List)
	routes.POST("", middleware.RequireRole(constants.RoleAdmin), r.user.Register)
}

func (r *Router) registerAuditRoutes(group *gin.RouterGroup) {
	routes := group.Group("/audit")
	routes.Use(middleware.AuthRequired(r.cfg), middleware.RequireRole(constants.RoleAdmin, constants.RoleSafetyManager))
	routes.GET("", r.audit.List)
}
