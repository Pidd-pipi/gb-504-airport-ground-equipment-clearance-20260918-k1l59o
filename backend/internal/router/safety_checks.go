package router

import (
	"groundclearance/internal/constants"
	"groundclearance/internal/middleware"

	"github.com/gin-gonic/gin"
)

func (r *Router) registerCheckRoutes(group *gin.RouterGroup) {
	routes := group.Group("/checks")
	routes.Use(middleware.AuthRequired(r.cfg))
	routes.GET("", r.check.List)
	routes.GET("/summary", r.check.Summary)
	routes.POST("", middleware.RequireRole(constants.RoleAdmin, constants.RoleSafetyManager, constants.RoleInspector), r.check.Create)
	routes.PATCH("/:id/review", middleware.RequireRole(constants.RoleAdmin, constants.RoleSafetyManager, constants.RoleInspector), r.check.Review)
}
