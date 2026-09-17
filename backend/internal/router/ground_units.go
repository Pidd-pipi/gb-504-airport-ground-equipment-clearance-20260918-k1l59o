package router

import (
	"groundclearance/internal/constants"
	"groundclearance/internal/middleware"

	"github.com/gin-gonic/gin"
)

func (r *Router) registerGroundUnitRoutes(group *gin.RouterGroup) {
	routes := group.Group("/ground-units")
	routes.Use(middleware.AuthRequired(r.cfg))
	routes.GET("", r.groundUnit.List)
	routes.GET("/summary", r.groundUnit.Summary)
	routes.GET("/:id", r.groundUnit.Get)
	routes.POST("", middleware.RequireRole(constants.RoleAdmin, constants.RoleSafetyManager), r.groundUnit.Create)
	routes.PATCH("/:id/state", middleware.RequireRole(constants.RoleAdmin, constants.RoleSafetyManager, constants.RoleInspector), r.groundUnit.ChangeState)
}
