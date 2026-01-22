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

### Sequential Thinking

Use **sequential-thinking** for complex problem-solving that requires structured reasoning:

<!-- CCA_WORKFLOW_POLICY -->
## CCA Workflow Policy

### Claude's Role (CRITICAL)
**Claude is the MANAGER, not the executor.**
- Plan and coordinate tasks
- Roles SSOT: `.autoflow/roles.json` (project-local only)
- Resolve roles before ANY action
- Route each action to the right delegate (primary mechanism)
- Treat hook blocking as a guardrail (backup), not the primary mechanism

### Non-Negotiables
- Do NOT use Write/Edit to modify repo files (delegate instead)
- Do NOT run repo-mutating Bash (redirect/tee/sed -i/rm/cp/mv into repo) (delegate instead)
- If the hook blocks something, follow its hint and delegate immediately

### Current Roles
- executor: codex+opencode (delegate)
- web_searcher: gemini (delegate)
- repo_searcher: codex (delegate) (enforced=true)
- git_manager: codex (delegate)

### Routing Cheatsheet (decide before every tool call)
- write/edit files or file-changing bash → executor: codex+opencode → MUST use cask (Codex will delegate to OpenCode via oask) (prefer /file-op)
- web search (WebSearch/WebFetch) → web_searcher: use gask "task"
- repo search (Grep/Glob or Bash rg/grep/git grep) → repo_searcher: use cask "task"
- large code analysis (>5 files) → use gask "analyze codebase..."
- git mutate (add/commit/push/merge/rebase/reset) → git_manager: use cask "task"
- read-only is OK directly: Read, and git status/log/diff/show (Grep/Glob is delegated)

### Output Contracts (MANDATORY, to save context)
- Repo search delegation must return: `keyFiles`, `hits` (path:line), `nextSteps` (no raw dumps)
- Executor delegation must return: `changedFiles`, `diffSummary`, `commands` (exit codes), `tests`, `notes/risks`
- Web search delegation must return: `conclusion`, `keyPoints`, `sources` (links/keywords)
- Review delegation must return: `correctness`, `risks`, `edgeCases`, `testSuggestions`

### Allowed Direct Operations (when role=claude)
- Read
- Read-only git: status/log/diff/show
- Write plans to ~/.claude/plans/** or .claude/plans/**
- Write to /tmp/**, .autoflow/**
<!-- /CCA_WORKFLOW_POLICY -->
