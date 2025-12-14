// Package vm provides a client for VictoriaMetrics/Prometheus API.
package vm

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/rs/zerolog"

	"inspection-tool/internal/config"
)

// Client is a client for the VictoriaMetrics/Prometheus API.
type Client struct {
	endpoint   string             // API endpoint
	timeout    time.Duration      // Request timeout
	retry      config.RetryConfig // Retry configuration
	httpClient *resty.Client      // HTTP client
	logger     zerolog.Logger     // Logger
}

// NewClient creates a new VictoriaMetrics/Prometheus API client.
func NewClient(cfg *config.VictoriaMetricsConfig, retryCfg *config.RetryConfig, logger zerolog.Logger) *Client {
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
		SetHeader("Content-Type", "application/x-www-form-urlencoded").
		SetRetryCount(retry.MaxRetries).
		SetRetryWaitTime(retry.BaseDelay).
		SetRetryMaxWaitTime(retry.BaseDelay * 8). // Max wait time for exponential backoff
		AddRetryCondition(retryCondition)

	return &Client{
		endpoint:   cfg.Endpoint,
		timeout:    timeout,
		retry:      retry,
		httpClient: httpClient,
		logger:     logger.With().Str("component", "vm-client").Logger(),
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

// Query executes an instant query at the /api/v1/query endpoint.
// Returns the query response containing the result data.
func (c *Client) Query(ctx context.Context, query string) (*QueryResponse, error) {
	return c.QueryWithFilter(ctx, query, nil)
}

// QueryWithFilter executes an instant query with optional host filtering.
// The filter is applied by injecting label matchers into the query.
func (c *Client) QueryWithFilter(ctx context.Context, query string, filter *HostFilter) (*QueryResponse, error) {
	// Apply host filter to query if specified
	finalQuery := query
	if filter != nil && !filter.IsEmpty() {
		finalQuery = c.injectLabelMatchers(query, filter)
	}

	c.logger.Debug().
		Str("query", finalQuery).
		Msg("executing PromQL query")

	var result QueryResponse

	resp, err := c.httpClient.R().
		SetContext(ctx).
		SetQueryParam("query", finalQuery).
		SetResult(&result).
		Get("/api/v1/query")

	if err != nil {
		c.logger.Error().Err(err).Str("query", finalQuery).Msg("failed to execute query")
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}

	// Check HTTP status code
	if resp.StatusCode() != http.StatusOK {
		c.logger.Error().
			Int("status_code", resp.StatusCode()).
			Str("body", string(resp.Body())).
			Str("query", finalQuery).
			Msg("VM API returned non-200 status")
		return nil, fmt.Errorf("VM API returned status %d: %s", resp.StatusCode(), string(resp.Body()))
	}

	// Check for API-level errors
	if !result.IsSuccess() {
		c.logger.Error().
			Str("error_type", result.ErrorType).
			Str("error", result.Error).
			Str("query", finalQuery).
			Msg("VM API returned error")
		return nil, fmt.Errorf("VM API error [%s]: %s", result.ErrorType, result.Error)
	}

	// Log warnings if any
	if len(result.Warnings) > 0 {
		c.logger.Warn().
			Strs("warnings", result.Warnings).
			Str("query", finalQuery).
			Msg("VM API returned warnings")
	}

	c.logger.Debug().
		Str("result_type", result.Data.ResultType).
		Int("result_count", len(result.Data.Result)).
		Msg("query executed successfully")

	return &result, nil
}

// QueryResults executes an instant query and returns parsed results.
// This is a convenience method that combines Query and ParseQueryResults.
func (c *Client) QueryResults(ctx context.Context, query string) ([]QueryResult, error) {
	return c.QueryResultsWithFilter(ctx, query, nil)
}

// QueryResultsWithFilter executes an instant query with optional host filtering
// and returns parsed results.
func (c *Client) QueryResultsWithFilter(ctx context.Context, query string, filter *HostFilter) ([]QueryResult, error) {
	resp, err := c.QueryWithFilter(ctx, query, filter)
	if err != nil {
		return nil, err
	}

	return ParseQueryResults(resp)
}

// QueryByIdent executes a query and returns results grouped by host identifier.
// This is useful for getting metric values per host.
func (c *Client) QueryByIdent(ctx context.Context, query string) (map[string]QueryResult, error) {
	return c.QueryByIdentWithFilter(ctx, query, nil)
}

// QueryByIdentWithFilter executes a query with optional host filtering
// and returns results grouped by host identifier.
func (c *Client) QueryByIdentWithFilter(ctx context.Context, query string, filter *HostFilter) (map[string]QueryResult, error) {
	results, err := c.QueryResultsWithFilter(ctx, query, filter)
	if err != nil {
		return nil, err
	}

	return GroupResultsByIdent(results), nil
}

// injectLabelMatchers injects label matchers into a PromQL query based on the filter.
// Business groups are joined with OR (regex ~), tags are added with AND.
func (c *Client) injectLabelMatchers(query string, filter *HostFilter) string {
	if filter == nil || filter.IsEmpty() {
		return query
	}

	var matchers []string

	// Business groups - OR relation using regex
	if len(filter.BusinessGroups) > 0 {
		// Escape special regex characters in business group names
		var escapedGroups []string
		for _, group := range filter.BusinessGroups {
			escapedGroups = append(escapedGroups, escapeRegex(group))
		}
		groups := strings.Join(escapedGroups, "|")
		matchers = append(matchers, fmt.Sprintf(`busigroup=~"%s"`, groups))
	}

	// Tags - AND relation
	for k, v := range filter.Tags {
		// Escape quotes in tag values
		escapedValue := strings.ReplaceAll(v, `"`, `\"`)
		matchers = append(matchers, fmt.Sprintf(`%s="%s"`, k, escapedValue))
	}

	if len(matchers) == 0 {
		return query
	}

	// Inject matchers into query
	return injectMatchersToQuery(query, matchers)
}

// injectMatchersToQuery injects label matchers into a PromQL query.
// It handles queries with existing label selectors by appending to them.
// Only injects into valid metric names (must start with letter or underscore),
// not into scalar values like numbers.
func injectMatchersToQuery(query string, matchers []string) string {
	if len(matchers) == 0 {
		return query
	}

	matcherStr := strings.Join(matchers, ", ")

	// Pattern to match metric name with optional label selector
	// Metric names must start with [a-zA-Z_] followed by [a-zA-Z0-9_]*
	// This excludes pure numbers like 100 which are scalar values
	// Examples: cpu_usage_active, cpu_usage_active{cpu="cpu-total"}, mem_available_percent
	re := regexp.MustCompile(`([a-zA-Z_][a-zA-Z0-9_]*)(\{[^}]*\})?`)

	return re.ReplaceAllStringFunc(query, func(match string) string {
		// Find if there's an existing label selector
		braceIdx := strings.Index(match, "{")
		if braceIdx == -1 {
			// No existing labels, add new selector
			return match + "{" + matcherStr + "}"
		}

		// Has existing labels, append matchers before closing brace
		metricName := match[:braceIdx]
		existingLabels := match[braceIdx+1 : len(match)-1] // Remove { and }

		if existingLabels == "" {
			return metricName + "{" + matcherStr + "}"
		}
		return metricName + "{" + existingLabels + ", " + matcherStr + "}"
	})
}

// escapeRegex escapes special regex characters in a string.
func escapeRegex(s string) string {
	// Characters that need escaping in regex
	special := []string{"\\", ".", "+", "*", "?", "^", "$", "(", ")", "[", "]", "{", "}", "|"}
	result := s
	for _, char := range special {
		result = strings.ReplaceAll(result, char, "\\"+char)
	}
	return result
}
