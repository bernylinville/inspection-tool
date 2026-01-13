# CMDB 数据库 ERD 文档

## 概述

本文档基于 `cmdb-platform-construction-proposal.md` 中的 SQL 表结构，给出 CMDB 数据库实体关系图（Mermaid erDiagram），并补充分模块视图、索引设计与初始化数据说明，便于研发与评审。

## 完整 ERD 图

```mermaid
erDiagram
    projects {
        BIGSERIAL id PK
        VARCHAR name
        VARCHAR code UK
        TEXT description
        VARCHAR owner
        VARCHAR status
        TIMESTAMP created_at
        TIMESTAMP updated_at
    }

    applications {
        BIGSERIAL id PK
        BIGINT project_id FK
        VARCHAR name
        VARCHAR code UK
        TEXT description
        VARCHAR owner
        VARCHAR status
        TIMESTAMP created_at
        TIMESTAMP updated_at
    }

    hosts {
        BIGSERIAL id PK
        VARCHAR ident UK
        VARCHAR hostname
        INET ip
        VARCHAR os
        VARCHAR os_version
        VARCHAR kernel_version
        INT cpu_cores
        VARCHAR cpu_model
        BIGINT memory_total
        VARCHAR status
        VARCHAR business_group
        VARCHAR env
        JSONB tags
        TIMESTAMP last_sync_at
        TIMESTAMP created_at
        TIMESTAMP updated_at
    }

    mysql_instances {
        BIGSERIAL id PK
        VARCHAR address UK
        VARCHAR ip
        INT port
        VARCHAR version
        VARCHAR cluster_mode
        VARCHAR server_id
        BIGINT host_id FK
        BIGINT application_id FK
        VARCHAR status
        TIMESTAMP last_sync_at
        TIMESTAMP created_at
        TIMESTAMP updated_at
    }

    redis_instances {
        BIGSERIAL id PK
        VARCHAR address UK
        VARCHAR ip
        INT port
        VARCHAR version
        VARCHAR cluster_mode
        VARCHAR role
        BIGINT host_id FK
        BIGINT application_id FK
        VARCHAR status
        TIMESTAMP last_sync_at
        TIMESTAMP created_at
        TIMESTAMP updated_at
    }

    nginx_instances {
        BIGSERIAL id PK
        VARCHAR address UK
        VARCHAR ip
        INT port
        VARCHAR version
        BIGINT host_id FK
        BIGINT application_id FK
        VARCHAR status
        TIMESTAMP last_sync_at
        TIMESTAMP created_at
        TIMESTAMP updated_at
    }

    tomcat_instances {
        BIGSERIAL id PK
        VARCHAR address UK
        VARCHAR ip
        INT port
        VARCHAR version
        VARCHAR jvm_version
        BIGINT host_id FK
        BIGINT application_id FK
        VARCHAR status
        TIMESTAMP last_sync_at
        TIMESTAMP created_at
        TIMESTAMP updated_at
    }

    elasticsearch_clusters {
        BIGSERIAL id PK
        VARCHAR cluster_name UK
        VARCHAR version
        INT node_count
        VARCHAR status
        BIGINT application_id FK
        TIMESTAMP last_sync_at
        TIMESTAMP created_at
        TIMESTAMP updated_at
    }

    inspection_jobs {
        BIGSERIAL id PK
        VARCHAR type
        VARCHAR trigger_type
        VARCHAR status
        TIMESTAMP start_time
        TIMESTAMP end_time
        INT duration_seconds
        VARCHAR report_excel_path
        VARCHAR report_html_path
        JSONB summary
        TEXT error_message
        VARCHAR created_by
        TIMESTAMP created_at
    }

    users {
        BIGSERIAL id PK
        VARCHAR username UK
        VARCHAR password_hash
        VARCHAR email
        VARCHAR display_name
        VARCHAR status
        TIMESTAMP last_login_at
        TIMESTAMP created_at
        TIMESTAMP updated_at
    }

    roles {
        BIGSERIAL id PK
        VARCHAR name UK
        TEXT description
        TIMESTAMP created_at
    }

    permissions {
        BIGSERIAL id PK
        VARCHAR name UK
        VARCHAR resource
        VARCHAR action
        TEXT description
        TIMESTAMP created_at
    }

    user_roles {
        BIGINT user_id PK, FK
        BIGINT role_id PK, FK
    }

    role_permissions {
        BIGINT role_id PK, FK
        BIGINT permission_id PK, FK
    }

    application_hosts {
        BIGINT application_id PK, FK
        BIGINT host_id PK, FK
        TIMESTAMP created_at
    }

    projects ||--o{ applications : contains
    applications ||--o{ application_hosts : contains
    hosts ||--o{ application_hosts : belongs
    hosts ||--o{ mysql_instances : runs
    hosts ||--o{ redis_instances : runs
    hosts ||--o{ nginx_instances : runs
    hosts ||--o{ tomcat_instances : runs
    applications ||--o{ mysql_instances : owns
    applications ||--o{ redis_instances : owns
    applications ||--o{ nginx_instances : owns
    applications ||--o{ tomcat_instances : owns
    applications ||--o{ elasticsearch_clusters : owns

    users ||--o{ user_roles : assigns
    roles ||--o{ user_roles : includes
    roles ||--o{ role_permissions : grants
    permissions ||--o{ role_permissions : includes
```

## 分模块 ERD 图

### 业务组织结构

```mermaid
erDiagram
    projects {
        BIGSERIAL id PK
        VARCHAR name
        VARCHAR code UK
        TEXT description
        VARCHAR owner
        VARCHAR status
        TIMESTAMP created_at
        TIMESTAMP updated_at
    }

    applications {
        BIGSERIAL id PK
        BIGINT project_id FK
        VARCHAR name
        VARCHAR code UK
        TEXT description
        VARCHAR owner
        VARCHAR status
        TIMESTAMP created_at
        TIMESTAMP updated_at
    }

    projects ||--o{ applications : contains
```

### 资产管理

```mermaid
erDiagram
    applications {
        BIGSERIAL id PK
        BIGINT project_id FK
        VARCHAR name
        VARCHAR code UK
        TEXT description
        VARCHAR owner
        VARCHAR status
        TIMESTAMP created_at
        TIMESTAMP updated_at
    }

    hosts {
        BIGSERIAL id PK
        VARCHAR ident UK
        VARCHAR hostname
        INET ip
        VARCHAR os
        VARCHAR os_version
        VARCHAR kernel_version
        INT cpu_cores
        VARCHAR cpu_model
        BIGINT memory_total
        VARCHAR status
        VARCHAR business_group
        VARCHAR env
        JSONB tags
        TIMESTAMP last_sync_at
        TIMESTAMP created_at
        TIMESTAMP updated_at
    }

    mysql_instances {
        BIGSERIAL id PK
        VARCHAR address UK
        VARCHAR ip
        INT port
        VARCHAR version
        VARCHAR cluster_mode
        VARCHAR server_id
        BIGINT host_id FK
        BIGINT application_id FK
        VARCHAR status
        TIMESTAMP last_sync_at
        TIMESTAMP created_at
        TIMESTAMP updated_at
    }

    redis_instances {
        BIGSERIAL id PK
        VARCHAR address UK
        VARCHAR ip
        INT port
        VARCHAR version
        VARCHAR cluster_mode
        VARCHAR role
        BIGINT host_id FK
        BIGINT application_id FK
        VARCHAR status
        TIMESTAMP last_sync_at
        TIMESTAMP created_at
        TIMESTAMP updated_at
    }

    nginx_instances {
        BIGSERIAL id PK
        VARCHAR address UK
        VARCHAR ip
        INT port
        VARCHAR version
        BIGINT host_id FK
        BIGINT application_id FK
        VARCHAR status
        TIMESTAMP last_sync_at
        TIMESTAMP created_at
        TIMESTAMP updated_at
    }

    tomcat_instances {
        BIGSERIAL id PK
        VARCHAR address UK
        VARCHAR ip
        INT port
        VARCHAR version
        VARCHAR jvm_version
        BIGINT host_id FK
        BIGINT application_id FK
        VARCHAR status
        TIMESTAMP last_sync_at
        TIMESTAMP created_at
        TIMESTAMP updated_at
    }

    elasticsearch_clusters {
        BIGSERIAL id PK
        VARCHAR cluster_name UK
        VARCHAR version
        INT node_count
        VARCHAR status
        BIGINT application_id FK
        TIMESTAMP last_sync_at
        TIMESTAMP created_at
        TIMESTAMP updated_at
    }

    application_hosts {
        BIGINT application_id PK, FK
        BIGINT host_id PK, FK
        TIMESTAMP created_at
    }

    applications ||--o{ application_hosts : contains
    hosts ||--o{ application_hosts : belongs
    hosts ||--o{ mysql_instances : runs
    hosts ||--o{ redis_instances : runs
    hosts ||--o{ nginx_instances : runs
    hosts ||--o{ tomcat_instances : runs
    applications ||--o{ mysql_instances : owns
    applications ||--o{ redis_instances : owns
    applications ||--o{ nginx_instances : owns
    applications ||--o{ tomcat_instances : owns
    applications ||--o{ elasticsearch_clusters : owns
```

### 用户权限

```mermaid
erDiagram
    users {
        BIGSERIAL id PK
        VARCHAR username UK
        VARCHAR password_hash
        VARCHAR email
        VARCHAR display_name
        VARCHAR status
        TIMESTAMP last_login_at
        TIMESTAMP created_at
        TIMESTAMP updated_at
    }

    roles {
        BIGSERIAL id PK
        VARCHAR name UK
        TEXT description
        TIMESTAMP created_at
    }

    permissions {
        BIGSERIAL id PK
        VARCHAR name UK
        VARCHAR resource
        VARCHAR action
        TEXT description
        TIMESTAMP created_at
    }

    user_roles {
        BIGINT user_id PK, FK
        BIGINT role_id PK, FK
    }

    role_permissions {
        BIGINT role_id PK, FK
        BIGINT permission_id PK, FK
    }

    users ||--o{ user_roles : assigns
    roles ||--o{ user_roles : includes
    roles ||--o{ role_permissions : grants
    permissions ||--o{ role_permissions : includes
```

## 索引设计

- 主键：所有表均使用 `id` 或联合主键（`user_roles`、`role_permissions`、`application_hosts`）作为 PK。
- 唯一索引：
  - `projects.code`、`applications.code`、`hosts.ident`、`mysql_instances.address`、`redis_instances.address`、`nginx_instances.address`、`tomcat_instances.address`、`elasticsearch_clusters.cluster_name`
  - `users.username`、`roles.name`、`permissions.name`
- 普通索引（SQL 已定义）：
  - `applications.project_id`
  - `hosts.ident`、`hosts.ip`、`hosts.business_group`
  - `application_hosts.application_id`、`application_hosts.host_id`
  - `mysql_instances.host_id`
  - `redis_instances.host_id`
  - `inspection_jobs.status`、`inspection_jobs.created_at`

## 初始化数据说明

- 角色初始化：`admin`（系统管理员）、`operator`（运维人员）、`viewer`（只读）。
- 权限初始化：
  - `hosts:read`、`hosts:write`、`hosts:delete`
  - `instances:read`、`instances:write`
  - `inspection:read`、`inspection:create`
  - `users:admin`
- 角色与权限的关联需要后续通过 `role_permissions` 明确配置。
