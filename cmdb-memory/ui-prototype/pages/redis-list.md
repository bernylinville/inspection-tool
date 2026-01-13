# Redis 实例列表

> 路由: /assets/middleware/redis

---

## 1. 页面布局

```
┌─────────────────────────────────────────────────────────────────┐
│  Redis 实例管理                                                 │
├─────────────────────────────────────────────────────────────────┤
│  [地址____] [角色▼] [状态▼]                  [查询] [重置]      │
├─────────────────────────────────────────────────────────────────┤
│  □ | 地址 | 版本 | 集群模式 | 角色 | 所属主机 | 状态 | 操作     │
│  ──────────────────────────────────────────────────────────────  │
│  □ | 10.0.0.1:6379 | 6.2 | 3m3s | master | host1 | 在线 | [详情]│
├─────────────────────────────────────────────────────────────────┤
│  共 10 条  [<] [1] [>]    每页 [20] 条                          │
└─────────────────────────────────────────────────────────────────┘
```

---

## 2. 数据字段需求

| 字段名 | 类型 | 说明 | 来源API |
|--------|------|------|---------|
| id | number | 实例ID | GET /api/v1/redis-instances |
| address | string | 地址 | GET /api/v1/redis-instances |
| version | string | 版本 | GET /api/v1/redis-instances |
| cluster_mode | string | 集群模式 | GET /api/v1/redis-instances |
| role | string | 角色(master/slave) | GET /api/v1/redis-instances |
| host_id | number | 所属主机 | GET /api/v1/redis-instances |
| status | string | 状态 | GET /api/v1/redis-instances |

---

## 3. API 依赖汇总

| API | 方法 | 用途 |
|-----|------|------|
| /api/v1/redis-instances | GET | 获取实例列表 |
| /api/v1/redis-instances/:id | GET | 获取实例详情 |
