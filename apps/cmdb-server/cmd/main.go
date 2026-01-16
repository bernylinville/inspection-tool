package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/spf13/viper"

	"inspection-tool/apps/cmdb-server/internal/api/handler"
	"inspection-tool/apps/cmdb-server/internal/api/router"
	"inspection-tool/apps/cmdb-server/internal/database"
	"inspection-tool/apps/cmdb-server/internal/proxy"
	"inspection-tool/apps/cmdb-server/internal/repository"
	"inspection-tool/apps/cmdb-server/internal/service/asset"
	"inspection-tool/apps/cmdb-server/internal/service/auth"
	inspection2 "inspection-tool/apps/cmdb-server/internal/service/inspection"
	"inspection-tool/apps/cmdb-server/internal/service/role"
	"inspection-tool/apps/cmdb-server/internal/service/sync"
	"inspection-tool/apps/cmdb-server/internal/service/user"
	"inspection-tool/pkg/config"
	"inspection-tool/pkg/n9e"
	"inspection-tool/pkg/vm"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "../cmdb-config.yaml", "path to config file")
	migrateOnly := flag.Bool("migrate", false, "run database migration only")
	flag.Parse()

	// Initialize logger
	log := zerolog.New(os.Stdout).With().Timestamp().Logger()

	// Load configuration
	viper.SetConfigFile(*configPath)
	if err := viper.ReadInConfig(); err != nil {
		log.Fatal().Err(err).Msg("Failed to read config file")
	}

	// Database configuration
	dbConfig := &database.Config{
		Host:         viper.GetString("database.host"),
		Port:         viper.GetInt("database.port"),
		User:         viper.GetString("database.user"),
		Password:     viper.GetString("database.password"),
		DBName:       viper.GetString("database.dbname"),
		SSLMode:      viper.GetString("database.sslmode"),
		MaxIdleConns: viper.GetInt("database.max_idle_conns"),
		MaxOpenConns: viper.GetInt("database.max_open_conns"),
		MaxLifetime:  viper.GetDuration("database.max_lifetime"),
	}

	// Initialize database
	db, err := database.Initialize(dbConfig, log)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize database")
	}
	defer database.Close(db)

	// Run migrations
	if err := database.AutoMigrate(db, log); err != nil {
		log.Fatal().Err(err).Msg("Failed to run database migrations")
	}

	// Seed database with initial data
	if err := database.Seed(db, log); err != nil {
		log.Fatal().Err(err).Msg("Failed to seed database")
	}

	if *migrateOnly {
		log.Info().Msg("Migration completed. Exiting.")
		return
	}

	// Initialize repositories
	userRepo := repository.NewUserRepository(db)
	roleRepo := repository.NewRoleRepository(db)
	permRepo := repository.NewPermissionRepository(db)
	hostRepo := repository.NewHostRepository(db)
	projectRepo := repository.NewProjectRepository(db)
	applicationRepo := repository.NewApplicationRepository(db)
	mysqlRepo := repository.NewMySQLInstanceRepository(db)
	redisRepo := repository.NewRedisInstanceRepository(db)
	nginxRepo := repository.NewNginxInstanceRepository(db)
	tomcatRepo := repository.NewTomcatInstanceRepository(db)
	elasticsearchRepo := repository.NewElasticsearchClusterRepository(db)
	inspectionJobRepo := repository.NewInspectionJobRepository(db)

	// Initialize external clients
	n9eClient := n9e.NewClient(&config.N9EConfig{
		Endpoint: viper.GetString("n9e.base_url"),
		Token:    viper.GetString("n9e.token"),
	}, &config.RetryConfig{
		MaxRetries: 3,
		BaseDelay:  1 * time.Second,
	}, log)

	vmClient := vm.NewClient(&config.VictoriaMetricsConfig{
		Endpoint: viper.GetString("victoriametrics.base_url"),
	}, &config.RetryConfig{
		MaxRetries: 3,
		BaseDelay:  1 * time.Second,
	}, log)

	alertProxy := proxy.NewAlertProxy(&proxy.FlashDutyConfig{
		Endpoint: viper.GetString("flashduty.base_url"),
		APIKey:   viper.GetString("flashduty.app_key"),
	}, log)

	// Initialize services
	jwtSecret := []byte(viper.GetString("jwt.secret"))
	jwtExpireHours := viper.GetInt("jwt.expire_hours")
	authService := auth.NewAuthService(userRepo, jwtSecret, jwtExpireHours, log)

	hostSyncService := sync.NewHostSyncService(n9eClient, hostRepo, log)

	assetService := asset.NewAssetService(db, projectRepo, applicationRepo, hostRepo, mysqlRepo, redisRepo, nginxRepo, tomcatRepo, elasticsearchRepo)

	monitorProxy := proxy.NewMonitorProxy(vmClient, log)

	inspectService := inspection2.NewInspectService(
		inspectionJobRepo,
		viper.GetString("inspection.cli_path"),
		viper.GetString("inspection.config_path"),
		viper.GetString("inspection.output_dir"),
		log,
	)

	userService := user.NewUserService(userRepo, roleRepo, authService, log)

	roleService := role.NewRoleService(roleRepo, permRepo, log)

	// Initialize handlers
	authHandler := handler.NewAuthHandler(authService, userRepo)
	assetHandler := handler.NewAssetHandler(assetService, hostSyncService)
	monitorHandler := handler.NewMonitorHandler(monitorProxy)
	alertHandler := handler.NewAlertHandler(alertProxy)
	inspectionHandler := handler.NewInspectionHandler(inspectService)
	userHandler := handler.NewUserHandler(userService)
	roleHandler := handler.NewRoleHandler(roleService)

	// Start HTTP server
	serverConfig := &router.Config{
		Mode:         viper.GetString("server.mode"),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	router := router.New(*serverConfig, log, router.Handlers{
		Auth:       authHandler,
		User:       userHandler,
		Role:       roleHandler,
		Asset:      assetHandler,
		Monitor:    monitorHandler,
		Alert:      alertHandler,
		Inspection: inspectionHandler,
	})
	router.SetupRoutes()

	if staticPath := viper.GetString("server.static_path"); staticPath != "" {
		router.SetupStaticRoutes(staticPath)
	}

	port := viper.GetString("server.port")
	addr := fmt.Sprintf(":%s", port)
	log.Info().Str("addr", addr).Msg("Starting server")
	if err := router.Engine().Run(addr); err != nil {
		log.Fatal().Err(err).Msg("Failed to start server")
	}
}
