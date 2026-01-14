package repository

import (
	"context"

	"gorm.io/gorm"

	"inspection-tool/apps/cmdb-server/internal/model"
)

type permissionRepository struct {
	db *gorm.DB
}

// NewPermissionRepository creates a new PermissionRepository.
func NewPermissionRepository(db *gorm.DB) PermissionRepository {
	return &permissionRepository{db: db}
}

func (r *permissionRepository) Create(ctx context.Context, permission *model.Permission) error {
	return r.db.WithContext(ctx).Create(permission).Error
}

func (r *permissionRepository) Update(ctx context.Context, permission *model.Permission) error {
	return r.db.WithContext(ctx).Save(permission).Error
}

func (r *permissionRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&model.Permission{}, id).Error
}

func (r *permissionRepository) FindByID(ctx context.Context, id int64) (*model.Permission, error) {
	var permission model.Permission
	if err := r.db.WithContext(ctx).First(&permission, id).Error; err != nil {
		return nil, err
	}
	return &permission, nil
}

func (r *permissionRepository) FindByName(ctx context.Context, name string) (*model.Permission, error) {
	var permission model.Permission
	if err := r.db.WithContext(ctx).Where("name = ?", name).First(&permission).Error; err != nil {
		return nil, err
	}
	return &permission, nil
}

func (r *permissionRepository) ListByResource(ctx context.Context, resource string) ([]model.Permission, error) {
	var permissions []model.Permission
	if err := r.db.WithContext(ctx).Where("resource = ?", resource).Find(&permissions).Error; err != nil {
		return nil, err
	}
	return permissions, nil
}

func (r *permissionRepository) List(ctx context.Context, opts ListOptions) ([]model.Permission, int64, error) {
	baseQuery := r.db.WithContext(ctx).Model(&model.Permission{})
	filteredQuery := applyFilters(baseQuery, opts.Filters)

	var total int64
	if err := filteredQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	opts = normalizeListOptions(opts)
	listQuery := applyFilters(baseQuery, opts.Filters)
	listQuery = applyOrder(listQuery, opts)
	listQuery = applyPagination(listQuery, opts)

	var permissions []model.Permission
	if err := listQuery.Find(&permissions).Error; err != nil {
		return nil, 0, err
	}

	return permissions, total, nil
}
