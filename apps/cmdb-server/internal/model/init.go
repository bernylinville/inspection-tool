package model

import (
	"fmt"

	"github.com/rs/zerolog"
	"gorm.io/gorm"
)

// DefaultRoles defines the default roles for the system
var DefaultRoles = []Role{
	{Name: "admin", Description: "系统管理员，拥有所有权限"},
	{Name: "operator", Description: "运维人员，可管理资产和执行巡检"},
	{Name: "viewer", Description: "只读用户，仅可查看数据"},
}

// DefaultPermissions defines the default permissions for the system
var DefaultPermissions = []Permission{
	{Name: "hosts:read", Resource: "hosts", Action: "read", Description: "查看主机"},
	{Name: "hosts:write", Resource: "hosts", Action: "write", Description: "管理主机"},
	{Name: "hosts:sync", Resource: "hosts", Action: "sync", Description: "同步主机"},
	{Name: "projects:read", Resource: "projects", Action: "read", Description: "查看项目"},
	{Name: "projects:write", Resource: "projects", Action: "write", Description: "管理项目"},
	{Name: "applications:read", Resource: "applications", Action: "read", Description: "查看应用"},
	{Name: "applications:write", Resource: "applications", Action: "write", Description: "管理应用"},
	{Name: "middleware:read", Resource: "middleware", Action: "read", Description: "查看中间件"},
	{Name: "middleware:write", Resource: "middleware", Action: "write", Description: "管理中间件"},
	{Name: "inspection:read", Resource: "inspection", Action: "read", Description: "查看巡检"},
	{Name: "inspection:create", Resource: "inspection", Action: "create", Description: "创建巡检任务"},
	{Name: "inspection:delete", Resource: "inspection", Action: "delete", Description: "删除巡检任务"},
	{Name: "users:read", Resource: "users", Action: "read", Description: "查看用户"},
	{Name: "users:write", Resource: "users", Action: "write", Description: "管理用户"},
	{Name: "roles:read", Resource: "roles", Action: "read", Description: "查看角色"},
	{Name: "roles:write", Resource: "roles", Action: "write", Description: "管理角色"},
	{Name: "monitor:read", Resource: "monitor", Action: "read", Description: "查看监控数据"},
	{Name: "alerts:read", Resource: "alerts", Action: "read", Description: "查看告警"},
}

// RolePermissionMapping defines permissions for each role
var RolePermissionMapping = map[string][]string{
	"admin": {
		"hosts:read",
		"hosts:write",
		"hosts:sync",
		"projects:read",
		"projects:write",
		"applications:read",
		"applications:write",
		"middleware:read",
		"middleware:write",
		"inspection:read",
		"inspection:create",
		"inspection:delete",
		"users:read",
		"users:write",
		"roles:read",
		"roles:write",
		"monitor:read",
		"alerts:read",
	},
	"operator": {
		"hosts:read",
		"hosts:write",
		"hosts:sync",
		"projects:read",
		"applications:read",
		"middleware:read",
		"middleware:write",
		"inspection:read",
		"inspection:create",
		"inspection:delete",
		"monitor:read",
		"alerts:read",
	},
	"viewer": {
		"hosts:read",
		"projects:read",
		"applications:read",
		"middleware:read",
		"inspection:read",
		"users:read",
		"roles:read",
		"monitor:read",
		"alerts:read",
	},
}

// InitializeBaseData initializes default roles and permissions.
func InitializeBaseData(db *gorm.DB, log zerolog.Logger) error {
	log.Info().Msg("initializing base data")

	permissionsByName := make(map[string]Permission, len(DefaultPermissions))
	for _, permission := range DefaultPermissions {
		var existing Permission
		if err := db.Where("name = ?", permission.Name).First(&existing).Error; err != nil {
			if err != gorm.ErrRecordNotFound {
				return fmt.Errorf("query permission %s: %w", permission.Name, err)
			}
			if err := db.Create(&permission).Error; err != nil {
				return fmt.Errorf("create permission %s: %w", permission.Name, err)
			}
			existing = permission
			log.Info().Str("permission", permission.Name).Msg("created permission")
		}
		permissionsByName[permission.Name] = existing
	}

	rolesByName := make(map[string]Role, len(DefaultRoles))
	for _, role := range DefaultRoles {
		var existing Role
		if err := db.Where("name = ?", role.Name).First(&existing).Error; err != nil {
			if err != gorm.ErrRecordNotFound {
				return fmt.Errorf("query role %s: %w", role.Name, err)
			}
			if err := db.Create(&role).Error; err != nil {
				return fmt.Errorf("create role %s: %w", role.Name, err)
			}
			existing = role
			log.Info().Str("role", role.Name).Msg("created role")
		}
		rolesByName[role.Name] = existing
	}

	for roleName, permissionNames := range RolePermissionMapping {
		role, ok := rolesByName[roleName]
		if !ok {
			return fmt.Errorf("role %s not found", roleName)
		}

		permissions := make([]Permission, 0, len(permissionNames))
		for _, permissionName := range permissionNames {
			permission, ok := permissionsByName[permissionName]
			if !ok {
				return fmt.Errorf("permission %s not found for role %s", permissionName, roleName)
			}
			permissions = append(permissions, permission)
		}

		if err := db.Model(&role).Association("Permissions").Replace(permissions); err != nil {
			return fmt.Errorf("assign permissions for role %s: %w", roleName, err)
		}
		log.Info().Str("role", roleName).Int("permission_count", len(permissions)).Msg("assigned permissions")
	}

	log.Info().Msg("base data initialization completed")
	return nil
}
