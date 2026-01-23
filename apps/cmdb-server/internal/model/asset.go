package model

import (
	"time"

	"gorm.io/datatypes"
)

// Project represents a business project
type Project struct {
	ID           int64  `gorm:"primaryKey"`
	Name         string `gorm:"size:100;not null"`
	Code         string `gorm:"size:50;uniqueIndex;not null"`
	Description  string `gorm:"type:text"`
	Owner        string `gorm:"size:100"`
	Status       string `gorm:"size:20;default:'active'"`
	HostCount    int    `gorm:"default:0"`
	N9EGroupID   int64  `gorm:"index"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	Applications []Application `gorm:"foreignKey:ProjectID"`
}

// TableName specifies the table name for Project
func (Project) TableName() string {
	return "projects"
}

// Application represents a business application
type Application struct {
	ID          int64  `gorm:"primaryKey"`
	ProjectID   int64  `gorm:"index"`
	Name        string `gorm:"size:100;not null"`
	Code        string `gorm:"size:50;uniqueIndex;not null"`
	Description string `gorm:"type:text"`
	Owner       string `gorm:"size:100"`
	Status      string `gorm:"size:20;default:'active'"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Project     *Project `gorm:"foreignKey:ProjectID"`
}

// TableName specifies the table name for Application
func (Application) TableName() string {
	return "applications"
}

// Host represents a server/host asset
type Host struct {
	ID            int64  `gorm:"primaryKey"`
	Ident         string `gorm:"size:100;uniqueIndex;not null"`
	Hostname      string `gorm:"size:255"`
	IP            string `gorm:"size:45;index"`
	OS            string `gorm:"size:50"`
	OSVersion     string `gorm:"size:100"`
	KernelVersion string `gorm:"size:100"`
	CPUCores      int
	CPUModel      string `gorm:"size:200"`
	MemoryTotal   int64
	Status        string         `gorm:"size:20;default:'active'"`
	BusinessGroup string         `gorm:"size:100;index"`
	Env           string         `gorm:"size:50"`
	Tags          datatypes.JSON `gorm:"type:jsonb"`
	LastSyncAt    *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// TableName specifies the table name for Host
func (Host) TableName() string {
	return "hosts"
}

// MySQLInstance represents a MySQL database instance
type MySQLInstance struct {
	ID            int64  `gorm:"primaryKey"`
	Address       string `gorm:"size:100;uniqueIndex;not null"`
	IP            string `gorm:"size:45"`
	Port          int
	Version       string `gorm:"size:50"`
	ClusterMode   string `gorm:"size:20"`
	ServerID      string `gorm:"size:50"`
	HostID        *int64 `gorm:"index"`
	ApplicationID *int64 `gorm:"index"`
	Status        string `gorm:"size:20;default:'active'"`
	LastSyncAt    *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
	Host          *Host        `gorm:"foreignKey:HostID"`
	Application   *Application `gorm:"foreignKey:ApplicationID"`
}

// TableName specifies the table name for MySQLInstance
func (MySQLInstance) TableName() string {
	return "mysql_instances"
}

// RedisInstance represents a Redis instance
type RedisInstance struct {
	ID            int64  `gorm:"primaryKey"`
	Address       string `gorm:"size:100;uniqueIndex;not null"`
	IP            string `gorm:"size:45"`
	Port          int
	Version       string `gorm:"size:50"`
	ClusterMode   string `gorm:"size:20"`
	Role          string `gorm:"size:20"`
	HostID        *int64 `gorm:"index"`
	ApplicationID *int64 `gorm:"index"`
	Status        string `gorm:"size:20;default:'active'"`
	LastSyncAt    *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
	Host          *Host        `gorm:"foreignKey:HostID"`
	Application   *Application `gorm:"foreignKey:ApplicationID"`
}

// TableName specifies the table name for RedisInstance
func (RedisInstance) TableName() string {
	return "redis_instances"
}

// NginxInstance represents an Nginx instance
type NginxInstance struct {
	ID            int64  `gorm:"primaryKey"`
	Address       string `gorm:"size:100;uniqueIndex;not null"`
	IP            string `gorm:"size:45"`
	Port          int
	Version       string `gorm:"size:50"`
	HostID        *int64 `gorm:"index"`
	ApplicationID *int64 `gorm:"index"`
	Status        string `gorm:"size:20;default:'active'"`
	LastSyncAt    *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
	Host          *Host        `gorm:"foreignKey:HostID"`
	Application   *Application `gorm:"foreignKey:ApplicationID"`
}

// TableName specifies the table name for NginxInstance
func (NginxInstance) TableName() string {
	return "nginx_instances"
}

// TomcatInstance represents a Tomcat instance
type TomcatInstance struct {
	ID            int64  `gorm:"primaryKey"`
	Address       string `gorm:"size:100;uniqueIndex;not null"`
	IP            string `gorm:"size:45"`
	Port          int
	Version       string `gorm:"size:50"`
	JVMVersion    string `gorm:"size:50"`
	HostID        *int64 `gorm:"index"`
	ApplicationID *int64 `gorm:"index"`
	Status        string `gorm:"size:20;default:'active'"`
	LastSyncAt    *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
	Host          *Host        `gorm:"foreignKey:HostID"`
	Application   *Application `gorm:"foreignKey:ApplicationID"`
}

// TableName specifies the table name for TomcatInstance
func (TomcatInstance) TableName() string {
	return "tomcat_instances"
}

// ElasticsearchCluster represents an Elasticsearch cluster
type ElasticsearchCluster struct {
	ID            int64  `gorm:"primaryKey"`
	ClusterName   string `gorm:"size:100;uniqueIndex;not null"`
	Version       string `gorm:"size:50"`
	NodeCount     int
	Status        string `gorm:"size:20;default:'active'"`
	ApplicationID *int64 `gorm:"index"`
	LastSyncAt    *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
	Application   *Application `gorm:"foreignKey:ApplicationID"`
}

// TableName specifies the table name for ElasticsearchCluster
func (ElasticsearchCluster) TableName() string {
	return "elasticsearch_clusters"
}

// ApplicationHost represents the many-to-many relationship between applications and hosts
type ApplicationHost struct {
	ApplicationID int64 `gorm:"primaryKey"`
	HostID        int64 `gorm:"primaryKey"`
	CreatedAt     time.Time
	Application   *Application `gorm:"foreignKey:ApplicationID"`
	Host          *Host        `gorm:"foreignKey:HostID"`
}

// TableName specifies the table name for ApplicationHost
func (ApplicationHost) TableName() string {
	return "application_hosts"
}
