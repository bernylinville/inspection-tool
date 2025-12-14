# 系统巡检工具 (Inspection Tool)

基于监控数据的无侵入式系统巡检工具。通过调用夜莺（N9E）和 VictoriaMetrics API 查询监控数据，生成 Excel 和 HTML 格式的巡检报告。

## 特点

- **无侵入式**：无需登录服务器，通过 API 获取监控数据
- **高性能**：支持 100+ 主机并行采集，errgroup 控制并发
- **专业报告**：生成 Excel（3 工作表 + 条件格式）和 HTML（响应式 + 排序）报告
- **灵活配置**：支持自定义阈值、主机筛选、报告格式
- **易于部署**：单二进制文件，零依赖，支持 Linux/macOS/Windows

## 系统架构

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│    Categraf     │────▶│   夜莺 (N9E)    │────▶│ VictoriaMetrics │
│   (数据采集)     │     │   (监控平台)     │     │   (时序数据库)   │
└─────────────────┘     └─────────────────┘     └─────────────────┘
                               │                        │
                               │ 元信息查询              │ 指标数据查询
                               ▼                        ▼
                        ┌─────────────────────────────────────┐
                        │         系统巡检工具                  │
                        │  ┌─────────┐  ┌─────────┐           │
                        │  │ 数据采集 │  │ 报告生成 │           │
                        │  └─────────┘  └─────────┘           │
                        └─────────────────────────────────────┘
                                        │
                        ┌───────────────┴───────────────┐
                        ▼                               ▼
                 ┌─────────────┐                ┌─────────────┐
                 │  Excel 报告  │                │  HTML 报告   │
                 └─────────────┘                └─────────────┘
```

## 快速开始

```bash
# 1. 下载或构建
make build

# 2. 创建配置文件
cp configs/config.example.yaml config.yaml

# 3. 设置 N9E Token（敏感信息建议使用环境变量）
export N9E_TOKEN="your-token-here"

# 4. 运行巡检
./bin/inspect run -c config.yaml
```

## 安装

### 从源码构建

**前置要求**：Go 1.25+

```bash
# 克隆项目
git clone https://github.com/your-org/inspection-tool.git
cd inspection-tool

# 构建本地二进制
make build

# 构建产物位于 bin/inspect
./bin/inspect version
```

### 交叉编译

```bash
# 构建多平台二进制
make build-all

# 产物：
# - bin/inspect-linux-amd64
# - bin/inspect-darwin-amd64
# - bin/inspect-darwin-arm64
# - bin/inspect-windows-amd64.exe
```

## 命令行用法

```bash
# 运行巡检（使用默认配置）
./bin/inspect run -c config.yaml

# 指定输出格式和目录
./bin/inspect run -c config.yaml -f excel,html -o ./reports

# 使用自定义指标定义文件
./bin/inspect run -c config.yaml -m custom_metrics.yaml

# 验证配置文件
./bin/inspect validate -c config.yaml

# 查看版本信息
./bin/inspect version

# 查看帮助
./bin/inspect --help
./bin/inspect run --help
```

### 命令行参数

| 参数 | 短选项 | 说明 | 默认值 |
|------|--------|------|--------|
| `--config` | `-c` | 配置文件路径 | `config.yaml` |
| `--format` | `-f` | 输出格式（excel,html） | 从配置文件读取 |
| `--output` | `-o` | 输出目录 | 从配置文件读取 |
| `--metrics` | `-m` | 指标定义文件 | `configs/metrics.yaml` |
| `--log-level` | - | 日志级别 | `info` |

### 退出码

| 退出码 | 含义 |
|--------|------|
| 0 | 巡检成功，无告警 |
| 1 | 巡检完成，有警告级别告警 |
| 2 | 巡检完成，有严重级别告警 |

## 配置说明

配置文件使用 YAML 格式，完整示例见 `configs/config.example.yaml`。

### 数据源配置

```yaml
datasources:
  # 夜莺 API（获取主机元信息）
  n9e:
    endpoint: "http://n9e.example.com:17000"
    token: "${N9E_TOKEN}"  # 支持环境变量
    timeout: 30s
    # 主机过滤（可选）：只获取指定标签的主机
    # query: "items=短剧项目"

  # VictoriaMetrics（获取指标数据）
  victoriametrics:
    endpoint: "http://vm.example.com:8428"
    timeout: 30s
```

### 巡检配置

```yaml
inspection:
  # 并发数（同时采集的主机数量）
  concurrency: 20
  # 单主机超时时间
  host_timeout: 10s
  # 主机筛选（可选）
  host_filter:
    business_groups:  # OR 关系
      - "生产环境"
      - "测试环境"
    tags:             # AND 关系
      env: "prod"
```

### 阈值配置

```yaml
thresholds:
  cpu_usage:
    warning: 70     # CPU > 70% 警告
    critical: 90    # CPU > 90% 严重
  memory_usage:
    warning: 70
    critical: 90
  disk_usage:       # 多磁盘取最大值判断
    warning: 70
    critical: 90
  zombie_processes:
    warning: 1      # > 0 警告
    critical: 10
  load_per_core:    # 负载/核心数
    warning: 0.7
    critical: 1.0
```

### 报告配置

```yaml
report:
  output_dir: "./reports"
  formats:
    - excel
    - html
  filename_template: "inspection_report_{{.Date}}"
  # 自定义 HTML 模板（可选）
  # html_template: "./templates/html/custom.tmpl"
  timezone: "Asia/Shanghai"
```

### 日志配置

```yaml
logging:
  level: info      # debug, info, warn, error
  format: json     # json, console
```

### 环境变量覆盖

所有配置项都支持环境变量覆盖，格式为 `INSPECT_<节>_<键>`：

```bash
# 覆盖 N9E Token
export INSPECT_DATASOURCES_N9E_TOKEN="your-token"

# 覆盖日志级别
export INSPECT_LOGGING_LEVEL="debug"
```

## 报告说明

### Excel 报告

生成包含 3 个工作表的 Excel 文件：

| 工作表 | 内容 |
|--------|------|
| 巡检概览 | 巡检时间、耗时、主机统计、告警统计、工具版本 |
| 详细数据 | 所有主机的完整指标数据，磁盘按挂载点分列 |
| 异常汇总 | 告警列表，按严重程度排序 |

**条件格式**：
- 警告级别：黄色背景 (`#FFEB9C`)
- 严重级别：红色背景 (`#FFC7CE`)
- 正常状态：绿色背景 (`#C6EFCE`)

### HTML 报告

响应式单页报告，特性包括：

- **摘要卡片**：主机统计、告警统计，颜色编码
- **主机详情表**：完整指标数据，支持点击表头排序
- **异常汇总表**：按严重程度排序
- **条件样式**：与 Excel 一致的颜色方案
- **打印优化**：专用打印样式
- **移动端适配**：响应式布局

**默认排序**：按状态严重程度降序（严重 > 警告 > 失败 > 正常）

## 巡检指标

### 已实现指标

| 分类 | 指标 | 说明 |
|------|------|------|
| CPU | cpu_usage | CPU 利用率 (%) |
| 内存 | memory_usage | 内存利用率 (%) |
| 内存 | memory_total | 内存总量 |
| 内存 | memory_available | 可分配内存 |
| 磁盘 | disk_usage | 磁盘利用率（按挂载点展开） |
| 磁盘 | disk_total | 磁盘总量（按挂载点展开） |
| 磁盘 | disk_free | 磁盘剩余（按挂载点展开） |
| 系统 | uptime | 运行时间 |
| 系统 | cpu_cores | CPU 核心数 |
| 系统 | load_1m/5m/15m | 系统负载 |
| 系统 | load_per_core | 单核负载 |
| 进程 | processes_total | 总进程数 |
| 进程 | processes_zombies | 僵尸进程数 |

### 待实现指标

以下指标在报告中显示 "N/A"：

- NTP 检查
- 公网访问检查
- 密码过期天数
- 密码策略
- 打开文件句柄数
- 系统参数检查

## HTML 模板自定义

支持用户自定义 HTML 报告模板：

1. 复制内置模板作为基础：
   ```bash
   # 内置模板位置（嵌入二进制）
   # 可参考 internal/report/html/templates/default.html
   ```

2. 修改模板文件

3. 配置模板路径：
   ```yaml
   report:
     html_template: "./templates/html/custom.tmpl"
   ```

**模板数据结构**：
- `{{.Title}}` - 报告标题
- `{{.InspectionTime}}` - 巡检时间
- `{{.Duration}}` - 巡检耗时
- `{{.Summary}}` - 主机摘要统计
- `{{.AlertSummary}}` - 告警摘要
- `{{.Hosts}}` - 主机列表
- `{{.Alerts}}` - 告警列表
- `{{.DiskPaths}}` - 磁盘挂载点列表
- `{{.Version}}` - 工具版本

## 定时任务

### Crontab 示例

```bash
# 每天早上 8 点执行巡检
0 8 * * * /opt/inspect/inspect run -c /etc/inspect/config.yaml >> /var/log/inspect.log 2>&1

# 每周一生成周报
0 9 * * 1 /opt/inspect/inspect run -c /etc/inspect/config.yaml -o /data/reports/weekly
```

### Systemd Timer

```ini
# /etc/systemd/system/inspect.service
[Unit]
Description=System Inspection Tool

[Service]
Type=oneshot
ExecStart=/opt/inspect/inspect run -c /etc/inspect/config.yaml
User=inspect

# /etc/systemd/system/inspect.timer
[Unit]
Description=Run inspection daily

[Timer]
OnCalendar=*-*-* 08:00:00
Persistent=true

[Install]
WantedBy=timers.target
```

## 常见问题

### Q: 如何只巡检特定主机？

有两种方式：

1. **N9E 级别过滤**（推荐）：在配置中设置 `query` 参数
   ```yaml
   datasources:
     n9e:
       query: "items=短剧项目"  # N9E 标签查询语法
   ```

2. **PromQL 级别过滤**：使用 `host_filter` 配置
   ```yaml
   inspection:
     host_filter:
       business_groups:
         - "生产环境"
       tags:
         env: "prod"
   ```

### Q: 磁盘显示了容器相关路径怎么办？

工具会自动过滤非物理磁盘，包括：
- NFS/CIFS 网络文件系统
- Kubernetes CSI/Longhorn 挂载
- overlay/overlay2 容器存储
- tmpfs/devtmpfs 临时文件系统

只保留真正的块设备（sd*, vd*, nvme*, dm-* 等）。

### Q: 报告时间显示不正确？

检查配置文件中的时区设置：
```yaml
report:
  timezone: "Asia/Shanghai"
```

### Q: 如何调试连接问题？

1. 设置日志级别为 debug：
   ```yaml
   logging:
     level: debug
     format: console
   ```

2. 或使用环境变量：
   ```bash
   INSPECT_LOGGING_LEVEL=debug ./bin/inspect run -c config.yaml
   ```

### Q: 单个主机采集失败会影响整体吗？

不会。工具使用 errgroup 并发控制，单个主机失败只会在报告中标记为"失败"状态，不影响其他主机的采集。

## 开发相关

### 构建和测试

```bash
# 构建
make build

# 运行测试（带竞态检测）
make test

# 代码检查（需要 golangci-lint）
make lint

# 生成覆盖率报告
make coverage

# 清理构建产物
make clean
```

### 项目结构

```
inspection-tool/
├── cmd/inspect/              # 程序入口
│   ├── main.go
│   └── cmd/                  # Cobra 命令
├── internal/
│   ├── config/               # 配置管理
│   ├── client/
│   │   ├── n9e/              # N9E API 客户端
│   │   └── vm/               # VictoriaMetrics 客户端
│   ├── model/                # 数据模型
│   ├── service/              # 业务逻辑
│   └── report/
│       ├── excel/            # Excel 报告生成
│       └── html/             # HTML 报告生成
├── configs/                  # 配置文件示例
│   ├── config.example.yaml
│   └── metrics.yaml
└── templates/html/           # 用户自定义模板目录
```

### 测试覆盖率

| 模块 | 覆盖率 |
|------|--------|
| N9E 客户端 | 91.6% |
| VM 客户端 | 94.0% |
| Config | 89.8% |
| Service | 80.1% |
| Excel 报告 | 89.6% |
| HTML 报告 | 90.4% |
| **总计** | **71.8%** |

## 许可证

[待定]

## 版本记录

| 版本 | 日期 | 说明 |
|------|------|------|
| v0.1.0 | 2025-12-14 | 初始版本，完成 MVP 功能 |
