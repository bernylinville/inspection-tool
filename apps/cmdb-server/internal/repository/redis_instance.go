package repository

import (
	"context"

	"gorm.io/gorm"

	"inspection-tool/apps/cmdb-server/internal/model"
)

type redisInstanceRepository struct {
	db *gorm.DB
}

// NewRedisInstanceRepository creates a new RedisInstanceRepository.
func NewRedisInstanceRepository(db *gorm.DB) RedisInstanceRepository {
	return &redisInstanceRepository{db: db}
}

func (r *redisInstanceRepository) Create(ctx context.Context, instance *model.RedisInstance) error {
	return r.db.WithContext(ctx).Create(instance).Error
}

func (r *redisInstanceRepository) Update(ctx context.Context, instance *model.RedisInstance) error {
	return r.db.WithContext(ctx).Save(instance).Error
}

func (r *redisInstanceRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&model.RedisInstance{}, id).Error
}

func (r *redisInstanceRepository) FindByID(ctx context.Context, id int64) (*model.RedisInstance, error) {
	var instance model.RedisInstance
	if err := r.db.WithContext(ctx).Preload("Host").Preload("Application").First(&instance, id).Error; err != nil {
		return nil, err
	}
	return &instance, nil
}

func (r *redisInstanceRepository) FindByAddress(ctx context.Context, address string) (*model.RedisInstance, error) {
	var instance model.RedisInstance
	if err := r.db.WithContext(ctx).Preload("Host").Preload("Application").Where("address = ?", address).First(&instance).Error; err != nil {
		return nil, err
	}
	return &instance, nil
}

func (r *redisInstanceRepository) ListByHostID(ctx context.Context, hostID int64) ([]model.RedisInstance, error) {
	var instances []model.RedisInstance
	if err := r.db.WithContext(ctx).
		Preload("Host").
		Preload("Application").
		Where("host_id = ?", hostID).
		Find(&instances).Error; err != nil {
		return nil, err
	}
	return instances, nil
}

func (r *redisInstanceRepository) List(ctx context.Context, opts ListOptions) ([]model.RedisInstance, int64, error) {
	baseQuery := r.db.WithContext(ctx).Model(&model.RedisInstance{})
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

	var instances []model.RedisInstance
	if err := listQuery.Find(&instances).Error; err != nil {
		return nil, 0, err
	}

	return instances, total, nil
}
