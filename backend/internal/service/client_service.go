package service

import (
	"log/slog"

	"cylawcase/internal/constants"
	"cylawcase/internal/model"
	"cylawcase/internal/repository"
	"cylawcase/internal/util"
)

// ClientService 客户业务逻辑。
type ClientService struct {
	repo     *repository.ClientRepository
	caseRepo *repository.CaseRepository
	logger   *slog.Logger
}

// NewClientService 构造客户服务。
func NewClientService(repo *repository.ClientRepository, caseRepo *repository.CaseRepository, logger *slog.Logger) *ClientService {
	return &ClientService{repo: repo, caseRepo: caseRepo, logger: logger}
}

// Create 新建客户。
func (s *ClientService) Create(name, idNumber, contact, address, remark string) (*model.Client, error) {
	c := &model.Client{Name: name, IDNumber: idNumber, Contact: contact, Address: address, Remark: remark}
	if err := s.repo.Create(c); err != nil {
		return nil, util.Wrap(err, "Client[name=%s] create failed", name)
	}
	s.logger.Info(constants.LogClientCreateSuccess, "client_id", c.ID)
	return c, nil
}

// Update 编辑客户。管理员可编辑任意客户；律师需为该客户案件的成员；助理只读。
func (s *ClientService) Update(id, userID uint64, role, name, idNumber, contact, address, remark string) (*model.Client, error) {
	c, err := s.repo.FindByID(id)
	if err != nil {
		return nil, util.Wrap(err, "Client[id=%d] update find failed", id)
	}
	if err := s.checkWritable(id, userID, role, "update"); err != nil {
		return nil, err
	}
	if name != "" {
		c.Name = name
	}
	if idNumber != "" {
		c.IDNumber = idNumber
	}
	if contact != "" {
		c.Contact = contact
	}
	if address != "" {
		c.Address = address
	}
	if remark != "" {
		c.Remark = remark
	}
	if err := s.repo.Update(c); err != nil {
		return nil, util.Wrap(err, "Client[id=%d] update save failed", id)
	}
	s.logger.Info(constants.LogClientUpdateSuccess, "client_id", c.ID)
	return c, nil
}

// Delete 删除客户。权限同 Update。
func (s *ClientService) Delete(id, userID uint64, role string) error {
	if err := s.checkWritable(id, userID, role, "delete"); err != nil {
		return err
	}
	if err := s.repo.Delete(id); err != nil {
		return util.Wrap(err, "Client[id=%d] delete failed", id)
	}
	s.logger.Info(constants.LogClientDeleteSuccess, "client_id", id)
	return nil
}

// List 分页检索客户；memberID > 0 时仅返回该用户为成员的案件所属客户。
func (s *ClientService) List(page, pageSize int, keyword string, memberID uint64) ([]model.Client, int64, error) {
	return s.repo.List(page, pageSize, keyword, memberID)
}

// GetWithCases 客户详情 + 历史案件。非管理员需为该客户至少一个案件的成员，
// 且历史案件仅返回其可见（为成员）的案件。
func (s *ClientService) GetWithCases(id, userID uint64, role string) (*model.Client, []model.Case, error) {
	c, err := s.repo.FindByID(id)
	if err != nil {
		return nil, nil, util.Wrap(err, "Client[id=%d] get failed", id)
	}
	memberID := userID
	if role == constants.RoleAdmin {
		memberID = 0
	}
	cases, err := s.caseRepo.ListMemberCasesByClient(id, memberID)
	if err != nil {
		return nil, nil, err
	}
	if memberID > 0 && len(cases) == 0 {
		s.logger.Warn(constants.LogCaseAccessDenied, "client_id", id, "user_id", userID, "action", "client_get")
		return nil, nil, util.NewAppError(constants.CodeCaseAccessDenied,
			constants.MsgCaseAccessDenied+"（Client[id="+u64(id)+"] role="+role+" not a case member）")
	}
	return c, cases, nil
}

// checkWritable 客户资料写权限：管理员全局；律师需为客户案件成员；助理只读。
func (s *ClientService) checkWritable(id, userID uint64, role, action string) error {
	if role == constants.RoleAdmin {
		return nil
	}
	if role != constants.RoleLawyer {
		s.logger.Warn(constants.LogCaseAccessDenied, "client_id", id, "user_id", userID, "action", "client_"+action)
		return util.NewAppError(constants.CodeCaseAccessDenied,
			constants.MsgCaseAccessDenied+"（Client[id="+u64(id)+"] "+action+" role="+role+" read only）")
	}
	cases, err := s.caseRepo.ListMemberCasesByClient(id, userID)
	if err != nil {
		return err
	}
	if len(cases) == 0 {
		s.logger.Warn(constants.LogCaseAccessDenied, "client_id", id, "user_id", userID, "action", "client_"+action)
		return util.NewAppError(constants.CodeCaseAccessDenied,
			constants.MsgCaseAccessDenied+"（Client[id="+u64(id)+"] "+action+" role="+role+" not a case member）")
	}
	return nil
}
