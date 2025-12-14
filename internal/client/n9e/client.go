// Package n9e provides a client for the N9E (Nightingale) API.
package n9e

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/rs/zerolog"

	"inspection-tool/internal/config"
	"inspection-tool/internal/model"
)

// Client is a client for the N9E API.
type Client struct {
	endpoint   string             // N9E API endpoint
	token      string             // Authentication token
	timeout    time.Duration      // Request timeout
	retry      config.RetryConfig // Retry configuration
	query      string             // Host filter query (e.g., "items=短剧项目")
	httpClient *resty.Client      // HTTP client
	logger     zerolog.Logger     // Logger
}

// NewClient creates a new N9E API client.
func NewClient(cfg *config.N9EConfig, retryCfg *config.RetryConfig, logger zerolog.Logger) *Client {
	// Set default timeout if not specified
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	// Set default retry config if not specified
	retry := config.RetryConfig{
		MaxRetries: 3,
		BaseDelay:  1 * time.Second,
	}
	if retryCfg != nil {
		retry = *retryCfg
	}

	// Create resty client
	httpClient := resty.New().
		SetBaseURL(cfg.Endpoint).
		SetTimeout(timeout).
		SetHeader("X-User-Token", cfg.Token).
		SetHeader("Content-Type", "application/json").
		SetRetryCount(retry.MaxRetries).
		SetRetryWaitTime(retry.BaseDelay).
		SetRetryMaxWaitTime(retry.BaseDelay * 8). // Max wait time for exponential backoff
		AddRetryCondition(retryCondition)

	return &Client{
		endpoint:   cfg.Endpoint,
		token:      cfg.Token,
		timeout:    timeout,
		retry:      retry,
		query:      cfg.Query,
		httpClient: httpClient,
		logger:     logger.With().Str("component", "n9e-client").Logger(),
	}
}

// retryCondition determines whether a request should be retried.
// Only retry on timeout, 5xx errors, or connection failures.
// Do not retry on 4xx errors.
func retryCondition(resp *resty.Response, err error) bool {
	// Retry on error (timeout, connection failure, etc.)
	if err != nil {
		return true
	}

	// Retry on 5xx server errors
	if resp != nil && resp.StatusCode() >= 500 {
		return true
	}

	// Do not retry on 4xx client errors
	return false
}

// GetTargets retrieves all target hosts from the N9E API.
// It fetches all targets with a large limit to get all hosts in one request.
// If a query filter is configured, it will be applied to filter hosts.
func (c *Client) GetTargets(ctx context.Context) ([]TargetData, error) {
	c.logger.Debug().Str("query", c.query).Msg("fetching targets from N9E")

	var result TargetsResponse

	queryParams := map[string]string{
		"limit": "10000", // Large limit to get all hosts
		"p":     "1",
	}

	// Add query filter if configured
	if c.query != "" {
		queryParams["query"] = c.query
	}

	resp, err := c.httpClient.R().
		SetContext(ctx).
		SetResult(&result).
		SetQueryParams(queryParams).
		Get("/api/n9e/targets")

	if err != nil {
		c.logger.Error().Err(err).Msg("failed to fetch targets")
		return nil, fmt.Errorf("failed to fetch targets: %w", err)
	}

	// Check HTTP status code
	if resp.StatusCode() != http.StatusOK {
		c.logger.Error().
			Int("status_code", resp.StatusCode()).
			Str("body", string(resp.Body())).
			Msg("N9E API returned non-200 status")
		return nil, fmt.Errorf("N9E API returned status %d: %s", resp.StatusCode(), string(resp.Body()))
	}

	// Check N9E API error field
	if result.Err != "" {
		c.logger.Error().Str("api_error", result.Err).Msg("N9E API returned error")
		return nil, fmt.Errorf("N9E API error: %s", result.Err)
	}

	c.logger.Info().Int("count", len(result.Dat.List)).Int("total", result.Dat.Total).Msg("fetched targets successfully")
	return result.Dat.List, nil
}

// GetTarget retrieves a single target host by its ident from the N9E API.
func (c *Client) GetTarget(ctx context.Context, ident string) (*TargetData, error) {
	c.logger.Debug().Str("ident", ident).Msg("fetching target from N9E")

	var result TargetResponse

	resp, err := c.httpClient.R().
		SetContext(ctx).
		SetResult(&result).
		Get("/api/n9e/target/" + ident)

	if err != nil {
		c.logger.Error().Err(err).Str("ident", ident).Msg("failed to fetch target")
		return nil, fmt.Errorf("failed to fetch target %s: %w", ident, err)
	}

	// Check HTTP status code
	if resp.StatusCode() != http.StatusOK {
		c.logger.Error().
			Int("status_code", resp.StatusCode()).
			Str("ident", ident).
			Str("body", string(resp.Body())).
			Msg("N9E API returned non-200 status")
		return nil, fmt.Errorf("N9E API returned status %d for target %s: %s",
			resp.StatusCode(), ident, string(resp.Body()))
	}

	// Check N9E API error field
	if result.Err != "" {
		c.logger.Error().
			Str("ident", ident).
			Str("api_error", result.Err).
			Msg("N9E API returned error")
		return nil, fmt.Errorf("N9E API error for target %s: %s", ident, result.Err)
	}

	c.logger.Debug().Str("ident", ident).Msg("fetched target successfully")
	return &result.Dat, nil
}

// GetHostMetas retrieves all hosts and converts them to HostMeta models.
// This is a convenience method that combines GetTargets and ToHostMeta conversion.
func (c *Client) GetHostMetas(ctx context.Context) ([]*model.HostMeta, error) {
	c.logger.Debug().Msg("fetching host metas from N9E")

	targets, err := c.GetTargets(ctx)
	if err != nil {
		return nil, err
	}

	var hosts []*model.HostMeta
	var failedCount int

	for _, target := range targets {
		hostMeta, err := target.ToHostMeta()
		if err != nil {
			c.logger.Warn().
				Err(err).
				Str("ident", target.Ident).
				Msg("failed to convert target to host meta, skipping")
			failedCount++
			continue
		}
		hosts = append(hosts, hostMeta)
	}

	if failedCount > 0 {
		c.logger.Warn().
			Int("failed", failedCount).
			Int("success", len(hosts)).
			Msg("some targets failed to convert")
	}

	c.logger.Info().Int("count", len(hosts)).Msg("fetched host metas successfully")
	return hosts, nil
}

// GetHostMetaByIdent retrieves a single host and converts it to HostMeta model.
func (c *Client) GetHostMetaByIdent(ctx context.Context, ident string) (*model.HostMeta, error) {
	target, err := c.GetTarget(ctx, ident)
	if err != nil {
		return nil, err
	}

	return target.ToHostMeta()
}
