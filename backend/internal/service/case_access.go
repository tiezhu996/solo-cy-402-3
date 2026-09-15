package service

import (
	"cylawcase/internal/constants"
	"cylawcase/internal/model"
	"cylawcase/internal/repository"
	"cylawcase/internal/util"
)

// CaseAccessLevel 案件访问级别，数值越大权限越高。
// 级别语义（与产品边界一一对应，所有入口统一使用）：
//   - AccessNone      非成员：无任何访问权
//   - AccessAssistant 助理成员：查看案件/客户/文档，不可见账单
//   - AccessCoLawyer  协办律师：额外可维护文档、查看账单，不可调整成员与案件状态
//   - AccessLead      主办律师：额外可调整成员、流转状态、维护账单、交接主办
//   - AccessAdmin     管理员：全局访问，不受成员关系限制
type CaseAccessLevel int

const (
	AccessNone CaseAccessLevel = iota
	AccessAssistant
	AccessCoLawyer
	AccessLead
	AccessAdmin
)

// String 返回级别名，用于错误信息透传。
func (l CaseAccessLevel) String() string {
	switch l {
	case AccessAssistant:
		return "assistant"
	case AccessCoLawyer:
		return "co_lawyer"
	case AccessLead:
		return "lead"
	case AccessAdmin:
		return "admin"
	default:
		return "none"
	}
}

// ResolveCaseAccess 计算用户对案件的访问级别。纯函数，成员关系即查即算，
// 成员调整后下一次请求立即生效。
func ResolveCaseAccess(c *model.Case, userID uint64, role string) CaseAccessLevel {
	if role == constants.RoleAdmin {
		return AccessAdmin
	}
	if c == nil {
		return AccessNone
	}
	if c.LeadLawyerID == userID {
		return AccessLead
	}
	if idListContains(c.CoLawyerIDs, userID) {
		return AccessCoLawyer
	}
	if idListContains(c.AssistantIDs, userID) {
		return AccessAssistant
	}
	return AccessNone
}

// CheckCaseAccess 加载案件并校验操作者访问级别，不足时返回 CodeCaseAccessDenied。
// 所有案件相关入口（案件/文档/账单/客户）统一经由此处判定，避免边界遗漏。
func CheckCaseAccess(caseRepo *repository.CaseRepository, caseID, userID uint64, role string, min CaseAccessLevel) (*model.Case, CaseAccessLevel, error) {
	c, err := caseRepo.FindByID(caseID)
	if err != nil {
		return nil, AccessNone, util.Wrap(err, "Case[id=%d] access check find failed", caseID)
	}
	level := ResolveCaseAccess(c, userID, role)
	if level < min {
		return nil, level, util.NewAppError(constants.CodeCaseAccessDenied,
			constants.MsgCaseAccessDenied+"（Case[id="+u64(caseID)+"] role="+role+" level="+level.String()+" required="+min.String()+"）")
	}
	return c, level, nil
}

// ValidateCaseMembers 校验成员角色匹配：主办/协办必须是律师（或管理员），
// 助理必须是助理角色，且主办不得同时出现在协办/助理列表中。
func ValidateCaseMembers(userRepo *repository.UserRepository, leadLawyerID uint64, coLawyerIDs, assistantIDs []uint64) error {
	lead, err := userRepo.FindByID(leadLawyerID)
	if err != nil {
		return util.Wrap(err, "Case[lead_lawyer_id=%d] members validate: lawyer not found", leadLawyerID)
	}
	if lead.Role != constants.RoleLawyer && lead.Role != constants.RoleAdmin {
		return util.NewAppError(constants.CodeValidationFailed,
			"Case[lead_lawyer_id="+u64(leadLawyerID)+"] members validate: role="+lead.Role+" not lawyer")
	}
	for _, id := range coLawyerIDs {
		if id == leadLawyerID {
			return util.NewAppError(constants.CodeValidationFailed,
				"Case[co_lawyer_ids] members validate: user="+u64(id)+" is lead lawyer")
		}
		u, err := userRepo.FindByID(id)
		if err != nil {
			return util.Wrap(err, "Case[co_lawyer_ids] members validate: user %d not found", id)
		}
		if u.Role != constants.RoleLawyer && u.Role != constants.RoleAdmin {
			return util.NewAppError(constants.CodeValidationFailed,
				"Case[co_lawyer_ids] members validate: user="+u64(id)+" role="+u.Role+" not lawyer")
		}
	}
	for _, id := range assistantIDs {
		if id == leadLawyerID {
			return util.NewAppError(constants.CodeValidationFailed,
				"Case[assistant_ids] members validate: user="+u64(id)+" is lead lawyer")
		}
		u, err := userRepo.FindByID(id)
		if err != nil {
			return util.Wrap(err, "Case[assistant_ids] members validate: user %d not found", id)
		}
		if u.Role != constants.RoleAssistant {
			return util.NewAppError(constants.CodeValidationFailed,
				"Case[assistant_ids] members validate: user="+u64(id)+" role="+u.Role+" not assistant")
		}
	}
	return nil
}

// normalizeIDList 去重并保持顺序，空输入归一为空列表。
func normalizeIDList(ids []uint64) model.IDList {
	seen := make(map[uint64]bool, len(ids))
	out := make(model.IDList, 0, len(ids))
	for _, id := range ids {
		if id == 0 || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}

// removeID 从列表中移除指定 ID（主办交接时清理新主办的历史成员身份）。
func removeID(list model.IDList, id uint64) model.IDList {
	out := make(model.IDList, 0, len(list))
	for _, v := range list {
		if v != id {
			out = append(out, v)
		}
	}
	return out
}

func idListContains(list model.IDList, id uint64) bool {
	for _, v := range list {
		if v == id {
			return true
		}
	}
	return false
}
