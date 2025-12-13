# 系统巡检工具 - 开发实施计划

> 本计划聚焦于 MVP 核心功能实现，共 41 个步骤，按依赖关系和优先级排序。

---

## 关键技术决策（已确认）

本节记录在实施前已确认的技术决策，确保实施过程中方向明确。

### 1. 夜莺 API 数据结构

**API 文档**：https://n9e.github.io/docs/usecase/api/

**响应结构示例**（基于 `n9e.json`）：

```json
{
  "dat": {
    "ident": "sd-k8s-master-1",
    "extend_info": "{\"cpu\":{...},\"memory\":{...},\"network\":{...},\"platform\":{...},\"filesystem\":[...]}"
  },
  "err": ""
}
```

**关键字段提取**（从 `extend_info` JSON 字符串解析）：

| 字段 | 路径 | 示例值 |
|------|------|--------|
| 主机名 | `platform.hostname` | `sd-k8s-master-1` |
| 操作系统 | `platform.os` | `GNU/Linux` |
| 内核版本 | `platform.kernel_release` | `5.14.0-503.38.1.el9_5.x86_64` |
| IP 地址 | `network.ipaddress` | `192.168.10.24` |
| CPU 核心数 | `cpu.cpu_cores` | `4` |
| CPU 型号 | `cpu.model_name` | `Intel(R) Xeon(R) Platinum 8378C CPU @ 2.80GHz` |
| 内存总量 | `memory.total` | `16496934912` (bytes) |

### 2. 主机标识匹配规则

- **匹配方式**：通过主机名（hostname）匹配
- **ident 清理**：ident 可能出现 `主机名@IP` 格式（如 `server-1@192.168.1.100`），需要提取 `@` 前的部分作为主机名
- **实现**：`strings.Split(ident, "@")[0]`

### 3. "待定" 巡检项处理策略

MVP 阶段策略：**预留接口，后续扩展**

| 巡检项 | MVP 处理 |
|--------|----------|
| 公网访问检查 | 预留接口，报告中显示 "N/A" |
| 密码过期(天) | 预留接口，报告中显示 "N/A" |
| 密码策略 | 预留接口，报告中显示 "N/A" |
| NTP 检查 | 预留接口，报告中显示 "N/A" |
| 打开文件句柄数 | 预留接口，报告中显示 "N/A" |
| 系统参数检查 | 预留接口，报告中显示 "N/A" |

### 4. 磁盘指标处理

- **展示方式**：分别显示每个挂载点的磁盘使用情况
- **聚合规则**：
  - `disk_used_percent`：取所有挂载点的最大值用于告警判断
  - `disk_total`：各挂载点单独显示，不聚合
  - `disk_free`：各挂载点单独显示，不聚合
- **过滤规则**：仅显示物理磁盘挂载点（排除 tmpfs、overlay 等）

### 5. 主机筛选逻辑

- **业务组筛选**：多个业务组之间是 **OR** 关系
- **标签筛选**：业务组和标签之间是 **AND** 关系
- **实现位置**：通过 PromQL 标签选择器在 VictoriaMetrics 查询时过滤

**示例 PromQL**：
```promql
cpu_usage_active{cpu="cpu-total", busigroup=~"group1|group2", env="prod"}
```

### 6. CPU 核心数来源

- **主要来源**：夜莺元信息 `extend_info.cpu.cpu_cores`
- **备选来源**：VictoriaMetrics `system_n_cpus` 指标

### 7. HTML 模板策略

- **设计原则**：模板外置，用户可自定义
- **模板位置**：`templates/html/report.tmpl`
- **技术选型**：MVP 使用纯 HTML + CSS + 内联 JS，后续可扩展为 Vue 等框架
- **加载顺序**：优先加载用户自定义模板，不存在时使用内置默认模板

### 8. 时区处理

- **统一时区**：`Asia/Shanghai`（中国时区）
- **影响范围**：巡检时间、最后重启时间、报告生成时间

### 9. 错误重试策略

| 配置项 | 默认值 | 说明 |
|--------|--------|------|
| 最大重试次数 | 3 | 单个请求最多重试 3 次 |
| 重试间隔 | 1s, 2s, 4s | 指数退避策略 |
| 可重试错误 | 超时、5xx、连接失败 | 4xx 错误不重试 |

### 10. Excel 条件格式

| 状态 | 背景色 | 说明 |
|------|--------|------|
| 正常 | 无 | 默认无背景 |
| 警告 | 黄色 `#FFEB9C` | 达到警告阈值 |
| 严重 | 红色 `#FFC7CE` | 达到严重阈值 |

### 11. HTML 排序功能

- **MVP 实现**：使用纯前端 JavaScript 实现客户端排序
- **排序列**：主机名、CPU%、内存%、磁盘%、状态
- **默认排序**：按状态严重程度降序（严重 > 警告 > 正常）

### 12. VictoriaMetrics 查询策略

- **查询方式**：一次性查询所有主机的某个指标（批量查询）
- **示例**：`cpu_usage_active{cpu="cpu-total"}` 返回所有主机的 CPU 数据
- **分批处理**：主机数量超过 500 时，按 500 个一批分批查询
- **并发控制**：最多 20 个并发查询

### 13. 测试覆盖率目标

| 模块 | 目标覆盖率 |
|------|------------|
| client (N9E/VM) | ≥ 70% |
| service (业务逻辑) | ≥ 80% |
| report (报告生成) | ≥ 60% |
| 整体 | ≥ 70% |

---

## 阶段一：项目初始化与基础架构（步骤 1-4）

### 步骤 1：初始化 Go 模块

**操作**：
- 在项目根目录执行 Go 模块初始化，模块名为 `inspection-tool`
- 设置 Go 版本为 1.25

**验证**：
- [ ] `go.mod` 文件存在且包含正确的模块名和 Go 版本
- [ ] 执行 `go mod tidy` 无报错

---

### 步骤 2：创建目录结构

**操作**：
按照技术栈文档创建以下目录结构：
- `cmd/inspect/` - 程序入口
- `internal/config/` - 配置管理
- `internal/client/n9e/` - 夜莺客户端
- `internal/client/vm/` - VictoriaMetrics 客户端
- `internal/model/` - 数据模型
- `internal/service/` - 业务逻辑
- `internal/report/excel/` - Excel 报告
- `internal/report/html/` - HTML 报告
- `configs/` - 配置文件示例
- `templates/html/` - 用户自定义 HTML 报告模板（外置）

**验证**：
- [ ] 所有目录已创建
- [ ] 使用 `tree` 或 `ls -R` 命令确认目录结构完整

---

### 步骤 3：添加核心依赖

**操作**：
依次添加以下依赖包：
1. CLI 框架：`github.com/spf13/cobra`
2. 配置管理：`github.com/spf13/viper`
3. HTTP 客户端：`github.com/go-resty/resty/v2`
4. Excel 生成：`github.com/xuri/excelize/v2`
5. 结构化日志：`github.com/rs/zerolog`
6. 并发控制：`golang.org/x/sync`
7. 数据验证：`github.com/go-playground/validator/v10`

**验证**：
- [ ] `go.mod` 中包含所有依赖
- [ ] 执行 `go mod download` 成功
- [ ] 执行 `go mod verify` 无错误

---

### 步骤 4：创建程序入口文件

**操作**：
- 在 `cmd/inspect/` 目录下创建 `main.go`
- 实现最简单的 main 函数，仅打印版本信息

**验证**：
- [ ] 执行 `go build -o bin/inspect ./cmd/inspect` 成功
- [ ] 运行 `./bin/inspect` 输出版本信息
- [ ] 二进制文件大小合理（应小于 20MB）

---

## 阶段二：配置管理模块（步骤 5-9）

### 步骤 5：定义配置结构体

**操作**：
- 在 `internal/config/config.go` 中定义配置结构体
- 包含以下配置节：
  - 数据源配置（N9E 和 VictoriaMetrics 的地址、认证信息、超时时间）
  - 巡检配置（并发数、主机超时、主机筛选条件）
  - 阈值配置（CPU、内存、磁盘、僵尸进程、负载的警告和严重阈值）
  - 报告配置（输出目录、输出格式、文件名模板、HTML 模板路径）
  - 日志配置（级别、格式）
  - HTTP 重试配置（最大重试次数、重试间隔）
- 为所有字段添加 `mapstructure` 和 `validate` 标签

**配置结构参考**：
```go
type RetryConfig struct {
    MaxRetries int           `mapstructure:"max_retries" validate:"gte=0,lte=10"`
    BaseDelay  time.Duration `mapstructure:"base_delay"`
}

type HostFilter struct {
    BusinessGroups []string          `mapstructure:"business_groups"` // OR 关系
    Tags           map[string]string `mapstructure:"tags"`            // AND 关系
}
```

**验证**：
- [ ] 结构体定义完整，字段命名符合 Go 规范
- [ ] 所有必填字段都有 `validate:"required"` 标签
- [ ] 执行 `go build ./internal/config/` 无编译错误

---

### 步骤 6：实现配置加载器

**操作**：
- 在 `internal/config/loader.go` 中实现配置加载功能
- 支持从 YAML 文件加载配置
- 支持环境变量覆盖（特别是敏感信息如 Token）
- 实现默认值设置（包括重试策略默认值）
- 设置默认时区为 `Asia/Shanghai`

**默认值**：
```go
defaults := map[string]interface{}{
    "inspection.concurrency":    20,
    "inspection.host_timeout":   "10s",
    "http.retry.max_retries":    3,
    "http.retry.base_delay":     "1s",
    "report.timezone":           "Asia/Shanghai",
}
```

**验证**：
- [ ] 能够正确加载 YAML 配置文件
- [ ] 环境变量能够覆盖配置文件中的值
- [ ] 缺少配置文件时返回明确错误信息
- [ ] 默认值正确应用

---

### 步骤 7：实现配置验证器

**操作**：
- 在 `internal/config/validator.go` 中实现配置验证
- 使用 validator 库验证必填字段
- 验证 URL 格式、数值范围等
- 返回清晰的验证错误信息

**验证**：
- [ ] 缺少必填字段时返回具体的字段名和错误原因
- [ ] 无效 URL 格式能被检测出来
- [ ] 阈值数值超出合理范围时报错

---

### 步骤 8：创建示例配置文件

**操作**：
- 在 `configs/` 目录下创建 `config.example.yaml`
- 包含所有配置项及详细注释说明
- 使用产品需求文档中的实际 API 地址作为示例
- 包含重试策略和时区配置

**配置示例片段**：
```yaml
# HTTP 重试策略
http:
  retry:
    max_retries: 3        # 最大重试次数
    base_delay: 1s        # 基础延迟（指数退避）

# 主机筛选（多业务组为 OR，与标签为 AND）
inspection:
  host_filter:
    business_groups:      # OR 关系
      - "生产环境"
      - "测试环境"
    tags:                 # AND 关系
      env: "prod"

# 报告配置
report:
  timezone: "Asia/Shanghai"
  html_template: "./templates/html/report.tmpl"  # 可选，用户自定义模板
```

**验证**：
- [ ] 配置文件格式正确，可被 YAML 解析器解析
- [ ] 所有配置项都有中文注释说明
- [ ] 敏感信息（Token）使用环境变量占位符

---

### 步骤 9：创建指标定义文件

**操作**：
- 在 `configs/` 目录下创建 `metrics.yaml`
- 定义所有巡检指标的 PromQL 查询表达式
- 包含指标名称、显示名称、查询语句、单位、分类
- 标记 "待定" 巡检项（预留接口）

**指标定义示例**：
```yaml
metrics:
  # 磁盘相关 - 需要按挂载点展开
  - name: disk_usage
    display_name: "磁盘利用率"
    query: 'disk_used_percent{path!~"/dev.*|/run.*|/sys.*"}'
    unit: "%"
    category: disk
    aggregate: max        # 告警判断取最大值
    expand_by_label: path # 按挂载点展开显示

  # 待定巡检项 - 预留接口
  - name: ntp_check
    display_name: "NTP 检查"
    query: ""             # 空查询，显示 N/A
    unit: ""
    category: system
    status: pending       # 标记为待实现
```

**验证**：
- [ ] 包含产品需求文档中定义的所有指标
- [ ] 每个指标都有完整的元数据定义
- [ ] YAML 格式正确无语法错误
- [ ] 待定项正确标记

---

## 阶段三：数据模型定义（步骤 10-13）

### 步骤 10：定义主机模型

**操作**：
- 在 `internal/model/host.go` 中定义主机相关结构体
- 包含：主机基本信息（主机名、IP、操作系统、版本、内核、CPU核心数）
- 定义主机状态枚举（正常、警告、严重、失败）
- 添加磁盘挂载点列表字段

**结构参考**：
```go
type HostMeta struct {
    Ident         string            `json:"ident"`
    Hostname      string            `json:"hostname"`       // 从 ident 清理得到
    IP            string            `json:"ip"`
    OS            string            `json:"os"`
    KernelVersion string            `json:"kernel_version"`
    CPUCores      int               `json:"cpu_cores"`      // 从夜莺元信息获取
    CPUModel      string            `json:"cpu_model"`
    MemoryTotal   int64             `json:"memory_total"`
    DiskMounts    []DiskMountInfo   `json:"disk_mounts"`    // 各挂载点信息
}

type DiskMountInfo struct {
    Path       string  `json:"path"`
    Total      int64   `json:"total"`
    Free       int64   `json:"free"`
    UsedPercent float64 `json:"used_percent"`
}
```

**验证**：
- [ ] 结构体字段覆盖产品需求文档中的所有主机属性
- [ ] JSON 标签正确设置
- [ ] 执行 `go build ./internal/model/` 无编译错误

---

### 步骤 11：定义指标模型

**操作**：
- 在 `internal/model/metric.go` 中定义指标相关结构体
- 包含：指标定义（名称、查询语句、单位、分类、聚合方式）
- 包含：指标值（原始值、格式化值、状态）
- 支持 "N/A" 值表示待定项

**验证**：
- [ ] 支持不同类型的指标值（百分比、字节、秒数等）
- [ ] 包含格式化显示所需的所有字段
- [ ] 支持待定项的 N/A 显示
- [ ] 编译无错误

---

### 步骤 12：定义告警模型

**操作**：
- 在 `internal/model/alert.go` 中定义告警相关结构体
- 包含：告警级别枚举（正常、警告、严重）
- 包含：告警详情（主机、指标、当前值、阈值、消息）

**验证**：
- [ ] 告警级别定义完整
- [ ] 告警信息包含定位问题所需的所有字段
- [ ] 编译无错误

---

### 步骤 13：定义巡检结果模型

**操作**：
- 在 `internal/model/inspection.go` 中定义巡检结果结构体
- 包含：巡检摘要（时间、耗时、主机统计）
- 包含：主机结果列表
- 包含：告警汇总列表
- 时间字段使用 `Asia/Shanghai` 时区

**验证**：
- [ ] 结构体能够承载完整的巡检数据
- [ ] 支持 JSON 序列化
- [ ] 时间格式正确（中国时区）
- [ ] 编译无错误

---

## 阶段四：API 客户端实现（步骤 14-19）

### 步骤 14：定义 N9E 客户端接口和类型

**操作**：
- 在 `internal/client/n9e/types.go` 中定义请求和响应类型
- 根据夜莺 API 响应格式定义结构体（参考关键技术决策第 1 节）
- 定义 `ExtendInfo` 结构体解析嵌套 JSON

**类型定义**：
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
    CPU        CPUInfo        `json:"cpu"`
    Memory     MemoryInfo     `json:"memory"`
    Network    NetworkInfo    `json:"network"`
    Platform   PlatformInfo   `json:"platform"`
    Filesystem []FSInfo       `json:"filesystem"`
}

type PlatformInfo struct {
    Hostname      string `json:"hostname"`
    OS            string `json:"os"`
    KernelRelease string `json:"kernel_release"`
}
```

**验证**：
- [ ] 类型定义与夜莺 API 响应格式匹配
- [ ] `ExtendInfo` 能够正确解析嵌套 JSON 字符串
- [ ] 包含所有需要的字段（主机名、IP、OS、版本等）
- [ ] 编译无错误

---

### 步骤 15：实现 N9E 客户端

**操作**：
- 在 `internal/client/n9e/client.go` 中实现客户端
- 实现构造函数，接收配置参数
- 实现获取主机列表方法
- 实现 Token 认证（X-User-Token 请求头）
- 实现超时和错误处理
- 集成重试机制（指数退避）
- 实现 ident 清理逻辑（处理 `主机名@IP` 格式）

**ident 清理实现**：
```go
func cleanIdent(ident string) string {
    // 处理 "hostname@192.168.1.100" 格式
    if idx := strings.Index(ident, "@"); idx > 0 {
        return ident[:idx]
    }
    return ident
}
```

**验证**：
- [ ] 使用真实 API 地址测试，能够获取主机列表
- [ ] Token 认证正确添加到请求头
- [ ] 网络超时时返回明确错误
- [ ] API 返回错误时能正确解析错误信息
- [ ] ident 清理逻辑正确（去除 @IP 后缀）
- [ ] 重试机制工作正常

---

### 步骤 16：编写 N9E 客户端单元测试

**操作**：
- 在 `internal/client/n9e/client_test.go` 中编写测试
- 使用 httptest 模拟 API 响应
- 测试正常响应、认证失败、超时等场景
- 测试 ident 清理逻辑
- 测试重试机制

**测试用例**：
- [ ] 正常获取主机列表
- [ ] ExtendInfo JSON 解析
- [ ] ident 清理（普通格式、@IP 格式）
- [ ] Token 认证失败（401）
- [ ] 网络超时重试
- [ ] 5xx 错误重试
- [ ] 4xx 错误不重试

**验证**：
- [ ] 执行 `go test ./internal/client/n9e/` 全部通过
- [ ] 测试覆盖率达到 70% 以上
- [ ] 包含正向和异常场景测试

---

### 步骤 17：定义 VictoriaMetrics 客户端类型

**操作**：
- 在 `internal/client/vm/types.go` 中定义类型
- 定义 Prometheus API 响应格式（status、data、resultType）
- 定义即时查询结果结构（metric 标签、value）

**验证**：
- [ ] 类型定义符合 Prometheus HTTP API 规范
- [ ] 能够正确解析 vector 类型响应
- [ ] 编译无错误

---

### 步骤 18：实现 VictoriaMetrics 客户端

**操作**：
- 在 `internal/client/vm/client.go` 中实现客户端
- 实现构造函数，接收配置参数
- 实现即时查询方法（/api/v1/query）
- 支持 PromQL 查询语句
- 实现结果解析，提取指标值和标签
- 集成重试机制
- 支持主机筛选标签注入（业务组 OR + 标签 AND）

**主机筛选实现**：
```go
func (c *Client) buildQuery(baseQuery string, filter *HostFilter) string {
    if filter == nil {
        return baseQuery
    }

    var labelMatchers []string

    // 业务组 - OR 关系
    if len(filter.BusinessGroups) > 0 {
        groups := strings.Join(filter.BusinessGroups, "|")
        labelMatchers = append(labelMatchers, fmt.Sprintf(`busigroup=~"%s"`, groups))
    }

    // 标签 - AND 关系
    for k, v := range filter.Tags {
        labelMatchers = append(labelMatchers, fmt.Sprintf(`%s="%s"`, k, v))
    }

    // 注入到查询语句
    // ...
}
```

**验证**：
- [ ] 使用真实 API 测试 CPU 利用率查询
- [ ] 能够正确解析返回的指标值
- [ ] 能够提取 ident/host 标签用于主机识别
- [ ] 查询语句正确进行 URL 编码
- [ ] 主机筛选标签正确注入
- [ ] 重试机制工作正常

---

### 步骤 19：编写 VictoriaMetrics 客户端单元测试

**操作**：
- 在 `internal/client/vm/client_test.go` 中编写测试
- 模拟各种 PromQL 查询响应
- 测试空结果、多结果、错误响应等场景
- 测试主机筛选标签注入
- 测试重试机制

**验证**：
- [ ] 执行 `go test ./internal/client/vm/` 全部通过
- [ ] 测试覆盖率达到 70% 以上
- [ ] 包含边界条件测试

---

## 阶段五：核心业务逻辑（步骤 20-23）

### 步骤 20：实现数据采集服务

**操作**：
- 在 `internal/service/collector.go` 中实现采集服务
- 实现 Collector 接口
- 整合 N9E 客户端获取主机元信息（包括 CPU 核心数）
- 整合 VM 客户端获取指标数据
- 将原始数据转换为内部模型
- 实现磁盘数据按挂载点展开
- 处理待定项（返回 N/A）

**磁盘数据处理**：
```go
func (c *Collector) processDiskMetrics(results []QueryResult) []DiskMountInfo {
    // 按 path 标签分组
    // 过滤 tmpfs、overlay 等非物理磁盘
    // 返回各挂载点数据列表
}
```

**验证**：
- [ ] 能够获取完整的主机元信息（包括 CPU 核心数）
- [ ] 能够获取所有配置的指标数据
- [ ] 数据正确映射到主机（通过清理后的 ident/hostname 匹配）
- [ ] 磁盘数据按挂载点展开
- [ ] 待定项显示 N/A
- [ ] 编译无错误

---

### 步骤 21：实现阈值评估服务

**操作**：
- 在 `internal/service/evaluator.go` 中实现评估服务
- 实现 Evaluator 接口
- 根据配置的阈值判断指标状态
- 生成告警信息（包含主机、指标、当前值、阈值）
- 确定主机整体状态（取最严重的告警级别）
- 磁盘告警使用各挂载点最大值判断

**验证**：
- [ ] CPU 75% 被判定为警告级别
- [ ] CPU 95% 被判定为严重级别
- [ ] 多个告警时主机状态取最严重级别
- [ ] 无异常时状态为正常
- [ ] 磁盘告警基于最大使用率判断

---

### 步骤 22：实现巡检编排服务

**操作**：
- 在 `internal/service/inspector.go` 中实现核心编排逻辑
- 协调配置加载、数据采集、阈值评估流程
- 实现并发采集（使用 errgroup 控制并发数，默认 20）
- 聚合所有主机结果，生成巡检摘要
- 单主机失败不影响整体流程
- 所有时间使用 `Asia/Shanghai` 时区

**验证**：
- [ ] 能够完成端到端的巡检流程
- [ ] 并发数受配置控制
- [ ] 单主机超时不阻塞其他主机
- [ ] 巡检摘要统计正确
- [ ] 时间显示为中国时区

---

### 步骤 23：编写业务逻辑单元测试

**操作**：
- 为 collector、evaluator、inspector 编写单元测试
- 使用 mock 对象隔离外部依赖
- 测试各种边界条件和异常场景

**验证**：
- [ ] 执行 `go test ./internal/service/` 全部通过
- [ ] 测试覆盖率达到 80% 以上
- [ ] 包含并发场景测试

---

## 阶段六：报告生成模块（步骤 24-32）

### 步骤 24：定义报告写入器接口

**操作**：
- 在 `internal/report/writer.go` 中定义 ReportWriter 接口
- 接口方法：Write(result, outputPath) error
- 接口方法：Format() string

**验证**：
- [ ] 接口定义清晰，职责单一
- [ ] 编译无错误

---

### 步骤 25：实现 Excel 报告生成器 - 基础结构

**操作**：
- 在 `internal/report/excel/writer.go` 中实现 ExcelWriter
- 实现 ReportWriter 接口
- 创建包含三个工作表的 Excel 文件：巡检概览、详细数据、异常汇总

**验证**：
- [ ] 生成的 Excel 文件可以正常打开
- [ ] 包含三个工作表且命名正确
- [ ] 文件保存到指定路径

---

### 步骤 26：实现 Excel 报告 - 巡检概览表

**操作**：
- 填充巡检概览工作表
- 包含：巡检时间（中国时区）、主机总数、正常/警告/严重/失败主机数
- 设置基本样式（标题加粗、列宽自适应）

**验证**：
- [ ] 概览数据与巡检结果一致
- [ ] 时间格式正确显示（中国时区）
- [ ] 样式美观可读

---

### 步骤 27：实现 Excel 报告 - 详细数据表

**操作**：
- 填充详细数据工作表
- 表头：主机名、IP、系统类型、系统版本、CPU 核心数、CPU%、内存%、各磁盘挂载点%、运行时间等
- 每行一台主机的完整数据
- 设置列宽和对齐方式
- **实现条件格式**：
  - 警告值：黄色背景 `#FFEB9C`
  - 严重值：红色背景 `#FFC7CE`
- 待定项显示 "N/A"

**验证**：
- [ ] 所有主机数据完整显示
- [ ] 数值格式正确（百分比、字节转换等）
- [ ] 条件格式正确应用（黄色警告、红色严重）
- [ ] 表头固定，便于滚动查看
- [ ] 磁盘各挂载点分列显示

---

### 步骤 28：实现 Excel 报告 - 异常汇总表

**操作**：
- 填充异常汇总工作表
- 表头：主机名、异常指标、当前值、阈值、告警级别
- 仅包含有告警的记录
- 按告警级别排序（严重优先）
- 应用条件格式（与详细数据表一致）

**验证**：
- [ ] 仅显示异常数据
- [ ] 告警级别正确标识
- [ ] 排序逻辑正确
- [ ] 条件格式正确应用

---

### 步骤 29：实现 HTML 报告生成器 - 基础结构

**操作**：
- 在 `internal/report/html/writer.go` 中实现 HTMLWriter
- 实现 ReportWriter 接口
- 使用 Go 标准库 html/template 渲染
- **支持外置模板**：
  - 优先加载配置中指定的用户自定义模板
  - 模板不存在时使用内置默认模板
- 创建默认 HTML 模板文件

**模板加载逻辑**：
```go
func (w *HTMLWriter) loadTemplate() (*template.Template, error) {
    // 1. 尝试加载用户自定义模板
    if w.config.HTMLTemplatePath != "" {
        if _, err := os.Stat(w.config.HTMLTemplatePath); err == nil {
            return template.ParseFiles(w.config.HTMLTemplatePath)
        }
    }
    // 2. 使用内置默认模板
    return template.ParseFS(embeddedTemplates, "templates/default.html")
}
```

**验证**：
- [ ] 生成的 HTML 文件可在浏览器正常打开
- [ ] 页面结构完整（头部、主体、尾部）
- [ ] 无 JavaScript 错误
- [ ] 用户自定义模板能够正确加载

---

### 步骤 30：实现 HTML 报告 - 摘要区域

**操作**：
- 在 HTML 模板中添加摘要卡片区域
- 显示巡检时间（中国时区）、主机统计数据
- 使用不同颜色区分正常/警告/严重状态

**颜色规范**：
- 正常：绿色 `#28a745`
- 警告：黄色 `#ffc107`
- 严重：红色 `#dc3545`

**验证**：
- [ ] 摘要数据正确显示
- [ ] 颜色编码直观（绿色正常、黄色警告、红色严重）
- [ ] 响应式布局，移动端可正常查看

---

### 步骤 31：实现 HTML 报告 - 主机详情表格

**操作**：
- 在 HTML 模板中添加主机详情表格
- 包含所有巡检指标列（包括各磁盘挂载点）
- 异常值使用背景色高亮
- **实现客户端排序功能**（纯 JavaScript）：
  - 可排序列：主机名、CPU%、内存%、磁盘%、状态
  - 默认排序：按状态严重程度降序
- 待定项显示 "N/A"

**排序实现**：
```javascript
// 简单的表格排序函数
function sortTable(columnIndex, type) {
    // type: 'string' | 'number' | 'status'
    // 状态排序权重：critical=3, warning=2, normal=1
}
```

**验证**：
- [ ] 表格数据完整正确
- [ ] 异常值高亮显示
- [ ] 表格在大量数据时仍可正常渲染
- [ ] 点击表头可排序
- [ ] 默认按状态排序

---

### 步骤 32：实现报告格式注册表

**操作**：
- 在 `internal/report/registry.go` 中实现格式注册表
- 支持按格式名称获取对应的 Writer
- 预注册 Excel 和 HTML 两种格式

**验证**：
- [ ] 能够通过 "excel" 获取 ExcelWriter
- [ ] 能够通过 "html" 获取 HTMLWriter
- [ ] 请求未知格式时返回明确错误

---

## 阶段七：CLI 命令实现（步骤 33-37）

### 步骤 33：实现根命令

**操作**：
- 使用 cobra 创建根命令
- 设置应用名称、描述、版本信息
- 添加全局标志：配置文件路径、日志级别

**验证**：
- [ ] 执行 `./bin/inspect` 显示帮助信息
- [ ] 执行 `./bin/inspect --help` 显示详细帮助
- [ ] 全局标志正确解析

---

### 步骤 34：实现 version 子命令

**操作**：
- 添加 version 子命令
- 显示版本号、构建时间、Go 版本

**验证**：
- [ ] 执行 `./bin/inspect version` 显示版本信息
- [ ] 版本号通过构建参数注入

---

### 步骤 35：实现 validate 子命令

**操作**：
- 添加 validate 子命令
- 加载并验证配置文件
- 输出验证结果（成功或具体错误）

**验证**：
- [ ] 有效配置文件显示验证成功
- [ ] 无效配置文件显示具体错误原因
- [ ] 配置文件不存在时提示明确错误

---

### 步骤 36：实现 run 子命令

**操作**：
- 添加 run 子命令
- 添加标志：--format（输出格式）、--output（输出目录）
- 整合完整的巡检流程：加载配置 → 数据采集 → 阈值评估 → 生成报告
- 输出执行进度和结果摘要

**验证**：
- [ ] 执行 `./bin/inspect run -c config.yaml` 完成巡检
- [ ] 在指定目录生成报告文件
- [ ] 控制台输出执行进度和摘要
- [ ] 执行时间在合理范围内

---

### 步骤 37：实现日志输出

**操作**：
- 集成 zerolog 日志库
- 根据配置设置日志级别和格式
- 在关键节点输出日志（开始、完成、错误）
- 日志时间使用 `Asia/Shanghai` 时区

**验证**：
- [ ] 日志级别可通过配置控制
- [ ] JSON 格式日志结构正确
- [ ] 错误日志包含足够的上下文信息
- [ ] 日志时间为中国时区

---

## 阶段八：构建与测试（步骤 38-41）

### 步骤 38：创建 Makefile

**操作**：
- 创建 Makefile 包含以下目标：
  - build：构建本地二进制
  - build-all：交叉编译多平台
  - test：运行测试
  - lint：代码检查
  - clean：清理构建产物
  - coverage：生成测试覆盖率报告

**验证**：
- [ ] `make build` 成功生成二进制文件
- [ ] `make test` 运行所有测试
- [ ] `make clean` 清理 bin 目录
- [ ] `make coverage` 生成覆盖率报告

---

### 步骤 39：端到端测试

**操作**：
- 使用真实 API 进行完整巡检测试
- 验证生成的 Excel 报告内容正确性
- 验证生成的 HTML 报告内容正确性
- 测试不同配置组合
- 验证磁盘各挂载点显示正确

**验证**：
- [ ] 完整巡检流程无错误
- [ ] Excel 报告数据与 API 返回一致
- [ ] Excel 条件格式正确应用
- [ ] HTML 报告在浏览器中正确显示
- [ ] HTML 排序功能正常
- [ ] 磁盘各挂载点数据正确
- [ ] 100+ 主机场景下性能可接受（< 5 分钟）

---

### 步骤 40：错误处理验证

**操作**：
- 测试各种异常场景：
  - API 不可达
  - Token 无效
  - 配置文件错误
  - 输出目录无写权限
  - 重试耗尽
- 确保所有错误都有清晰的提示

**验证**：
- [ ] 每种错误场景都有明确的错误信息
- [ ] 程序不会因异常而崩溃
- [ ] 错误退出码非零
- [ ] 重试日志记录正确

---

### 步骤 41：创建 README 文档

**操作**：
- 编写 README.md 包含：
  - 项目简介
  - 快速开始指南
  - 配置说明（包括重试策略、时区配置）
  - 命令行用法
  - HTML 模板自定义说明
  - 常见问题

**验证**：
- [ ] 新用户可按照 README 成功运行工具
- [ ] 配置项说明完整
- [ ] 示例命令可直接复制使用
- [ ] 模板自定义说明清晰

---

## 检查点汇总

### MVP 完成标准

- [ ] 能够通过命令行执行巡检
- [ ] 能够从夜莺获取主机元信息（包括 CPU 核心数）
- [ ] 能够从 VictoriaMetrics 获取指标数据
- [ ] 能够生成包含三个工作表的 Excel 报告（含条件格式）
- [ ] 能够生成响应式 HTML 报告（含排序功能）
- [ ] 支持配置文件自定义数据源和阈值
- [ ] 支持并发采集 100+ 主机
- [ ] 支持主机筛选（业务组 OR + 标签 AND）
- [ ] 磁盘各挂载点分别显示
- [ ] 待定项显示 N/A 并预留接口
- [ ] 所有时间使用中国时区
- [ ] HTTP 请求支持重试机制
- [ ] 单元测试覆盖率达标（整体 ≥ 70%）
- [ ] 文档完整可用

---

## 附录：步骤与阶段对照表

| 阶段 | 步骤范围 | 内容 |
|------|----------|------|
| 阶段一 | 步骤 1-4 | 项目初始化与基础架构 |
| 阶段二 | 步骤 5-9 | 配置管理模块 |
| 阶段三 | 步骤 10-13 | 数据模型定义 |
| 阶段四 | 步骤 14-19 | API 客户端实现 |
| 阶段五 | 步骤 20-23 | 核心业务逻辑 |
| 阶段六 | 步骤 24-32 | 报告生成模块 |
| 阶段七 | 步骤 33-37 | CLI 命令实现 |
| 阶段八 | 步骤 38-41 | 构建与测试 |

---

## 依赖关系图

```
步骤 1-4（项目初始化）
    ↓
步骤 5-9（配置管理）
    ↓
步骤 10-13（数据模型）
    ↓
步骤 14-19（API 客户端）← 依赖步骤 5-13
    ↓
步骤 20-23（业务逻辑）← 依赖步骤 10-19
    ↓
步骤 24-32（报告生成）← 依赖步骤 10-13
    ↓
步骤 33-37（CLI 命令）← 依赖步骤 5-9, 20-32
    ↓
步骤 38-41（构建测试）← 依赖所有步骤
```

---

## 版本记录

| 版本 | 日期 | 说明 |
|------|------|------|
| v1.0 | 2025-12-13 | 初始版本，聚焦 MVP 核心功能 |
| v1.1 | 2025-12-13 | 补充关键技术决策：N9E API 结构、ident 清理、磁盘展开、筛选逻辑、时区、重试策略等 |
