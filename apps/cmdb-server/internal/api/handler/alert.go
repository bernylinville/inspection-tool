package handler

import (
	"errors"
	"net/http"
	"strconv"
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
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Total   int64       `json:"total,omitempty"`
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
		c.JSON(http.StatusInternalServerError, AlertResponse{
			Code:    50001,
			Message: "failed to query alerts: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, AlertResponse{
		Code:    0,
		Message: "success",
		Data:    resp.Data,
		Total:   resp.Total,
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
		c.JSON(http.StatusInternalServerError, AlertResponse{
			Code:    50001,
			Message: "failed to query incidents: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, AlertResponse{
		Code:    0,
		Message: "success",
		Data:    resp.Data,
		Total:   resp.Total,
	})
}

func (h *AlertHandler) GetIncident(c *gin.Context) {
	_ = c.Param("id")

	c.JSON(http.StatusNotImplemented, AlertResponse{
		Code:    50100,
		Message: "FlashDuty API does not support single incident query. Use list endpoint with filters.",
	})
}
