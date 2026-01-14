package role

import (
	"context"
	"errors"

	"github.com/rs/zerolog"

	"inspection-tool/apps/cmdb-server/internal/model"
	"inspection-tool/apps/cmdb-server/internal/repository"
)

type CreateRoleRequest struct {
	Name          string
	Description   string
	PermissionIDs []int64
}

type UpdateRoleRequest struct {
	Name        *string
	Description *string
}

type RoleService struct {
	roleRepo repository.RoleRepository
	permRepo repository.PermissionRepository
	logger   zerolog.Logger
}

func NewRoleService(roleRepo repository.RoleRepository, permRepo repository.PermissionRepository, logger zerolog.Logger) *RoleService {
	return &RoleService{
		roleRepo: roleRepo,
		permRepo: permRepo,
		logger:   logger,
	}
}

func (s *RoleService) CreateRole(ctx context.Context, req *CreateRoleRequest) (*model.Role, error) {
	if req == nil {
		return nil, errors.New("create role request is nil")
	}

	role := &model.Role{
		Name:        req.Name,
		Description: req.Description,
	}

	if err := s.roleRepo.Create(ctx, role); err != nil {
		return nil, err
	}

	if len(req.PermissionIDs) > 0 {
		if err := s.AssignPermissions(ctx, role.ID, req.PermissionIDs); err != nil {
			return nil, err
		}
		role, err := s.roleRepo.FindByID(ctx, role.ID)
		if err != nil {
			return nil, err
		}
		return role, nil
	}

	return role, nil
}

func (s *RoleService) UpdateRole(ctx context.Context, id int64, req *UpdateRoleRequest) (*model.Role, error) {
	if req == nil {
		return nil, errors.New("update role request is nil")
	}

	role, err := s.roleRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		role.Name = *req.Name
	}
	if req.Description != nil {
		role.Description = *req.Description
	}

	if err := s.roleRepo.Update(ctx, role); err != nil {
		return nil, err
	}

	return role, nil
}

func (s *RoleService) DeleteRole(ctx context.Context, id int64) error {
	return s.roleRepo.Delete(ctx, id)
}

func (s *RoleService) GetRole(ctx context.Context, id int64) (*model.Role, error) {
	return s.roleRepo.FindByID(ctx, id)
}

func (s *RoleService) ListRoles(ctx context.Context, opts repository.ListOptions) ([]model.Role, int64, error) {
	return s.roleRepo.List(ctx, opts)
}

func (s *RoleService) AssignPermissions(ctx context.Context, roleID int64, permissionIDs []int64) error {
	role, err := s.roleRepo.FindByID(ctx, roleID)
	if err != nil {
		return err
	}

	permissions := make([]model.Permission, 0, len(permissionIDs))
	for _, permissionID := range permissionIDs {
		permission, err := s.permRepo.FindByID(ctx, permissionID)
		if err != nil {
			return err
		}
		permissions = append(permissions, *permission)
	}
	role.Permissions = permissions

	return s.roleRepo.Update(ctx, role)
}

func (s *RoleService) GetRolePermissions(ctx context.Context, roleID int64) ([]model.Permission, error) {
	role, err := s.roleRepo.FindByID(ctx, roleID)
	if err != nil {
		return nil, err
	}
	return role.Permissions, nil
}
