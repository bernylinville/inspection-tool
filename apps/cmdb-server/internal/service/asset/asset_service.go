package asset

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"inspection-tool/apps/cmdb-server/internal/model"
	"inspection-tool/apps/cmdb-server/internal/repository"
)

// Errors
var (
	ErrProjectNotFound       = errors.New("project not found")
	ErrProjectCodeExists     = errors.New("project code already exists")
	ErrApplicationNotFound   = errors.New("application not found")
	ErrApplicationCodeExists = errors.New("application code already exists")
	ErrHostNotFound          = errors.New("host not found")
	ErrHostIdentExists       = errors.New("host ident already exists")
)

// CreateProjectRequest represents the request to create a project
type CreateProjectRequest struct {
	Name        string `json:"name" binding:"required"`
	Code        string `json:"code" binding:"required"`
	Description string `json:"description"`
	Owner       string `json:"owner"`
}

// UpdateProjectRequest represents the request to update a project
type UpdateProjectRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Owner       *string `json:"owner"`
	Status      *string `json:"status"`
}

// CreateApplicationRequest represents the request to create an application
type CreateApplicationRequest struct {
	ProjectID   int64  `json:"project_id" binding:"required"`
	Name        string `json:"name" binding:"required"`
	Code        string `json:"code" binding:"required"`
	Description string `json:"description"`
	Owner       string `json:"owner"`
}

// UpdateApplicationRequest represents the request to update an application
type UpdateApplicationRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Owner       *string `json:"owner"`
	Status      *string `json:"status"`
}

// CreateHostRequest represents the request to create a host
type CreateHostRequest struct {
	Ident         string            `json:"ident" binding:"required"`
	Hostname      string            `json:"hostname"`
	IP            string            `json:"ip"`
	OS            string            `json:"os"`
	OSVersion     string            `json:"os_version"`
	KernelVersion string            `json:"kernel_version"`
	CPUCores      int               `json:"cpu_cores"`
	CPUModel      string            `json:"cpu_model"`
	MemoryTotal   int64             `json:"memory_total"`
	BusinessGroup string            `json:"business_group"`
	Env           string            `json:"env"`
	Tags          map[string]string `json:"tags"`
}

// UpdateHostRequest represents the request to update a host
type UpdateHostRequest struct {
	Hostname      *string           `json:"hostname"`
	IP            *string           `json:"ip"`
	OS            *string           `json:"os"`
	OSVersion     *string           `json:"os_version"`
	KernelVersion *string           `json:"kernel_version"`
	CPUCores      *int              `json:"cpu_cores"`
	CPUModel      *string           `json:"cpu_model"`
	MemoryTotal   *int64            `json:"memory_total"`
	BusinessGroup *string           `json:"business_group"`
	Env           *string           `json:"env"`
	Tags          map[string]string `json:"tags"`
	Status        *string           `json:"status"`
}

// AssetService provides asset management operations
type AssetService struct {
	db                       *gorm.DB
	projectRepo              repository.ProjectRepository
	applicationRepo          repository.ApplicationRepository
	hostRepo                 repository.HostRepository
	mysqlInstanceRepo        repository.MySQLInstanceRepository
	redisInstanceRepo        repository.RedisInstanceRepository
	nginxInstanceRepo        repository.NginxInstanceRepository
	tomcatInstanceRepo       repository.TomcatInstanceRepository
	elasticsearchClusterRepo repository.ElasticsearchClusterRepository
}

// NewAssetService creates a new AssetService
func NewAssetService(
	db *gorm.DB,
	projectRepo repository.ProjectRepository,
	applicationRepo repository.ApplicationRepository,
	hostRepo repository.HostRepository,
	mysqlInstanceRepo repository.MySQLInstanceRepository,
	redisInstanceRepo repository.RedisInstanceRepository,
	nginxInstanceRepo repository.NginxInstanceRepository,
	tomcatInstanceRepo repository.TomcatInstanceRepository,
	elasticsearchClusterRepo repository.ElasticsearchClusterRepository,
) *AssetService {
	return &AssetService{
		db:                       db,
		projectRepo:              projectRepo,
		applicationRepo:          applicationRepo,
		hostRepo:                 hostRepo,
		mysqlInstanceRepo:        mysqlInstanceRepo,
		redisInstanceRepo:        redisInstanceRepo,
		nginxInstanceRepo:        nginxInstanceRepo,
		tomcatInstanceRepo:       tomcatInstanceRepo,
		elasticsearchClusterRepo: elasticsearchClusterRepo,
	}
}

// ==================== Project Operations ====================

// CreateProject creates a new project
func (s *AssetService) CreateProject(ctx context.Context, req *CreateProjectRequest) (*model.Project, error) {
	// Check if code already exists
	existing, err := s.projectRepo.FindByCode(ctx, req.Code)
	if err == nil && existing != nil {
		return nil, ErrProjectCodeExists
	}

	project := &model.Project{
		Name:        req.Name,
		Code:        req.Code,
		Description: req.Description,
		Owner:       req.Owner,
		Status:      "active",
	}

	if err := s.projectRepo.Create(ctx, project); err != nil {
		return nil, err
	}

	return project, nil
}

// UpdateProject updates an existing project
func (s *AssetService) UpdateProject(ctx context.Context, id int64, req *UpdateProjectRequest) (*model.Project, error) {
	project, err := s.projectRepo.FindByID(ctx, id)
	if err != nil {
		return nil, ErrProjectNotFound
	}

	if req.Name != nil {
		project.Name = *req.Name
	}
	if req.Description != nil {
		project.Description = *req.Description
	}
	if req.Owner != nil {
		project.Owner = *req.Owner
	}
	if req.Status != nil {
		project.Status = *req.Status
	}

	if err := s.projectRepo.Update(ctx, project); err != nil {
		return nil, err
	}

	return project, nil
}

// DeleteProject deletes a project
func (s *AssetService) DeleteProject(ctx context.Context, id int64) error {
	_, err := s.projectRepo.FindByID(ctx, id)
	if err != nil {
		return ErrProjectNotFound
	}

	return s.projectRepo.Delete(ctx, id)
}

// GetProject retrieves a project by ID
func (s *AssetService) GetProject(ctx context.Context, id int64) (*model.Project, error) {
	project, err := s.projectRepo.FindByID(ctx, id)
	if err != nil {
		return nil, ErrProjectNotFound
	}
	return project, nil
}

// ListProjects lists projects with pagination
func (s *AssetService) ListProjects(ctx context.Context, opts repository.ListOptions) ([]model.Project, int64, error) {
	return s.projectRepo.List(ctx, opts)
}

// ==================== Application Operations ====================

// CreateApplication creates a new application
func (s *AssetService) CreateApplication(ctx context.Context, req *CreateApplicationRequest) (*model.Application, error) {
	// Check if project exists
	_, err := s.projectRepo.FindByID(ctx, req.ProjectID)
	if err != nil {
		return nil, ErrProjectNotFound
	}

	// Check if code already exists
	existing, err := s.applicationRepo.FindByCode(ctx, req.Code)
	if err == nil && existing != nil {
		return nil, ErrApplicationCodeExists
	}

	app := &model.Application{
		ProjectID:   req.ProjectID,
		Name:        req.Name,
		Code:        req.Code,
		Description: req.Description,
		Owner:       req.Owner,
		Status:      "active",
	}

	if err := s.applicationRepo.Create(ctx, app); err != nil {
		return nil, err
	}

	return app, nil
}

// UpdateApplication updates an existing application
func (s *AssetService) UpdateApplication(ctx context.Context, id int64, req *UpdateApplicationRequest) (*model.Application, error) {
	app, err := s.applicationRepo.FindByID(ctx, id)
	if err != nil {
		return nil, ErrApplicationNotFound
	}

	if req.Name != nil {
		app.Name = *req.Name
	}
	if req.Description != nil {
		app.Description = *req.Description
	}
	if req.Owner != nil {
		app.Owner = *req.Owner
	}
	if req.Status != nil {
		app.Status = *req.Status
	}

	if err := s.applicationRepo.Update(ctx, app); err != nil {
		return nil, err
	}

	return app, nil
}

// DeleteApplication deletes an application
func (s *AssetService) DeleteApplication(ctx context.Context, id int64) error {
	_, err := s.applicationRepo.FindByID(ctx, id)
	if err != nil {
		return ErrApplicationNotFound
	}

	return s.applicationRepo.Delete(ctx, id)
}

// GetApplication retrieves an application by ID
func (s *AssetService) GetApplication(ctx context.Context, id int64) (*model.Application, error) {
	app, err := s.applicationRepo.FindByID(ctx, id)
	if err != nil {
		return nil, ErrApplicationNotFound
	}
	return app, nil
}

// ListApplications lists applications with pagination
func (s *AssetService) ListApplications(ctx context.Context, opts repository.ListOptions) ([]model.Application, int64, error) {
	return s.applicationRepo.List(ctx, opts)
}

// ListApplicationsByProject lists applications by project ID
func (s *AssetService) ListApplicationsByProject(ctx context.Context, projectID int64) ([]model.Application, error) {
	return s.applicationRepo.ListByProjectID(ctx, projectID)
}

// ==================== Host Operations ====================

// CreateHost creates a new host
func (s *AssetService) CreateHost(ctx context.Context, req *CreateHostRequest) (*model.Host, error) {
	// Check if ident already exists
	existing, err := s.hostRepo.FindByIdent(ctx, req.Ident)
	if err == nil && existing != nil {
		return nil, ErrHostIdentExists
	}

	host := &model.Host{
		Ident:         req.Ident,
		Hostname:      req.Hostname,
		IP:            req.IP,
		OS:            req.OS,
		OSVersion:     req.OSVersion,
		KernelVersion: req.KernelVersion,
		CPUCores:      req.CPUCores,
		CPUModel:      req.CPUModel,
		MemoryTotal:   req.MemoryTotal,
		BusinessGroup: req.BusinessGroup,
		Env:           req.Env,
		Status:        "active",
	}

	if err := s.hostRepo.Create(ctx, host); err != nil {
		return nil, err
	}

	return host, nil
}

// UpdateHost updates an existing host
func (s *AssetService) UpdateHost(ctx context.Context, id int64, req *UpdateHostRequest) (*model.Host, error) {
	host, err := s.hostRepo.FindByID(ctx, id)
	if err != nil {
		return nil, ErrHostNotFound
	}

	if req.Hostname != nil {
		host.Hostname = *req.Hostname
	}
	if req.IP != nil {
		host.IP = *req.IP
	}
	if req.OS != nil {
		host.OS = *req.OS
	}
	if req.OSVersion != nil {
		host.OSVersion = *req.OSVersion
	}
	if req.KernelVersion != nil {
		host.KernelVersion = *req.KernelVersion
	}
	if req.CPUCores != nil {
		host.CPUCores = *req.CPUCores
	}
	if req.CPUModel != nil {
		host.CPUModel = *req.CPUModel
	}
	if req.MemoryTotal != nil {
		host.MemoryTotal = *req.MemoryTotal
	}
	if req.BusinessGroup != nil {
		host.BusinessGroup = *req.BusinessGroup
	}
	if req.Env != nil {
		host.Env = *req.Env
	}
	if req.Status != nil {
		host.Status = *req.Status
	}

	if err := s.hostRepo.Update(ctx, host); err != nil {
		return nil, err
	}

	return host, nil
}

// DeleteHost deletes a host
func (s *AssetService) DeleteHost(ctx context.Context, id int64) error {
	_, err := s.hostRepo.FindByID(ctx, id)
	if err != nil {
		return ErrHostNotFound
	}

	return s.hostRepo.Delete(ctx, id)
}

// GetHost retrieves a host by ID
func (s *AssetService) GetHost(ctx context.Context, id int64) (*model.Host, error) {
	host, err := s.hostRepo.FindByID(ctx, id)
	if err != nil {
		return nil, ErrHostNotFound
	}
	return host, nil
}

// ListHosts lists hosts with pagination
func (s *AssetService) ListHosts(ctx context.Context, opts repository.ListOptions) ([]model.Host, int64, error) {
	return s.hostRepo.List(ctx, opts)
}

// ListHostsByBusinessGroup lists hosts by business group
func (s *AssetService) ListHostsByBusinessGroup(ctx context.Context, businessGroup string) ([]model.Host, error) {
	return s.hostRepo.ListByBusinessGroup(ctx, businessGroup)
}

// AssociateHostToApplication associates a host with an application
func (s *AssetService) AssociateHostToApplication(ctx context.Context, appID, hostID int64) error {
	_, err := s.applicationRepo.FindByID(ctx, appID)
	if err != nil {
		return ErrApplicationNotFound
	}

	_, err = s.hostRepo.FindByID(ctx, hostID)
	if err != nil {
		return ErrHostNotFound
	}

	assoc := &model.ApplicationHost{
		ApplicationID: appID,
		HostID:        hostID,
	}

	return s.db.WithContext(ctx).Create(assoc).Error
}

// DisassociateHostFromApplication removes the association between a host and an application
func (s *AssetService) DisassociateHostFromApplication(ctx context.Context, appID, hostID int64) error {
	return s.db.WithContext(ctx).
		Where("application_id = ? AND host_id = ?", appID, hostID).
		Delete(&model.ApplicationHost{}).Error
}

// ==================== MySQL Instance Operations ====================

func (s *AssetService) GetMySQLInstance(ctx context.Context, id int64) (*model.MySQLInstance, error) {
	return s.mysqlInstanceRepo.FindByID(ctx, id)
}

func (s *AssetService) ListMySQLInstances(ctx context.Context, opts repository.ListOptions) ([]model.MySQLInstance, int64, error) {
	return s.mysqlInstanceRepo.List(ctx, opts)
}

func (s *AssetService) ListMySQLInstancesByHost(ctx context.Context, hostID int64) ([]model.MySQLInstance, error) {
	return s.mysqlInstanceRepo.ListByHostID(ctx, hostID)
}

func (s *AssetService) DeleteMySQLInstance(ctx context.Context, id int64) error {
	return s.mysqlInstanceRepo.Delete(ctx, id)
}

func (s *AssetService) GetRedisInstance(ctx context.Context, id int64) (*model.RedisInstance, error) {
	return s.redisInstanceRepo.FindByID(ctx, id)
}

func (s *AssetService) ListRedisInstances(ctx context.Context, opts repository.ListOptions) ([]model.RedisInstance, int64, error) {
	return s.redisInstanceRepo.List(ctx, opts)
}

func (s *AssetService) ListRedisInstancesByHost(ctx context.Context, hostID int64) ([]model.RedisInstance, error) {
	return s.redisInstanceRepo.ListByHostID(ctx, hostID)
}

func (s *AssetService) DeleteRedisInstance(ctx context.Context, id int64) error {
	return s.redisInstanceRepo.Delete(ctx, id)
}

func (s *AssetService) GetNginxInstance(ctx context.Context, id int64) (*model.NginxInstance, error) {
	return s.nginxInstanceRepo.FindByID(ctx, id)
}

func (s *AssetService) ListNginxInstances(ctx context.Context, opts repository.ListOptions) ([]model.NginxInstance, int64, error) {
	return s.nginxInstanceRepo.List(ctx, opts)
}

func (s *AssetService) ListNginxInstancesByHost(ctx context.Context, hostID int64) ([]model.NginxInstance, error) {
	return s.nginxInstanceRepo.ListByHostID(ctx, hostID)
}

func (s *AssetService) DeleteNginxInstance(ctx context.Context, id int64) error {
	return s.nginxInstanceRepo.Delete(ctx, id)
}

func (s *AssetService) GetTomcatInstance(ctx context.Context, id int64) (*model.TomcatInstance, error) {
	return s.tomcatInstanceRepo.FindByID(ctx, id)
}

func (s *AssetService) ListTomcatInstances(ctx context.Context, opts repository.ListOptions) ([]model.TomcatInstance, int64, error) {
	return s.tomcatInstanceRepo.List(ctx, opts)
}

func (s *AssetService) ListTomcatInstancesByHost(ctx context.Context, hostID int64) ([]model.TomcatInstance, error) {
	return s.tomcatInstanceRepo.ListByHostID(ctx, hostID)
}

func (s *AssetService) DeleteTomcatInstance(ctx context.Context, id int64) error {
	return s.tomcatInstanceRepo.Delete(ctx, id)
}

func (s *AssetService) GetElasticsearchCluster(ctx context.Context, id int64) (*model.ElasticsearchCluster, error) {
	return s.elasticsearchClusterRepo.FindByID(ctx, id)
}

func (s *AssetService) ListElasticsearchClusters(ctx context.Context, opts repository.ListOptions) ([]model.ElasticsearchCluster, int64, error) {
	return s.elasticsearchClusterRepo.List(ctx, opts)
}

func (s *AssetService) DeleteElasticsearchCluster(ctx context.Context, id int64) error {
	return s.elasticsearchClusterRepo.Delete(ctx, id)
}
