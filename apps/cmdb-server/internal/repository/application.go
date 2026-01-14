package repository

import (
	"context"

	"gorm.io/gorm"

	"inspection-tool/apps/cmdb-server/internal/model"
)

type applicationRepository struct {
	db *gorm.DB
}

// NewApplicationRepository creates a new ApplicationRepository.
func NewApplicationRepository(db *gorm.DB) ApplicationRepository {
	return &applicationRepository{db: db}
}

func (r *applicationRepository) Create(ctx context.Context, application *model.Application) error {
	return r.db.WithContext(ctx).Create(application).Error
}

func (r *applicationRepository) Update(ctx context.Context, application *model.Application) error {
	return r.db.WithContext(ctx).Save(application).Error
}

func (r *applicationRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&model.Application{}, id).Error
}

func (r *applicationRepository) FindByID(ctx context.Context, id int64) (*model.Application, error) {
	var application model.Application
	if err := r.db.WithContext(ctx).Preload("Project").First(&application, id).Error; err != nil {
		return nil, err
	}
	return &application, nil
}

func (r *applicationRepository) FindByCode(ctx context.Context, code string) (*model.Application, error) {
	var application model.Application
	if err := r.db.WithContext(ctx).Preload("Project").Where("code = ?", code).First(&application).Error; err != nil {
		return nil, err
	}
	return &application, nil
}

func (r *applicationRepository) ListByProjectID(ctx context.Context, projectID int64) ([]model.Application, error) {
	var applications []model.Application
	if err := r.db.WithContext(ctx).Preload("Project").Where("project_id = ?", projectID).Find(&applications).Error; err != nil {
		return nil, err
	}
	return applications, nil
}

func (r *applicationRepository) List(ctx context.Context, opts ListOptions) ([]model.Application, int64, error) {
	baseQuery := r.db.WithContext(ctx).Model(&model.Application{})
	filteredQuery := applyFilters(baseQuery, opts.Filters)

	var total int64
	if err := filteredQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	opts = normalizeListOptions(opts)
	listQuery := applyFilters(baseQuery, opts.Filters)
	listQuery = applyOrder(listQuery, opts)
	listQuery = applyPagination(listQuery, opts)
	listQuery = listQuery.Preload("Project")

	var applications []model.Application
	if err := listQuery.Find(&applications).Error; err != nil {
		return nil, 0, err
	}

	return applications, total, nil
}
