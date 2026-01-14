package user

import (
	"context"
	"errors"

	"github.com/rs/zerolog"

	"inspection-tool/apps/cmdb-server/internal/model"
	"inspection-tool/apps/cmdb-server/internal/repository"
	"inspection-tool/apps/cmdb-server/internal/service/auth"
)

var (
	ErrUserExists          = errors.New("user already exists")
	ErrOldPasswordIncorrect = errors.New("old password is incorrect")
)

type CreateUserRequest struct {
	Username    string
	Password    string
	Email       string
	DisplayName string
	RoleIDs     []int64
}

type UpdateUserRequest struct {
	Email       *string
	DisplayName *string
	Status      *string
}

type UserService struct {
	userRepo    repository.UserRepository
	roleRepo    repository.RoleRepository
	authService *auth.AuthService
	logger      zerolog.Logger
}

func NewUserService(userRepo repository.UserRepository, roleRepo repository.RoleRepository, authService *auth.AuthService, logger zerolog.Logger) *UserService {
	return &UserService{
		userRepo:    userRepo,
		roleRepo:    roleRepo,
		authService: authService,
		logger:      logger,
	}
}

func (s *UserService) CreateUser(ctx context.Context, req *CreateUserRequest) (*model.User, error) {
	if req == nil {
		return nil, errors.New("create user request is nil")
	}
	if existing, err := s.userRepo.FindByUsername(ctx, req.Username); err == nil && existing != nil {
		return nil, ErrUserExists
	}

	hashedPassword, err := s.authService.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	user := &model.User{
		Username:     req.Username,
		PasswordHash: hashedPassword,
		Email:        req.Email,
		DisplayName:  req.DisplayName,
		Status:       "active",
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	if len(req.RoleIDs) > 0 {
		if err := s.AssignRoles(ctx, user.ID, req.RoleIDs); err != nil {
			return nil, err
		}
		user, err = s.userRepo.FindByID(ctx, user.ID)
		if err != nil {
			return nil, err
		}
	}

	return user, nil
}

func (s *UserService) UpdateUser(ctx context.Context, id int64, req *UpdateUserRequest) (*model.User, error) {
	if req == nil {
		return nil, errors.New("update user request is nil")
	}

	user, err := s.userRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Email != nil {
		user.Email = *req.Email
	}
	if req.DisplayName != nil {
		user.DisplayName = *req.DisplayName
	}
	if req.Status != nil {
		user.Status = *req.Status
	}

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *UserService) DeleteUser(ctx context.Context, id int64) error {
	return s.userRepo.Delete(ctx, id)
}

func (s *UserService) GetUser(ctx context.Context, id int64) (*model.User, error) {
	return s.userRepo.FindByID(ctx, id)
}

func (s *UserService) ListUsers(ctx context.Context, opts repository.ListOptions) ([]model.User, int64, error) {
	return s.userRepo.List(ctx, opts)
}

func (s *UserService) AssignRoles(ctx context.Context, userID int64, roleIDs []int64) error {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return err
	}

	roles := make([]model.Role, 0, len(roleIDs))
	for _, roleID := range roleIDs {
		role, err := s.roleRepo.FindByID(ctx, roleID)
		if err != nil {
			return err
		}
		roles = append(roles, *role)
	}
	user.Roles = roles

	return s.userRepo.Update(ctx, user)
}

func (s *UserService) ChangePassword(ctx context.Context, userID int64, oldPwd, newPwd string) error {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return err
	}
	if err := s.authService.VerifyPassword(user.PasswordHash, oldPwd); err != nil {
		return ErrOldPasswordIncorrect
	}
	newHash, err := s.authService.HashPassword(newPwd)
	if err != nil {
		return err
	}
	user.PasswordHash = newHash
	return s.userRepo.Update(ctx, user)
}
