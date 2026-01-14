package model

import "time"

// User represents a system user
type User struct {
	ID           int64  `gorm:"primaryKey"`
	Username     string `gorm:"size:50;uniqueIndex;not null"`
	PasswordHash string `gorm:"size:255;not null"`
	Email        string `gorm:"size:100"`
	DisplayName  string `gorm:"size:100"`
	Status       string `gorm:"size:20;default:'active'"`
	LastLoginAt  *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
	Roles        []Role `gorm:"many2many:user_roles;"`
}

// TableName specifies the table name for User
func (User) TableName() string {
	return "users"
}

// Role represents a user role for RBAC
type Role struct {
	ID          int64  `gorm:"primaryKey"`
	Name        string `gorm:"size:50;uniqueIndex;not null"`
	Description string `gorm:"type:text"`
	CreatedAt   time.Time
	Users       []User       `gorm:"many2many:user_roles;"`
	Permissions []Permission `gorm:"many2many:role_permissions;"`
}

// TableName specifies the table name for Role
func (Role) TableName() string {
	return "roles"
}

// Permission represents a permission for RBAC
type Permission struct {
	ID          int64  `gorm:"primaryKey"`
	Name        string `gorm:"size:100;uniqueIndex;not null"`
	Resource    string `gorm:"size:50"`
	Action      string `gorm:"size:20"`
	Description string `gorm:"type:text"`
	CreatedAt   time.Time
	Roles       []Role `gorm:"many2many:role_permissions;"`
}

// TableName specifies the table name for Permission
func (Permission) TableName() string {
	return "permissions"
}
