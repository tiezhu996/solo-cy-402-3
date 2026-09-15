package dto

import "time"

// CaseCreateRequest 创建案件请求。
type CaseCreateRequest struct {
	ClientID     uint64   `json:"client_id" binding:"required"`
	LeadLawyerID uint64   `json:"lead_lawyer_id" binding:"required"`
	Title        string   `json:"title" binding:"required,max=200"`
	CaseType     string   `json:"case_type" binding:"required,oneof=civil criminal administrative commercial labor"`
	Summary      string   `json:"summary"`
	AcceptDate   *string  `json:"accept_date"`
	CoLawyerIDs  []uint64 `json:"co_lawyer_ids"`
	AssistantIDs []uint64 `json:"assistant_ids"`
}

// CaseUpdateRequest 更新案件基本信息请求（成员调整走 CaseMembersRequest）。
type CaseUpdateRequest struct {
	Title   string `json:"title" binding:"max=200"`
	Summary string `json:"summary"`
}

// CaseStatusRequest 状态流转请求。
type CaseStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=filed investigating hearing closed archived"`
}

// CaseAssignRequest 主办律师交接请求。
type CaseAssignRequest struct {
	LeadLawyerID uint64 `json:"lead_lawyer_id" binding:"required"`
}

// CaseMembersRequest 案件成员调整请求（全量替换协办/助理列表）。
type CaseMembersRequest struct {
	CoLawyerIDs  []uint64 `json:"co_lawyer_ids"`
	AssistantIDs []uint64 `json:"assistant_ids"`
}

// CaseMemberInfo 成员用户信息。
type CaseMemberInfo struct {
	ID       uint64 `json:"id"`
	Username string `json:"username"`
	RealName string `json:"real_name"`
	Role     string `json:"role"`
}

// CaseMembersResponse 案件成员视图。
type CaseMembersResponse struct {
	LeadLawyerID uint64           `json:"lead_lawyer_id"`
	CoLawyerIDs  []uint64         `json:"co_lawyer_ids"`
	AssistantIDs []uint64         `json:"assistant_ids"`
	LeadLawyer   CaseMemberInfo   `json:"lead_lawyer"`
	CoLawyers    []CaseMemberInfo `json:"co_lawyers"`
	Assistants   []CaseMemberInfo `json:"assistants"`
}

// ParseAcceptDate 解析接受日期字符串。
func ParseAcceptDate(s string) (*time.Time, error) {
	if s == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return nil, err
	}
	return &t, nil
}
