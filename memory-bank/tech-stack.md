# 系统巡检工具 - 技术栈方案

## 1. 概述

本文档基于 `product-requirement-document.md` 中的产品需求，提供完整的技术栈推荐方案。

### 1.1 设计原则

| 原则       | 说明                                     |
| ---------- | ---------------------------------------- |
| **合适性** | 高效、可靠地支持所有核心功能与非功能需求 |
| **简单性** | 架构清晰、学习曲线平缓、易于维护         |
| **健壮性** | 性能、安全性、可扩展性表现稳健           |

---

## 2. 技术栈选型

### 2.1 核心语言：Go 1.25

**选择 Go 而非 Node.js 的理由：**

| 维度           | Go                                                   | Node.js                              |
| -------------- | ---------------------------------------------------- | ------------------------------------ |
| **并发处理**   | goroutine 原生支持，轻量高效，适合 100+ 主机并行查询 | 基于事件循环，CPU 密集型任务表现一般 |
| **部署简便性** | 编译为单一二进制文件，零依赖部署                     | 需要 Node.js 运行时和 node_modules   |
| **性能**       | 编译型语言，内存占用低，启动快                       | 解释型，内存占用较高                 |
| **生态契合**   | 夜莺（N9E）本身是 Go 编写，API 客户端可参考官方实现  | 无直接生态优势                       |
| **Excel 生成** | excelize 库功能完善，支持条件格式                    | exceljs 功能类似但性能稍逊           |
| **运维友好**   | 单文件部署，适合 cron 定时任务                       | 需要 pm2 等进程管理                  |

### 2.2 核心依赖库

```go
// go.mod 核心依赖
module inspection-tool

go 1.25

require (
    // CLI 框架
    github.com/spf13/cobra v1.10.2

    // 配置管理
    github.com/spf13/viper v1.21.0

    // HTTP 客户端（简化 REST 调用）
    github.com/go-resty/resty/v2 v2.17.0

    // Excel 生成（支持条件格式、样式）
    github.com/xuri/excelize/v2 v2.10.0

    // 结构化日志
    github.com/rs/zerolog v1.34.0

    // 并发控制
    golang.org/x/sync v0.19.0

    // 数据验证
    github.com/go-playground/validator/v10 v10.29.0
)
```

### 2.3 依赖库选型理由

| 库          | 用途        | 选型理由                                   |
| ----------- | ----------- | ------------------------------------------ |
| `cobra`     | CLI 框架    | 业界标准，kubectl/docker 等均使用          |
| `viper`     | 配置管理    | 支持 YAML/JSON/环境变量，与 cobra 无缝集成 |
| `resty`     | HTTP 客户端 | 链式调用，自动重试，简化 REST API 调用     |
| `excelize`  | Excel 生成  | 功能最完善的 Go Excel 库，支持条件格式     |
| `zerolog`   | 日志        | 零内存分配，JSON 结构化输出，性能优秀      |
| `errgroup`  | 并发控制    | 官方扩展库，优雅处理并发错误               |
| `validator` | 数据验证    | 声明式验证，减少样板代码                   |

---

## 3. 系统架构

### 3.1 整体架构图

```
┌─────────────────────────────────────────────────────────────────────┐
│                           CLI Layer                                  │
│                    (cobra - 命令行解析)                               │
│              inspect run | inspect validate | inspect version        │
└─────────────────────────────────────────────────────────────────────┘
                                   │
                                   ▼
┌─────────────────────────────────────────────────────────────────────┐
│                         Service Layer                                │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐      │
│  │   Inspector     │  │   Evaluator     │  │   Reporter      │      │
│  │   (巡检编排)     │  │   (阈值评估)     │  │   (报告生成)     │      │
│  │                 │  │                 │  │                 │      │
│  │ - 协调数据采集   │  │ - 阈值判断      │  │ - 格式化输出     │      │
│  │ - 并发控制      │  │ - 告警分级      │  │ - 多格式支持     │      │
│  │ - 结果聚合      │  │ - 异常标记      │  │ - 模板渲染      │      │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘      │
└─────────────────────────────────────────────────────────────────────┘
                                   │
                                   ▼
┌─────────────────────────────────────────────────────────────────────┐
│                         Client Layer                                 │
│  ┌─────────────────────────────┐  ┌─────────────────────────────┐   │
│  │       N9E Client            │  │       VM Client             │   │
│  │                             │  │                             │   │
│  │ - 主机列表查询               │  │ - 即时查询 /api/v1/query    │   │
│  │ - 主机元信息获取             │  │ - 范围查询 /api/v1/query_range │
│  │ - Token 认证                │  │ - 批量指标查询              │   │
│  └─────────────────────────────┘  └─────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────┘
                                   │
                                   ▼
┌─────────────────────────────────────────────────────────────────────┐
│                         Report Layer                                 │
│  ┌───────────────┐  ┌───────────────┐  ┌───────────────┐            │
│  │ ExcelWriter   │  │ HTMLWriter    │  │ PDFWriter     │            │
│  │               │  │               │  │ (预留接口)     │            │
│  │ - 多工作表    │  │ - 响应式布局   │  │               │            │
│  │ - 条件格式    │  │ - 模板渲染    │  │               │            │
│  │ - 样式美化    │  │ - 图表支持    │  │               │            │
│  └───────────────┘  └───────────────┘  └───────────────┘            │
└─────────────────────────────────────────────────────────────────────┘
```

### 3.2 数据流图

```
┌──────────┐     ┌──────────┐     ┌──────────┐     ┌──────────┐
│  配置加载 │────▶│  数据采集 │────▶│  阈值评估 │────▶│  报告生成 │
└──────────┘     └──────────┘     └──────────┘     └──────────┘
     │                │                │                │
     ▼                ▼                ▼                ▼
 config.yaml    N9E + VM API     告警分级判断     Excel + HTML
```

### 3.3 核心接口设计

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

## 4. 目录结构

```
inspection-tool/
├── cmd/
│   └── inspect/
│       └── main.go                 # 程序入口
│
├── internal/
│   ├── config/
│   │   ├── config.go               # 配置结构定义
│   │   ├── loader.go               # 配置加载（支持 YAML + 环境变量）
│   │   └── validator.go            # 配置校验
│   │
│   ├── client/
│   │   ├── n9e/
│   │   │   ├── client.go           # 夜莺 API 客户端
│   │   │   ├── types.go            # 请求/响应类型定义
│   │   │   └── client_test.go      # 单元测试
│   │   └── vm/
│   │       ├── client.go           # VictoriaMetrics 客户端
│   │       ├── types.go            # PromQL 查询类型
│   │       └── client_test.go
│   │
│   ├── model/
│   │   ├── host.go                 # 主机模型
│   │   ├── metric.go               # 指标模型
│   │   ├── inspection.go           # 巡检结果模型
│   │   └── alert.go                # 告警模型
│   │
│   ├── service/
│   │   ├── inspector.go            # 巡检服务（核心编排逻辑）
│   │   ├── collector.go            # 数据采集服务
│   │   ├── evaluator.go            # 阈值评估服务
│   │   └── inspector_test.go
│   │
│   └── report/
│       ├── writer.go               # ReportWriter 接口定义
│       ├── registry.go             # 报告格式注册表
│       ├── excel/
│       │   ├── writer.go           # Excel 报告生成
│       │   ├── style.go            # 样式定义（条件格式）
│       │   └── writer_test.go
│       └── html/
│           ├── writer.go           # HTML 报告生成
│           └── templates/
│               ├── report.html     # 主模板
│               └── partials/
│                   ├── header.html
│                   ├── summary.html
│                   └── table.html
│
├── configs/
│   ├── config.example.yaml         # 配置示例（含详细注释）
│   └── metrics.yaml                # 指标定义（可自定义）
│
├── templates/
│   └── html/
│       └── report.tmpl             # HTML 报告模板
│
├── scripts/
│   ├── build.sh                    # 构建脚本
│   └── release.sh                  # 发布脚本
│
├── go.mod
├── go.sum
├── Makefile                        # 构建、测试、发布命令
├── README.md
└── CLAUDE.md
```

---

## 5. 配置文件设计

### 5.1 主配置文件 (config.yaml)

```yaml
# 数据源配置
datasources:
  n9e:
    endpoint: "http://${nightingale_api_address}:17000"
    token: "${N9E_TOKEN}" # 支持环境变量
    timeout: 30s
  victoriametrics:
    endpoint: "http://${nightingale_api_address}:8428"
    timeout: 30s

# 巡检配置
inspection:
  # 并发控制
  concurrency: 20
  # 单主机超时
  host_timeout: 10s
  # 主机筛选（可选）
  host_filter:
    business_groups: [] # 按业务组筛选
    tags: {} # 按标签筛选

# 阈值配置
thresholds:
  cpu_usage:
    warning: 70
    critical: 90
  memory_usage:
    warning: 70
    critical: 90
  disk_usage:
    warning: 70
    critical: 90
  zombie_processes:
    warning: 1
    critical: 10
  load_per_core:
    warning: 0.7
    critical: 1.0

# 报告配置
report:
  # 输出目录
  output_dir: "./reports"
  # 输出格式
  formats:
    - excel
    - html
  # 文件名模板
  filename_template: "inspection_report_{{.Date}}"

# 日志配置
logging:
  level: info # debug, info, warn, error
  format: json # json, console
```

### 5.2 指标定义文件 (metrics.yaml)

```yaml
# 指标定义 - 支持自定义扩展
metrics:
  # CPU 相关
  - name: cpu_usage
    display_name: "CPU 利用率"
    query: 'cpu_usage_active{cpu="cpu-total"}'
    unit: "%"
    category: cpu

  # 内存相关
  - name: memory_usage
    display_name: "内存利用率"
    query: "100 - mem_available_percent"
    unit: "%"
    category: memory

  - name: memory_total
    display_name: "内存总量"
    query: "mem_total"
    unit: "bytes"
    format: "size" # 自动转换为 GB
    category: memory

  # 磁盘相关
  - name: disk_usage
    display_name: "磁盘利用率"
    query: "disk_used_percent"
    unit: "%"
    category: disk
    aggregate: max # 多磁盘取最大值

  # 系统相关
  - name: uptime
    display_name: "运行时间"
    query: "system_uptime"
    unit: "seconds"
    format: "duration" # 转换为天/小时
    category: system

  - name: load_1m
    display_name: "1分钟负载"
    query: "system_load1"
    unit: ""
    category: system

  # 进程相关
  - name: processes_total
    display_name: "总进程数"
    query: "processes_total"
    unit: "个"
    category: process

  - name: processes_zombies
    display_name: "僵尸进程数"
    query: "processes_zombies"
    unit: "个"
    category: process
```

---

## 6. 核心数据模型

```go
// internal/model/inspection.go

// 巡检结果
type InspectionResult struct {
    InspectionTime time.Time       `json:"inspection_time"`
    Duration       time.Duration   `json:"duration"`
    Summary        Summary         `json:"summary"`
    Hosts          []HostResult    `json:"hosts"`
    Alerts         []Alert         `json:"alerts"`
}

// 摘要统计
type Summary struct {
    TotalHosts    int `json:"total_hosts"`
    NormalHosts   int `json:"normal_hosts"`
    WarningHosts  int `json:"warning_hosts"`
    CriticalHosts int `json:"critical_hosts"`
    FailedHosts   int `json:"failed_hosts"`
}

// 单主机巡检结果
type HostResult struct {
    Hostname    string            `json:"hostname"`
    IP          string            `json:"ip"`
    OS          string            `json:"os"`
    OSVersion   string            `json:"os_version"`
    Kernel      string            `json:"kernel"`
    Status      HostStatus        `json:"status"`  // normal, warning, critical
    Metrics     map[string]Metric `json:"metrics"`
    Alerts      []Alert           `json:"alerts"`
    CollectedAt time.Time         `json:"collected_at"`
    Error       string            `json:"error,omitempty"`
}

// 指标值
type Metric struct {
    Name         string      `json:"name"`
    DisplayName  string      `json:"display_name"`
    Value        float64     `json:"value"`
    FormattedVal string      `json:"formatted_value"`
    Unit         string      `json:"unit"`
    Status       AlertLevel  `json:"status"`
}

// 告警
type Alert struct {
    Host       string     `json:"host"`
    MetricName string     `json:"metric_name"`
    Level      AlertLevel `json:"level"`
    Value      float64    `json:"value"`
    Threshold  float64    `json:"threshold"`
    Message    string     `json:"message"`
}

type AlertLevel string

const (
    AlertLevelNormal   AlertLevel = "normal"
    AlertLevelWarning  AlertLevel = "warning"
    AlertLevelCritical AlertLevel = "critical"
)
```

---

## 7. 并发处理策略

```go
// Worker Pool 模式处理 100+ 主机
func (i *Inspector) collectWithWorkerPool(ctx context.Context, hosts []string) ([]HostData, error) {
    const maxWorkers = 20  // 控制并发数，避免 API 压力过大

    g, ctx := errgroup.WithContext(ctx)
    g.SetLimit(maxWorkers)

    results := make(chan HostData, len(hosts))

    for _, host := range hosts {
        host := host
        g.Go(func() error {
            data, err := i.collectSingleHost(ctx, host)
            if err != nil {
                // 单主机失败不影响整体，记录错误继续
                i.logger.Warn().Str("host", host).Err(err).Msg("collect failed")
                return nil
            }
            results <- data
            return nil
        })
    }

    // 等待所有任务完成
    if err := g.Wait(); err != nil {
        return nil, err
    }
    close(results)

    // 收集结果
    var hostDataList []HostData
    for data := range results {
        hostDataList = append(hostDataList, data)
    }
    return hostDataList, nil
}
```

---

## 8. 部署与运维

### 8.1 构建配置 (Makefile)

```makefile
VERSION := $(shell git describe --tags --always --dirty)
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS := -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)

.PHONY: build
build:
	go build -ldflags "$(LDFLAGS)" -o bin/inspect ./cmd/inspect

.PHONY: build-all
build-all:
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/inspect-linux-amd64 ./cmd/inspect
	GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/inspect-darwin-amd64 ./cmd/inspect
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/inspect-windows-amd64.exe ./cmd/inspect

.PHONY: test
test:
	go test -v -race -cover ./...

.PHONY: lint
lint:
	golangci-lint run
```

### 8.2 运行方式

```bash
# 基本使用
./inspect run -c config.yaml

# 指定输出格式
./inspect run -c config.yaml --format excel,html

# 指定输出目录
./inspect run -c config.yaml --output ./reports

# 验证配置
./inspect validate -c config.yaml

# 查看版本
./inspect version

# 定时任务（cron）
0 8 * * * /opt/inspection-tool/inspect run -c /etc/inspect/config.yaml >> /var/log/inspect.log 2>&1
```

### 8.3 Docker 部署（可选）

```dockerfile
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o inspect ./cmd/inspect

FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
COPY --from=builder /app/inspect /usr/local/bin/
COPY configs/config.example.yaml /etc/inspect/config.yaml
ENTRYPOINT ["inspect"]
CMD ["run", "-c", "/etc/inspect/config.yaml"]
```

### 8.4 安全考虑

| 安全维度         | 措施                                           |
| ---------------- | ---------------------------------------------- |
| **敏感信息保护** | Token 支持环境变量注入，不硬编码；日志自动脱敏 |
| **文件权限**     | 配置文件建议权限设置为 `600`                   |
| **网络安全**     | 支持 HTTPS 连接，可配置 TLS 证书验证           |
| **资源限制**     | 并发数可配置，超时机制防止长时间阻塞           |

---

## 9. 实现路线图

### Phase 1 - MVP（预计 3-5 天）

- [ ] 项目初始化、目录结构
- [ ] 配置加载模块
- [ ] 夜莺 API 客户端（主机列表、元信息）
- [ ] VictoriaMetrics 客户端（即时查询）
- [ ] 基础 Excel 报告生成
- [ ] 基础 HTML 报告生成
- [ ] CLI 命令实现

### Phase 2 - 增强（预计 2-3 天）

- [ ] 条件格式（异常值高亮）
- [ ] 并发优化与错误重试
- [ ] 阈值评估与告警分级
- [ ] 报告样式美化
- [ ] 单元测试覆盖

### Phase 3 - 扩展（按需）

- [ ] PDF 报告支持
- [ ] 邮件发送功能
- [ ] 自定义报告模板
- [ ] Web UI 界面
- [ ] Prometheus 指标暴露（自监控）

---

## 10. 总结

### 10.1 技术栈一览

| 维度            | 方案                 |
| --------------- | -------------------- |
| **语言**        | Go 1.25              |
| **CLI 框架**    | cobra + viper        |
| **HTTP 客户端** | resty                |
| **Excel 生成**  | excelize             |
| **HTML 模板**   | 标准库 html/template |
| **日志**        | zerolog              |
| **并发控制**    | errgroup             |
| **配置格式**    | YAML                 |
| **部署方式**    | 单二进制 / Docker    |

### 10.2 核心优势

1. **高效并发**：Go 原生 goroutine 支持，轻松处理 100+ 主机并行查询
2. **简单部署**：单二进制文件，零依赖，运维友好
3. **易于扩展**：接口化设计，支持新增报告格式和巡检项
4. **配置驱动**：YAML 配置文件，支持自定义巡检项和阈值
5. **生态契合**：与夜莺（N9E）同为 Go 技术栈，可参考官方实现

---

## 版本记录

| 版本 | 日期       | 说明     |
| ---- | ---------- | -------- |
| v1.0 | 2025-12-13 | 初始版本 |
