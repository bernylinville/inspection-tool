# Tomcat 实例列表

> 路由: /assets/middleware/tomcat

---

## 1. 页面布局

```
┌─────────────────────────────────────────────────────────────────┐
│  Tomcat 实例管理                                                │
├─────────────────────────────────────────────────────────────────┤
│  [关键字____] [状态▼] [所属主机▼]         [查询] [重置]         │
├─────────────────────────────────────────────────────────────────┤
│  □ | 实例地址 | 版本 | Java版本 | Catalina路径 | 所属主机 | 状态 | 操作 │
│  ─────────────────────────────────────────────────────────────  │
│  □ | 10.0.0.2:8080 | 9.0.82 | 1.8 | /opt/tomcat | host2 | 在线 | [详情][删除] │
├─────────────────────────────────────────────────────────────────┤
│  共 15 条  [<] [1] [>]    每页 [20] 条                          │
└─────────────────────────────────────────────────────────────────┘
```

---

## 2. 组件说明

| 组件 | 类型 | 说明 |
|------|------|------|
| 筛选表单 | BasicForm | 关键字、状态、所属主机 |
| 数据表格 | BasicTable | Tomcat 实例列表 |

---

## 3. 数据字段需求

| 字段名 | 类型 | 说明 | 来源API |
|--------|------|------|---------|
| id | number | 实例ID | GET /api/v1/tomcat-instances |
| address | string | 实例地址 | GET /api/v1/tomcat-instances |
| version | string | 版本 | GET /api/v1/tomcat-instances |
| java_version | string | Java版本 | GET /api/v1/tomcat-instances |
| catalina_home | string | Catalina路径 | GET /api/v1/tomcat-instances |
| host_id | number | 所属主机 | GET /api/v1/tomcat-instances |
| status | string | 状态 | GET /api/v1/tomcat-instances |

---

## 4. API 依赖汇总

| API | 方法 | 用途 |
|-----|------|------|
| /api/v1/tomcat-instances | GET | 获取实例列表 |
| /api/v1/tomcat-instances/:id | DELETE | 删除实例 |
