package repository

import (
	"errors"
	"fmt"

	"cylawcase/internal/model"

	"gorm.io/gorm"
)

// UserRepository 用户仓储。
type UserRepository struct {
	db *gorm.DB
}

// NewUserRepository 构造用户仓储。
func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create 创建用户。
func (r *UserRepository) Create(u *model.User) error {
	if err := r.db.Create(u).Error; err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

// FindByID 按 ID 查询用户。
func (r *UserRepository) FindByID(id uint64) (*model.User, error) {
	var u model.User
	if err := r.db.First(&u, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("find user by id: %w", err)
	}
	return &u, nil
}

// FindByUsername 按用户名查询用户。
func (r *UserRepository) FindByUsername(username string) (*model.User, error) {
	var u model.User
	if err := r.db.Where("username = ?", username).First(&u).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("find user by username: %w", err)
	}
	return &u, nil
}

// FindByIDs 批量按 ID 查询用户（用于案件成员视图）。
func (r *UserRepository) FindByIDs(ids []uint64) ([]model.User, error) {
	list := []model.User{}
	if len(ids) == 0 {
		return list, nil
	}
	if err := r.db.Where("id IN ?", ids).Order("id ASC").Find(&list).Error; err != nil {
		return nil, fmt.Errorf("find users by ids: %w", err)
	}
	return list, nil
}

// ListLawyers 查询律师列表。
func (r *UserRepository) ListLawyers() ([]model.User, error) {
	var list []model.User
	if err := r.db.Where("role = ?", "lawyer").Order("id ASC").Find(&list).Error; err != nil {
		return nil, fmt.Errorf("list lawyers: %w", err)
	}
	return list, nil
}

// ListByRole 按角色查询用户列表（用于案件成员候选：协办律师/助理）。
func (r *UserRepository) ListByRole(role string) ([]model.User, error) {
	var list []model.User
	if err := r.db.Where("role = ?", role).Order("id ASC").Find(&list).Error; err != nil {
		return nil, fmt.Errorf("list users by role: %w", err)
	}
	return list, nil
}

// Update 更新用户。
func (r *UserRepository) Update(u *model.User) error {
	if err := r.db.Save(u).Error; err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	return nil
}

// List 分页查询用户。
func (r *UserRepository) List(page, pageSize int) ([]model.User, int64, error) {
	var list []model.User
	var total int64
	q := r.db.Model(&model.User{})
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count users: %w", err)
	}
	if err := q.Order("id ASC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, fmt.Errorf("list users: %w", err)
	}
	return list, total, nil
}
