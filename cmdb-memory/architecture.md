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

### 5.4 目录结构 (Phase 3 Step 3.12 已实现)

```
apps/cmdb-server/
├── cmd/
│   └── main.go              # 程序入口，支持 -migrate 参数
├── internal/
│   ├── api/
│   │   ├── router/
│   │   │   └── router.go    # Gin 路由器 (Step 3.1, 3.6-3.11 更新)
│   │   ├── middleware/
│   │   │   ├── auth.go      # JWT 认证中间件 (Step 3.2)
│   │   │   ├── casbin.go    # Casbin 授权中间件 (Step 3.3)
│   │   │   └── error.go     # 统一错误处理中间件 (Step 3.12)
│   │   └── handler/
│   │       ├── health.go    # 健康检查端点 (Step 3.4)
│   │       ├── auth.go      # 认证端点 (Step 3.5)
│   │       ├── user.go      # 用户管理端点 (Step 3.6)
│   │       ├── role.go      # 角色管理端点 (Step 3.7)
│   │       ├── asset.go     # 资产管理端点 (Step 3.8)
│   │       ├── monitor.go   # 监控透传端点 (Step 3.9)
│   │       ├── alert.go     # 告警透传端点 (Step 3.10)
│   │       └── inspection.go # 巡检管理端点 (Step 3.11)
│   ├── database/
│   │   └── database.go      # GORM 连接、连接池、AutoMigrate
│   ├── model/
│   │   ├── asset.go         # 资产模型 (9个)
│   │   ├── init.go          # 基础数据初始化
│   │   ├── user.go          # 用户权限模型 (3个)
│   │   └── inspection.go    # 巡检任务模型 (1个)
│   ├── repository/
│   │   ├── application.go           # 应用仓库
│   │   ├── elasticsearch_cluster.go # ES 集群仓库
│   │   ├── host.go                  # 主机仓库
│   │   ├── inspection_job.go        # 巡检任务仓库
│   │   ├── mysql_instance.go        # MySQL 实例仓库
│   │   ├── nginx_instance.go        # Nginx 实例仓库
│   │   ├── permission.go            # 权限仓库
│   │   ├── project.go               # 项目仓库
│   │   ├── redis_instance.go        # Redis 实例仓库
│   │   ├── repository.go            # 仓库接口与分页定义
│   │   ├── role.go                  # 角色仓库
│   │   ├── tomcat_instance.go       # Tomcat 实例仓库
│   │   └── user.go                  # 用户仓库
│   ├── service/
│   │   ├── auth/
│   │   │   └── auth_service.go      # 认证服务
│   │   ├── user/
│   │   │   └── user_service.go      # 用户服务
│   │   ├── role/
│   │   │   └── role_service.go      # 角色服务
│   │   ├── sync/
│   │   │   ├── host_sync_service.go       # 主机同步服务
│   │   │   └── instance_discovery_service.go # 中间件实例发现服务
│   │   ├── asset/
│   │   │   └── asset_service.go     # 资产管理服务 (Step 2.6)
│   │   └── inspection/
│   │       └── inspect_service.go   # 巡检管理服务 (Step 2.9)
│   └── proxy/
│       ├── monitor_proxy.go         # 监控透传服务 (Step 2.7)
│       └── alert_proxy.go           # 告警透传服务 (Step 2.8)
├── configs/
│   ├── config.yaml          # 配置文件
│   ├── casbin_model.conf    # Casbin RBAC 模型 (Step 2.10)
│   └── casbin_policy.csv    # Casbin 权限策略 (Step 2.10)
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

### 5.6 Repository Layer (Phase 1 Step 1.6-1.8)

| 接口 | 方法 |
|------|------|
| Repository[T] | Create, Update, Delete, FindByID, List |
| HostRepository | Repository[Host], FindByIdent, FindByIP, ListByBusinessGroup |
| ProjectRepository | Repository[Project], FindByCode |
| ApplicationRepository | Repository[Application], FindByCode, ListByProjectID |
| MySQLInstanceRepository | Repository[MySQLInstance], FindByAddress, ListByHostID |
| RedisInstanceRepository | Repository[RedisInstance], FindByAddress, ListByHostID |
| NginxInstanceRepository | Repository[NginxInstance], FindByAddress, ListByHostID |
| TomcatInstanceRepository | Repository[TomcatInstance], FindByAddress, ListByHostID |
| ElasticsearchClusterRepository | Repository[ElasticsearchCluster], FindByClusterName |
| UserRepository | Repository[User], FindByUsername |
| RoleRepository | Repository[Role], FindByName |
| PermissionRepository | Repository[Permission], FindByName, ListByResource |
| InspectionJobRepository | Repository[InspectionJob], ListByStatus, ListByType |

ListOptions 用于分页与过滤：Page、PageSize、OrderBy、Order、Filters。

### 5.7 Base Data Initialization

- 默认角色：admin、operator、viewer
- 默认权限：18 项，覆盖 hosts、projects、applications、middleware、inspection、users、roles、monitor、alerts
- InitializeBaseData：创建默认角色/权限并为角色绑定权限

### 5.8 Service Layer (Phase 2 Step 2.1-2.3)

| 服务 | 文件 | 方法 |
|------|------|------|
| AuthService | auth/auth_service.go | Login, ValidateToken, RefreshToken, HashPassword, VerifyPassword |
| UserService | user/user_service.go | CreateUser, UpdateUser, DeleteUser, GetUser, ListUsers, AssignRoles, ChangePassword |
| RoleService | role/role_service.go | CreateRole, UpdateRole, DeleteRole, GetRole, ListRoles, AssignPermissions, GetRolePermissions |
| HostSyncService | sync/host_sync_service.go | SyncHosts |
| InstanceDiscoveryService | sync/instance_discovery_service.go | DiscoverAll, DiscoverMySQL, DiscoverRedis, DiscoverNginx, DiscoverTomcat, DiscoverElasticsearch |

### 5.9 Service Layer (Phase 2 Step 2.6-2.8)

| 服务 | 文件 | 方法 |
|------|------|------|
| AssetService | asset/asset_service.go | CreateProject, UpdateProject, DeleteProject, GetProject, ListProjects, CreateApplication, UpdateApplication, DeleteApplication, GetApplication, ListApplications, ListApplicationsByProject, CreateHost, UpdateHost, DeleteHost, GetHost, ListHosts, ListHostsByBusinessGroup, AssociateHostToApplication, DisassociateHostFromApplication, Get/List/Delete for MySQL/Redis/Nginx/Tomcat/ES |
| MonitorProxy | proxy/monitor_proxy.go | Query, QueryRange, QueryRangeForECharts |
| AlertProxy | proxy/alert_proxy.go | ListAlerts, ListIncidents |

### 5.10 Service Layer (Phase 2 Step 2.9-2.10)

| 服务 | 文件 | 方法 |
|------|------|------|
| InspectService | inspection/inspect_service.go | CreateJob, GetJob, ListJobs, ListJobsByStatus, ListJobsByType, DeleteJob, UpdateStatus |

### 5.11 Casbin RBAC Configuration (Phase 2 Step 2.10)

#### casbin_model.conf
- Request Definition: (sub, obj, act)
- Policy Definition: (sub, obj, act)
- Role Definition: g = _, _ (支持角色继承)
- Matcher: g(r.sub, p.sub) && keyMatch(r.obj, p.obj) && (r.act == p.act || p.act == "*")

#### casbin_policy.csv 权限矩阵
| 角色 | hosts | projects | applications | middleware | inspection | users | roles | monitor | alerts |
|------|-------|----------|--------------|------------|------------|-------|-------|---------|--------|
| admin | * | * | * | * | * | * | * | * | * |
| operator | r/w | r | r/w | r/w | r/w | - | - | r/w | r/w |
| viewer | r | r | r | r | r | r | r | r | r |

角色继承: admin → operator → viewer

### 5.12 API Layer (Phase 3 Step 3.1-3.5)

#### Router (router/router.go)
| 组件 | 说明 |
|------|------|
| Config | Mode, ReadTimeout, WriteTimeout |
| Router | Gin Engine 封装 |
| Middlewares | Recovery, RequestLogger, CORS |

#### Auth Middleware (middleware/auth.go)
| 方法 | 说明 |
|------|------|
| RequireAuth() | JWT Token 验证，注入 user_id/username/roles 到 Context |
| GetUserID/GetUsername/GetRoles | Context 辅助函数 |

#### Casbin Middleware (middleware/casbin.go)
| 方法 | 说明 |
|------|------|
| RequirePermission() | 基于角色的权限检查 |
| extractResource | URL 路径转资源名 (/api/v1/users/1 → /users) |
| mapMethodToAction | HTTP 方法转操作 (GET→read, POST/PUT/DELETE→write) |

#### Health Handler (handler/health.go)
| 方法 | 说明 |
|------|------|
| HealthCheck() | 完整健康检查，检测 DB 和 Redis 连接状态 |
| checkDatabase() | PostgreSQL 连接检测，5s 超时 |
| checkRedis() | Redis 连接检测，5s 超时 |
| SimpleHealthCheck() | 轻量级状态检查 |

#### Auth Handler (handler/auth.go)
| 方法 | 说明 |
|------|------|
| Login() | POST /api/v1/auth/login，用户登录 |
| Logout() | POST /api/v1/auth/logout，用户登出 |
| Refresh() | POST /api/v1/auth/refresh，刷新 Token |

#### User Handler (handler/user.go) - Step 3.6
| 方法 | 说明 |
|------|------|
| ListUsers() | GET /api/v1/users，分页查询用户列表 |
| CreateUser() | POST /api/v1/users，创建用户 |
| GetUser() | GET /api/v1/users/:id，获取用户详情 |
| UpdateUser() | PUT /api/v1/users/:id，更新用户信息 |
| DeleteUser() | DELETE /api/v1/users/:id，删除用户 |
| AssignRoles() | PUT /api/v1/users/:id/roles，分配角色 |
| ChangePassword() | PUT /api/v1/users/:id/password，修改密码 |

#### Role Handler (handler/role.go) - Step 3.7
| 方法 | 说明 |
|------|------|
| ListRoles() | GET /api/v1/roles，分页查询角色列表 |
| CreateRole() | POST /api/v1/roles，创建角色 |
| GetRole() | GET /api/v1/roles/:id，获取角色详情 |
| UpdateRole() | PUT /api/v1/roles/:id，更新角色信息 |
| DeleteRole() | DELETE /api/v1/roles/:id，删除角色 |
| AssignPermissions() | PUT /api/v1/roles/:id/permissions，分配权限 |

#### Asset Handler (handler/asset.go) - Step 3.8
| 方法 | 说明 |
|------|------|
| ListProjects() | GET /api/v1/projects，分页查询项目列表 |
| CreateProject() | POST /api/v1/projects，创建项目 |
| GetProject() | GET /api/v1/projects/:id，获取项目详情 |
| UpdateProject() | PUT /api/v1/projects/:id，更新项目信息 |
| DeleteProject() | DELETE /api/v1/projects/:id，删除项目 |
| ListApplications() | GET /api/v1/applications，分页查询应用列表 |
| CreateApplication() | POST /api/v1/applications，创建应用 |
| GetApplication() | GET /api/v1/applications/:id，获取应用详情 |
| UpdateApplication() | PUT /api/v1/applications/:id，更新应用信息 |
| DeleteApplication() | DELETE /api/v1/applications/:id，删除应用 |
| ListHosts() | GET /api/v1/hosts，分页查询主机列表 |
| CreateHost() | POST /api/v1/hosts，创建主机 |
| GetHost() | GET /api/v1/hosts/:id，获取主机详情 |
| UpdateHost() | PUT /api/v1/hosts/:id，更新主机信息 |
| DeleteHost() | DELETE /api/v1/hosts/:id，删除主机 |
| SyncHosts() | POST /api/v1/hosts/sync，从 N9E 同步主机 |
| ListMySQLInstances() | GET /api/v1/mysql，查询 MySQL 实例列表 |
| GetMySQLInstance() | GET /api/v1/mysql/:id，获取 MySQL 实例详情 |
| DeleteMySQLInstance() | DELETE /api/v1/mysql/:id，删除 MySQL 实例 |
| ListRedisInstances() | GET /api/v1/redis，查询 Redis 实例列表 |
| GetRedisInstance() | GET /api/v1/redis/:id，获取 Redis 实例详情 |
| DeleteRedisInstance() | DELETE /api/v1/redis/:id，删除 Redis 实例 |
| ListNginxInstances() | GET /api/v1/nginx，查询 Nginx 实例列表 |
| GetNginxInstance() | GET /api/v1/nginx/:id，获取 Nginx 实例详情 |
| DeleteNginxInstance() | DELETE /api/v1/nginx/:id，删除 Nginx 实例 |
| ListTomcatInstances() | GET /api/v1/tomcat，查询 Tomcat 实例列表 |
| GetTomcatInstance() | GET /api/v1/tomcat/:id，获取 Tomcat 实例详情 |
| DeleteTomcatInstance() | DELETE /api/v1/tomcat/:id，删除 Tomcat 实例 |
| ListElasticsearchClusters() | GET /api/v1/elasticsearch，查询 ES 集群列表 |
| GetElasticsearchCluster() | GET /api/v1/elasticsearch/:id，获取 ES 集群详情 |
| DeleteElasticsearchCluster() | DELETE /api/v1/elasticsearch/:id，删除 ES 集群 |

#### Monitor Handler (handler/monitor.go) - Step 3.9
| 方法 | 说明 |
|------|------|
| Query() | GET /api/v1/monitor/query，实时指标查询 (透传 VM) |
| QueryRange() | GET /api/v1/monitor/query_range，历史趋势查询 (支持 format=echarts) |

#### Alert Handler (handler/alert.go) - Step 3.10
| 方法 | 说明 |
|------|------|
| ListAlerts() | GET /api/v1/alerts，告警列表查询 (透传 FlashDuty) |
| GetAlert() | GET /api/v1/alerts/:id，告警详情 (FlashDuty 不支持，返回 501) |
| ListIncidents() | GET /api/v1/incidents，故障列表查询 (透传 FlashDuty) |
| GetIncident() | GET /api/v1/incidents/:id，故障详情 (FlashDuty 不支持，返回 501) |

#### Inspection Handler (handler/inspection.go) - Step 3.11
| 方法 | 说明 |
|------|------|
| ListJobs() | GET /api/v1/inspection/jobs，巡检任务列表 (分页、状态/类型筛选) |
| CreateJob() | POST /api/v1/inspection/jobs，创建并触发巡检任务 |
| GetJob() | GET /api/v1/inspection/jobs/:id，获取巡检任务详情 |
| DeleteJob() | DELETE /api/v1/inspection/jobs/:id，删除巡检任务 (禁止删除运行中任务) |

#### Error Middleware (middleware/error.go) - Step 3.12
| 组件 | 说明 |
|------|------|
| ErrorResponse | 统一错误响应结构 (code, message, details) |
| AppError | 应用错误类型 (code, message, httpStatus, details) |
| ErrorHandler() | Gin 错误处理中间件 |
| AbortWithError() | 中断请求并返回错误响应 |
| AbortWithAppError() | 中断请求并返回 AppError |

#### Error Codes (Step 3.12)
| 代码 | HTTP状态 | 说明 |
|------|----------|------|
| 0 | 200 | 成功 |
| 1001 | 400 | 无效请求 |
| 1002 | 400 | 验证失败 |
| 2001 | 401 | 未授权 |
| 2002 | 401 | 无效Token |
| 2003 | 401 | Token过期 |
| 3001 | 403 | 禁止访问 |
| 4001 | 404 | 未找到 |
| 5001 | 500 | 内部错误 |
| 5002 | 500 | 数据库错误 |
| 5003 | 502 | 外部API错误 |

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

### Frontend API Client (web/apps/web-antd/src/api/)

| Directory/File | Purpose | Status |
|----------------|---------|--------|
| cmdb/types.ts | TypeScript type definitions for all CMDB entities | ✅ |
| cmdb/auth.ts | Authentication API functions | ✅ |
| cmdb/user.ts | User management API | ✅ |
| cmdb/role.ts | Role management API | ✅ |
| cmdb/asset.ts | Asset (Project/Application/Host) API | ✅ |
| cmdb/middleware.ts | Middleware instance API | ✅ |
| cmdb/monitor.ts | Monitoring query API | ✅ |
| cmdb/alert.ts | Alert/Incident API | ✅ |
| cmdb/inspection.ts | Inspection job API | ✅ |
| cmdb/health.ts | Health check API | ✅ |
| cmdb/index.ts | Barrel export | ✅ |
| request.ts | Axios client with interceptors | ✅ |

#### API Client Architecture

- Uses @vben/request library with RequestClient class
- Response format: { code: number, message: string, data: T }
- Success code: 0
- Automatic Bearer token injection via request interceptor
- Token refresh with authenticateResponseInterceptor
- Error display via errorMessageResponseInterceptor

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
| 2026-01-14 | v1.2 | Phase 1 数据层实现 (Step 1.6-1.8): Repository层、基础数据初始化 |
| 2026-01-14 | v1.3 | Phase 2 服务层实现 (Step 2.1-2.3): 认证、用户、角色服务 |
| 2026-01-14 | v1.4 | Phase 2 服务层实现 (Step 2.4-2.5): 主机同步服务、中间件实例发现服务 |
| 2026-01-14 | v1.5 | Phase 2 服务层实现 (Step 2.6-2.8): 资产管理服务、监控透传服务、告警透传服务 |
| 2026-01-14 | v1.6 | Phase 2 服务层实现 (Step 2.9-2.10): 巡检管理服务、Casbin RBAC 配置 |
| 2026-01-14 | v1.7 | Phase 3 API层实现 (Step 3.1-3.3): Gin路由器、JWT认证中间件、Casbin授权中间件 |
| 2026-01-14 | v1.8 | Phase 3 API层实现 (Step 3.4-3.5): 健康检查端点、认证端点 |
| 2026-01-14 | v1.9 | Phase 3 API层实现 (Step 3.6-3.7): 用户管理端点、角色管理端点 |
| 2026-01-14 | v1.10 | Phase 3 API层实现 (Step 3.8-3.9): 资产管理端点、监控透传端点 |
| 2026-01-15 | v1.11 | Phase 3 API层实现 (Step 3.10): 告警透传端点 |
| 2026-01-15 | v1.12 | Phase 3 API层实现 (Step 3.11-3.12): 巡检管理端点、统一错误处理中间件 |
