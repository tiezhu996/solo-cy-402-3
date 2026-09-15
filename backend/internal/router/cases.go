package router

import (
	"cylawcase/internal/constants"
	"cylawcase/internal/middleware"

	"github.com/gin-gonic/gin"
)

// registerCaseRoutes 案件路由。
// 创建案件限律师/管理员；成员调整、状态流转、主办交接在 service 层按案件访问级别判定。
func (r *Router) registerCaseRoutes(g *gin.RouterGroup) {
	cases := g.Group("/cases")
	cases.Use(middleware.AuthRequired(r.cfg))
	cases.GET("", r.caseH.List)
	cases.GET("/:id", r.caseH.Get)
	cases.POST("", middleware.RequireRole(constants.RoleAdmin, constants.RoleLawyer), r.caseH.Create)
	cases.PUT("/:id", r.caseH.Update)
	cases.POST("/:id/status", r.caseH.ChangeStatus)
	cases.POST("/:id/assign", r.caseH.Assign)
	cases.GET("/:id/members", r.caseH.GetMembers)
	cases.PUT("/:id/members", r.caseH.UpdateMembers)
}
