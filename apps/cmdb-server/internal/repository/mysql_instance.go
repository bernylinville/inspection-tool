package repository

import (
	"context"

	"gorm.io/gorm"

	"inspection-tool/apps/cmdb-server/internal/model"
)

type mySQLInstanceRepository struct {
	db *gorm.DB
}

// NewMySQLInstanceRepository creates a new MySQLInstanceRepository.
func NewMySQLInstanceRepository(db *gorm.DB) MySQLInstanceRepository {
	return &mySQLInstanceRepository{db: db}
}

func (r *mySQLInstanceRepository) Create(ctx context.Context, instance *model.MySQLInstance) error {
	return r.db.WithContext(ctx).Create(instance).Error
}

func (r *mySQLInstanceRepository) Update(ctx context.Context, instance *model.MySQLInstance) error {
	return r.db.WithContext(ctx).Save(instance).Error
}

func (r *mySQLInstanceRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&model.MySQLInstance{}, id).Error
}

func (r *mySQLInstanceRepository) FindByID(ctx context.Context, id int64) (*model.MySQLInstance, error) {
	var instance model.MySQLInstance
	if err := r.db.WithContext(ctx).Preload("Host").Preload("Application").First(&instance, id).Error; err != nil {
		return nil, err
	}
	return &instance, nil
}

func (r *mySQLInstanceRepository) FindByAddress(ctx context.Context, address string) (*model.MySQLInstance, error) {
	var instance model.MySQLInstance
	if err := r.db.WithContext(ctx).Preload("Host").Preload("Application").Where("address = ?", address).First(&instance).Error; err != nil {
		return nil, err
	}
	return &instance, nil
}

func (r *mySQLInstanceRepository) ListByHostID(ctx context.Context, hostID int64) ([]model.MySQLInstance, error) {
	var instances []model.MySQLInstance
	if err := r.db.WithContext(ctx).
		Preload("Host").
		Preload("Application").
		Where("host_id = ?", hostID).
		Find(&instances).Error; err != nil {
		return nil, err
	}
	return instances, nil
}

func (r *mySQLInstanceRepository) List(ctx context.Context, opts ListOptions) ([]model.MySQLInstance, int64, error) {
	baseQuery := r.db.WithContext(ctx).Model(&model.MySQLInstance{})
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

	var instances []model.MySQLInstance
	if err := listQuery.Find(&instances).Error; err != nil {
		return nil, 0, err
	}

	return instances, total, nil
}
