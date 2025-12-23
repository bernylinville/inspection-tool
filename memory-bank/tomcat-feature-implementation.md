# Tomcat 应用巡检功能 - 开发实施计划

> 本计划聚焦于 Tomcat 应用巡检功能的 MVP 实现，支持容器和二进制部署两种方式。

---

## 功能概述

基于 Categraf exec 插件执行 Shell 脚本采集的 Tomcat 监控数据，实现 Tomcat 应用巡检功能。

### 巡检项清单

| 巡检项              | 说明                      | 数据来源                                         | MVP 状态     |
| ------------------- | ------------------------- | ------------------------------------------------ | ------------ |
| 巡检时间            | 巡检执行时间              | 系统时间                                         | ✅ 待实现    |
| IP 地址             | Tomcat 实例 IP            | `tomcat_up{ip}` 或 `tomcat_info{ip}` 标签        | ✅ 已采集    |
| 应用类型            | Tomcat                    | `tomcat_info{app_type}` 标签                     | ✅ 已采集    |
| 安装路径            | Tomcat 安装目录           | `tomcat_info{install_path}` 标签                 | ✅ 已采集    |
| 日志路径            | 日志目录                  | `tomcat_info{log_path}` 标签                     | ✅ 已采集    |
| 端口                | HTTP 监听端口             | `tomcat_up{port}` 或 `tomcat_info{port}` 标签    | ✅ 已采集    |
| 连接数              | 当前 HTTP 连接数          | `tomcat_connections`                             | ✅ 已采集    |
| 应用配置            | JVM 配置参数              | `tomcat_info{jvm_config}` 标签                   | ✅ 已采集    |
| 是否普通用户启动    | 非 root 用户启动          | `tomcat_non_root_user`                           | ✅ 已采集    |
| Tomcat 版本         | 版本号                    | `tomcat_info{version}` 标签                      | ✅ 已采集    |
| 容器名称            | Docker 容器名             | `tomcat_up{container}` 标签                      | ✅ 已采集    |
| 运行时长            | 容器/进程运行时间         | `tomcat_uptime_seconds`                          | ✅ 已采集    |
| 最近错误日志时间    | 最后一条错误日志时间      | `tomcat_last_error_timestamp`                    | ✅ 已采集    |

---

## 关键技术决策

### 1. 部署模式支持

**MVP 优先支持容器部署**，同时预留二进制部署扩展能力。

| 部署方式   | 配置文件位置                    | 检测方式                        |
| ---------- | ------------------------------- | ------------------------------- |
| 容器部署   | 宿主机 volume 挂载目录          | `container` 标签非空            |
| 二进制部署 | Tomcat 安装目录下的 conf 目录   | `container` 标签为空            |

### 2. 实例标识方式

- **主要标识**：`container` 标签（容器名）或 `port` 标签（端口）
- **唯一标识组合**：`agent_hostname:container` 或 `agent_hostname:port`
- **示例**：`GX-MFUI-BE-01:tomcat-18001` 或 `GX-MFUI-BE-01:18001`

**标识生成规则**：
```go
// 优先使用 container，无 container 时使用 port
if container != "" {
    Identifier = fmt.Sprintf("%s:%s", agentHostname, container)
} else {
    Identifier = fmt.Sprintf("%s:%d", agentHostname, port)
}
```

### 3. 监控指标映射（基于 tomcat.txt 验证）

以下指标**已验证可用**（Prometheus 格式）：

```yaml
# exec 脚本采集的指标
tomcat_up                        # 容器/进程运行状态 (1=运行, 0=停止)
tomcat_info                      # 基本信息（包含多个标签）
tomcat_connections               # 当前 HTTP 连接数
tomcat_non_root_user             # 是否非 root 用户启动 (1=是, 0=否)
tomcat_uptime_seconds            # 运行时长（秒）
tomcat_pid                       # 容器主进程 PID
tomcat_last_error_timestamp      # 最近错误日志 Unix 时间戳

# 可用标签
agent_hostname   # 主机名
container        # 容器名称
ip               # IP 地址
port             # HTTP 端口
app_type         # 应用类型 (tomcat)
install_path     # 安装路径
log_path         # 日志路径
version          # Tomcat 版本
jvm_config       # JVM 配置参数
```

### 4. 巡检状态判断规则

| 巡检项              | 正常条件                         | 警告条件                         | 严重条件                         |
| ------------------- | -------------------------------- | -------------------------------- | -------------------------------- |
| 运行状态            | `tomcat_up=1`                    | -                                | `tomcat_up=0`                    |
| 是否普通用户启动    | `non_root_user=1`                | -                                | `non_root_user=0` (root 启动)    |
| 最近错误日志        | 无最近 1 小时错误                | 1 小时内有错误                   | 10 分钟内有错误                  |
| 连接数              | `< 配置最大值 70%`               | `70%-90%`                        | `> 90%`                          |

### 5. 验证环境

**开发和功能验证使用陕西营销活动环境**：
- VictoriaMetrics 指向陕西营销活动监控系统
- Categraf exec 插件已配置并验证可采集数据

### 6. 采集脚本配置

**exec.toml 配置**：

```toml
[[instances]]
commands = [
    "/opt/context/categraf/scripts/tomcat/collect_tomcat_info.sh"
]
timeout = 10
interval_times = 4
data_format = "prometheus"
```

---

## 阶段一：Categraf 采集配置部署（步骤 1-2）✅ 已完成

### 步骤 1：部署 Tomcat 巡检采集脚本 ✅ 已完成

**操作**：

- 将 `collect_tomcat_info.sh` 脚本复制到 Tomcat 服务器的 `/opt/context/categraf/scripts/tomcat/` 目录
- 赋予脚本执行权限：`chmod +x /opt/context/categraf/scripts/tomcat/collect_tomcat_info.sh`
- 手动执行脚本验证输出格式正确

**验证**：

- [x] 执行脚本输出 Prometheus 格式数据
- [x] 输出包含 `tomcat_up`、`tomcat_info`、`tomcat_connections` 等指标
- [x] `tomcat_info` 标签包含 `port`、`app_type`、`install_path`、`log_path`、`version`、`jvm_config`
- [x] `tomcat_up` 标签包含 `container`、`ip`、`port`
- [x] 各数值正确（运行状态、连接数、运行时长等）

---

### 步骤 2：配置 Categraf exec 插件并验证采集 ✅ 已完成

**操作**：

- 将 `exec.toml` 复制到 Categraf 的 `conf/input.exec/` 目录
- 确保 commands 路径与脚本实际位置一致
- 确保 `data_format = "prometheus"`
- 执行 `./categraf --test --inputs exec` 验证采集
- 重启 Categraf 服务

**验证**：

- [x] 执行 `./categraf --test --inputs exec` 输出所有 `tomcat_*` 指标
- [x] Categraf 服务已重启生效
- [x] N9E 监控平台可查询到 exec 脚本上报的监控数据

---

## 阶段二：数据模型与配置扩展（步骤 3-4）

### 步骤 3：定义 Tomcat 数据模型

**操作**：

- 在 `internal/model/` 目录下创建 `tomcat.go` 文件
- 参照 `nginx.go`、`mysql.go` 的结构定义以下内容：
  - `TomcatInstance` 结构体：
    - Identifier (string): 唯一标识
    - Hostname (string): 主机名 (agent_hostname)
    - IP (string): IP 地址
    - Port (int): HTTP 监听端口
    - Container (string): 容器名称
    - ApplicationType (string): 应用类型 (tomcat)
    - Version (string): Tomcat 版本
    - InstallPath (string): 安装路径
    - LogPath (string): 日志路径
    - JVMConfig (string): JVM 配置参数
  - `TomcatInspectionResult` 结构体：
    - Instance (*TomcatInstance)
    - Up (bool): 运行状态
    - Connections (int): 当前连接数
    - UptimeSeconds (int64): 运行时长（秒）
    - UptimeFormatted (string): 格式化显示
    - LastErrorTimestamp (int64): 最近错误日志时间戳
    - LastErrorTimeFormatted (string): 格式化显示
    - NonRootUser (bool): 是否非 root 用户启动
    - PID (int): 容器主进程 PID
    - Status (TomcatInstanceStatus): 整体状态
    - Alerts ([]*TomcatAlert): 告警列表
  - `TomcatInstanceStatus` 枚举和 `TomcatAlert` 结构体

**验证**：

- [ ] 执行 `go build ./internal/model/` 无编译错误
- [ ] 结构体字段覆盖所有巡检项

---

### 步骤 4：扩展配置结构并创建指标定义文件

**操作**：

- 在 `internal/config/config.go` 中添加：

```go
type TomcatInspectionConfig struct {
    Enabled        bool             `mapstructure:"enabled"`
    InstanceFilter TomcatFilter     `mapstructure:"instance_filter"`
    Thresholds     TomcatThresholds `mapstructure:"thresholds"`
}

type TomcatFilter struct {
    HostnamePatterns []string          `mapstructure:"hostname_patterns"`
    ContainerPatterns []string         `mapstructure:"container_patterns"`
    BusinessGroups   []string          `mapstructure:"business_groups"`
    Tags             map[string]string `mapstructure:"tags"`
}

type TomcatThresholds struct {
    LastErrorWarningMinutes  int `mapstructure:"last_error_warning_minutes"`  // 默认 60
    LastErrorCriticalMinutes int `mapstructure:"last_error_critical_minutes"` // 默认 10
}
```

- 在 `configs/` 目录下创建 `tomcat-metrics.yaml`
- 更新 `configs/config.example.yaml` 添加 Tomcat 配置节

**验证**：

- [ ] 执行 `go build ./internal/config/` 无编译错误
- [ ] 配置加载正确

---

## 阶段三：Tomcat 巡检服务实现（步骤 5-6）

### 步骤 5：实现 Tomcat 采集器和评估器

**操作**：

- 创建 `internal/service/tomcat_collector.go`：
  - `DiscoverInstances`: 查询 `tomcat_up` 发现实例，提取标签信息
  - `CollectMetrics`: 采集所有 `tomcat_*` 指标
  - 从 `tomcat_info` 标签提取 `port`、`app_type`、`install_path`、`log_path`、`version`、`jvm_config`

- 创建 `internal/service/tomcat_evaluator.go`：
  - 运行状态评估（tomcat_up）
  - 非 root 用户启动评估
  - 最近错误日志时间评估
  - 告警生成

**验证**：

- [ ] 执行 `go build ./internal/service/` 无编译错误
- [ ] 单元测试通过

---

### 步骤 6：实现 Tomcat 巡检服务并集成到主服务

**操作**：

- 创建 `internal/service/tomcat_inspector.go`
- 修改 `internal/service/inspector.go` 集成 Tomcat 巡检
- 更新 `model/inspection.go` 添加 TomcatResults 字段

**验证**：

- [ ] 执行 `go build ./internal/service/` 无编译错误
- [ ] 集成测试通过

---

## 阶段四：报告生成与验收（步骤 7-8）

### 步骤 7：扩展报告生成器支持 Tomcat

**操作**：

- 修改 `internal/report/excel/writer.go` 添加 Tomcat Sheet
- 修改 `templates/html/combined.html` 添加 Tomcat 区域
- 更新报告统计添加 Tomcat 统计

**Excel 列定义**：

| 列名             | 数据来源                    |
| ---------------- | --------------------------- |
| 主机名           | agent_hostname              |
| IP 地址          | ip 标签                     |
| 应用类型         | app_type 标签               |
| 端口             | port 标签                   |
| 容器名           | container 标签              |
| 版本             | version 标签                |
| 安装路径         | install_path 标签           |
| 日志路径         | log_path 标签               |
| JVM 配置         | jvm_config 标签             |
| 连接数           | tomcat_connections          |
| 运行时长         | tomcat_uptime_seconds 格式化 |
| 非 root 用户     | tomcat_non_root_user        |
| 最近错误时间     | tomcat_last_error_timestamp 格式化 |
| 状态             | 评估结果                    |

**验证**：

- [ ] Excel 报告包含 Tomcat Sheet
- [ ] HTML 报告包含 Tomcat 区域

---

### 步骤 8：端到端验收测试

**操作**：

- 在验证环境执行完整巡检
- 验证结果完整性和准确性
- 验证与 Host/MySQL/Redis/Nginx 巡检兼容性

**验证**：

- [ ] 巡检命令成功完成
- [ ] Tomcat 实例正确发现
- [ ] 所有巡检项数据正确
- [ ] 告警规则正确触发
- [ ] 报告正确生成
- [ ] 更新 `memory-bank/progress.md`

---

## 附录 A：已验证的采集数据

**执行时间**：2025-12-23

**执行命令**：
```bash
/opt/context/categraf/scripts/tomcat/collect_tomcat_info.sh
```

**实际输出示例**：
```prometheus
# Tomcat Container: tomcat-18001
tomcat_up{agent_hostname="GX-MFUI-BE-01",container="tomcat-18001",ip="172.31.110.49",port="18001"} 1
tomcat_info{agent_hostname="GX-MFUI-BE-01",container="tomcat-18001",ip="172.31.110.49",port="18001",app_type="tomcat",install_path="/opt/context/tomcat-18001",log_path="/data/logs/tomcat-18001",version="8.5.99",jvm_config="-Xms2g -Xmx5g "} 1
tomcat_connections{agent_hostname="GX-MFUI-BE-01",container="tomcat-18001",ip="172.31.110.49",port="18001"} 2
tomcat_non_root_user{agent_hostname="GX-MFUI-BE-01",container="tomcat-18001",ip="172.31.110.49",port="18001"} 1
tomcat_uptime_seconds{agent_hostname="GX-MFUI-BE-01",container="tomcat-18001",ip="172.31.110.49",port="18001"} 1259
tomcat_pid{agent_hostname="GX-MFUI-BE-01",container="tomcat-18001",ip="172.31.110.49",port="18001"} 1829181
tomcat_last_error_timestamp{agent_hostname="GX-MFUI-BE-01",container="tomcat-18001",ip="172.31.110.49",port="18001"} 0
```

**数据解读**：

| 指标                  | 值                           | 说明                        |
| --------------------- | ---------------------------- | --------------------------- |
| tomcat_up             | 1                            | ✅ 容器 tomcat-18001 运行中  |
| container             | tomcat-18001                 | ✅ 容器名称                  |
| ip                    | 172.31.110.49                | ✅ 实例 IP 地址              |
| port                  | 18001                        | ✅ HTTP 监听端口             |
| app_type              | tomcat                       | ✅ 应用类型                  |
| install_path          | /opt/context/tomcat-18001    | ✅ 安装路径                  |
| log_path              | /data/logs/tomcat-18001      | ✅ 日志路径                  |
| version               | 8.5.99                       | ✅ Tomcat 版本               |
| jvm_config            | -Xms2g -Xmx5g                | ✅ JVM 配置                  |
| tomcat_connections    | 2                            | ✅ 当前连接数                |
| tomcat_non_root_user  | 1                            | ✅ 非 root 用户启动          |
| tomcat_uptime_seconds | 1259                         | ✅ 运行 1259 秒              |
| last_error_timestamp  | 0                            | ✅ 无错误日志                |

---

## 附录 B：采集脚本和配置路径

| 文件             | 路径                                                          |
| ---------------- | ------------------------------------------------------------- |
| 采集脚本         | `scripts/tomcat/collect_tomcat_info.sh`                       |
| exec 配置        | `scripts/tomcat/exec.toml`                                    |
| 服务器脚本位置   | `/opt/context/categraf/scripts/tomcat/collect_tomcat_info.sh` |
| 服务器配置位置   | `/opt/context/categraf/conf/input.exec/exec.toml`             |

---

## 附录 C：tomcat-metrics.yaml 配置

```yaml
tomcat_metrics:
  # 运行状态
  - name: tomcat_up
    display_name: "运行状态"
    query: "tomcat_up"
    category: status
    note: "1=运行中, 0=已停止"

  # 基础信息
  - name: tomcat_info
    display_name: "实例信息"
    query: "tomcat_info"
    category: info
    label_extract: [port, app_type, install_path, log_path, version, jvm_config]

  # 连接数
  - name: tomcat_connections
    display_name: "当前连接数"
    query: "tomcat_connections"
    category: connection

  # 非 root 用户
  - name: tomcat_non_root_user
    display_name: "非 root 用户启动"
    query: "tomcat_non_root_user"
    category: security
    note: "1=是, 0=否"

  # 运行时长
  - name: tomcat_uptime_seconds
    display_name: "运行时长"
    query: "tomcat_uptime_seconds"
    category: status
    format: duration

  # 进程 PID
  - name: tomcat_pid
    display_name: "进程 PID"
    query: "tomcat_pid"
    category: status

  # 最近错误日志时间
  - name: tomcat_last_error_timestamp
    display_name: "最近错误日志时间"
    query: "tomcat_last_error_timestamp"
    category: log
    format: timestamp
```

---

## 附录 D：示例配置文件

```yaml
# Tomcat 应用巡检配置
tomcat:
  enabled: true

  # 实例筛选（可选）
  instance_filter:
    hostname_patterns:
      - "GX-MFUI-*"
    container_patterns:
      - "tomcat-*"
    business_groups:
      - "生产Tomcat"
    tags:
      env: "prod"

  # 阈值配置
  thresholds:
    # 最近错误日志时间阈值（分钟）
    last_error_warning_minutes: 60   # 1 小时内有错误 → 警告
    last_error_critical_minutes: 10  # 10 分钟内有错误 → 严重
```

---

## 附录 E：需求澄清与边界说明

### 1. 容器 vs 二进制部署的判断依据

**通过 `container` 标签是否存在/非空判断**：
```go
if container != "" {
    DeploymentType = "container"
} else {
    DeploymentType = "binary"
}
```

### 2. 错误日志时间戳为 0 的处理

- `last_error_timestamp = 0` 表示 **从未有错误日志**
- `LastErrorTimeFormatted = "无错误"`
- 状态评估为 **正常**

### 3. JVM 配置为空的处理

- 如果无法获取 JVM 配置，`jvm_config` 标签为空字符串
- 报告中显示为 **"-"** 或 **"未配置"**

### 4. IP 地址获取

- 直接从 `tomcat_up` 或 `tomcat_info` 指标的 `ip` 标签获取
- 无需调用夜莺 API 查询元信息

---

## 实施进度

| 阶段           | 步骤                    | 状态       | 完成日期   |
| -------------- | ----------------------- | ---------- | ---------- |
| 一、采集配置   | 1. 部署采集脚本         | ✅ 已完成  | 2025-12-23 |
|                | 2. 配置 exec 插件       | ✅ 已完成  | 2025-12-23 |
| 二、数据模型   | 3. 定义 Tomcat 数据模型 | ⏳ 待开始  | -          |
|                | 4. 扩展配置结构         | ⏳ 待开始  | -          |
| 三、服务实现   | 5. 实现采集器和评估器   | ⏳ 待开始  | -          |
|                | 6. 集成到主服务         | ⏳ 待开始  | -          |
| 四、报告验收   | 7. 扩展报告生成器       | ⏳ 待开始  | -          |
|                | 8. 端到端验收           | ⏳ 待开始  | -          |

---

## 版本记录

| 日期       | 变更说明                                                |
| ---------- | ------------------------------------------------------- |
| 2025-12-23 | 初始版本，基于 Categraf exec 脚本采集实现 Tomcat 巡检   |
