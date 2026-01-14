package repository

import (
	"context"

	"gorm.io/gorm"

	"inspection-tool/apps/cmdb-server/internal/model"
)

type projectRepository struct {
	db *gorm.DB
}

// NewProjectRepository creates a new ProjectRepository.
func NewProjectRepository(db *gorm.DB) ProjectRepository {
	return &projectRepository{db: db}
}

func (r *projectRepository) Create(ctx context.Context, project *model.Project) error {
	return r.db.WithContext(ctx).Create(project).Error
}

func (r *projectRepository) Update(ctx context.Context, project *model.Project) error {
	return r.db.WithContext(ctx).Save(project).Error
}

func (r *projectRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&model.Project{}, id).Error
}

func (r *projectRepository) FindByID(ctx context.Context, id int64) (*model.Project, error) {
	var project model.Project
	if err := r.db.WithContext(ctx).Preload("Applications").First(&project, id).Error; err != nil {
		return nil, err
	}
	return &project, nil
}

func (r *projectRepository) FindByCode(ctx context.Context, code string) (*model.Project, error) {
	var project model.Project
	if err := r.db.WithContext(ctx).Preload("Applications").Where("code = ?", code).First(&project).Error; err != nil {
		return nil, err
	}
	return &project, nil
}

func (r *projectRepository) List(ctx context.Context, opts ListOptions) ([]model.Project, int64, error) {
	baseQuery := r.db.WithContext(ctx).Model(&model.Project{})
	filteredQuery := applyFilters(baseQuery, opts.Filters)

	var total int64
	if err := filteredQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	opts = normalizeListOptions(opts)
	listQuery := applyFilters(baseQuery, opts.Filters)
	listQuery = applyOrder(listQuery, opts)
	listQuery = applyPagination(listQuery, opts)
	listQuery = listQuery.Preload("Applications")

	var projects []model.Project
	if err := listQuery.Find(&projects).Error; err != nil {
		return nil, 0, err
	}

	return projects, total, nil
}
