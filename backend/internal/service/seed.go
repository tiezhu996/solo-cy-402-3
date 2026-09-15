package service

import (
	"fmt"
	"log/slog"
	"time"

	"cylawcase/internal/constants"
	"cylawcase/internal/model"
	"cylawcase/internal/util"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SeedService 启动时幂等写入预置数据。
type SeedService struct {
	db        *gorm.DB
	uploadDir string
	logger    *slog.Logger
}

// NewSeedService 构造种子服务。
func NewSeedService(db *gorm.DB, uploadDir string, logger *slog.Logger) *SeedService {
	return &SeedService{db: db, uploadDir: uploadDir, logger: logger}
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
		{CaseNo: "CY20260003", Title: "劳动争议仲裁案（已结）", CaseType: constants.CaseTypeLabor, Status: constants.CaseStatusClosed, ClientID: 2, LeadLawyerID: 2, CoLawyerIDs: model.IDList{}, AssistantIDs: model.IDList{}, Summary: "劳动仲裁已裁决结案。", CloseDate: &now},
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

// presetDocuments 预置历史文档：保留原始 ID、文件地址与案件归属，与 database/init.sql 一致。
var presetDocuments = []model.Document{
	{ID: 1, Title: "民事起诉状", FileType: "complaint", FileURL: "/uploads/case1_complaint.pdf", CaseID: 1, UploaderID: 2},
	{ID: 2, Title: "买卖合同证据清单", FileType: "evidence", FileURL: "/uploads/case1_evidence.pdf", CaseID: 1, UploaderID: 2},
	{ID: 3, Title: "一审判决书", FileType: "judgment", FileURL: "/uploads/case3_judgment.pdf", CaseID: 3, UploaderID: 2},
}

// SeedPresetDocuments 幂等补齐预置文档：每次启动执行，并发安全。
// 记录用 INSERT ... ON CONFLICT DO NOTHING 原子去重，随后无条件重读——
// 多实例同时启动时只有一份记录生效，竞态落败方读取胜出方的记录继续启动；
// 文件由 EnsureFile 原子落位，已存在内容绝不改写，重复启动不产生重复记录或重复文件。
func (s *SeedService) SeedPresetDocuments() error {
	for _, p := range presetDocuments {
		// 案件不存在时跳过（本地开发库可能只有部分案件），避免产生错归属记录。
		var caseCount int64
		if err := s.db.Model(&model.Case{}).Where("id = ?", p.CaseID).Count(&caseCount).Error; err != nil {
			return fmt.Errorf("seed preset documents: count case %d: %w", p.CaseID, err)
		}
		if caseCount == 0 {
			s.logger.Warn(constants.LogSeedDocumentBackfill, "document_id", p.ID, "case_id", p.CaseID, "reason", "case missing, skipped")
			continue
		}
		// 并发安全写入：冲突时不报错、不改动已有记录。
		doc := p
		doc.UploadTime = time.Now()
		if err := s.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&doc).Error; err != nil {
			return fmt.Errorf("seed preset documents: upsert document %d: %w", p.ID, err)
		}
		// 无论本次是否真正插入都重新读取：竞态中落败的一方读取胜出方的记录，
		// 以现有记录的文件地址为准继续启动。
		var existing model.Document
		if err := s.db.First(&existing, p.ID).Error; err != nil {
			return fmt.Errorf("seed preset documents: reload document %d: %w", p.ID, err)
		}
		path, err := util.ResolveUploadPath(s.uploadDir, existing.FileURL)
		if err != nil {
			return fmt.Errorf("seed preset documents: resolve %s: %w", existing.FileURL, err)
		}
		created, err := util.EnsureFile(path, util.PlaceholderPDF(presetPlaceholderLines(existing, existing.FileURL)))
		if err != nil {
			return fmt.Errorf("seed preset documents: ensure file %s: %w", path, err)
		}
		if created {
			s.logger.Info(constants.LogSeedDocumentBackfill, "document_id", p.ID, "action", "file backfilled", "path", path)
		}
	}
	// 显式 ID 插入后同步自增序列，避免后续插入主键冲突（幂等，可重复执行）。
	if err := s.db.Exec(`SELECT setval(pg_get_serial_sequence('documents', 'id'), (SELECT COALESCE(MAX(id), 1) FROM documents))`).Error; err != nil {
		return fmt.Errorf("seed preset documents: sync sequence: %w", err)
	}
	return nil
}

// presetPlaceholderLines 占位 PDF 内容（ASCII，Helvetica 不支持中文）。
func presetPlaceholderLines(d model.Document, fileURL string) []string {
	return []string{
		"LexCase Preset Document (placeholder)",
		fmt.Sprintf("Document ID: %d", d.ID),
		fmt.Sprintf("Case ID: %d", d.CaseID),
		fmt.Sprintf("Type: %s", d.FileType),
		fmt.Sprintf("Source URL: %s", fileURL),
		"This file was backfilled by the system seed",
		"because the original file was missing.",
	}
}
