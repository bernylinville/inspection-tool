---
doc_type: issue-fix
issue: 2026-05-14-middleware-filter-report-scope
status: completed
severity: P1
tags: [host-filter, middleware, report, victoriametrics]
created_at: "2026-05-14"
related:
  - middleware-filter-report-scope-report.md
  - middleware-filter-report-scope-analysis.md
---

# Middleware Filter 报告范围修复记录

## 修复摘要

修复 MySQL、Redis、Nginx、Tomcat、Elasticsearch 等中间件域未继承 `inspection.host_filter`，导致组合报告出现全量实例的问题。修复后，`inspection.host_filter` 会作为中间件域的默认 VM 标签范围；域级 `instance_filter` 仍可通过地址、主机名、容器名、节点名或显式业务标签进一步收窄。

同时把 Host 修复中验证过的 tag-only fallback 扩展到中间件 VM 查询：当业务组 + 标签查询在 VM 中无结果、但配置包含标签时，自动用标签-only 重试，适配当前指标存在 `items` 标签但缺少 `busigroup` 标签的环境。

## 代码改动

- `internal/config/loader.go`
  - 配置加载和校验后调用全局范围继承逻辑。
- `internal/config/filter_scope.go`
  - 新增 `inspection.host_filter` 到 MySQL / Redis / Nginx / Tomcat / Elasticsearch `instance_filter` 的默认继承逻辑。
  - 保留域级地址 / 主机名 / 容器名 / 节点名规则；域级显式业务组不被覆盖；域级标签与全局标签合并，域级同名标签优先。
- `internal/config/filter_scope_test.go`
  - 覆盖全局范围继承到各中间件域。
  - 覆盖域级显式业务组 / 标签不被覆盖的边界。
- `internal/service/vm_filter.go`
  - 新增中间件 VM 查询 tag-only fallback helper。
- `internal/service/*_collector.go`
  - MySQL、Redis、Nginx、Tomcat、Elasticsearch 的发现、指标和辅助查询统一使用 fallback helper。
- `internal/service/vm_filter_test.go`
  - 覆盖业务组 + 标签无结果时重试标签-only，并确保 fallback 不再带 `busigroup`。
- `configs/config.example.yaml`、`README.md`
  - 补充 `inspection.host_filter` 作为中间件默认范围的说明。

## 验证

- `env -u GOROOT go test ./internal/config ./internal/service -count=1`：通过。
- `env -u GOROOT go test ./...`：通过。
- `env -u GOROOT make build`：通过，生成 `bin/inspect`。
- `./bin/inspect run -c config.yaml`：已复跑，报告生成成功。
  - Host：52（修复前 Host 已由上次 issue 收敛为 52）。
  - MySQL：3（修复前为 54）。
  - Redis：0（修复前为 96）。
  - Nginx：0（保持目标范围内无实例）。
  - Tomcat：0（修复前为 9）。
  - Excel 报告：`reports/inspection_report_2026-05-14.xlsx` 生成成功；`MySQL 巡检` 工作表 3 条数据，`Redis 巡检` / `Nginx 巡检` 无数据行。
  - HTML 报告：`reports/inspection_report_2026-05-14.html` 生成成功；概览显示 Host 52、MySQL 3、Redis 0、Nginx 0、Tomcat 0。

## 剩余风险

- 当前命令退出码仍为 2，是目标范围内实际巡检存在 critical 项导致，不是报告范围或生成失败。
- 如果某个域显式配置了与全局 `inspection.host_filter` 同名但不同值的标签，域级标签优先；这是为了避免覆盖用户明确配置，但也意味着该域可主动偏离全局范围。
- Excel 当前未生成 Tomcat 工作表属于既有报告支持范围问题，本次只验证 HTML Tomcat 结果已收敛；如需补齐 Excel Tomcat，可另开 issue / feature。
