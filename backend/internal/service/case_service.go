package service

import (
	"fmt"
	"log/slog"
	"time"

	"cylawcase/internal/constants"
	"cylawcase/internal/model"
	"cylawcase/internal/repository"
	"cylawcase/internal/util"
)

// CaseService 案件业务逻辑。
type CaseService struct {
	repo       *repository.CaseRepository
	clientRepo *repository.ClientRepository
	userRepo   *repository.UserRepository
	logger     *slog.Logger
}

// NewCaseService 构造案件服务。
func NewCaseService(repo *repository.CaseRepository, clientRepo *repository.ClientRepository,
	userRepo *repository.UserRepository, logger *slog.Logger) *CaseService {
	return &CaseService{repo: repo, clientRepo: clientRepo, userRepo: userRepo, logger: logger}
}

// Create 创建案件。
func (s *CaseService) Create(clientID, leadLawyerID uint64, title, caseType, summary string,
	acceptDate *time.Time, coLawyerIDs, assistantIDs []uint64) (*model.Case, error) {
	if !constants.IsValidCaseType(caseType) {
		return nil, util.NewAppError(constants.CodeValidationFailed, "Case[case_type="+caseType+"] create: invalid type")
	}
	if _, err := s.clientRepo.FindByID(clientID); err != nil {
		return nil, util.Wrap(err, "Case[client_id=%d] create: client not found", clientID)
	}
	co := normalizeIDList(coLawyerIDs)
	assistants := normalizeIDList(assistantIDs)
	if err := ValidateCaseMembers(s.userRepo, leadLawyerID, co, assistants); err != nil {
		return nil, err
	}
	c := &model.Case{
		CaseNo:       genCaseNo(),
		Title:        title,
		CaseType:     caseType,
		Status:       constants.CaseStatusFiled,
		AcceptDate:   acceptDate,
		Summary:      summary,
		ClientID:     clientID,
		LeadLawyerID: leadLawyerID,
		CoLawyerIDs:  co,
		AssistantIDs: assistants,
	}
	if err := s.repo.Create(c); err != nil {
		s.logger.Error(constants.LogCaseCreateFailed, "error", err.Error())
		return nil, util.Wrap(err, "Case[title=%s] create failed", title)
	}
	s.logger.Info(constants.LogCaseCreateSuccess, "case_id", c.ID, "case_no", c.CaseNo)
	return c, nil
}

// Update 更新案件基本信息（标题/摘要）。仅管理员或主办律师。
func (s *CaseService) Update(id, userID uint64, role, title, summary string) (*model.Case, error) {
	c, _, err := CheckCaseAccess(s.repo, id, userID, role, AccessLead)
	if err != nil {
		s.logger.Warn(constants.LogCaseAccessDenied, "case_id", id, "user_id", userID, "action", "update")
		return nil, err
	}
	if title != "" {
		c.Title = title
	}
	if summary != "" {
		c.Summary = summary
	}
	if err := s.repo.Update(c); err != nil {
		return nil, util.Wrap(err, "Case[id=%d] update save failed", id)
	}
	s.logger.Info(constants.LogCaseUpdateSuccess, "case_id", c.ID)
	return c, nil
}

// ChangeStatus 案件状态流转。仅管理员或主办律师。
func (s *CaseService) ChangeStatus(id, userID uint64, role, status string) (*model.Case, error) {
	c, level, err := CheckCaseAccess(s.repo, id, userID, role, AccessLead)
	if err != nil {
		s.logger.Warn(constants.LogCaseAccessDenied, "case_id", id, "user_id", userID, "action", "status")
		return nil, err
	}
	if !constants.IsValidCaseStatus(status) {
		return nil, util.NewAppError(constants.CodeValidationFailed, "Case[id="+u64(id)+"] status invalid: "+status)
	}
	if level != AccessAdmin && !canFlow(c.Status, status) {
		return nil, util.NewAppError(constants.CodeCaseStatusConflict, "Case[id="+u64(id)+"] status conflict: "+c.Status+" -> "+status)
	}
	c.Status = status
	if status == constants.CaseStatusClosed && c.CloseDate == nil {
		now := time.Now()
		c.CloseDate = &now
	}
	if err := s.repo.Update(c); err != nil {
		s.logger.Error(constants.LogCaseStatusChangeFailed, "error", err.Error())
		return nil, util.Wrap(err, "Case[id=%d] status change save failed", id)
	}
	s.logger.Info(constants.LogCaseStatusChangeSuccess, "case_id", c.ID, "status", status)
	return c, nil
}

// Assign 主办律师交接。仅管理员或现任主办律师；交接后管理权随 lead_lawyer_id 一并转移，
// 新主办若此前是协办/助理成员，则自动从成员列表移除。
func (s *CaseService) Assign(id, userID uint64, role string, leadLawyerID uint64) (*model.Case, error) {
	c, _, err := CheckCaseAccess(s.repo, id, userID, role, AccessLead)
	if err != nil {
		s.logger.Warn(constants.LogCaseAccessDenied, "case_id", id, "user_id", userID, "action", "assign")
		return nil, err
	}
	if err := ValidateCaseMembers(s.userRepo, leadLawyerID, removeID(c.CoLawyerIDs, leadLawyerID), removeID(c.AssistantIDs, leadLawyerID)); err != nil {
		return nil, util.Wrap(err, "Case[id=%d] assign failed: lawyer not match", id)
	}
	c.LeadLawyerID = leadLawyerID
	c.CoLawyerIDs = removeID(c.CoLawyerIDs, leadLawyerID)
	c.AssistantIDs = removeID(c.AssistantIDs, leadLawyerID)
	if err := s.repo.Update(c); err != nil {
		s.logger.Error(constants.LogCaseAssignFailed, "error", err.Error())
		return nil, util.Wrap(err, "Case[id=%d] assign save failed", id)
	}
	s.logger.Info(constants.LogCaseAssignSuccess, "case_id", c.ID, "lead_lawyer_id", leadLawyerID)
	return c, nil
}

// UpdateMembers 调整协办律师与助理。仅管理员或主办律师；全量替换，立即生效。
func (s *CaseService) UpdateMembers(id, userID uint64, role string, coLawyerIDs, assistantIDs []uint64) (*model.Case, error) {
	c, _, err := CheckCaseAccess(s.repo, id, userID, role, AccessLead)
	if err != nil {
		s.logger.Warn(constants.LogCaseAccessDenied, "case_id", id, "user_id", userID, "action", "members")
		return nil, err
	}
	co := normalizeIDList(coLawyerIDs)
	assistants := normalizeIDList(assistantIDs)
	if err := ValidateCaseMembers(s.userRepo, c.LeadLawyerID, co, assistants); err != nil {
		s.logger.Warn(constants.LogCaseMembersUpdateFailed, "case_id", id, "error", err.Error())
		return nil, err
	}
	c.CoLawyerIDs = co
	c.AssistantIDs = assistants
	if err := s.repo.Update(c); err != nil {
		s.logger.Error(constants.LogCaseMembersUpdateFailed, "error", err.Error())
		return nil, util.Wrap(err, "Case[id=%d] members update save failed", id)
	}
	s.logger.Info(constants.LogCaseMembersUpdateSuccess, "case_id", c.ID,
		"co_lawyer_ids", fmt.Sprintf("%v", []uint64(co)), "assistant_ids", fmt.Sprintf("%v", []uint64(assistants)))
	return c, nil
}

// GetMembers 案件成员视图（主办/协办/助理用户信息）。案件成员即可查看。
func (s *CaseService) GetMembers(id, userID uint64, role string) (*model.Case, *model.User, []model.User, []model.User, error) {
	c, _, err := CheckCaseAccess(s.repo, id, userID, role, AccessAssistant)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	lead, err := s.userRepo.FindByID(c.LeadLawyerID)
	if err != nil {
		return nil, nil, nil, nil, util.Wrap(err, "Case[id=%d] members load lead failed", id)
	}
	co, err := s.userRepo.FindByIDs([]uint64(c.CoLawyerIDs))
	if err != nil {
		return nil, nil, nil, nil, util.Wrap(err, "Case[id=%d] members load co lawyers failed", id)
	}
	assistants, err := s.userRepo.FindByIDs([]uint64(c.AssistantIDs))
	if err != nil {
		return nil, nil, nil, nil, util.Wrap(err, "Case[id=%d] members load assistants failed", id)
	}
	return c, lead, co, assistants, nil
}

// List 分页查询案件；memberID > 0 时仅返回该用户为成员的案件。
func (s *CaseService) List(page, pageSize int, caseType, status string, lawyerID, memberID uint64, startDate, endDate *time.Time) ([]model.Case, int64, error) {
	return s.repo.List(page, pageSize, caseType, status, lawyerID, memberID, startDate, endDate)
}

// Get 案件详情。仅案件成员或管理员可见。
func (s *CaseService) Get(id, userID uint64, role string) (*model.Case, error) {
	c, _, err := CheckCaseAccess(s.repo, id, userID, role, AccessAssistant)
	if err != nil {
		return nil, err
	}
	return c, nil
}

// canFlow 案件状态机：filed->investigating->hearing->closed->archived，允许回退到上一步。
func canFlow(from, to string) bool {
	idx := map[string]int{constants.CaseStatusFiled: 0, constants.CaseStatusInvestigating: 1,
		constants.CaseStatusHearing: 2, constants.CaseStatusClosed: 3, constants.CaseStatusArchived: 4}
	a, okA := idx[from]
	b, okB := idx[to]
	if !okA || !okB {
		return false
	}
	return b == a+1 || b == a-1 || b == a
}

func genCaseNo() string {
	return fmt.Sprintf("CY%d%04d", time.Now().Year(), time.Now().UnixNano()%10000)
}

func u64(v uint64) string {
	return fmt.Sprintf("%d", v)
}
