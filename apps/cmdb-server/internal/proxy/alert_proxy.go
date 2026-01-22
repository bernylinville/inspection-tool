package proxy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/rs/zerolog"
)

var (
	ErrAlertQueryFailed        = errors.New("alert query failed")
	ErrInvalidAPIKey           = errors.New("invalid API key")
	ErrAlertServiceUnavailable = errors.New("alert service unavailable")
)

type FlashDutyConfig struct {
	Endpoint string
	APIKey   string
	Timeout  time.Duration
}

type AlertListRequest struct {
	StartTime int64  `json:"start_time"`
	EndTime   int64  `json:"end_time"`
	OrderBy   string `json:"orderby,omitempty"`
	Page      int    `json:"page,omitempty"`
	PageSize  int    `json:"page_size,omitempty"`
}

type Alert struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Severity    string `json:"alert_severity"`
	Status      string `json:"status"`
	CreatedAt   int64  `json:"created_at"`
	UpdatedAt   int64  `json:"updated_at"`
}

// AlertDataWrapper wraps the data field from FlashDuty API
type AlertDataWrapper struct {
	Total int64   `json:"total"`
	Items []Alert `json:"items"`
}

type AlertListResponse struct {
	Code    int              `json:"code"`
	Message string           `json:"message"`
	Data    AlertDataWrapper `json:"data"`
}

type IncidentListRequest struct {
	StartTime int64  `json:"start_time"`
	EndTime   int64  `json:"end_time"`
	Status    string `json:"status,omitempty"`
	Page      int    `json:"page,omitempty"`
	PageSize  int    `json:"page_size,omitempty"`
}

type Incident struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
	Status      string `json:"status"`
	CreatedAt   int64  `json:"created_at"`
	UpdatedAt   int64  `json:"updated_at"`
}

// IncidentDataWrapper wraps the data field from FlashDuty API
type IncidentDataWrapper struct {
	Total int64      `json:"total"`
	Items []Incident `json:"items"`
}

type IncidentListResponse struct {
	Code    int                 `json:"code"`
	Message string              `json:"message"`
	Data    IncidentDataWrapper `json:"data"`
}

type AlertProxy struct {
	config     *FlashDutyConfig
	httpClient *resty.Client
	logger     zerolog.Logger
}

// isTransportError checks if the error is a transport-level failure
func isTransportError(err error) bool {
	if errors.Is(err, io.EOF) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	if strings.Contains(err.Error(), "connection refused") {
		return true
	}
	return false
}

func NewAlertProxy(cfg *FlashDutyConfig, logger zerolog.Logger) *AlertProxy {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	httpClient := resty.New().
		SetBaseURL(cfg.Endpoint).
		SetTimeout(timeout).
		SetHeader("Content-Type", "application/json")

	return &AlertProxy{
		config:     cfg,
		httpClient: httpClient,
		logger:     logger.With().Str("component", "alert-proxy").Logger(),
	}
}

func (p *AlertProxy) ListAlerts(ctx context.Context, req *AlertListRequest) (*AlertListResponse, error) {
	p.logger.Debug().
		Int64("start_time", req.StartTime).
		Int64("end_time", req.EndTime).
		Msg("querying alerts")

	var result AlertListResponse

	resp, err := p.httpClient.R().
		SetContext(ctx).
		SetQueryParam("app_key", p.config.APIKey).
		SetBody(req).
		SetResult(&result).
		Post("/alert/list")

	if err != nil {
		p.logger.Error().Err(err).Msg("failed to query alerts")
		if isTransportError(err) {
			return nil, ErrAlertServiceUnavailable
		}
		return nil, ErrAlertQueryFailed
	}

	if resp.StatusCode() == http.StatusUnauthorized {
		return nil, ErrInvalidAPIKey
	}

	if resp.StatusCode() != http.StatusOK {
		p.logger.Error().
			Int("status_code", resp.StatusCode()).
			Str("body", string(resp.Body())).
			Msg("FlashDuty API returned non-200 status")
		return nil, fmt.Errorf("FlashDuty API returned status %d", resp.StatusCode())
	}

	return &result, nil
}

func (p *AlertProxy) ListIncidents(ctx context.Context, req *IncidentListRequest) (*IncidentListResponse, error) {
	p.logger.Debug().
		Int64("start_time", req.StartTime).
		Int64("end_time", req.EndTime).
		Msg("querying incidents")

	var result IncidentListResponse

	resp, err := p.httpClient.R().
		SetContext(ctx).
		SetQueryParam("app_key", p.config.APIKey).
		SetBody(req).
		SetResult(&result).
		Post("/incident/list")

	if err != nil {
		p.logger.Error().Err(err).Msg("failed to query incidents")
		if isTransportError(err) {
			return nil, ErrAlertServiceUnavailable
		}
		return nil, ErrAlertQueryFailed
	}

	if resp.StatusCode() == http.StatusUnauthorized {
		return nil, ErrInvalidAPIKey
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("FlashDuty API returned status %d", resp.StatusCode())
	}

	return &result, nil
}
