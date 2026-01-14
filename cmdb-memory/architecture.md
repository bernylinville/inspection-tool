# 内部运维平台 - 统一架构文档

> 版本：v1.0 | 创建日期：2026-01-12

---

## 1. 项目概述

### 1.1 平台定位

| 应用 | 定位 | 状态 |
|------|------|------|
| **巡检工具 CLI** | 基于监控数据的无侵入式系统巡检，生成 Excel/HTML 报告 | ✅ 已实现 |
| **CMDB 资产管理平台** | 资产管理 + 数据聚合展示，Web 化管理界面 | 🚧 开发中 |

### 1.2 核心理念

- **Monorepo 架构**：两个应用共享代码库，统一版本管理
- **代码复用**：公共库（N9E Client、VM Client）被两个应用共享
- **外部系统只读**：N9E、VictoriaMetrics、FlashDuty 均为只读调用

### 1.3 数据流

Categraf → 夜莺 N9E → VictoriaMetrics → 内部运维平台（CLI/Web）

---

## 2. Monorepo 目录结构

### 2.1 整体结构

my-internal-platform/
├── go.work                    # Go Workspace
├── pkg/                       # 公共库
│   ├── n9e/                   # N9E Client
│   ├── vm/                    # VM Client
│   ├── model/                 # 通用模型
│   └── evaluator/             # 健康评分
├── apps/
│   ├── cli-tool/              # 巡检工具 CLI
│   └── cmdb-server/           # CMDB 后端
├── web/                       # Vue 前端
├── configs/                   # 配置文件
├── migrations/                # 数据库迁移
└── Makefile

### 2.2 Go Workspace 配置

go 1.25
use (
    ./pkg
    ./apps/cli-tool
    ./apps/cmdb-server
)

---

## 3. 公共库 (pkg/)

| 模块 | 作用 | 复用来源 |
|------|------|----------|
| n9e/ | N9E API 客户端 | internal/client/n9e |
| vm/ | VictoriaMetrics 客户端 | internal/client/vm |
| model/ | 通用数据模型 | internal/model |
| evaluator/ | 健康评分逻辑 | internal/service/evaluator |

---

## 4. 巡检工具 CLI (apps/cli-tool/)

### 4.1 已实现功能

- Host 巡检：CPU、内存、磁盘、负载、进程
- MySQL 巡检：MGR/双主/主从模式
- Redis 巡检：3m3s/3m6s 集群模式
- Nginx 巡检：二进制和容器部署
- Tomcat 巡检
- Elasticsearch 巡检

### 4.2 命令

- inspect run：执行巡检
- inspect validate：验证配置
- inspect version：显示版本

### 4.3 报告格式

- Excel：3工作表（概览、详情、告警）
- HTML：响应式布局，支持排序

---

## 5. CMDB 后端 (apps/cmdb-server/)

### 5.1 技术栈

- 框架：Go Gin
- ORM：GORM
- 权限：Casbin (RBAC)
- 缓存：Redis

### 5.2 核心模块

| 模块 | 功能 |
|------|------|
| 资产管理 | 项目、应用、主机、中间件实例的 CRUD |
| 监控透传 | MonitorProxy - 透传 VM 查询，不存储 |
| 告警透传 | AlertProxy - 透传 FlashDuty，只读 |
| 巡检管理 | 调用 CLI 执行巡检，存储报告 |
| 用户权限 | 用户、角色、权限管理 |

### 5.3 外部集成

| 系统 | 调用方式 | 用途 |
|------|----------|------|
| N9E | REST API | 主机同步（只读）|
| VictoriaMetrics | PromQL API | 监控数据透传（只读）|
| FlashDuty | Open API | 告警展示（只读）|

---

### 5.4 目录结构 (Phase 1 已实现)

```
apps/cmdb-server/
├── cmd/
│   └── main.go              # 程序入口，支持 -migrate 参数
├── internal/
│   ├── database/
│   │   └── database.go      # GORM 连接、连接池、AutoMigrate
│   └── model/
│       ├── asset.go         # 资产模型 (9个)
│       ├── user.go          # 用户权限模型 (3个)
│       └── inspection.go    # 巡检任务模型 (1个)
├── configs/
│   └── config.yaml          # 配置文件
├── go.mod                   # Go 模块定义
└── go.sum                   # 依赖校验
```

### 5.5 数据模型定义

#### asset.go (9 models)
| 模型 | 表名 | 说明 |
|------|------|------|
| Project | projects | 项目 |
| Application | applications | 应用 |
| Host | hosts | 主机 |
| MySQLInstance | mysql_instances | MySQL实例 |
| RedisInstance | redis_instances | Redis实例 |
| NginxInstance | nginx_instances | Nginx实例 |
| TomcatInstance | tomcat_instances | Tomcat实例 |
| ElasticsearchCluster | elasticsearch_clusters | ES集群 |
| ApplicationHost | application_hosts | 应用-主机关联 |

#### user.go (3 models)
| 模型 | 表名 | 说明 |
|------|------|------|
| User | users | 用户 |
| Role | roles | 角色 |
| Permission | permissions | 权限 |

#### inspection.go (1 model)
| 模型 | 表名 | 说明 |
|------|------|------|
| InspectionJob | inspection_jobs | 巡检任务 |

## 6. Vue 前端 (web/)

### 6.1 技术栈

- 框架：Vue 3 + Vite
- UI：Vben Admin
- 图表：ECharts
- 状态：Pinia

### 6.2 模块结构

| 模块 | 功能 |
|------|------|
| asset/ | 资产管理（项目、应用、主机、中间件）|
| monitor/ | 监控展示（实时指标、历史趋势）|
| alert/ | 告警展示（FlashDuty 透传）|
| inspect/ | 巡检管理（任务、报告、历史）|
| system/ | 系统设置（用户、角色、权限）|

---

## 7. 数据库设计

### 7.1 核心表

| 表名 | 用途 |
|------|------|
| projects | 项目信息 |
| applications | 应用信息 |
| hosts | 主机资产 |
| mysql_instances | MySQL 实例 |
| redis_instances | Redis 实例 |
| nginx_instances | Nginx 实例 |
| tomcat_instances | Tomcat 实例 |
| elasticsearch_clusters | ES 集群 |
| inspection_jobs | 巡检任务 |
| inspection_reports | 巡检报告 |
| users | 用户 |
| roles | 角色 |
| permissions | 权限 |

### 7.2 关联关系

- projects 1:N applications
- applications N:M hosts
- hosts 1:N mysql_instances/redis_instances/...
- users N:M roles
- roles N:M permissions

---

## 8. 外部系统集成

### 8.1 N9E（夜莺）

- 用途：主机元信息同步
- 接口：/api/n9e/targets
- 认证：X-User-Token
- 频率：定时同步（可配置）

### 8.2 VictoriaMetrics

- 用途：监控数据透传
- 接口：/api/v1/query, /api/v1/query_range
- 认证：无
- 模式：实时透传，不存储

### 8.3 FlashDuty

- 用途：告警数据展示
- 接口：Open API
- 认证：API Key
- 模式：只读透传

---

## 9. 版本记录

| 日期 | 版本 | 变更 |
|------|------|------|
| 2026-01-12 | v1.0 | 初始版本，统一架构文档 |
| 2026-01-14 | v1.1 | Phase 1 数据层实现 (Step 1.1-1.5) |
