package repository

import (
	"context"

	"gorm.io/gorm"

	"inspection-tool/apps/cmdb-server/internal/model"
)

type roleRepository struct {
	db *gorm.DB
}

// NewRoleRepository creates a new RoleRepository.
func NewRoleRepository(db *gorm.DB) RoleRepository {
	return &roleRepository{db: db}
}

func (r *roleRepository) Create(ctx context.Context, role *model.Role) error {
	return r.db.WithContext(ctx).Create(role).Error
}

func (r *roleRepository) Update(ctx context.Context, role *model.Role) error {
	return r.db.WithContext(ctx).Save(role).Error
}

func (r *roleRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&model.Role{}, id).Error
}

func (r *roleRepository) FindByID(ctx context.Context, id int64) (*model.Role, error) {
	var role model.Role
	if err := r.db.WithContext(ctx).Preload("Permissions").First(&role, id).Error; err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *roleRepository) FindByName(ctx context.Context, name string) (*model.Role, error) {
	var role model.Role
	if err := r.db.WithContext(ctx).Preload("Permissions").Where("name = ?", name).First(&role).Error; err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *roleRepository) List(ctx context.Context, opts ListOptions) ([]model.Role, int64, error) {
	baseQuery := r.db.WithContext(ctx).Model(&model.Role{})
	filteredQuery := applyFilters(baseQuery, opts.Filters)

	var total int64
	if err := filteredQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	opts = normalizeListOptions(opts)
	listQuery := applyFilters(baseQuery, opts.Filters)
	listQuery = applyOrder(listQuery, opts)
	listQuery = applyPagination(listQuery, opts)
	listQuery = listQuery.Preload("Permissions")

	var roles []model.Role
	if err := listQuery.Find(&roles).Error; err != nil {
		return nil, 0, err
	}

	return roles, total, nil
}
