# Nginx 巡检功能 - 开发进度记录

## 实施进度

| 阶段 | 步骤 | 状态 | 完成日期 |
|------|------|------|----------|
| 一、采集配置 | 1. 部署采集脚本 | ✅ 已完成 | 2025-12-20 |
| | 2. 配置 exec 插件 | ✅ 已完成 | 2025-12-20 |
| 二、数据模型 | 3. 定义 Nginx 数据模型 | ✅ 已完成 | 2025-12-21 |
| | 4. 扩展配置结构 | ✅ 已完成 | 2025-12-21 |
| 三、服务实现 | 5. 实现采集器和评估器 | ⏳ 待开始 | - |
| | 6. 集成到主服务 | ⏳ 待开始 | - |
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

## 下一步工作

步骤 5：实现 Nginx 采集器和评估器

**待创建文件**:
- `internal/service/nginx_collector.go` - Nginx 数据采集器
- `internal/service/nginx_evaluator.go` - Nginx 阈值评估器

**功能要点**:
- 查询 `nginx_info` 发现实例
- 采集 nginx_* 指标（插件 + exec）
- 采集 `nginx_upstream_check_*` 指标
- 连接使用率评估
- 错误日志时间评估
- Upstream 后端状态评估

**待审核后开始实施**
