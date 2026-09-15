package repository

import (
	"fmt"

	"gorm.io/gorm"
)

// MigrateCaseMemberSplit 成员模型升级迁移：
// 历史数据把助理也混在 co_lawyer_ids 里，这里把助理角色成员迁到 assistant_ids，
// co_lawyer_ids 只保留律师。WHERE 条件保证幂等，可随每次启动重复执行。
func MigrateCaseMemberSplit(db *gorm.DB) error {
	if err := db.Exec(`UPDATE cases SET co_lawyer_ids = '[]'::jsonb WHERE co_lawyer_ids IS NULL`).Error; err != nil {
		return fmt.Errorf("backfill co_lawyer_ids: %w", err)
	}
	if err := db.Exec(`UPDATE cases SET assistant_ids = '[]'::jsonb WHERE assistant_ids IS NULL`).Error; err != nil {
		return fmt.Errorf("backfill assistant_ids: %w", err)
	}
	if err := db.Exec(`
UPDATE cases c SET
  assistant_ids = COALESCE((
    SELECT jsonb_agg(e) FROM jsonb_array_elements(c.co_lawyer_ids) e
    WHERE EXISTS (SELECT 1 FROM users u WHERE u.id = (e #>> '{}')::bigint AND u.role = 'assistant')
  ), '[]'::jsonb),
  co_lawyer_ids = COALESCE((
    SELECT jsonb_agg(e) FROM jsonb_array_elements(c.co_lawyer_ids) e
    WHERE NOT EXISTS (SELECT 1 FROM users u WHERE u.id = (e #>> '{}')::bigint AND u.role = 'assistant')
  ), '[]'::jsonb)
WHERE EXISTS (
  SELECT 1 FROM jsonb_array_elements(c.co_lawyer_ids) e, users u
  WHERE u.id = (e #>> '{}')::bigint AND u.role = 'assistant'
)`).Error; err != nil {
		return fmt.Errorf("split case members: %w", err)
	}
	return nil
}
