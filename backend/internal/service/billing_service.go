package service

import (
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"cylawcase/internal/constants"
	"cylawcase/internal/model"
	"cylawcase/internal/repository"
	"cylawcase/internal/util"
)

// BillingService 账单业务逻辑。
type BillingService struct {
	repo       *repository.BillingRepository
	caseRepo   *repository.CaseRepository
	clientRepo *repository.ClientRepository
	logger     *slog.Logger
}

// NewBillingService 构造账单服务。
func NewBillingService(repo *repository.BillingRepository, caseRepo *repository.CaseRepository,
	clientRepo *repository.ClientRepository, logger *slog.Logger) *BillingService {
	return &BillingService{repo: repo, caseRepo: caseRepo, clientRepo: clientRepo, logger: logger}
}

// Create 创建账单。仅管理员或主办律师；账单客户必须与案件客户一致，否则按失败处理。
func (s *BillingService) Create(caseID, clientID uint64, userID uint64, role, billingType string, amount float64, invoiceInfo string) (*model.Billing, error) {
	if !constants.IsValidBillingType(billingType) {
		return nil, util.NewAppError(constants.CodeValidationFailed, "Billing[billing_type="+billingType+"] create: invalid type")
	}
	cs, _, err := CheckCaseAccess(s.caseRepo, caseID, userID, role, AccessLead)
	if err != nil {
		s.logger.Warn(constants.LogCaseAccessDenied, "case_id", caseID, "user_id", userID, "action", "billing_create")
		return nil, err
	}
	if !MatchBillingClient(cs.ClientID, clientID) {
		s.logger.Warn(constants.LogBillingClientMismatch, "case_id", caseID, "case_client_id", cs.ClientID, "client_id", clientID)
		return nil, util.NewAppError(constants.CodeValidationFailed,
			constants.MsgBillingClientMismatch+"（Billing[case_id="+u64(caseID)+",client_id="+u64(clientID)+"] create: case client_id="+u64(cs.ClientID)+"）")
	}
	if _, err := s.clientRepo.FindByID(clientID); err != nil {
		return nil, util.Wrap(err, "Billing[client_id=%d] create: client not found", clientID)
	}
	if amount < 0 {
		return nil, util.NewAppError(constants.CodeValidationFailed, "Billing[amount="+strconv.FormatFloat(amount, 'f', 2, 64)+"] create: amount must be >= 0")
	}
	b := &model.Billing{BillNo: genBillNo(), BillingType: billingType,
		Amount: amount, Status: constants.BillingStatusPending,
		CaseID: caseID, ClientID: clientID, InvoiceInfo: invoiceInfo}
	if err := s.repo.Create(b); err != nil {
		s.logger.Error(constants.LogBillingCreateFailed, "error", err.Error())
		return nil, util.Wrap(err, "Billing[case_id=%d] create failed", caseID)
	}
	s.logger.Info(constants.LogBillingCreateSuccess, "billing_id", b.ID, "bill_no", b.BillNo)
	return b, nil
}

// MarkPaid 标记支付（pending -> paid）。仅管理员或主办律师。
func (s *BillingService) MarkPaid(id, userID uint64, role string) (*model.Billing, error) {
	b, err := s.writableBilling(id, userID, role, "paid")
	if err != nil {
		return nil, err
	}
	if b.Status != constants.BillingStatusPending {
		s.logger.Warn(constants.LogBillingPaidFailed, "billing_id", id, "status", b.Status)
		return nil, util.NewAppError(constants.CodeBillingStatusConflict, "Billing[id="+u64(id)+"] paid failed: status="+b.Status)
	}
	b.Status = constants.BillingStatusPaid
	if err := s.repo.Update(b); err != nil {
		return nil, util.Wrap(err, "Billing[id=%d] paid save failed", id)
	}
	s.logger.Info(constants.LogBillingPaidSuccess, "billing_id", b.ID)
	return b, nil
}

// MarkInvoiced 开票（paid -> invoiced）。仅管理员或主办律师。
func (s *BillingService) MarkInvoiced(id, userID uint64, role, invoiceInfo string) (*model.Billing, error) {
	b, err := s.writableBilling(id, userID, role, "invoiced")
	if err != nil {
		return nil, err
	}
	if b.Status != constants.BillingStatusPaid {
		return nil, util.NewAppError(constants.CodeBillingStatusConflict, "Billing[id="+u64(id)+"] invoiced failed: status="+b.Status)
	}
	b.Status = constants.BillingStatusInvoiced
	if invoiceInfo != "" {
		b.InvoiceInfo = invoiceInfo
	}
	if err := s.repo.Update(b); err != nil {
		return nil, util.Wrap(err, "Billing[id=%d] invoiced save failed", id)
	}
	s.logger.Info(constants.LogBillingInvoicedSuccess, "billing_id", b.ID)
	return b, nil
}

// Void 作废账单。仅管理员或主办律师。
func (s *BillingService) Void(id, userID uint64, role string) (*model.Billing, error) {
	b, err := s.writableBilling(id, userID, role, "void")
	if err != nil {
		return nil, err
	}
	if b.Status == constants.BillingStatusVoid {
		return nil, util.NewAppError(constants.CodeBillingStatusConflict, "Billing[id="+u64(id)+"] void failed: already void")
	}
	b.Status = constants.BillingStatusVoid
	if err := s.repo.Update(b); err != nil {
		return nil, util.Wrap(err, "Billing[id=%d] void save failed", id)
	}
	s.logger.Info(constants.LogBillingVoidSuccess, "billing_id", b.ID)
	return b, nil
}

// writableBilling 加载账单并校验操作者对其案件具备主办级权限。
func (s *BillingService) writableBilling(id, userID uint64, role, action string) (*model.Billing, error) {
	b, err := s.repo.FindByID(id)
	if err != nil {
		return nil, util.Wrap(err, "Billing[id=%d] %s find failed", id, action)
	}
	if _, _, err := CheckCaseAccess(s.caseRepo, b.CaseID, userID, role, AccessLead); err != nil {
		s.logger.Warn(constants.LogCaseAccessDenied, "case_id", b.CaseID, "user_id", userID, "action", "billing_"+action)
		return nil, err
	}
	return b, nil
}

// List 分页查询账单；memberID > 0 时仅返回该用户为成员的案件账单。
func (s *BillingService) List(page, pageSize int, caseID, clientID uint64, status string, memberID uint64) ([]model.Billing, int64, error) {
	return s.repo.List(page, pageSize, caseID, clientID, status, memberID)
}

// ListByCase 查询某案件账单。主办/协办律师可查看，助理不可见账单。
func (s *BillingService) ListByCase(caseID, userID uint64, role string) ([]model.Billing, error) {
	if _, _, err := CheckCaseAccess(s.caseRepo, caseID, userID, role, AccessCoLawyer); err != nil {
		return nil, err
	}
	return s.repo.ListByCase(caseID)
}

// Summary 本月应收/已收/待收汇总；memberID > 0 时仅统计该用户为成员的案件账单。
func (s *BillingService) Summary(memberID uint64) (map[string]float64, error) {
	sum, err := s.repo.Summary(time.Now(), memberID)
	if err != nil {
		return nil, err
	}
	s.logger.Info(constants.LogBillingSummary, "summary", fmt.Sprintf("%v", sum))
	return sum, nil
}

// MatchBillingClient 账单客户归属校验：账单只能挂在案件对应客户名下。
func MatchBillingClient(caseClientID, clientID uint64) bool {
	return caseClientID == clientID
}

func genBillNo() string {
	return fmt.Sprintf("BILL%d%06d", time.Now().Year(), time.Now().UnixNano()%1000000)
}
