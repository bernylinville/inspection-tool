package repository

import (
	"context"

	"gorm.io/gorm"

	"inspection-tool/apps/cmdb-server/internal/model"
)

// ListOptions defines pagination and filtering options for list queries.
type ListOptions struct {
	Page     int
	PageSize int
	OrderBy  string
	Order    string
	Filters  map[string]interface{}
}

// Repository defines common CRUD operations for entities.
type Repository[T any] interface {
	Create(ctx context.Context, entity *T) error
	Update(ctx context.Context, entity *T) error
	Delete(ctx context.Context, id int64) error
	FindByID(ctx context.Context, id int64) (*T, error)
	List(ctx context.Context, opts ListOptions) ([]T, int64, error)
}

// HostRepository defines host-specific data operations.
type HostRepository interface {
	Repository[model.Host]
	FindByIdent(ctx context.Context, ident string) (*model.Host, error)
	FindByIP(ctx context.Context, ip string) (*model.Host, error)
	ListByBusinessGroup(ctx context.Context, businessGroup string) ([]model.Host, error)
}

// ProjectRepository defines project-specific data operations.
type ProjectRepository interface {
	Repository[model.Project]
	FindByCode(ctx context.Context, code string) (*model.Project, error)
}

// ApplicationRepository defines application-specific data operations.
type ApplicationRepository interface {
	Repository[model.Application]
	FindByCode(ctx context.Context, code string) (*model.Application, error)
	ListByProjectID(ctx context.Context, projectID int64) ([]model.Application, error)
}

// MySQLInstanceRepository defines MySQL instance-specific data operations.
type MySQLInstanceRepository interface {
	Repository[model.MySQLInstance]
	FindByAddress(ctx context.Context, address string) (*model.MySQLInstance, error)
	ListByHostID(ctx context.Context, hostID int64) ([]model.MySQLInstance, error)
}

// RedisInstanceRepository defines Redis instance-specific data operations.
type RedisInstanceRepository interface {
	Repository[model.RedisInstance]
	FindByAddress(ctx context.Context, address string) (*model.RedisInstance, error)
	ListByHostID(ctx context.Context, hostID int64) ([]model.RedisInstance, error)
}

// NginxInstanceRepository defines Nginx instance-specific data operations.
type NginxInstanceRepository interface {
	Repository[model.NginxInstance]
	FindByAddress(ctx context.Context, address string) (*model.NginxInstance, error)
	ListByHostID(ctx context.Context, hostID int64) ([]model.NginxInstance, error)
}

// TomcatInstanceRepository defines Tomcat instance-specific data operations.
type TomcatInstanceRepository interface {
	Repository[model.TomcatInstance]
	FindByAddress(ctx context.Context, address string) (*model.TomcatInstance, error)
	ListByHostID(ctx context.Context, hostID int64) ([]model.TomcatInstance, error)
}

// ElasticsearchClusterRepository defines Elasticsearch cluster-specific data operations.
type ElasticsearchClusterRepository interface {
	Repository[model.ElasticsearchCluster]
	FindByClusterName(ctx context.Context, clusterName string) (*model.ElasticsearchCluster, error)
}

// UserRepository defines user-specific data operations.
type UserRepository interface {
	Repository[model.User]
	FindByUsername(ctx context.Context, username string) (*model.User, error)
}

// RoleRepository defines role-specific data operations.
type RoleRepository interface {
	Repository[model.Role]
	FindByName(ctx context.Context, name string) (*model.Role, error)
}

// PermissionRepository defines permission-specific data operations.
type PermissionRepository interface {
	Repository[model.Permission]
	FindByName(ctx context.Context, name string) (*model.Permission, error)
	ListByResource(ctx context.Context, resource string) ([]model.Permission, error)
}

// InspectionJobRepository defines inspection job data operations.
type InspectionJobRepository interface {
	Repository[model.InspectionJob]
	ListByStatus(ctx context.Context, status string) ([]model.InspectionJob, error)
	ListByType(ctx context.Context, jobType string) ([]model.InspectionJob, error)
}

const defaultPageSize = 20

func normalizeListOptions(opts ListOptions) ListOptions {
	if opts.Page <= 0 {
		opts.Page = 1
	}
	if opts.PageSize <= 0 {
		opts.PageSize = defaultPageSize
	}
	return opts
}

func applyFilters(db *gorm.DB, filters map[string]interface{}) *gorm.DB {
	if len(filters) == 0 {
		return db
	}
	return db.Where(filters)
}

func applyOrder(db *gorm.DB, opts ListOptions) *gorm.DB {
	if opts.OrderBy == "" {
		return db
	}
	order := opts.OrderBy
	if opts.Order != "" {
		order = opts.OrderBy + " " + opts.Order
	}
	return db.Order(order)
}

func applyPagination(db *gorm.DB, opts ListOptions) *gorm.DB {
	offset := (opts.Page - 1) * opts.PageSize
	return db.Offset(offset).Limit(opts.PageSize)
}
