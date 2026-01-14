package router

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

type Config struct {
	Mode         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

type Router struct {
	engine *gin.Engine
	logger zerolog.Logger
	config Config
}

func New(config Config, logger zerolog.Logger) *Router {
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
		engine: engine,
		logger: logger,
		config: config,
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

func (r *Router) setupPublicRoutes(rg *gin.RouterGroup) {
	auth := rg.Group("/auth")
	{
		auth.POST("/login", placeholder("login"))
		auth.POST("/refresh", placeholder("refresh"))
	}
}

func (r *Router) setupProtectedRoutes(rg *gin.RouterGroup) {
	auth := rg.Group("/auth")
	{
		auth.POST("/logout", placeholder("logout"))
	}

	users := rg.Group("/users")
	{
		users.GET("", placeholder("list users"))
		users.POST("", placeholder("create user"))
		users.GET("/:id", placeholder("get user"))
		users.PUT("/:id", placeholder("update user"))
		users.DELETE("/:id", placeholder("delete user"))
	}

	roles := rg.Group("/roles")
	{
		roles.GET("", placeholder("list roles"))
		roles.POST("", placeholder("create role"))
		roles.GET("/:id", placeholder("get role"))
		roles.PUT("/:id", placeholder("update role"))
		roles.DELETE("/:id", placeholder("delete role"))
	}

	projects := rg.Group("/projects")
	{
		projects.GET("", placeholder("list projects"))
		projects.POST("", placeholder("create project"))
		projects.GET("/:id", placeholder("get project"))
		projects.PUT("/:id", placeholder("update project"))
		projects.DELETE("/:id", placeholder("delete project"))
	}

	applications := rg.Group("/applications")
	{
		applications.GET("", placeholder("list applications"))
		applications.POST("", placeholder("create application"))
		applications.GET("/:id", placeholder("get application"))
		applications.PUT("/:id", placeholder("update application"))
		applications.DELETE("/:id", placeholder("delete application"))
	}

	hosts := rg.Group("/hosts")
	{
		hosts.GET("", placeholder("list hosts"))
		hosts.POST("", placeholder("create host"))
		hosts.GET("/:id", placeholder("get host"))
		hosts.PUT("/:id", placeholder("update host"))
		hosts.DELETE("/:id", placeholder("delete host"))
		hosts.POST("/sync", placeholder("sync hosts"))
		hosts.GET("/:id/metrics", placeholder("get host metrics"))
	}

	r.setupMiddlewareRoutes(rg)

	monitor := rg.Group("/monitor")
	{
		monitor.GET("/query", placeholder("query metrics"))
		monitor.GET("/query_range", placeholder("query range metrics"))
	}

	alerts := rg.Group("/alerts")
	{
		alerts.GET("", placeholder("list alerts"))
		alerts.GET("/:id", placeholder("get alert"))
	}

	incidents := rg.Group("/incidents")
	{
		incidents.GET("", placeholder("list incidents"))
		incidents.GET("/:id", placeholder("get incident"))
	}

	inspection := rg.Group("/inspection")
	{
		jobs := inspection.Group("/jobs")
		{
			jobs.GET("", placeholder("list jobs"))
			jobs.POST("", placeholder("create job"))
			jobs.GET("/:id", placeholder("get job"))
			jobs.DELETE("/:id", placeholder("delete job"))
		}
	}
}

func (r *Router) setupMiddlewareRoutes(rg *gin.RouterGroup) {
	mysql := rg.Group("/mysql")
	{
		mysql.GET("", placeholder("list mysql instances"))
		mysql.GET("/:id", placeholder("get mysql instance"))
		mysql.DELETE("/:id", placeholder("delete mysql instance"))
	}

	redis := rg.Group("/redis")
	{
		redis.GET("", placeholder("list redis instances"))
		redis.GET("/:id", placeholder("get redis instance"))
		redis.DELETE("/:id", placeholder("delete redis instance"))
	}

	nginx := rg.Group("/nginx")
	{
		nginx.GET("", placeholder("list nginx instances"))
		nginx.GET("/:id", placeholder("get nginx instance"))
		nginx.DELETE("/:id", placeholder("delete nginx instance"))
	}

	tomcat := rg.Group("/tomcat")
	{
		tomcat.GET("", placeholder("list tomcat instances"))
		tomcat.GET("/:id", placeholder("get tomcat instance"))
		tomcat.DELETE("/:id", placeholder("delete tomcat instance"))
	}

	elasticsearch := rg.Group("/elasticsearch")
	{
		elasticsearch.GET("", placeholder("list elasticsearch clusters"))
		elasticsearch.GET("/:id", placeholder("get elasticsearch cluster"))
		elasticsearch.DELETE("/:id", placeholder("delete elasticsearch cluster"))
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
