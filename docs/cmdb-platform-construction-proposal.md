# CMDB 管理平台建设方案

> 创建日期：2026-01-11  
> 版本：v3.0（最终版）  
> 基于：巡检工具 v0.5.0（Host + MySQL + Redis + Nginx + Tomcat + ES 巡检功能）

---

## 目录

1. [执行摘要](#1-执行摘要)
2. [项目现状分析](#2-项目现状分析)
3. [平台定位与边界](#3-平台定位与边界)
4. [技术架构设计](#4-技术架构设计)
5. [代码复用方案](#5-代码复用方案)
6. [核心功能设计](#6-核心功能设计)
7. [外部系统集成](#7-外部系统集成)
8. [数据库设计](#8-数据库设计)
9. [技术栈选型](#9-技术栈选型)
10. [项目实施路线图](#10-项目实施路线图)
11. [安全与权限设计](#11-安全与权限设计)
12. [部署方案](#12-部署方案)
13. [总结](#13-总结)

---

## 1. 执行摘要

### 1.1 核心结论

| 维度         | 结论                                                 |
| ------------ | ---------------------------------------------------- |
| **项目定位** | CMDB 资产管理平台（不是监控平台）                    |
| **可行性**   | ✅ 完全可行，可复用 80%+ 现有巡检工具核心代码        |
| **代码管理** | Monorepo + Go Workspace                              |
| **技术栈**   | Go (Gin) + Vue (Vben Admin) + PostgreSQL             |
| **外部集成** | N9E、VictoriaMetrics、FlashDuty（均为 **只读调用**） |
| **预计工期** | 6-8 周（单人开发）                                   |

### 1.2 关键设计决策

| 决策                  | 理由                                         |
| --------------------- | -------------------------------------------- |
| **不需要 ClickHouse** | 历史数据直接查 VictoriaMetrics，无需额外存储 |
| **FlashDuty 只读**    | 仅调用 API 获取告警数据，不推送告警          |
| **监控数据不存储**    | 透传 VM 查询，实时/历史数据都从 VM 获取      |
| **N9E 只读**          | 仅获取主机元信息，不写入配置                 |

### 1.3 现有可复用资产

| 模块               | 复用价值   | 复用方式                  |
| ------------------ | ---------- | ------------------------- |
| N9E Client         | ⭐⭐⭐⭐⭐ | 直接引用 - 主机元信息获取 |
| VM Client          | ⭐⭐⭐⭐⭐ | 直接引用 - 监控数据查询   |
| Model 层           | ⭐⭐⭐⭐   | 参考复用 - 数据模型基础   |
| Evaluator          | ⭐⭐⭐⭐   | 参考复用 - 健康评分逻辑   |
| Collector 并发模式 | ⭐⭐⭐⭐   | 参考复用 - errgroup 模式  |

---

## 2. 项目现状分析

### 2.1 巡检工具能力全景

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                        巡检工具 (inspection-tool) 现状                            │
├─────────────────────────────────────────────────────────────────────────────────┤
│                                                                                 │
│  ┌─────────────────────────────────────────────────────────────────────────┐   │
│  │  已实现功能 ✅                                                           │   │
│  │                                                                          │   │
│  │  • Host 巡检：CPU、内存、磁盘、负载、进程、Uptime                         │   │
│  │  • MySQL 巡检：MGR 集群、主从、连接数、Binlog、慢查询                     │   │
│  │  • Redis 巡检：Cluster 模式、角色识别、复制延迟                           │   │
│  │  • Nginx 巡检：连接状态、请求处理                                         │   │
│  │  • Tomcat 巡检：JVM、线程池、连接器                                       │   │
│  │  • Elasticsearch 巡检：集群状态、节点、分片                               │   │
│  │  • 报告输出：Excel（多 Sheet + 条件格式）、HTML（响应式）                 │   │
│  └─────────────────────────────────────────────────────────────────────────┘   │
│                                                                                 │
│  ┌─────────────────────────────────────────────────────────────────────────┐   │
│  │  技术实现亮点                                                            │   │
│  │                                                                          │   │
│  │  • 并发采集：errgroup + worker pool（默认 20 并发）                       │   │
│  │  • 接口设计：HostCollector、HostEvaluator、ReportWriter                  │   │
│  │  • 泛型接口：InstanceDiscoverer[T] 支持多种中间件扩展                     │   │
│  │  • 标签注入：PromQL 动态注入业务组/标签过滤                               │   │
│  │  • 测试覆盖：整体覆盖率 85%+                                              │   │
│  └─────────────────────────────────────────────────────────────────────────┘   │
│                                                                                 │
└─────────────────────────────────────────────────────────────────────────────────┘
```

### 2.2 数据流现状

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│    Categraf     │────▶│   夜莺 (N9E)    │────▶│ VictoriaMetrics │
│   (数据采集)     │     │   (监控平台)     │     │   (时序数据库)   │
│ Host+MySQL+Redis│     │                 │     │                 │
│ +Nginx+Tomcat+ES│     │                 │     │                 │
└─────────────────┘     └─────────────────┘     └─────────────────┘
                               │                        │
                               │ GET 主机元信息          │ GET 指标数据
                               │ (X-User-Token)         │ (PromQL)
                               ▼                        ▼
                        ┌─────────────────────────────────────┐
                        │           巡检工具 CLI                │
                        │  ┌──────────┐ ┌──────────┐          │
                        │  │ Collector│ │ Evaluator│          │
                        │  └──────────┘ └──────────┘          │
                        │             ↓                       │
                        │       ┌───────────────┐             │
                        │       │   Reporter    │             │
                        │       └───────────────┘             │
                        └─────────────────────────────────────┘
                                        │
                        ┌───────────────┴───────────────┐
                        ▼                               ▼
                 ┌─────────────┐                ┌─────────────┐
                 │  Excel 报告  │                │  HTML 报告   │
                 └─────────────┘                └─────────────┘
```

### 2.3 核心代码结构

```
inspection-tool/
├── cmd/inspect/                  # CLI 入口（Cobra）
│   ├── main.go
│   └── cmd/
│       └── root.go               # run/validate/version 命令
├── internal/
│   ├── client/
│   │   ├── n9e/                  # N9E Client（主机元信息）
│   │   │   ├── client.go         # GetHostMetas(), GetTarget()
│   │   │   └── types.go          # TargetData, ExtendInfo
│   │   └── vm/                   # VM Client（PromQL 查询）
│   │       ├── client.go         # Query(), QueryRange()
│   │       └── types.go          # QueryResponse, Sample
│   ├── config/                   # 配置管理（Viper）
│   ├── model/                    # 数据模型
│   │   ├── host.go               # HostMeta, HostResult
│   │   ├── metric.go             # MetricValue, MetricDefinition
│   │   ├── alert.go              # Alert, AlertLevel
│   │   ├── mysql.go              # MySQLInstance, MySQLAlert
│   │   ├── redis.go              # RedisInstance
│   │   └── elasticsearch.go      # ESCluster
│   ├── service/
│   │   ├── interfaces.go         # 核心接口定义
│   │   ├── collector.go          # Host 数据采集（errgroup）
│   │   ├── evaluator.go          # 阈值评估
│   │   ├── inspector.go          # 巡检编排
│   │   ├── mysql_*.go            # MySQL 巡检实现
│   │   ├── redis_*.go            # Redis 巡检实现
│   │   └── ...                   # 其他中间件
│   └── report/
│       ├── writer.go             # ReportWriter 接口
│       ├── excel/                # Excel 生成器
│       └── html/                 # HTML 生成器
└── configs/
    ├── config.example.yaml       # 主配置模板
    ├── metrics.yaml              # Host 指标定义
    ├── mysql-metrics.yaml        # MySQL 指标定义
    └── ...                       # 其他指标配置
```

---

## 3. 平台定位与边界

### 3.1 核心定位

**CMDB 管理平台是资产管理和数据聚合展示平台，不是监控/告警平台。**

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           平台职责边界                                       │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌───────────────────────────────────────────────────────────────────┐     │
│  │  CMDB 平台职责 ✅                                                  │     │
│  │                                                                    │     │
│  │  • 资产管理：主机、MySQL、Redis、Nginx、Tomcat、ES 等资产的 CRUD   │     │
│  │  • 数据聚合：调用外部 API，聚合展示监控/告警数据（不存储）          │     │
│  │  • 巡检管理：触发巡检、查看历史报告                                 │     │
│  │  • 业务拓扑：项目 → 应用 → 资产 的关联管理                          │     │
│  │  • 权限控制：用户、角色、权限管理                                   │     │
│  └───────────────────────────────────────────────────────────────────┘     │
│                                                                             │
│  ┌───────────────────────────────────────────────────────────────────┐     │
│  │  不属于 CMDB 平台职责 ❌                                           │     │
│  │                                                                    │     │
│  │  • 监控数据采集/存储 → Categraf + VictoriaMetrics                  │     │
│  │  • 告警规则配置/触发/推送 → N9E + FlashDuty                        │     │
│  │  • 历史监控数据存储 → VictoriaMetrics（已存储，直接查）             │     │
│  │  • 告警数据存储 → FlashDuty（已存储，直接查）                       │     │
│  └───────────────────────────────────────────────────────────────────┘     │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 3.2 与外部系统的关系

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         系统关系图                                           │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│                        ┌─────────────────┐                                  │
│                        │  CMDB 管理平台   │                                  │
│                        │  (本项目)        │                                  │
│                        └────────┬────────┘                                  │
│                                 │                                           │
│              ┌──────────────────┼──────────────────┐                        │
│              │                  │                  │                        │
│              ▼                  ▼                  ▼                        │
│     ┌────────────────┐ ┌────────────────┐ ┌────────────────┐               │
│     │     N9E        │ │ VictoriaMetrics│ │   FlashDuty    │               │
│     │  (监控平台)     │ │  (时序数据库)   │ │  (告警平台)     │               │
│     ├────────────────┤ ├────────────────┤ ├────────────────┤               │
│     │ 调用方式:       │ │ 调用方式:       │ │ 调用方式:       │               │
│     │ • GET 主机列表  │ │ • GET 实时指标  │ │ • GET 告警列表  │               │
│     │ • GET 主机详情  │ │ • GET 历史趋势  │ │ • GET 告警详情  │               │
│     │                │ │                │ │ • GET 故障列表  │               │
│     │ 只读 ✅        │ │ 只读 ✅        │ │ 只读 ✅        │               │
│     │ 不写入 ❌      │ │ 不写入 ❌      │ │ 不推送 ❌      │               │
│     └────────────────┘ └────────────────┘ └────────────────┘               │
│                                                                             │
│     说明：                                                                   │
│     • N9E 负责告警规则，直接推送到 FlashDuty                                 │
│     • CMDB 只查看 FlashDuty 中的告警，不参与告警推送                         │
│     • VictoriaMetrics 存储所有历史监控数据，CMDB 直接查询                    │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 3.3 功能优先级

| 功能模块         | 优先级 | 说明                                         |
| ---------------- | ------ | -------------------------------------------- |
| **资产管理**     | P0     | 主机、MySQL、Redis、Nginx、Tomcat CRUD       |
| **资产同步**     | P0     | 从 N9E 同步主机，从 VM 发现中间件实例        |
| **实时监控展示** | P0     | 调用 VM API 展示实时指标（透传，不存储）     |
| **历史趋势展示** | P0     | 调用 VM API 展示历史趋势（透传，不存储）     |
| **告警展示**     | P0     | 调用 FlashDuty API 展示告警列表/详情（只读） |
| **巡检管理**     | P0     | Web 触发巡检、查看历史报告                   |
| **用户权限**     | P0     | RBAC 权限控制                                |
| **业务拓扑**     | P1     | 项目 → 应用 → 资产关联                       |
| **健康评分**     | P1     | 基于指标和告警的健康评分                     |
| **大盘展示**     | P2     | 汇总展示，向上汇报                           |

---

## 4. 技术架构设计

### 4.1 整体架构

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           CMDB 平台整体架构                                  │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                         前端层                                       │   │
│  │  ┌─────────────────────────────────────────────────────────────┐    │   │
│  │  │              Vue Vben Admin (Vue 3 + TypeScript)             │    │   │
│  │  │  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌────────┐ │    │   │
│  │  │  │  资产   │ │  监控   │ │  告警   │ │  巡检   │ │  系统  │ │    │   │
│  │  │  │  管理   │ │  展示   │ │  展示   │ │  管理   │ │  设置  │ │    │   │
│  │  │  └─────────┘ └─────────┘ └─────────┘ └─────────┘ └────────┘ │    │   │
│  │  └─────────────────────────────────────────────────────────────┘    │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                       │                                     │
│                                       ▼                                     │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                         后端层                                       │   │
│  │  ┌─────────────────────────────────────────────────────────────┐    │   │
│  │  │                   Go API Server (Gin)                        │    │   │
│  │  │                                                              │    │   │
│  │  │  ┌──────────────────────────────────────────────────────┐   │    │   │
│  │  │  │                    业务模块                           │   │    │   │
│  │  │  │  • AssetService    - 资产 CRUD                        │   │    │   │
│  │  │  │  • SyncService     - 资产同步（N9E/VM）               │   │    │   │
│  │  │  │  • MonitorProxy    - 监控数据透传（VM）               │   │    │   │
│  │  │  │  • AlertProxy      - 告警数据透传（FlashDuty 只读）   │   │    │   │
│  │  │  │  • InspectService  - 巡检触发/历史                    │   │    │   │
│  │  │  └──────────────────────────────────────────────────────┘   │    │   │
│  │  │                                                              │    │   │
│  │  │  ┌──────────────────────────────────────────────────────┐   │    │   │
│  │  │  │              复用层 (pkg/) - 来自巡检工具              │   │    │   │
│  │  │  │  • N9E Client   - 主机元信息                          │   │    │   │
│  │  │  │  • VM Client    - PromQL 查询（实时+历史）            │   │    │   │
│  │  │  │  • Evaluator    - 健康评分                            │   │    │   │
│  │  │  └──────────────────────────────────────────────────────┘   │    │   │
│  │  └─────────────────────────────────────────────────────────────┘    │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                       │                                     │
│           ┌───────────────────────────┼───────────────────────────┐         │
│           ▼                           ▼                           ▼         │
│  ┌────────────────┐         ┌────────────────┐         ┌────────────────┐  │
│  │  PostgreSQL    │         │ 外部系统(只读)  │         │    Redis       │  │
│  │  • 资产数据    │         │ • N9E API      │         │  • 缓存        │  │
│  │  • 用户权限    │         │ • VM API       │         │  • 会话        │  │
│  │  • 巡检记录    │         │ • FlashDuty API│         │                │  │
│  └────────────────┘         └────────────────┘         └────────────────┘  │
│                                                                             │
│  注意：没有 ClickHouse，监控历史数据直接查 VictoriaMetrics                   │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 4.2 数据流设计

#### 监控数据流（透传 VictoriaMetrics）

```
┌────────────────────────────────────────────────────────────────────────────┐
│                     监控数据流 - 透传 VictoriaMetrics                       │
├────────────────────────────────────────────────────────────────────────────┤
│                                                                            │
│  实时监控:                                                                  │
│  ┌─────────┐    ┌─────────────┐    ┌─────────────────┐    ┌─────────┐     │
│  │  前端   │───▶│ CMDB 后端   │───▶│ VictoriaMetrics │───▶│  前端   │     │
│  │  请求   │    │ MonitorProxy│    │ /api/v1/query   │    │ ECharts │     │
│  └─────────┘    └─────────────┘    └─────────────────┘    └─────────┘     │
│                                                                            │
│  历史趋势:                                                                  │
│  ┌─────────┐    ┌─────────────┐    ┌─────────────────┐    ┌─────────┐     │
│  │  前端   │───▶│ CMDB 后端   │───▶│ VictoriaMetrics │───▶│  前端   │     │
│  │  请求   │    │ MonitorProxy│    │/api/v1/query_range│  │ ECharts │     │
│  └─────────┘    └─────────────┘    └─────────────────┘    └─────────┘     │
│                                                                            │
│  关键点：                                                                   │
│  • 后端只做 Proxy + 格式化，不存储监控数据                                  │
│  • VictoriaMetrics 本身存储历史数据，无需 ClickHouse                        │
│  • 实时数据：/api/v1/query                                                  │
│  • 历史数据：/api/v1/query_range                                            │
│                                                                            │
└────────────────────────────────────────────────────────────────────────────┘
```

#### 告警数据流（只读 FlashDuty）

```
┌────────────────────────────────────────────────────────────────────────────┐
│                     告警数据流 - 只读 FlashDuty                             │
├────────────────────────────────────────────────────────────────────────────┤
│                                                                            │
│  告警推送（CMDB 不参与）:                                                   │
│  ┌─────────┐    ┌─────────┐    ┌─────────────┐                             │
│  │  N9E    │───▶│  告警   │───▶│  FlashDuty  │ → 通知值班人员              │
│  │ 告警规则 │    │  触发   │    │  (存储告警)  │                             │
│  └─────────┘    └─────────┘    └─────────────┘                             │
│                                                                            │
│  告警查看（CMDB 只读）:                                                     │
│  ┌─────────┐    ┌─────────────┐    ┌─────────────┐    ┌─────────┐         │
│  │  前端   │───▶│ CMDB 后端   │───▶│  FlashDuty  │───▶│  前端   │         │
│  │  请求   │    │ AlertProxy  │    │  Open API   │    │  展示   │         │
│  └─────────┘    └─────────────┘    └─────────────┘    └─────────┘         │
│                                                                            │
│  关键点：                                                                   │
│  • N9E 直接对接 FlashDuty，负责告警推送                                     │
│  • CMDB 只调用 FlashDuty Open API 查看告警                                  │
│  • 不需要 Webhook，不需要推送告警                                           │
│                                                                            │
└────────────────────────────────────────────────────────────────────────────┘
```

---

## 5. 代码复用方案

### 5.1 Monorepo 目录结构

```
my-internal-platform/                   # Git Root
├── go.work                             # Go Workspace
├── go.work.sum
│
├── pkg/                                # 公共库（从 inspection-tool 抽取）
│   ├── go.mod
│   ├── n9e/                            # N9E Client（直接复用）
│   │   ├── client.go
│   │   └── types.go
│   ├── vm/                             # VM Client（直接复用）
│   │   ├── client.go
│   │   └── types.go
│   ├── model/                          # 通用模型
│   └── evaluator/                      # 健康评分
│
├── apps/
│   ├── cli-tool/                       # 原巡检工具 CLI
│   │   ├── go.mod
│   │   ├── main.go
│   │   └── cmd/
│   │
│   └── cmdb-server/                    # CMDB 后端
│       ├── go.mod
│       ├── main.go
│       └── internal/
│           ├── handler/                # HTTP 处理器
│           ├── service/                # 业务逻辑
│           ├── repository/             # 数据访问
│           └── proxy/                  # 外部 API 代理
│               ├── monitor_proxy.go    # VM 透传
│               └── alert_proxy.go      # FlashDuty 只读
│
├── web/                                # Vue 前端
│   ├── package.json
│   └── src/
│       └── modules/
│           ├── asset/                  # 资产管理
│           ├── monitor/                # 监控展示
│           ├── alert/                  # 告警展示
│           └── inspect/                # 巡检管理
│
├── api/
│   └── openapi.yaml                    # API 文档
│
├── configs/
├── migrations/                         # 数据库迁移
└── Makefile
```

### 5.2 Go Workspace 配置

```go
// go.work
go 1.25

use (
    ./pkg
    ./apps/cli-tool
    ./apps/cmdb-server
)
```

### 5.3 复用示例代码

#### MonitorProxy - 监控数据透传

```go
// apps/cmdb-server/internal/proxy/monitor_proxy.go
package proxy

import (
    "context"
    "fmt"
    "time"

    "my-platform/pkg/vm"
)

// MonitorProxy 监控数据透传（不存储）
type MonitorProxy struct {
    vmClient *vm.Client  // 复用现有 VM Client
}

// GetRealtimeMetrics 实时指标（透传 VM /api/v1/query）
func (p *MonitorProxy) GetRealtimeMetrics(ctx context.Context, ident string) (*MetricsData, error) {
    cpuResult, _ := p.vmClient.Query(ctx,
        fmt.Sprintf(`cpu_usage_active{cpu="cpu-total", ident="%s"}`, ident))
    memResult, _ := p.vmClient.Query(ctx,
        fmt.Sprintf(`100 - mem_available_percent{ident="%s"}`, ident))

    return &MetricsData{
        CPUUsage:    parseValue(cpuResult),
        MemoryUsage: parseValue(memResult),
    }, nil
}

// GetHistoryMetrics 历史趋势（透传 VM /api/v1/query_range）
func (p *MonitorProxy) GetHistoryMetrics(ctx context.Context, ident string, start, end time.Time) (*HistoryData, error) {
    query := fmt.Sprintf(`cpu_usage_active{cpu="cpu-total", ident="%s"}`, ident)
    result, err := p.vmClient.QueryRange(ctx, query, start, end, "5m")
    if err != nil {
        return nil, err
    }
    return formatForECharts(result), nil
}
```

#### AlertProxy - FlashDuty 告警只读

```go
// apps/cmdb-server/internal/proxy/alert_proxy.go
package proxy

import (
    "context"
    "strconv"

    "github.com/go-resty/resty/v2"
)

// AlertProxy FlashDuty 告警只读代理
type AlertProxy struct {
    apiKey   string
    endpoint string
    http     *resty.Client
}

// ListAlerts 获取告警列表（只读）
func (p *AlertProxy) ListAlerts(ctx context.Context, filter *AlertFilter) ([]*Alert, error) {
    resp, err := p.http.R().
        SetContext(ctx).
        SetHeader("Authorization", "Bearer "+p.apiKey).
        SetQueryParams(map[string]string{
            "status": filter.Status,
            "page":   strconv.Itoa(filter.Page),
            "size":   strconv.Itoa(filter.Size),
        }).
        Get(p.endpoint + "/open-api/v1/alerts")

    if err != nil {
        return nil, err
    }
    return parseAlerts(resp.Body())
}

// GetAlert 获取告警详情（只读）
func (p *AlertProxy) GetAlert(ctx context.Context, alertID string) (*AlertDetail, error) {
    resp, err := p.http.R().
        SetContext(ctx).
        SetHeader("Authorization", "Bearer "+p.apiKey).
        Get(p.endpoint + "/open-api/v1/alerts/" + alertID)

    if err != nil {
        return nil, err
    }
    return parseAlertDetail(resp.Body())
}
```

---

## 6. 核心功能设计

### 6.1 资产管理

| 资产类型    | 数据来源            | 同步方式           |
| ----------- | ------------------- | ------------------ |
| 主机        | N9E API             | 定时同步（每小时） |
| MySQL 实例  | VM (`mysql_up`)     | 定时发现（每小时） |
| Redis 实例  | VM (`redis_up`)     | 定时发现（每小时） |
| Nginx 实例  | VM (`nginx_up`)     | 定时发现（每小时） |
| Tomcat 实例 | VM (`tomcat_up`)    | 定时发现（每小时） |
| ES 集群     | VM (`es_cluster_*`) | 定时发现（每小时） |
| 项目/应用   | 手动录入            | -                  |

### 6.2 监控展示（透传 VM）

| 功能     | VM API                | 说明         |
| -------- | --------------------- | ------------ |
| 实时监控 | `/api/v1/query`       | 当前值       |
| 历史趋势 | `/api/v1/query_range` | 时间范围查询 |

支持的时间范围：1h、6h、24h、7d、30d

### 6.3 告警展示（只读 FlashDuty）

| 功能     | FlashDuty API                     | 说明       |
| -------- | --------------------------------- | ---------- |
| 告警列表 | `GET /open-api/v1/alerts`         | 分页、筛选 |
| 告警详情 | `GET /open-api/v1/alerts/{id}`    | 详情       |
| 故障列表 | `GET /open-api/v1/incidents`      | 故障       |
| 故障详情 | `GET /open-api/v1/incidents/{id}` | 详情       |

### 6.4 巡检管理

**触发方式**：

- Web 界面手动触发
- API 触发
- 定时任务（Cron）

**实现方式**：调用现有 CLI 工具

```go
func (s *InspectService) TriggerInspection(ctx context.Context, req *InspectRequest) (*InspectJob, error) {
    job := &InspectJob{
        Type:        req.Type,
        TriggerType: "manual",
        Status:      "running",
        StartTime:   time.Now(),
        CreatedBy:   req.UserID,
    }

    // 保存任务记录
    if err := s.repo.Create(job); err != nil {
        return nil, err
    }

    // 异步执行巡检
    go func() {
        cmd := exec.Command(s.cliPath, "run", "-c", s.configPath, "--format", "excel,html")
        if err := cmd.Run(); err != nil {
            job.Status = "failed"
            job.ErrorMessage = err.Error()
        } else {
            job.Status = "completed"
        }
        job.EndTime = time.Now()
        s.repo.Update(job)
    }()

    return job, nil
}
```

---

## 7. 外部系统集成

### 7.1 集成汇总

| 外部系统        | 调用方式   | 用途          | 写入    |
| --------------- | ---------- | ------------- | ------- |
| N9E             | REST API   | 主机同步      | ❌ 只读 |
| VictoriaMetrics | PromQL API | 实时+历史监控 | ❌ 只读 |
| FlashDuty       | Open API   | 告警展示      | ❌ 只读 |
| CLI 巡检工具    | 命令行     | 巡检执行      | -       |

### 7.2 不需要的集成

| 功能                | 为什么不需要     |
| ------------------- | ---------------- |
| FlashDuty Event API | N9E 直接推送告警 |
| FlashDuty Webhook   | 只需要查看告警   |
| ClickHouse          | VM 存储历史数据  |
| 监控数据存储        | 透传 VM 查询     |

### 7.3 FlashDuty Open API 参考

| API                               | 用途     |
| --------------------------------- | -------- |
| `GET /open-api/v1/alerts`         | 告警列表 |
| `GET /open-api/v1/alerts/{id}`    | 告警详情 |
| `GET /open-api/v1/incidents`      | 故障列表 |
| `GET /open-api/v1/incidents/{id}` | 故障详情 |

文档：https://developer.flashcat.cloud/zh/flashduty/open-api/quickstart

---

## 8. 数据库设计

### 8.1 存储策略

**只需要 PostgreSQL**：

| 数据类型 | 存储       | 说明                 |
| -------- | ---------- | -------------------- |
| 资产数据 | PostgreSQL | 核心 CMDB 数据       |
| 用户权限 | PostgreSQL | RBAC                 |
| 巡检记录 | PostgreSQL | 任务状态、报告路径   |
| 监控数据 | **不存储** | 透传 VictoriaMetrics |
| 告警数据 | **不存储** | 透传 FlashDuty       |

### 8.2 核心表结构

```sql
-- ============================
-- 业务组织结构
-- ============================

-- 项目表
CREATE TABLE projects (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    code VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    owner VARCHAR(100),
    status VARCHAR(20) DEFAULT 'active',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 业务应用表
CREATE TABLE applications (
    id BIGSERIAL PRIMARY KEY,
    project_id BIGINT REFERENCES projects(id),
    name VARCHAR(255) NOT NULL,
    code VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    owner VARCHAR(100),
    status VARCHAR(20) DEFAULT 'active',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_applications_project_id ON applications(project_id);

-- ============================
-- 资产管理
-- ============================

-- 主机资产表
CREATE TABLE hosts (
    id BIGSERIAL PRIMARY KEY,
    ident VARCHAR(255) UNIQUE NOT NULL,
    hostname VARCHAR(255) NOT NULL,
    ip INET NOT NULL,
    os VARCHAR(100),
    os_version VARCHAR(100),
    kernel_version VARCHAR(100),
    cpu_cores INT,
    cpu_model VARCHAR(255),
    memory_total BIGINT,
    status VARCHAR(20) DEFAULT 'online',
    business_group VARCHAR(100),
    env VARCHAR(50),
    application_id BIGINT REFERENCES applications(id),
    tags JSONB DEFAULT '{}',
    last_sync_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_hosts_ident ON hosts(ident);
CREATE INDEX idx_hosts_ip ON hosts(ip);
CREATE INDEX idx_hosts_application_id ON hosts(application_id);
CREATE INDEX idx_hosts_business_group ON hosts(business_group);

-- MySQL 实例表
CREATE TABLE mysql_instances (
    id BIGSERIAL PRIMARY KEY,
    address VARCHAR(255) UNIQUE NOT NULL,
    ip VARCHAR(50) NOT NULL,
    port INT NOT NULL,
    version VARCHAR(50),
    cluster_mode VARCHAR(50),
    server_id VARCHAR(100),
    host_id BIGINT REFERENCES hosts(id),
    application_id BIGINT REFERENCES applications(id),
    status VARCHAR(20) DEFAULT 'online',
    last_sync_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_mysql_instances_host_id ON mysql_instances(host_id);

-- Redis 实例表
CREATE TABLE redis_instances (
    id BIGSERIAL PRIMARY KEY,
    address VARCHAR(255) UNIQUE NOT NULL,
    ip VARCHAR(50) NOT NULL,
    port INT NOT NULL,
    version VARCHAR(50),
    cluster_mode VARCHAR(50),
    role VARCHAR(20),
    host_id BIGINT REFERENCES hosts(id),
    application_id BIGINT REFERENCES applications(id),
    status VARCHAR(20) DEFAULT 'online',
    last_sync_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_redis_instances_host_id ON redis_instances(host_id);

-- Nginx 实例表
CREATE TABLE nginx_instances (
    id BIGSERIAL PRIMARY KEY,
    address VARCHAR(255) UNIQUE NOT NULL,
    ip VARCHAR(50) NOT NULL,
    port INT NOT NULL,
    version VARCHAR(50),
    host_id BIGINT REFERENCES hosts(id),
    application_id BIGINT REFERENCES applications(id),
    status VARCHAR(20) DEFAULT 'online',
    last_sync_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Tomcat 实例表
CREATE TABLE tomcat_instances (
    id BIGSERIAL PRIMARY KEY,
    address VARCHAR(255) UNIQUE NOT NULL,
    ip VARCHAR(50) NOT NULL,
    port INT NOT NULL,
    version VARCHAR(50),
    jvm_version VARCHAR(50),
    host_id BIGINT REFERENCES hosts(id),
    application_id BIGINT REFERENCES applications(id),
    status VARCHAR(20) DEFAULT 'online',
    last_sync_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Elasticsearch 集群表
CREATE TABLE elasticsearch_clusters (
    id BIGSERIAL PRIMARY KEY,
    cluster_name VARCHAR(255) UNIQUE NOT NULL,
    version VARCHAR(50),
    node_count INT,
    status VARCHAR(20) DEFAULT 'green',
    application_id BIGINT REFERENCES applications(id),
    last_sync_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================
-- 巡检管理
-- ============================

-- 巡检任务表
CREATE TABLE inspection_jobs (
    id BIGSERIAL PRIMARY KEY,
    type VARCHAR(50) NOT NULL,           -- host, mysql, redis, all
    trigger_type VARCHAR(50) NOT NULL,   -- manual, api, cron
    status VARCHAR(50) NOT NULL,         -- pending, running, completed, failed
    start_time TIMESTAMP NOT NULL,
    end_time TIMESTAMP,
    duration_seconds INT,
    report_excel_path VARCHAR(500),
    report_html_path VARCHAR(500),
    summary JSONB,
    error_message TEXT,
    created_by VARCHAR(100),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_inspection_jobs_status ON inspection_jobs(status);
CREATE INDEX idx_inspection_jobs_created_at ON inspection_jobs(created_at);

-- ============================
-- 用户权限
-- ============================

-- 用户表
CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    username VARCHAR(100) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    email VARCHAR(255),
    display_name VARCHAR(255),
    status VARCHAR(20) DEFAULT 'active',
    last_login_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 角色表
CREATE TABLE roles (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 用户角色关联
CREATE TABLE user_roles (
    user_id BIGINT REFERENCES users(id) ON DELETE CASCADE,
    role_id BIGINT REFERENCES roles(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, role_id)
);

-- 权限表
CREATE TABLE permissions (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) UNIQUE NOT NULL,
    resource VARCHAR(100) NOT NULL,
    action VARCHAR(50) NOT NULL,
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 角色权限关联
CREATE TABLE role_permissions (
    role_id BIGINT REFERENCES roles(id) ON DELETE CASCADE,
    permission_id BIGINT REFERENCES permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

-- ============================
-- 初始化数据
-- ============================

-- 初始化角色
INSERT INTO roles (name, description) VALUES
('admin', '系统管理员 - 所有权限'),
('operator', '运维人员 - 资产和巡检管理'),
('viewer', '只读用户 - 查看权限');

-- 初始化权限
INSERT INTO permissions (name, resource, action, description) VALUES
('hosts:read', 'hosts', 'read', '查看主机'),
('hosts:write', 'hosts', 'write', '编辑主机'),
('hosts:delete', 'hosts', 'delete', '删除主机'),
('instances:read', 'instances', 'read', '查看实例'),
('instances:write', 'instances', 'write', '编辑实例'),
('inspection:read', 'inspection', 'read', '查看巡检'),
('inspection:create', 'inspection', 'create', '执行巡检'),
('users:admin', 'users', 'admin', '用户管理');
```

---

## 9. 技术栈选型

### 9.1 后端

```yaml
框架:
  web: Gin v1.10+
  orm: GORM v2.0+
  config: Viper
  validation: go-playground/validator

权限:
  rbac: Casbin v2.0+
  jwt: golang-jwt/jwt v5

工具:
  http: go-resty/resty/v2 # 复用巡检工具
  log: rs/zerolog # 复用巡检工具
  cron: robfig/cron/v3
```

### 9.2 前端

```yaml
框架: Vue 3.4+ / Vite 5.0+ / TypeScript 5.0+
UI: Ant Design Vue 4.x
图表: ECharts 5.x
状态: Pinia 2.x
路由: Vue Router 4.x
HTTP: Axios
```

### 9.3 基础设施

```yaml
数据库: PostgreSQL 14+
缓存: Redis 6+
部署: Docker / 单二进制
```

---

## 10. 项目实施路线图

### 10.1 总体计划（6-8 周）

```
Phase 1: 基础框架 (2周)
├── Week 1: Monorepo 改造、后端基础框架
└── Week 2: 前端基础框架、用户认证

Phase 2: 核心功能 (3周)
├── Week 3: 资产管理（Host + 同步）
├── Week 4: 中间件实例 + 监控展示（透传 VM）
└── Week 5: 告警展示（只读 FlashDuty）+ 巡检管理

Phase 3: 完善 (1-2周)
├── Week 6: 业务拓扑、健康评分
└── Week 7: 测试、部署、文档
```

### 10.2 详细任务清单

#### Phase 1: 基础框架 (2周)

**Week 1: 后端基础**

- [ ] 创建 Monorepo 目录结构
- [ ] 配置 Go Workspace
- [ ] 抽取 pkg/（n9e, vm client）
- [ ] Gin 项目初始化
- [ ] GORM + PostgreSQL 集成
- [ ] 数据库迁移脚本

**Week 2: 前端基础 + 认证**

- [ ] Vue Vben Admin 初始化
- [ ] Casbin RBAC 配置
- [ ] JWT 认证中间件
- [ ] 登录/登出功能
- [ ] 用户管理页面

#### Phase 2: 核心功能 (3周)

**Week 3: 资产管理**

- [ ] 主机 CRUD API
- [ ] N9E 同步服务
- [ ] 主机列表页面
- [ ] 主机详情页面
- [ ] 定时同步任务

**Week 4: 中间件 + 监控**

- [ ] MySQL/Redis/Nginx/Tomcat 实例管理
- [ ] VM 实例发现服务
- [ ] MonitorProxy 实现
- [ ] 监控 ECharts 组件
- [ ] 实例详情页面（含实时监控）

**Week 5: 告警 + 巡检**

- [ ] AlertProxy（FlashDuty 只读）
- [ ] 告警列表页面
- [ ] 告警详情页面
- [ ] 巡检触发 API
- [ ] 巡检历史页面
- [ ] 报告下载功能

#### Phase 3: 完善 (1-2周)

**Week 6: 高级功能**

- [ ] 项目/应用管理
- [ ] 资产关联配置
- [ ] 健康评分展示
- [ ] 大盘概览页面

**Week 7: 收尾**

- [ ] 单元测试
- [ ] 集成测试
- [ ] 部署脚本
- [ ] 用户文档
- [ ] API 文档

---

## 11. 安全与权限设计

### 11.1 RBAC 权限模型

```
角色层次:
├── admin     - 所有权限
├── operator  - 资产/巡检管理
└── viewer    - 只读

资源权限:
├── hosts: read, write, delete
├── instances: read, write, delete
├── inspection: read, create
└── users: admin
```

### 11.2 API 安全措施

| 措施      | 实现方式            |
| --------- | ------------------- |
| 身份认证  | JWT Token + Refresh |
| 接口鉴权  | Casbin 中间件       |
| 请求限流  | Redis 滑动窗口      |
| SQL 注入  | GORM 参数化查询     |
| XSS 防护  | 前端输入转义        |
| CSRF 防护 | CSRF Token          |

---

## 12. 部署方案

### 12.1 单二进制部署（推荐）

```bash
# 构建（前端嵌入后端）
make build

# 部署
scp bin/cmdb-server server:/opt/cmdb/
systemctl start cmdb
```

### 12.2 Docker Compose（开发环境）

```yaml
version: "3.8"
services:
  postgres:
    image: postgres:14
    environment:
      POSTGRES_DB: cmdb
      POSTGRES_USER: cmdb
      POSTGRES_PASSWORD: cmdb123
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data

  redis:
    image: redis:7
    ports:
      - "6379:6379"

  backend:
    build: .
    ports:
      - "8080:8080"
    environment:
      - DB_HOST=postgres
      - REDIS_HOST=redis
    depends_on:
      - postgres
      - redis

volumes:
  postgres_data:
```

### 12.3 Systemd 服务配置

```ini
# /etc/systemd/system/cmdb.service
[Unit]
Description=CMDB Management Platform
After=network.target postgresql.service

[Service]
Type=simple
User=cmdb
WorkingDirectory=/opt/cmdb
ExecStart=/opt/cmdb/cmdb-server -c /opt/cmdb/config.yaml
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

---

## 13. 总结

### 13.1 核心设计原则

1. **CMDB 是资产管理平台，不是监控/告警平台**
2. **外部系统只读调用，不存储冗余数据**
3. **监控数据透传 VM，告警数据透传 FlashDuty**
4. **最大化复用现有巡检工具代码**

### 13.2 不需要的组件

| 组件              | 理由                           |
| ----------------- | ------------------------------ |
| ClickHouse        | 历史数据直接查 VictoriaMetrics |
| FlashDuty 推送    | N9E 直接推送告警               |
| FlashDuty Webhook | 只需查看告警                   |
| 监控数据存储      | 透传 VM                        |
| 告警数据存储      | 透传 FlashDuty                 |

### 13.3 技术选型总结

| 层次   | 选择               |
| ------ | ------------------ |
| 后端   | Gin + GORM         |
| 前端   | Vue Vben Admin     |
| 数据库 | PostgreSQL（唯一） |
| 缓存   | Redis              |

### 13.4 预计工期

**6-8 周**（单人全栈开发）

### 13.5 可复用模块清单

```
inspection-tool/internal/client/
├── n9e/   ← 直接复用（主机元信息获取）
└── vm/    ← 直接复用（PromQL 查询，含 query_range）

inspection-tool/internal/service/
├── collector.go   ← 参考并发模式（errgroup）
└── evaluator.go   ← 参考健康评分逻辑

inspection-tool/internal/model/
└── *.go           ← 参考数据模型设计
```

---

## 附录

### A. FlashDuty Open API 参考

| API                               | 用途     |
| --------------------------------- | -------- |
| `GET /open-api/v1/alerts`         | 告警列表 |
| `GET /open-api/v1/alerts/{id}`    | 告警详情 |
| `GET /open-api/v1/incidents`      | 故障列表 |
| `GET /open-api/v1/incidents/{id}` | 故障详情 |

文档：https://developer.flashcat.cloud/zh/flashduty/open-api/quickstart

### B. VictoriaMetrics API 参考

| API                       | 用途         |
| ------------------------- | ------------ |
| `GET /api/v1/query`       | 实时查询     |
| `GET /api/v1/query_range` | 历史范围查询 |

### C. N9E API 参考

| API                        | 用途     |
| -------------------------- | -------- |
| `GET /api/n9e/targets`     | 主机列表 |
| `GET /api/n9e/target/{id}` | 主机详情 |

认证方式：`X-User-Token` Header

---

> 文档版本：v3.0  
> 最后更新：2026-01-11
