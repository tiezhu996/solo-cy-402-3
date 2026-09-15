package service

import (
	"log/slog"
	"time"

	"cylawcase/internal/constants"
	"cylawcase/internal/model"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// SeedService 启动时幂等写入预置数据。
type SeedService struct {
	db     *gorm.DB
	logger *slog.Logger
}

// NewSeedService 构造种子服务。
func NewSeedService(db *gorm.DB, logger *slog.Logger) *SeedService {
	return &SeedService{db: db, logger: logger}
}

// Seed 当 users 表为空时写入管理员、律师、助理与示例案件数据。
func (s *SeedService) Seed() error {
	var count int64
	if err := s.db.Model(&model.User{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	adminHash, _ := bcrypt.GenerateFromPassword([]byte("Admin@123"), bcrypt.DefaultCost)
	userHash, _ := bcrypt.GenerateFromPassword([]byte("User@123"), bcrypt.DefaultCost)
	users := []model.User{
		{Username: "admin", PasswordHash: string(adminHash), RealName: "系统管理员", Role: constants.RoleAdmin, Email: "admin@cylawcase.dev", Phone: "13800000001"},
		{Username: "lawyer", PasswordHash: string(userHash), RealName: "张律师", Role: constants.RoleLawyer, LicenseNo: "LAW1101010001", Email: "lawyer@cylawcase.dev", Phone: "13800000002"},
		{Username: "assistant", PasswordHash: string(userHash), RealName: "李助理", Role: constants.RoleAssistant, Email: "assistant@cylawcase.dev", Phone: "13800000003"},
		{Username: "lawyer2", PasswordHash: string(userHash), RealName: "王律师", Role: constants.RoleLawyer, LicenseNo: "LAW1101010002", Email: "lawyer2@cylawcase.dev", Phone: "13800000004"},
	}
	now := time.Now()
	cases := []model.Case{
		{CaseNo: "CY20260001", Title: "华信科技买卖合同纠纷", CaseType: constants.CaseTypeCommercial, Status: constants.CaseStatusInvestigating, ClientID: 1, LeadLawyerID: 2, CoLawyerIDs: model.IDList{4}, AssistantIDs: model.IDList{3}, Summary: "货款催收与合同违约赔偿。"},
		{CaseNo: "CY20260002", Title: "陈晓明民间借贷纠纷", CaseType: constants.CaseTypeCivil, Status: constants.CaseStatusFiled, ClientID: 2, LeadLawyerID: 2, CoLawyerIDs: model.IDList{}, AssistantIDs: model.IDList{}, Summary: "借款 50 万元及利息追偿。"},
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		for i := range users {
			if err := tx.Create(&users[i]).Error; err != nil {
				return err
			}
		}
		for i := range cases {
			cases[i].AcceptDate = &now
			if err := tx.Create(&cases[i]).Error; err != nil {
				return err
			}
		}
		s.logger.Info("seed data created")
		return nil
	})
}
