package handler

import (
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"inspection-tool/apps/cmdb-server/internal/proxy"
)

type AlertHandler struct {
	alertProxy *proxy.AlertProxy
}

func NewAlertHandler(alertProxy *proxy.AlertProxy) *AlertHandler {
	return &AlertHandler{
		alertProxy: alertProxy,
	}
}

type AlertResponse struct {
	Code               int         `json:"code"`
	Message            string      `json:"message"`
	Data               interface{} `json:"data,omitempty"`
	Total              int64       `json:"total,omitempty"`
	ServiceUnavailable bool        `json:"service_unavailable,omitempty"`
}

// PaginatedAlertResponse represents paginated alert response matching frontend expectations
type PaginatedAlertResponse struct {
	Items      interface{} `json:"items"`
	Total      int64       `json:"total"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
	TotalPages int         `json:"total_pages"`
}

type WrappedPaginatedResponse struct {
	Code    int                    `json:"code"`
	Message string                 `json:"message,omitempty"`
	Data    PaginatedAlertResponse `json:"data"`
}

const defaultTimeRangeSeconds = 86400

func (h *AlertHandler) ListAlerts(c *gin.Context) {
	now := time.Now().Unix()
	defaultStart := now - defaultTimeRangeSeconds

	startStr := c.DefaultQuery("start_time", strconv.FormatInt(defaultStart, 10))
	endStr := c.DefaultQuery("end_time", strconv.FormatInt(now, 10))
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "20")
	orderBy := c.DefaultQuery("orderby", "created_at")

	start, err := strconv.ParseInt(startStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, AlertResponse{
			Code:    40001,
			Message: "invalid start_time parameter",
		})
		return
	}

	end, err := strconv.ParseInt(endStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, AlertResponse{
			Code:    40001,
			Message: "invalid end_time parameter",
		})
		return
	}

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	req := &proxy.AlertListRequest{
		StartTime: start,
		EndTime:   end,
		OrderBy:   orderBy,
		Page:      page,
		PageSize:  pageSize,
	}

	resp, err := h.alertProxy.ListAlerts(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, proxy.ErrInvalidAPIKey) {
			c.JSON(http.StatusUnauthorized, AlertResponse{
				Code:    40100,
				Message: "invalid FlashDuty API key",
			})
			return
		}
		if errors.Is(err, proxy.ErrAlertServiceUnavailable) {
			c.JSON(http.StatusOK, WrappedPaginatedResponse{
				Code: 0,
				Data: PaginatedAlertResponse{
					Items:      []interface{}{},
					Total:      0,
					Page:       page,
					PageSize:   pageSize,
					TotalPages: 0,
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, AlertResponse{
			Code:    50001,
			Message: "failed to query alerts: " + err.Error(),
		})
		return
	}

	// Calculate total pages
	totalPages := int(math.Ceil(float64(resp.Data.Total) / float64(pageSize)))

	c.JSON(http.StatusOK, WrappedPaginatedResponse{
		Code: 0,
		Data: PaginatedAlertResponse{
			Items:      resp.Data.Items,
			Total:      resp.Data.Total,
			Page:       page,
			PageSize:   pageSize,
			TotalPages: totalPages,
		},
	})
}

func (h *AlertHandler) GetAlert(c *gin.Context) {
	_ = c.Param("id")

	c.JSON(http.StatusNotImplemented, AlertResponse{
		Code:    50100,
		Message: "FlashDuty API does not support single alert query. Use list endpoint with filters.",
	})
}

func (h *AlertHandler) ListIncidents(c *gin.Context) {
	now := time.Now().Unix()
	defaultStart := now - defaultTimeRangeSeconds

	startStr := c.DefaultQuery("start_time", strconv.FormatInt(defaultStart, 10))
	endStr := c.DefaultQuery("end_time", strconv.FormatInt(now, 10))
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "20")
	status := c.Query("status")

	start, err := strconv.ParseInt(startStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, AlertResponse{
			Code:    40001,
			Message: "invalid start_time parameter",
		})
		return
	}

	end, err := strconv.ParseInt(endStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, AlertResponse{
			Code:    40001,
			Message: "invalid end_time parameter",
		})
		return
	}

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	req := &proxy.IncidentListRequest{
		StartTime: start,
		EndTime:   end,
		Status:    status,
		Page:      page,
		PageSize:  pageSize,
	}

	resp, err := h.alertProxy.ListIncidents(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, proxy.ErrInvalidAPIKey) {
			c.JSON(http.StatusUnauthorized, AlertResponse{
				Code:    40100,
				Message: "invalid FlashDuty API key",
			})
			return
		}
		if errors.Is(err, proxy.ErrAlertServiceUnavailable) {
			c.JSON(http.StatusOK, WrappedPaginatedResponse{
				Code: 0,
				Data: PaginatedAlertResponse{
					Items:      []interface{}{},
					Total:      0,
					Page:       page,
					PageSize:   pageSize,
					TotalPages: 0,
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, AlertResponse{
			Code:    50001,
			Message: "failed to query incidents: " + err.Error(),
		})
		return
	}

	// Calculate total pages
	totalPages := int(math.Ceil(float64(resp.Data.Total) / float64(pageSize)))

	c.JSON(http.StatusOK, WrappedPaginatedResponse{
		Code: 0,
		Data: PaginatedAlertResponse{
			Items:      resp.Data.Items,
			Total:      resp.Data.Total,
			Page:       page,
			PageSize:   pageSize,
			TotalPages: totalPages,
		},
	})
}

func (h *AlertHandler) GetIncident(c *gin.Context) {
	_ = c.Param("id")

	c.JSON(http.StatusNotImplemented, AlertResponse{
		Code:    50100,
		Message: "FlashDuty API does not support single incident query. Use list endpoint with filters.",
	})
}

// AlertTrendResult represents the statistics data for alert trends
type AlertTrendResult struct {
	Labels   []string `json:"labels"`
	Critical []int    `json:"critical"`
	Warning  []int    `json:"warning"`
}

// GetStatistics returns alert trend statistics for dashboard
func (h *AlertHandler) GetStatistics(c *gin.Context) {
	now := time.Now()

	// Initialize 7-day buckets
	labels := make([]string, 7)
	critical := make([]int, 7)
	warning := make([]int, 7)

	for i := 0; i < 7; i++ {
		date := now.AddDate(0, 0, i-6)
		labels[i] = fmt.Sprintf("%d/%d", int(date.Month()), date.Day())
	}

	// Fetch 7 days of alerts from FlashDuty
	startTime := now.AddDate(0, 0, -6).Truncate(24 * time.Hour).Unix()
	endTime := now.Unix()

	req := &proxy.AlertListRequest{
		StartTime: startTime,
		EndTime:   endTime,
		PageSize:  1000, // Get all alerts in range
	}

	resp, err := h.alertProxy.ListAlerts(c.Request.Context(), req)
	if err != nil {
		// Return empty data with service_unavailable flag on error
		c.JSON(http.StatusOK, gin.H{
			"code":                0,
			"message":             "success",
			"data":                gin.H{"labels": labels, "critical": critical, "warning": warning},
			"service_unavailable": true,
		})
		return
	}

	// Aggregate by day and severity
	for _, alert := range resp.Data.Items {
		alertTime := time.Unix(alert.CreatedAt, 0)
		dayIndex := int(now.Sub(alertTime).Hours() / 24)
		if dayIndex < 0 || dayIndex > 6 {
			continue
		}
		idx := 6 - dayIndex // Reverse index (oldest first)

		switch strings.ToLower(alert.Severity) {
		case "critical", "p0", "p1":
			critical[idx]++
		case "warning", "p2", "p3", "info":
			warning[idx]++
		default:
			warning[idx]++ // Default to warning
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":                0,
		"message":             "success",
		"data":                gin.H{"labels": labels, "critical": critical, "warning": warning},
		"service_unavailable": false,
	})
}
