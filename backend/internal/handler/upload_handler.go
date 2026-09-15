package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"cylawcase/internal/config"
	"cylawcase/internal/constants"
	"cylawcase/internal/middleware"
	"cylawcase/internal/repository"
	"cylawcase/internal/service"
	"cylawcase/internal/util"

	"github.com/gin-gonic/gin"
)

// UploadHandler 文件上传处理器。
type UploadHandler struct {
	cfg      *config.Config
	caseRepo *repository.CaseRepository
	logger   *slog.Logger
}

// NewUploadHandler 构造上传处理器。
func NewUploadHandler(cfg *config.Config, caseRepo *repository.CaseRepository, logger *slog.Logger) *UploadHandler {
	return &UploadHandler{cfg: cfg, caseRepo: caseRepo, logger: logger}
}

// UploadFile 上传案件文件。
// 必须先通过案件成员范围与文档维护权限校验（主办/协办/管理员），
// 任一条件不满足直接拒绝，不在磁盘留下无案件归属的文件。
func (h *UploadHandler) UploadFile(c *gin.Context) {
	caseID, err := strconv.ParseUint(c.PostForm("case_id"), 10, 64)
	if err != nil || caseID == 0 {
		Fail(c, http.StatusBadRequest, constants.CodeBadRequest, "Upload file: case_id required")
		return
	}
	if _, _, err := service.CheckCaseAccess(h.caseRepo, caseID, middleware.GetUserID(c), middleware.GetUserRole(c), service.AccessCoLawyer); err != nil {
		h.logger.Warn(constants.LogUploadAccessDenied, "case_id", caseID, "user_id", middleware.GetUserID(c))
		h.wrapError(c, err, "Upload file access denied")
		return
	}
	file, err := c.FormFile("file")
	if err != nil {
		Fail(c, http.StatusBadRequest, constants.CodeBadRequest, "Upload file: missing file")
		return
	}
	url, err := util.SaveUploadedFile(h.cfg.UploadDir, h.cfg.UploadMaxMB, file)
	if err != nil {
		h.logger.Error(constants.LogUploadFileFailed, "error", err.Error())
		Fail(c, http.StatusBadRequest, constants.CodeBadRequest, "Upload file failed: "+err.Error())
		return
	}
	h.logger.Info(constants.LogUploadFileSuccess, "url", url, "case_id", caseID)
	OK(c, gin.H{"url": url})
}

// UploadAvatar 上传头像（个人资料，与案件无关，仅需登录）。
func (h *UploadHandler) UploadAvatar(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		Fail(c, http.StatusBadRequest, constants.CodeBadRequest, "Upload avatar: missing file")
		return
	}
	url, err := util.SaveUploadedFile(h.cfg.UploadDir, h.cfg.UploadMaxMB, file)
	if err != nil {
		h.logger.Error(constants.LogUploadFileFailed, "error", err.Error())
		Fail(c, http.StatusBadRequest, constants.CodeBadRequest, "Upload avatar failed: "+err.Error())
		return
	}
	h.logger.Info(constants.LogUploadFileSuccess, "url", url, "scene", "avatar")
	OK(c, gin.H{"url": url})
}

func (h *UploadHandler) wrapError(c *gin.Context, err error, ctx string) {
	var appErr *util.AppError
	if errors.As(err, &appErr) {
		c.Set("audit_detail", appErr.Message)
		h.logger.Warn("upload handler error", "context", ctx, "error", appErr.Error())
		Fail(c, appErrorStatus(appErr.Code), appErr.Code, appErr.Message)
		return
	}
	h.logger.Error("upload handler error", "context", ctx, "error", err.Error())
	Fail(c, http.StatusInternalServerError, constants.CodeInternalError, constants.MsgInternalError)
}
