package proxy

import (
	"context"
	"errors"

	"github.com/rs/zerolog"

	"inspection-tool/pkg/vm"
)

var (
	ErrQueryFailed = errors.New("query failed")
	ErrEmptyQuery  = errors.New("query cannot be empty")
)

type QueryRequest struct {
	Query string `json:"query" binding:"required"`
}

type QueryRangeRequest struct {
	Query string `json:"query" binding:"required"`
	Start int64  `json:"start" binding:"required"`
	End   int64  `json:"end" binding:"required"`
	Step  int64  `json:"step"`
}

type EChartsDataPoint struct {
	Timestamp int64   `json:"timestamp"`
	Value     float64 `json:"value"`
}

type EChartsSeries struct {
	Name   string             `json:"name"`
	Labels map[string]string  `json:"labels"`
	Data   []EChartsDataPoint `json:"data"`
}

type EChartsResponse struct {
	Series []EChartsSeries `json:"series"`
}

type MonitorProxy struct {
	vmClient *vm.Client
	logger   zerolog.Logger
}

func NewMonitorProxy(vmClient *vm.Client, logger zerolog.Logger) *MonitorProxy {
	return &MonitorProxy{
		vmClient: vmClient,
		logger:   logger.With().Str("component", "monitor-proxy").Logger(),
	}
}

func (p *MonitorProxy) Query(ctx context.Context, req *QueryRequest) (*vm.QueryResponse, error) {
	if req.Query == "" {
		return nil, ErrEmptyQuery
	}

	p.logger.Debug().Str("query", req.Query).Msg("executing instant query")

	resp, err := p.vmClient.Query(ctx, req.Query)
	if err != nil {
		p.logger.Error().Err(err).Str("query", req.Query).Msg("query failed")
		return nil, ErrQueryFailed
	}

	return resp, nil
}

func (p *MonitorProxy) QueryRange(ctx context.Context, req *QueryRangeRequest) (*vm.QueryResponse, error) {
	if req.Query == "" {
		return nil, ErrEmptyQuery
	}

	step := req.Step
	if step <= 0 {
		step = 60
	}

	p.logger.Debug().
		Str("query", req.Query).
		Int64("start", req.Start).
		Int64("end", req.End).
		Int64("step", step).
		Msg("executing range query")

	resp, err := p.vmClient.QueryRange(ctx, req.Query, req.Start, req.End, step)
	if err != nil {
		p.logger.Error().Err(err).Str("query", req.Query).Msg("range query failed")
		return nil, ErrQueryFailed
	}

	return resp, nil
}

func (p *MonitorProxy) QueryRangeForECharts(ctx context.Context, req *QueryRangeRequest) (*EChartsResponse, error) {
	resp, err := p.QueryRange(ctx, req)
	if err != nil {
		return nil, err
	}

	return p.convertToECharts(resp), nil
}

func (p *MonitorProxy) convertToECharts(resp *vm.QueryResponse) *EChartsResponse {
	result := &EChartsResponse{
		Series: make([]EChartsSeries, 0, len(resp.Data.Result)),
	}

	for _, sample := range resp.Data.Result {
		series := EChartsSeries{
			Name:   sample.Metric.Name(),
			Labels: sample.Metric,
			Data:   make([]EChartsDataPoint, 0, len(sample.Values)),
		}

		for _, v := range sample.Values {
			val, err := v.Value()
			if err != nil || v.IsNaN() {
				continue
			}
			series.Data = append(series.Data, EChartsDataPoint{
				Timestamp: v.TimestampUnix() * 1000,
				Value:     val,
			})
		}

		result.Series = append(result.Series, series)
	}

	return result
}
