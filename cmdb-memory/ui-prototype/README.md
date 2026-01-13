# CMDB 资产管理平台 - UI 原型文档

> 版本：v1.0 | 创建日期：2026-01-13
> 基于：vue-vben-admin 框架

---

## 1. 原型总览

本文档集定义 CMDB 资产管理平台的界面原型，用于反向推导 API 字段需求。

### 1.1 平台定位

轻量级资产管理平台，聚合而非替代现有系统：
- **资产管理**：主机、中间件实例的 CRUD
- **数据聚合**：透传 VictoriaMetrics 监控数据、FlashDuty 告警数据（只读）
- **巡检管理**：Web 触发巡检、查看历史报告
- **权限控制**：基于 RBAC 的用户权限管理

### 1.2 目标用户

| 角色 | 核心诉求 |
|------|----------|
| 运维工程师 (SRE) | 资产查看、巡检触发、告警聚合 |
| 开发人员 | 查看应用关联资产的监控数据 |
| 运维主管 | 巡检报告、资产统计、健康概览 |

---

## 2. 菜单结构

```
总览
├── 仪表盘
资产管理
├── 项目
├── 应用
├── 主机
└── 中间件
    ├── MySQL
    ├── Redis
    ├── Nginx
    ├── Tomcat
    └── Elasticsearch
监控
└── 监控查询
告警
└── 告警列表
巡检
└── 巡检任务
系统管理
├── 用户管理
└── 角色管理
```

---

## 3. 页面清单与路由规划

| 序号 | 页面 | 路由 | 文件 | 菜单位置 |
|------|------|------|------|----------|
| 1 | 登录 | /login | login.md | - |
| 2 | 仪表盘 | /dashboard | dashboard.md | 总览 |
| 3 | 项目列表 | /assets/projects | project-list.md | 资产管理 > 项目 |
| 4 | 项目详情 | /assets/projects/:id | project-detail.md | - |
| 5 | 应用列表 | /assets/applications | application-list.md | 资产管理 > 应用 |
| 6 | 应用详情 | /assets/applications/:id | application-detail.md | - |
| 7 | 主机列表 | /assets/hosts | host-list.md | 资产管理 > 主机 |
| 8 | 主机详情 | /assets/hosts/:id | host-detail.md | - |
| 9 | MySQL 实例 | /assets/middleware/mysql | mysql-list.md | 资产管理 > 中间件 > MySQL |
| 10 | Redis 实例 | /assets/middleware/redis | redis-list.md | 资产管理 > 中间件 > Redis |
| 11 | Nginx 实例 | /assets/middleware/nginx | nginx-list.md | 资产管理 > 中间件 > Nginx |
| 12 | Tomcat 实例 | /assets/middleware/tomcat | tomcat-list.md | 资产管理 > 中间件 > Tomcat |
| 13 | ES 集群 | /assets/middleware/elasticsearch | elasticsearch-list.md | 资产管理 > 中间件 > ES |
| 14 | 监控查询 | /monitor/query | monitor-query.md | 监控 |
| 15 | 告警列表 | /alerts | alert-list.md | 告警 |
| 16 | 巡检任务 | /inspection/jobs | inspection-list.md | 巡检 |
| 17 | 巡检详情 | /inspection/jobs/:id | inspection-detail.md | - |
| 18 | 用户管理 | /system/users | user-list.md | 系统管理 > 用户 |
| 19 | 角色管理 | /system/roles | role-list.md | 系统管理 > 角色 |

---

## 4. 全局数据字典

### 4.1 通用状态枚举

| 枚举名 | 值 | 说明 | 颜色 |
|--------|-----|------|------|
| Status | active | 启用 | green |
| Status | inactive | 禁用 | gray |
| HostStatus | online | 在线 | green |
| HostStatus | offline | 离线 | red |

### 4.2 中间件状态

| 枚举名 | 值 | 说明 | 颜色 |
|--------|-----|------|------|
| MySQLClusterMode | mgr | MGR 模式 | blue |
| MySQLClusterMode | dual-master | 双主模式 | purple |
| MySQLClusterMode | master-slave | 主从模式 | cyan |
| RedisRole | master | 主节点 | blue |
| RedisRole | slave | 从节点 | gray |
| ESClusterStatus | green | 健康 | green |
| ESClusterStatus | yellow | 警告 | yellow |
| ESClusterStatus | red | 异常 | red |

### 4.3 告警级别

| 枚举名 | 值 | 说明 | 颜色 |
|--------|-----|------|------|
| AlertSeverity | critical | 严重 | red |
| AlertSeverity | warning | 警告 | orange |
| AlertSeverity | info | 信息 | blue |
| AlertStatus | firing | 触发中 | red |
| AlertStatus | resolved | 已恢复 | green |

### 4.4 巡检状态

| 枚举名 | 值 | 说明 | 颜色 |
|--------|-----|------|------|
| InspectionStatus | pending | 等待中 | gray |
| InspectionStatus | running | 执行中 | blue |
| InspectionStatus | completed | 已完成 | green |
| InspectionStatus | failed | 失败 | red |

### 4.5 错误码

| 错误码 | HTTP Status | 说明 |
|--------|-------------|------|
| 0 | 200 | 成功 |
| 1001 | 400 | 请求参数错误 |
| 2001 | 401 | 未认证 |
| 3001 | 403 | 权限不足 |
| 4001 | 404 | 资源不存在 |
| 5001 | 500 | 服务器内部错误 |

---

## 5. 文档索引

### 5.1 通用组件
- [通用组件规范](./components/common.md)

### 5.2 页面原型
- [登录页](./pages/login.md)
- [仪表盘](./pages/dashboard.md)
- [项目列表](./pages/project-list.md) / [项目详情](./pages/project-detail.md)
- [应用列表](./pages/application-list.md) / [应用详情](./pages/application-detail.md)
- [主机列表](./pages/host-list.md) / [主机详情](./pages/host-detail.md)
- [MySQL 实例](./pages/mysql-list.md)
- [Redis 实例](./pages/redis-list.md)
- [Nginx 实例](./pages/nginx-list.md)
- [Tomcat 实例](./pages/tomcat-list.md)
- [ES 集群](./pages/elasticsearch-list.md)
- [监控查询](./pages/monitor-query.md)
- [告警列表](./pages/alert-list.md)
- [巡检任务](./pages/inspection-list.md) / [巡检详情](./pages/inspection-detail.md)
- [用户管理](./pages/user-list.md)
- [角色管理](./pages/role-list.md)

---

## 6. 版本记录

| 版本 | 日期 | 说明 |
|------|------|------|
| v1.0 | 2026-01-13 | 初始版本 |
