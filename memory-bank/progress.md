# 系统巡检工具 - 开发进度记录

## 当前状态

**阶段**: 阶段六 - 报告生成模块（进行中）
**进度**: 步骤 24/41 完成

---

## 已完成步骤

### 步骤 1：初始化 Go 模块 ✅

**完成日期**: 2025-12-13

**执行内容**:
1. 执行 `go mod init inspection-tool` 初始化 Go 模块
2. 确认 Go 版本为 1.25.5（符合 1.25 系列要求）
3. 执行 `go mod tidy` 验证无报错

**生成文件**:
- `go.mod` - Go 模块定义文件

**验证结果**:
- [x] `go.mod` 文件存在且包含正确的模块名
- [x] Go 版本设置为 1.25.5
- [x] `go mod tidy` 执行无错误

---

### 步骤 2：创建目录结构 ✅

**完成日期**: 2025-12-13

**执行内容**:
1. 创建 `cmd/inspect/` - 程序入口目录
2. 创建 `internal/config/` - 配置管理
3. 创建 `internal/client/n9e/` - 夜莺 API 客户端
4. 创建 `internal/client/vm/` - VictoriaMetrics 客户端
5. 创建 `internal/model/` - 数据模型
6. 创建 `internal/service/` - 业务逻辑
7. 创建 `internal/report/excel/` - Excel 报告生成
8. 创建 `internal/report/html/` - HTML 报告生成
9. 创建 `configs/` - 配置文件示例
10. 创建 `templates/html/` - 用户自定义 HTML 模板（外置）
11. 为所有空目录添加 `.gitkeep` 文件

**生成目录结构**:
```
inspection-tool/
├── cmd/inspect/              # 程序入口
├── configs/                  # 配置文件示例
├── internal/
│   ├── client/
│   │   ├── n9e/              # 夜莺 API 客户端
│   │   └── vm/               # VictoriaMetrics 客户端
│   ├── config/               # 配置管理
│   ├── model/                # 数据模型
│   ├── report/
│   │   ├── excel/            # Excel 报告生成
│   │   └── html/             # HTML 报告生成
│   └── service/              # 业务逻辑
└── templates/html/           # 用户自定义 HTML 模板
```

**验证结果**:
- [x] 所有目录已创建
- [x] 使用 `tree` 命令确认目录结构完整
- [x] 所有空目录包含 `.gitkeep` 文件

---

### 步骤 3：添加核心依赖 ✅

**完成日期**: 2025-12-13

**执行内容**:
1. 添加 CLI 框架：`github.com/spf13/cobra v1.10.2`
2. 添加配置管理：`github.com/spf13/viper v1.21.0`
3. 添加 HTTP 客户端：`github.com/go-resty/resty/v2 v2.17.0`
4. 添加 Excel 生成：`github.com/xuri/excelize/v2 v2.10.0`
5. 添加结构化日志：`github.com/rs/zerolog v1.34.0`
6. 添加并发控制：`golang.org/x/sync v0.19.0`
7. 添加数据验证：`github.com/go-playground/validator/v10 v10.29.0`

**自动添加的传递依赖**:
- `github.com/fsnotify/fsnotify` - 文件系统监听（viper 依赖）
- `github.com/go-viper/mapstructure/v2` - 结构体映射（viper 依赖）
- `github.com/spf13/pflag` - 命令行标志（cobra 依赖）
- `golang.org/x/crypto` - 加密库（validator 依赖）
- `golang.org/x/net` - 网络扩展（resty 依赖）
- 等 20+ 个传递依赖

**验证结果**:
- [x] `go.mod` 中包含所有 7 个核心依赖
- [x] `go mod download` 执行成功
- [x] `go mod verify` 输出 "all modules verified"

---

### 步骤 4：创建程序入口文件 ✅

**完成日期**: 2025-12-13

**执行内容**:
1. 在 `cmd/inspect/` 目录下创建 `main.go`
2. 实现版本信息输出功能
3. 支持通过 `-ldflags` 注入版本号、构建时间、Git 提交哈希
4. 显示 Go 版本和运行平台信息

**生成文件**:
- `cmd/inspect/main.go` - 程序入口文件
- `bin/inspect` - 编译后的二进制文件

**代码结构**:
```go
var (
    Version   = "dev"      // 通过 -ldflags 注入
    BuildTime = "unknown"  // 通过 -ldflags 注入
    GitCommit = "unknown"  // 通过 -ldflags 注入
)
```

**验证结果**:
- [x] `go build -o bin/inspect ./cmd/inspect` 执行成功
- [x] 运行 `./bin/inspect` 输出版本信息
- [x] 二进制文件大小 2.2MB（小于 20MB 限制）
- [x] `-ldflags` 版本注入功能正常

**运行输出示例**:
```
Inspection Tool dev
Build Time: unknown
Git Commit: unknown
Go Version: go1.25.5
OS/Arch: linux/amd64
```

---

### 步骤 5：定义配置结构体 ✅

**完成日期**: 2025-12-13

**执行内容**:
1. 在 `internal/config/config.go` 中定义完整的配置结构体
2. 定义数据源配置（N9E、VictoriaMetrics）
3. 定义巡检配置（并发数、超时、主机筛选）
4. 定义阈值配置（CPU、内存、磁盘、僵尸进程、负载）
5. 定义报告配置（输出目录、格式、模板、时区）
6. 定义日志配置（级别、格式）
7. 定义 HTTP 重试配置（重试次数、延迟）
8. 为所有字段添加 `mapstructure` 和 `validate` 标签

**生成文件**:
- `internal/config/config.go` - 配置结构体定义

**配置结构概览**:
```go
type Config struct {
    Datasources DatasourcesConfig  // N9E + VM 数据源
    Inspection  InspectionConfig   // 并发、超时、筛选
    Thresholds  ThresholdsConfig   // 告警阈值
    Report      ReportConfig       // 报告输出设置
    Logging     LoggingConfig      // 日志配置
    HTTP        HTTPConfig         // 重试策略
}
```

**验证结果**:
- [x] 结构体定义完整，字段命名符合 Go 规范
- [x] 所有必填字段都有 `validate:"required"` 标签
- [x] 执行 `go build ./internal/config/` 无编译错误

---

### 步骤 6：实现配置加载器 ✅

**完成日期**: 2025-12-13

**执行内容**:
1. 在 `internal/config/loader.go` 中实现配置加载功能
2. 使用 viper 库读取 YAML 配置文件
3. 支持环境变量覆盖（前缀 `INSPECT_`，如 `INSPECT_DATASOURCES_N9E_TOKEN`）
4. 实现所有配置项的默认值设置
5. 编写单元测试验证功能

**生成文件**:
- `internal/config/loader.go` - 配置加载器实现
- `internal/config/loader_test.go` - 配置加载器单元测试

**核心函数**:
```go
func Load(configPath string) (*Config, error)  // 加载配置文件
func setDefaults(v *viper.Viper)               // 设置默认值
```

**默认值配置**:
| 配置项 | 默认值 |
|--------|--------|
| `inspection.concurrency` | 20 |
| `inspection.host_timeout` | 10s |
| `http.retry.max_retries` | 3 |
| `http.retry.base_delay` | 1s |
| `report.timezone` | Asia/Shanghai |
| `report.output_dir` | ./reports |
| `report.formats` | [excel, html] |
| `logging.level` | info |
| `logging.format` | json |
| `datasources.*.timeout` | 30s |
| `thresholds.cpu_usage` | warning=70, critical=90 |
| `thresholds.memory_usage` | warning=70, critical=90 |
| `thresholds.disk_usage` | warning=70, critical=90 |

**验证结果**:
- [x] 能够正确加载 YAML 配置文件
- [x] 环境变量能够覆盖配置文件中的值
- [x] 缺少配置文件时返回明确错误信息
- [x] 默认值正确应用
- [x] 执行 `go build ./internal/config/` 无编译错误
- [x] 执行 `go test ./internal/config/` 全部通过（4 个测试用例）

---

### 步骤 7：实现配置验证器 ✅

**完成日期**: 2025-12-13

**执行内容**:
1. 在 `internal/config/validator.go` 中实现配置验证功能
2. 使用 `go-playground/validator/v10` 库执行结构体验证
3. 实现自定义验证规则：
   - `threshold_order`：验证 warning < critical
   - `timezone`：验证时区字符串有效性
4. 实现友好的错误消息翻译
5. 将验证集成到 `Load()` 函数
6. 编写完整的单元测试

**生成文件**:
- `internal/config/validator.go` - 配置验证器实现
- `internal/config/validator_test.go` - 配置验证器单元测试

**核心类型**:
```go
type ValidationError struct {
    Field   string      // 字段路径 (e.g., "datasources.n9e.endpoint")
    Tag     string      // 验证标签 (e.g., "required", "url")
    Value   interface{} // 实际值
    Message string      // 友好错误信息
}

type ValidationErrors []*ValidationError
```

**验证覆盖**:
| 验证类型 | 说明 |
|----------|------|
| 必填字段 | N9E endpoint、token；VM endpoint |
| URL 格式 | endpoint 字段必须是有效 URL |
| 数值范围 | concurrency 1-100；阈值 ≥ 0；重试次数 0-10 |
| 枚举值 | 报告格式 excel/html；日志级别 debug/info/warn/error |
| 阈值逻辑 | warning 必须小于 critical |
| 时区有效性 | 验证时区字符串可被解析 |

**验证结果**:
- [x] 缺少必填字段时返回具体的字段名和错误原因
- [x] 无效 URL 格式能被检测出来
- [x] 阈值数值超出合理范围时报错
- [x] 执行 `go test ./internal/config/` 全部通过（26 个测试用例）
- [x] 测试覆盖率 88.4%（超过目标 80%）
- [x] 执行 `go build ./cmd/inspect` 无编译错误

---

### 步骤 8：创建示例配置文件 ✅

**完成日期**: 2025-12-13

**执行内容**:
1. 在 `configs/` 目录下创建 `config.example.yaml`
2. 包含所有 6 个配置节（datasources, inspection, thresholds, report, logging, http）
3. 使用产品需求文档中的实际 API 地址作为示例
4. 所有配置项都有详细的中文注释说明
5. Token 使用环境变量占位符 `${N9E_TOKEN}`
6. 包含环境变量覆盖使用说明

**生成文件**:
- `configs/config.example.yaml` - 示例配置文件

**配置文件特性**:
| 特性 | 说明 |
|------|------|
| 完整性 | 覆盖所有配置项，与 config.go 结构体完全匹配 |
| 中文注释 | 每个配置节和关键配置项都有中文说明 |
| 环境变量 | Token 使用占位符，支持 `INSPECT_DATASOURCES_N9E_TOKEN` 覆盖 |
| 默认值 | 与 loader.go 中的 setDefaults() 保持一致 |
| 实际地址 | 使用产品需求文档中的 N9E 和 VM 地址 |

**验证结果**:
- [x] YAML 格式正确（Python yaml.safe_load 验证通过）
- [x] 配置加载成功（使用 config.Load() 验证通过）
- [x] 环境变量覆盖功能正常（`INSPECT_DATASOURCES_N9E_TOKEN` 测试通过）
- [x] 所有配置项都有中文注释说明

---

### 步骤 9：创建指标定义文件 ✅

**完成日期**: 2025-12-13

**执行内容**:
1. 在 `configs/` 目录下创建 `metrics.yaml` 指标定义文件
2. 定义所有巡检指标的 PromQL 查询表达式
3. 包含指标名称、显示名称、查询语句、单位、分类、格式化类型
4. 标记待定巡检项（status: pending）
5. 为特殊指标添加聚合方式和标签展开配置

**生成文件**:
- `configs/metrics.yaml` - 指标定义文件

**指标统计**:
| 分类 | 数量 | 说明 |
|------|------|------|
| CPU | 1 | cpu_usage |
| 内存 | 4 | memory_usage, total, free, available |
| 磁盘 | 3 | disk_usage, total, free（按挂载点展开） |
| 系统 | 6 | uptime, cpu_cores, load_1m/5m/15m, load_per_core |
| 进程 | 2 | processes_total, zombies |
| 待定项 | 7 | ntp_check, public_network, password_expiry 等 |
| **总计** | **23** | |

**指标字段结构**:
```yaml
- name: string           # 指标唯一标识
  display_name: string   # 中文显示名称
  query: string          # PromQL 查询表达式
  unit: string           # 单位（%、bytes、seconds 等）
  category: string       # 分类（cpu、memory、disk、system、process）
  format: string         # 格式化类型（size、duration、percent）
  aggregate: string      # 聚合方式（max、min、avg）
  expand_by_label: string # 按标签展开（如 path）
  status: string         # 待定项标记为 pending
  note: string           # 备注说明
```

**验证结果**:
- [x] YAML 格式正确（Python yaml.safe_load 验证通过）
- [x] 包含产品需求文档中定义的所有指标（23 个）
- [x] 每个指标都有完整的元数据定义
- [x] 待定项正确标记 `status: pending`（7 个）
- [x] PromQL 表达式与 `categraf-linux-metrics.json` 一致
- [x] 中文注释清晰完整

---

### 步骤 10：定义主机模型 ✅

**完成日期**: 2025-12-13

**执行内容**:
1. 在 `internal/model/host.go` 中定义主机相关结构体
2. 定义 `HostStatus` 主机状态枚举（normal/warning/critical/failed）
3. 定义 `DiskMountInfo` 磁盘挂载点信息结构体
4. 定义 `HostMeta` 主机元信息结构体
5. 实现 `CleanIdent()` 辅助函数（处理 `hostname@IP` 格式）

**生成文件**:
- `internal/model/host.go` - 主机模型定义

**结构体定义**:
```go
// 主机状态枚举
type HostStatus string
const (
    HostStatusNormal   HostStatus = "normal"
    HostStatusWarning  HostStatus = "warning"
    HostStatusCritical HostStatus = "critical"
    HostStatusFailed   HostStatus = "failed"
)

// 磁盘挂载点信息
type DiskMountInfo struct {
    Path        string  `json:"path"`
    Total       int64   `json:"total"`
    Free        int64   `json:"free"`
    UsedPercent float64 `json:"used_percent"`
}

// 主机元信息
type HostMeta struct {
    Ident         string          `json:"ident"`
    Hostname      string          `json:"hostname"`
    IP            string          `json:"ip"`
    OS            string          `json:"os"`
    OSVersion     string          `json:"os_version"`
    KernelVersion string          `json:"kernel_version"`
    CPUCores      int             `json:"cpu_cores"`
    CPUModel      string          `json:"cpu_model"`
    MemoryTotal   int64           `json:"memory_total"`
    DiskMounts    []DiskMountInfo `json:"disk_mounts"`
}
```

**验证结果**:
- [x] 结构体字段覆盖产品需求文档中的所有主机属性
- [x] JSON 标签正确设置（小写、下划线分隔）
- [x] 执行 `go build ./internal/model/` 无编译错误
- [x] 包含 package 级和 type 级注释
- [x] 代码风格与 `internal/config/config.go` 一致

---

### 步骤 11：定义指标模型 ✅

**完成日期**: 2025-12-13

**执行内容**:
1. 在 `internal/model/metric.go` 中定义指标相关结构体
2. 定义 `MetricStatus` 指标状态枚举（normal/warning/critical/pending）
3. 定义 `MetricCategory` 分类枚举（cpu/memory/disk/system/process）
4. 定义 `MetricFormat` 格式化类型枚举（percent/size/duration/number）
5. 定义 `AggregateType` 聚合方式枚举（max/min/avg）
6. 定义 `MetricDefinition` 指标定义结构体（对应 metrics.yaml）
7. 定义 `MetricValue` 指标值结构体（采集后的数据）
8. 定义 `HostMetrics` 主机指标集合结构体
9. 定义 `MetricsConfig` 用于加载 metrics.yaml
10. 实现辅助函数：`IsPending()`、`HasExpandLabel()`、`NewNAMetricValue()`、`NewMetricValue()`

**生成文件**:
- `internal/model/metric.go` - 指标模型定义

**结构体定义**:
```go
// 指标状态枚举
type MetricStatus string
const (
    MetricStatusNormal   MetricStatus = "normal"
    MetricStatusWarning  MetricStatus = "warning"
    MetricStatusCritical MetricStatus = "critical"
    MetricStatusPending  MetricStatus = "pending"
)

// 指标定义（从 metrics.yaml 加载）
type MetricDefinition struct {
    Name          string         `yaml:"name"`
    DisplayName   string         `yaml:"display_name"`
    Query         string         `yaml:"query"`
    Unit          string         `yaml:"unit"`
    Category      MetricCategory `yaml:"category"`
    Format        MetricFormat   `yaml:"format,omitempty"`
    Aggregate     AggregateType  `yaml:"aggregate,omitempty"`
    ExpandByLabel string         `yaml:"expand_by_label,omitempty"`
    Status        string         `yaml:"status,omitempty"`
    Note          string         `yaml:"note,omitempty"`
}

// 指标采集值
type MetricValue struct {
    Name           string            `json:"name"`
    RawValue       float64           `json:"raw_value"`
    FormattedValue string            `json:"formatted_value"`
    Status         MetricStatus      `json:"status"`
    Labels         map[string]string `json:"labels,omitempty"`
    IsNA           bool              `json:"is_na"`
    Timestamp      int64             `json:"timestamp,omitempty"`
}

// 主机指标集合
type HostMetrics struct {
    Hostname string                  `json:"hostname"`
    Metrics  map[string]*MetricValue `json:"metrics"`
}
```

**验证结果**:
- [x] 支持不同类型的指标值（百分比、字节、秒数等）
- [x] 包含格式化显示所需的所有字段
- [x] 支持待定项的 N/A 显示（`NewNAMetricValue()` 函数）
- [x] 执行 `go build ./internal/model/` 无编译错误
- [x] 包含 package 级和 type 级注释
- [x] 代码风格与 `host.go` 一致

---

### 步骤 12：定义告警模型 ✅

**完成日期**: 2025-12-13

**执行内容**:
1. 在 `internal/model/alert.go` 中定义告警相关结构体
2. 定义 `AlertLevel` 告警级别枚举（normal/warning/critical）
3. 定义 `Alert` 告警详情结构体
4. 定义 `AlertSummary` 告警摘要结构体
5. 实现辅助函数：`NewAlert()`、`IsWarning()`、`IsCritical()`、`NewAlertSummary()`

**生成文件**:
- `internal/model/alert.go` - 告警模型定义

**结构体定义**:
```go
// 告警级别枚举
type AlertLevel string
const (
    AlertLevelNormal   AlertLevel = "normal"
    AlertLevelWarning  AlertLevel = "warning"
    AlertLevelCritical AlertLevel = "critical"
)

// 告警详情
type Alert struct {
    Hostname          string            `json:"hostname"`
    MetricName        string            `json:"metric_name"`
    MetricDisplayName string            `json:"metric_display_name"`
    CurrentValue      float64           `json:"current_value"`
    FormattedValue    string            `json:"formatted_value"`
    WarningThreshold  float64           `json:"warning_threshold"`
    CriticalThreshold float64           `json:"critical_threshold"`
    Level             AlertLevel        `json:"level"`
    Message           string            `json:"message"`
    Labels            map[string]string `json:"labels,omitempty"`
}

// 告警摘要
type AlertSummary struct {
    TotalAlerts   int `json:"total_alerts"`
    WarningCount  int `json:"warning_count"`
    CriticalCount int `json:"critical_count"`
}
```

**验证结果**:
- [x] 告警级别定义完整（Normal、Warning、Critical）
- [x] 告警信息包含定位问题所需的所有字段
- [x] 执行 `go build ./internal/model/` 无编译错误
- [x] 代码风格与 `host.go`、`metric.go` 一致
- [x] 包含 package 级和 type 级注释

---

### 步骤 13：定义巡检结果模型 ✅

**完成日期**: 2025-12-13

**执行内容**:
1. 在 `internal/model/inspection.go` 中定义巡检结果相关结构体
2. 定义 `InspectionSummary` 巡检摘要结构体（主机统计）
3. 定义 `HostResult` 单主机巡检结果结构体
4. 定义 `InspectionResult` 完整巡检结果结构体
5. 实现辅助函数：`NewInspectionSummary()`、`NewHostResult()`、`NewInspectionResult()`
6. 实现辅助方法：`SetMetric()`、`GetMetric()`、`AddAlert()`、`AddHost()`、`Finalize()`
7. 实现查询方法：`GetCriticalHosts()`、`GetWarningHosts()`、`GetFailedHosts()`、`HasCritical()`、`HasWarning()`、`HasAlerts()`

**生成文件**:
- `internal/model/inspection.go` - 巡检结果模型定义

**结构体定义**:
```go
// 巡检摘要
type InspectionSummary struct {
    TotalHosts    int `json:"total_hosts"`    // 主机总数
    NormalHosts   int `json:"normal_hosts"`   // 正常主机数
    WarningHosts  int `json:"warning_hosts"`  // 警告主机数
    CriticalHosts int `json:"critical_hosts"` // 严重主机数
    FailedHosts   int `json:"failed_hosts"`   // 采集失败主机数
}

// 单主机巡检结果
type HostResult struct {
    Hostname      string                  `json:"hostname"`        // 主机名
    IP            string                  `json:"ip"`              // IP 地址
    OS            string                  `json:"os"`              // 操作系统类型
    OSVersion     string                  `json:"os_version"`      // 操作系统版本
    KernelVersion string                  `json:"kernel_version"`  // 内核版本
    CPUCores      int                     `json:"cpu_cores"`       // CPU 核心数
    CPUModel      string                  `json:"cpu_model"`       // CPU 型号
    MemoryTotal   int64                   `json:"memory_total"`    // 内存总量
    Status        HostStatus              `json:"status"`          // 整体状态
    Metrics       map[string]*MetricValue `json:"metrics"`         // 指标集合
    Alerts        []*Alert                `json:"alerts"`          // 告警列表
    CollectedAt   time.Time               `json:"collected_at"`    // 采集时间
    Error         string                  `json:"error,omitempty"` // 错误信息
}

// 完整巡检结果
type InspectionResult struct {
    InspectionTime time.Time          `json:"inspection_time"` // 巡检开始时间
    Duration       time.Duration      `json:"duration"`        // 巡检耗时
    Summary        *InspectionSummary `json:"summary"`         // 摘要统计
    Hosts          []*HostResult      `json:"hosts"`           // 主机结果列表
    Alerts         []*Alert           `json:"alerts"`          // 所有告警列表
    AlertSummary   *AlertSummary      `json:"alert_summary"`   // 告警摘要
    Version        string             `json:"version"`         // 工具版本号
}
```

**验证结果**:
- [x] 结构体能够承载完整的巡检数据
- [x] 支持 JSON 序列化（所有字段都有 json 标签）
- [x] 时间字段使用 `time.Time` 类型（运行时设置 Asia/Shanghai 时区）
- [x] 执行 `go build ./internal/model/` 无编译错误
- [x] 执行 `go build ./...` 整个项目编译无错误
- [x] 代码风格与 `host.go`、`metric.go`、`alert.go` 一致
- [x] 包含 package 级和 type 级注释

---

### 步骤 14：定义 N9E 客户端接口和类型 ✅

**完成日期**: 2025-12-13

**执行内容**:
1. 在 `internal/client/n9e/types.go` 中定义夜莺 API 请求和响应类型
2. 定义 `TargetResponse` 和 `TargetsResponse` API 响应结构体
3. 定义 `ExtendInfo` 结构体解析嵌套 JSON 字符串
4. 定义子结构体：`CPUInfo`、`MemoryInfo`、`NetworkInfo`、`PlatformInfo`、`FilesystemInfo`
5. 实现辅助函数：`ParseExtendInfo()`、`ToHostMeta()`
6. 实现类型转换方法：`GetCPUCores()`、`GetTotal()`、`GetSizeBytes()`、`IsPhysicalDisk()`
7. 编写完整的单元测试（12 个测试用例）

**生成文件**:
- `internal/client/n9e/types.go` - N9E API 类型定义
- `internal/client/n9e/types_test.go` - 单元测试

**类型定义概览**:
```go
// API 响应
type TargetResponse struct {
    Dat TargetData `json:"dat"`
    Err string     `json:"err"`
}

type TargetData struct {
    Ident      string `json:"ident"`
    ExtendInfo string `json:"extend_info"` // JSON 字符串，需要二次解析
}

// ExtendInfo 解析结构
type ExtendInfo struct {
    CPU        CPUInfo           `json:"cpu"`
    Memory     MemoryInfo        `json:"memory"`
    Network    NetworkInfo       `json:"network"`
    Platform   PlatformInfo      `json:"platform"`
    Filesystem []FilesystemInfo  `json:"filesystem"`
}

// 平台信息
type PlatformInfo struct {
    Hostname      string `json:"hostname"`
    OS            string `json:"os"`
    KernelRelease string `json:"kernel_release"`
    // ... 更多字段
}
```

**关键功能**:
| 功能 | 说明 |
|------|------|
| `ParseExtendInfo()` | 解析 extend_info JSON 字符串 |
| `ToHostMeta()` | 将 N9E 数据转换为内部 HostMeta 模型 |
| `IsPhysicalDisk()` | 过滤物理磁盘，排除 tmpfs、overlay、containerd 等 |
| `GetCPUCores()` | 从字符串解析 CPU 核心数 |
| `GetTotal()` | 从字符串解析内存总量 |

**验证结果**:
- [x] 类型定义与夜莺 API 响应格式匹配（使用 `n9e.json` 验证）
- [x] `ExtendInfo` 能够正确解析嵌套 JSON 字符串
- [x] 包含所有需要的字段（主机名、IP、OS、版本、CPU、内存等）
- [x] 执行 `go build ./internal/client/n9e/` 无编译错误
- [x] 执行 `go build ./...` 整个项目编译无错误
- [x] 执行 `go test ./internal/client/n9e/` 全部通过（12 个测试用例）
- [x] 测试覆盖率 82.2%（超过目标 70%）
- [x] 物理磁盘过滤正确排除 tmpfs、overlay、containerd 挂载点

---

### 步骤 15：实现 N9E 客户端 ✅

**完成日期**: 2025-12-13

**执行内容**:
1. 在 `internal/client/n9e/client.go` 中实现 N9E API 客户端
2. 实现 `NewClient()` 构造函数，接收配置参数
3. 实现 `GetTargets()` 获取所有主机列表方法
4. 实现 `GetTarget()` 获取单主机详情方法
5. 实现 `GetHostMetas()` 获取主机元信息便捷方法
6. 实现 `GetHostMetaByIdent()` 获取单主机元信息方法
7. 实现 Token 认证（X-User-Token 请求头）
8. 实现超时和错误处理
9. 集成重试机制（使用 resty 内置指数退避）

**生成文件**:
- `internal/client/n9e/client.go` - N9E API 客户端实现

**客户端结构**:
```go
type Client struct {
    endpoint   string             // N9E API 地址
    token      string             // 认证 Token
    timeout    time.Duration      // 请求超时
    retry      config.RetryConfig // 重试配置
    httpClient *resty.Client      // HTTP 客户端
    logger     zerolog.Logger     // 日志记录器
}
```

**核心方法**:
| 方法 | 功能 |
|------|------|
| `NewClient()` | 创建客户端，配置认证头和重试策略 |
| `GetTargets()` | 获取所有主机基本信息列表 |
| `GetTarget()` | 获取单主机详细信息 |
| `GetHostMetas()` | 获取所有主机元信息（转换为内部模型） |
| `GetHostMetaByIdent()` | 获取单主机元信息 |

**重试机制**:
- 使用 resty 内置 `SetRetryCount()` 和 `SetRetryWaitTime()`
- 最大重试次数：从配置读取（默认 3 次）
- 重试间隔：指数退避（baseDelay * 2^attempt）
- 可重试错误：超时、5xx、连接失败
- 4xx 错误不重试

**验证结果**:
- [x] 执行 `go build ./internal/client/n9e/` 无编译错误
- [x] 执行 `go build ./...` 整个项目编译无错误
- [x] 执行 `go test ./internal/client/n9e/` 全部通过（12 个测试用例）
- [x] Token 认证正确添加到请求头
- [x] 重试机制配置正确（指数退避）
- [x] 错误处理包含明确的上下文信息
- [x] ident 清理逻辑通过 `model.CleanIdent()` 和 `ToHostMeta()` 调用

---

### 步骤 16：编写 N9E 客户端单元测试 ✅

**完成日期**: 2025-12-13

**执行内容**:
1. 在 `internal/client/n9e/client_test.go` 中编写完整的单元测试
2. 使用 `net/http/httptest` 模拟 N9E API 服务器
3. 实现 5 个基础功能测试（构造函数、获取主机列表/详情、元信息转换）
4. 实现 4 个错误处理测试（401、404、API 错误、主机不存在）
5. 实现 5 个重试机制测试（重试条件函数、5xx 重试、4xx 不重试、最大重试次数）
6. 实现 3 个边界条件测试（空列表、部分失败、上下文取消）

**生成文件**:
- `internal/client/n9e/client_test.go` - N9E 客户端单元测试

**测试用例列表**:
| 类别 | 测试函数 | 场景 |
|------|----------|------|
| 基础功能 | `TestNewClient` | 客户端构造（含默认值） |
| 基础功能 | `TestGetTargets_Success` | 正常获取主机列表 |
| 基础功能 | `TestGetTarget_Success` | 正常获取单主机 |
| 基础功能 | `TestGetHostMetas_Success` | 获取主机元信息列表 |
| 基础功能 | `TestGetHostMetaByIdent_Success` | 获取单主机元信息 |
| 错误处理 | `TestGetTargets_Unauthorized` | 401 认证失败 |
| 错误处理 | `TestGetTargets_NotFound` | 404 资源不存在 |
| 错误处理 | `TestGetTargets_APIError` | N9E API 返回错误 |
| 错误处理 | `TestGetTarget_NotFound` | 单主机不存在 |
| 重试机制 | `TestRetryCondition` | 重试条件函数（8 个子测试） |
| 重试机制 | `TestGetTargets_ServerError_Retry` | 5xx 错误重试成功 |
| 重试机制 | `TestGetTargets_4xx_NoRetry` | 4xx 不重试 |
| 重试机制 | `TestGetTargets_MaxRetries_Exceeded` | 最大重试次数耗尽 |
| 边界条件 | `TestGetTargets_EmptyList` | 空主机列表 |
| 边界条件 | `TestGetHostMetas_PartialFailure` | 部分转换失败 |
| 边界条件 | `TestGetTargets_ContextCanceled` | 上下文取消 |

**验证结果**:
- [x] 执行 `go test ./internal/client/n9e/` 全部通过（29 个测试用例）
- [x] 测试覆盖率达到 91.6%（超过目标 70%）
- [x] 包含正向和异常场景测试
- [x] Token 认证头验证正确
- [x] 重试机制测试完整（5xx 重试、4xx 不重试、最大重试次数）
- [x] 边界条件测试完整（空列表、部分失败、上下文取消）

---

### 步骤 17：定义 VictoriaMetrics 客户端类型 ✅

**完成日期**: 2025-12-13

**执行内容**:
1. 在 `internal/client/vm/types.go` 中定义 VictoriaMetrics/Prometheus API 类型
2. 定义 `QueryResponse` API 响应结构体（status、data、errorType、error、warnings）
3. 定义 `QueryData` 结果数据结构体（resultType、result）
4. 定义 `Sample` 样本结构体（metric、value、values）
5. 定义 `Metric` 类型（map[string]string）用于存储标签
6. 定义 `SampleValue` 类型（[2]interface{}）解析 [timestamp, value] 数组
7. 定义 `QueryResult` 便捷结构体用于处理结果
8. 定义 `HostFilter` 主机筛选结构体（业务组、标签）
9. 实现辅助函数和方法：
   - `IsSuccess()`、`IsVector()`、`IsMatrix()`
   - `GetIdent()`、`GetLabel()`、`Name()`
   - `Timestamp()`、`TimestampUnix()`、`Value()`、`MustValue()`、`IsNaN()`
   - `ParseQueryResults()`、`GroupResultsByIdent()`、`IsEmpty()`
10. 编写完整的单元测试（14 个测试用例，48 个子测试）

**生成文件**:
- `internal/client/vm/types.go` - VM API 类型定义
- `internal/client/vm/types_test.go` - 单元测试

**类型定义概览**:
```go
// API 响应结构
type QueryResponse struct {
    Status    string     `json:"status"`    // success 或 error
    Data      QueryData  `json:"data"`      // 查询数据
    ErrorType string     `json:"errorType"` // 错误类型
    Error     string     `json:"error"`     // 错误信息
    Warnings  []string   `json:"warnings"`  // 警告信息
}

type QueryData struct {
    ResultType string   `json:"resultType"` // vector, matrix, scalar, string
    Result     []Sample `json:"result"`     // 结果样本列表
}

type Sample struct {
    Metric Metric        `json:"metric"` // 指标标签
    Value  SampleValue   `json:"value"`  // 即时查询值
    Values []SampleValue `json:"values"` // 范围查询值列表
}

type SampleValue [2]interface{} // [timestamp, value]

// 主机筛选
type HostFilter struct {
    BusinessGroups []string          // 业务组（OR 关系）
    Tags           map[string]string // 标签（AND 关系）
}
```

**关键功能**:
| 功能 | 说明 |
|------|------|
| `ParseQueryResults()` | 将 QueryResponse 转换为 []QueryResult 便于处理 |
| `GetIdent()` | 从标签中提取主机标识符（优先级：ident > host > instance） |
| `GroupResultsByIdent()` | 按主机标识符分组结果 |
| `IsNaN()` | 检测 NaN/Inf/无效值 |
| `Value()` | 安全解析字符串格式的指标值为 float64 |

**验证结果**:
- [x] 类型定义符合 Prometheus HTTP API 规范
- [x] 能够正确解析 vector 类型响应（JSON 解析测试通过）
- [x] 能够正确解析 error 类型响应
- [x] 执行 `go build ./internal/client/vm/` 无编译错误
- [x] 执行 `go build ./...` 整个项目编译无错误
- [x] 执行 `go test ./internal/client/vm/` 全部通过（14 个测试用例）
- [x] 测试覆盖率达到 93.0%（超过目标 70%）

---

### 步骤 18：实现 VictoriaMetrics 客户端 ✅

**完成日期**: 2025-12-13

**执行内容**:
1. 在 `internal/client/vm/client.go` 中实现 VictoriaMetrics API 客户端
2. 实现 `NewClient()` 构造函数，接收配置参数
3. 实现即时查询方法 `Query()` 和 `QueryWithFilter()`
4. 支持 PromQL 查询语句，正确进行 URL 编码
5. 实现结果解析，提取指标值和标签
6. 集成重试机制（指数退避，5xx 重试、4xx 不重试）
7. 实现主机筛选标签注入功能（业务组 OR + 标签 AND）
8. 提供便捷方法：`QueryResults()`、`QueryByIdent()` 及其带筛选版本
9. 编写完整的单元测试（14 个客户端测试用例）

**生成文件**:
- `internal/client/vm/client.go` - VM API 客户端实现
- `internal/client/vm/client_test.go` - 客户端单元测试

**客户端结构**:
```go
type Client struct {
    endpoint   string             // API endpoint
    timeout    time.Duration      // Request timeout
    retry      config.RetryConfig // Retry configuration
    httpClient *resty.Client      // HTTP client
    logger     zerolog.Logger     // Logger
}
```

**核心方法**:
| 方法 | 功能 |
|------|------|
| `NewClient()` | 创建客户端，配置超时和重试策略 |
| `Query()` | 执行即时查询 /api/v1/query |
| `QueryWithFilter()` | 带主机筛选的即时查询 |
| `QueryResults()` | 查询并解析为 []QueryResult |
| `QueryByIdent()` | 查询并按主机标识符分组 |
| `injectLabelMatchers()` | 注入主机筛选标签到 PromQL |

**主机筛选实现**:
```go
// 业务组 - OR 关系使用正则
busigroup=~"生产环境|测试环境"

// 标签 - AND 关系
env="prod", region="cn-east"

// 注入到查询
cpu_usage_active{cpu="cpu-total"}
  → cpu_usage_active{cpu="cpu-total", busigroup=~"生产环境|测试环境", env="prod"}
```

**重试机制**:
- 使用 resty 内置 `SetRetryCount()` 和 `SetRetryWaitTime()`
- 最大重试次数：从配置读取（默认 3 次）
- 重试间隔：指数退避（baseDelay * 2^attempt）
- 可重试错误：超时、5xx、连接失败
- 4xx 错误不重试

**验证结果**:
- [x] 执行 `go build ./internal/client/vm/` 无编译错误
- [x] 执行 `go build ./...` 整个项目编译无错误
- [x] 执行 `go test ./internal/client/vm/` 全部通过（28 个测试用例）
- [x] 测试覆盖率达到 94.0%（超过目标 70%）
- [x] 即时查询功能正常（Query、QueryResults、QueryByIdent）
- [x] 主机筛选标签注入正确（业务组 OR + 标签 AND）
- [x] 重试机制工作正常（5xx 重试、4xx 不重试）
- [x] 错误处理包含明确的上下文信息

---

### 步骤 19：编写 VictoriaMetrics 客户端单元测试 ✅

**完成日期**: 2025-12-13

**执行内容**:
1. 在 `internal/client/vm/client_test.go` 中补充更多测试用例
2. 实现带筛选的查询方法测试（QueryResultsWithFilter、QueryByIdentWithFilter）
3. 实现多主机、多指标场景测试（5 台主机并行结果验证）
4. 实现 NaN/Inf 值处理测试（验证无效值被正确过滤）
5. 实现 Matrix 类型响应错误处理测试
6. 实现超时测试（50ms 超时场景）
7. 实现最大重试次数耗尽测试（2 次重试 + 初始请求 = 3 次尝试）
8. 实现无效 JSON 响应处理测试
9. 实现空筛选和 nil 筛选测试
10. 实现主机标识符优先级测试（ident > host > instance）
11. 实现特殊字符转义测试（业务组名含 `.` `*` 等正则特殊字符）
12. 实现重复 ident 处理测试（多样本同 ident 时取最后一个）

**生成文件**:
- `internal/client/vm/client_test.go` - 补充 14 个新测试用例（含子测试）

**测试用例列表**:
| 类别 | 测试函数 | 场景 |
|------|----------|------|
| 带筛选方法 | `TestClient_QueryResultsWithFilter_Success` | QueryResultsWithFilter 成功场景 |
| 带筛选方法 | `TestClient_QueryByIdentWithFilter_Success` | QueryByIdentWithFilter 成功场景 |
| 多结果场景 | `TestClient_MultipleHosts_MultipleMetrics` | 5 台主机批量查询 |
| 异常值处理 | `TestClient_NaN_Inf_Values` | NaN/+Inf/-Inf 过滤 |
| 类型检查 | `TestClient_MatrixTypeResponse_Error` | Matrix 类型报错 |
| 重试机制 | `TestClient_MaxRetries_Exhausted` | 重试次数耗尽 |
| 超时处理 | `TestClient_Timeout` | 请求超时 |
| JSON 解析 | `TestClient_InvalidJSON_Response` | 无效 JSON 处理 |
| 筛选处理 | `TestClient_QueryWithFilter_NilFilter` | nil 筛选保持查询不变 |
| 筛选处理 | `TestClient_QueryWithFilter_EmptyFilter` | 空筛选保持查询不变 |
| 标识符逻辑 | `TestClient_HostIdentPriority` | ident > host > instance 优先级（3 个子测试） |
| 特殊字符 | `TestClient_SpecialCharactersInFilter` | 正则特殊字符转义 |
| 重复处理 | `TestClient_DuplicateIdent` | 同 ident 多样本取最后值 |

**验证结果**:
- [x] 执行 `go test ./internal/client/vm/` 全部通过（42 个测试用例，含子测试）
- [x] 测试覆盖率达到 **94.0%**（远超目标 70%）
- [x] 包含正向和异常场景测试
- [x] 主机筛选标签注入测试完整（业务组 OR + 标签 AND）
- [x] 重试机制测试完整（5xx 重试、4xx 不重试、最大重试次数）
- [x] 边界条件测试完整（空结果、NaN 值、超时、无效 JSON）
- [x] 执行 `go build ./...` 整个项目编译无错误

**测试覆盖率总结（阶段四完成）**:
| 模块 | 覆盖率 | 目标 |
|------|--------|------|
| N9E 客户端 | 91.6% | ≥70% ✅ |
| VM 客户端 | 94.0% | ≥70% ✅ |
| Config | 88.4% | ≥80% ✅ |

---

### 步骤 20：实现数据采集服务 ✅

**完成日期**: 2025-12-13

**执行内容**:
1. 在 `internal/service/collector.go` 中实现数据采集服务
2. 定义 `Collector` 结构体和 `CollectionResult` 结果类型
3. 实现 `NewCollector()` 构造函数
4. 实现 `CollectHostMetas()` 主机元信息采集
5. 实现 `CollectMetrics()` 指标数据采集
6. 实现 `collectSimpleMetric()` 普通指标采集
7. 实现 `collectExpandedMetric()` 按标签展开指标采集（磁盘按 path 展开）
8. 实现 `setPendingMetrics()` 待定项处理（返回 N/A）
9. 实现 `CollectAll()` 完整采集流程
10. 实现 `buildVMHostFilter()` 主机筛选配置转换
11. 编写完整的单元测试（15 个测试用例）

**生成文件**:
- `internal/service/collector.go` - 数据采集服务实现（350 行）
- `internal/service/collector_test.go` - 单元测试（954 行）

**核心结构体**:
```go
// 采集失败的主机
type FailedHost struct {
    Hostname string // 主机名
    Error    string // 错误信息
}

// 采集结果
type CollectionResult struct {
    Hosts       []*model.HostMeta             // 主机元信息列表
    HostMetrics map[string]*model.HostMetrics // 按主机名分组的指标数据
    FailedHosts []FailedHost                  // 采集失败的主机
    CollectedAt time.Time                     // 采集时间
}

// 数据采集器
type Collector struct {
    n9eClient  *n9e.Client
    vmClient   *vm.Client
    config     *config.Config
    metrics    []*model.MetricDefinition
    hostFilter *vm.HostFilter
    logger     zerolog.Logger
}
```

**核心方法**:
| 方法 | 功能 |
|------|------|
| `NewCollector()` | 创建采集器，转换主机筛选配置 |
| `CollectAll()` | 执行完整采集流程 |
| `CollectHostMetas()` | 从 N9E 获取主机元信息 |
| `CollectMetrics()` | 从 VM 获取指标数据 |
| `collectSimpleMetric()` | 采集普通指标 |
| `collectExpandedMetric()` | 采集按标签展开的指标（如磁盘） |
| `setPendingMetrics()` | 设置待定项为 N/A |
| `buildVMHostFilter()` | 转换配置中的主机筛选 |

**磁盘指标展开逻辑**:
```go
// 按 path 标签展开，生成：
// disk_usage:/     → 根分区使用率
// disk_usage:/home → home 分区使用率
// disk_usage:/var  → var 分区使用率
// disk_usage_max   → 聚合最大值（用于告警判断）
```

**测试用例列表**:
| 类别 | 测试函数 | 场景 |
|------|----------|------|
| 构造函数 | `TestNewCollector` | 基础构造 |
| 构造函数 | `TestNewCollector_WithHostFilter` | 带主机筛选 |
| 主机采集 | `TestCollector_CollectHostMetas_Success` | 正常获取主机 |
| 主机采集 | `TestCollector_CollectHostMetas_N9EError` | N9E 错误 |
| 主机采集 | `TestCollector_CollectHostMetas_EmptyList` | 空主机列表 |
| 主机采集 | `TestCollector_CollectHostMetas_ContextCanceled` | 上下文取消 |
| 指标采集 | `TestCollector_CollectMetrics_Success` | 正常获取指标 |
| 指标采集 | `TestCollector_CollectMetrics_PendingMetrics` | 待定项返回 N/A |
| 指标采集 | `TestCollector_CollectMetrics_VMQueryError` | VM 查询失败 |
| 完整流程 | `TestCollector_CollectAll_Success` | 完整流程成功 |
| 完整流程 | `TestCollector_CollectAll_NoHosts` | 无主机场景 |
| 完整流程 | `TestCollector_CollectAll_N9EFailure` | N9E 失败 |
| 主机筛选 | `TestCollector_HostFilter_Applied` | 筛选正确注入 |
| 主机筛选 | `TestCollector_HostFilter_Nil` | 无筛选保持原查询 |
| 磁盘展开 | `TestCollector_ExpandedMetric_DiskByPath` | 磁盘按挂载点展开 |

**验证结果**:
- [x] 执行 `go build ./internal/service/` 无编译错误
- [x] 执行 `go test ./internal/service/` 全部通过（15 个测试用例）
- [x] 测试覆盖率达到 **92.5%**（超过目标 80%）
- [x] 主机元信息采集正常（包括 CPU 核心数）
- [x] 指标数据采集正常（普通指标 + 展开指标）
- [x] 磁盘数据按挂载点展开（生成 disk_usage:/ 等）
- [x] 聚合值计算正确（disk_usage_max）
- [x] 待定项正确返回 N/A
- [x] 主机筛选正确应用到 VM 查询

---

### 步骤 21：实现阈值评估服务 ✅

**完成日期**: 2025-12-13

**执行内容**:
1. 在 `internal/service/evaluator.go` 中实现阈值评估服务
2. 定义 `Evaluator` 结构体和 `EvaluationResult`、`HostEvaluationResult` 结果类型
3. 实现 `NewEvaluator()` 构造函数
4. 实现 `EvaluateHost()` 单主机评估方法
5. 实现 `EvaluateAll()` 批量评估方法
6. 实现 `evaluateMetric()` 单指标评估私有方法
7. 实现 `getThreshold()` 指标名到阈值映射方法
8. 实现 `determineHostStatus()` 主机状态确定方法
9. 实现辅助方法：`getMetricDisplayName()`、`buildAlertMessage()`、`getMetricUnit()`
10. 编写完整的单元测试（30+ 个测试用例）

**生成文件**:
- `internal/service/evaluator.go` - 阈值评估服务实现（270 行）
- `internal/service/evaluator_test.go` - 单元测试（847 行）

**核心结构体**:
```go
// 单主机评估结果
type HostEvaluationResult struct {
    Hostname string                          // 主机名
    Status   model.HostStatus                // 主机整体状态
    Metrics  map[string]*model.MetricValue   // 更新后的指标
    Alerts   []*model.Alert                  // 该主机的告警列表
}

// 完整评估结果
type EvaluationResult struct {
    HostResults []*HostEvaluationResult // 各主机评估结果
    Alerts      []*model.Alert          // 所有告警列表
    Summary     *model.AlertSummary     // 告警摘要
}

// 阈值评估器
type Evaluator struct {
    thresholds *config.ThresholdsConfig
    metricDefs map[string]*model.MetricDefinition
    logger     zerolog.Logger
}
```

**核心方法**:
| 方法 | 功能 |
|------|------|
| `NewEvaluator()` | 创建评估器，指标定义映射 |
| `EvaluateAll()` | 批量评估所有主机 |
| `EvaluateHost()` | 评估单个主机所有指标 |
| `evaluateMetric()` | 评估单个指标，生成告警 |
| `getThreshold()` | 指标名映射到阈值配置 |
| `determineHostStatus()` | 根据告警确定主机状态 |

**指标阈值映射**:
| 指标名称 | 阈值配置项 | 警告 | 严重 |
|----------|------------|------|------|
| `cpu_usage` | CPUUsage | 70% | 90% |
| `memory_usage` | MemoryUsage | 70% | 90% |
| `disk_usage_max` | DiskUsage | 70% | 90% |
| `processes_zombies` | ZombieProcesses | 1 | 10 |
| `load_per_core` | LoadPerCore | 0.7 | 1.0 |

**特殊处理**:
- 磁盘使用 `disk_usage_max` 聚合值进行告警判断
- 扩展指标（如 `disk_usage:/home`）仅用于展示，不触发告警
- N/A 指标（`IsNA=true`）跳过评估，状态设为 `MetricStatusPending`
- 主机状态取最严重告警级别（Critical > Warning > Normal）

**测试用例列表**:
| 类别 | 测试数 | 场景 |
|------|--------|------|
| 构造函数 | 2 | 正常构造、nil 指标定义 |
| CPU 评估 | 4 | Normal/Warning/Critical/精确阈值 |
| 内存评估 | 3 | 三级阈值评估 |
| 磁盘评估 | 3 | 聚合最大值、扩展指标不告警、多磁盘场景 |
| 负载评估 | 3 | 三级阈值评估 |
| 僵尸进程 | 3 | 0/警告/严重场景 |
| N/A 处理 | 2 | 单独/混合指标 |
| 主机状态 | 3 | 最严重/正常/仅警告 |
| 批量评估 | 2 | 多主机/空输入 |
| 边界条件 | 5 | 空指标/nil/未配置阈值等 |
| 辅助函数 | 4 | 显示名称/阈值获取/消息构建/状态判断 |

**验证结果**:
- [x] CPU 75% 被判定为警告级别
- [x] CPU 95% 被判定为严重级别
- [x] 多个告警时主机状态取最严重级别
- [x] 无异常时状态为正常
- [x] 磁盘告警基于 `disk_usage_max`（最大使用率）判断
- [x] N/A 指标不触发告警
- [x] 执行 `go build ./internal/service/` 无编译错误
- [x] 执行 `go test ./internal/service/` 全部通过（49 个测试用例）
- [x] 测试覆盖率达到 **94.0%**（超过目标 80%）

---

### 步骤 22：实现巡检编排服务 ✅

**完成日期**: 2025-12-13

**执行内容**:
1. 在 `internal/service/inspector.go` 中实现核心巡检编排逻辑
2. 定义 `Inspector` 结构体和 `InspectorOption` 函数选项
3. 实现 `NewInspector()` 构造函数，支持时区加载和版本设置
4. 实现 `WithVersion()` 函数选项设置工具版本号
5. 实现 `Run()` 核心巡检流程方法
6. 实现 `buildInspectionResult()` 合并采集和评估结果
7. 实现 `GetTimezone()` 和 `GetVersion()` 访问方法
8. 编写完整的单元测试（10 个测试用例）

**生成文件**:
- `internal/service/inspector.go` - 巡检编排服务实现（192 行）
- `internal/service/inspector_test.go` - 单元测试（729 行）

**核心结构体**:
```go
// 巡检编排器
type Inspector struct {
    collector *Collector        // 数据采集器
    evaluator *Evaluator        // 阈值评估器
    config    *config.Config    // 配置
    timezone  *time.Location    // 时区（Asia/Shanghai）
    version   string            // 工具版本号
    logger    zerolog.Logger    // 日志
}

// 函数选项
type InspectorOption func(*Inspector)
```

**核心方法**:
| 方法 | 功能 |
|------|------|
| `NewInspector()` | 创建编排器，加载时区，应用选项 |
| `WithVersion()` | 设置工具版本号 |
| `Run()` | 执行完整巡检流程 |
| `buildInspectionResult()` | 合并采集和评估结果 |
| `GetTimezone()` | 获取配置的时区 |
| `GetVersion()` | 获取配置的版本号 |

**Run() 执行流程**:
```
1. 记录开始时间（Asia/Shanghai）
2. 调用 Collector.CollectAll() 获取数据
3. 调用 Evaluator.EvaluateAll() 评估阈值
4. 合并结果到 InspectionResult
   ├── 为每个主机创建 HostResult
   ├── 填充主机元信息（来自 CollectionResult.Hosts）
   ├── 填充指标数据（来自 EvaluationResult.HostResults）
   ├── 填充告警信息（来自 EvaluationResult.HostResults）
   └── 处理失败主机（来自 CollectionResult.FailedHosts）
5. 调用 Finalize() 计算摘要
6. 返回 InspectionResult
```

**测试用例列表**:
| 类别 | 测试函数 | 场景 |
|------|----------|------|
| 构造函数 | `TestNewInspector` | 基础构造、with version、无效时区 |
| 成功流程 | `TestInspector_Run_Success` | 完整巡检流程 |
| 成功流程 | `TestInspector_Run_NoHosts` | 无主机场景 |
| 告警评估 | `TestInspector_Run_WithWarning` | 警告级别告警 |
| 告警评估 | `TestInspector_Run_WithCritical` | 严重级别告警 |
| 多主机 | `TestInspector_Run_MultipleHosts` | 3 台主机混合状态 |
| 错误处理 | `TestInspector_Run_CollectorError` | Collector 失败 |
| 时区处理 | `TestInspector_Timezone` | Asia/Shanghai 正确应用 |
| 版本信息 | `TestInspector_GetVersion` | 版本号获取 |
| 上下文取消 | `TestInspector_Run_ContextCanceled` | 上下文超时/取消 |

**验证结果**:
- [x] 能够完成端到端的巡检流程
- [x] 所有时间使用 Asia/Shanghai 时区
- [x] 巡检摘要统计正确（TotalHosts, NormalHosts, WarningHosts, CriticalHosts）
- [x] 告警摘要统计正确（TotalAlerts, WarningCount, CriticalCount）
- [x] 单主机失败不影响整体流程
- [x] 执行 `go build ./internal/service/` 无编译错误
- [x] 执行 `go test ./internal/service/` 全部通过（10 个 Inspector 测试）
- [x] 测试覆盖率达到 **93.4%**（超过目标 80%）

---

### 步骤 23：编写业务逻辑单元测试（补充并发功能） ✅

**完成日期**: 2025-12-13

**执行内容**:
1. 发现步骤 22 的并发采集功能未实现（代码为串行）
2. 在 `collector.go` 中实现 errgroup 并发采集
3. 添加 `collectSimpleMetricConcurrent()` 并发安全方法
4. 添加 `collectExpandedMetricConcurrent()` 并发安全方法
5. 使用 `sync.Mutex` 保护 `hostMetricsMap` 并发写入
6. 并发数由 `config.Inspection.Concurrency` 控制（默认 20）
7. 添加 4 个并发测试用例

**修改文件**:
- `internal/service/collector.go` - 添加 errgroup 并发逻辑（500 行）
- `internal/service/collector_test.go` - 添加并发测试（1303 行）

**并发实现**:
```go
// 使用 errgroup 并发采集，限制并发数
g, ctx := errgroup.WithContext(ctx)
concurrency := c.config.Inspection.Concurrency
if concurrency <= 0 {
    concurrency = 20 // 默认值
}
g.SetLimit(concurrency)

var mu sync.Mutex // 保护 hostMetricsMap 的并发写入

for _, metric := range activeMetrics {
    metric := metric // 捕获循环变量
    g.Go(func() error {
        // 采集逻辑，使用 mutex 保护写入
        return nil // 单个指标失败不中止整体
    })
}
```

**新增测试用例**:
| 测试函数 | 场景 |
|----------|------|
| `TestCollector_CollectMetrics_Concurrent` | 5 指标并发采集验证 |
| `TestCollector_ConcurrencyLimit` | 并发限制验证（设置 2，验证最大并发≤2） |
| `TestCollector_CollectMetrics_PartialFailure_Concurrent` | 部分失败不影响其他 |
| `TestCollector_CollectMetrics_ContextCancel_Concurrent` | 上下文取消快速退出 |

**验证结果**:
- [x] 执行 `go build ./internal/service/` 无编译错误
- [x] 执行 `go test ./internal/service/` 全部通过（57 个测试用例）
- [x] 执行 `go test -race ./internal/service/` 无数据竞争
- [x] 测试覆盖率达到 **80.1%**（≥80% 目标）
- [x] 并发数受 `config.Inspection.Concurrency` 控制
- [x] 单个指标失败不影响其他指标采集

---

### 步骤 24：定义报告写入器接口 ✅

**完成日期**: 2025-12-13

**执行内容**:
1. 在 `internal/report/writer.go` 中定义 ReportWriter 接口
2. 定义 `Write()` 方法 - 将巡检结果写入文件
3. 定义 `Format()` 方法 - 返回格式标识符
4. 添加完整的包级和接口级注释

**生成文件**:
- `internal/report/writer.go` - ReportWriter 接口定义

**接口定义**:
```go
// ReportWriter defines the interface for generating inspection reports.
type ReportWriter interface {
    // Write generates a report from the inspection result and saves it
    // to the specified output path.
    Write(result *model.InspectionResult, outputPath string) error

    // Format returns the format identifier for this writer.
    // Common values are "excel" and "html".
    Format() string
}
```

**设计考量**:
| 考量点 | 决策 |
|--------|------|
| 参数类型 | 使用 `*model.InspectionResult` 指针避免大对象拷贝 |
| 路径参数 | `outputPath` 由调用方控制，包含完整路径和扩展名 |
| 错误处理 | 返回 `error` 用于报告生成或文件写入失败 |
| 扩展性 | 接口简洁，易于实现新格式（如 PDF） |

**验证结果**:
- [x] 文件创建在正确位置 `internal/report/writer.go`
- [x] 包注释遵循项目规范
- [x] 接口方法定义清晰，职责单一
- [x] 执行 `go build ./internal/report/` 无编译错误
- [x] 执行 `go build ./...` 整个项目编译无错误

---

## 下一步骤

**步骤 25**: 实现 Excel 报告生成器 - 基础结构
- 在 `internal/report/excel/writer.go` 中实现 ExcelWriter
- 实现 ReportWriter 接口
- 创建包含三个工作表的 Excel 文件：巡检概览、详细数据、异常汇总
- 使用 `xuri/excelize/v2` 库

---

## 版本记录

| 日期 | 步骤 | 说明 |
|------|------|------|
| 2025-12-13 | 步骤 1 | 初始化 Go 模块完成 |
| 2025-12-13 | 步骤 2 | 创建目录结构完成 |
| 2025-12-13 | 步骤 3 | 添加核心依赖完成 |
| 2025-12-13 | 步骤 4 | 创建程序入口文件完成（阶段一完成） |
| 2025-12-13 | 步骤 5 | 定义配置结构体完成（阶段二开始） |
| 2025-12-13 | 步骤 6 | 实现配置加载器完成 |
| 2025-12-13 | 步骤 7 | 实现配置验证器完成 |
| 2025-12-13 | 步骤 8 | 创建示例配置文件完成 |
| 2025-12-13 | 步骤 9 | 创建指标定义文件完成（阶段二完成） |
| 2025-12-13 | 步骤 10 | 定义主机模型完成（阶段三开始） |
| 2025-12-13 | 步骤 11 | 定义指标模型完成 |
| 2025-12-13 | 步骤 12 | 定义告警模型完成 |
| 2025-12-13 | 步骤 13 | 定义巡检结果模型完成（阶段三完成） |
| 2025-12-13 | 步骤 14 | 定义 N9E 客户端接口和类型完成（阶段四开始） |
| 2025-12-13 | 步骤 15 | 实现 N9E 客户端完成 |
| 2025-12-13 | 步骤 16 | 编写 N9E 客户端单元测试完成（覆盖率 91.6%） |
| 2025-12-13 | 步骤 17 | 定义 VictoriaMetrics 客户端类型完成（覆盖率 93.0%） |
| 2025-12-13 | 步骤 18 | 实现 VictoriaMetrics 客户端完成（覆盖率 94.0%） |
| 2025-12-13 | 步骤 19 | 编写 VictoriaMetrics 客户端单元测试完成（阶段四完成） |
| 2025-12-13 | 步骤 20 | 实现数据采集服务完成（覆盖率 92.5%，阶段五开始） |
| 2025-12-13 | 步骤 21 | 实现阈值评估服务完成（覆盖率 94.0%） |
| 2025-12-13 | 步骤 22 | 实现巡检编排服务完成（覆盖率 93.4%，阶段五完成） |
| 2025-12-13 | 步骤 23 | 补充并发采集功能 + 并发测试（覆盖率 80.1%） |
| 2025-12-13 | 步骤 24 | 定义报告写入器接口（阶段六开始） |
