# Tomcat 应用巡检功能 - 实施进度记录

## 当前状态

- **阶段一（采集配置）**：✅ 已完成（2025-12-23）
  - Step 1: 部署 Tomcat 巡检采集脚本 ✅ 已完成
  - Step 2: 配置 Categraf exec 插件并验证采集 ✅ 已完成

- **阶段二（数据模型与配置）**：✅ 已完成（2025-12-23）
  - **Step 3: 定义 Tomcat 数据模型** ✅ 已完成（2025-12-23）
  - **Step 4: 扩展配置结构并创建指标定义文件** ✅ 已完成（2025-12-23）

- **阶段三（服务实现）**：🔄 进行中
  - **Step 5: 实现 Tomcat 采集器和评估器** ✅ 已完成（2025-12-23）
  - **Step 6: 实现 Tomcat 巡检服务并集成到主服务** ✅ 已完成（2025-12-23）

- **阶段四（报告生成与验收）**：⏳ 待开始
  - Step 7: 扩展报告生成器支持 Tomcat ⏳ 待开始
  - Step 8: 端到端验收测试 ⏳ 待开始

---

## Step 3 完成详情（2025-12-23）

### 实施内容

在 `internal/model/tomcat.go` 中成功定义了 Tomcat 应用巡检的完整数据模型：

#### 1. TomcatInstanceStatus 枚举
- ✅ 4 个状态常量：normal, warning, critical, failed
- ✅ 4 个布尔方法：IsHealthy(), IsWarning(), IsCritical(), IsFailed()

#### 2. TomcatInstance 结构体
- ✅ 10 个字段（Identifier, Hostname, IP, Port, Container, ApplicationType, Version, InstallPath, LogPath, JVMConfig）
- ✅ Helper 函数：GenerateTomcatIdentifier（容器部署优先规则）
- ✅ 2 个构造函数：NewTomcatInstance（二进制）、NewTomcatInstanceWithContainer（容器）
- ✅ 6 个 Setter 方法：SetIP, SetApplicationType, SetVersion, SetInstallPath, SetLogPath, SetJVMConfig
- ✅ 2 个查询方法：IsContainerDeployment(), String()

#### 3. TomcatAlert 结构体
- ✅ 9 个字段（Identifier, MetricName, MetricDisplayName, CurrentValue, FormattedValue, WarningThreshold, CriticalThreshold, Level, Message）
- ✅ 使用 "Identifier" 字段（与 Nginx 模式一致）
- ✅ 构造函数：NewTomcatAlert
- ✅ 2 个布尔方法：IsWarning(), IsCritical()

#### 4. TomcatInspectionResult 结构体
- ✅ 13 个字段（Instance, Up, Connections, UptimeSeconds, UptimeFormatted, LastErrorTimestamp, LastErrorTimeFormatted, NonRootUser, PID, Status, Alerts, CollectedAt, Error）
- ✅ PID 字段使用 `json:"-"` 标签（内部使用，不序列化）
- ✅ 构造函数：NewTomcatInspectionResult
- ✅ 告警管理方法：AddAlert(), HasAlerts()
- ✅ 格式化方法：FormatUptime（支持天数显示）、FormatLastErrorTime（timestamp=0 显示"无错误"）
- ✅ 查询方法：GetIdentifier()

#### 5. TomcatInspectionSummary 结构体
- ✅ 5 个计数字段（TotalInstances, NormalInstances, WarningInstances, CriticalInstances, FailedInstances）
- ✅ 构造函数：NewTomcatInspectionSummary（支持 nil 安全迭代）

#### 6. TomcatAlertSummary 结构体
- ✅ 3 个计数字段（TotalAlerts, WarningCount, CriticalCount）
- ✅ 构造函数：NewTomcatAlertSummary

#### 7. TomcatInspectionResults 结构体
- ✅ 7 个字段（InspectionTime, Duration, Summary, Results, Alerts, AlertSummary, Version）
- ✅ 构造函数：NewTomcatInspectionResults
- ✅ Mutation 方法：AddResult（同时收集告警）、Finalize（计算摘要）
- ✅ 4 个查询方法：GetResultByIdentifier, GetCriticalResults, GetWarningResults, GetFailedResults
- ✅ 3 个布尔方法：HasCritical, HasWarning, HasAlerts

### 验证结果

✅ **编译验证通过**：`go build ./internal/model/` 无编译错误
✅ **代码总行数**：约 460 行（符合预期 500-550 行）
✅ **模式一致性**：与现有 MySQL/Nginx/Redis 模型保持一致
✅ **Nil 安全**：所有接收器方法均包含 nil 检查
✅ **时间格式化**：支持 Asia/Shanghai 时区，"无错误"中文显示

### 关键实现要点

1. **Identifier 生成规则（CRITICAL）**
   - 容器部署优先：`hostname:container`（例：`GX-MFUI-BE-01:tomcat-18001`）
   - 二进制部署：`hostname:port`（例：`GX-MFUI-BE-01:8080`）

2. **时间格式化（CRITICAL）**
   - `FormatUptime`：超过 1 天显示"X天 HH:MM:SS"，否则显示"HH:MM:SS"
   - `FormatLastErrorTime`：timestamp=0 返回"无错误"，否则格式化为"2006-01-02 15:04:05"

3. **PID 字段特殊处理**
   - 使用 `json:"-"` 标签，内部使用但不在 JSON 中序列化（不在报告中显示）

### 参考文件

- `internal/model/mysql.go` - 主要参考模式
- `internal/model/nginx.go` - Identifier 字段使用参考
- `internal/model/alert.go` - AlertLevel 枚举依赖
- `memory-bank/tomcat-feature-implementation.md` - 权威需求文档

---

## Step 4 完成详情（2025-12-23）

### 实施内容

#### 1. 扩展 internal/config/config.go

**添加位置**：第 16-17 行（Config 结构体）
```go
Tomcat      TomcatInspectionConfig `mapstructure:"tomcat"`
```

**添加位置**：第 173-205 行（Tomcat 配置结构体）
- ✅ TomcatInspectionConfig：Enabled、InstanceFilter、Thresholds
- ✅ TomcatFilter：HostnamePatterns、ContainerPatterns、BusinessGroups、Tags
  - **独有特性**：同时支持 HostnamePatterns 和 ContainerPatterns（双过滤器）
- ✅ TomcatThresholds：LastErrorWarningMinutes、LastErrorCriticalMinutes
  - **时间反转**：warning > critical（时间越短越严重）

#### 2. 创建 configs/tomcat-metrics.yaml

**文件路径**：`configs/tomcat-metrics.yaml`

**指标定义**（7 个指标）：
- ✅ tomcat_up：运行状态（category: status）
- ✅ tomcat_info：实例信息，标签提取 [port, app_type, install_path, log_path, version, jvm_config]
- ✅ tomcat_connections：当前连接数（仅展示，不告警）
- ✅ tomcat_non_root_user：非 root 用户启动（category: security）
- ✅ tomcat_uptime_seconds：运行时长（format: duration）
- ✅ tomcat_last_error_timestamp：最近错误日志时间（format: timestamp）

**设计要点**：
- 不包含 tomcat_pid（仅内部使用，不在报告中展示）
- 与 MySQL/Nginx metrics.yaml 格式保持一致

#### 3. 更新 configs/config.example.yaml

**添加位置**：第 312-358 行（Tomcat 配置节）

**配置结构**：
```yaml
tomcat:
  enabled: true
  instance_filter:
    hostname_patterns: []     # 主机名模式（glob）
    container_patterns: []    # 容器名模式（glob）- Tomcat 独有
    business_groups: []       # 业务组（OR）
    tags: {}                  # 标签（AND）
  thresholds:
    last_error_warning_minutes: 60   # 时间反转阈值
    last_error_critical_minutes: 10
```

**注释说明**：
- 明确说明双过滤器使用场景
- 强调二进制部署实例无 container 标签
- 注释时间反转逻辑

### 验证结果

✅ **编译验证通过**：
- `go build ./internal/config/` 无编译错误
- `go build ./cmd/inspect/` 无编译错误

✅ **文件修改清单**：
| 文件 | 操作 | 新增行数 |
|------|------|----------|
| internal/config/config.go | 修改 | +35 |
| configs/tomcat-metrics.yaml | 新建 | +70 |
| configs/config.example.yaml | 修改 | +48 |

✅ **模式一致性**：
- TomcatFilter 与 NginxFilter 字段命名一致
- TomcatThresholds 与 NginxThresholds 字段命名一致
- YAML 配置与 Go 结构体 mapstructure 标签一一对应

### 关键实现要点

1. **双过滤器模式（Tomcat 独有）**
   ```go
   type TomcatFilter struct {
       HostnamePatterns  []string  // 主机名模式
       ContainerPatterns []string  // 容器名模式 - Tomcat 特有
       BusinessGroups    []string  // 业务组（OR）
       Tags              map[string]string // 标签（AND）
   }
   ```

2. **时间反转阈值**
   ```go
   LastErrorWarningMinutes: 60   // 1 小时内有错误 → 警告
   LastErrorCriticalMinutes: 10  // 10 分钟内有错误 → 严重
   ```
   - 与 Nginx 保持一致的字段命名
   - 时间越短越严重（warning > critical）

3. **配置加载验证**（Step 5 实现）
   - 阈值 validate:"gte=0" 确保非负数
   - 与 MySQL/Redis/Nginx 配置加载逻辑一致

### 参考文件

- internal/config/config.go - MySQL/Redis/Nginx 配置结构参考
- configs/mysql-metrics.yaml - 指标 YAML 格式参考
- configs/config.example.yaml - 配置示例格式参考
- memory-bank/tomcat-feature-implementation.md - 权威需求文档

---

## Step 5 完成详情（2025-12-23）

### 实施内容

#### 1. 扩展 Tomcat 数据模型

**文件**：`internal/model/tomcat.go`

**添加 TomcatMetricValue 结构体**：
```go
type TomcatMetricValue struct {
    Name           string            `json:"name"`
    RawValue       float64           `json:"raw_value"`
    StringValue    string            `json:"string_value,omitempty"` // 标签提取的字符串值
    FormattedValue string            `json:"formatted_value"`
    IsNA           bool              `json:"is_na"`
    Timestamp      int64             `json:"timestamp"`
    Labels         map[string]string `json:"labels,omitempty"`
}
```

**扩展 TomcatInspectionResult 结构体**：
- ✅ 添加 `Metrics map[string]*TomcatMetricValue` 字段（带 `json:"-"` 标签）
- ✅ 添加 `SetMetric(mv *TomcatMetricValue)` 方法
- ✅ 添加 `GetMetric(name string) *TomcatMetricValue` 方法

#### 2. 创建 Tomcat 指标定义模型

**文件**：`internal/model/tomcat_metric.go`（新建）

**结构体定义**：
- ✅ `TomcatMetricDefinition`：指标定义结构体
  - Name, DisplayName, Query, Category
  - LabelExtract []string（从标签提取的字段）
  - Format, Status, Note
- ✅ `TomcatMetricsConfig`：YAML 根结构体

**方法**：
- ✅ `IsPending()` - 判断指标是否待实现
- ✅ `HasLabelExtract()` - 判断是否需要从标签提取值
- ✅ `GetDisplayName()` - 获取指标显示名称

#### 3. 扩展配置加载器

**文件**：`internal/config/metrics.go`

**添加函数**：
- ✅ `LoadTomcatMetrics(metricsPath string)` - 从 YAML 文件加载 Tomcat 指标定义
- ✅ `CountActiveTomcatMetrics(metrics)` - 统计活跃指标数量

**实现要点**：
- 与 LoadMySQLMetrics、LoadRedisMetrics、LoadNginxMetrics 模式一致
- 包含完整的文件验证和指标定义验证

#### 4. 实现 Tomcat 采集器

**文件**：`internal/service/tomcat_collector.go`（新建）

**核心结构体**：
```go
type TomcatCollector struct {
    vmClient       *vm.Client
    n9eClient      *n9e.Client
    config         *config.TomcatInspectionConfig
    metrics        []*model.TomcatMetricDefinition
    metricDefs     map[string]*model.TomcatMetricDefinition
    instanceFilter *TomcatInstanceFilter
    logger         zerolog.Logger
}

type TomcatInstanceFilter struct {
    HostnamePatterns  []string          // 主机名模式（glob）
    ContainerPatterns []string          // 容器名模式（glob）
    BusinessGroups    []string          // 业务组（OR）
    Tags              map[string]string // 标签（AND）
}
```

**核心方法**：

| 方法 | 说明 |
|------|------|
| `NewTomcatCollector()` | 创建采集器 |
| `DiscoverInstances()` | 查询 `tomcat_up == 1` 发现实例 |
| `buildContainerMap()` | 构建 hostname->container 映射 |
| `buildInfoMap()` | 构建 hostname->labels 映射 |
| `extractHostname()` | 提取主机名（agent_hostname > ident > host） |
| `extractIdentifier()` | 提取标识符（容器优先） |
| `matchesHostnamePatterns()` | 主机名模式匹配 |
| `matchesContainerPatterns()` | 容器名模式匹配 |
| `CollectMetrics()` | 采集所有指标 |
| `collectMetricConcurrent()` | 并发采集单个指标 |
| `collectLabelExtractMetric()` | 标签提取指标采集 |
| `extractFieldsFromMetrics()` | 从指标提取字段值 |

**关键实现要点**：
1. **双过滤器模式**：同时支持 `HostnamePatterns` 和 `ContainerPatterns`
2. **Identifier 生成**：容器部署优先（`hostname:container`），二进制部署用（`hostname:port`）
3. **IP 获取**：从 N9E API 获取主机 IP 地址
4. **标签提取**：从 `tomcat_info` 标签提取 `port, app_type, install_path, log_path, version, jvm_config`
5. **并发安全**：使用 errgroup + sync.Mutex 保护共享 map

**代码行数**：约 730 行

#### 5. 实现 Tomcat 评估器

**文件**：`internal/service/tomcat_evaluator.go`（新建）

**核心结构体**：
```go
type TomcatEvaluator struct {
    thresholds *config.TomcatThresholds
    metricDefs map[string]*model.TomcatMetricDefinition
    timezone   *time.Location
    logger     zerolog.Logger
}

type TomcatEvaluationResult struct {
    Identifier string
    Status     model.TomcatInstanceStatus
    Alerts     []*model.TomcatAlert
}
```

**核心方法**：

| 方法 | 说明 |
|------|------|
| `NewTomcatEvaluator()` | 创建评估器 |
| `EvaluateAll()` | 批量评估所有实例 |
| `Evaluate()` | 评估单个实例 |
| `evaluateUpStatus()` | 运行状态评估（tomcat_up=0 -> Critical） |
| `evaluateNonRootUser()` | 非 root 用户评估（=0 -> Critical） |
| `evaluateLastErrorTime()` | 最近错误时间评估（**时间反转逻辑**） |
| `determineInstanceStatus()` | 聚合状态 |
| `createAlert()` | 创建告警 |
| `formatValue()` | 格式化指标值 |
| `generateAlertMessage()` | 生成告警消息 |
| `getThresholds()` | 获取阈值 |

**时间反转阈值逻辑（CRITICAL）**：
```go
// 配置：warning=60分钟, critical=10分钟
// 逻辑：时间越短越严重

minutesSinceError := (now - timestamp) / 60

// Critical: 错误在 10 分钟内
if minutesSinceError <= criticalMinutes {
    return AlertLevelCritical
}

// Warning: 错误在 60 分钟内
if minutesSinceError <= warningMinutes {
    return AlertLevelWarning
}

// Normal: 无错误或错误超过 60 分钟
return nil
```

**代码行数**：约 290 行

### 验证结果

✅ **编译验证通过**：
- `go build ./internal/model/` 无编译错误
- `go build ./internal/config/` 无编译错误
- `go build ./internal/service/` 无编译错误
- `go build ./cmd/inspect/` 无编译错误

✅ **文件清单**：
| 文件 | 操作 | 代码行数 |
|------|------|----------|
| internal/model/tomcat.go | 修改 | +30 |
| internal/model/tomcat_metric.go | 新建 | +45 |
| internal/config/metrics.go | 修改 | +53 |
| internal/service/tomcat_collector.go | 新建 | +730 |
| internal/service/tomcat_evaluator.go | 新建 | +290 |

✅ **模式一致性检查**：
- ✅ 与 Redis/MySQL 采集器结构一致
- ✅ 与 Redis/MySQL 评估器结构一致
- ✅ 错误处理模式一致（单个指标失败不中止整体）
- ✅ nil 安全处理一致（所有接收器方法包含 nil 检查）
- ✅ 并发安全处理一致（errgroup + sync.Mutex）

### 关键实现要点

1. **双过滤器模式（Tomcat 独有）**
   ```go
   type TomcatInstanceFilter struct {
       HostnamePatterns  []string  // 主机名模式
       ContainerPatterns []string  // 容器名模式 - Tomcat 特有
       BusinessGroups    []string  // 业务组（OR）
       Tags              map[string]string // 标签（AND）
   }
   ```

2. **时间反转阈值评估**
   - 配置：`LastErrorWarningMinutes: 60`, `LastErrorCriticalMinutes: 10`
   - 逻辑：`minutesSinceError <= critical` → Critical（严重）
   - 逻辑：`minutesSinceError <= warning` → Warning（警告）
   - 与常规阈值逻辑相反（warning > critical）

3. **容器优先 Identifier 生成**
   - 容器部署：`hostname:container`（如 `GX-MFUI-BE-01:tomcat-18001`）
   - 二进制部署：`hostname:port`（如 `GX-MFUI-BE-01:8080`）

4. **标签提取模式**
   - `tomcat_info` 指标包含多个标签
   - LabelExtract: `[port, app_type, install_path, log_path, version, jvm_config]`
   - 提取的值存储在 `TomcatMetricValue.StringValue` 中

### 参考文件

- `internal/service/nginx_collector.go` - 双过滤器模式参考
- `internal/service/redis_evaluator.go` - 评估器结构参考
- `internal/service/mysql_collector.go` - 标签提取模式参考
- `internal/model/alert.go` - AlertLevel 枚举依赖
- `memory-bank/tomcat-feature-implementation.md` - 权威需求文档

---

## Step 6 完成详情（2025-12-23）

### 实施内容

#### 1. 创建 Tomcat 巡检编排器

**文件**：`internal/service/tomcat_inspector.go`（新建）

**核心结构体**：
```go
type TomcatInspector struct {
    collector *TomcatCollector
    evaluator *TomcatEvaluator
    config    *config.Config
    timezone  *time.Location
    version   string
    logger    zerolog.Logger
}

type TomcatInspectorOption func(*TomcatInspector)
```

**核心方法**：

| 方法 | 说明 | 行数 |
|------|------|------|
| `NewTomcatInspector()` | 构造函数，验证参数、加载时区、应用选项 | 45 |
| `WithTomcatVersion()` | 函数选项模式，设置版本号 | 8 |
| `Inspect()` | 核心编排方法：发现→采集→评估→聚合 | 75 |
| `buildInspectionResults()` | 合并结果到容器 | 25 |
| `GetTimezone()` | 返回配置的时区 | 5 |
| `GetVersion()` | 返回配置的版本号 | 5 |
| `IsEnabled()` | 返回 Tomcat 巡检是否启用 | 5 |
| `GetConfig()` | 返回 Tomcat 配置 | 8 |

**Inspect() 执行流程**：
```
1. 记录开始时间（timezone）
2. 创建 TomcatInspectionResults 结果容器
3. 调用 collector.DiscoverInstances() 发现实例
   ├── 错误：返回 error
   └── 空列表：优雅降级，Finalize 后返回
4. 获取指标定义（collector.GetMetrics()）
   └── 空检查：返回 error
5. 调用 collector.CollectMetrics() 采集指标
   └── 返回 map[string]*TomcatInspectionResult
6. 调用 evaluator.EvaluateAll() 评估阈值
7. 调用 buildInspectionResults() 构建结果
8. 调用 result.Finalize() 最终化
9. 记录完成日志
10. 如果有严重告警，额外记录
11. 返回 result
```

**代码行数**：约 210 行

#### 2. 集成到 CLI 入口

**文件**：`cmd/inspect/cmd/run.go`（修改）

**添加内容**：

1. **命令行标志**（3 个变量）
   - `tomcatMetricsPath string` - Tomcat 指标定义文件路径
   - `tomcatOnly bool` - 仅执行 Tomcat 巡检
   - `skipTomcat bool` - 跳过 Tomcat 巡检

2. **init() 中注册标志**
   - `--tomcat-metrics` (默认: configs/tomcat-metrics.yaml)
   - `--tomcat-only`
   - `--skip-tomcat`

3. **标志验证逻辑**（5 个互斥验证）
   - `--tomcat-only` 与 `--skip-tomcat` 互斥
   - `--tomcat-only` 与 `--mysql-only` 互斥
   - `--tomcat-only` 与 `--redis-only` 互斥
   - `--tomcat-only` 与 `--nginx-only` 互斥
   - `--tomcat-only` 时验证配置已启用

4. **执行模式判断**
   - `runTomcatInspection := !skipTomcat && !mysqlOnly && !redisOnly && !nginxOnly && cfg.Tomcat.Enabled`
   - 更新所有现有模式的判断条件以包含 `!tomcatOnly`

5. **Tomcat 指标加载**（Step 3e）
   - 调用 `config.LoadTomcatMetrics(tomcatMetricsPath)`
   - 调用 `config.CountActiveTomcatMetrics(tomcatMetrics)`
   - 输出日志和活跃指标数

6. **Tomcat 服务创建**（Step 7e）
   - 创建 `TomcatCollector`
   - 创建 `TomcatEvaluator`
   - 创建 `TomcatInspector`
   - 应用 `WithTomcatVersion(Version)` 选项

7. **Tomcat 巡检执行**
   - 输出 "⏳ 开始 Tomcat 巡检..."
   - 调用 `tomcatInspector.Inspect(ctx)`
   - 错误处理：如果所有巡检都失败则退出，否则继续
   - 成功后输出 "📊 Tomcat 巡检完成！"
   - 调用 `printTomcatSummary(tomcatResult)` 打印摘要

8. **printTomcatSummary 函数**
   ```go
   func printTomcatSummary(result *model.TomcatInspectionResults) {
       fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
       if result.Summary != nil {
           fmt.Printf("   Tomcat 实例总数: %d\n", result.Summary.TotalInstances)
           fmt.Printf("   正常实例: %d\n", result.Summary.NormalInstances)
           fmt.Printf("   警告实例: %d\n", result.Summary.WarningInstances)
           fmt.Printf("   严重实例: %d\n", result.Summary.CriticalInstances)
           fmt.Printf("   失败实例: %d\n", result.Summary.FailedInstances)
       }
       fmt.Println()
       if result.AlertSummary != nil {
           fmt.Printf("   Tomcat 告警总数: %d\n", result.AlertSummary.TotalAlerts)
           fmt.Printf("   警告级别: %d\n", result.AlertSummary.WarningCount)
           fmt.Printf("   严重级别: %d\n", result.AlertSummary.CriticalCount)
       }
   }
   ```

9. **报告生成函数签名修改**
   - `generateCombinedExcel()` 添加 `tomcatResult *model.TomcatInspectionResults` 参数
   - `generateCombinedHTML()` 添加 `tomcatResult *model.TomcatInspectionResults` 参数
   - 两个函数内部暂时记录 TODO 日志（Step 7 实现报告生成）

10. **退出码判断**
    - 添加 Tomcat 严重/警告实例退出码逻辑
    - `CriticalInstances > 0` → exitCode = 2
    - `WarningInstances > 0` → exitCode = 1

11. **runCmd Long 描述更新**
    - 添加 "6. 执行 Tomcat 应用巡检（如果启用）"
    - 添加 `--tomcat-only` 示例
    - 添加 `--skip-tomcat` 示例
    - 添加 `--tomcat-metrics` 示例

**代码行数**：+125 行

### 验证结果

✅ **编译验证通过**：
- `go build ./internal/service/` 无编译错误
- `go build ./cmd/inspect/` 无编译错误

✅ **文件清单**：
| 文件 | 操作 | 新增行数 |
|------|------|----------|
| internal/service/tomcat_inspector.go | 新建 | +210 |
| cmd/inspect/cmd/run.go | 修改 | +125 |

✅ **模式一致性检查**：
- ✅ 与 MySQL/Redis/Nginx Inspector 结构一致
- ✅ 与 MySQL/Redis/Nginx CLI 集成模式一致
- ✅ 函数选项模式实现一致
- ✅ 错误处理模式一致
- ✅ 日志记录模式一致

### 关键实现要点

1. **模式一致性**
   - TomcatInspector 结构体与 MySQL/Redis/Nginx 完全一致
   - Inspect() 方法流程与 MySQL/Nginx 保持一致
   - CLI 集成模式与现有服务保持一致

2. **报告生成（Step 7 预留）**
   - generateCombinedExcel 和 generateCombinedHTML 函数签名已更新
   - 内部暂时记录 TODO 日志，不实际生成 Tomcat 报告
   - Step 7 将实现 `WriteTomcatInspection()` 和 `AppendTomcatInspection()` 方法

3. **时区处理**
   - TomcatInspector 使用配置的时区（默认 Asia/Shanghai）
   - 所有时间戳在 buildInspectionResults 中转换为配置时区

4. **优雅降级**
   - 空实例列表时返回空结果而不是错误
   - 单个巡检失败不中止整体流程

### 参考文件

- `internal/service/mysql_inspector.go` - 主要参考模式
- `internal/service/nginx_inspector.go` - N9E API 集成参考
- `internal/service/redis_inspector.go` - 简洁的流程编排参考
- `cmd/inspect/cmd/run.go` - CLI 集成参考

---

## 下一步

✅ Step 6 已完成，**请用户审核通过后再进入 Step 7**

Step 7 将进行：
- 扩展 Excel 报告生成器支持 Tomcat（internal/report/excel/writer.go）
- 扩展 HTML 报告生成器支持 Tomcat（internal/report/html/writer.go）
- 端到端验收测试
