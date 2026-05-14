---
doc_type: issue-report
issue: 2026-05-14-host-filter-report-scope
status: confirmed
severity: P1
summary: inspect run 使用 host_filter 后报告仍包含夜莺全量主机
tags: [host-filter, report, n9e, victoriametrics]
created_at: "2026-05-14"
---

# Host Filter 报告范围 Issue Report

## 1. 问题现象

执行 `./bin/inspect run -c config.yaml` 后，配置中指定了 `items=重庆传媒数字乡村-电信侧` 范围，但报告主机明细包含夜莺返回的全量主机。

## 2. 复现步骤

1. 使用当前本地 `config.yaml`。
2. 执行 `./bin/inspect run -c config.yaml`。
3. 查看终端统计和 `reports/inspection_report_*.html` / `reports/inspection_report_*.xlsx`。
4. 观察到：主机报告范围没有按 `inspection.host_filter.tags.items` 收敛。

复现频率：稳定复现。

## 3. 期望 vs 实际

**期望行为**：报告主机列表只包含 `items=重庆传媒数字乡村-电信侧` 对应主机。

**实际行为**：N9E 返回 764 台主机，报告把这些 N9E 主机都纳入主机结果，未命中的主机被标为采集失败。

## 4. 环境信息

- 涉及模块 / 功能：Host 巡检、报告生成。
- 相关文件 / 函数：`internal/service/collector.go`、`internal/report/excel/writer.go`。
- 运行环境：本地 dev，使用真实 `config.yaml` 数据源。
- 其他上下文：运行时不输出 token / 生产地址。

## 5. 严重程度

**P1** — 巡检范围错误会让交付报告包含不属于目标项目的全量机器，影响报告可信度。

## 备注

复现日志关键证据：修复前 N9E 返回 764 台，主机报告总数 764；修复后主机报告总数 52。
