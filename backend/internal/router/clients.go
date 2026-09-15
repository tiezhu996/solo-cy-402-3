package router

import (
	"cylawcase/internal/constants"
	"cylawcase/internal/middleware"

	"github.com/gin-gonic/gin"
)

// registerClientRoutes 客户路由。
// 查看按案件成员关系过滤（service 层）；写操作限律师/管理员，律师还需为客户案件成员。
func (r *Router) registerClientRoutes(g *gin.RouterGroup) {
	clients := g.Group("/clients")
	clients.Use(middleware.AuthRequired(r.cfg))
	clients.GET("", r.client.List)
	clients.GET("/:id", r.client.Get)
	clients.POST("", middleware.RequireRole(constants.RoleAdmin, constants.RoleLawyer), r.client.Create)
	clients.PUT("/:id", middleware.RequireRole(constants.RoleAdmin, constants.RoleLawyer), r.client.Update)
	clients.DELETE("/:id", middleware.RequireRole(constants.RoleAdmin, constants.RoleLawyer), r.client.Delete)
}
