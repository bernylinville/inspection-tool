package repository

import (
	"context"

	"gorm.io/gorm"

	"inspection-tool/apps/cmdb-server/internal/model"
)

type nginxInstanceRepository struct {
	db *gorm.DB
}

// NewNginxInstanceRepository creates a new NginxInstanceRepository.
func NewNginxInstanceRepository(db *gorm.DB) NginxInstanceRepository {
	return &nginxInstanceRepository{db: db}
}

func (r *nginxInstanceRepository) Create(ctx context.Context, instance *model.NginxInstance) error {
	return r.db.WithContext(ctx).Create(instance).Error
}

func (r *nginxInstanceRepository) Update(ctx context.Context, instance *model.NginxInstance) error {
	return r.db.WithContext(ctx).Save(instance).Error
}

func (r *nginxInstanceRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&model.NginxInstance{}, id).Error
}

func (r *nginxInstanceRepository) FindByID(ctx context.Context, id int64) (*model.NginxInstance, error) {
	var instance model.NginxInstance
	if err := r.db.WithContext(ctx).Preload("Host").Preload("Application").First(&instance, id).Error; err != nil {
		return nil, err
	}
	return &instance, nil
}

func (r *nginxInstanceRepository) FindByAddress(ctx context.Context, address string) (*model.NginxInstance, error) {
	var instance model.NginxInstance
	if err := r.db.WithContext(ctx).Preload("Host").Preload("Application").Where("address = ?", address).First(&instance).Error; err != nil {
		return nil, err
	}
	return &instance, nil
}

func (r *nginxInstanceRepository) ListByHostID(ctx context.Context, hostID int64) ([]model.NginxInstance, error) {
	var instances []model.NginxInstance
	if err := r.db.WithContext(ctx).
		Preload("Host").
		Preload("Application").
		Where("host_id = ?", hostID).
		Find(&instances).Error; err != nil {
		return nil, err
	}
	return instances, nil
}

func (r *nginxInstanceRepository) List(ctx context.Context, opts ListOptions) ([]model.NginxInstance, int64, error) {
	baseQuery := r.db.WithContext(ctx).Model(&model.NginxInstance{})
	filteredQuery := applyFilters(baseQuery, opts.Filters)

	var total int64
	if err := filteredQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	opts = normalizeListOptions(opts)
	listQuery := applyFilters(baseQuery, opts.Filters)
	listQuery = applyOrder(listQuery, opts)
	listQuery = applyPagination(listQuery, opts)
	listQuery = listQuery.Preload("Host").Preload("Application")

	var instances []model.NginxInstance
	if err := listQuery.Find(&instances).Error; err != nil {
		return nil, 0, err
	}

	return instances, total, nil
}
