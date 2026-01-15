package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"inspection-tool/apps/cmdb-server/internal/model"
	"inspection-tool/apps/cmdb-server/internal/repository"
	"inspection-tool/apps/cmdb-server/internal/service/inspection"
)

type InspectionHandler struct {
	inspectService *inspection.InspectService
}

func NewInspectionHandler(inspectService *inspection.InspectService) *InspectionHandler {
	return &InspectionHandler{
		inspectService: inspectService,
	}
}

type InspectionResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type InspectionListResponse struct {
	Code       int         `json:"code"`
	Message    string      `json:"message"`
	Data       interface{} `json:"data,omitempty"`
	Total      int64       `json:"total,omitempty"`
	Page       int         `json:"page,omitempty"`
	PageSize   int         `json:"page_size,omitempty"`
	TotalPages int         `json:"total_pages,omitempty"`
}

type CreateJobRequest struct {
	Type        string `json:"type" binding:"required"`
	TriggerType string `json:"trigger_type"`
}

type JobResponse struct {
	ID              int64       `json:"id"`
	Type            string      `json:"type"`
	TriggerType     string      `json:"trigger_type"`
	Status          string      `json:"status"`
	CreatedBy       string      `json:"created_by"`
	CreatedAt       string      `json:"created_at"`
	StartTime       *string     `json:"start_time,omitempty"`
	EndTime         *string     `json:"end_time,omitempty"`
	DurationSeconds int         `json:"duration_seconds,omitempty"`
	ReportExcelPath string      `json:"report_excel_path,omitempty"`
	ReportHTMLPath  string      `json:"report_html_path,omitempty"`
	Summary         interface{} `json:"summary,omitempty"`
	ErrorMessage    string      `json:"error_message,omitempty"`
}

func (h *InspectionHandler) ListJobs(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "20")
	status := c.Query("status")
	jobType := c.Query("type")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	opts := repository.ListOptions{
		Page:     page,
		PageSize: pageSize,
		OrderBy:  "created_at",
		Order:    "desc",
		Filters:  make(map[string]interface{}),
	}

	if status != "" {
		opts.Filters["status"] = status
	}
	if jobType != "" {
		opts.Filters["type"] = jobType
	}

	jobs, total, err := h.inspectService.ListJobs(c.Request.Context(), opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, InspectionResponse{
			Code:    50001,
			Message: "failed to list jobs: " + err.Error(),
		})
		return
	}

	jobResponses := make([]JobResponse, len(jobs))
	for i, job := range jobs {
		jobResponses[i] = toJobResponse(&job)
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	c.JSON(http.StatusOK, InspectionListResponse{
		Code:       0,
		Message:    "success",
		Data:       jobResponses,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	})
}

func (h *InspectionHandler) CreateJob(c *gin.Context) {
	var req CreateJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, InspectionResponse{
			Code:    40001,
			Message: "invalid request: " + err.Error(),
		})
		return
	}

	username, _ := c.Get("username")
	usernameStr, _ := username.(string)

	serviceReq := &inspection.CreateJobRequest{
		Type:        req.Type,
		TriggerType: req.TriggerType,
		CreatedBy:   usernameStr,
	}

	job, err := h.inspectService.CreateJob(c.Request.Context(), serviceReq)
	if err != nil {
		if errors.Is(err, inspection.ErrInvalidJobType) {
			c.JSON(http.StatusBadRequest, InspectionResponse{
				Code:    40002,
				Message: "invalid job type: must be one of host, mysql, redis, nginx, tomcat, elasticsearch",
			})
			return
		}
		if errors.Is(err, inspection.ErrCLINotFound) {
			c.JSON(http.StatusInternalServerError, InspectionResponse{
				Code:    50002,
				Message: "inspection CLI not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, InspectionResponse{
			Code:    50001,
			Message: "failed to create job: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, InspectionResponse{
		Code:    0,
		Message: "job created successfully",
		Data:    toJobResponse(job),
	})
}

func (h *InspectionHandler) GetJob(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, InspectionResponse{
			Code:    40001,
			Message: "invalid job id",
		})
		return
	}

	job, err := h.inspectService.GetJob(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, inspection.ErrJobNotFound) {
			c.JSON(http.StatusNotFound, InspectionResponse{
				Code:    40401,
				Message: "job not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, InspectionResponse{
			Code:    50001,
			Message: "failed to get job: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, InspectionResponse{
		Code:    0,
		Message: "success",
		Data:    toJobResponse(job),
	})
}

func (h *InspectionHandler) DeleteJob(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, InspectionResponse{
			Code:    40001,
			Message: "invalid job id",
		})
		return
	}

	err = h.inspectService.DeleteJob(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, inspection.ErrJobNotFound) {
			c.JSON(http.StatusNotFound, InspectionResponse{
				Code:    40401,
				Message: "job not found",
			})
			return
		}
		if errors.Is(err, inspection.ErrJobRunning) {
			c.JSON(http.StatusConflict, InspectionResponse{
				Code:    40901,
				Message: "cannot delete a running job",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, InspectionResponse{
			Code:    50001,
			Message: "failed to delete job: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, InspectionResponse{
		Code:    0,
		Message: "job deleted successfully",
	})
}

func toJobResponse(job *model.InspectionJob) JobResponse {
	if job == nil {
		return JobResponse{}
	}

	resp := JobResponse{
		ID:              job.ID,
		Type:            job.Type,
		TriggerType:     job.TriggerType,
		Status:          job.Status,
		CreatedBy:       job.CreatedBy,
		CreatedAt:       job.CreatedAt.Format("2006-01-02 15:04:05"),
		DurationSeconds: job.DurationSeconds,
		ReportExcelPath: job.ReportExcelPath,
		ReportHTMLPath:  job.ReportHTMLPath,
		ErrorMessage:    job.ErrorMessage,
	}

	if job.StartTime != nil {
		startStr := job.StartTime.Format("2006-01-02 15:04:05")
		resp.StartTime = &startStr
	}

	if job.EndTime != nil {
		endStr := job.EndTime.Format("2006-01-02 15:04:05")
		resp.EndTime = &endStr
	}

	if len(job.Summary) > 0 {
		resp.Summary = job.Summary
	}

	return resp
}
