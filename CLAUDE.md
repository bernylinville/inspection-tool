# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

基于监控数据的无侵入式系统巡检工具，采用 Monorepo 结构：`pkg/` 共享基础库，`apps/*` 提供应用。支持巡检类型：Host、MySQL、Redis、Nginx、Tomcat、Elasticsearch。通过调用夜莺（N9E）和 VictoriaMetrics API 查询监控数据，生成 Excel 和 HTML 格式的巡检报告。

**数据流**: Categraf → 夜莺 N9E → VictoriaMetrics → 本工具 → Excel/HTML 报告

## Tech Stack

- **Language**: Go 1.25
- **Workspace**: Go Workspace (go.work)
- **CLI**: cobra + viper
- **HTTP Client**: go-resty/resty/v2
- **Excel**: xuri/excelize/v2
- **Logging**: rs/zerolog
- **Concurrency**: golang.org/x/sync (errgroup)
- **Validation**: go-playground/validator/v10

## Build & Run Commands

```bash
# Build
make build              # Build CLI
make build-all          # Cross-compile (linux/darwin/windows)

# Test
make test-pkg           # Run pkg/* tests
make test-cli           # Run apps/inspect-cli tests

# Lint
make lint               # Run golangci-lint

# Run
./bin/inspect run -c config.yaml
./bin/inspect run -c config.yaml --mysql-only
./bin/inspect run -c config.yaml --redis-only
./bin/inspect validate -c config.yaml
./bin/inspect version
```

## Architecture

```text
CLI Layer (cobra)
    ↓
Service Layer (Inspector → Collector → Evaluator → Reporter)
    ↓
Client Layer (N9E Client, VM Client)
    ↓
Report Layer (ExcelWriter, HTMLWriter)
```

## Service Layer Pattern

以 `collector → evaluator → inspector` 的流水线组织核心逻辑：collector 聚合数据，evaluator 评估阈值，inspector 编排流程并驱动报告输出。

## Directory Structure

```text
go.work                        # Workspace root
pkg/                           # Shared library
├── config/                    # Config & validation
├── model/                     # Data models
├── n9e/                       # N9E client
└── vm/                        # VictoriaMetrics client
apps/
├── inspect-cli/               # CLI app
└── cmdb-server/               # CMDB server skeleton
configs/                       # Metric definitions
```

## Module Dependencies

`pkg/` 为基础库，`apps/*` 通过 go.work 引入并依赖 `pkg/`，禁止反向依赖。

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

<!-- CCA_WORKFLOW_POLICY -->
## CCA Workflow Policy

### Claude's Role (CRITICAL)
**Claude is the MANAGER, not the executor.**
- Plan and coordinate tasks
- Check role assignments before ANY action
- Delegate to appropriate executor (cask/oask/gask)
- NEVER execute file modifications directly

### Role-Based Delegation
| Role | Scope | Delegate |
|------|-------|----------|
| executor | Write/Edit, file-changing Bash | cask/oask |
| searcher | WebSearch/WebFetch | cask/oask/gask |
| git_manager | git add/commit/push/merge | cask/oask/gask |

### Commands
- codex: cask "task"
- opencode: oask "task"  
- gemini: gask "task"

### Rules (MANDATORY)
1. If role ≠ claude, delegate IMMEDIATELY without asking user
2. DO NOT ask user to choose execution method - auto-delegate based on role
3. DO NOT offer options like "委托给 Codex" or "直接执行"
4. Allowed direct: Read, Grep, Glob, .autoflow/**, /tmp/**
<!-- /CCA_WORKFLOW_POLICY -->
