# Inspection-Tool 改进任务清单

> 基于项目质量评估报告生成 | 评分: 72/100
> 
> 生成时间: 2026-01-09

---

## 优先级说明

| 优先级 | 含义 | 时间要求 |
|--------|------|----------|
| **P0** | 阻塞发布，必须立即修复 | 1-2 天 |
| **P1** | 重要改进，影响生产质量 | 1 周内 |
| **P2** | 锦上添花，提升工程成熟度 | 2 周内 |
| **P3** | 长期优化，技术债务清理 | 按计划 |

---

## P0 - 阻塞发布 (必须修复)

### 1. 修复 Excel 报告测试失败

**问题描述**: 
```
--- FAIL: TestWriter_MySQLSheet_Headers (0.00s)
--- FAIL: TestWriter_MySQLSheet_DataMapping (0.00s)
--- FAIL: TestWriter_MySQLSheet_ConditionalFormat (0.00s)
```

**影响**: Excel 报告是核心交付物，测试失败意味着无法保证报告正确性

**执行步骤**:
- [ ] 1.1 运行 `go test -v ./internal/report/excel/...` 查看详细错误信息
- [ ] 1.2 定位 `internal/report/excel/writer_test.go` 中失败的断言
- [ ] 1.3 检查 MySQL 工作表的表头定义是否与测试期望一致
- [ ] 1.4 检查数据映射逻辑是否正确处理所有字段
- [ ] 1.5 检查条件格式的单元格范围计算是否正确
- [ ] 1.6 修复代码或更新测试用例
- [ ] 1.7 运行完整测试套件确认修复: `make test`

**验收标准**: `go test ./internal/report/excel/...` 全部通过

---

### 2. 添加 VictoriaMetrics 认证支持

**问题描述**: 当前 VM 客户端不支持任何认证方式，无法连接需要认证的 VM 实例

**文件位置**: 
- `internal/client/vm/client.go`
- `internal/config/config.go`

**执行步骤**:
- [ ] 2.1 在 `VictoriaMetricsConfig` 结构体中添加认证字段:
  ```go
  type VictoriaMetricsConfig struct {
      Endpoint  string
      Timeout   time.Duration
      // 新增字段
      Username  string `mapstructure:"username"`
      Password  string `mapstructure:"password"`
      Token     string `mapstructure:"token"`      // Bearer Token
      TLSConfig *TLSConfig `mapstructure:"tls"`   // 可选 TLS
  }
  ```
- [ ] 2.2 更新 `internal/client/vm/client.go` 的 `NewClient` 函数，根据配置设置认证:
  - 如果有 Token: 设置 `Authorization: Bearer <token>` 头
  - 如果有 Username/Password: 设置 Basic Auth
- [ ] 2.3 更新 `configs/config.example.yaml` 添加认证配置示例
- [ ] 2.4 添加环境变量支持: `INSPECT_DATASOURCES_VICTORIAMETRICS_TOKEN`
- [ ] 2.5 编写单元测试验证认证头正确设置
- [ ] 2.6 更新 README.md 文档说明认证配置方式

**验收标准**: 
- 可通过配置文件或环境变量配置 VM 认证
- 单元测试覆盖 Basic Auth 和 Bearer Token 两种方式

---

## P1 - 重要改进 (1周内完成)

### 3. 重构 run.go 代码膨胀问题

**问题描述**: `cmd/inspect/cmd/run.go` 有 1057 行，包含过多重复的服务初始化代码

**执行步骤**:
- [ ] 3.1 创建 `internal/service/orchestrator.go` 文件
- [ ] 3.2 定义 `InspectionOrchestrator` 结构体，负责协调所有巡检服务
- [ ] 3.3 创建服务注册机制:
  ```go
  type ServiceInspector interface {
      Name() string
      Enabled() bool
      Run(ctx context.Context) (*ServiceResult, error)
  }
  
  func (o *Orchestrator) Register(inspector ServiceInspector)
  func (o *Orchestrator) RunAll(ctx context.Context) (*CombinedResult, error)
  ```
- [ ] 3.4 将 Host/MySQL/Redis/Nginx/Tomcat/ES 的初始化逻辑迁移到各自的 Inspector
- [ ] 3.5 重构 `run.go` 只保留:
  - CLI 参数解析
  - 配置加载
  - Orchestrator 初始化和调用
  - 报告生成调用
- [ ] 3.6 确保 `run.go` 行数降至 300 行以内
- [ ] 3.7 运行完整测试确保功能不变

**验收标准**: 
- `run.go` 行数 < 300
- 新增服务类型只需实现接口并注册

---

### 4. 补充 ElasticSearch 服务测试

**问题描述**: ElasticSearch 服务实现完全没有测试覆盖 (0%)

**文件位置**:
- `internal/service/elasticsearch_collector.go`
- `internal/service/elasticsearch_evaluator.go`
- `internal/service/elasticsearch_inspector.go`

**执行步骤**:
- [ ] 4.1 创建 `internal/service/elasticsearch_collector_test.go`
  - 测试实例发现逻辑
  - 测试指标采集逻辑
  - Mock VM 客户端返回
- [ ] 4.2 创建 `internal/service/elasticsearch_evaluator_test.go`
  - 测试阈值评估逻辑
  - 使用表驱动测试覆盖各种告警场景
- [ ] 4.3 创建 `internal/service/elasticsearch_inspector_test.go`
  - 测试完整巡检流程
  - Mock Collector 和 Evaluator
- [ ] 4.4 确保覆盖率达到 80% 以上

**验收标准**: `go test -cover ./internal/service/... | grep elasticsearch` 显示 >= 80%

---

### 5. 补充 Tomcat 服务测试

**问题描述**: Tomcat 服务只有模型测试，采集和评估逻辑未覆盖 (~20%)

**执行步骤**:
- [ ] 5.1 创建 `internal/service/tomcat_collector_test.go`
  - 参考 `mysql_collector_test.go` 的模式
  - 测试实例发现和指标采集
- [ ] 5.2 创建 `internal/service/tomcat_evaluator_test.go`
  - 使用表驱动测试
  - 覆盖线程池、会话、响应时间等告警场景
- [ ] 5.3 确保覆盖率达到 80% 以上

**验收标准**: Tomcat 相关服务覆盖率 >= 80%

---

### 6. 修复被忽略的错误

**问题描述**: 多处使用 `_ = func()` 模式忽略错误返回值

**文件位置**:
- `internal/service/mysql_inspector.go`
- `internal/service/redis_inspector.go`
- `internal/service/tomcat_inspector.go`
- `internal/report/html/writer.go`

**执行步骤**:
- [ ] 6.1 搜索所有忽略错误的位置:
  ```bash
  grep -rn "_ = " internal/ --include="*.go" | grep -v "_test.go"
  ```
- [ ] 6.2 对于 `_ = evaluator.EvaluateAll(...)` 模式:
  - 改为记录错误日志: `if err := ...; err != nil { log.Warn().Err(err).Msg(...) }`
  - 或聚合错误到结果中
- [ ] 6.3 对于 `timezone, _ = time.LoadLocation(...)` 模式:
  - 创建 `internal/util/time.go` 工具函数
  - 加载失败时使用 UTC 并记录警告
  ```go
  func LoadTimezone(name string, logger zerolog.Logger) *time.Location {
      loc, err := time.LoadLocation(name)
      if err != nil {
          logger.Warn().Str("timezone", name).Msg("failed to load timezone, using UTC")
          return time.UTC
      }
      return loc
  }
  ```
- [ ] 6.4 运行 `golangci-lint run` 确认无 errcheck 警告

**验收标准**: `grep -rn "_ = " internal/ | grep -v "_test.go"` 结果为空或都有合理注释

---

## P2 - 锦上添花 (2周内完成)

### 7. 添加熔断器支持

**问题描述**: API 客户端缺乏熔断机制，服务降级时会持续重试

**执行步骤**:
- [ ] 7.1 添加依赖: `go get github.com/sony/gobreaker`
- [ ] 7.2 在 `internal/client/` 下创建 `circuitbreaker.go`:
  ```go
  type CircuitBreakerConfig struct {
      Name          string
      MaxRequests   uint32        // 半开状态最大请求数
      Interval      time.Duration // 统计周期
      Timeout       time.Duration // 开启状态超时
      FailureRatio  float64       // 失败率阈值
  }
  ```
- [ ] 7.3 在 N9E 客户端中集成熔断器
- [ ] 7.4 在 VM 客户端中集成熔断器
- [ ] 7.5 添加配置项支持开关熔断器
- [ ] 7.6 编写单元测试验证熔断逻辑

**验收标准**: 连续 5 次失败后自动熔断，30 秒后进入半开状态

---

### 8. 实现 N9E 分页查询

**问题描述**: 当前使用 `limit=10000` 硬编码，超过限制的主机无法获取

**文件位置**: `internal/client/n9e/client.go`

**执行步骤**:
- [ ] 8.1 修改 `GetTargets` 方法支持分页:
  ```go
  func (c *Client) GetTargets(ctx context.Context) ([]*Target, error) {
      var allTargets []*Target
      page := 1
      limit := 1000
      for {
          targets, total, err := c.getTargetsPage(ctx, page, limit)
          if err != nil { return nil, err }
          allTargets = append(allTargets, targets...)
          if len(allTargets) >= total { break }
          page++
      }
      return allTargets, nil
  }
  ```
- [ ] 8.2 添加配置项 `n9e.page_size` 控制每页大小
- [ ] 8.3 添加单元测试模拟多页返回
- [ ] 8.4 添加日志记录分页进度

**验收标准**: 可正确获取超过 10000 台主机的环境

---

### 9. 添加 CLI 集成测试

**问题描述**: CLI 层覆盖率为 0%，集成问题无法提前发现

**执行步骤**:
- [ ] 9.1 创建 `cmd/inspect/cmd/run_test.go`
- [ ] 9.2 编写表驱动集成测试:
  ```go
  func TestRunCommand(t *testing.T) {
      tests := []struct {
          name     string
          args     []string
          wantErr  bool
          wantCode int
      }{
          {"valid config", []string{"-c", "testdata/valid.yaml"}, false, 0},
          {"missing config", []string{"-c", "nonexistent.yaml"}, true, 1},
          {"invalid format", []string{"-c", "valid.yaml", "-f", "pdf"}, true, 1},
      }
      // ...
  }
  ```
- [ ] 9.3 创建 `cmd/inspect/cmd/testdata/` 目录放置测试配置文件
- [ ] 9.4 Mock 外部依赖 (N9E/VM 服务)
- [ ] 9.5 确保 CLI 覆盖率达到 50% 以上

**验收标准**: `go test ./cmd/...` 通过且覆盖率 >= 50%

---

### 10. 修复文档覆盖率不一致问题

**问题描述**: README 声称 85.2% 覆盖率，实际加权平均约 58%

**执行步骤**:
- [ ] 10.1 运行 `go test -cover ./...` 获取真实覆盖率数据
- [ ] 10.2 更新 README.md 中的覆盖率表格
- [ ] 10.3 添加 CI 自动生成覆盖率徽章
- [ ] 10.4 在 Makefile 中添加覆盖率检查目标:
  ```makefile
  coverage-check:
      @go test -coverprofile=coverage.out ./...
      @go tool cover -func=coverage.out | grep total
  ```

**验收标准**: README 中的覆盖率数据与 `make coverage-check` 输出一致

---

## P3 - 长期优化

### 11. 添加 OpenTelemetry 支持

**问题描述**: 工具自身缺乏可观测性，难以追踪性能问题

**执行步骤**:
- [ ] 11.1 添加 OTEL SDK 依赖
- [ ] 11.2 在巡检关键路径添加 Span:
  - 主机元信息采集
  - 指标查询
  - 报告生成
- [ ] 11.3 添加配置项控制是否启用
- [ ] 11.4 支持导出到 Jaeger/Zipkin

---

### 12. 添加 gosec 安全扫描

**问题描述**: 缺乏安全静态分析

**执行步骤**:
- [ ] 12.1 安装 gosec: `go install github.com/securego/gosec/v2/cmd/gosec@latest`
- [ ] 12.2 在 Makefile 中添加 `security` 目标
- [ ] 12.3 修复发现的安全问题
- [ ] 12.4 集成到 CI 流程

---

### 13. 重构报告生成为 Visitor 模式

**问题描述**: `generateCombinedExcel/HTML` 函数使用复杂 `if` 链判断可选结果

**执行步骤**:
- [ ] 13.1 定义 `ReportVisitor` 接口
- [ ] 13.2 每种服务结果实现 `Accept(visitor)` 方法
- [ ] 13.3 Excel/HTML Writer 实现 Visitor
- [ ] 13.4 移除硬编码的条件判断

---

## 完成检查清单

### 发布前必须完成
- [ ] P0-1: Excel 测试全部通过
- [ ] P0-2: VM 认证支持已实现
- [ ] 所有 `make test` 通过
- [ ] 所有 `make lint` 通过

### 质量目标
- [ ] 整体测试覆盖率 >= 70%
- [ ] 无忽略的错误返回值
- [ ] run.go 行数 < 300

### 文档同步
- [ ] README 覆盖率数据准确
- [ ] 配置示例包含新增选项
- [ ] CHANGELOG 已更新

---

## 进度追踪

| 任务 | 优先级 | 状态 | 负责人 | 完成日期 |
|------|--------|------|--------|----------|
| 1. 修复 Excel 测试 | P0 | ⬜ 待开始 | | |
| 2. VM 认证支持 | P0 | ⬜ 待开始 | | |
| 3. 重构 run.go | P1 | ⬜ 待开始 | | |
| 4. ES 测试 | P1 | ⬜ 待开始 | | |
| 5. Tomcat 测试 | P1 | ⬜ 待开始 | | |
| 6. 修复忽略错误 | P1 | ⬜ 待开始 | | |
| 7. 熔断器 | P2 | ⬜ 待开始 | | |
| 8. N9E 分页 | P2 | ⬜ 待开始 | | |
| 9. CLI 测试 | P2 | ⬜ 待开始 | | |
| 10. 文档修复 | P2 | ⬜ 待开始 | | |
| 11. OTEL | P3 | ⬜ 待开始 | | |
| 12. gosec | P3 | ⬜ 待开始 | | |
| 13. Visitor 重构 | P3 | ⬜ 待开始 | | |

---

*本文档由项目质量评估自动生成，请根据实际情况调整优先级和执行步骤*
