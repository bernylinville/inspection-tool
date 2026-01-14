package repository

import (
	"context"

	"gorm.io/gorm"

	"inspection-tool/apps/cmdb-server/internal/model"
)

type inspectionJobRepository struct {
	db *gorm.DB
}

// NewInspectionJobRepository creates a new InspectionJobRepository.
func NewInspectionJobRepository(db *gorm.DB) InspectionJobRepository {
	return &inspectionJobRepository{db: db}
}

func (r *inspectionJobRepository) Create(ctx context.Context, job *model.InspectionJob) error {
	return r.db.WithContext(ctx).Create(job).Error
}

func (r *inspectionJobRepository) Update(ctx context.Context, job *model.InspectionJob) error {
	return r.db.WithContext(ctx).Save(job).Error
}

func (r *inspectionJobRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&model.InspectionJob{}, id).Error
}

func (r *inspectionJobRepository) FindByID(ctx context.Context, id int64) (*model.InspectionJob, error) {
	var job model.InspectionJob
	if err := r.db.WithContext(ctx).First(&job, id).Error; err != nil {
		return nil, err
	}
	return &job, nil
}

func (r *inspectionJobRepository) ListByStatus(ctx context.Context, status string) ([]model.InspectionJob, error) {
	var jobs []model.InspectionJob
	if err := r.db.WithContext(ctx).Where("status = ?", status).Find(&jobs).Error; err != nil {
		return nil, err
	}
	return jobs, nil
}

func (r *inspectionJobRepository) ListByType(ctx context.Context, jobType string) ([]model.InspectionJob, error) {
	var jobs []model.InspectionJob
	if err := r.db.WithContext(ctx).Where("type = ?", jobType).Find(&jobs).Error; err != nil {
		return nil, err
	}
	return jobs, nil
}

func (r *inspectionJobRepository) List(ctx context.Context, opts ListOptions) ([]model.InspectionJob, int64, error) {
	baseQuery := r.db.WithContext(ctx).Model(&model.InspectionJob{})
	filteredQuery := applyFilters(baseQuery, opts.Filters)

	var total int64
	if err := filteredQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	opts = normalizeListOptions(opts)
	listQuery := applyFilters(baseQuery, opts.Filters)
	listQuery = applyOrder(listQuery, opts)
	listQuery = applyPagination(listQuery, opts)

	var jobs []model.InspectionJob
	if err := listQuery.Find(&jobs).Error; err != nil {
		return nil, 0, err
	}

	return jobs, total, nil
}
