# Nginx 巡检功能 - 开发实施计划

> 本计划聚焦于 Nginx/OpenResty 巡检功能的新增实现，共 8 个步骤。

---

## 功能概述

基于 Categraf 采集的 Nginx 监控数据，实现 Nginx 巡检功能。支持二进制编译部署和容器部署（容器部署时配置文件通过 volume 挂载到宿主机）。

### 巡检项清单

| 巡检项                  | 说明                      | 数据来源                                         | MVP 状态     |
| ----------------------- | ------------------------- | ------------------------------------------------ | ------------ |
| 巡检时间                | 巡检执行时间              | 系统时间                                         | ✅ 待实现    |
| IP 地址                 | Nginx 实例 IP             | `agent_hostname` 标签 + 夜莺元信息               | ✅ 待实现    |
| 应用类型                | Nginx/OpenResty           | `nginx_info{app_type}`                           | ✅ 已采集    |
| 安装路径                | Nginx 安装目录            | `nginx_info{install_path}`                       | ✅ 已采集    |
| 错误日志路径            | error_log 路径            | `nginx_last_error_timestamp{error_log_path}`     | ✅ 已采集    |
| 访问日志路径            | access_log 路径           | exec 脚本可扩展                                  | ⏳ 后续扩展  |
| 端口                    | 监听端口                  | `nginx_info{port}`                               | ✅ 已采集    |
| 40x 错误页重定向配置    | error_page 4xx 配置       | `nginx_error_page_4xx`                           | ✅ 已采集    |
| 50x 错误页重定向配置    | error_page 5xx 配置       | `nginx_error_page_5xx`                           | ✅ 已采集    |
| 最近一次错误日志时间    | 最后一条 error 日志时间   | `nginx_last_error_timestamp`                     | ✅ 已采集    |
| 连接数                  | 当前活跃连接数            | `nginx_active`（nginx 插件）                     | ✅ 待实现    |
| Worker Processes        | worker 进程数配置         | `nginx_worker_processes`                         | ✅ 已采集    |
| Worker Connections      | 单 worker 最大连接数      | `nginx_worker_connections`                       | ✅ 已采集    |
| 是否普通用户启动        | 非 root 用户启动          | `nginx_non_root_user`                            | ✅ 已采集    |
| 容器名称                | Docker 容器名             | `nginx_up{container}`                            | ✅ 已采集    |

---

## 关键技术决策

### 1. 实例标识方式

- **主要标识**：`agent_hostname` 标签（主机名）
- **辅助标识**：`port` 标签（监听端口）或 `container` 标签（容器名）
- **唯一标识**：`agent_hostname:port` 或 `agent_hostname:container` 组合
- **示例**：`GX-NM-MNS-NGX-01:18099` 或 `GX-NM-MNS-NGX-01:mns-openresty`

### 2. 现有监控指标（nginx 插件）

```yaml
# 来自 nginx.toml 配置
nginx_up                     # 连接状态 (1=正常)
nginx_active                 # 活跃连接数
nginx_accepts                # 接受的连接总数
nginx_handled                # 处理的连接总数
nginx_requests               # 请求总数
nginx_reading                # 正在读取的连接数
nginx_writing                # 正在写入的连接数
nginx_waiting                # 等待中的连接数

# 可用标签
agent_hostname, port, server, target, component, product, env, business
```

### 3. Upstream 健康检查指标（nginx_upstream_check 插件）

```yaml
# 来自 nginx_upstream_check.toml 配置
nginx_upstream_check_status_code    # 后端状态 (1=正常, 0=异常)
nginx_upstream_check_rise           # 连续成功次数
nginx_upstream_check_fall           # 连续失败次数

# 可用标签
agent_hostname, name (后端地址), upstream (upstream名称), type (检查类型)
```

### 4. 扩展采集指标（exec 脚本 - Prometheus 格式）✅ 已实现

通过 `scripts/nginx/collect_nginx_info.sh` 脚本采集，输出 **Prometheus 格式**，已成功上报到 N9E 监控平台。

**指标说明**：

| 指标名称                    | 类型   | 标签                                              | 说明                              |
| --------------------------- | ------ | ------------------------------------------------- | --------------------------------- |
| nginx_up                    | gauge  | agent_hostname, container                         | 容器/进程运行状态 (1=运行)        |
| nginx_info                  | gauge  | agent_hostname, port, app_type, install_path, version | 基本信息指标 (值固定为1)         |
| nginx_worker_processes      | gauge  | agent_hostname                                    | worker 进程数                     |
| nginx_worker_connections    | gauge  | agent_hostname                                    | 单 worker 最大连接数              |
| nginx_non_root_user         | gauge  | agent_hostname                                    | 是否非 root 用户启动 (1=是)       |
| nginx_error_page_4xx        | gauge  | agent_hostname                                    | 是否配置 4xx 错误页 (1=是)        |
| nginx_error_page_5xx        | gauge  | agent_hostname                                    | 是否配置 5xx 错误页 (1=是)        |
| nginx_last_error_timestamp  | gauge  | agent_hostname, error_log_path                    | 最近错误日志 Unix 时间戳          |

### 5. 巡检状态判断规则

| 巡检项              | 正常条件                         | 警告条件                         | 严重条件                         |
| ------------------- | -------------------------------- | -------------------------------- | -------------------------------- |
| 连接状态            | `nginx_up=1`                     | -                                | `nginx_up=0`                     |
| 连接使用率          | `<70%`                           | `70%-90%`                        | `>90%`                           |
| 最近错误日志        | 无最近 1 小时错误                | 1 小时内有错误                   | 10 分钟内有错误                  |
| 40x/50x 错误页配置  | 已配置 (=1)                      | -                                | 未配置 (=0)                      |
| 是否普通用户启动    | 是 (=1)                          | -                                | 否 (=0, root 启动)               |
| Upstream 后端状态   | `status_code=1`                  | -                                | `status_code=0`                  |

**连接使用率计算**：`nginx_active / (nginx_worker_processes * nginx_worker_connections) * 100%`

---

## 阶段一：Categraf 采集配置部署（步骤 1-2）✅ 已完成

### 步骤 1：部署 Nginx 巡检采集脚本 ✅ 已完成

**操作**：

- 将 `scripts/nginx/collect_nginx_info.sh` 复制到 Nginx 服务器的 `/opt/context/categraf/scripts/nginx/` 目录
- 赋予脚本执行权限：`chmod +x /opt/context/categraf/scripts/nginx/collect_nginx_info.sh`
- 手动执行脚本验证输出格式正确

**验证**：

- [x] 执行脚本输出 Prometheus 格式数据
- [x] 输出包含 nginx_up、nginx_info、nginx_worker_processes 等指标
- [x] nginx_info 标签包含 port、app_type、install_path、version
- [x] nginx_last_error_timestamp 标签包含 error_log_path
- [x] 各数值正确（worker_processes=2, worker_connections=102400）

---

### 步骤 2：配置 Categraf exec 插件并验证采集 ✅ 已完成

**操作**：

- 将 `scripts/nginx/exec.toml` 复制到 Categraf 的 `conf/input.exec/` 目录
- 确保 commands 路径与脚本实际位置一致
- 确保 `data_format = "prometheus"`
- 执行 `./categraf --test --inputs exec` 验证采集

**exec.toml 配置**：

```toml
[[instances]]
commands = [
    "/opt/context/categraf/scripts/nginx/collect_nginx_info.sh"
]
timeout = 10
interval_times = 4
data_format = "prometheus"
```

**验证**：

- [x] 执行 `./categraf --test --inputs exec` 输出所有 nginx_* 指标
- [x] Categraf 服务已重启生效
- [x] N9E 监控平台可查询到 exec 脚本上报的监控数据

---

## 阶段二：数据模型与配置扩展（步骤 3-4）

### 步骤 3：定义 Nginx 数据模型

**操作**：

- 在 `internal/model/` 目录下创建 `nginx.go` 文件
- 参照 `mysql.go` 和 `redis.go` 的结构定义以下内容：
  - `NginxInstance` 结构体：
    - Identifier (string): 唯一标识
    - Hostname (string): 主机名 (agent_hostname)
    - IP (string): IP 地址
    - Port (int): 监听端口
    - Container (string): 容器名称（容器部署时）
    - ApplicationType (string): 应用类型 (nginx/openresty)
    - Version (string): 版本号
    - InstallPath (string): 安装路径
    - ErrorLogPath (string): 错误日志路径
  - `NginxInspectionResult` 结构体：
    - Instance (*NginxInstance)
    - Up (bool): 运行状态
    - ActiveConnections (int): 活跃连接数
    - WorkerProcesses (int): worker 进程数
    - WorkerConnections (int): 单 worker 最大连接数
    - ConnectionUsagePercent (float64): 连接使用率（计算值）
    - ErrorPage4xxConfigured (bool): 4xx 错误页配置
    - ErrorPage5xxConfigured (bool): 5xx 错误页配置
    - LastErrorTimestamp (int64): 最近错误日志时间戳
    - LastErrorTimeFormatted (string): 格式化显示
    - NonRootUser (bool): 是否非 root 用户启动
    - UpstreamStatus ([]NginxUpstreamStatus): 后端健康状态
    - Status (NginxInstanceStatus): 整体状态
    - Alerts ([]*NginxAlert): 告警列表
  - `NginxUpstreamStatus` 结构体
  - `NginxInstanceStatus` 枚举和 `NginxAlert` 结构体

**验证**：

- [ ] 执行 `go build ./internal/model/` 无编译错误
- [ ] 结构体字段覆盖所有巡检项

---

### 步骤 4：扩展配置结构并创建指标定义文件

**操作**：

- 在 `internal/config/config.go` 中添加：

```go
type NginxInspectionConfig struct {
    Enabled        bool           `mapstructure:"enabled"`
    InstanceFilter NginxFilter    `mapstructure:"instance_filter"`
    Thresholds     NginxThresholds `mapstructure:"thresholds"`
}

type NginxFilter struct {
    HostnamePatterns []string          `mapstructure:"hostname_patterns"`
    BusinessGroups   []string          `mapstructure:"business_groups"`
    Tags             map[string]string `mapstructure:"tags"`
}

type NginxThresholds struct {
    ConnectionUsageWarning   float64 `mapstructure:"connection_usage_warning"`   // 默认 70
    ConnectionUsageCritical  float64 `mapstructure:"connection_usage_critical"`  // 默认 90
    LastErrorWarningMinutes  int     `mapstructure:"last_error_warning_minutes"`  // 默认 60
    LastErrorCriticalMinutes int     `mapstructure:"last_error_critical_minutes"` // 默认 10
}
```

- 在 `configs/` 目录下创建 `nginx-metrics.yaml`
- 更新 `configs/config.example.yaml` 添加 Nginx 配置节

**验证**：

- [ ] 执行 `go build ./internal/config/` 无编译错误
- [ ] 配置加载正确

---

## 阶段三：Nginx 巡检服务实现（步骤 5-6）

### 步骤 5：实现 Nginx 采集器和评估器

**操作**：

- 创建 `internal/service/nginx_collector.go`：
  - `DiscoverInstances`: 查询 `nginx_info` 发现实例，提取标签信息
  - `CollectMetrics`: 采集 nginx_*（插件）和 nginx_*（exec）指标
  - `CollectUpstreamStatus`: 采集 nginx_upstream_check_* 指标

- 创建 `internal/service/nginx_evaluator.go`：
  - 连接状态评估（nginx_up）
  - 连接使用率评估
  - 最近错误日志时间评估
  - 错误页配置评估
  - 非 root 用户评估
  - Upstream 后端状态评估
  - 告警生成

**验证**：

- [ ] 执行 `go build ./internal/service/` 无编译错误
- [ ] 单元测试通过

---

### 步骤 6：实现 Nginx 巡检服务并集成到主服务

**操作**：

- 创建 `internal/service/nginx_inspector.go`
- 修改 `internal/service/inspector.go` 集成 Nginx 巡检
- 更新 `model/inspection.go` 添加 NginxResults 字段

**验证**：

- [ ] 执行 `go build ./internal/service/` 无编译错误
- [ ] 集成测试通过

---

## 阶段四：报告生成与验收（步骤 7-8）

### 步骤 7：扩展报告生成器支持 Nginx

**操作**：

- 修改 `internal/report/excel/generator.go` 添加 Nginx Sheet
- 修改 `templates/html/report.tmpl` 添加 Nginx 区域
- 更新 `internal/report/summary.go` 添加 Nginx 统计

**Excel 列定义**：

| 列名 | 数据来源 |
|------|----------|
| 主机名 | agent_hostname |
| IP 地址 | 夜莺元信息 |
| 应用类型 | app_type 标签 |
| 端口 | port 标签 |
| 容器名 | container 标签 |
| 版本 | version 标签 |
| 安装路径 | install_path 标签 |
| 错误日志路径 | error_log_path 标签 |
| 活跃连接数 | nginx_active |
| 连接使用率 | 计算值 |
| Worker Processes | nginx_worker_processes |
| Worker Connections | nginx_worker_connections |
| 4xx 错误页 | nginx_error_page_4xx |
| 5xx 错误页 | nginx_error_page_5xx |
| 最近错误时间 | nginx_last_error_timestamp 格式化 |
| 非 root 用户 | nginx_non_root_user |
| 状态 | 评估结果 |

**验证**：

- [ ] Excel 报告包含 Nginx Sheet
- [ ] HTML 报告包含 Nginx 区域

---

### 步骤 8：端到端验收测试

**操作**：

- 在验证环境执行完整巡检
- 验证结果完整性和准确性
- 验证与 Host/MySQL/Redis 巡检兼容性

**验证**：

- [ ] 巡检命令成功完成
- [ ] Nginx 实例正确发现
- [ ] 所有巡检项数据正确
- [ ] 告警规则正确触发
- [ ] 报告正确生成
- [ ] 更新 `memory-bank/progress.md`

---

## 附录 A：已验证的采集数据 ✅

**执行时间**：2025-12-20

**执行命令**：
```bash
/opt/context/categraf/scripts/nginx/collect_nginx_info.sh
```

**实际输出**：
```prometheus
# HELP nginx_up Whether the nginx container is running
# TYPE nginx_up gauge
nginx_up{agent_hostname="GX-NM-MNS-NGX-01",container="mns-openresty"} 1

# HELP nginx_info Nginx/OpenResty information
# TYPE nginx_info gauge
nginx_info{agent_hostname="GX-NM-MNS-NGX-01",port="18099",app_type="openresty",install_path="/opt/context/mns-openresty",version="1.27.1"} 1

# HELP nginx_worker_processes Number of worker processes
# TYPE nginx_worker_processes gauge
nginx_worker_processes{agent_hostname="GX-NM-MNS-NGX-01"} 2

# HELP nginx_worker_connections Max worker connections
# TYPE nginx_worker_connections gauge
nginx_worker_connections{agent_hostname="GX-NM-MNS-NGX-01"} 102400

# HELP nginx_non_root_user Whether nginx runs as non-root user (1=yes, 0=no)
# TYPE nginx_non_root_user gauge
nginx_non_root_user{agent_hostname="GX-NM-MNS-NGX-01"} 1

# HELP nginx_error_page_4xx Whether 4xx error pages are configured
# TYPE nginx_error_page_4xx gauge
nginx_error_page_4xx{agent_hostname="GX-NM-MNS-NGX-01"} 1

# HELP nginx_error_page_5xx Whether 5xx error pages are configured
# TYPE nginx_error_page_5xx gauge
nginx_error_page_5xx{agent_hostname="GX-NM-MNS-NGX-01"} 1

# HELP nginx_last_error_timestamp Timestamp of last error log entry
# TYPE nginx_last_error_timestamp gauge
nginx_last_error_timestamp{agent_hostname="GX-NM-MNS-NGX-01",error_log_path="/data/logs/mns-openresty/global/error.log"} 1766238463
```

**数据解读**：

| 指标 | 值 | 说明 |
|------|-----|------|
| nginx_up | 1 | ✅ 容器 mns-openresty 运行中 |
| port | 18099 | ✅ 监听端口 |
| app_type | openresty | ✅ OpenResty 部署 |
| install_path | /opt/context/mns-openresty | ✅ 安装目录 |
| version | 1.27.1 | ✅ OpenResty 版本 |
| worker_processes | 2 | ✅ 2 个 worker 进程 |
| worker_connections | 102400 | ✅ 单 worker 最大 102400 连接 |
| non_root_user | 1 | ✅ 非 root 用户启动 |
| error_page_4xx | 1 | ✅ 已配置 4xx 错误页 |
| error_page_5xx | 1 | ✅ 已配置 5xx 错误页 |
| last_error_timestamp | 1766238463 | ✅ 最近错误时间 2025-12-20 21:47:43 |
| error_log_path | /data/logs/mns-openresty/global/error.log | ✅ 错误日志路径 |
| container | mns-openresty | ✅ 容器名称 |

---

## 附录 B：采集脚本和配置路径

| 文件 | 路径 |
|------|------|
| 采集脚本 | `scripts/nginx/collect_nginx_info.sh` |
| exec 配置 | `scripts/nginx/exec.toml` |
| 服务器脚本位置 | `/opt/context/categraf/scripts/nginx/collect_nginx_info.sh` |
| 服务器配置位置 | `/opt/context/categraf/conf/input.exec/exec.toml` |

---

## 实施进度

| 阶段 | 步骤 | 状态 | 完成日期 |
|------|------|------|----------|
| 一、采集配置 | 1. 部署采集脚本 | ✅ 已完成 | 2025-12-20 |
| | 2. 配置 exec 插件 | ✅ 已完成 | 2025-12-20 |
| 二、数据模型 | 3. 定义 Nginx 数据模型 | ⏳ 待开始 | - |
| | 4. 扩展配置结构 | ⏳ 待开始 | - |
| 三、服务实现 | 5. 实现采集器和评估器 | ⏳ 待开始 | - |
| | 6. 集成到主服务 | ⏳ 待开始 | - |
| 四、报告验收 | 7. 扩展报告生成器 | ⏳ 待开始 | - |
| | 8. 端到端验收 | ⏳ 待开始 | - |

---

## 附录 C：需求澄清与边界说明

本附录明确关键技术实现细节和边界条件处理规则，确保开发实现 100% 符合需求预期。

### 1. Upstream 健康检查的适用范围

**问题**：是否所有 Nginx 实例都启用了 `nginx_upstream_check` 模块？

**澄清**：
- ❌ 只有作为反向代理的 Nginx 才有 upstream 配置
- 纯静态文件服务器查询 `nginx_upstream_check_status_code` 无数据时，`UpstreamStatus` 返回 **空数组**
- ✅ **UpstreamStatus 字段允许为空数组**，报告中显示 "N/A" 或 "-"

---

### 2. 实例唯一标识规则

**问题**：`NginxInstance.Identifier` 字段的具体生成规则是什么？

**澄清**：
```go
// 优先使用 container，无 container 时使用 port
if container != "" {
    Identifier = fmt.Sprintf("%s:%s", agentHostname, container)
} else {
    Identifier = fmt.Sprintf("%s:%d", agentHostname, port)
}
```

**示例**：
- 容器部署：`GX-NM-MNS-NGX-01:mns-openresty`
- 二进制部署：`GX-NM-MNS-NGX-01:80`

**同一主机同时有容器和二进制部署如何区分？**
- ✅ 通过 `container` 标签是否存在自动区分，不会冲突

---

### 3. IP 地址获取逻辑

**问题**：具体流程是什么？如果找不到对应 hostname 如何处理？

**澄清**：

**获取流程**：
1. 从 `nginx_info` 指标获取 `agent_hostname` 标签值
2. 调用夜莺 API 查询该 hostname 的元信息
3. 从元信息的 `extend_info.network.ipaddress` 提取 IP

**找不到对应 hostname 的处理**：
- ✅ **标记为 "N/A"**，不跳过该实例
- 与 MySQL/Redis 巡检处理方式保持一致

---

### 4. 主机筛选逻辑细节

**问题**：`HostnamePatterns` 支持什么格式？

**澄清**：
- ✅ **通配符（glob 模式）**：如 `GX-NM-*`、`*-NGX-*`
- ❌ **不支持正则表达式**
- 实现方式：使用 `filepath.Match` 或类似函数
- 与 MySQL/Redis 的 `AddressPatterns` 保持一致

---

### 5. 容器 vs 二进制部署的判断依据

**问题**：如何判断部署方式？

**澄清**：
```go
// 通过 container 标签是否存在/非空判断
if container != "" {
    DeploymentType = "container"
} else {
    DeploymentType = "binary"
}
```

**二进制部署时 Container 字段**：
- ✅ 为 **空字符串 `""`**

---

### 6. 边界情况处理

#### 6.1 连接使用率计算

**问题**：数据缺失或异常时如何处理？

**澄清**：

**nginx_active 查询不到数据**：
- ✅ 标记为 "N/A"，不触发告警

**worker_processes 或 worker_connections 为 0**：
```go
if workerProcesses == 0 || workerConnections == 0 {
    ConnectionUsagePercent = -1  // 表示无法计算
    // 报告中显示 "N/A"
}
```

#### 6.2 错误日志时间戳为 0

**问题**：如何显示和评估？

**澄清**：
- `last_error_timestamp = 0` 表示 **从未有错误日志**
- `LastErrorTimeFormatted = "无错误"`
- 状态评估为 **正常**（无需告警）

---

### 7. 数据模型未明确定义的结构

#### 7.1 NginxUpstreamStatus 结构体

**完整定义**：
```go
type NginxUpstreamStatus struct {
    UpstreamName   string `json:"upstream_name"`   // upstream 标签
    BackendAddress string `json:"backend_address"` // name 标签 (IP:Port)
    Status         bool   `json:"status"`          // status_code=1 为 true
    RiseCount      int    `json:"rise_count"`      // rise 次数
    FallCount      int    `json:"fall_count"`      // fall 次数
}
```

#### 7.2 NginxAlert 结构

**澄清**：
- ✅ **复用现有的 `model.Alert` 结构**（与 MySQL/Redis 保持一致）
- 只需添加 Nginx 特定的告警类型常量

---

### 8. nginx-metrics.yaml 结构

**参照 mysql-metrics.yaml 结构**：

```yaml
nginx_metrics:
  # 标准 nginx 插件指标
  - name: nginx_up
    display_name: "连接状态"
    query: "nginx_up"
    category: connection
    note: "1=正常, 0=连接失败"

  - name: nginx_active
    display_name: "活跃连接数"
    query: "nginx_active"
    category: connection

  # exec 扩展指标
  - name: nginx_info
    display_name: "实例信息"
    query: "nginx_info"
    category: info
    label_extract: [port, app_type, install_path, version]

  - name: nginx_worker_processes
    display_name: "Worker 进程数"
    query: "nginx_worker_processes"
    category: config

  - name: nginx_worker_connections
    display_name: "Worker 最大连接数"
    query: "nginx_worker_connections"
    category: config

  - name: nginx_non_root_user
    display_name: "非 root 用户启动"
    query: "nginx_non_root_user"
    category: security
    note: "1=是, 0=否"

  - name: nginx_error_page_4xx
    display_name: "4xx 错误页配置"
    query: "nginx_error_page_4xx"
    category: config
    note: "1=已配置, 0=未配置"

  - name: nginx_error_page_5xx
    display_name: "5xx 错误页配置"
    query: "nginx_error_page_5xx"
    category: config
    note: "1=已配置, 0=未配置"

  - name: nginx_last_error_timestamp
    display_name: "最近错误日志时间"
    query: "nginx_last_error_timestamp"
    category: log
    format: timestamp
    label_extract: [error_log_path]

  # Upstream 健康检查
  - name: nginx_upstream_check_status_code
    display_name: "Upstream 后端状态"
    query: "nginx_upstream_check_status_code"
    category: upstream
    note: "1=正常, 0=异常"
```

---

### 9. 告警规则的优先级

**问题**：多个告警时是否有优先级？整体状态如何确定？

**澄清**：

**告警列表**：
- 按触发顺序记录，不排序

**整体状态取最严重级别**：
```go
func determineOverallStatus(alerts []*Alert) NginxInstanceStatus {
    maxLevel := Normal
    for _, alert := range alerts {
        if alert.Level > maxLevel {
            maxLevel = alert.Level
        }
    }
    return maxLevel
}
```

**级别顺序**：`Critical > Warning > Normal`

---

### 10. 与现有功能的集成细节

#### 10.1 InspectionReport 新增字段

**修改 `model/inspection.go`**：
```go
type InspectionReport struct {
    // 现有字段
    HostResults   []*HostInspectionResult
    MySQLResults  []*MySQLInspectionResult
    RedisResults  []*RedisInspectionResult
    // 新增字段
    NginxResults  []*NginxInspectionResult  // ✅ 需要新增
}
```

#### 10.2 报告展示方式

**Excel 报告**：
- ✅ 新增 **"Nginx巡检"** Sheet（独立工作表）
- 与 MySQL/Redis 保持一致的格式和样式

**HTML 报告**：
- ✅ 新增 **"Nginx 巡检"** 区域卡片
- 独立的数据表格和汇总统计

**与 Host 巡检的关系**：
- ❌ **不合并展示**
- ✅ 作为独立模块单独呈现

---

## 版本记录

| 日期 | 变更说明 |
|------|----------|
| 2025-12-20 | 新增附录 C：需求澄清与边界说明，明确 10 项关键实现细节 |
