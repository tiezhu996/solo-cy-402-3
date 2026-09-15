package repository

import (
	"fmt"

	"gorm.io/gorm"
)

// ScopeCaseMember 案件成员可见性过滤：主办律师 / 协办律师 / 助理任一命中即可见。
// memberID 为 0 时不加过滤（管理员全局可见）。
// 案件、文档、账单、客户四个仓储共用此作用域，保证各入口边界一致。
func ScopeCaseMember(memberID uint64) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if memberID == 0 {
			return db
		}
		ids := fmt.Sprintf("[%d]", memberID)
		return db.Where("lead_lawyer_id = ? OR co_lawyer_ids @> ?::jsonb OR assistant_ids @> ?::jsonb", memberID, ids, ids)
	}
}
