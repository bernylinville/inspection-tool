package handler

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"inspection-tool/apps/cmdb-server/internal/model"
	"inspection-tool/apps/cmdb-server/internal/repository"
	"inspection-tool/apps/cmdb-server/internal/service/user"
)

type UserHandler struct {
	userService *user.UserService
}

func NewUserHandler(userService *user.UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
}

type CreateUserRequest struct {
	Username    string  `json:"username" binding:"required,min=3,max=50"`
	Password    string  `json:"password" binding:"required,min=6,max=100"`
	Email       string  `json:"email" binding:"omitempty,email"`
	DisplayName string  `json:"display_name" binding:"omitempty,max=100"`
	RoleIDs     []int64 `json:"role_ids" binding:"omitempty"`
}

type UpdateUserRequest struct {
	Email       *string `json:"email" binding:"omitempty,email"`
	DisplayName *string `json:"display_name" binding:"omitempty,max=100"`
	Status      *string `json:"status" binding:"omitempty,oneof=active inactive"`
}

type AssignRolesRequest struct {
	RoleIDs []int64 `json:"role_ids" binding:"required"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6,max=100"`
}

type UserResponse struct {
	ID          int64              `json:"id"`
	Username    string             `json:"username"`
	Email       string             `json:"email"`
	DisplayName string             `json:"display_name"`
	Status      string             `json:"status"`
	Roles       []UserRoleResponse `json:"roles,omitempty"`
	CreatedAt   string             `json:"created_at"`
	UpdatedAt   string             `json:"updated_at"`
}

type UserRoleResponse struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type UserListResponse struct {
	Code    int           `json:"code"`
	Message string        `json:"message"`
	Data    *UserListData `json:"data,omitempty"`
}

type UserListData struct {
	Items      []UserResponse `json:"items"`
	Total      int64          `json:"total"`
	Page       int            `json:"page"`
	PageSize   int            `json:"page_size"`
	TotalPages int            `json:"total_pages"`
}

type UserDetailResponse struct {
	Code    int           `json:"code"`
	Message string        `json:"message"`
	Data    *UserResponse `json:"data,omitempty"`
}

func (h *UserHandler) ListUsers(c *gin.Context) {
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

	if status := c.Query("status"); status != "" {
		opts.Filters = map[string]interface{}{
			"status": status,
		}
	}

	users, total, err := h.userService.ListUsers(c.Request.Context(), opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, UserListResponse{
			Code:    50001,
			Message: "failed to list users: " + err.Error(),
		})
		return
	}

	items := make([]UserResponse, 0, len(users))
	for _, u := range users {
		items = append(items, convertUserToResponse(&u))
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	c.JSON(http.StatusOK, UserListResponse{
		Code:    0,
		Message: "success",
		Data: &UserListData{
			Items:      items,
			Total:      total,
			Page:       page,
			PageSize:   pageSize,
			TotalPages: totalPages,
		},
	})
}

func (h *UserHandler) CreateUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, UserDetailResponse{
			Code:    40001,
			Message: "invalid request: " + err.Error(),
		})
		return
	}

	createReq := &user.CreateUserRequest{
		Username:    req.Username,
		Password:    req.Password,
		Email:       req.Email,
		DisplayName: req.DisplayName,
		RoleIDs:     req.RoleIDs,
	}

	newUser, err := h.userService.CreateUser(c.Request.Context(), createReq)
	if err != nil {
		code := 50001
		status := http.StatusInternalServerError
		msg := "failed to create user: " + err.Error()

		if errors.Is(err, user.ErrUserExists) {
			code = 40901
			status = http.StatusConflict
			msg = "user already exists"
		}

		c.JSON(status, UserDetailResponse{
			Code:    code,
			Message: msg,
		})
		return
	}

	resp := convertUserToResponse(newUser)
	c.JSON(http.StatusCreated, UserDetailResponse{
		Code:    0,
		Message: "user created successfully",
		Data:    &resp,
	})
}

func (h *UserHandler) GetUser(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, UserDetailResponse{
			Code:    40001,
			Message: "invalid user id",
		})
		return
	}

	u, err := h.userService.GetUser(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, UserDetailResponse{
			Code:    40401,
			Message: "user not found",
		})
		return
	}

	resp := convertUserToResponse(u)
	c.JSON(http.StatusOK, UserDetailResponse{
		Code:    0,
		Message: "success",
		Data:    &resp,
	})
}

func (h *UserHandler) UpdateUser(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, UserDetailResponse{
			Code:    40001,
			Message: "invalid user id",
		})
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, UserDetailResponse{
			Code:    40001,
			Message: "invalid request: " + err.Error(),
		})
		return
	}

	updateReq := &user.UpdateUserRequest{
		Email:       req.Email,
		DisplayName: req.DisplayName,
		Status:      req.Status,
	}

	updatedUser, err := h.userService.UpdateUser(c.Request.Context(), id, updateReq)
	if err != nil {
		c.JSON(http.StatusNotFound, UserDetailResponse{
			Code:    40401,
			Message: "user not found",
		})
		return
	}

	resp := convertUserToResponse(updatedUser)
	c.JSON(http.StatusOK, UserDetailResponse{
		Code:    0,
		Message: "user updated successfully",
		Data:    &resp,
	})
}

func (h *UserHandler) DeleteUser(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    40001,
			"message": "invalid user id",
		})
		return
	}

	if err := h.userService.DeleteUser(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    40401,
			"message": "user not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "user deleted successfully",
	})
}

func (h *UserHandler) AssignRoles(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    40001,
			"message": "invalid user id",
		})
		return
	}

	var req AssignRolesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    40001,
			"message": "invalid request: " + err.Error(),
		})
		return
	}

	if err := h.userService.AssignRoles(c.Request.Context(), id, req.RoleIDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    50001,
			"message": "failed to assign roles: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "roles assigned successfully",
	})
}

func (h *UserHandler) ChangePassword(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    40001,
			"message": "invalid user id",
		})
		return
	}

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    40001,
			"message": "invalid request: " + err.Error(),
		})
		return
	}

	if err := h.userService.ChangePassword(c.Request.Context(), id, req.OldPassword, req.NewPassword); err != nil {
		code := 50001
		status := http.StatusInternalServerError
		msg := "failed to change password"

		if errors.Is(err, user.ErrOldPasswordIncorrect) {
			code = 40002
			status = http.StatusBadRequest
			msg = "old password is incorrect"
		}

		c.JSON(status, gin.H{
			"code":    code,
			"message": msg,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "password changed successfully",
	})
}

func convertUserToResponse(u *model.User) UserResponse {
	roles := make([]UserRoleResponse, 0, len(u.Roles))
	for _, r := range u.Roles {
		roles = append(roles, UserRoleResponse{
			ID:   r.ID,
			Name: r.Name,
		})
	}

	return UserResponse{
		ID:          u.ID,
		Username:    u.Username,
		Email:       u.Email,
		DisplayName: u.DisplayName,
		Status:      u.Status,
		Roles:       roles,
		CreatedAt:   u.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   u.UpdatedAt.Format(time.RFC3339),
	}
}
