package router

import (
	"groundclearance/internal/constants"
	"groundclearance/internal/middleware"

	"github.com/gin-gonic/gin"
)

func (r *Router) registerTurnaroundRoutes(group *gin.RouterGroup) {
	routes := group.Group("/turnarounds")
	routes.Use(middleware.AuthRequired(r.cfg))
	routes.GET("", r.turnaround.List)
	routes.GET("/summary", r.turnaround.Summary)
	routes.GET("/:id", r.turnaround.Get)
	routes.GET("/:id/readiness", r.turnaround.Readiness)
	routes.POST("", middleware.RequireRole(constants.RoleAdmin, constants.RoleSafetyManager), r.turnaround.Create)
	routes.PATCH("/:id/status", middleware.RequireRole(constants.RoleAdmin, constants.RoleSafetyManager), r.turnaround.ChangeStatus)
}
