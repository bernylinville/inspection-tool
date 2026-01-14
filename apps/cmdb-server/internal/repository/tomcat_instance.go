package repository

import (
	"context"

	"gorm.io/gorm"

	"inspection-tool/apps/cmdb-server/internal/model"
)

type tomcatInstanceRepository struct {
	db *gorm.DB
}

// NewTomcatInstanceRepository creates a new TomcatInstanceRepository.
func NewTomcatInstanceRepository(db *gorm.DB) TomcatInstanceRepository {
	return &tomcatInstanceRepository{db: db}
}

func (r *tomcatInstanceRepository) Create(ctx context.Context, instance *model.TomcatInstance) error {
	return r.db.WithContext(ctx).Create(instance).Error
}

func (r *tomcatInstanceRepository) Update(ctx context.Context, instance *model.TomcatInstance) error {
	return r.db.WithContext(ctx).Save(instance).Error
}

func (r *tomcatInstanceRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&model.TomcatInstance{}, id).Error
}

func (r *tomcatInstanceRepository) FindByID(ctx context.Context, id int64) (*model.TomcatInstance, error) {
	var instance model.TomcatInstance
	if err := r.db.WithContext(ctx).Preload("Host").Preload("Application").First(&instance, id).Error; err != nil {
		return nil, err
	}
	return &instance, nil
}

func (r *tomcatInstanceRepository) FindByAddress(ctx context.Context, address string) (*model.TomcatInstance, error) {
	var instance model.TomcatInstance
	if err := r.db.WithContext(ctx).Preload("Host").Preload("Application").Where("address = ?", address).First(&instance).Error; err != nil {
		return nil, err
	}
	return &instance, nil
}

func (r *tomcatInstanceRepository) ListByHostID(ctx context.Context, hostID int64) ([]model.TomcatInstance, error) {
	var instances []model.TomcatInstance
	if err := r.db.WithContext(ctx).
		Preload("Host").
		Preload("Application").
		Where("host_id = ?", hostID).
		Find(&instances).Error; err != nil {
		return nil, err
	}
	return instances, nil
}

func (r *tomcatInstanceRepository) List(ctx context.Context, opts ListOptions) ([]model.TomcatInstance, int64, error) {
	baseQuery := r.db.WithContext(ctx).Model(&model.TomcatInstance{})
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

	var instances []model.TomcatInstance
	if err := listQuery.Find(&instances).Error; err != nil {
		return nil, 0, err
	}

	return instances, total, nil
}
