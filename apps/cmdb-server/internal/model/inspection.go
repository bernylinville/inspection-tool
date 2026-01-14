package model

import (
	"time"

	"gorm.io/datatypes"
)

// InspectionJob represents an inspection task record
type InspectionJob struct {
	ID              int64  `gorm:"primaryKey"`
	Type            string `gorm:"size:50;not null"`
	TriggerType     string `gorm:"size:20;not null"`
	Status          string `gorm:"size:20;index;default:'pending'"`
	StartTime       *time.Time
	EndTime         *time.Time
	DurationSeconds int
	ReportExcelPath string         `gorm:"size:500"`
	ReportHTMLPath  string         `gorm:"size:500"`
	Summary         datatypes.JSON `gorm:"type:jsonb"`
	ErrorMessage    string         `gorm:"type:text"`
	CreatedBy       string         `gorm:"size:50"`
	CreatedAt       time.Time      `gorm:"index"`
}

// TableName specifies the table name for InspectionJob
func (InspectionJob) TableName() string {
	return "inspection_jobs"
}

// InspectionType constants
const (
	InspectionTypeHost          = "host"
	InspectionTypeMySQL         = "mysql"
	InspectionTypeRedis         = "redis"
	InspectionTypeNginx         = "nginx"
	InspectionTypeTomcat        = "tomcat"
	InspectionTypeElasticsearch = "elasticsearch"
)

// TriggerType constants
const (
	TriggerTypeManual    = "manual"
	TriggerTypeScheduled = "scheduled"
)

// JobStatus constants
const (
	JobStatusPending = "pending"
	JobStatusRunning = "running"
	JobStatusSuccess = "success"
	JobStatusFailed  = "failed"
)
