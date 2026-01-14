package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"inspection-tool/apps/cmdb-server/internal/proxy"
)

type MonitorHandler struct {
	monitorProxy *proxy.MonitorProxy
}

func NewMonitorHandler(monitorProxy *proxy.MonitorProxy) *MonitorHandler {
	return &MonitorHandler{
		monitorProxy: monitorProxy,
	}
}

type MonitorQueryResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type MonitorEChartsResponse struct {
	Code    int                    `json:"code"`
	Message string                 `json:"message"`
	Data    *proxy.EChartsResponse `json:"data,omitempty"`
}

func (h *MonitorHandler) Query(c *gin.Context) {
	query := c.Query("query")
	if query == "" {
		c.JSON(http.StatusBadRequest, MonitorQueryResponse{
			Code:    40001,
			Message: "query parameter is required",
		})
		return
	}

	req := &proxy.QueryRequest{
		Query: query,
	}

	resp, err := h.monitorProxy.Query(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, MonitorQueryResponse{
			Code:    50001,
			Message: "query failed: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, MonitorQueryResponse{
		Code:    0,
		Message: "success",
		Data:    resp,
	})
}

func (h *MonitorHandler) QueryRange(c *gin.Context) {
	query := c.Query("query")
	if query == "" {
		c.JSON(http.StatusBadRequest, MonitorQueryResponse{
			Code:    40001,
			Message: "query parameter is required",
		})
		return
	}

	startStr := c.Query("start")
	endStr := c.Query("end")
	stepStr := c.DefaultQuery("step", "60")

	if startStr == "" || endStr == "" {
		c.JSON(http.StatusBadRequest, MonitorQueryResponse{
			Code:    40001,
			Message: "start and end parameters are required",
		})
		return
	}

	start, err := strconv.ParseInt(startStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, MonitorQueryResponse{
			Code:    40001,
			Message: "invalid start parameter",
		})
		return
	}

	end, err := strconv.ParseInt(endStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, MonitorQueryResponse{
			Code:    40001,
			Message: "invalid end parameter",
		})
		return
	}

	step, err := strconv.ParseInt(stepStr, 10, 64)
	if err != nil {
		step = 60
	}

	format := c.DefaultQuery("format", "raw")

	req := &proxy.QueryRangeRequest{
		Query: query,
		Start: start,
		End:   end,
		Step:  step,
	}

	if format == "echarts" {
		resp, err := h.monitorProxy.QueryRangeForECharts(c.Request.Context(), req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, MonitorEChartsResponse{
				Code:    50001,
				Message: "query range failed: " + err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, MonitorEChartsResponse{
			Code:    0,
			Message: "success",
			Data:    resp,
		})
		return
	}

	resp, err := h.monitorProxy.QueryRange(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, MonitorQueryResponse{
			Code:    50001,
			Message: "query range failed: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, MonitorQueryResponse{
		Code:    0,
		Message: "success",
		Data:    resp,
	})
}
