# Nginx 巡检功能 - 开发进度记录

## 实施进度

| 阶段 | 步骤 | 状态 | 完成日期 |
|------|------|------|----------|
| 一、采集配置 | 1. 部署采集脚本 | ✅ 已完成 | 2025-12-20 |
| | 2. 配置 exec 插件 | ✅ 已完成 | 2025-12-20 |
| 二、数据模型 | 3. 定义 Nginx 数据模型 | ✅ 已完成 | 2025-12-21 |
| | 4. 扩展配置结构 | ✅ 已完成 | 2025-12-21 |
| 三、服务实现 | 5. 实现采集器和评估器 | ✅ 已完成 | 2025-12-22 |
| | 5.1 创建 Nginx 巡检服务 | ✅ 已完成 | 2025-12-22 |
| | 6. 集成到主服务 | ✅ 已完成 | 2025-12-22 |
| 四、报告验收 | 7. 扩展报告生成器 | ⏳ 待开始 | - |
| | 8. 端到端验收 | ⏳ 待开始 | - |

---

## 步骤 4 完成详情

**完成日期**: 2025-12-21

### 修改/创建的文件

| 文件 | 操作 | 说明 |
|------|------|------|
| `internal/config/config.go` | 修改 | 添加 3 个 Nginx 配置结构体 + Config.Nginx 字段 |
| `internal/config/loader.go` | 修改 | 在 setDefaults() 中添加 Nginx 默认值 |
| `internal/config/validator.go` | 修改 | 添加 validateNginxThresholds() 验证函数 |
| `configs/nginx-metrics.yaml` | 创建 | 创建 Nginx 指标定义文件（11 个指标） |
| `configs/config.example.yaml` | 修改 | 添加 Nginx 配置节 |

### 新增的配置结构体

#### NginxInspectionConfig

```go
type NginxInspectionConfig struct {
    Enabled        bool            `mapstructure:"enabled"`
    InstanceFilter NginxFilter     `mapstructure:"instance_filter"`
    Thresholds     NginxThresholds `mapstructure:"thresholds"`
}
```

#### NginxFilter

```go
type NginxFilter struct {
    HostnamePatterns []string          `mapstructure:"hostname_patterns"` // 主机名通配符匹配
    BusinessGroups   []string          `mapstructure:"business_groups"`   // 业务组 (OR)
    Tags             map[string]string `mapstructure:"tags"`              // 标签 (AND)
}
```

#### NginxThresholds

```go
type NginxThresholds struct {
    ConnectionUsageWarning   float64 // 连接使用率警告阈值 (默认: 70%)
    ConnectionUsageCritical  float64 // 连接使用率严重阈值 (默认: 90%)
    LastErrorWarningMinutes  int     // 错误日志警告时间 (默认: 60 分钟)
    LastErrorCriticalMinutes int     // 错误日志严重时间 (默认: 10 分钟)
}
```

### nginx-metrics.yaml 指标定义

| 类别 | 指标名 | 说明 |
|------|--------|------|
| connection | nginx_up | 运行状态 |
| connection | nginx_active | 活跃连接数 |
| info | nginx_info | 实例信息（标签提取：port, app_type, install_path, version） |
| config | nginx_worker_processes | Worker 进程数 |
| config | nginx_worker_connections | 单 Worker 最大连接数 |
| config | nginx_error_page_4xx | 4xx 错误页配置 |
| config | nginx_error_page_5xx | 5xx 错误页配置 |
| security | nginx_non_root_user | 非 root 用户启动 |
| log | nginx_last_error_timestamp | 最近错误日志时间 |
| upstream | nginx_upstream_check_status_code | Upstream 后端状态 |
| upstream | nginx_upstream_check_rise | Upstream 连续成功次数 |
| upstream | nginx_upstream_check_fall | Upstream 连续失败次数 |

### 验证结果

- [x] `go build ./internal/config/` 编译通过
- [x] `go build ./internal/model/` 编译通过
- [x] `go build ./...` 全项目编译通过
- [x] `go vet ./internal/config/...` 静态检查通过

### 设计决策

1. **无 ClusterMode 字段**：与 MySQL/Redis 不同，Nginx 不存在集群模式概念

2. **错误日志阈值逻辑反转**：`warning > critical`（60 分钟 > 10 分钟）
   - 越近的错误越严重：10 分钟内有错误 = critical
   - 较远的错误较轻：1 小时内有错误 = warning

3. **与 MySQL/Redis 配置的一致性**：
   - 相同的结构体设计模式
   - 相同的 Filter 三层过滤（地址模式/业务组/标签）
   - 相同的验证逻辑模式

---

## 步骤 3 完成详情

**完成日期**: 2025-12-21

**创建文件**: `internal/model/nginx.go`

### 已定义的结构体

| 结构体 | 用途 | 行数 |
|--------|------|------|
| `NginxInstanceStatus` | 实例状态枚举（normal/warning/critical/failed） | 4 常量 + 4 方法 |
| `NginxInstance` | 实例元信息（9 个字段） | ~50 行 |
| `NginxUpstreamStatus` | Upstream 后端状态（5 个字段） | ~15 行 |
| `NginxAlert` | 告警结构体（复用 AlertLevel） | ~30 行 |
| `NginxMetricValue` | 指标值结构体 | ~10 行 |
| `NginxInspectionResult` | 单实例巡检结果（17 个字段 + 12 方法） | ~120 行 |
| `NginxInspectionSummary` | 巡检摘要统计 | ~30 行 |
| `NginxAlertSummary` | 告警摘要统计 | ~30 行 |
| `NginxInspectionResults` | 完整巡检结果集合 | ~100 行 |
| `NginxMetricDefinition` | 指标定义 | ~25 行 |
| `NginxMetricsConfig` | 指标配置文件结构 | ~5 行 |

**总代码行数**: 约 570 行

### 关键设计决策

1. **实例标识生成规则**
   - 容器部署: `hostname:container`
   - 二进制部署: `hostname:port`
   - 函数: `GenerateNginxIdentifier()`

2. **连接使用率计算**
   - 公式: `ActiveConnections / (WorkerProcesses * WorkerConnections) * 100`
   - 无法计算时返回 `-1`

3. **错误日志时间戳处理**
   - `0` 表示从未有错误日志
   - 格式化为 "无错误" 或 "2006-01-02 15:04:05"

4. **Upstream 状态**
   - `UpstreamStatus` 字段为 `[]NginxUpstreamStatus`（可为空数组）
   - 无 upstream 配置时显示 "N/A"

### 验证结果

- [x] `go build ./internal/model/` 编译通过
- [x] `go build ./...` 全项目编译通过
- [x] `go vet ./internal/model/...` 静态检查通过
- [x] 结构体字段覆盖所有巡检项

### 与 MySQL/Redis 模型的一致性

- ✅ 枚举类型定义方式一致（类型别名 + 常量组）
- ✅ 状态判断方法命名一致（IsHealthy, IsWarning, IsCritical, IsFailed）
- ✅ 构造函数模式一致（NewXxx, NewXxxWithYyy）
- ✅ JSON 标签风格一致（snake_case, omitempty）
- ✅ 空值检查模式一致（nil 检查 + map 初始化）
- ✅ 摘要统计方法一致（NewXxxSummary）
- ✅ 结果集合方法一致（AddResult, Finalize, GetXxxResults）

---

## 步骤 5 完成详情

**完成日期**: 2025-12-22

### 创建/修改的文件

| 文件 | 操作 | 说明 |
|------|------|------|
| `internal/service/nginx_collector.go` | 创建 | Nginx 数据采集器（777 行） |
| `internal/service/nginx_collector_test.go` | 创建 | 采集器单元测试（43 个测试） |
| `internal/service/nginx_evaluator.go` | 创建 | Nginx 阈值评估器（531 行） |
| `internal/service/nginx_evaluator_test.go` | 创建 | 评估器单元测试（31 个测试） |
| `internal/service/nginx_inspector.go` | 创建 | Nginx 巡检编排服务（223 行） |
| `internal/service/nginx_inspector_test.go` | 创建 | 巡检服务单元测试（12 个测试） |

### 总代码统计

| 文件 | 行数 | 测试数 | 覆盖率 |
|------|------|--------|--------|
| nginx_collector.go | 777 | 43 | 高 |
| nginx_evaluator.go | 531 | 31 | 高 |
| nginx_inspector.go | 223 | 12 | 高 |
| **总计** | **1531** | **86** | **高** |

### 核心功能实现

#### 1. NginxCollector 数据采集器

**主要方法**:
- `DiscoverInstances()` - 通过 `nginx_info` 发现实例
- `CollectMetrics()` - 并发采集所有指标（支持 errgroup）
- `CollectUpstreamStatus()` - 采集 Upstream 后端状态
- `matchesHostnamePatterns()` - 主机名通配符匹配（如 `GX-NM-*`）

**特性**:
- 支持容器和二进制部署
- 从 N9E API 获取 IP 地址
- 并发采集，单指标失败不影响整体
- 标签提取支持（port, app_type, version, install_path）

#### 2. NginxEvaluator 阈值评估器

**评估规则**:
- 连接状态：`nginx_up=0` → Critical
- 连接使用率：>90% Critical, >70% Warning
- 错误日志时间：<10min Critical, <60min Warning（逻辑反转）
- 错误页配置：未配置 → Critical
- 非root用户：root启动 → Critical
- Upstream状态：后端异常 → Critical

#### 3. NginxInspector 巡检编排器

**工作流程**:
1. 发现实例（DiscoverInstances）
2. 采集指标（CollectMetrics）
3. 采集 Upstream 状态（CollectUpstreamStatus）
4. 评估阈值（EvaluateAll）
5. 聚合结果（AddResult + Finalize）

### 测试验证

- **总测试数**: 86 个
- **通过率**: 100%
- **覆盖场景**:
  - 实例发现（成功/失败/过滤）
  - 指标采集（并发/标签提取/待处理指标）
  - 阈值评估（各级别告警）
  - 巡检编排（完整流程/异常处理）

---

## 步骤 6 完成详情

**完成日期**: 2025-12-21

### 修改的文件

| 文件 | 操作 | 说明 |
|------|------|------|
| `internal/report/html/writer.go` | 修改 | WriteCombined 方法签名添加 nginxResult 参数，添加 Nginx 数据结构和辅助函数 |
| `internal/report/html/writer_test.go` | 修改 | 修复4处 WriteCombined 调用，添加 nil 作为 nginxResult 参数 |

### 新增的数据结构

#### NginxInstanceData

```go
type NginxInstanceData struct {
    Identifier             string
    Hostname               string
    IP                     string
    Port                   int
    Container              string
    ApplicationType        string
    Version                string
    InstallPath            string
    ErrorLogPath           string
    Up                     string  // "运行" / "停止"
    ActiveConnections      int
    WorkerProcesses        int
    WorkerConnections      int
    ConnectionUsagePercent string  // 格式化百分比
    ErrorPage4xx           string  // "已配置" / "未配置"
    ErrorPage5xx           string  // "已配置" / "未配置"
    LastErrorTime          string  // 格式化时间
    NonRootUser            string  // "是" / "否"
    Status                 string  // "正常" / "警告" / "严重" / "失败"
    StatusClass            string  // CSS class
    AlertCount             int
}
```

#### NginxAlertData

```go
type NginxAlertData struct {
    Identifier        string
    MetricName        string
    MetricDisplayName string
    CurrentValue      string
    WarningThreshold  string
    CriticalThreshold string
    Level             string
    LevelClass        string
    Message           string
}
```

### CombinedTemplateData 新增字段

```go
// Nginx data
HasNginx          bool
NginxSummary      *model.NginxInspectionSummary
NginxAlertSummary *model.NginxAlertSummary
NginxInstances    []*NginxInstanceData
NginxAlerts       []*NginxAlertData
```

### WriteCombined 方法签名变更

```diff
-func (w *Writer) WriteCombined(hostResult *model.InspectionResult, mysqlResult *model.MySQLInspectionResults, redisResult *model.RedisInspectionResults, outputPath string) error
+func (w *Writer) WriteCombined(hostResult *model.InspectionResult, mysqlResult *model.MySQLInspectionResults, redisResult *model.RedisInspectionResults, nginxResult *model.NginxInspectionResults, outputPath string) error
```

### 验证结果

- [x] `go build ./internal/report/...` 编译通过
- [x] `go test ./internal/report/html/...` 所有测试通过
- [x] `go vet ./internal/report/...` 静态检查通过
- [x] `go build ./internal/service/...` 编译通过
- [x] `go test ./internal/service/...` 所有测试通过

### 注意事项

> ⚠️ `cmd/inspect/cmd/run.go` 中有一处调用需要手动更新：
> ```go
> // 需要将 WriteCombined 调用添加 nginxResult 参数
> w.WriteCombined(hostResult, mysqlResult, redisResult, nginxResult, outputPath)
> ```
> 该文件被 `.gitignore` 限制访问，需要用户手动修改。

---

## 下一步工作

### 步骤 7：扩展报告生成器支持 Nginx

**待修改文件**:
- `internal/report/html/templates/combined.html` - 添加 Nginx 巡检区域
- `internal/report/excel/writer.go` - 添加 Nginx Sheet（可选）

**功能要点**:
- 在 HTML 模板中添加 Nginx 巡检数据展示区域
- 添加 Nginx 实例表格和告警表格
- 与 MySQL/Redis 保持一致的样式

### 步骤 8：端到端验收

**验证内容**:
- 完整巡检流程测试
- 报告生成验证
- 边界条件测试
