package router

import (
	"groundclearance/internal/constants"
	"groundclearance/internal/middleware"

	"github.com/gin-gonic/gin"
)

func (r *Router) registerClearanceRoutes(group *gin.RouterGroup) {
	routes := group.Group("/clearance")
	routes.Use(middleware.AuthRequired(r.cfg))
	routes.GET("", r.clearance.List)
	routes.GET("/summary", r.clearance.Summary)
	routes.GET("/:id", r.clearance.Get)
	routes.POST("", middleware.RequireRole(constants.RoleAdmin, constants.RoleSafetyManager), r.clearance.Decide)
}
