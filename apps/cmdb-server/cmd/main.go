package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/spf13/viper"

	"inspection-tool/apps/cmdb-server/internal/api/router"
	"inspection-tool/apps/cmdb-server/internal/database"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "configs/config.yaml", "path to config file")
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

	if *migrateOnly {
		log.Info().Msg("Migration completed. Exiting.")
		return
	}

	// Start HTTP server
	serverConfig := &router.Config{
		Mode:         viper.GetString("server.mode"),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	router := router.New(*serverConfig, log, router.Handlers{})
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
