# MySQL 数据库巡检功能 - 开发实施计划

> 本计划聚焦于 MySQL 8.0 MGR 集群巡检功能的 MVP 实现。

---

## 功能概述

基于 Categraf 采集的 MySQL 监控数据，实现 MySQL 数据库巡检功能。巡检项包括：

| 巡检项              | 说明                | 数据来源                                                      | MVP 状态     |
| ------------------- | ------------------- | ------------------------------------------------------------- | ------------ |
| 巡检时间            | 巡检执行时间        | 系统时间                                                      | ✅ 已实现    |
| IP 地址             | 数据库实例 IP       | `address` 标签                                                | ✅ 已实现    |
| 数据库类型          | MySQL               | 固定值                                                        | ✅ 已实现    |
| 数据库端口          | 实例端口            | `address` 标签解析                                            | ✅ 已实现    |
| 数据库版本          | MySQL 版本号        | `mysql_version_info` 指标 `version` 标签                      | ✅ 已实现    |
| Server ID           | MGR 成员 ID         | `mysql_innodb_cluster_mgr_role_primary` 指标 `member_id` 标签 | ✅ 已实现    |
| 是否普通用户启动    | 非 root 用户启动    | -                                                             | ⏳ 显示 N/A |
| Slave 是否启动      | 从库复制线程状态    | 8.0 MGR: 不适用                                               | ⏳ 显示 N/A |
| 同步状态是否正常    | MGR 同步状态        | `mysql_innodb_cluster_mgr_state_online`                       | ✅ 已实现    |
| 是否开启慢查询日志  | slow_query_log 状态 | `mysql_variables_slow_query_log`                              | ✅ 已实现    |
| 日志路径            | 慢查询日志路径      | `slow_query_log_file` 标签                                    | ✅ 已实现    |
| 最大连接数          | max_connections     | `mysql_global_variables_max_connections`                      | ✅ 已实现    |
| Binlog 配置是否正常 | binlog 开启状态     | `mysql_binlog_file_count` > 0                                 | ✅ 已实现    |
| Binlog 保留时长     | binlog 过期配置     | `mysql_variables_binlog_expire_logs_seconds`                  | ✅ 已实现    |
| MySQL 连接权限      | 监控用户权限检查    | `mysql_up` = 1 表示连接正常                                   | ✅ 已实现    |

---

## 关键技术决策

### 1. MySQL 集群模式支持策略

**MVP 仅支持 MySQL 8.0 MGR 模式**，生产环境不会出现不同架构的 MySQL 集群混用情况。

| 版本 | 集群模式        | 优先级      | 同步状态判断方式                             |
| ---- | --------------- | ----------- | -------------------------------------------- |
| 8.0  | MGR (1 主 N 从) | P0 MVP 实现 | `mgr_state_online=1` 且成员数达到预期        |
| 5.7  | 双主/主从       | P1 后续扩展 | `Slave_IO_Running` 且 `Slave_SQL_Running`    |

### 2. 实例标识方式

- **主要标识**：`address` 标签（格式：`IP:Port`）
- **示例**：`172.18.182.130:33306`
- **解析**：分割冒号获取 IP 和端口
- **约束**：同一服务器不会运行多个 MySQL 实例（无需处理多实例场景）

### 3. 监控指标映射（基于 mysql.txt 分析）

```yaml
# 可直接使用的指标
mysql_up                                    # 连接状态 (1=正常)
mysql_version_info{version="8.0.39"}       # 版本信息
mysql_global_variables_max_connections      # 最大连接数
mysql_global_status_threads_connected       # 当前连接数
mysql_global_status_uptime                  # 运行时间
mysql_binlog_file_count                     # binlog 文件数
mysql_binlog_size_bytes                     # binlog 总大小

# MGR 相关指标（自定义查询）
mysql_innodb_cluster_mgr_member_count       # MGR 在线成员数
mysql_innodb_cluster_mgr_role_primary       # 是否为 PRIMARY 节点
mysql_innodb_cluster_mgr_state_online       # 节点是否 ONLINE
```

### 4. 扩展采集项（已部署）

以下 Categraf mysql.toml 配置**已部署到陕西营销活动生产环境**，用于开发验证：

```toml
# 自定义查询: MySQL 变量
[[instances.queries]]
mesurement = "variables"
timeout = "5s"
request = '''
SELECT
  @@slow_query_log as slow_query_log,
  @@slow_query_log_file as slow_query_log_file,
  @@binlog_expire_logs_seconds as binlog_expire_logs_seconds,
  @@server_id as server_id
'''
metric_fields = ["slow_query_log", "binlog_expire_logs_seconds", "server_id"]
label_fields = ["slow_query_log_file"]
```

**实际采集的指标名称**（Categraf 命名规则：`mysql_{measurement}_{field}`）：

| 巡检项          | 指标名称                                     | 示例值                                   |
| --------------- | -------------------------------------------- | ---------------------------------------- |
| 慢查询日志状态  | `mysql_variables_slow_query_log`             | 1 (开启)                                 |
| 慢查询日志路径  | `slow_query_log_file` 标签                   | `/data/logs/mysql-33306/mysql-slow.log`  |
| Binlog 保留时长 | `mysql_variables_binlog_expire_logs_seconds` | 604800 (7 天)                            |
| Server ID       | `mysql_variables_server_id`                  | 130                                      |

### 5. 巡检状态判断规则

| 巡检项          | 正常条件                         | 警告条件                         | 严重条件                         |
| --------------- | -------------------------------- | -------------------------------- | -------------------------------- |
| 连接权限        | `mysql_up=1`                     | -                                | `mysql_up=0`                     |
| 同步状态（MGR） | `mgr_state_online=1`             | -                                | `mgr_state_online=0`             |
| MGR 成员数      | `count >= expected`              | `count = expected - 1`           | `count < expected - 1`           |
| 连接使用率      | `<70%`                           | `70%-90%`                        | `>90%`                           |
| Binlog 配置     | `binlog_file_count>0`            | -                                | `binlog_file_count=0`            |

**MGR 成员数阈值说明**：
- 默认期望成员数为 3（1 主 2 从架构）
- 该值在 `config.yaml` 中可配置，支持 5、7、9 节点扩展
- 成员数 = 期望值时为正常
- 成员数 = 期望值 - 1 时为警告（掉 1 个节点）
- 成员数 < 期望值 - 1 时为严重（掉 2 个及以上节点）

### 6. 集群模式配置

**不支持自动检测**，需在 `config.yaml` 中手动指定集群模式：

```yaml
mysql:
  cluster_mode: "mgr"  # 可选值: mgr, dual-master, master-slave
```

### 7. 验证环境

**开发和功能验证使用陕西营销活动环境**（而非短剧项目环境）：
- N9E API 和 VictoriaMetrics 指向陕西营销活动监控系统
- Host 巡检也使用该环境验证

---

## 阶段一：数据模型与配置扩展（步骤 1-4）

### 步骤 1：定义 MySQL 实例模型

**操作**：

- 在 `internal/model/` 目录下创建 `mysql.go` 文件
- 定义 `MySQLInstance` 结构体，包含以下字段：
  - Address (string): 实例地址 (IP:Port)
  - IP (string): IP 地址
  - Port (int): 端口号
  - DatabaseType (string): 数据库类型，固定为 "MySQL"
  - Version (string): 数据库版本
  - InnoDBVersion (string): InnoDB 版本
  - ServerID (string): Server ID
  - ClusterMode (string): 集群模式 (MGR/双主/主从)
- 定义 `MySQLInstanceStatus` 枚举（normal/warning/critical/failed）

**验证**：

- [ ] 执行 `go build ./internal/model/` 无编译错误
- [ ] 结构体字段覆盖所有巡检项

---

### 步骤 2：定义 MySQL 巡检结果模型

**操作**：

- 在 `internal/model/mysql.go` 中添加以下结构体：
  - `MySQLInspectionResult`: 单个实例的巡检结果
    - Instance (\*MySQLInstance): 实例元信息
    - ConnectionStatus (bool): 连接状态
    - SlaveRunning (bool): Slave 是否运行（MGR 模式显示 N/A）
    - SyncStatus (bool): 同步状态
    - SlowQueryLogEnabled (bool): 慢查询日志状态
    - SlowQueryLogPath (string): 日志路径
    - MaxConnections (int): 最大连接数
    - CurrentConnections (int): 当前连接数
    - BinlogEnabled (bool): Binlog 是否启用
    - BinlogExpireSeconds (int): Binlog 保留时长
    - MGRMemberCount (int): MGR 成员数（仅 MGR 模式）
    - MGRRole (string): MGR 角色 (PRIMARY/SECONDARY)
    - MGRStateOnline (bool): MGR 节点是否在线
    - NonRootUser (string): 是否普通用户启动（MVP 阶段固定为 "N/A"）
    - Status (MySQLInstanceStatus): 整体状态
    - Alerts ([]MySQLAlert): 告警列表
  - `MySQLAlert`: 告警详情

**验证**：

- [ ] 执行 `go build ./internal/model/` 无编译错误
- [ ] 结构体字段能承载所有巡检数据

---

### 步骤 3：扩展配置结构体

**操作**：

- 在 `internal/config/config.go` 中添加 MySQL 巡检配置：

  ```go
  type MySQLInspectionConfig struct {
      Enabled           bool              `mapstructure:"enabled"`
      ClusterMode       string            `mapstructure:"cluster_mode"` // "mgr", "dual-master", "master-slave"
      InstanceFilter    MySQLFilter       `mapstructure:"instance_filter"`
      Thresholds        MySQLThresholds   `mapstructure:"thresholds"`
  }

  type MySQLFilter struct {
      AddressPatterns   []string          `mapstructure:"address_patterns"` // 地址匹配模式
      BusinessGroups    []string          `mapstructure:"business_groups"`
      Tags              map[string]string `mapstructure:"tags"`
  }

  type MySQLThresholds struct {
      ConnectionUsageWarning  float64 `mapstructure:"connection_usage_warning"`  // 默认 70
      ConnectionUsageCritical float64 `mapstructure:"connection_usage_critical"` // 默认 90
      MGRMemberCountExpected  int     `mapstructure:"mgr_member_count_expected"` // 默认 3，可配置为 5/7/9
  }
  ```

- 在主配置结构体 `Config` 中添加 `MySQL MySQLInspectionConfig` 字段

**说明**：

- `cluster_mode` 不支持 `auto`，必须手动指定
- `mgr_member_count_expected` 用于定义期望的 MGR 节点数，告警阈值基于此值计算：
  - 警告阈值 = expected - 1
  - 严重阈值 = expected - 2

**验证**：

- [ ] 执行 `go build ./internal/config/` 无编译错误
- [ ] 配置加载器能正确解析 MySQL 配置节

---

### 步骤 4：创建 MySQL 指标定义文件

**操作**：

- 在 `configs/` 目录下创建 `mysql-metrics.yaml` 文件
- 定义所有 MySQL 巡检指标的 PromQL 查询表达式
- 使用与 `metrics.yaml` 相同的格式

**指标定义内容**：

```yaml
mysql_metrics:
  # 基础指标
  - name: mysql_up
    display_name: "连接状态"
    query: "mysql_up"
    category: connection
    note: "1=正常, 0=连接失败"

  - name: mysql_version
    display_name: "数据库版本"
    query: "mysql_version_info"
    category: info
    label_extract: version
    note: "从 version 标签提取版本号"

  # MGR 相关指标 (8.0)
  - name: mgr_member_count
    display_name: "MGR 成员数"
    query: "mysql_innodb_cluster_mgr_member_count"
    category: mgr
    cluster_mode: mgr
    note: "MGR 集群在线成员数量"

  - name: mgr_role_primary
    display_name: "MGR 主节点"
    query: "mysql_innodb_cluster_mgr_role_primary"
    category: mgr
    cluster_mode: mgr
    label_extract: member_id
    note: "1=PRIMARY, 0=SECONDARY"

  - name: mgr_state_online
    display_name: "MGR 在线状态"
    query: "mysql_innodb_cluster_mgr_state_online"
    category: mgr
    cluster_mode: mgr
    note: "1=ONLINE, 0=OFFLINE/RECOVERING"

  # 连接相关
  - name: max_connections
    display_name: "最大连接数"
    query: "mysql_global_variables_max_connections"
    category: connection

  - name: current_connections
    display_name: "当前连接数"
    query: "mysql_global_status_threads_connected"
    category: connection

  # Binlog 相关
  - name: binlog_file_count
    display_name: "Binlog 文件数"
    query: "mysql_binlog_file_count"
    category: binlog

  - name: binlog_size
    display_name: "Binlog 总大小"
    query: "mysql_binlog_size_bytes"
    category: binlog
    format: size

  # 运行状态
  - name: uptime
    display_name: "运行时间"
    query: "mysql_global_status_uptime"
    category: status
    format: duration

  # 慢查询日志（已通过自定义查询采集）
  - name: slow_query_log
    display_name: "慢查询日志"
    query: "mysql_variables_slow_query_log"
    category: log
    note: "1=开启, 0=关闭"

  - name: slow_query_log_file
    display_name: "慢查询日志路径"
    query: "mysql_variables_slow_query_log"
    category: log
    label_extract: slow_query_log_file
    note: "从 slow_query_log_file 标签提取"

  - name: binlog_expire_seconds
    display_name: "Binlog 保留时长"
    query: "mysql_variables_binlog_expire_logs_seconds"
    category: binlog
    format: duration
    note: "单位：秒，默认 604800 (7天)"

  - name: server_id
    display_name: "Server ID"
    query: "mysql_variables_server_id"
    category: info
    note: "MySQL 实例的 server_id"

  # 待定项 - MVP 阶段显示 N/A
  - name: non_root_user
    display_name: "非 root 用户启动"
    query: ""
    category: security
    status: pending
    note: "MVP 阶段显示 N/A，后续扩展实现"

  - name: slave_running
    display_name: "Slave 是否启动"
    query: ""
    category: replication
    status: pending
    note: "8.0 MGR 模式不适用，显示 N/A"
```

**验证**：

- [ ] YAML 文件格式正确，可被解析
- [ ] 所有巡检项都有对应的指标定义
- [ ] 待定项正确标记 `status: pending`

---

## 阶段二：MySQL 数据采集服务（步骤 5-8）

### 步骤 5：创建 MySQL 采集器接口

**操作**：

- 在 `internal/service/` 目录下创建 `mysql_collector.go` 文件
- 定义 `MySQLCollector` 结构体：
  - vmClient (\*vm.Client): VictoriaMetrics 客户端
  - config (\*config.MySQLInspectionConfig): MySQL 配置
  - metrics ([]\*model.MySQLMetricDefinition): 指标定义列表
  - logger (zerolog.Logger): 日志器
- 实现构造函数 `NewMySQLCollector`

**验证**：

- [ ] 执行 `go build ./internal/service/` 无编译错误

---

### 步骤 6：实现 MySQL 实例发现

**操作**：

- 在 `MySQLCollector` 中实现 `DiscoverInstances` 方法
- 查询 `mysql_up` 指标获取所有 MySQL 实例的 `address` 标签
- 解析 `address` 标签提取 IP 和端口
- 根据配置的 `InstanceFilter` 过滤实例
- 返回 `[]*model.MySQLInstance` 列表

**实现逻辑**：

```go
func (c *MySQLCollector) DiscoverInstances(ctx context.Context) ([]*model.MySQLInstance, error) {
    // 1. 查询 mysql_up 获取所有实例
    // 2. 从 address 标签解析 IP:Port
    // 3. 应用过滤规则
    // 4. 构建实例列表
}
```

**验证**：

- [ ] 能够正确发现所有 MySQL 实例
- [ ] IP 和端口解析正确
- [ ] 过滤规则正确应用
- [ ] 执行 `go test ./internal/service/ -run TestMySQLDiscoverInstances` 通过

---

### 步骤 7：实现 MySQL 指标采集

**操作**：

- 在 `MySQLCollector` 中实现 `CollectMetrics` 方法
- 按实例 `address` 标签进行过滤查询
- 采集所有配置的 MySQL 指标
- 处理标签提取（如从 `mysql_version_info` 提取 `version` 标签值）
- 返回 `map[string]*model.MySQLMetrics` (key 为 address)

**关键处理**：

- 从 `mysql_version_info` 指标的 `version` 标签提取版本号
- 从 `mysql_innodb_cluster_mgr_role_primary` 的 `member_id` 标签提取 Server ID
- 集群模式从配置读取（不自动检测）

**验证**：

- [ ] 能够采集所有 MySQL 指标
- [ ] 版本号正确提取
- [ ] Server ID 正确提取
- [ ] 执行 `go test ./internal/service/ -run TestMySQLCollectMetrics` 通过

---

### 步骤 8：编写 MySQL 采集器单元测试

**操作**：

- 在 `internal/service/mysql_collector_test.go` 中编写测试
- 使用 httptest 模拟 VictoriaMetrics API 响应
- 测试用例覆盖：
  - 正常发现 MySQL 实例
  - 正常采集所有指标
  - 地址解析（IP:Port 格式）
  - 版本标签提取
  - 实例过滤

**验证**：

- [ ] 执行 `go test ./internal/service/ -run TestMySQL` 全部通过
- [ ] 测试覆盖率达到 70% 以上

---

## 阶段三：MySQL 巡检评估服务（步骤 9-11）

### 步骤 9：实现 MySQL 状态评估器

**操作**：

- 在 `internal/service/` 目录下创建 `mysql_evaluator.go` 文件
- 定义 `MySQLEvaluator` 结构体
- 实现 `Evaluate` 方法：
  - 根据阈值配置评估各指标状态
  - 生成告警信息
  - 确定实例整体状态（取最严重级别）

**评估规则实现**：

```go
func (e *MySQLEvaluator) evaluateConnectionUsage(current, max int) AlertLevel {
    usage := float64(current) / float64(max) * 100
    if usage > e.thresholds.ConnectionUsageCritical {
        return AlertLevelCritical
    }
    if usage > e.thresholds.ConnectionUsageWarning {
        return AlertLevelWarning
    }
    return AlertLevelNormal
}

func (e *MySQLEvaluator) evaluateMGRMemberCount(count int) AlertLevel {
    expected := e.thresholds.MGRMemberCountExpected
    if count < expected - 1 {
        return AlertLevelCritical  // 掉 2 个及以上节点
    }
    if count < expected {
        return AlertLevelWarning   // 掉 1 个节点
    }
    return AlertLevelNormal        // 成员数达到或超过期望值
}

func (e *MySQLEvaluator) evaluateMGRStateOnline(online bool) AlertLevel {
    if !online {
        return AlertLevelCritical
    }
    return AlertLevelNormal
}
```

**验证**：

- [ ] 连接使用率 75% 被判定为警告
- [ ] 连接使用率 95% 被判定为严重
- [ ] MGR 成员数 = expected - 1 被判定为警告
- [ ] MGR 成员数 < expected - 1 被判定为严重
- [ ] MGR 节点离线被判定为严重
- [ ] 执行 `go test ./internal/service/ -run TestMySQLEvaluator` 通过

---

### 步骤 10：实现 MySQL 巡检编排服务

**操作**：

- 在 `internal/service/` 目录下创建 `mysql_inspector.go` 文件
- 定义 `MySQLInspector` 结构体，整合采集器和评估器
- 实现 `Inspect` 方法，协调完整巡检流程：
  1. 发现实例
  2. 采集指标
  3. 评估状态
  4. 汇总结果

**验证**：

- [ ] 能够完成端到端的 MySQL 巡检流程
- [ ] 单实例失败不影响其他实例
- [ ] 巡检摘要统计正确
- [ ] 执行 `go test ./internal/service/ -run TestMySQLInspector` 通过

---

### 步骤 11：编写 MySQL 巡检服务集成测试

**操作**：

- 在 `internal/service/mysql_inspector_test.go` 中编写集成测试
- 模拟完整的巡检场景：
  - 正常 MGR 集群（3 节点全部在线）
  - MGR 节点离线场景（1 节点掉线 → 警告）
  - MGR 多节点离线场景（2 节点掉线 → 严重）
  - 连接数过高场景
  - 多实例巡检

**验证**：

- [ ] 执行 `go test ./internal/service/ -run TestMySQLInspector` 全部通过
- [ ] 各场景告警逻辑正确

---

## 阶段四：报告生成扩展（步骤 12-15）

### 步骤 12：扩展 Excel 报告 - MySQL 工作表

**操作**：

- 在 `internal/report/excel/writer.go` 中添加 MySQL 报告功能
- 创建 "MySQL 巡检" 工作表（独立于 Host 巡检工作表）
- 表头：巡检时间、IP 地址、端口、数据库版本、Server ID、集群模式、同步状态、最大连接数、当前连接数、Binlog 状态、整体状态
- 应用条件格式（与主机巡检一致）

**验证**：

- [ ] Excel 文件包含 "MySQL 巡检" 工作表
- [ ] 表头完整正确
- [ ] 数据填充正确
- [ ] 条件格式正确应用

---

### 步骤 13：扩展 Excel 报告 - MySQL 异常汇总

**操作**：

- 新增 "MySQL 异常" 工作表（独立于 Host 异常汇总）
- 仅包含有告警的 MySQL 实例记录
- 显示告警类型、当前值、阈值

**验证**：

- [ ] 异常汇总表正确筛选告警记录
- [ ] 告警详情完整

---

### 步骤 14：扩展 HTML 报告 - MySQL 区域

**操作**：

- 在 HTML 模板中添加 MySQL 巡检区域（独立区域，位于 Host 巡检区域之后）
- 使用与主机巡检一致的卡片和表格样式
- 添加 MySQL 统计摘要卡片
- 实现表格排序功能

**验证**：

- [ ] HTML 报告正确显示 MySQL 巡检区域
- [ ] 样式与主机巡检区域一致
- [ ] 排序功能正常

---

### 步骤 15：更新示例配置文件

**操作**：

- 在 `configs/config.example.yaml` 中添加 MySQL 配置示例
- 添加详细注释说明各配置项

**配置示例**：

```yaml
# MySQL 数据库巡检配置
mysql:
  enabled: true
  # 集群模式（必须手动指定，不支持自动检测）
  # 可选值: mgr, dual-master, master-slave
  cluster_mode: "mgr"

  # 实例筛选（可选）
  instance_filter:
    address_patterns:
      - "172.18.182.*"
    business_groups:
      - "生产MySQL"
    tags:
      env: "prod"

  # 阈值配置
  thresholds:
    # 连接使用率阈值
    connection_usage_warning: 70   # 警告阈值 (%)
    connection_usage_critical: 90  # 严重阈值 (%)

    # MGR 期望成员数（默认 3，可配置为 5/7/9 等奇数）
    # 告警规则：
    #   - 成员数 = expected: 正常
    #   - 成员数 = expected - 1: 警告
    #   - 成员数 < expected - 1: 严重
    mgr_member_count_expected: 3
```

**验证**：

- [ ] 配置文件格式正确
- [ ] 配置可被正确加载和验证

---

## 阶段五：CLI 扩展与测试（步骤 16-18）

### 步骤 16：扩展 run 命令支持 MySQL 巡检

**操作**：

- 在 `cmd/inspect/cmd/run.go` 中集成 MySQL 巡检
- 添加 `--mysql-only` 标志（仅执行 MySQL 巡检）
- 添加 `--skip-mysql` 标志（跳过 MySQL 巡检）
- 在巡检流程中加入 MySQL 巡检步骤

**验证**：

- [ ] `./bin/inspect run -c config.yaml` 包含 MySQL 巡检
- [ ] `--mysql-only` 标志正确工作
- [ ] `--skip-mysql` 标志正确工作

---

### 步骤 17：端到端测试

**操作**：

- 使用**陕西营销活动环境**进行验证测试
- 配置真实 VictoriaMetrics 地址进行测试
- 验证 Excel 报告中 MySQL 数据正确性
- 验证 HTML 报告中 MySQL 区域显示正确
- 测试 MGR 集群场景

**验证**：

- [ ] MySQL 实例正确发现
- [ ] 版本、Server ID 正确提取
- [ ] MGR 状态正确判断
- [ ] 报告生成完整

---

### 步骤 18：更新文档

**操作**：

- 更新 README.md 添加 MySQL 巡检说明
- 添加 Categraf mysql.toml 配置参考
- 添加 MySQL 巡检项说明

**验证**：

- [ ] 文档完整描述 MySQL 巡检功能
- [ ] 配置示例可直接使用

---

## 验证计划

### 自动化测试

执行以下命令验证所有测试通过：

```bash
# 运行所有 MySQL 相关测试
go test ./internal/model/ -run TestMySQL -v
go test ./internal/service/ -run TestMySQL -v
go test ./internal/report/ -run TestMySQL -v

# 运行完整测试套件
go test ./... -v

# 检查测试覆盖率
go test ./internal/service/ -coverprofile=coverage.out
go tool cover -func=coverage.out | grep mysql
```

### 手动验证（使用陕西营销活动环境）

1. **配置验证**：

   ```bash
   ./bin/inspect validate -c config.yaml
   ```

   预期：配置验证通过，无错误信息

2. **MySQL 巡检执行**：

   ```bash
   ./bin/inspect run -c config.yaml --mysql-only
   ```

   预期：

   - 控制台显示发现的 MySQL 实例数量
   - 显示各实例的巡检状态
   - 生成包含 MySQL 工作表的 Excel 报告

3. **报告内容验证**：
   - 打开生成的 Excel 文件
   - 确认 "MySQL 巡检" 工作表存在
   - 确认数据与 VictoriaMetrics 查询结果一致

---

## 待定项说明

以下巡检项在 MVP 阶段显示 "N/A"，后续通过扩展 Categraf 采集配置实现：

| 巡检项           | MVP 处理                         | 后续实现方式               |
| ---------------- | -------------------------------- | -------------------------- |
| 是否普通用户启动 | 显示 N/A                         | 扩展 Categraf 自定义脚本   |
| Slave 是否启动   | MGR 模式显示 N/A（不适用）       | 5.7 主从模式扩展时实现     |

---

## 后续扩展计划

### P1: 5.7 主从/双主支持

- 添加 `mysql_slave_status_*` 指标采集
- 实现主从复制状态判断
- 增加 Seconds_Behind_Master 延迟告警

### P2: 更多巡检项

- 表空间使用情况
- 锁等待检测
- 长事务检测
- 性能 Schema 分析

---

## 版本记录

| 版本 | 日期       | 说明                                                                 |
| ---- | ---------- | -------------------------------------------------------------------- |
| v1.0 | 2025-12-15 | 初始版本，聚焦 MySQL 8.0 MGR 巡检                                    |
| v1.1 | 2025-12-15 | 根据澄清问题更新：明确 N/A 项、MGR 阈值变量化、移除 auto 模式、指定验证环境 |
