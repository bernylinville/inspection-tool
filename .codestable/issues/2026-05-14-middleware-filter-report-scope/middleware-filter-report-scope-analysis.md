---
doc_type: issue-analysis
issue: 2026-05-14-middleware-filter-report-scope
status: confirmed
root_cause_type: logic
related:
  - middleware-filter-report-scope-report.md
tags: [host-filter, middleware, report, victoriametrics]
created_at: "2026-05-14"
---

# Middleware Filter 报告范围根因分析

## 1. 问题定位

| 关键位置 | 说明 |
|---|---|
| `cmd/inspect/cmd/run.go:421` | MySQL Collector 只接收 `cfg.MySQL`，没有把全局 `cfg.Inspection.HostFilter` 传给中间件域。 |
| `cmd/inspect/cmd/run.go:436` | Redis Collector 同样只使用 `cfg.Redis.InstanceFilter`。 |
| `cmd/inspect/cmd/run.go:451` | Nginx Collector 同样只使用 `cfg.Nginx.InstanceFilter`。 |
| `cmd/inspect/cmd/run.go:466` | Tomcat Collector 同样只使用 `cfg.Tomcat.InstanceFilter`。 |
| `cmd/inspect/cmd/run.go:480` | Elasticsearch Collector 同样只使用 `cfg.Elasticsearch.InstanceFilter`。 |
| `internal/service/mysql_collector.go:82` | `buildInstanceFilter` 只读取 MySQL `instance_filter`，为空时返回 nil，后续 VM 查询不带业务范围。 |
| `internal/service/redis_collector.go:80` | Redis `buildInstanceFilter` 只读取 Redis `instance_filter`，为空时返回 nil。 |
| `internal/service/nginx_collector.go:77` | Nginx `buildInstanceFilter` 只读取 Nginx `instance_filter`，为空时返回 nil。 |
| `internal/service/tomcat_collector.go:82` | Tomcat `buildInstanceFilter` 只读取 Tomcat `instance_filter`，为空时返回 nil。 |
| `internal/service/elasticsearch_collector.go:42` | Elasticsearch `buildInstanceFilter` 只读取 ES `instance_filter`，为空时返回 nil。 |

## 2. 失败路径还原

**正常路径**：用户配置 `inspection.host_filter` 指定业务范围 → Host 与中间件 VM 查询都带同一业务范围 → 组合报告只展示该业务范围内的主机和实例。

**失败路径**：用户配置 `inspection.host_filter` → Host Collector 使用该过滤器并在 VM 查询后收敛 Host 列表 → 中间件 Collector 的 `instance_filter` 为空，VM 查询未注入任何业务标签 → 中间件实例发现得到全量指标 → 组合报告展示全量 MySQL / Redis / Tomcat 等实例。

**分叉点**：`internal/service/*_collector.go` 的 `buildInstanceFilter` 只看域级 `instance_filter`，没有继承全局 Host 过滤范围。

## 3. 根因

**根因类型**：logic

**根因描述**：系统存在两层过滤配置：`inspection.host_filter` 和各中间件域的 `instance_filter`。此前实现把 `host_filter` 视为 Host 专属，仅 Host Collector 使用；中间件域在 `instance_filter` 未配置业务组或标签时没有任何 PromQL 范围约束，因此报告范围退回到 VM 全量指标。上次 Host 修复只解决了 N9E 主机列表与 VM 指标范围不一致，未覆盖中间件域的过滤继承。

**是否有多个根因**：主根因单一。实际环境中还存在 VM 指标无 `busigroup` 标签但有 `items` 标签的情况，因此业务组 + 标签查询会返回 0，需要沿用 Host 修复中的 tag-only fallback，避免中间件继承全局范围后误变为空。

## 4. 影响面

- **影响范围**：组合报告和各中间件单域报告的数据范围。
- **潜在受害模块**：MySQL、Redis、Nginx、Tomcat、Elasticsearch Collector；Excel / HTML 组合报告输入。
- **数据完整性风险**：不会写外部数据，但会生成错误范围的报告。
- **严重程度复核**：维持 P1。

## 5. 修复方案

### 方案 A：在 CLI 创建每个 Collector 前手动拷贝 host_filter
- **做什么**：在 `cmd/inspect/cmd/run.go` 中把 `inspection.host_filter` 写入各域 `instance_filter`。
- **优点**：改动入口集中。
- **缺点 / 风险**：过滤语义藏在 CLI 层；测试和复用较弱；其他加载配置的调用路径可能漏掉。
- **影响面**：`cmd/inspect/cmd/run.go`。

### 方案 B：配置加载后把 host_filter 作为域级默认范围，并在中间件 VM 查询加 tag-only fallback
- **做什么**：在配置加载阶段把 `inspection.host_filter` 的业务组 / 标签继承到空的域级 `instance_filter`；保留域级地址、主机名、容器名等进一步收窄条件；中间件 VM 查询遇到业务组 + 标签 0 结果时重试标签-only。
- **优点**：配置语义统一；所有运行入口共享；适配当前 VM 标签实际情况；改动不需要更改 Collector 构造签名。
- **缺点 / 风险**：`host_filter` 从 Host 专属语义扩展为全局默认范围，需要文档说明。
- **影响面**：`internal/config/loader.go`、新增配置测试、`internal/service/*_collector.go`、文档注释。

### 方案 C：要求用户在每个中间件 instance_filter 重复配置 items 标签
- **做什么**：不改代码，修改 `config.yaml` 的 MySQL / Redis / Nginx / Tomcat / ES `instance_filter.tags`。
- **优点**：无代码风险。
- **缺点 / 风险**：配置重复且容易漏；与用户对全局业务范围的直觉不符；问题会在后续项目继续复现。
- **影响面**：用户配置。

### 推荐方案

**推荐方案 B**，理由：将全局业务范围变成所有巡检域的默认范围，域级过滤继续负责进一步收窄；同时复用 tag-only fallback 处理当前 VM 标签缺失 `busigroup` 的真实环境。
