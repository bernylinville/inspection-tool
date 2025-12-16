# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

基于监控数据的无侵入式系统巡检工具。通过调用夜莺（N9E）和 VictoriaMetrics API 查询监控数据，生成 Excel 和 HTML 格式的巡检报告。

**数据流**: Categraf → 夜莺 N9E → VictoriaMetrics → 本工具 → Excel/HTML 报告

## Tech Stack

- **Language**: Go 1.25
- **CLI**: cobra + viper
- **HTTP Client**: go-resty/resty/v2
- **Excel**: xuri/excelize/v2
- **Logging**: rs/zerolog
- **Concurrency**: golang.org/x/sync (errgroup)
- **Validation**: go-playground/validator/v10

## Build & Run Commands

```bash
# Build
make build              # Build local binary
make build-all          # Cross-compile (linux/darwin/windows)

# Test
make test               # Run tests with race detection
go test -v ./internal/client/...  # Test specific package

# Lint
make lint               # Run golangci-lint

# Run
./bin/inspect run -c config.yaml
./bin/inspect run -c config.yaml --format excel,html --output ./reports
./bin/inspect validate -c config.yaml
./bin/inspect version
```

## Architecture

```text
CLI Layer (cobra)
    ↓
Service Layer (Inspector → Evaluator → Reporter)
    ↓
Client Layer (N9E Client, VM Client)
    ↓
Report Layer (ExcelWriter, HTMLWriter)
```

### Key Interfaces

```go
// Report writer - extensible for new formats
type ReportWriter interface {
    Write(result *InspectionResult, outputPath string) error
    Format() string
}

// Data collector
type Collector interface {
    CollectHostMeta(ctx context.Context, hosts []string) ([]HostMeta, error)
    CollectMetrics(ctx context.Context, hosts []string, metrics []MetricDef) ([]HostMetrics, error)
}

// Threshold evaluator
type Evaluator interface {
    Evaluate(metrics *HostMetrics, thresholds []Threshold) []Alert
}
```

## Directory Structure

```text
cmd/inspect/main.go           # Entry point
internal/
├── config/                   # Config loading & validation
├── client/
│   ├── n9e/                  # N9E API client (host metadata)
│   └── vm/                   # VictoriaMetrics client (metrics query)
├── model/                    # Data models (Host, Metric, Alert, InspectionResult)
├── service/                  # Business logic (Inspector, Collector, Evaluator)
└── report/
    ├── excel/                # Excel report with conditional formatting
    └── html/                 # HTML report with templates
configs/
├── config.example.yaml       # Main config template
└── metrics.yaml              # Metric definitions (PromQL queries)
```

## Data Sources

| Source          | Purpose                                 | Auth                |
| --------------- | --------------------------------------- | ------------------- |
| N9E API         | Host metadata (OS, version, kernel, IP) | X-User-Token header |
| VictoriaMetrics | Metrics via PromQL (/api/v1/query)      | None                |

## Key Metrics (PromQL)

```yaml
cpu_usage: cpu_usage_active{cpu="cpu-total"}
memory_usage: 100 - mem_available_percent
disk_usage: disk_used_percent
uptime: system_uptime
processes_zombies: processes_zombies
load_1m: system_load1
```

## Alert Thresholds

| Metric    | Warning | Critical |
| --------- | ------- | -------- |
| CPU       | >70%    | >90%     |
| Memory    | >70%    | >90%     |
| Disk      | >70%    | >90%     |
| Zombies   | >0      | >10      |
| Load/core | >0.7    | >1.0     |

## Concurrency Pattern

Worker pool with errgroup, max 20 concurrent workers for 100+ hosts parallel collection. Single host failure doesn't abort the entire inspection.

## MCP Tool Instructions

### Context7

Always use **context7** when code generation, setup or configuration steps, or library/API documentation is needed. Automatically use Context7 MCP tools to resolve library IDs and get library docs without explicit user request.

### Brave Search

Use **Brave Search MCP Server** for up-to-date information, current events, or fact-checking. Available capabilities include:

- Web search
- Local business search
- Image search
- Video search
- News search
- AI-powered summarization

### Sequential Thinking

Use **sequential-thinking** for complex problem-solving that requires structured reasoning:

- Break down complex problems into manageable steps
- Revise and refine thoughts as understanding deepens
- Branch into alternative paths of reasoning
- Adjust the total number of thoughts dynamically
- Generate and verify solution hypotheses

## 重要提示

- 写任何代码前必须完整阅读 @memory-bank/architecture.md（包含完整数据库结构）
- 写任何代码前必须完整阅读 @memory-bank/product-requirement-document.md
- 每完成一个重大功能或里程碑后，必须更新 @memory-bank/architecture.md
