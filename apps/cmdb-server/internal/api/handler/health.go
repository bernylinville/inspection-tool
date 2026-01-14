package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// HealthHandler handles health check endpoints
type HealthHandler struct {
	db    *gorm.DB
	redis *redis.Client
}

// NewHealthHandler creates a new HealthHandler
func NewHealthHandler(db *gorm.DB, redisClient *redis.Client) *HealthHandler {
	return &HealthHandler{
		db:    db,
		redis: redisClient,
	}
}

// ComponentStatus represents the status of a single component
type ComponentStatus struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
	Latency string `json:"latency,omitempty"`
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status     string                     `json:"status"`
	Timestamp  string                     `json:"timestamp"`
	Components map[string]ComponentStatus `json:"components"`
}

// HealthCheck handles GET /health
// @Summary Health check endpoint
// @Description Check the health status of the service and its dependencies
// @Tags health
// @Produce json
// @Success 200 {object} HealthResponse
// @Failure 503 {object} HealthResponse
// @Router /health [get]
func (h *HealthHandler) HealthCheck(c *gin.Context) {
	response := HealthResponse{
		Status:     "healthy",
		Timestamp:  time.Now().Format(time.RFC3339),
		Components: make(map[string]ComponentStatus),
	}

	// Check database connection
	dbStatus := h.checkDatabase()
	response.Components["database"] = dbStatus
	if dbStatus.Status != "healthy" {
		response.Status = "unhealthy"
	}

	// Check Redis connection (if configured)
	if h.redis != nil {
		redisStatus := h.checkRedis()
		response.Components["redis"] = redisStatus
		if redisStatus.Status != "healthy" {
			response.Status = "unhealthy"
		}
	}

	statusCode := http.StatusOK
	if response.Status == "unhealthy" {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, response)
}

// checkDatabase checks the database connection
func (h *HealthHandler) checkDatabase() ComponentStatus {
	start := time.Now()

	sqlDB, err := h.db.DB()
	if err != nil {
		return ComponentStatus{
			Status:  "unhealthy",
			Message: "failed to get database connection: " + err.Error(),
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := sqlDB.PingContext(ctx); err != nil {
		return ComponentStatus{
			Status:  "unhealthy",
			Message: "database ping failed: " + err.Error(),
		}
	}

	return ComponentStatus{
		Status:  "healthy",
		Latency: time.Since(start).String(),
	}
}

// checkRedis checks the Redis connection
func (h *HealthHandler) checkRedis() ComponentStatus {
	start := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := h.redis.Ping(ctx).Err(); err != nil {
		return ComponentStatus{
			Status:  "unhealthy",
			Message: "redis ping failed: " + err.Error(),
		}
	}

	return ComponentStatus{
		Status:  "healthy",
		Latency: time.Since(start).String(),
	}
}

// SimpleHealthCheck handles GET /health for simple status check (no dependencies)
// This is used by the router's default health endpoint
func SimpleHealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"message":   "CMDB Server is running",
		"timestamp": time.Now().Format(time.RFC3339),
	})
}
