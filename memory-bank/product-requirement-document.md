# 基于监控数据的系统巡检工具 - 产品需求文档

## 1. 概述

### 1.1 背景

当前系统巡检通过在服务器上批量执行 Shell 脚本获取巡检信息，存在以下问题：

- 需要登录服务器执行脚本，操作繁琐
- 脚本执行对服务器有一定资源消耗
- 难以实现自动化和定时巡检

现有基础设施已通过 Categraf 采集监控数据，推送至夜莺（N9E），再转写到 VictoriaMetrics 存储。

### 1.2 目标

开发一个基于监控数据的系统巡检工具，通过调用 API 接口查询监控数据，实现无侵入式系统巡检，并生成 Excel 和 HTML 格式的巡检报告。

---

## 2. 系统架构

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│    Categraf     │────▶│   夜莺 (N9E)    │────▶│ VictoriaMetrics │
│   (数据采集)     │     │   (监控平台)     │     │   (时序数据库)   │
└─────────────────┘     └─────────────────┘     └─────────────────┘
                               │                        │
                               │ 元信息查询              │ 指标数据查询
                               ▼                        ▼
                        ┌─────────────────────────────────────┐
                        │         系统巡检工具                  │
                        │  ┌─────────┐  ┌─────────┐           │
                        │  │ 数据采集 │  │ 报告生成 │           │
                        │  └─────────┘  └─────────┘           │
                        └─────────────────────────────────────┘
                                        │
                        ┌───────────────┴───────────────┐
                        ▼                               ▼
                 ┌─────────────┐                ┌─────────────┐
                 │  Excel 报告  │                │  HTML 报告   │
                 └─────────────┘                └─────────────┘
```

---

## 3. 数据源配置

### 3.1 夜莺 API

| 配置项   | 值                                        |
| -------- | ----------------------------------------- |
| 地址     | `http://${nightingale_api_address}:17000` |
| 认证方式 | Token 认证                                |
| 认证头   | `X-User-Token`                            |
| Token    | `${nightingale_api_token}`                |

**用途**：获取监控对象元信息（系统类型、系统版本、内核版本、主机名、IP 地址等）

### 3.2 VictoriaMetrics API

| 配置项       | 值                                       |
| ------------ | ---------------------------------------- |
| 查询地址     | `http://${nightingale_api_address}:8428` |
| 即时查询接口 | `/api/v1/query`                          |
| 范围查询接口 | `/api/v1/query_range`                    |

**用途**：获取指标类监控数据（CPU、内存、磁盘、进程等）

---

## 4. 巡检项目定义

### 4.1 巡检项目清单

| 序号 | 巡检项           | 数据来源        | 指标/接口                     | 单位  |
| ---- | ---------------- | --------------- | ----------------------------- | ----- |
| 1    | 巡检时间         | 系统生成        | 当前时间                      | -     |
| 2    | 系统类型         | 夜莺元信息      | 主机标签 `os`                 | -     |
| 3    | 系统版本         | 夜莺元信息      | 主机标签                      | -     |
| 4    | 内核版本         | 夜莺元信息      | 主机标签                      | -     |
| 5    | 系统时间         | VictoriaMetrics | 查询时间戳                    | -     |
| 6    | 运行时间         | VictoriaMetrics | `system_uptime`               | 秒    |
| 7    | 最后一次重启时间 | 计算            | 当前时间 - 运行时间           | -     |
| 8    | 主机名           | VictoriaMetrics | 标签 `ident` / `host`         | -     |
| 9    | IP 地址          | 夜莺元信息      | 主机 IP                       | -     |
| 10   | 公网访问         | 待定            | 需额外检测                    | -     |
| 11   | 密码过期(天)     | 待定            | 监控数据不包含                | 天    |
| 12   | 密码策略         | 待定            | 监控数据不包含                | -     |
| 13   | CPU 利用率       | VictoriaMetrics | `cpu_usage_active`            | %     |
| 14   | 内存总量         | VictoriaMetrics | `mem_total`                   | bytes |
| 15   | 内存空闲         | VictoriaMetrics | `mem_free`                    | bytes |
| 16   | 可分配内存       | VictoriaMetrics | `mem_available`               | bytes |
| 17   | 内存利用率       | VictoriaMetrics | `100 - mem_available_percent` | %     |
| 18   | 磁盘总量         | VictoriaMetrics | `disk_total`                  | bytes |
| 19   | 磁盘剩余         | VictoriaMetrics | `disk_free`                   | bytes |
| 20   | 磁盘利用率       | VictoriaMetrics | `disk_used_percent`           | %     |
| 21   | NTP 检查         | 待定            | 需额外检测                    | -     |
| 22   | 总进程数         | VictoriaMetrics | `processes_total`             | 个    |
| 23   | 僵尸进程数       | VictoriaMetrics | `processes_zombies`           | 个    |
| 24   | 打开文件句柄数   | 待定            | 需额外采集                    | 个    |
| 25   | 句柄数最大值     | 待定            | 需额外采集                    | 个    |
| 26   | 系统参数检查     | 待定            | 需额外采集                    | -     |

### 4.2 指标查询表达式

基于 `categraf-linux-metrics.json` 定义的指标表达式：

```yaml
# CPU 相关
cpu_utilization: cpu_usage_active
cpu_idle: cpu_usage_idle
cpu_iowait: cpu_usage_iowait
cpu_system: cpu_usage_system
cpu_user: cpu_usage_user

# 内存相关
mem_total: mem_total
mem_free: mem_free
mem_available: mem_available
mem_available_percent: mem_available_percent
mem_used: mem_used
mem_used_percent: mem_used_percent
mem_utilization: 100 - mem_available_percent

# 交换空间
swap_total: mem_swap_total
swap_free: mem_swap_free
swap_used: mem_swap_total - mem_swap_free
swap_utilization: (mem_swap_total - mem_swap_free)/mem_swap_total * 100 and mem_swap_total > 0

# 磁盘相关
disk_total: disk_total
disk_free: disk_free
disk_used: disk_used
disk_used_percent: disk_used_percent
disk_inode_used_percent: disk_inodes_used / disk_inodes_total * 100

# 系统负载
load_1m: system_load1
load_5m: system_load5
load_15m: system_load15
cpu_cores: system_n_cpus
uptime: system_uptime

# 进程相关
processes_total: processes_total
processes_running: processes_running
processes_sleeping: processes_sleeping
processes_blocked: processes_blocked
processes_zombies: processes_zombies

# 网络连接
tcp_established: netstat_tcp_established
tcp_time_wait: netstat_tcp_time_wait
tcp_listen: netstat_tcp_listen
udp_socket: netstat_udp_socket
```

---

## 5. 功能需求

### 5.1 数据采集模块

#### 5.1.1 夜莺元信息查询

- 调用夜莺 API 获取主机列表及元信息
- 支持按业务组、标签筛选主机
- 获取主机的系统类型、版本、内核版本等信息

#### 5.1.2 VictoriaMetrics 指标查询

- 使用即时查询接口 `/api/v1/query` 获取最新指标值
- 支持批量查询多个主机的指标
- 支持按标签（`ident`、`host`）筛选主机

### 5.2 报告生成模块

#### 5.2.1 Excel 报告

- 文件格式：`.xlsx`
- 包含以下工作表：
  - **巡检概览**：巡检时间、主机总数、异常主机数
  - **详细数据**：每台主机的完整巡检数据
  - **异常汇总**：超出阈值的指标汇总
- 支持条件格式（异常值高亮显示）

#### 5.2.2 HTML 报告

- 响应式布局，支持移动端查看
- 包含以下内容：
  - 巡检摘要卡片
  - 主机列表表格（支持排序、筛选）
  - 异常指标高亮显示
  - 图表展示（可选）

### 5.3 告警阈值配置

| 指标           | 警告阈值 | 严重阈值 |
| -------------- | -------- | -------- |
| CPU 利用率     | > 70%    | > 90%    |
| 内存利用率     | > 70%    | > 90%    |
| 磁盘利用率     | > 70%    | > 90%    |
| 僵尸进程数     | > 0      | > 10     |
| 系统负载(单核) | > 0.7    | > 1.0    |

---

## 6. 接口调用示例

### 6.1 夜莺 API 调用

```bash
# 获取主机列表
curl -X GET \
  -H "X-User-Token: ${nightingale_api_token}" \
  -H "Content-Type: application/json" \
  "http://${nightingale_api_address}:17000/api/n9e/targets"
```

### 6.2 VictoriaMetrics API 调用

```bash
# 即时查询 - 获取所有主机的 CPU 利用率
curl -G "http://${nightingale_api_address}:8428/api/v1/query" \
  --data-urlencode 'query=cpu_usage_active{cpu="cpu-total"}'

# 即时查询 - 获取所有主机的内存利用率
curl -G "http://${nightingale_api_address}:8428/api/v1/query" \
  --data-urlencode 'query=100 - mem_available_percent'

# 即时查询 - 获取所有主机的磁盘利用率
curl -G "http://${nightingale_api_address}:8428/api/v1/query" \
  --data-urlencode 'query=disk_used_percent'

# 即时查询 - 获取系统运行时间
curl -G "http://${nightingale_api_address}:8428/api/v1/query" \
  --data-urlencode 'query=system_uptime'
```

---

## 7. 输出示例

### 7.1 Excel 报告结构

```
inspection_report_20251213.xlsx
├── 巡检概览
│   ├── 巡检时间: 2025-12-13 10:00:00
│   ├── 主机总数: 50
│   ├── 正常主机: 45
│   └── 异常主机: 5
├── 详细数据
│   └── [主机名, IP, 系统类型, CPU%, 内存%, 磁盘%, ...]
└── 异常汇总
    └── [主机名, 异常指标, 当前值, 阈值]
```

### 7.2 HTML 报告结构

```html
inspection_report_20251213.html ├── 页头（巡检时间、统计摘要） ├──
异常告警区域（红色/黄色高亮） ├── 主机详情表格 │ ├── 表头（可排序） │ └──
数据行（异常值高亮） └── 页脚（生成时间、工具版本）
```

---

## 8. 非功能需求

### 8.1 性能要求

- 支持同时巡检 100+ 台主机
- 单次巡检完成时间 < 5 分钟
- 报告生成时间 < 30 秒

### 8.2 可配置性

- 支持配置文件定义数据源地址、认证信息
- 支持自定义巡检项目和阈值
- 支持自定义报告模板

### 8.3 可扩展性

- 预留接口支持新增巡检项目
- 支持新增报告输出格式（如 PDF）

---

## 9. 待确认事项

| 序号 | 事项               | 说明                                   |
| ---- | ------------------ | -------------------------------------- |
| 1    | 夜莺主机元信息接口 | 需确认获取主机系统类型、版本的具体 API |
| 2    | 密码策略检查       | 监控数据不包含，是否需要保留此项       |
| 3    | NTP 检查           | 是否有对应的监控指标                   |
| 4    | 文件句柄数         | 是否已通过 Categraf 采集               |
| 5    | 系统参数检查       | 具体检查哪些参数，是否有对应指标       |
| 6    | 公网访问检查       | 检查逻辑和实现方式                     |

---

## 10. 版本记录

| 版本 | 日期       | 作者 | 说明     |
| ---- | ---------- | ---- | -------- |
| v1.0 | 2025-12-13 | -    | 初始版本 |
