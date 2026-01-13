# ES 集群列表

> 路由: /assets/middleware/elasticsearch

---

## 1. 页面布局

```
┌─────────────────────────────────────────────────────────────────┐
│  ES 集群管理                                                    │
├─────────────────────────────────────────────────────────────────┤
│  [关键字____] [状态▼]                     [查询] [重置]         │
├─────────────────────────────────────────────────────────────────┤
│  □ | 集群名称 | 版本 | 状态 | 节点数 | 分片数 | 索引数 | 存储用量 | 操作 │
│  ─────────────────────────────────────────────────────────────  │
│  □ | es-prod | 8.11 | green | 6 | 120 | 560 | 3.2TB | [详情]    │
├─────────────────────────────────────────────────────────────────┤
│  共 5 条  [<] [1] [>]    每页 [20] 条                           │
└─────────────────────────────────────────────────────────────────┘
```

---

## 2. 组件说明

| 组件 | 类型 | 说明 |
|------|------|------|
| 筛选表单 | BasicForm | 关键字、状态 |
| 数据表格 | BasicTable | ES 集群列表 |

---

## 3. 数据字段需求

| 字段名 | 类型 | 说明 | 来源API |
|--------|------|------|---------|
| id | number | 集群ID | GET /api/v1/es-instances |
| cluster_name | string | 集群名称 | GET /api/v1/es-instances |
| version | string | 版本 | GET /api/v1/es-instances |
| status | string | 状态 | GET /api/v1/es-instances |
| node_count | number | 节点数 | GET /api/v1/es-instances |
| shard_count | number | 分片数 | GET /api/v1/es-instances |
| index_count | number | 索引数 | GET /api/v1/es-instances |
| storage_used | string | 存储用量 | GET /api/v1/es-instances |

---

## 4. API 依赖汇总

| API | 方法 | 用途 |
|-----|------|------|
| /api/v1/es-instances | GET | 获取集群列表 |
| /api/v1/es-instances/:id | DELETE | 删除集群 |
