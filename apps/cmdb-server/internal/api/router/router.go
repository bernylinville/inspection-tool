package router

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"inspection-tool/apps/cmdb-server/internal/api/handler"
)

type Config struct {
	Mode         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

type Handlers struct {
	Auth       *handler.AuthHandler
	User       *handler.UserHandler
	Role       *handler.RoleHandler
	Asset      *handler.AssetHandler
	Monitor    *handler.MonitorHandler
	Alert      *handler.AlertHandler
	Inspection *handler.InspectionHandler
}

type Router struct {
	engine   *gin.Engine
	logger   zerolog.Logger
	config   Config
	handlers Handlers
}

func New(config Config, logger zerolog.Logger, handlers Handlers) *Router {
	switch config.Mode {
	case "release":
		gin.SetMode(gin.ReleaseMode)
	case "test":
		gin.SetMode(gin.TestMode)
	default:
		gin.SetMode(gin.DebugMode)
	}

	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(requestLogger(logger))
	engine.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	return &Router{
		engine:   engine,
		logger:   logger,
		config:   config,
		handlers: handlers,
	}
}

func (r *Router) Engine() *gin.Engine {
	return r.engine
}

func (r *Router) SetupRoutes() {
	r.engine.GET("/health", healthCheck)

	v1 := r.engine.Group("/api/v1")
	{
		r.setupPublicRoutes(v1)
		r.setupProtectedRoutes(v1)
	}
}

func (r *Router) SetupStaticRoutes(staticPath string) {
	staticPath = filepath.Clean(staticPath)
	if _, err := os.Stat(staticPath); os.IsNotExist(err) {
		r.logger.Warn().Str("path", staticPath).Msg("Static files directory not found, skipping static file serving")
		return
	}

	r.engine.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
			return
		}

		filePath := filepath.Join(staticPath, c.Request.URL.Path)
		if _, err := os.Stat(filePath); err == nil {
			c.File(filePath)
			return
		}

		c.File(filepath.Join(staticPath, "index.html"))
	})
}

func (r *Router) setupPublicRoutes(rg *gin.RouterGroup) {
	auth := rg.Group("/auth")
	{
		if r.handlers.Auth != nil {
			auth.POST("/login", r.handlers.Auth.Login)
			auth.POST("/refresh", r.handlers.Auth.Refresh)
		} else {
			auth.POST("/login", placeholder("login"))
			auth.POST("/refresh", placeholder("refresh"))
		}
	}
}

func (r *Router) setupProtectedRoutes(rg *gin.RouterGroup) {
	auth := rg.Group("/auth")
	{
		if r.handlers.Auth != nil {
			auth.POST("/logout", r.handlers.Auth.Logout)
			auth.GET("/codes", r.handlers.Auth.GetAccessCodes)
		} else {
			auth.POST("/logout", placeholder("logout"))
			auth.GET("/codes", placeholder("get access codes"))
		}
	}

	rg.GET("/user/info", func(c *gin.Context) {
		if r.handlers.Auth != nil {
			r.handlers.Auth.GetCurrentUser(c)
		} else {
			placeholder("get current user")(c)
		}
	})

	users := rg.Group("/users")
	{
		if r.handlers.User != nil {
			users.GET("", r.handlers.User.ListUsers)
			users.POST("", r.handlers.User.CreateUser)
			users.GET("/:id", r.handlers.User.GetUser)
			users.PUT("/:id", r.handlers.User.UpdateUser)
			users.DELETE("/:id", r.handlers.User.DeleteUser)
			users.PUT("/:id/roles", r.handlers.User.AssignRoles)
			users.PUT("/:id/password", r.handlers.User.ChangePassword)
		} else {
			users.GET("", placeholder("list users"))
			users.POST("", placeholder("create user"))
			users.GET("/:id", placeholder("get user"))
			users.PUT("/:id", placeholder("update user"))
			users.DELETE("/:id", placeholder("delete user"))
		}
	}

	roles := rg.Group("/roles")
	{
		if r.handlers.Role != nil {
			roles.GET("", r.handlers.Role.ListRoles)
			roles.POST("", r.handlers.Role.CreateRole)
			roles.GET("/:id", r.handlers.Role.GetRole)
			roles.PUT("/:id", r.handlers.Role.UpdateRole)
			roles.DELETE("/:id", r.handlers.Role.DeleteRole)
			roles.PUT("/:id/permissions", r.handlers.Role.AssignPermissions)
		} else {
			roles.GET("", placeholder("list roles"))
			roles.POST("", placeholder("create role"))
			roles.GET("/:id", placeholder("get role"))
			roles.PUT("/:id", placeholder("update role"))
			roles.DELETE("/:id", placeholder("delete role"))
		}
	}

	projects := rg.Group("/projects")
	{
		if r.handlers.Asset != nil {
			projects.GET("", r.handlers.Asset.ListProjects)
			projects.POST("", r.handlers.Asset.CreateProject)
			projects.GET("/:id", r.handlers.Asset.GetProject)
			projects.PUT("/:id", r.handlers.Asset.UpdateProject)
			projects.DELETE("/:id", r.handlers.Asset.DeleteProject)
		} else {
			projects.GET("", placeholder("list projects"))
			projects.POST("", placeholder("create project"))
			projects.GET("/:id", placeholder("get project"))
			projects.PUT("/:id", placeholder("update project"))
			projects.DELETE("/:id", placeholder("delete project"))
		}
	}

	applications := rg.Group("/applications")
	{
		if r.handlers.Asset != nil {
			applications.GET("", r.handlers.Asset.ListApplications)
			applications.POST("", r.handlers.Asset.CreateApplication)
			applications.GET("/:id", r.handlers.Asset.GetApplication)
			applications.PUT("/:id", r.handlers.Asset.UpdateApplication)
			applications.DELETE("/:id", r.handlers.Asset.DeleteApplication)
		} else {
			applications.GET("", placeholder("list applications"))
			applications.POST("", placeholder("create application"))
			applications.GET("/:id", placeholder("get application"))
			applications.PUT("/:id", placeholder("update application"))
			applications.DELETE("/:id", placeholder("delete application"))
		}
	}

	hosts := rg.Group("/hosts")
	{
		if r.handlers.Asset != nil {
			hosts.GET("", r.handlers.Asset.ListHosts)
			hosts.POST("", r.handlers.Asset.CreateHost)
			hosts.GET("/:id", r.handlers.Asset.GetHost)
			hosts.PUT("/:id", r.handlers.Asset.UpdateHost)
			hosts.DELETE("/:id", r.handlers.Asset.DeleteHost)
			hosts.POST("/sync", r.handlers.Asset.SyncHosts)
			hosts.GET("/:id/metrics", placeholder("get host metrics"))
		} else {
			hosts.GET("", placeholder("list hosts"))
			hosts.POST("", placeholder("create host"))
			hosts.GET("/:id", placeholder("get host"))
			hosts.PUT("/:id", placeholder("update host"))
			hosts.DELETE("/:id", placeholder("delete host"))
			hosts.POST("/sync", placeholder("sync hosts"))
			hosts.GET("/:id/metrics", placeholder("get host metrics"))
		}
	}

	r.setupMiddlewareRoutes(rg)

	monitor := rg.Group("/monitor")
	{
		if r.handlers.Monitor != nil {
			monitor.GET("/query", r.handlers.Monitor.Query)
			monitor.GET("/query_range", r.handlers.Monitor.QueryRange)
		} else {
			monitor.GET("/query", placeholder("query metrics"))
			monitor.GET("/query_range", placeholder("query range metrics"))
		}
	}

	alerts := rg.Group("/alerts")
	{
		if r.handlers.Alert != nil {
			alerts.GET("", r.handlers.Alert.ListAlerts)
			alerts.GET("/:id", r.handlers.Alert.GetAlert)
		} else {
			alerts.GET("", placeholder("list alerts"))
			alerts.GET("/:id", placeholder("get alert"))
		}
	}

	incidents := rg.Group("/incidents")
	{
		if r.handlers.Alert != nil {
			incidents.GET("", r.handlers.Alert.ListIncidents)
			incidents.GET("/:id", r.handlers.Alert.GetIncident)
		} else {
			incidents.GET("", placeholder("list incidents"))
			incidents.GET("/:id", placeholder("get incident"))
		}
	}

	inspection := rg.Group("/inspection")
	{
		jobs := inspection.Group("/jobs")
		{
			if r.handlers.Inspection != nil {
				jobs.GET("", r.handlers.Inspection.ListJobs)
				jobs.POST("", r.handlers.Inspection.CreateJob)
				jobs.GET("/:id", r.handlers.Inspection.GetJob)
				jobs.DELETE("/:id", r.handlers.Inspection.DeleteJob)
			} else {
				jobs.GET("", placeholder("list jobs"))
				jobs.POST("", placeholder("create job"))
				jobs.GET("/:id", placeholder("get job"))
				jobs.DELETE("/:id", placeholder("delete job"))
			}
		}
	}
}

func (r *Router) setupMiddlewareRoutes(rg *gin.RouterGroup) {
	mysql := rg.Group("/mysql")
	{
		if r.handlers.Asset != nil {
			mysql.GET("", r.handlers.Asset.ListMySQLInstances)
			mysql.GET("/:id", r.handlers.Asset.GetMySQLInstance)
			mysql.DELETE("/:id", r.handlers.Asset.DeleteMySQLInstance)
		} else {
			mysql.GET("", placeholder("list mysql instances"))
			mysql.GET("/:id", placeholder("get mysql instance"))
			mysql.DELETE("/:id", placeholder("delete mysql instance"))
		}
	}

	redis := rg.Group("/redis")
	{
		if r.handlers.Asset != nil {
			redis.GET("", r.handlers.Asset.ListRedisInstances)
			redis.GET("/:id", r.handlers.Asset.GetRedisInstance)
			redis.DELETE("/:id", r.handlers.Asset.DeleteRedisInstance)
		} else {
			redis.GET("", placeholder("list redis instances"))
			redis.GET("/:id", placeholder("get redis instance"))
			redis.DELETE("/:id", placeholder("delete redis instance"))
		}
	}

	nginx := rg.Group("/nginx")
	{
		if r.handlers.Asset != nil {
			nginx.GET("", r.handlers.Asset.ListNginxInstances)
			nginx.GET("/:id", r.handlers.Asset.GetNginxInstance)
			nginx.DELETE("/:id", r.handlers.Asset.DeleteNginxInstance)
		} else {
			nginx.GET("", placeholder("list nginx instances"))
			nginx.GET("/:id", placeholder("get nginx instance"))
			nginx.DELETE("/:id", placeholder("delete nginx instance"))
		}
	}

	tomcat := rg.Group("/tomcat")
	{
		if r.handlers.Asset != nil {
			tomcat.GET("", r.handlers.Asset.ListTomcatInstances)
			tomcat.GET("/:id", r.handlers.Asset.GetTomcatInstance)
			tomcat.DELETE("/:id", r.handlers.Asset.DeleteTomcatInstance)
		} else {
			tomcat.GET("", placeholder("list tomcat instances"))
			tomcat.GET("/:id", placeholder("get tomcat instance"))
			tomcat.DELETE("/:id", placeholder("delete tomcat instance"))
		}
	}

	elasticsearch := rg.Group("/elasticsearch")
	{
		if r.handlers.Asset != nil {
			elasticsearch.GET("", r.handlers.Asset.ListElasticsearchClusters)
			elasticsearch.GET("/:id", r.handlers.Asset.GetElasticsearchCluster)
			elasticsearch.DELETE("/:id", r.handlers.Asset.DeleteElasticsearchCluster)
		} else {
			elasticsearch.GET("", placeholder("list elasticsearch clusters"))
			elasticsearch.GET("/:id", placeholder("get elasticsearch cluster"))
			elasticsearch.DELETE("/:id", placeholder("delete elasticsearch cluster"))
		}
	}

	rg.POST("/instances/discover", placeholder("discover instances"))
}

func healthCheck(c *gin.Context) {
	c.JSON(200, gin.H{
		"status":  "ok",
		"message": "CMDB Server is running",
	})
}

func placeholder(name string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(501, gin.H{
			"code":    501,
			"message": "Not implemented: " + name,
		})
	}
}

func requestLogger(logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		statusCode := c.Writer.Status()

		event := logger.Info()
		if statusCode >= 400 {
			event = logger.Warn()
		}
		if statusCode >= 500 {
			event = logger.Error()
		}

		if raw != "" {
			path = path + "?" + raw
		}

		event.
			Str("method", c.Request.Method).
			Str("path", path).
			Int("status", statusCode).
			Dur("latency", latency).
			Str("client_ip", c.ClientIP()).
			Msg("HTTP request")
	}
}
