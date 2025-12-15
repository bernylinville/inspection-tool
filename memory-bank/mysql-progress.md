# MySQL 数据库巡检功能 - 开发进度记录

## 当前状态

**阶段**: 阶段二 - MySQL 数据采集服务（进行中）
**进度**: 步骤 5/18 完成

---

## 已完成步骤

### 步骤 1：定义 MySQL 实例模型 ✅

**完成日期**: 2025-12-15

**执行内容**:
1. 在 `internal/model/` 目录下创建 `mysql.go` 文件
2. 定义 `MySQLInstanceStatus` 枚举（normal/warning/critical/failed）
3. 定义 `MySQLClusterMode` 枚举（mgr/dual-master/master-slave）
4. 定义 `MySQLMGRRole` 枚举（PRIMARY/SECONDARY/UNKNOWN）
5. 定义 `MySQLInstance` 结构体，包含以下字段：
   - Address (string): 实例地址 (IP:Port)
   - IP (string): IP 地址
   - Port (int): 端口号
   - DatabaseType (string): 数据库类型，固定为 "MySQL"
   - Version (string): 数据库版本
   - InnoDBVersion (string): InnoDB 版本
   - ServerID (string): Server ID
   - ClusterMode (MySQLClusterMode): 集群模式
6. 实现辅助函数：
   - `ParseAddress()`: 解析 "IP:Port" 格式地址
   - `NewMySQLInstance()`: 从地址创建实例
   - `NewMySQLInstanceWithClusterMode()`: 带集群模式创建实例
7. 实现检验方法：
   - `IsHealthy()`, `IsWarning()`, `IsCritical()`, `IsFailed()` (MySQLInstanceStatus)
   - `IsMGR()`, `IsDualMaster()`, `IsMasterSlave()` (MySQLClusterMode)
   - `IsPrimary()`, `IsSecondary()` (MySQLMGRRole)
8. 实现设置和显示方法：
   - `SetVersion()`, `SetServerID()`, `String()`

**生成文件**:
- `internal/model/mysql.go` - MySQL 实例模型定义

**代码结构概览**:
```go
// 枚举类型
type MySQLInstanceStatus string  // normal, warning, critical, failed
type MySQLClusterMode string     // mgr, dual-master, master-slave
type MySQLMGRRole string         // PRIMARY, SECONDARY, UNKNOWN

// 核心结构体
type MySQLInstance struct {
    Address       string           `json:"address"`        // 实例地址 (IP:Port)
    IP            string           `json:"ip"`             // IP 地址
    Port          int              `json:"port"`           // 端口号
    DatabaseType  string           `json:"database_type"`  // 固定为 "MySQL"
    Version       string           `json:"version"`        // 数据库版本
    InnoDBVersion string           `json:"innodb_version"` // InnoDB 版本
    ServerID      string           `json:"server_id"`      // Server ID
    ClusterMode   MySQLClusterMode `json:"cluster_mode"`   // 集群模式
}

// 辅助函数
func ParseAddress(address string) (ip string, port int, err error)
func NewMySQLInstance(address string) *MySQLInstance
func NewMySQLInstanceWithClusterMode(address string, clusterMode MySQLClusterMode) *MySQLInstance
```

**验证结果**:
- [x] 执行 `go build ./internal/model/` 无编译错误
- [x] 执行 `go build ./...` 整个项目编译无错误
- [x] 结构体字段覆盖所有巡检项

---

### 步骤 2：定义 MySQL 巡检结果模型 ✅

**完成日期**: 2025-12-15

**执行内容**:
1. 在 `internal/model/mysql.go` 中添加 `MySQLAlert` 结构体
2. 在 `internal/model/mysql.go` 中添加 `MySQLInspectionResult` 结构体
3. 在 `internal/model/mysql.go` 中添加 `MySQLInspectionSummary` 结构体
4. 在 `internal/model/mysql.go` 中添加 `MySQLAlertSummary` 结构体
5. 在 `internal/model/mysql.go` 中添加 `MySQLInspectionResults` 结构体（集合）
6. 实现辅助方法

**新增结构体**:

```go
// MySQL 告警结构体
type MySQLAlert struct {
    Address           string     `json:"address"`             // 实例地址
    MetricName        string     `json:"metric_name"`         // 指标名称
    MetricDisplayName string     `json:"metric_display_name"` // 指标中文显示名称
    CurrentValue      float64    `json:"current_value"`       // 当前值
    FormattedValue    string     `json:"formatted_value"`     // 格式化后的当前值
    WarningThreshold  float64    `json:"warning_threshold"`   // 警告阈值
    CriticalThreshold float64    `json:"critical_threshold"`  // 严重阈值
    Level             AlertLevel `json:"level"`               // 复用 alert.go 的 AlertLevel
    Message           string     `json:"message"`             // 告警消息
}

// MySQL 单实例巡检结果
type MySQLInspectionResult struct {
    Instance            *MySQLInstance      // 实例元信息
    ConnectionStatus    bool                // mysql_up = 1
    SlaveRunning        bool                // Slave 线程状态（MGR 显示 N/A）
    SyncStatus          bool                // 同步是否正常
    SlowQueryLogEnabled bool                // 慢查询日志状态
    SlowQueryLogPath    string              // 慢查询日志路径
    MaxConnections      int                 // 最大连接数
    CurrentConnections  int                 // 当前连接数
    BinlogEnabled       bool                // Binlog 是否启用
    BinlogExpireSeconds int                 // Binlog 保留时长
    MGRMemberCount      int                 // MGR 成员数（仅 MGR 模式）
    MGRRole             MySQLMGRRole        // MGR 角色
    MGRStateOnline      bool                // MGR 节点是否在线
    NonRootUser         string              // MVP 阶段固定为 "N/A"
    Uptime              int64               // 运行时间（秒）
    Status              MySQLInstanceStatus // 整体状态
    Alerts              []*MySQLAlert       // 告警列表
    CollectedAt         time.Time           // 采集时间
    Error               string              // 错误信息
}

// MySQL 巡检摘要统计
type MySQLInspectionSummary struct {
    TotalInstances    int  // 实例总数
    NormalInstances   int  // 正常实例数
    WarningInstances  int  // 警告实例数
    CriticalInstances int  // 严重实例数
    FailedInstances   int  // 采集失败实例数
}

// MySQL 告警摘要统计
type MySQLAlertSummary struct {
    TotalAlerts   int  // 告警总数
    WarningCount  int  // 警告级别数量
    CriticalCount int  // 严重级别数量
}

// MySQL 巡检结果集合
type MySQLInspectionResults struct {
    InspectionTime time.Time               // 巡检开始时间
    Duration       time.Duration           // 巡检耗时
    Summary        *MySQLInspectionSummary // 摘要统计
    Results        []*MySQLInspectionResult// 实例结果列表
    Alerts         []*MySQLAlert           // 所有告警列表
    AlertSummary   *MySQLAlertSummary      // 告警摘要统计
    Version        string                  // 工具版本号
}
```

**新增辅助方法**:
- `NewMySQLAlert()` - 创建告警
- `MySQLAlert.IsWarning()`, `MySQLAlert.IsCritical()` - 告警级别判断
- `NewMySQLInspectionResult()` - 从实例创建巡检结果
- `MySQLInspectionResult.AddAlert()` - 添加告警并更新状态
- `MySQLInspectionResult.HasAlerts()` - 是否有告警
- `MySQLInspectionResult.GetConnectionUsagePercent()` - 计算连接使用率
- `MySQLInspectionResult.GetAddress()` - 获取实例地址
- `NewMySQLInspectionSummary()` - 从结果列表创建摘要
- `NewMySQLAlertSummary()` - 从告警列表创建摘要
- `NewMySQLInspectionResults()` - 创建结果集合
- `MySQLInspectionResults.AddResult()` - 添加实例结果
- `MySQLInspectionResults.Finalize()` - 完成巡检，计算摘要
- `MySQLInspectionResults.GetResultByAddress()` - 按地址查找结果
- `MySQLInspectionResults.GetCriticalResults()` - 获取严重状态结果
- `MySQLInspectionResults.GetWarningResults()` - 获取警告状态结果
- `MySQLInspectionResults.GetFailedResults()` - 获取失败状态结果
- `MySQLInspectionResults.HasCritical()`, `HasWarning()`, `HasAlerts()` - 状态判断

**验证结果**:
- [x] 执行 `go build ./internal/model/` 无编译错误
- [x] 执行 `go build ./...` 整个项目编译无错误
- [x] 结构体字段能承载所有巡检数据

---

### 步骤 3：扩展配置结构体 ✅

**完成日期**: 2025-12-15

**执行内容**:
1. 在 `internal/config/config.go` 中添加 `MySQLInspectionConfig` 结构体
2. 添加 `MySQLFilter` 实例筛选配置结构体
3. 添加 `MySQLThresholds` 阈值配置结构体
4. 在主配置 `Config` 结构体中添加 `MySQL` 字段

**新增结构体**:

```go
// MySQLInspectionConfig contains configurations for MySQL inspection.
type MySQLInspectionConfig struct {
    Enabled        bool            `mapstructure:"enabled"`
    ClusterMode    string          `mapstructure:"cluster_mode" validate:"omitempty,oneof=mgr dual-master master-slave"`
    InstanceFilter MySQLFilter     `mapstructure:"instance_filter"`
    Thresholds     MySQLThresholds `mapstructure:"thresholds"`
}

// MySQLFilter defines MySQL instance filtering criteria.
type MySQLFilter struct {
    AddressPatterns []string          `mapstructure:"address_patterns"` // Address matching patterns (e.g., "172.18.182.*")
    BusinessGroups  []string          `mapstructure:"business_groups"`  // Business groups (OR relation)
    Tags            map[string]string `mapstructure:"tags"`             // Tags (AND relation)
}

// MySQLThresholds contains threshold configurations for MySQL alerts.
type MySQLThresholds struct {
    ConnectionUsageWarning  float64 `mapstructure:"connection_usage_warning" validate:"gte=0,lte=100"`  // Default: 70
    ConnectionUsageCritical float64 `mapstructure:"connection_usage_critical" validate:"gte=0,lte=100"` // Default: 90
    MGRMemberCountExpected  int     `mapstructure:"mgr_member_count_expected" validate:"gte=1"`         // Default: 3
}
```

**配置结构变更**:

主配置 `Config` 中添加了 `MySQL` 字段：
```go
type Config struct {
    Datasources DatasourcesConfig     `mapstructure:"datasources" validate:"required"`
    Inspection  InspectionConfig      `mapstructure:"inspection"`
    Thresholds  ThresholdsConfig      `mapstructure:"thresholds"`
    Report      ReportConfig          `mapstructure:"report"`
    Logging     LoggingConfig         `mapstructure:"logging"`
    HTTP        HTTPConfig            `mapstructure:"http"`
    MySQL       MySQLInspectionConfig `mapstructure:"mysql"` // 新增
}
```

**阈值说明**:

| 配置项 | 默认值 | 说明 |
|--------|--------|------|
| `connection_usage_warning` | 70 | 连接使用率警告阈值 (%) |
| `connection_usage_critical` | 90 | 连接使用率严重阈值 (%) |
| `mgr_member_count_expected` | 3 | MGR 期望成员数 |

**验证结果**:
- [x] 执行 `go build ./internal/config/` 无编译错误
- [x] 执行 `go build ./...` 整个项目编译无错误

---

### 步骤 4：创建 MySQL 指标定义文件 ✅

**完成日期**: 2025-12-15

**执行内容**:
1. 在 `configs/` 目录下创建 `mysql-metrics.yaml` 文件
2. 定义 16 个 MySQL 巡检指标的 PromQL 查询表达式
3. 使用与 `metrics.yaml` 相同的 YAML 格式

**指标分类**:

| 分类 | 指标数量 | 说明 |
|------|----------|------|
| connection | 3 | mysql_up, max_connections, current_connections |
| info | 3 | mysql_version, server_id |
| mgr | 3 | mgr_member_count, mgr_role_primary, mgr_state_online |
| binlog | 3 | binlog_file_count, binlog_size, binlog_expire_seconds |
| log | 2 | slow_query_log, slow_query_log_file |
| status | 1 | uptime |
| pending | 2 | non_root_user, slave_running (MVP 显示 N/A) |

**指标定义结构**:
```yaml
mysql_metrics:
  - name: mysql_up
    display_name: "连接状态"
    query: "mysql_up"
    category: connection
    note: "1=正常, 0=连接失败"

  - name: mgr_member_count
    display_name: "MGR 成员数"
    query: "mysql_innodb_cluster_mgr_member_count"
    category: mgr
    cluster_mode: mgr
    note: "MGR 集群在线成员数量，默认期望值为 3"
```

**生成文件**:
- `configs/mysql-metrics.yaml` - MySQL 指标定义文件（16 个指标）

**验证结果**:
- [x] YAML 文件格式验证通过（python yaml.safe_load）
- [x] 所有巡检项都有对应的指标定义
- [x] 待定项正确标记 `status: pending`
- [x] 与 `mysql-feature-implementation.md` 中的指标定义一致

---

### 步骤 5：创建 MySQL 采集器接口 ✅

**完成日期**: 2025-12-15

**执行内容**:
1. 在 `internal/model/mysql.go` 中添加 `MySQLMetricDefinition` 结构体
2. 在 `internal/service/` 目录下创建 `mysql_collector.go` 文件
3. 定义 `MySQLCollector` 结构体
4. 定义 `MySQLInstanceFilter` 结构体
5. 实现构造函数 `NewMySQLCollector`
6. 实现辅助方法

**新增代码 - internal/model/mysql.go**:

```go
// MySQLMetricDefinition defines a MySQL metric to be collected.
type MySQLMetricDefinition struct {
    Name         string `yaml:"name"`          // 指标唯一标识
    DisplayName  string `yaml:"display_name"`  // 中文显示名称
    Query        string `yaml:"query"`         // PromQL 查询表达式
    Category     string `yaml:"category"`      // 分类
    ClusterMode  string `yaml:"cluster_mode"`  // 适用的集群模式（可选）
    LabelExtract string `yaml:"label_extract"` // 从指标标签提取值（可选）
    Format       string `yaml:"format"`        // 格式化类型（可选）
    Status       string `yaml:"status"`        // 状态（pending=待实现）
    Note         string `yaml:"note"`          // 备注说明
}

// 辅助方法
func (m *MySQLMetricDefinition) IsPending() bool
func (m *MySQLMetricDefinition) HasLabelExtract() bool
func (m *MySQLMetricDefinition) IsForClusterMode(mode MySQLClusterMode) bool
func (m *MySQLMetricDefinition) GetDisplayName() string
```

**新增文件 - internal/service/mysql_collector.go**:

```go
// MySQLCollector is the data collection service for MySQL instances.
type MySQLCollector struct {
    vmClient       *vm.Client
    config         *config.MySQLInspectionConfig
    metrics        []*model.MySQLMetricDefinition
    instanceFilter *MySQLInstanceFilter
    logger         zerolog.Logger
}

// MySQLInstanceFilter defines filtering criteria for MySQL instances.
type MySQLInstanceFilter struct {
    AddressPatterns []string          // Address patterns (e.g., "172.18.182.*")
    BusinessGroups  []string          // Business groups (OR relation)
    Tags            map[string]string // Tags (AND relation)
}

// 构造函数和辅助方法
func NewMySQLCollector(cfg, vmClient, metrics, logger) *MySQLCollector
func (c *MySQLCollector) buildInstanceFilter() *MySQLInstanceFilter
func (c *MySQLCollector) GetConfig() *config.MySQLInspectionConfig
func (c *MySQLCollector) GetMetrics() []*model.MySQLMetricDefinition
func (c *MySQLCollector) GetInstanceFilter() *MySQLInstanceFilter
func (f *MySQLInstanceFilter) IsEmpty() bool
func (f *MySQLInstanceFilter) ToVMHostFilter() *vm.HostFilter
```

**生成文件**:
- `internal/service/mysql_collector.go` - MySQL 采集器结构体和构造函数

**验证结果**:
- [x] 执行 `go build ./internal/model/` 无编译错误
- [x] 执行 `go build ./internal/service/` 无编译错误
- [x] 执行 `go build ./...` 整个项目编译无错误
- [x] 代码风格与现有 Collector 一致

---

## 下一步骤

**步骤 6：实现 MySQL 实例发现**（等待用户验证步骤 5）

待实现内容：
- 在 `MySQLCollector` 中实现 `DiscoverInstances` 方法
- 查询 `mysql_up` 指标获取所有 MySQL 实例的 `address` 标签
- 解析 `address` 标签提取 IP 和端口
- 根据配置的 `InstanceFilter` 过滤实例
- 返回 `[]*model.MySQLInstance` 列表

⚠️ **注意**：等待用户验证测试步骤 5 后再开始步骤 6

---

## 版本记录

| 日期 | 步骤 | 说明 |
|------|------|------|
| 2025-12-15 | 步骤 1 | 定义 MySQL 实例模型完成 |
| 2025-12-15 | 步骤 2 | 定义 MySQL 巡检结果模型完成 |
| 2025-12-15 | 步骤 3 | 扩展配置结构体完成 |
| 2025-12-15 | 步骤 4 | 创建 MySQL 指标定义文件完成，阶段一全部完成 |
| 2025-12-15 | 步骤 5 | 创建 MySQL 采集器接口完成，阶段二开始 |
