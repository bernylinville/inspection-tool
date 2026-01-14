package repository

import (
	"context"

	"gorm.io/gorm"

	"inspection-tool/apps/cmdb-server/internal/model"
)

type elasticsearchClusterRepository struct {
	db *gorm.DB
}

// NewElasticsearchClusterRepository creates a new ElasticsearchClusterRepository.
func NewElasticsearchClusterRepository(db *gorm.DB) ElasticsearchClusterRepository {
	return &elasticsearchClusterRepository{db: db}
}

func (r *elasticsearchClusterRepository) Create(ctx context.Context, cluster *model.ElasticsearchCluster) error {
	return r.db.WithContext(ctx).Create(cluster).Error
}

func (r *elasticsearchClusterRepository) Update(ctx context.Context, cluster *model.ElasticsearchCluster) error {
	return r.db.WithContext(ctx).Save(cluster).Error
}

func (r *elasticsearchClusterRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&model.ElasticsearchCluster{}, id).Error
}

func (r *elasticsearchClusterRepository) FindByID(ctx context.Context, id int64) (*model.ElasticsearchCluster, error) {
	var cluster model.ElasticsearchCluster
	if err := r.db.WithContext(ctx).Preload("Application").First(&cluster, id).Error; err != nil {
		return nil, err
	}
	return &cluster, nil
}

func (r *elasticsearchClusterRepository) FindByClusterName(ctx context.Context, clusterName string) (*model.ElasticsearchCluster, error) {
	var cluster model.ElasticsearchCluster
	if err := r.db.WithContext(ctx).Preload("Application").Where("cluster_name = ?", clusterName).First(&cluster).Error; err != nil {
		return nil, err
	}
	return &cluster, nil
}

func (r *elasticsearchClusterRepository) List(ctx context.Context, opts ListOptions) ([]model.ElasticsearchCluster, int64, error) {
	baseQuery := r.db.WithContext(ctx).Model(&model.ElasticsearchCluster{})
	filteredQuery := applyFilters(baseQuery, opts.Filters)

	var total int64
	if err := filteredQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	opts = normalizeListOptions(opts)
	listQuery := applyFilters(baseQuery, opts.Filters)
	listQuery = applyOrder(listQuery, opts)
	listQuery = applyPagination(listQuery, opts)
	listQuery = listQuery.Preload("Application")

	var clusters []model.ElasticsearchCluster
	if err := listQuery.Find(&clusters).Error; err != nil {
		return nil, 0, err
	}

	return clusters, total, nil
}
