package database

import (
	"fmt"
	"time"

	"github.com/rs/zerolog"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"inspection-tool/apps/cmdb-server/internal/model"
)

// Config holds database configuration
type Config struct {
	Host         string
	Port         int
	User         string
	Password     string
	DBName       string
	SSLMode      string
	MaxIdleConns int
	MaxOpenConns int
	MaxLifetime  time.Duration
}

// DB is the global database instance
var DB *gorm.DB

// Initialize creates a new database connection and runs migrations
func Initialize(cfg *Config, log zerolog.Logger) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode,
	)

	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	}

	db, err := gorm.Open(postgres.Open(dsn), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// Configure connection pool
	if cfg.MaxIdleConns > 0 {
		sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	} else {
		sqlDB.SetMaxIdleConns(10)
	}

	if cfg.MaxOpenConns > 0 {
		sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	} else {
		sqlDB.SetMaxOpenConns(100)
	}

	if cfg.MaxLifetime > 0 {
		sqlDB.SetConnMaxLifetime(cfg.MaxLifetime)
	} else {
		sqlDB.SetConnMaxLifetime(time.Hour)
	}

	log.Info().Msg("Database connection established")

	DB = db
	return db, nil
}

// AutoMigrate runs database migrations for all models
func AutoMigrate(db *gorm.DB, log zerolog.Logger) error {
	log.Info().Msg("Starting database migration...")

	err := db.AutoMigrate(
		// Business organization
		&model.Project{},
		&model.Application{},

		// Asset management
		&model.Host{},
		&model.MySQLInstance{},
		&model.RedisInstance{},
		&model.NginxInstance{},
		&model.TomcatInstance{},
		&model.ElasticsearchCluster{},
		&model.ApplicationHost{},

		// User and permissions
		&model.User{},
		&model.Role{},
		&model.Permission{},

		// Inspection
		&model.InspectionJob{},
	)

	if err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	log.Info().Msg("Database migration completed successfully")
	return nil
}

// Seed initializes default data in the database
func Seed(db *gorm.DB, log zerolog.Logger) error {
	log.Info().Msg("Starting database seeding...")

	if err := model.InitializeBaseData(db, log); err != nil {
		return fmt.Errorf("failed to initialize base data: %w", err)
	}

	var adminRole model.Role
	if err := db.Where("name = ?", "admin").First(&adminRole).Error; err != nil {
		return fmt.Errorf("admin role not found: %w", err)
	}

	var existingUser model.User
	if err := db.Where("username = ?", "admin").First(&existingUser).Error; err == nil {
		log.Info().Msg("Admin user already exists, skipping creation")
		return nil
	} else if err != gorm.ErrRecordNotFound {
		return fmt.Errorf("failed to check existing admin user: %w", err)
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash admin password: %w", err)
	}

	adminUser := &model.User{
		Username:     "admin",
		PasswordHash: string(hashedPassword),
		DisplayName:  "System Administrator",
		Email:        "admin@cmdb.local",
		Status:       "active",
		Roles:        []model.Role{adminRole},
	}

	if err := db.Create(adminUser).Error; err != nil {
		return fmt.Errorf("failed to create admin user: %w", err)
	}

	log.Info().Msg("Database seeding completed successfully")
	return nil
}

// Close closes the database connection
func Close(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
