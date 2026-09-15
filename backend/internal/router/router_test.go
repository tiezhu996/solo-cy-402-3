package router

import (
	"testing"

	"cylawcase/internal/config"
	"cylawcase/internal/handler"
	"cylawcase/internal/util"

	"log/slog"
)

// TestSetupRoutes 路由注册冒烟测试：Gin 在路由冲突时会 panic，
// 此处用空依赖装配全部路由，验证 /cases/:id 与 /cases/:id/members 等组合不冲突。
func TestSetupRoutes(t *testing.T) {
	cfg := &config.Config{
		ServerPort:         "8080",
		JWTSecret:          "test_secret",
		JWTExpireHours:     1,
		RateLimitPerMinute: 100,
		UploadDir:          t.TempDir(),
		UploadMaxMB:        1,
	}
	logger := util.NewLogger(slog.LevelError)
	r := New(cfg, nil, logger,
		handler.NewUserHandler(nil, logger),
		handler.NewClientHandler(nil, logger),
		handler.NewCaseHandler(nil, logger),
		handler.NewDocumentHandler(nil, logger),
		handler.NewBillingHandler(nil, logger),
		handler.NewUploadHandler(cfg, nil, logger),
		handler.NewAuditLogHandler(nil, logger),
	)
	engine := r.Setup()
	want := map[string]bool{
		"GET /api/v1/cases/:id":            false,
		"GET /api/v1/cases/:id/members":    false,
		"PUT /api/v1/cases/:id/members":    false,
		"POST /api/v1/cases/:id/assign":    false,
		"POST /api/v1/cases/:id/status":    false,
		"GET /api/v1/users/assistants":     false,
		"GET /api/v1/billings/summary":     false,
		"GET /api/v1/billings/by-case/:id": false,
		"POST /api/v1/upload/file":         false,
		"POST /api/v1/upload/avatar":       false,
	}
	for _, ri := range engine.Routes() {
		key := ri.Method + " " + ri.Path
		if _, ok := want[key]; ok {
			want[key] = true
		}
	}
	for key, found := range want {
		if !found {
			t.Errorf("route %s not registered", key)
		}
	}
}
