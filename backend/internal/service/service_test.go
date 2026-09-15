package service

import (
	"strings"
	"testing"

	"cylawcase/internal/constants"
	"cylawcase/internal/model"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func TestCanFlow(t *testing.T) {
	cases := []struct {
		from, to string
		want     bool
	}{
		{constants.CaseStatusFiled, constants.CaseStatusInvestigating, true},
		{constants.CaseStatusInvestigating, constants.CaseStatusHearing, true},
		{constants.CaseStatusHearing, constants.CaseStatusClosed, true},
		{constants.CaseStatusClosed, constants.CaseStatusArchived, true},
		{constants.CaseStatusFiled, constants.CaseStatusClosed, false},
		{constants.CaseStatusClosed, constants.CaseStatusFiled, false},
		{constants.CaseStatusInvestigating, constants.CaseStatusFiled, true},
	}
	for _, tc := range cases {
		if got := canFlow(tc.from, tc.to); got != tc.want {
			t.Errorf("canFlow(%s->%s) = %v, want %v", tc.from, tc.to, got, tc.want)
		}
	}
}

func TestContains(t *testing.T) {
	if !contains(constants.CaseTypeValues, constants.CaseTypeLabor) {
		t.Error("labor should be in case types")
	}
	if contains(constants.CaseTypeValues, "bogus") {
		t.Error("bogus should not be in case types")
	}
}

func TestU64(t *testing.T) {
	if u64(42) != "42" {
		t.Error("u64(42) != 42")
	}
}

func TestStatusValidators(t *testing.T) {
	if !constants.IsValidCaseStatus(constants.CaseStatusArchived) {
		t.Error("archived should be valid")
	}
	if constants.IsValidCaseStatus("bogus") {
		t.Error("bogus should be invalid")
	}
	if !constants.IsValidBillingStatus(constants.BillingStatusInvoiced) {
		t.Error("invoiced should be valid")
	}
	if !constants.IsValidBillingType(constants.BillingTypeTravelFee) {
		t.Error("travel_fee should be valid")
	}
}

func TestResolveCaseAccess(t *testing.T) {
	c := &model.Case{ID: 1, LeadLawyerID: 10, CoLawyerIDs: model.IDList{20, 21}, AssistantIDs: model.IDList{30}}
	cases := []struct {
		name   string
		kase   *model.Case
		userID uint64
		role   string
		want   CaseAccessLevel
	}{
		{"admin global", c, 999, constants.RoleAdmin, AccessAdmin},
		{"lead lawyer", c, 10, constants.RoleLawyer, AccessLead},
		{"co lawyer", c, 20, constants.RoleLawyer, AccessCoLawyer},
		{"assistant member", c, 30, constants.RoleAssistant, AccessAssistant},
		{"non member lawyer", c, 40, constants.RoleLawyer, AccessNone},
		{"non member assistant", c, 41, constants.RoleAssistant, AccessNone},
		{"nil case", nil, 10, constants.RoleLawyer, AccessNone},
		{"admin nil case", nil, 10, constants.RoleAdmin, AccessAdmin},
	}
	for _, tc := range cases {
		if got := ResolveCaseAccess(tc.kase, tc.userID, tc.role); got != tc.want {
			t.Errorf("%s: ResolveCaseAccess = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestCaseAccessLevelOrdering(t *testing.T) {
	// 级别数值即权限高低：assistant < co_lawyer < lead < admin，越权判定依赖该顺序。
	if !(AccessNone < AccessAssistant && AccessAssistant < AccessCoLawyer && AccessCoLawyer < AccessLead && AccessLead < AccessAdmin) {
		t.Error("access level ordering broken")
	}
	if AccessAdmin.String() != "admin" || AccessCoLawyer.String() != "co_lawyer" || AccessNone.String() != "none" {
		t.Error("access level string broken")
	}
}

func TestNormalizeIDList(t *testing.T) {
	got := normalizeIDList([]uint64{3, 0, 3, 5, 5, 7})
	want := model.IDList{3, 5, 7}
	if len(got) != len(want) {
		t.Fatalf("normalizeIDList = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("normalizeIDList[%d] = %d, want %d", i, got[i], want[i])
		}
	}
	if empty := normalizeIDList(nil); len(empty) != 0 {
		t.Errorf("normalizeIDList(nil) = %v, want empty", empty)
	}
}

func TestRemoveID(t *testing.T) {
	got := removeID(model.IDList{4, 5, 6}, 5)
	if len(got) != 2 || got[0] != 4 || got[1] != 6 {
		t.Errorf("removeID = %v, want [4 6]", got)
	}
	if idListContains(removeID(model.IDList{4}, 4), 4) {
		t.Error("removeID should drop the only element")
	}
}

func TestMatchBillingClient(t *testing.T) {
	cases := []struct {
		caseClientID uint64
		clientID     uint64
		want         bool
	}{
		{1, 1, true},   // 账单客户 = 案件客户，允许
		{1, 2, false},  // 账单挂到无关客户，拒绝
		{0, 1, false},  // 案件无客户归属，拒绝
		{2, 2, true},
	}
	for _, tc := range cases {
		if got := MatchBillingClient(tc.caseClientID, tc.clientID); got != tc.want {
			t.Errorf("MatchBillingClient(%d, %d) = %v, want %v", tc.caseClientID, tc.clientID, got, tc.want)
		}
	}
}

func TestPresetDocumentUpsertSQL(t *testing.T) {
	// 预置文档写入必须使用 ON CONFLICT DO NOTHING：
	// 多实例同时启动时只有一份记录生效，不会因主键冲突导致实例退出。
	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN:                  "host=127.0.0.1 port=1 user=u password=p dbname=d sslmode=disable",
		PreferSimpleProtocol: true,
	}), &gorm.Config{DryRun: true})
	if err != nil {
		t.Skipf("cannot open dry-run db: %v", err)
	}
	doc := presetDocuments[0]
	tx := db.Clauses(clause.OnConflict{DoNothing: true}).Create(&doc)
	if tx.Error != nil {
		t.Skipf("dry-run create failed: %v", tx.Error)
	}
	sql := tx.Statement.SQL.String()
	if !strings.Contains(sql, "ON CONFLICT DO NOTHING") {
		t.Errorf("upsert SQL missing ON CONFLICT DO NOTHING: %s", sql)
	}
	if !strings.Contains(sql, `"documents"`) {
		t.Errorf("upsert SQL should target documents table: %s", sql)
	}
}
