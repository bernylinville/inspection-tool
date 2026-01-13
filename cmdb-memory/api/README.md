# CMDB 资产管理平台 API 文档

## 概述

CMDB 资产管理平台提供 RESTful API，用于管理 IT 基础设施资产、监控数据透传和巡检任务管理。

## 基础信息

| 项目 | 说明 |
|------|------|
| API 版本 | v1 |
| 基础路径 | /api/v1 |
| 数据格式 | JSON |
| 字符编码 | UTF-8 |

## 认证方式

使用 JWT Bearer Token 认证：

```
Authorization: Bearer <token>
```

### 获取 Token

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username": "admin", "password": "your-password"}'
```

### Token 刷新

```bash
curl -X POST http://localhost:8080/api/v1/auth/refresh \
  -H 'Content-Type: application/json' \
  -d '{"refresh_token": "your-refresh-token"}'
```

## API 模块

| 模块 | 说明 | 端点数 |
|------|------|--------|
| auth | 用户认证 | 3 |
| users | 用户管理 | 5 |
| roles | 角色管理 | 5 |
| projects | 项目管理 | 5 |
| applications | 应用管理 | 5 |
| hosts | 主机资产 | 7 |
| mysql-instances | MySQL 实例 | 4 |
| redis-instances | Redis 实例 | 2 |
| nginx-instances | Nginx 实例 | 2 |
| tomcat-instances | Tomcat 实例 | 2 |
| elasticsearch-clusters | ES 集群 | 2 |
| monitor | 监控透传 | 2 |
| alerts | 告警透传 | 4 |
| inspection | 巡检管理 | 4 |
| health | 健康检查 | 1 |

**总计：约 53 个端点**

## 错误码说明

| 错误码 | HTTP Status | 说明 |
|--------|-------------|------|
| 0 | 200 | 成功 |
| 1001 | 400 | 请求参数错误 |
| 1002 | 400 | 数据验证失败 |
| 2001 | 401 | 未认证 |
| 2002 | 401 | Token 过期 |
| 2003 | 401 | Token 无效 |
| 3001 | 403 | 权限不足 |
| 4001 | 404 | 资源不存在 |
| 5001 | 500 | 服务器内部错误 |
| 5002 | 500 | 数据库错误 |
| 5003 | 502 | 外部服务调用失败 |

## 响应格式

### 成功响应

```json
{
  "code": 0,
  "message": "success",
  "data": { ... }
}
```

### 分页响应

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "items": [...],
    "total": 100,
    "page": 1,
    "page_size": 20
  }
}
```

### 错误响应

```json
{
  "code": 1001,
  "message": "请求参数错误",
  "details": "username is required"
}
```

## 主要 API 端点

### 认证模块

| Method | Path | 说明 |
|--------|------|------|
| POST | /api/v1/auth/login | 用户登录 |
| POST | /api/v1/auth/logout | 用户登出 |
| POST | /api/v1/auth/refresh | 刷新 Token |

### 资产管理

| Method | Path | 说明 |
|--------|------|------|
| GET | /api/v1/hosts | 获取主机列表 |
| POST | /api/v1/hosts | 创建主机 |
| GET | /api/v1/hosts/{id} | 获取主机详情 |
| PUT | /api/v1/hosts/{id} | 更新主机 |
| DELETE | /api/v1/hosts/{id} | 删除主机 |
| POST | /api/v1/hosts/sync | 从 N9E 同步主机 |
| GET | /api/v1/hosts/{id}/metrics | 获取主机实时指标 |

### 监控透传

| Method | Path | 说明 |
|--------|------|------|
| GET | /api/v1/monitor/query | 实时指标查询 (PromQL) |
| GET | /api/v1/monitor/query_range | 历史趋势查询 |

### 巡检管理

| Method | Path | 说明 |
|--------|------|------|
| GET | /api/v1/inspection/jobs | 获取巡检任务列表 |
| POST | /api/v1/inspection/jobs | 触发巡检任务 |
| GET | /api/v1/inspection/jobs/{id} | 获取巡检任务详情 |
| DELETE | /api/v1/inspection/jobs/{id} | 删除巡检任务 |

## Swagger UI 使用

### 本地预览

```bash
# 使用 Docker 启动 Swagger UI
docker run -p 8081:8080 -e SWAGGER_JSON=/api/openapi.yaml -v $(pwd):/api swaggerapi/swagger-ui

# 访问 http://localhost:8081
```

### 在线预览

将 openapi.yaml 上传至 [Swagger Editor](https://editor.swagger.io/) 在线预览。

## 文件说明

| 文件 | 说明 |
|------|------|
| openapi.yaml | OpenAPI 3.1 规范文件 |
| README.md | API 说明文档（本文件） |

## 版本历史

| 版本 | 日期 | 说明 |
|------|------|------|
| 1.0.0 | 2026-01-13 | 初始版本 |
