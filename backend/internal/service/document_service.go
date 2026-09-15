package service

import (
	"log/slog"
	"time"

	"cylawcase/internal/constants"
	"cylawcase/internal/model"
	"cylawcase/internal/repository"
	"cylawcase/internal/util"
)

// DocumentService 文档业务逻辑。
type DocumentService struct {
	repo     *repository.DocumentRepository
	caseRepo *repository.CaseRepository
	logger   *slog.Logger
}

// NewDocumentService 构造文档服务。
func NewDocumentService(repo *repository.DocumentRepository, caseRepo *repository.CaseRepository, logger *slog.Logger) *DocumentService {
	return &DocumentService{repo: repo, caseRepo: caseRepo, logger: logger}
}

// Create 上传文档。仅管理员/主办/协办律师可维护文档，助理只读。
func (s *DocumentService) Create(caseID, uploaderID uint64, role, title, fileType, fileURL string) (*model.Document, error) {
	if _, _, err := CheckCaseAccess(s.caseRepo, caseID, uploaderID, role, AccessCoLawyer); err != nil {
		s.logger.Warn(constants.LogCaseAccessDenied, "case_id", caseID, "user_id", uploaderID, "action", "document_upload")
		return nil, err
	}
	if !contains(constants.DocumentFileTypeValues, fileType) {
		return nil, util.NewAppError(constants.CodeValidationFailed, "Document[file_type="+fileType+"] upload: invalid type")
	}
	d := &model.Document{Title: title, FileType: fileType, FileURL: fileURL, UploadTime: time.Now(),
		CaseID: caseID, UploaderID: uploaderID}
	if err := s.repo.Create(d); err != nil {
		s.logger.Error(constants.LogDocumentUploadFailed, "error", err.Error())
		return nil, util.Wrap(err, "Document[case_id=%d] upload create failed", caseID)
	}
	s.logger.Info(constants.LogDocumentUploadSuccess, "document_id", d.ID)
	return d, nil
}

// ListByCase 按案件查看文档。案件成员（含助理）即可查看。
func (s *DocumentService) ListByCase(caseID, userID uint64, role string) ([]model.Document, error) {
	if _, _, err := CheckCaseAccess(s.caseRepo, caseID, userID, role, AccessAssistant); err != nil {
		return nil, err
	}
	return s.repo.ListByCase(caseID)
}

// List 文档中心分页查询；memberID > 0 时仅返回该用户为成员的案件文档。
func (s *DocumentService) List(page, pageSize int, fileType, keyword string, memberID uint64) ([]model.Document, int64, error) {
	return s.repo.List(page, pageSize, fileType, keyword, memberID)
}

// Delete 删除文档。仅管理员/主办/协办律师。
func (s *DocumentService) Delete(id, userID uint64, role string) error {
	d, err := s.repo.FindByID(id)
	if err != nil {
		return util.Wrap(err, "Document[id=%d] delete find failed", id)
	}
	if _, _, err := CheckCaseAccess(s.caseRepo, d.CaseID, userID, role, AccessCoLawyer); err != nil {
		s.logger.Warn(constants.LogCaseAccessDenied, "case_id", d.CaseID, "user_id", userID, "action", "document_delete")
		return err
	}
	if err := s.repo.Delete(id); err != nil {
		return util.Wrap(err, "Document[id=%d] delete failed", id)
	}
	s.logger.Info(constants.LogDocumentDeleteSuccess, "document_id", id)
	return nil
}
