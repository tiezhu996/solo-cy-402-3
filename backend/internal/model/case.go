package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// Case 案件实体。
type Case struct {
	ID           uint64     `gorm:"primaryKey" json:"id"`
	CaseNo       string     `gorm:"size:50;uniqueIndex;not null" json:"case_no"`
	Title        string     `gorm:"size:200;not null" json:"title"`
	CaseType     string     `gorm:"size:30;not null;default:civil" json:"case_type"`
	Status       string     `gorm:"size:30;not null;default:filed;index" json:"status"`
	AcceptDate   *time.Time `json:"accept_date"`
	CloseDate    *time.Time `json:"close_date"`
	Summary      string     `gorm:"type:text" json:"summary"`
	ClientID     uint64     `gorm:"not null;index" json:"client_id"`
	LeadLawyerID uint64     `gorm:"not null;index" json:"lead_lawyer_id"`
	CoLawyerIDs  IDList     `gorm:"type:jsonb" json:"co_lawyer_ids"`
	AssistantIDs IDList     `gorm:"type:jsonb;not null;default:'[]'" json:"assistant_ids"`
	CreatedAt    time.Time  `json:"created_at"`
}

// TableName 指定表名。
func (Case) TableName() string { return "cases" }

// IDList 用户 ID 列表，以 JSONB 存储、以 JSON 数组序列化。
type IDList []uint64

// Value 实现 driver.Valuer，空列表落库为 []。
func (l IDList) Value() (driver.Value, error) {
	if l == nil {
		return "[]", nil
	}
	raw, err := json.Marshal(l)
	if err != nil {
		return nil, fmt.Errorf("marshal id list: %w", err)
	}
	return string(raw), nil
}

// Scan 实现 sql.Scanner。
func (l *IDList) Scan(src any) error {
	switch v := src.(type) {
	case nil:
		*l = IDList{}
		return nil
	case []byte:
		return json.Unmarshal(v, l)
	case string:
		return json.Unmarshal([]byte(v), l)
	default:
		return fmt.Errorf("scan id list: unsupported type %T", src)
	}
}

// MarshalJSON 保证空列表序列化为 [] 而非 null。
func (l IDList) MarshalJSON() ([]byte, error) {
	if l == nil {
		return []byte("[]"), nil
	}
	return json.Marshal([]uint64(l))
}
