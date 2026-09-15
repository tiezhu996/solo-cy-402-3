package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"cylawcase/internal/constants"
	"cylawcase/internal/dto"
	"cylawcase/internal/middleware"
	"cylawcase/internal/model"
	"cylawcase/internal/service"
	"cylawcase/internal/util"

	"github.com/gin-gonic/gin"
)

// CaseHandler 案件 HTTP 处理器。
type CaseHandler struct {
	svc    *service.CaseService
	logger *slog.Logger
}

// NewCaseHandler 构造案件处理器。
func NewCaseHandler(svc *service.CaseService, logger *slog.Logger) *CaseHandler {
	return &CaseHandler{svc: svc, logger: logger}
}

// List 案件列表。非管理员仅返回其为成员的案件。
func (h *CaseHandler) List(c *gin.Context) {
	var q dto.PageQuery
	_ = c.ShouldBindQuery(&q)
	q.Normalize()
	lawyerID, _ := strconv.ParseUint(c.Query("lead_lawyer_id"), 10, 64)
	var startDate, endDate *time.Time
	if s := c.Query("start_date"); s != "" {
		if t, err := time.Parse("2006-01-02", s); err == nil {
			startDate = &t
		}
	}
	if s := c.Query("end_date"); s != "" {
		if t, err := time.Parse("2006-01-02", s); err == nil {
			endDate = &t
		}
	}
	list, total, err := h.svc.List(q.Page, q.PageSize, c.Query("case_type"), c.Query("status"), lawyerID,
		memberScope(c), startDate, endDate)
	if err != nil {
		h.wrapError(c, err, "Case list failed")
		return
	}
	OK(c, pageResponse(list, total, q.Page, q.PageSize))
}

// Get 案件详情。仅案件成员或管理员。
func (h *CaseHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		Fail(c, http.StatusBadRequest, constants.CodeBadRequest, "Case[id] get: invalid id")
		return
	}
	cs, err := h.svc.Get(id, middleware.GetUserID(c), middleware.GetUserRole(c))
	if err != nil {
		h.wrapError(c, err, "Case get failed")
		return
	}
	OK(c, cs)
}

// Create 创建案件。
func (h *CaseHandler) Create(c *gin.Context) {
	var req dto.CaseCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, constants.CodeBadRequest, "Case create: "+err.Error())
		return
	}
	acceptDate, err := dto.ParseAcceptDate(valueOrEmpty(req.AcceptDate))
	if err != nil {
		Fail(c, http.StatusBadRequest, constants.CodeBadRequest, "Case create: invalid accept_date")
		return
	}
	cs, err := h.svc.Create(req.ClientID, req.LeadLawyerID, req.Title, req.CaseType, req.Summary, acceptDate,
		req.CoLawyerIDs, req.AssistantIDs)
	if err != nil {
		h.wrapError(c, err, "Case[title="+req.Title+"] create failed")
		return
	}
	OKWithMessage(c, constants.MsgCaseCreated, cs)
}

// Update 更新案件基本信息。仅管理员或主办律师。
func (h *CaseHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		Fail(c, http.StatusBadRequest, constants.CodeBadRequest, "Case[id] update: invalid id")
		return
	}
	var req dto.CaseUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, constants.CodeBadRequest, "Case[id="+strconv.FormatUint(id, 10)+"] update: "+err.Error())
		return
	}
	cs, err := h.svc.Update(id, middleware.GetUserID(c), middleware.GetUserRole(c), req.Title, req.Summary)
	if err != nil {
		h.wrapError(c, err, "Case update failed")
		return
	}
	OK(c, cs)
}

// ChangeStatus 状态流转。仅管理员或主办律师。
func (h *CaseHandler) ChangeStatus(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		Fail(c, http.StatusBadRequest, constants.CodeBadRequest, "Case[id] status: invalid id")
		return
	}
	var req dto.CaseStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, constants.CodeBadRequest, "Case[id="+strconv.FormatUint(id, 10)+"] status: "+err.Error())
		return
	}
	cs, err := h.svc.ChangeStatus(id, middleware.GetUserID(c), middleware.GetUserRole(c), req.Status)
	if err != nil {
		h.wrapError(c, err, "Case status change failed")
		return
	}
	OK(c, cs)
}

// Assign 主办律师交接。仅管理员或现任主办律师，交接后管理权同时转移。
func (h *CaseHandler) Assign(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		Fail(c, http.StatusBadRequest, constants.CodeBadRequest, "Case[id] assign: invalid id")
		return
	}
	var req dto.CaseAssignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, constants.CodeBadRequest, "Case[id="+strconv.FormatUint(id, 10)+"] assign: "+err.Error())
		return
	}
	cs, err := h.svc.Assign(id, middleware.GetUserID(c), middleware.GetUserRole(c), req.LeadLawyerID)
	if err != nil {
		h.wrapError(c, err, "Case assign failed")
		return
	}
	OK(c, cs)
}

// GetMembers 案件成员视图。案件成员即可查看。
func (h *CaseHandler) GetMembers(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		Fail(c, http.StatusBadRequest, constants.CodeBadRequest, "Case[id] members: invalid id")
		return
	}
	cs, lead, coLawyers, assistants, err := h.svc.GetMembers(id, middleware.GetUserID(c), middleware.GetUserRole(c))
	if err != nil {
		h.wrapError(c, err, "Case members get failed")
		return
	}
	OK(c, dto.CaseMembersResponse{
		LeadLawyerID: cs.LeadLawyerID,
		CoLawyerIDs:  idSlice(cs.CoLawyerIDs),
		AssistantIDs: idSlice(cs.AssistantIDs),
		LeadLawyer:   memberInfo(*lead),
		CoLawyers:    memberInfos(coLawyers),
		Assistants:   memberInfos(assistants),
	})
}

// UpdateMembers 调整协办律师与助理。仅管理员或主办律师，立即生效。
func (h *CaseHandler) UpdateMembers(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		Fail(c, http.StatusBadRequest, constants.CodeBadRequest, "Case[id] members update: invalid id")
		return
	}
	var req dto.CaseMembersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, constants.CodeBadRequest, "Case[id="+strconv.FormatUint(id, 10)+"] members update: "+err.Error())
		return
	}
	cs, err := h.svc.UpdateMembers(id, middleware.GetUserID(c), middleware.GetUserRole(c), req.CoLawyerIDs, req.AssistantIDs)
	if err != nil {
		h.wrapError(c, err, "Case members update failed")
		return
	}
	OKWithMessage(c, constants.MsgCaseMembersUpdated, cs)
}

// memberScope 非管理员返回其用户 ID 作为成员过滤，管理员返回 0 表示全局。
func memberScope(c *gin.Context) uint64 {
	if middleware.GetUserRole(c) == constants.RoleAdmin {
		return 0
	}
	return middleware.GetUserID(c)
}

func memberInfo(u model.User) dto.CaseMemberInfo {
	return dto.CaseMemberInfo{ID: u.ID, Username: u.Username, RealName: u.RealName, Role: u.Role}
}

// idSlice 归一化 ID 列表，保证空列表序列化为 [] 而非 null。
func idSlice(l model.IDList) []uint64 {
	if l == nil {
		return []uint64{}
	}
	return []uint64(l)
}

func memberInfos(list []model.User) []dto.CaseMemberInfo {
	out := make([]dto.CaseMemberInfo, 0, len(list))
	for _, u := range list {
		out = append(out, memberInfo(u))
	}
	return out
}

func (h *CaseHandler) wrapError(c *gin.Context, err error, ctx string) {
	var appErr *util.AppError
	if errors.As(err, &appErr) {
		c.Set("audit_detail", appErr.Message)
		h.logger.Warn("case handler error", "context", ctx, "error", appErr.Error())
		Fail(c, appErrorStatus(appErr.Code), appErr.Code, appErr.Message)
		return
	}
	h.logger.Error("case handler error", "context", ctx, "error", err.Error())
	Fail(c, http.StatusInternalServerError, constants.CodeInternalError, constants.MsgInternalError)
}

func valueOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
