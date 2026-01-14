package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"inspection-tool/apps/cmdb-server/internal/model"
	"inspection-tool/apps/cmdb-server/internal/repository"
	"inspection-tool/apps/cmdb-server/internal/service/role"
)

type RoleHandler struct {
	roleService *role.RoleService
}

func NewRoleHandler(roleService *role.RoleService) *RoleHandler {
	return &RoleHandler{
		roleService: roleService,
	}
}

type CreateRoleRequest struct {
	Name          string  `json:"name" binding:"required,min=2,max=50"`
	Description   string  `json:"description" binding:"omitempty,max=200"`
	PermissionIDs []int64 `json:"permission_ids" binding:"omitempty"`
}

type UpdateRoleRequest struct {
	Name        *string `json:"name" binding:"omitempty,min=2,max=50"`
	Description *string `json:"description" binding:"omitempty,max=200"`
}

type AssignPermissionsRequest struct {
	PermissionIDs []int64 `json:"permission_ids" binding:"required"`
}

type RoleResponse struct {
	ID          int64                `json:"id"`
	Name        string               `json:"name"`
	Description string               `json:"description"`
	Permissions []PermissionResponse `json:"permissions,omitempty"`
	CreatedAt   string               `json:"created_at"`
	UpdatedAt   string               `json:"updated_at"`
}

type PermissionResponse struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Resource string `json:"resource"`
	Action   string `json:"action"`
}

type RoleListResponse struct {
	Code    int           `json:"code"`
	Message string        `json:"message"`
	Data    *RoleListData `json:"data,omitempty"`
}

type RoleListData struct {
	Items      []RoleResponse `json:"items"`
	Total      int64          `json:"total"`
	Page       int            `json:"page"`
	PageSize   int            `json:"page_size"`
	TotalPages int            `json:"total_pages"`
}

type RoleDetailResponse struct {
	Code    int           `json:"code"`
	Message string        `json:"message"`
	Data    *RoleResponse `json:"data,omitempty"`
}

func (h *RoleHandler) ListRoles(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	opts := repository.ListOptions{
		Page:     page,
		PageSize: pageSize,
	}

	roles, total, err := h.roleService.ListRoles(c.Request.Context(), opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, RoleListResponse{
			Code:    50001,
			Message: "failed to list roles: " + err.Error(),
		})
		return
	}

	items := make([]RoleResponse, 0, len(roles))
	for _, r := range roles {
		items = append(items, convertRoleToResponse(&r))
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	c.JSON(http.StatusOK, RoleListResponse{
		Code:    0,
		Message: "success",
		Data: &RoleListData{
			Items:      items,
			Total:      total,
			Page:       page,
			PageSize:   pageSize,
			TotalPages: totalPages,
		},
	})
}

func (h *RoleHandler) CreateRole(c *gin.Context) {
	var req CreateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, RoleDetailResponse{
			Code:    40001,
			Message: "invalid request: " + err.Error(),
		})
		return
	}

	createReq := &role.CreateRoleRequest{
		Name:          req.Name,
		Description:   req.Description,
		PermissionIDs: req.PermissionIDs,
	}

	newRole, err := h.roleService.CreateRole(c.Request.Context(), createReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, RoleDetailResponse{
			Code:    50001,
			Message: "failed to create role: " + err.Error(),
		})
		return
	}

	resp := convertRoleToResponse(newRole)
	c.JSON(http.StatusCreated, RoleDetailResponse{
		Code:    0,
		Message: "role created successfully",
		Data:    &resp,
	})
}

func (h *RoleHandler) GetRole(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, RoleDetailResponse{
			Code:    40001,
			Message: "invalid role id",
		})
		return
	}

	r, err := h.roleService.GetRole(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, RoleDetailResponse{
			Code:    40401,
			Message: "role not found",
		})
		return
	}

	resp := convertRoleToResponse(r)
	c.JSON(http.StatusOK, RoleDetailResponse{
		Code:    0,
		Message: "success",
		Data:    &resp,
	})
}

func (h *RoleHandler) UpdateRole(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, RoleDetailResponse{
			Code:    40001,
			Message: "invalid role id",
		})
		return
	}

	var req UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, RoleDetailResponse{
			Code:    40001,
			Message: "invalid request: " + err.Error(),
		})
		return
	}

	updateReq := &role.UpdateRoleRequest{
		Name:        req.Name,
		Description: req.Description,
	}

	updatedRole, err := h.roleService.UpdateRole(c.Request.Context(), id, updateReq)
	if err != nil {
		c.JSON(http.StatusNotFound, RoleDetailResponse{
			Code:    40401,
			Message: "role not found",
		})
		return
	}

	resp := convertRoleToResponse(updatedRole)
	c.JSON(http.StatusOK, RoleDetailResponse{
		Code:    0,
		Message: "role updated successfully",
		Data:    &resp,
	})
}

func (h *RoleHandler) DeleteRole(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    40001,
			"message": "invalid role id",
		})
		return
	}

	if err := h.roleService.DeleteRole(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    40401,
			"message": "role not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "role deleted successfully",
	})
}

func (h *RoleHandler) AssignPermissions(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    40001,
			"message": "invalid role id",
		})
		return
	}

	var req AssignPermissionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    40001,
			"message": "invalid request: " + err.Error(),
		})
		return
	}

	if err := h.roleService.AssignPermissions(c.Request.Context(), id, req.PermissionIDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    50001,
			"message": "failed to assign permissions: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "permissions assigned successfully",
	})
}

func convertRoleToResponse(r *model.Role) RoleResponse {
	permissions := make([]PermissionResponse, 0, len(r.Permissions))
	for _, p := range r.Permissions {
		permissions = append(permissions, PermissionResponse{
			ID:       p.ID,
			Name:     p.Name,
			Resource: p.Resource,
			Action:   p.Action,
		})
	}

	return RoleResponse{
		ID:          r.ID,
		Name:        r.Name,
		Description: r.Description,
		Permissions: permissions,
		CreatedAt:   r.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   r.CreatedAt.Format(time.RFC3339),
	}
}
