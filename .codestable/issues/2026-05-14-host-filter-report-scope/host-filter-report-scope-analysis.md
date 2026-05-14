---
doc_type: issue-analysis
issue: 2026-05-14-host-filter-report-scope
status: confirmed
root_cause_type: logic
tags: [host-filter, report, n9e, victoriametrics]
created_at: "2026-05-14"
related:
  - host-filter-report-scope-report.md
---

# Host Filter 报告范围根因分析

## 1. 问题定位

| 关键位置 | 说明 |
|---|---|
| `internal/service/collector.go:102` | `CollectAll` 先从 N9E 获取主机元信息，再调用 VM 采集指标。 |
| `internal/service/collector.go:110` | `CollectMetrics` 对 VM 查询应用 `host_filter`，但没有反向收敛 N9E 主机列表。 |
| `internal/service/collector.go:120` | 失败主机判断遍历的是 N9E 主机列表，导致过滤范围外主机进入报告并被标为 failed。 |
| `internal/client/vm/client.go:81` | `QueryWithFilter` 只把筛选条件注入 PromQL，不影响 N9E `/api/n9e/targets` 返回范围。 |
| `internal/report/excel/writer.go:1585` | Redis 多集群 sheet 直接使用完整 cluster ID，遇到长域名 ID 会超过 Excel 31 字符限制。 |

## 2. 失败路径还原

**正常路径**：用户在 `config.yaml` 配置 `inspection.host_filter` → VM 查询只返回目标范围指标 → 报告只展示目标范围主机。

**失败路径**：用户配置 `inspection.host_filter` → VM 查询被过滤，但 N9E 仍返回全量 target → `CollectAll` 用全量 N9E 主机生成 HostResult → 未匹配到指标的范围外主机被标为 failed → 报告包含全量机器。

**分叉点**：`internal/service/collector.go:110` — VM 指标范围和 N9E 元信息范围没有对齐。

## 3. 根因

**根因类型**：logic

**根因描述**：`inspection.host_filter` 是 PromQL 级过滤，只作用于 VM 查询。Host 报告的数据模型同时依赖 N9E 主机元信息和 VM 指标结果，但旧逻辑没有在 VM 查询后用实际指标结果收敛 N9E 主机列表，因此报告主机范围以 N9E 全量 target 为准。

**是否有多个根因**：主问题单一。验证过程中另发现 Excel Redis 多集群 sheet 名未处理 31 字符限制，导致报告生成阻断，作为同次报告闭环一并修复。

## 4. 影响面

- **影响范围**：Host 巡检报告范围；组合 HTML / Excel 都会受到 HostResult 输入影响。
- **潜在受害模块**：Excel / HTML 报告输出、命令退出码统计、主机异常汇总。
- **数据完整性风险**：不会修改外部数据，但会生成错误范围的交付报告。
- **严重程度复核**：维持 P1。

## 5. 修复方案

### 方案 A：在 N9E 查询层强制拼接过滤条件
- **做什么**：把 `inspection.host_filter` 转成 N9E `query` 参数。
- **优点**：源头收敛主机元信息。
- **缺点 / 风险**：N9E business group / tag 查询语法与 VM label 不完全等价，容易误转。
- **影响面**：N9E client、配置语义、文档。

### 方案 B：用 VM 实际返回的指标范围收敛 N9E 主机列表
- **做什么**：VM 过滤查询完成后，只保留有非 N/A 指标的主机进入 HostResult；当前环境中 business group label 未命中时，保留 tag-only fallback 以尊重 `items=重庆传媒数字乡村-电信侧`。
- **优点**：改动集中在 Collector；与现有 PromQL 过滤模型一致；直接修复报告范围泄漏。
- **缺点 / 风险**：如果目标范围内所有主机都没有任何活跃指标，PromQL 级过滤无法从 VM 反推出应报告的主机清单。
- **影响面**：`internal/service/collector.go`、Collector 单元测试。

### 方案 C：要求用户改用 `datasources.n9e.query`
- **做什么**：不改代码，只要求配置使用 N9E query。
- **优点**：无代码风险。
- **缺点 / 风险**：现有 `inspection.host_filter` 文档承诺可用于主机筛选，问题会继续复现。
- **影响面**：用户配置和文档。

### 推荐方案

**推荐方案 B**，理由：最小代码改动，直接保证报告输入范围与 VM 过滤后的实际巡检数据一致；当前用户指定的 `items=重庆传媒数字乡村-电信侧` 可以被验证为 52 台主机。
