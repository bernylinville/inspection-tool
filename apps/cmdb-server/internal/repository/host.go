package repository

import (
	"context"

	"gorm.io/gorm"

	"inspection-tool/apps/cmdb-server/internal/model"
)

type hostRepository struct {
	db *gorm.DB
}

// NewHostRepository creates a new HostRepository.
func NewHostRepository(db *gorm.DB) HostRepository {
	return &hostRepository{db: db}
}

func (r *hostRepository) Create(ctx context.Context, host *model.Host) error {
	return r.db.WithContext(ctx).Create(host).Error
}

func (r *hostRepository) Update(ctx context.Context, host *model.Host) error {
	return r.db.WithContext(ctx).Save(host).Error
}

func (r *hostRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&model.Host{}, id).Error
}

func (r *hostRepository) FindByID(ctx context.Context, id int64) (*model.Host, error) {
	var host model.Host
	if err := r.db.WithContext(ctx).First(&host, id).Error; err != nil {
		return nil, err
	}
	return &host, nil
}

func (r *hostRepository) FindByIdent(ctx context.Context, ident string) (*model.Host, error) {
	var host model.Host
	if err := r.db.WithContext(ctx).Where("ident = ?", ident).First(&host).Error; err != nil {
		return nil, err
	}
	return &host, nil
}

func (r *hostRepository) FindByIP(ctx context.Context, ip string) (*model.Host, error) {
	var host model.Host
	if err := r.db.WithContext(ctx).Where("ip = ?", ip).First(&host).Error; err != nil {
		return nil, err
	}
	return &host, nil
}

func (r *hostRepository) ListByBusinessGroup(ctx context.Context, businessGroup string) ([]model.Host, error) {
	var hosts []model.Host
	if err := r.db.WithContext(ctx).Where("business_group = ?", businessGroup).Find(&hosts).Error; err != nil {
		return nil, err
	}
	return hosts, nil
}

func (r *hostRepository) List(ctx context.Context, opts ListOptions) ([]model.Host, int64, error) {
	baseQuery := r.db.WithContext(ctx).Model(&model.Host{})
	filteredQuery := applyFilters(baseQuery, opts.Filters)

	var total int64
	if err := filteredQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	opts = normalizeListOptions(opts)
	listQuery := applyFilters(baseQuery, opts.Filters)
	listQuery = applyOrder(listQuery, opts)
	listQuery = applyPagination(listQuery, opts)

	var hosts []model.Host
	if err := listQuery.Find(&hosts).Error; err != nil {
		return nil, 0, err
	}

	return hosts, total, nil
}
