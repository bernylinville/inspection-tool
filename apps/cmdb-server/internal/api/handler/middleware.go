package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"inspection-tool/apps/cmdb-server/internal/service/sync"
)

type DiscoverMiddlewareResponse struct {
	Code    int                       `json:"code"`
	Message string                    `json:"message"`
	Data    *DiscoverMiddlewareResult `json:"data,omitempty"`
}

type DiscoverMiddlewareResult struct {
	MySQL         int    `json:"mysql"`
	Redis         int    `json:"redis"`
	Nginx         int    `json:"nginx"`
	Tomcat        int    `json:"tomcat"`
	Elasticsearch int    `json:"elasticsearch"`
	Duration      string `json:"duration"`
}

type MiddlewareHandler struct {
	discoveryService *sync.InstanceDiscoveryService
}

func NewMiddlewareHandler(discoveryService *sync.InstanceDiscoveryService) *MiddlewareHandler {
	return &MiddlewareHandler{
		discoveryService: discoveryService,
	}
}

func (h *MiddlewareHandler) DiscoverInstances(c *gin.Context) {
	result, err := h.discoveryService.DiscoverAll(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, DiscoverMiddlewareResponse{
			Code:    500,
			Message: "middleware discovery failed: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, DiscoverMiddlewareResponse{
		Code:    0,
		Message: "middleware discovery completed",
		Data: &DiscoverMiddlewareResult{
			MySQL:         result.MySQL,
			Redis:         result.Redis,
			Nginx:         result.Nginx,
			Tomcat:        result.Tomcat,
			Elasticsearch: result.Elasticsearch,
			Duration:      result.Duration.String(),
		},
	})
}
