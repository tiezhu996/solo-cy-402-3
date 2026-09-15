package router

import (
	"cylawcase/internal/middleware"

	"github.com/gin-gonic/gin"
)

// registerDocumentRoutes 文档路由。
// 下载与文档列表同一成员边界（任意案件成员），逐请求校验，成员移除后立即失效。
func (r *Router) registerDocumentRoutes(g *gin.RouterGroup) {
	docs := g.Group("/documents")
	docs.Use(middleware.AuthRequired(r.cfg))
	docs.GET("", r.document.List)
	docs.POST("", r.document.Create)
	docs.GET("/by-case/:id", r.document.ListByCase)
	docs.GET("/:id/download", r.document.Download)
	docs.DELETE("/:id", r.document.Delete)
}
