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
make build              # Build local binary (static, -trimpath -s -w)
make build-all          # Cross-compile (linux/darwin/windows)
make test               # Run tests with race detection
make lint               # Run golangci-lint
make modernize          # 代码现代化自动修复
make fix                # modernize + test
make deps               # go mod tidy + verify
make tools              # Install golangci-lint, modernize

./bin/inspect run -c config.yaml
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
type ReportWriter interface {
    Write(result *InspectionResult, outputPath string) error
    Format() string
}

type Collector interface {
    CollectHostMeta(ctx context.Context, hosts []string) ([]HostMeta, error)
    CollectMetrics(ctx context.Context, hosts []string, metrics []MetricDef) ([]HostMetrics, error)
}

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
├── model/                    # Data models
├── service/                  # Business logic
└── report/
    ├── excel/                # Excel report
    └── html/                 # HTML report
```

## Concurrency Pattern

Worker pool with errgroup, max 20 concurrent workers. Single host failure doesn't abort the entire inspection.

---

## Go 1.25 编码规范

### 必须使用

```go
for i := range n { }                    // 替代 for i := 0; i < n; i++
for p := range strings.SplitSeq(s, ",") // 替代 strings.Split 循环
for line := range strings.Lines(s)      // 按行迭代
slices.Contains(items, x)               // 替代手写 contains
maps.Clone(m)                           // 替代手写 clone
cmp.Or(val, "default")                  // 替代 if val == "" 判断
slog.Info("msg", "key", val)            // 替代 log.Printf
wg.Go(func() { work() })                // 替代 wg.Add(1); go func(){defer wg.Done()...}
for b.Loop() { }                        // 基准测试替代 b.N 循环
runtime.AddCleanup(obj, fn, arg)        // 替代 SetFinalizer
```

### 禁止使用

| ❌ 禁止 | ✅ 替代 | 原因 |
|--------|--------|------|
| `interface{}` | `any` | Go 1.18+ 类型别名 |
| `ioutil.*` | `io.*` / `os.*` | 已废弃 |
| `strings.Split` 循环 | `strings.SplitSeq` | 按需迭代，减少内存分配 |
| `log.Printf` | `slog` / `zerolog` | 结构化日志 |
| `json:",omitempty"` (time.Time) | `json:",omitzero"` | Go 1.24+ 零值处理 |
| `fmt.Sprintf("%s:%d", host, port)` | `net.JoinHostPort` | 正确处理 IPv6 |
| `time.Sleep` | `time.Ticker` / Context | 避免不可控的阻塞 |
| `panic` | `return error` | CLI 工具应优雅退出 |
| `fmt.Print` (业务逻辑中) | `zerolog` / `slog` | 保持日志格式统一 |
| `excelize.NewFile()` 不保存 | `f.SaveAs()` / `defer f.Close()` | 防止文件句柄泄露 |

---

## Testing Guidelines

- **Unit Tests**: Place in the same package as `_test.go`. Use `t.Parallel()` where possible.
- **Mocking**: Do not use heavyweight mock frameworks. Define interfaces locally (consumer-driven) and create simple struct mocks manually.
- **Integration Tests**: Place in `test/` directory or mark with `//go:build integration`.
- **Table Driven**: Prefer table-driven tests for logic with multiple edge cases.

---

## Error Handling

- Use `fmt.Errorf("context: %w", err)` for wrapping errors.
- Use `errors.Join(errs...)` when aggregating multiple errors (e.g., in loops or defer).
- Avoid checking `err != nil` repeatedly if a functional option pattern or error accumulator struct can be used.

---

## Git Convention

Follow Conventional Commits:

- `feat: allow custom thresholds`
- `fix: crash on empty n9e response`
- `chore: update dependencies`
- `docs: update architecture diagram`

---

## MCP Tool Instructions

- Always use **mcp__context7** when code generation, setup or configuration steps, or library/API documentation is needed.
- Use **mcp__brave-search** for up-to-date information, current events, or fact-checking.
- Use **mcp__sequential-thinking** for complex problem-solving that requires structured reasoning.

## 重要提示

- 写任何代码前必须完整阅读 @memory-bank/architecture.md
- 写任何代码前必须完整阅读 @memory-bank/product-requirement-document.md
- 每完成一个重大功能或里程碑后，必须更新 @memory-bank/architecture.md
