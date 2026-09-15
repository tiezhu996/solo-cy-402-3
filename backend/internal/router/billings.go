package router

import (
	"cylawcase/internal/constants"
	"cylawcase/internal/middleware"

	"github.com/gin-gonic/gin"
)

// registerBillingRoutes 账单路由。
// 助理角色整体不可见账单（查看与改动都禁止），故组级限定律师/管理员；
// 组内再由 service 层按案件成员关系细分：查看需主办/协办，改动需主办/管理员。
func (r *Router) registerBillingRoutes(g *gin.RouterGroup) {
	billings := g.Group("/billings")
	billings.Use(middleware.AuthRequired(r.cfg), middleware.RequireRole(constants.RoleAdmin, constants.RoleLawyer))
	billings.GET("", r.billing.List)
	billings.GET("/summary", r.billing.Summary)
	billings.GET("/by-case/:id", r.billing.ListByCase)
	billings.POST("", r.billing.Create)
	billings.POST("/:id/paid", r.billing.MarkPaid)
	billings.POST("/:id/invoiced", r.billing.MarkInvoiced)
	billings.POST("/:id/void", r.billing.Void)
}
