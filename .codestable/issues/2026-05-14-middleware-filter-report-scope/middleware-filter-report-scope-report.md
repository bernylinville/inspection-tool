---
doc_type: issue-report
issue: 2026-05-14-middleware-filter-report-scope
status: confirmed
severity: P1
summary: 中间件报告未按 inspection.host_filter 收敛到指定业务范围
tags: [host-filter, middleware, report, victoriametrics]
created_at: "2026-05-14"
---

# Middleware Filter 报告范围 Issue Report

## 1. 问题现象

执行 `./bin/inspect run -c config.yaml` 后，Host 报告已按 `items=重庆传媒数字乡村-电信侧` 收敛，但 MySQL、Redis、Tomcat 等中间件报告仍出现全量实例。

## 2. 复现步骤

1. 使用当前本地 `config.yaml`。
2. 执行 `./bin/inspect run -c config.yaml`。
3. 查看终端统计和 `reports/inspection_report_*.html` / `reports/inspection_report_*.xlsx`。
4. 观察到：Host 总数为目标范围内 52 台，但 MySQL / Redis / Tomcat 仍按全量指标发现实例。

复现频率：稳定复现。

## 3. 期望 vs 实际

**期望行为**：配置 `inspection.host_filter` 后，组合报告中的 Host、MySQL、Redis、Nginx、Tomcat、Elasticsearch 都默认限制在同一业务范围；各域 `instance_filter` 只用于进一步收窄或显式覆盖域级范围。

**实际行为**：`inspection.host_filter` 只影响 Host 巡检；MySQL / Redis / Nginx / Tomcat / Elasticsearch 的 `instance_filter` 为空时直接查询全量 VM 指标，导致报告混入范围外实例。

## 4. 环境信息

- 涉及模块 / 功能：MySQL、Redis、Nginx、Tomcat、Elasticsearch 巡检与组合报告生成。
- 相关文件 / 函数：`cmd/inspect/cmd/run.go`、`internal/config/loader.go`、`internal/service/*_collector.go`。
- 运行环境：本地 dev，使用真实 `config.yaml` 数据源。
- 其他上下文：运行时不输出 token；报告目录为本地生成产物，不纳入提交。

## 5. 严重程度

**P1** — 巡检交付报告范围错误，会把不属于目标项目的中间件实例纳入报告，影响报告可信度。

## 备注

当前基线证据：Host 报告总数 52；同次运行 MySQL 54、Redis 96、Tomcat 9，均未继承 `inspection.host_filter` 的业务标签范围。
