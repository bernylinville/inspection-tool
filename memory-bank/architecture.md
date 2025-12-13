# 系统巡检工具 - 架构文档

## 项目概述

基于监控数据的无侵入式系统巡检工具，通过调用夜莺（N9E）和 VictoriaMetrics API 查询监控数据，生成 Excel 和 HTML 格式的巡检报告。

---

## 当前文件结构

```
inspection-tool/
├── go.mod                      # Go 模块定义
├── go.sum                      # 依赖校验文件
├── CLAUDE.md                   # Claude Code 项目指令
├── mise.toml                   # mise 版本管理配置
├── n9e.json                    # 夜莺 API 响应示例数据
├── categraf-linux-metrics.json # Categraf 指标定义参考
├── bin/
│   └── inspect                 # 编译后的二进制文件（已生成）
├── cmd/
│   └── inspect/
│       └── main.go             # 程序入口（已实现）
├── configs/                    # 配置文件示例
│   ├── config.example.yaml     # 主配置示例（已创建）
│   └── metrics.yaml            # 指标定义文件（已创建）
├── internal/
│   ├── client/
│   │   ├── n9e/                # 夜莺 API 客户端（待实现）
│   │   └── vm/                 # VictoriaMetrics 客户端（待实现）
│   ├── config/
│   │   └── config.go          # 配置结构体定义（已实现）
│   ├── model/                  # 数据模型
│   │   ├── host.go             # 主机模型（已实现）
│   │   └── metric.go           # 指标模型（已实现）
│   ├── report/
│   │   ├── excel/              # Excel 报告生成（待实现）
│   │   └── html/               # HTML 报告生成（待实现）
│   └── service/                # 业务逻辑（待实现）
├── templates/
│   └── html/                   # 用户自定义 HTML 模板（外置）
└── memory-bank/
    ├── product-requirement-document.md  # 产品需求文档
    ├── tech-stack.md                    # 技术栈方案
    ├── implementation-plan.md           # 开发实施计划
    ├── progress.md                      # 开发进度记录
    └── architecture.md                  # 架构文档（本文件）
```

---

## 目录说明

### 程序入口 (cmd/inspect/)

| 文件 | 作用 | 状态 |
|------|------|------|
| `main.go` | 程序入口，版本信息输出，后续集成 cobra CLI | ✅ 已实现 |

**版本信息变量**（通过 `-ldflags` 注入）：
```go
var (
    Version   = "dev"      // 版本号
    BuildTime = "unknown"  // 构建时间
    GitCommit = "unknown"  // Git 提交哈希
)
```

### 配置管理 (internal/config/)

| 文件 | 作用 | 状态 |
|------|------|------|
| `config.go` | 配置结构体定义 | ✅ 已实现 |
| `loader.go` | 配置加载（YAML + 环境变量） | ✅ 已实现 |
| `loader_test.go` | 配置加载器单元测试 | ✅ 已实现 |
| `validator.go` | 配置验证（必填、格式、范围、阈值逻辑） | ✅ 已实现 |
| `validator_test.go` | 配置验证器单元测试 | ✅ 已实现 |

**配置结构体概览**：
```go
type Config struct {
    Datasources DatasourcesConfig  // N9E + VictoriaMetrics 数据源配置
    Inspection  InspectionConfig   // 并发数、超时、主机筛选
    Thresholds  ThresholdsConfig   // CPU/内存/磁盘/僵尸进程/负载阈值
    Report      ReportConfig       // 输出目录、格式、模板、时区
    Logging     LoggingConfig      // 日志级别、格式
    HTTP        HTTPConfig         // 重试次数、延迟（指数退避）
}
```

### API 客户端 (internal/client/)

| 目录/文件 | 作用 |
|-----------|------|
| `n9e/client.go` | 夜莺 API 客户端（主机元信息） |
| `n9e/types.go` | 夜莺请求/响应类型定义 |
| `vm/client.go` | VictoriaMetrics 客户端（指标查询） |
| `vm/types.go` | PromQL 查询类型定义 |

### 数据模型 (internal/model/)

| 文件 | 作用 | 状态 |
|------|------|------|
| `host.go` | 主机模型（HostMeta、DiskMountInfo、HostStatus） | ✅ 已实现 |
| `metric.go` | 指标模型（MetricDefinition、MetricValue、HostMetrics） | ✅ 已实现 |
| `alert.go` | 告警模型（级别、阈值） | 待实现 |
| `inspection.go` | 巡检结果模型（摘要、主机列表） | 待实现 |

**主机模型结构**：
```go
type HostMeta struct {
    Ident         string          `json:"ident"`          // 原始标识符
    Hostname      string          `json:"hostname"`       // 主机名
    IP            string          `json:"ip"`             // IP 地址
    OS            string          `json:"os"`             // 操作系统
    OSVersion     string          `json:"os_version"`     // 系统版本
    KernelVersion string          `json:"kernel_version"` // 内核版本
    CPUCores      int             `json:"cpu_cores"`      // CPU 核心数
    CPUModel      string          `json:"cpu_model"`      // CPU 型号
    MemoryTotal   int64           `json:"memory_total"`   // 内存总量
    DiskMounts    []DiskMountInfo `json:"disk_mounts"`    // 磁盘挂载点
}
```

**指标模型结构**：
```go
// 指标定义（从 metrics.yaml 加载）
type MetricDefinition struct {
    Name          string         `yaml:"name"`           // 指标唯一标识
    DisplayName   string         `yaml:"display_name"`   // 中文显示名称
    Query         string         `yaml:"query"`          // PromQL 查询表达式
    Unit          string         `yaml:"unit"`           // 单位
    Category      MetricCategory `yaml:"category"`       // 分类
    Format        MetricFormat   `yaml:"format"`         // 格式化类型
    Aggregate     AggregateType  `yaml:"aggregate"`      // 聚合方式
    ExpandByLabel string         `yaml:"expand_by_label"`// 按标签展开
    Status        string         `yaml:"status"`         // pending=待实现
}

// 指标采集值
type MetricValue struct {
    Name           string            `json:"name"`            // 指标名称
    RawValue       float64           `json:"raw_value"`       // 原始数值
    FormattedValue string            `json:"formatted_value"` // 格式化后的值
    Status         MetricStatus      `json:"status"`          // 评估状态
    Labels         map[string]string `json:"labels"`          // 标签
    IsNA           bool              `json:"is_na"`           // 是否为 N/A
}
```

### 业务逻辑 (internal/service/)

| 文件 | 作用 |
|------|------|
| `inspector.go` | 巡检编排服务（核心流程） |
| `collector.go` | 数据采集服务 |
| `evaluator.go` | 阈值评估服务 |

### 报告生成 (internal/report/)

| 目录/文件 | 作用 |
|-----------|------|
| `writer.go` | ReportWriter 接口定义 |
| `registry.go` | 报告格式注册表 |
| `excel/writer.go` | Excel 报告生成 |
| `html/writer.go` | HTML 报告生成 |

### 配置文件 (configs/)

| 文件 | 作用 | 状态 |
|------|------|------|
| `config.example.yaml` | 主配置示例（数据源、阈值、报告设置） | ✅ 已创建 |
| `metrics.yaml` | 指标定义（PromQL 查询表达式，23 个指标） | ✅ 已创建 |

### 用户模板 (templates/html/)

| 文件 | 作用 |
|------|------|
| `report.tmpl` | 用户自定义 HTML 报告模板（可选） |

---

## 核心数据流

```
配置加载 → 数据采集(N9E + VM) → 阈值评估 → 报告生成(Excel/HTML)
```

---

## 核心接口

```go
// 报告写入器接口 - 支持扩展新格式
type ReportWriter interface {
    Write(result *InspectionResult, outputPath string) error
    Format() string  // 返回格式名称：excel, html, pdf
}

// 数据采集器接口
type Collector interface {
    CollectHostMeta(ctx context.Context, hosts []string) ([]HostMeta, error)
    CollectMetrics(ctx context.Context, hosts []string, metrics []MetricDef) ([]HostMetrics, error)
}

// 阈值评估器接口
type Evaluator interface {
    Evaluate(metrics *HostMetrics, thresholds []Threshold) []Alert
}
```

---

## 版本记录

| 日期 | 变更 |
|------|------|
| 2025-12-13 | 初始版本，完成步骤 1（Go 模块初始化） |
| 2025-12-13 | 完成步骤 2（创建目录结构），更新架构文档 |
| 2025-12-13 | 完成步骤 4（程序入口），阶段一全部完成 |
| 2025-12-13 | 完成步骤 5（配置结构体），阶段二开始 |
| 2025-12-13 | 完成步骤 6（配置加载器），添加 loader.go 和测试文件 |
| 2025-12-13 | 完成步骤 7（配置验证器），添加 validator.go 和测试文件 |
| 2025-12-13 | 完成步骤 8（示例配置文件），添加 config.example.yaml |
| 2025-12-13 | 完成步骤 9（指标定义文件），添加 metrics.yaml，阶段二完成 |
| 2025-12-13 | 完成步骤 10（主机模型），添加 host.go，阶段三开始 |
| 2025-12-13 | 完成步骤 11（指标模型），添加 metric.go |
