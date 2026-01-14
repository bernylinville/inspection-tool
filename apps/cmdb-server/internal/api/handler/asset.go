package handler

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"inspection-tool/apps/cmdb-server/internal/model"
	"inspection-tool/apps/cmdb-server/internal/repository"
	"inspection-tool/apps/cmdb-server/internal/service/asset"
	"inspection-tool/apps/cmdb-server/internal/service/sync"
)

// ==================== Request/Response Structs ====================

// Project Request/Response
type CreateProjectRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=100"`
	Code        string `json:"code" binding:"required,min=1,max=50"`
	Description string `json:"description" binding:"omitempty,max=500"`
	Owner       string `json:"owner" binding:"omitempty,max=100"`
}

type UpdateProjectRequest struct {
	Name        *string `json:"name" binding:"omitempty,min=1,max=100"`
	Description *string `json:"description" binding:"omitempty,max=500"`
	Owner       *string `json:"owner" binding:"omitempty,max=100"`
	Status      *string `json:"status" binding:"omitempty,oneof=active inactive"`
}

type ProjectResponse struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Code        string `json:"code"`
	Description string `json:"description"`
	Owner       string `json:"owner"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type ProjectListResponse struct {
	Code    int              `json:"code"`
	Message string           `json:"message"`
	Data    *ProjectListData `json:"data,omitempty"`
}

type ProjectListData struct {
	Items      []ProjectResponse `json:"items"`
	Total      int64             `json:"total"`
	Page       int               `json:"page"`
	PageSize   int               `json:"page_size"`
	TotalPages int               `json:"total_pages"`
}

type ProjectDetailResponse struct {
	Code    int              `json:"code"`
	Message string           `json:"message"`
	Data    *ProjectResponse `json:"data,omitempty"`
}

// Application Request/Response
type CreateApplicationRequest struct {
	ProjectID   int64  `json:"project_id" binding:"required"`
	Name        string `json:"name" binding:"required,min=1,max=100"`
	Code        string `json:"code" binding:"required,min=1,max=50"`
	Description string `json:"description" binding:"omitempty,max=500"`
	Owner       string `json:"owner" binding:"omitempty,max=100"`
}

type UpdateApplicationRequest struct {
	Name        *string `json:"name" binding:"omitempty,min=1,max=100"`
	Description *string `json:"description" binding:"omitempty,max=500"`
	Owner       *string `json:"owner" binding:"omitempty,max=100"`
	Status      *string `json:"status" binding:"omitempty,oneof=active inactive"`
}

type ApplicationResponse struct {
	ID          int64  `json:"id"`
	ProjectID   int64  `json:"project_id"`
	Name        string `json:"name"`
	Code        string `json:"code"`
	Description string `json:"description"`
	Owner       string `json:"owner"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type ApplicationListResponse struct {
	Code    int                  `json:"code"`
	Message string               `json:"message"`
	Data    *ApplicationListData `json:"data,omitempty"`
}

type ApplicationListData struct {
	Items      []ApplicationResponse `json:"items"`
	Total      int64                 `json:"total"`
	Page       int                   `json:"page"`
	PageSize   int                   `json:"page_size"`
	TotalPages int                   `json:"total_pages"`
}

type ApplicationDetailResponse struct {
	Code    int                  `json:"code"`
	Message string               `json:"message"`
	Data    *ApplicationResponse `json:"data,omitempty"`
}

// Host Request/Response
type CreateHostRequest struct {
	Ident         string            `json:"ident" binding:"required,min=1,max=100"`
	Hostname      string            `json:"hostname" binding:"omitempty,max=255"`
	IP            string            `json:"ip" binding:"omitempty,max=45"`
	OS            string            `json:"os" binding:"omitempty,max=50"`
	OSVersion     string            `json:"os_version" binding:"omitempty,max=100"`
	KernelVersion string            `json:"kernel_version" binding:"omitempty,max=100"`
	CPUCores      int               `json:"cpu_cores" binding:"omitempty,min=0"`
	CPUModel      string            `json:"cpu_model" binding:"omitempty,max=200"`
	MemoryTotal   int64             `json:"memory_total" binding:"omitempty,min=0"`
	BusinessGroup string            `json:"business_group" binding:"omitempty,max=100"`
	Env           string            `json:"env" binding:"omitempty,max=50"`
	Tags          map[string]string `json:"tags" binding:"omitempty"`
}

type UpdateHostRequest struct {
	Hostname      *string           `json:"hostname" binding:"omitempty,max=255"`
	IP            *string           `json:"ip" binding:"omitempty,max=45"`
	OS            *string           `json:"os" binding:"omitempty,max=50"`
	OSVersion     *string           `json:"os_version" binding:"omitempty,max=100"`
	KernelVersion *string           `json:"kernel_version" binding:"omitempty,max=100"`
	CPUCores      *int              `json:"cpu_cores" binding:"omitempty,min=0"`
	CPUModel      *string           `json:"cpu_model" binding:"omitempty,max=200"`
	MemoryTotal   *int64            `json:"memory_total" binding:"omitempty,min=0"`
	BusinessGroup *string           `json:"business_group" binding:"omitempty,max=100"`
	Env           *string           `json:"env" binding:"omitempty,max=50"`
	Tags          map[string]string `json:"tags" binding:"omitempty"`
	Status        *string           `json:"status" binding:"omitempty,oneof=active inactive"`
}

type HostResponse struct {
	ID            int64             `json:"id"`
	Ident         string            `json:"ident"`
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
	Tags          map[string]string `json:"tags,omitempty"`
	Status        string            `json:"status"`
	LastSyncAt    *string           `json:"last_sync_at,omitempty"`
	CreatedAt     string            `json:"created_at"`
	UpdatedAt     string            `json:"updated_at"`
}

type HostListResponse struct {
	Code    int           `json:"code"`
	Message string        `json:"message"`
	Data    *HostListData `json:"data,omitempty"`
}

type HostListData struct {
	Items      []HostResponse `json:"items"`
	Total      int64          `json:"total"`
	Page       int            `json:"page"`
	PageSize   int            `json:"page_size"`
	TotalPages int            `json:"total_pages"`
}

type HostDetailResponse struct {
	Code    int           `json:"code"`
	Message string        `json:"message"`
	Data    *HostResponse `json:"data,omitempty"`
}

type SyncHostsResponse struct {
	Code    int            `json:"code"`
	Message string         `json:"message"`
	Data    *SyncHostsData `json:"data,omitempty"`
}

type SyncHostsData struct {
	TotalHosts   int    `json:"total_hosts"`
	NewHosts     int    `json:"new_hosts"`
	UpdatedHosts int    `json:"updated_hosts"`
	FailedHosts  int    `json:"failed_hosts"`
	Duration     string `json:"duration"`
}

// Middleware Instance Responses
type MySQLInstanceResponse struct {
	ID            int64   `json:"id"`
	Address       string  `json:"address"`
	IP            string  `json:"ip"`
	Port          int     `json:"port"`
	Version       string  `json:"version"`
	ClusterMode   string  `json:"cluster_mode"`
	ServerID      string  `json:"server_id"`
	HostID        *int64  `json:"host_id,omitempty"`
	ApplicationID *int64  `json:"application_id,omitempty"`
	Status        string  `json:"status"`
	LastSyncAt    *string `json:"last_sync_at,omitempty"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
}

type MySQLInstanceListResponse struct {
	Code    int                    `json:"code"`
	Message string                 `json:"message"`
	Data    *MySQLInstanceListData `json:"data,omitempty"`
}

type MySQLInstanceListData struct {
	Items      []MySQLInstanceResponse `json:"items"`
	Total      int64                   `json:"total"`
	Page       int                     `json:"page"`
	PageSize   int                     `json:"page_size"`
	TotalPages int                     `json:"total_pages"`
}

type MySQLInstanceDetailResponse struct {
	Code    int                    `json:"code"`
	Message string                 `json:"message"`
	Data    *MySQLInstanceResponse `json:"data,omitempty"`
}

type RedisInstanceResponse struct {
	ID            int64   `json:"id"`
	Address       string  `json:"address"`
	IP            string  `json:"ip"`
	Port          int     `json:"port"`
	Version       string  `json:"version"`
	ClusterMode   string  `json:"cluster_mode"`
	Role          string  `json:"role"`
	HostID        *int64  `json:"host_id,omitempty"`
	ApplicationID *int64  `json:"application_id,omitempty"`
	Status        string  `json:"status"`
	LastSyncAt    *string `json:"last_sync_at,omitempty"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
}

type RedisInstanceListResponse struct {
	Code    int                    `json:"code"`
	Message string                 `json:"message"`
	Data    *RedisInstanceListData `json:"data,omitempty"`
}

type RedisInstanceListData struct {
	Items      []RedisInstanceResponse `json:"items"`
	Total      int64                   `json:"total"`
	Page       int                     `json:"page"`
	PageSize   int                     `json:"page_size"`
	TotalPages int                     `json:"total_pages"`
}

type RedisInstanceDetailResponse struct {
	Code    int                    `json:"code"`
	Message string                 `json:"message"`
	Data    *RedisInstanceResponse `json:"data,omitempty"`
}

type NginxInstanceResponse struct {
	ID            int64   `json:"id"`
	Address       string  `json:"address"`
	IP            string  `json:"ip"`
	Port          int     `json:"port"`
	Version       string  `json:"version"`
	HostID        *int64  `json:"host_id,omitempty"`
	ApplicationID *int64  `json:"application_id,omitempty"`
	Status        string  `json:"status"`
	LastSyncAt    *string `json:"last_sync_at,omitempty"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
}

type NginxInstanceListResponse struct {
	Code    int                    `json:"code"`
	Message string                 `json:"message"`
	Data    *NginxInstanceListData `json:"data,omitempty"`
}

type NginxInstanceListData struct {
	Items      []NginxInstanceResponse `json:"items"`
	Total      int64                   `json:"total"`
	Page       int                     `json:"page"`
	PageSize   int                     `json:"page_size"`
	TotalPages int                     `json:"total_pages"`
}

type NginxInstanceDetailResponse struct {
	Code    int                    `json:"code"`
	Message string                 `json:"message"`
	Data    *NginxInstanceResponse `json:"data,omitempty"`
}

type TomcatInstanceResponse struct {
	ID            int64   `json:"id"`
	Address       string  `json:"address"`
	IP            string  `json:"ip"`
	Port          int     `json:"port"`
	Version       string  `json:"version"`
	JVMVersion    string  `json:"jvm_version"`
	HostID        *int64  `json:"host_id,omitempty"`
	ApplicationID *int64  `json:"application_id,omitempty"`
	Status        string  `json:"status"`
	LastSyncAt    *string `json:"last_sync_at,omitempty"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
}

type TomcatInstanceListResponse struct {
	Code    int                     `json:"code"`
	Message string                  `json:"message"`
	Data    *TomcatInstanceListData `json:"data,omitempty"`
}

type TomcatInstanceListData struct {
	Items      []TomcatInstanceResponse `json:"items"`
	Total      int64                    `json:"total"`
	Page       int                      `json:"page"`
	PageSize   int                      `json:"page_size"`
	TotalPages int                      `json:"total_pages"`
}

type TomcatInstanceDetailResponse struct {
	Code    int                     `json:"code"`
	Message string                  `json:"message"`
	Data    *TomcatInstanceResponse `json:"data,omitempty"`
}

type ElasticsearchClusterResponse struct {
	ID            int64   `json:"id"`
	ClusterName   string  `json:"cluster_name"`
	Version       string  `json:"version"`
	NodeCount     int     `json:"node_count"`
	ApplicationID *int64  `json:"application_id,omitempty"`
	Status        string  `json:"status"`
	LastSyncAt    *string `json:"last_sync_at,omitempty"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
}

type ElasticsearchClusterListResponse struct {
	Code    int                           `json:"code"`
	Message string                        `json:"message"`
	Data    *ElasticsearchClusterListData `json:"data,omitempty"`
}

type ElasticsearchClusterListData struct {
	Items      []ElasticsearchClusterResponse `json:"items"`
	Total      int64                          `json:"total"`
	Page       int                            `json:"page"`
	PageSize   int                            `json:"page_size"`
	TotalPages int                            `json:"total_pages"`
}

type ElasticsearchClusterDetailResponse struct {
	Code    int                           `json:"code"`
	Message string                        `json:"message"`
	Data    *ElasticsearchClusterResponse `json:"data,omitempty"`
}

// ==================== Handler ====================

type AssetHandler struct {
	assetService    *asset.AssetService
	hostSyncService *sync.HostSyncService
}

func NewAssetHandler(assetService *asset.AssetService, hostSyncService *sync.HostSyncService) *AssetHandler {
	return &AssetHandler{
		assetService:    assetService,
		hostSyncService: hostSyncService,
	}
}

// ==================== Helper Functions ====================

func getPaginationParams(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return page, pageSize
}

func calculateTotalPages(total int64, pageSize int) int {
	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}
	return totalPages
}

func formatTimePtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format(time.RFC3339)
	return &s
}

// ==================== Project Endpoints ====================

func (h *AssetHandler) ListProjects(c *gin.Context) {
	page, pageSize := getPaginationParams(c)

	opts := repository.ListOptions{
		Page:     page,
		PageSize: pageSize,
	}

	if status := c.Query("status"); status != "" {
		opts.Filters = map[string]interface{}{
			"status": status,
		}
	}

	projects, total, err := h.assetService.ListProjects(c.Request.Context(), opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ProjectListResponse{
			Code:    50001,
			Message: "failed to list projects: " + err.Error(),
		})
		return
	}

	items := make([]ProjectResponse, 0, len(projects))
	for _, p := range projects {
		items = append(items, convertProjectToResponse(&p))
	}

	c.JSON(http.StatusOK, ProjectListResponse{
		Code:    0,
		Message: "success",
		Data: &ProjectListData{
			Items:      items,
			Total:      total,
			Page:       page,
			PageSize:   pageSize,
			TotalPages: calculateTotalPages(total, pageSize),
		},
	})
}

func (h *AssetHandler) CreateProject(c *gin.Context) {
	var req CreateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ProjectDetailResponse{
			Code:    40001,
			Message: "invalid request: " + err.Error(),
		})
		return
	}

	createReq := &asset.CreateProjectRequest{
		Name:        req.Name,
		Code:        req.Code,
		Description: req.Description,
		Owner:       req.Owner,
	}

	project, err := h.assetService.CreateProject(c.Request.Context(), createReq)
	if err != nil {
		code := 50001
		status := http.StatusInternalServerError
		msg := "failed to create project: " + err.Error()

		if errors.Is(err, asset.ErrProjectCodeExists) {
			code = 40901
			status = http.StatusConflict
			msg = "project code already exists"
		}

		c.JSON(status, ProjectDetailResponse{
			Code:    code,
			Message: msg,
		})
		return
	}

	resp := convertProjectToResponse(project)
	c.JSON(http.StatusCreated, ProjectDetailResponse{
		Code:    0,
		Message: "project created successfully",
		Data:    &resp,
	})
}

func (h *AssetHandler) GetProject(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ProjectDetailResponse{
			Code:    40001,
			Message: "invalid project id",
		})
		return
	}

	project, err := h.assetService.GetProject(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, ProjectDetailResponse{
			Code:    40401,
			Message: "project not found",
		})
		return
	}

	resp := convertProjectToResponse(project)
	c.JSON(http.StatusOK, ProjectDetailResponse{
		Code:    0,
		Message: "success",
		Data:    &resp,
	})
}

func (h *AssetHandler) UpdateProject(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ProjectDetailResponse{
			Code:    40001,
			Message: "invalid project id",
		})
		return
	}

	var req UpdateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ProjectDetailResponse{
			Code:    40001,
			Message: "invalid request: " + err.Error(),
		})
		return
	}

	updateReq := &asset.UpdateProjectRequest{
		Name:        req.Name,
		Description: req.Description,
		Owner:       req.Owner,
		Status:      req.Status,
	}

	project, err := h.assetService.UpdateProject(c.Request.Context(), id, updateReq)
	if err != nil {
		c.JSON(http.StatusNotFound, ProjectDetailResponse{
			Code:    40401,
			Message: "project not found",
		})
		return
	}

	resp := convertProjectToResponse(project)
	c.JSON(http.StatusOK, ProjectDetailResponse{
		Code:    0,
		Message: "project updated successfully",
		Data:    &resp,
	})
}

func (h *AssetHandler) DeleteProject(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    40001,
			"message": "invalid project id",
		})
		return
	}

	if err := h.assetService.DeleteProject(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    40401,
			"message": "project not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "project deleted successfully",
	})
}

func convertProjectToResponse(p *model.Project) ProjectResponse {
	return ProjectResponse{
		ID:          p.ID,
		Name:        p.Name,
		Code:        p.Code,
		Description: p.Description,
		Owner:       p.Owner,
		Status:      p.Status,
		CreatedAt:   p.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   p.UpdatedAt.Format(time.RFC3339),
	}
}

func (h *AssetHandler) ListApplications(c *gin.Context) {
	page, pageSize := getPaginationParams(c)

	opts := repository.ListOptions{
		Page:     page,
		PageSize: pageSize,
	}

	if status := c.Query("status"); status != "" {
		opts.Filters = map[string]interface{}{
			"status": status,
		}
	}

	if projectID := c.Query("project_id"); projectID != "" {
		if pid, err := strconv.ParseInt(projectID, 10, 64); err == nil {
			if opts.Filters == nil {
				opts.Filters = make(map[string]interface{})
			}
			opts.Filters["project_id"] = pid
		}
	}

	apps, total, err := h.assetService.ListApplications(c.Request.Context(), opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ApplicationListResponse{
			Code:    50001,
			Message: "failed to list applications: " + err.Error(),
		})
		return
	}

	items := make([]ApplicationResponse, 0, len(apps))
	for _, a := range apps {
		items = append(items, convertApplicationToResponse(&a))
	}

	c.JSON(http.StatusOK, ApplicationListResponse{
		Code:    0,
		Message: "success",
		Data: &ApplicationListData{
			Items:      items,
			Total:      total,
			Page:       page,
			PageSize:   pageSize,
			TotalPages: calculateTotalPages(total, pageSize),
		},
	})
}

func (h *AssetHandler) CreateApplication(c *gin.Context) {
	var req CreateApplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ApplicationDetailResponse{
			Code:    40001,
			Message: "invalid request: " + err.Error(),
		})
		return
	}

	createReq := &asset.CreateApplicationRequest{
		ProjectID:   req.ProjectID,
		Name:        req.Name,
		Code:        req.Code,
		Description: req.Description,
		Owner:       req.Owner,
	}

	app, err := h.assetService.CreateApplication(c.Request.Context(), createReq)
	if err != nil {
		code := 50001
		status := http.StatusInternalServerError
		msg := "failed to create application: " + err.Error()

		if errors.Is(err, asset.ErrApplicationCodeExists) {
			code = 40901
			status = http.StatusConflict
			msg = "application code already exists"
		} else if errors.Is(err, asset.ErrProjectNotFound) {
			code = 40402
			status = http.StatusNotFound
			msg = "project not found"
		}

		c.JSON(status, ApplicationDetailResponse{
			Code:    code,
			Message: msg,
		})
		return
	}

	resp := convertApplicationToResponse(app)
	c.JSON(http.StatusCreated, ApplicationDetailResponse{
		Code:    0,
		Message: "application created successfully",
		Data:    &resp,
	})
}

func (h *AssetHandler) GetApplication(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ApplicationDetailResponse{
			Code:    40001,
			Message: "invalid application id",
		})
		return
	}

	app, err := h.assetService.GetApplication(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, ApplicationDetailResponse{
			Code:    40401,
			Message: "application not found",
		})
		return
	}

	resp := convertApplicationToResponse(app)
	c.JSON(http.StatusOK, ApplicationDetailResponse{
		Code:    0,
		Message: "success",
		Data:    &resp,
	})
}

func (h *AssetHandler) UpdateApplication(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ApplicationDetailResponse{
			Code:    40001,
			Message: "invalid application id",
		})
		return
	}

	var req UpdateApplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ApplicationDetailResponse{
			Code:    40001,
			Message: "invalid request: " + err.Error(),
		})
		return
	}

	updateReq := &asset.UpdateApplicationRequest{
		Name:        req.Name,
		Description: req.Description,
		Owner:       req.Owner,
		Status:      req.Status,
	}

	app, err := h.assetService.UpdateApplication(c.Request.Context(), id, updateReq)
	if err != nil {
		c.JSON(http.StatusNotFound, ApplicationDetailResponse{
			Code:    40401,
			Message: "application not found",
		})
		return
	}

	resp := convertApplicationToResponse(app)
	c.JSON(http.StatusOK, ApplicationDetailResponse{
		Code:    0,
		Message: "application updated successfully",
		Data:    &resp,
	})
}

func (h *AssetHandler) DeleteApplication(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    40001,
			"message": "invalid application id",
		})
		return
	}

	if err := h.assetService.DeleteApplication(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    40401,
			"message": "application not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "application deleted successfully",
	})
}

func convertApplicationToResponse(a *model.Application) ApplicationResponse {
	return ApplicationResponse{
		ID:          a.ID,
		ProjectID:   a.ProjectID,
		Name:        a.Name,
		Code:        a.Code,
		Description: a.Description,
		Owner:       a.Owner,
		Status:      a.Status,
		CreatedAt:   a.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   a.UpdatedAt.Format(time.RFC3339),
	}
}

func (h *AssetHandler) ListHosts(c *gin.Context) {
	page, pageSize := getPaginationParams(c)

	opts := repository.ListOptions{
		Page:     page,
		PageSize: pageSize,
	}

	if status := c.Query("status"); status != "" {
		opts.Filters = map[string]interface{}{
			"status": status,
		}
	}

	if bg := c.Query("business_group"); bg != "" {
		if opts.Filters == nil {
			opts.Filters = make(map[string]interface{})
		}
		opts.Filters["business_group"] = bg
	}

	hosts, total, err := h.assetService.ListHosts(c.Request.Context(), opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, HostListResponse{
			Code:    50001,
			Message: "failed to list hosts: " + err.Error(),
		})
		return
	}

	items := make([]HostResponse, 0, len(hosts))
	for _, h := range hosts {
		items = append(items, convertHostToResponse(&h))
	}

	c.JSON(http.StatusOK, HostListResponse{
		Code:    0,
		Message: "success",
		Data: &HostListData{
			Items:      items,
			Total:      total,
			Page:       page,
			PageSize:   pageSize,
			TotalPages: calculateTotalPages(total, pageSize),
		},
	})
}

func (h *AssetHandler) CreateHost(c *gin.Context) {
	var req CreateHostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, HostDetailResponse{
			Code:    40001,
			Message: "invalid request: " + err.Error(),
		})
		return
	}

	createReq := &asset.CreateHostRequest{
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
		Tags:          req.Tags,
	}

	host, err := h.assetService.CreateHost(c.Request.Context(), createReq)
	if err != nil {
		code := 50001
		status := http.StatusInternalServerError
		msg := "failed to create host: " + err.Error()

		if errors.Is(err, asset.ErrHostIdentExists) {
			code = 40901
			status = http.StatusConflict
			msg = "host ident already exists"
		}

		c.JSON(status, HostDetailResponse{
			Code:    code,
			Message: msg,
		})
		return
	}

	resp := convertHostToResponse(host)
	c.JSON(http.StatusCreated, HostDetailResponse{
		Code:    0,
		Message: "host created successfully",
		Data:    &resp,
	})
}

func (h *AssetHandler) GetHost(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, HostDetailResponse{
			Code:    40001,
			Message: "invalid host id",
		})
		return
	}

	host, err := h.assetService.GetHost(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, HostDetailResponse{
			Code:    40401,
			Message: "host not found",
		})
		return
	}

	resp := convertHostToResponse(host)
	c.JSON(http.StatusOK, HostDetailResponse{
		Code:    0,
		Message: "success",
		Data:    &resp,
	})
}

func (h *AssetHandler) UpdateHost(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, HostDetailResponse{
			Code:    40001,
			Message: "invalid host id",
		})
		return
	}

	var req UpdateHostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, HostDetailResponse{
			Code:    40001,
			Message: "invalid request: " + err.Error(),
		})
		return
	}

	updateReq := &asset.UpdateHostRequest{
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
		Tags:          req.Tags,
		Status:        req.Status,
	}

	host, err := h.assetService.UpdateHost(c.Request.Context(), id, updateReq)
	if err != nil {
		c.JSON(http.StatusNotFound, HostDetailResponse{
			Code:    40401,
			Message: "host not found",
		})
		return
	}

	resp := convertHostToResponse(host)
	c.JSON(http.StatusOK, HostDetailResponse{
		Code:    0,
		Message: "host updated successfully",
		Data:    &resp,
	})
}

func (h *AssetHandler) DeleteHost(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    40001,
			"message": "invalid host id",
		})
		return
	}

	if err := h.assetService.DeleteHost(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    40401,
			"message": "host not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "host deleted successfully",
	})
}

func (h *AssetHandler) SyncHosts(c *gin.Context) {
	if h.hostSyncService == nil {
		c.JSON(http.StatusServiceUnavailable, SyncHostsResponse{
			Code:    50301,
			Message: "host sync service not available",
		})
		return
	}

	result, err := h.hostSyncService.SyncHosts(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, SyncHostsResponse{
			Code:    50001,
			Message: "failed to sync hosts: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, SyncHostsResponse{
		Code:    0,
		Message: "hosts synced successfully",
		Data: &SyncHostsData{
			TotalHosts:   result.TotalHosts,
			NewHosts:     result.NewHosts,
			UpdatedHosts: result.UpdatedHosts,
			FailedHosts:  result.FailedHosts,
			Duration:     result.Duration.String(),
		},
	})
}

func convertHostToResponse(h *model.Host) HostResponse {
	resp := HostResponse{
		ID:            h.ID,
		Ident:         h.Ident,
		Hostname:      h.Hostname,
		IP:            h.IP,
		OS:            h.OS,
		OSVersion:     h.OSVersion,
		KernelVersion: h.KernelVersion,
		CPUCores:      h.CPUCores,
		CPUModel:      h.CPUModel,
		MemoryTotal:   h.MemoryTotal,
		BusinessGroup: h.BusinessGroup,
		Env:           h.Env,
		Status:        h.Status,
		LastSyncAt:    formatTimePtr(h.LastSyncAt),
		CreatedAt:     h.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     h.UpdatedAt.Format(time.RFC3339),
	}
	return resp
}

func (h *AssetHandler) ListMySQLInstances(c *gin.Context) {
	page, pageSize := getPaginationParams(c)

	opts := repository.ListOptions{
		Page:     page,
		PageSize: pageSize,
	}

	instances, total, err := h.assetService.ListMySQLInstances(c.Request.Context(), opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, MySQLInstanceListResponse{
			Code:    50001,
			Message: "failed to list mysql instances: " + err.Error(),
		})
		return
	}

	items := make([]MySQLInstanceResponse, 0, len(instances))
	for _, inst := range instances {
		items = append(items, convertMySQLInstanceToResponse(&inst))
	}

	c.JSON(http.StatusOK, MySQLInstanceListResponse{
		Code:    0,
		Message: "success",
		Data: &MySQLInstanceListData{
			Items:      items,
			Total:      total,
			Page:       page,
			PageSize:   pageSize,
			TotalPages: calculateTotalPages(total, pageSize),
		},
	})
}

func (h *AssetHandler) GetMySQLInstance(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, MySQLInstanceDetailResponse{
			Code:    40001,
			Message: "invalid mysql instance id",
		})
		return
	}

	inst, err := h.assetService.GetMySQLInstance(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, MySQLInstanceDetailResponse{
			Code:    40401,
			Message: "mysql instance not found",
		})
		return
	}

	resp := convertMySQLInstanceToResponse(inst)
	c.JSON(http.StatusOK, MySQLInstanceDetailResponse{
		Code:    0,
		Message: "success",
		Data:    &resp,
	})
}

func (h *AssetHandler) DeleteMySQLInstance(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    40001,
			"message": "invalid mysql instance id",
		})
		return
	}

	if err := h.assetService.DeleteMySQLInstance(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    40401,
			"message": "mysql instance not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "mysql instance deleted successfully",
	})
}

func convertMySQLInstanceToResponse(m *model.MySQLInstance) MySQLInstanceResponse {
	return MySQLInstanceResponse{
		ID:            m.ID,
		Address:       m.Address,
		IP:            m.IP,
		Port:          m.Port,
		Version:       m.Version,
		ClusterMode:   m.ClusterMode,
		ServerID:      m.ServerID,
		HostID:        m.HostID,
		ApplicationID: m.ApplicationID,
		Status:        m.Status,
		LastSyncAt:    formatTimePtr(m.LastSyncAt),
		CreatedAt:     m.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     m.UpdatedAt.Format(time.RFC3339),
	}
}

func (h *AssetHandler) ListRedisInstances(c *gin.Context) {
	page, pageSize := getPaginationParams(c)

	opts := repository.ListOptions{
		Page:     page,
		PageSize: pageSize,
	}

	instances, total, err := h.assetService.ListRedisInstances(c.Request.Context(), opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, RedisInstanceListResponse{
			Code:    50001,
			Message: "failed to list redis instances: " + err.Error(),
		})
		return
	}

	items := make([]RedisInstanceResponse, 0, len(instances))
	for _, inst := range instances {
		items = append(items, convertRedisInstanceToResponse(&inst))
	}

	c.JSON(http.StatusOK, RedisInstanceListResponse{
		Code:    0,
		Message: "success",
		Data: &RedisInstanceListData{
			Items:      items,
			Total:      total,
			Page:       page,
			PageSize:   pageSize,
			TotalPages: calculateTotalPages(total, pageSize),
		},
	})
}

func (h *AssetHandler) GetRedisInstance(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, RedisInstanceDetailResponse{
			Code:    40001,
			Message: "invalid redis instance id",
		})
		return
	}

	inst, err := h.assetService.GetRedisInstance(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, RedisInstanceDetailResponse{
			Code:    40401,
			Message: "redis instance not found",
		})
		return
	}

	resp := convertRedisInstanceToResponse(inst)
	c.JSON(http.StatusOK, RedisInstanceDetailResponse{
		Code:    0,
		Message: "success",
		Data:    &resp,
	})
}

func (h *AssetHandler) DeleteRedisInstance(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    40001,
			"message": "invalid redis instance id",
		})
		return
	}

	if err := h.assetService.DeleteRedisInstance(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    40401,
			"message": "redis instance not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "redis instance deleted successfully",
	})
}

func convertRedisInstanceToResponse(r *model.RedisInstance) RedisInstanceResponse {
	return RedisInstanceResponse{
		ID:            r.ID,
		Address:       r.Address,
		IP:            r.IP,
		Port:          r.Port,
		Version:       r.Version,
		ClusterMode:   r.ClusterMode,
		Role:          r.Role,
		HostID:        r.HostID,
		ApplicationID: r.ApplicationID,
		Status:        r.Status,
		LastSyncAt:    formatTimePtr(r.LastSyncAt),
		CreatedAt:     r.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     r.UpdatedAt.Format(time.RFC3339),
	}
}

func (h *AssetHandler) ListNginxInstances(c *gin.Context) {
	page, pageSize := getPaginationParams(c)

	opts := repository.ListOptions{
		Page:     page,
		PageSize: pageSize,
	}

	instances, total, err := h.assetService.ListNginxInstances(c.Request.Context(), opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, NginxInstanceListResponse{
			Code:    50001,
			Message: "failed to list nginx instances: " + err.Error(),
		})
		return
	}

	items := make([]NginxInstanceResponse, 0, len(instances))
	for _, inst := range instances {
		items = append(items, convertNginxInstanceToResponse(&inst))
	}

	c.JSON(http.StatusOK, NginxInstanceListResponse{
		Code:    0,
		Message: "success",
		Data: &NginxInstanceListData{
			Items:      items,
			Total:      total,
			Page:       page,
			PageSize:   pageSize,
			TotalPages: calculateTotalPages(total, pageSize),
		},
	})
}

func (h *AssetHandler) GetNginxInstance(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, NginxInstanceDetailResponse{
			Code:    40001,
			Message: "invalid nginx instance id",
		})
		return
	}

	inst, err := h.assetService.GetNginxInstance(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, NginxInstanceDetailResponse{
			Code:    40401,
			Message: "nginx instance not found",
		})
		return
	}

	resp := convertNginxInstanceToResponse(inst)
	c.JSON(http.StatusOK, NginxInstanceDetailResponse{
		Code:    0,
		Message: "success",
		Data:    &resp,
	})
}

func (h *AssetHandler) DeleteNginxInstance(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    40001,
			"message": "invalid nginx instance id",
		})
		return
	}

	if err := h.assetService.DeleteNginxInstance(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    40401,
			"message": "nginx instance not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "nginx instance deleted successfully",
	})
}

func convertNginxInstanceToResponse(n *model.NginxInstance) NginxInstanceResponse {
	return NginxInstanceResponse{
		ID:            n.ID,
		Address:       n.Address,
		IP:            n.IP,
		Port:          n.Port,
		Version:       n.Version,
		HostID:        n.HostID,
		ApplicationID: n.ApplicationID,
		Status:        n.Status,
		LastSyncAt:    formatTimePtr(n.LastSyncAt),
		CreatedAt:     n.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     n.UpdatedAt.Format(time.RFC3339),
	}
}

func (h *AssetHandler) ListTomcatInstances(c *gin.Context) {
	page, pageSize := getPaginationParams(c)

	opts := repository.ListOptions{
		Page:     page,
		PageSize: pageSize,
	}

	instances, total, err := h.assetService.ListTomcatInstances(c.Request.Context(), opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, TomcatInstanceListResponse{
			Code:    50001,
			Message: "failed to list tomcat instances: " + err.Error(),
		})
		return
	}

	items := make([]TomcatInstanceResponse, 0, len(instances))
	for _, inst := range instances {
		items = append(items, convertTomcatInstanceToResponse(&inst))
	}

	c.JSON(http.StatusOK, TomcatInstanceListResponse{
		Code:    0,
		Message: "success",
		Data: &TomcatInstanceListData{
			Items:      items,
			Total:      total,
			Page:       page,
			PageSize:   pageSize,
			TotalPages: calculateTotalPages(total, pageSize),
		},
	})
}

func (h *AssetHandler) GetTomcatInstance(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, TomcatInstanceDetailResponse{
			Code:    40001,
			Message: "invalid tomcat instance id",
		})
		return
	}

	inst, err := h.assetService.GetTomcatInstance(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, TomcatInstanceDetailResponse{
			Code:    40401,
			Message: "tomcat instance not found",
		})
		return
	}

	resp := convertTomcatInstanceToResponse(inst)
	c.JSON(http.StatusOK, TomcatInstanceDetailResponse{
		Code:    0,
		Message: "success",
		Data:    &resp,
	})
}

func (h *AssetHandler) DeleteTomcatInstance(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    40001,
			"message": "invalid tomcat instance id",
		})
		return
	}

	if err := h.assetService.DeleteTomcatInstance(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    40401,
			"message": "tomcat instance not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "tomcat instance deleted successfully",
	})
}

func convertTomcatInstanceToResponse(t *model.TomcatInstance) TomcatInstanceResponse {
	return TomcatInstanceResponse{
		ID:            t.ID,
		Address:       t.Address,
		IP:            t.IP,
		Port:          t.Port,
		Version:       t.Version,
		JVMVersion:    t.JVMVersion,
		HostID:        t.HostID,
		ApplicationID: t.ApplicationID,
		Status:        t.Status,
		LastSyncAt:    formatTimePtr(t.LastSyncAt),
		CreatedAt:     t.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     t.UpdatedAt.Format(time.RFC3339),
	}
}

func (h *AssetHandler) ListElasticsearchClusters(c *gin.Context) {
	page, pageSize := getPaginationParams(c)

	opts := repository.ListOptions{
		Page:     page,
		PageSize: pageSize,
	}

	clusters, total, err := h.assetService.ListElasticsearchClusters(c.Request.Context(), opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ElasticsearchClusterListResponse{
			Code:    50001,
			Message: "failed to list elasticsearch clusters: " + err.Error(),
		})
		return
	}

	items := make([]ElasticsearchClusterResponse, 0, len(clusters))
	for _, cluster := range clusters {
		items = append(items, convertElasticsearchClusterToResponse(&cluster))
	}

	c.JSON(http.StatusOK, ElasticsearchClusterListResponse{
		Code:    0,
		Message: "success",
		Data: &ElasticsearchClusterListData{
			Items:      items,
			Total:      total,
			Page:       page,
			PageSize:   pageSize,
			TotalPages: calculateTotalPages(total, pageSize),
		},
	})
}

func (h *AssetHandler) GetElasticsearchCluster(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ElasticsearchClusterDetailResponse{
			Code:    40001,
			Message: "invalid elasticsearch cluster id",
		})
		return
	}

	cluster, err := h.assetService.GetElasticsearchCluster(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, ElasticsearchClusterDetailResponse{
			Code:    40401,
			Message: "elasticsearch cluster not found",
		})
		return
	}

	resp := convertElasticsearchClusterToResponse(cluster)
	c.JSON(http.StatusOK, ElasticsearchClusterDetailResponse{
		Code:    0,
		Message: "success",
		Data:    &resp,
	})
}

func (h *AssetHandler) DeleteElasticsearchCluster(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    40001,
			"message": "invalid elasticsearch cluster id",
		})
		return
	}

	if err := h.assetService.DeleteElasticsearchCluster(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    40401,
			"message": "elasticsearch cluster not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "elasticsearch cluster deleted successfully",
	})
}

func convertElasticsearchClusterToResponse(e *model.ElasticsearchCluster) ElasticsearchClusterResponse {
	return ElasticsearchClusterResponse{
		ID:            e.ID,
		ClusterName:   e.ClusterName,
		Version:       e.Version,
		NodeCount:     e.NodeCount,
		ApplicationID: e.ApplicationID,
		Status:        e.Status,
		LastSyncAt:    formatTimePtr(e.LastSyncAt),
		CreatedAt:     e.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     e.UpdatedAt.Format(time.RFC3339),
	}
}
