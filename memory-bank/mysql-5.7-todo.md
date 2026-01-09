# MySQL 5.7 主从/双主模式巡检开发计划

> 当前已支持 MySQL 8.0 MGR 模式，本文档规划 MySQL 5.7 主从/双主模式的扩展实现。

---

## 功能概述

### 目标

在现有 MySQL 巡检框架基础上，扩展支持 MySQL 5.7 的两种集群模式：

| 模式 | 配置值 | 说明 | 典型架构 |
|------|--------|------|----------|
| 主从模式 | `master-slave` | 一主多从 | 1 Master + N Slave |
| 双主模式 | `dual-master` | 互为主从 | 2 Master (互相复制) |

### 与 MGR 模式的差异

| 对比项 | MySQL 8.0 MGR | MySQL 5.7 主从/双主 |
|--------|---------------|---------------------|
| 同步状态判断 | `mgr_state_online` | `Slave_IO_Running` + `Slave_SQL_Running` |
| 成员数检查 | `mgr_member_count` | 不适用 |
| 角色判断 | `mgr_role_primary` | 基于复制关系判断 |
| 复制延迟 | 无 | `Seconds_Behind_Master` |
| 复制状态指标 | MGR 专属指标 | `mysql_slave_status_*` 系列指标 |

---

## 前置条件

### 1. Categraf 采集配置

需要在 Categraf `mysql.toml` 中启用 slave 状态采集：

```toml
[[instances]]
address = "127.0.0.1:3306"
username = "monitor"
password = "your-password"

# 关键配置：启用 slave 状态采集
gather_slave_status = true   # MySQL 5.7 主从/双主必须设为 true
gather_variables = true
```

### 2. 预期采集的指标

启用 `gather_slave_status = true` 后，Categraf 会采集以下指标：

| 指标名称 | 说明 | 值域 |
|----------|------|------|
| `mysql_slave_status_slave_io_running` | IO 线程状态 | 1=Yes, 0=No |
| `mysql_slave_status_slave_sql_running` | SQL 线程状态 | 1=Yes, 0=No |
| `mysql_slave_status_seconds_behind_master` | 复制延迟秒数 | 整数，NULL 时为 -1 |
| `mysql_slave_status_master_host` | 主库地址 | 标签值 |
| `mysql_slave_status_master_port` | 主库端口 | 标签值 |
| `mysql_slave_status_last_io_error` | 最后 IO 错误 | 标签值 |
| `mysql_slave_status_last_sql_error` | 最后 SQL 错误 | 标签值 |

---

## 开发任务分解

### 阶段一：指标定义与配置扩展

#### 1. 扩展 MySQL 指标定义文件

**文件**: `configs/mysql-metrics.yaml`

**操作**:
- [ ] 1.1 添加主从/双主模式专属指标定义
- [ ] 1.2 为每个指标设置 `cluster_mode` 过滤条件
- [ ] 1.3 添加复制延迟相关指标

**新增指标定义**:

```yaml
# =============================================================================
# 主从/双主模式专属指标 (MySQL 5.7)
# =============================================================================

# Slave IO 线程状态
- name: slave_io_running
  display_name: "Slave IO 线程"
  query: "mysql_slave_status_slave_io_running"
  category: replication
  cluster_mode: master-slave  # 也适用于 dual-master
  note: "1=运行中, 0=停止"

# Slave SQL 线程状态
- name: slave_sql_running
  display_name: "Slave SQL 线程"
  query: "mysql_slave_status_slave_sql_running"
  category: replication
  cluster_mode: master-slave
  note: "1=运行中, 0=停止"

# 复制延迟
- name: seconds_behind_master
  display_name: "复制延迟(秒)"
  query: "mysql_slave_status_seconds_behind_master"
  category: replication
  cluster_mode: master-slave
  format: duration
  note: "复制落后主库的秒数，-1 表示无法获取"

# Master 信息（从标签提取）
- name: master_host
  display_name: "主库地址"
  query: "mysql_slave_status_slave_io_running"
  category: replication
  cluster_mode: master-slave
  label_extract: master_host
  note: "复制的源主库地址"

- name: master_port
  display_name: "主库端口"
  query: "mysql_slave_status_slave_io_running"
  category: replication
  cluster_mode: master-slave
  label_extract: master_port
  note: "复制的源主库端口"

# 复制错误信息
- name: last_io_error
  display_name: "最后 IO 错误"
  query: "mysql_slave_status_slave_io_running"
  category: replication
  cluster_mode: master-slave
  label_extract: last_io_error
  note: "最近一次 IO 线程错误信息"

- name: last_sql_error
  display_name: "最后 SQL 错误"
  query: "mysql_slave_status_slave_io_running"
  category: replication
  cluster_mode: master-slave
  label_extract: last_sql_error
  note: "最近一次 SQL 线程错误信息"
```

**验收标准**:
- [ ] YAML 文件格式正确，可被 `config.LoadMySQLMetrics()` 解析
- [ ] 主从模式指标与 MGR 模式指标正确区分

---

#### 2. 扩展配置阈值定义

**文件**: `internal/config/config.go`

**操作**:
- [ ] 2.1 在 `MySQLThresholds` 结构体中添加复制延迟阈值

**新增字段**:

```go
type MySQLThresholds struct {
    // 现有字段
    ConnectionUsageWarning  float64 `mapstructure:"connection_usage_warning"`
    ConnectionUsageCritical float64 `mapstructure:"connection_usage_critical"`
    MGRMemberCountExpected  int     `mapstructure:"mgr_member_count_expected"`
    
    // 新增: 主从/双主模式阈值
    ReplicationLagWarning   int `mapstructure:"replication_lag_warning"`   // 复制延迟警告阈值(秒)，默认 60
    ReplicationLagCritical  int `mapstructure:"replication_lag_critical"`  // 复制延迟严重阈值(秒)，默认 300
}
```

- [ ] 2.2 在 `internal/config/loader.go` 中添加默认值

```go
v.SetDefault("mysql.thresholds.replication_lag_warning", 60)    // 1 分钟
v.SetDefault("mysql.thresholds.replication_lag_critical", 300)  // 5 分钟
```

**验收标准**:
- [ ] 配置加载器能正确解析新阈值
- [ ] 未配置时使用合理默认值

---

#### 3. 更新示例配置文件

**文件**: `configs/config.example.yaml`

**操作**:
- [ ] 3.1 添加主从/双主模式的配置示例
- [ ] 3.2 添加复制延迟阈值配置说明

**新增配置示例**:

```yaml
mysql:
  enabled: true
  
  # 集群模式配置
  # 可选值: 
  #   - mgr: MySQL 8.0 Group Replication
  #   - master-slave: MySQL 5.7 主从模式
  #   - dual-master: MySQL 5.7 双主模式
  cluster_mode: "master-slave"
  
  thresholds:
    # 连接使用率阈值
    connection_usage_warning: 70
    connection_usage_critical: 90
    
    # MGR 模式专属 (仅 cluster_mode=mgr 时生效)
    mgr_member_count_expected: 3
    
    # 主从/双主模式专属 (仅 cluster_mode=master-slave/dual-master 时生效)
    # 复制延迟阈值（秒）
    replication_lag_warning: 60     # 延迟 > 60 秒触发警告
    replication_lag_critical: 300   # 延迟 > 300 秒触发严重告警
```

**验收标准**:
- [ ] 配置示例格式正确
- [ ] 注释清晰说明各模式的适用范围

---

### 阶段二：数据模型扩展

#### 4. 扩展 MySQL 巡检结果模型

**文件**: `internal/model/mysql.go`

**操作**:
- [ ] 4.1 在 `MySQLInspectionResult` 中添加主从模式专属字段

**新增字段**:

```go
type MySQLInspectionResult struct {
    // ... 现有字段 ...
    
    // 主从/双主模式专属字段 (仅非 MGR 模式有效)
    SlaveIORunning      bool   `json:"slave_io_running"`       // Slave IO 线程状态
    SlaveSQLRunning     bool   `json:"slave_sql_running"`      // Slave SQL 线程状态
    SecondsBehindMaster int    `json:"seconds_behind_master"`  // 复制延迟秒数，-1 表示无法获取
    MasterHost          string `json:"master_host"`            // 主库地址
    MasterPort          int    `json:"master_port"`            // 主库端口
    LastIOError         string `json:"last_io_error"`          // 最后 IO 错误
    LastSQLError        string `json:"last_sql_error"`         // 最后 SQL 错误
    IsMaster            bool   `json:"is_master"`              // 是否为主库 (通过复制关系推断)
}
```

- [ ] 4.2 添加辅助方法判断复制状态

```go
// IsSlaveHealthy returns true if slave replication is running normally.
// Only applicable for master-slave and dual-master modes.
func (r *MySQLInspectionResult) IsSlaveHealthy() bool {
    return r.SlaveIORunning && r.SlaveSQLRunning
}

// GetReplicationStatus returns a human-readable replication status.
func (r *MySQLInspectionResult) GetReplicationStatus() string {
    if r.Instance.ClusterMode.IsMGR() {
        if r.MGRStateOnline {
            return "ONLINE"
        }
        return "OFFLINE"
    }
    
    if r.IsMaster {
        return "MASTER"
    }
    
    if r.SlaveIORunning && r.SlaveSQLRunning {
        return "复制正常"
    }
    if !r.SlaveIORunning && !r.SlaveSQLRunning {
        return "复制停止"
    }
    if !r.SlaveIORunning {
        return "IO线程停止"
    }
    return "SQL线程停止"
}
```

**验收标准**:
- [ ] 编译通过: `go build ./internal/model/`
- [ ] 单元测试: `go test ./internal/model/ -run TestMySQL -v`

---

### 阶段三：采集器扩展

#### 5. 扩展 MySQL 采集器支持主从指标

**文件**: `internal/service/mysql_collector.go`

**操作**:
- [ ] 5.1 修改 `filterActiveMetrics` 方法，支持 `master-slave` 和 `dual-master` 模式过滤
- [ ] 5.2 实现主从状态指标采集逻辑
- [ ] 5.3 实现主库/从库角色判断逻辑

**角色判断逻辑**:

```go
// 判断实例是主库还是从库
// 1. 如果 slave_io_running/slave_sql_running 指标存在，说明配置了复制，是从库
// 2. 如果上述指标不存在或为 NULL，说明是主库
func (c *MySQLCollector) determineRole(metrics map[string]*model.MySQLMetricValue) bool {
    slaveIO := metrics["slave_io_running"]
    slaveSql := metrics["slave_sql_running"]
    
    // 如果没有 slave 状态指标，认为是主库
    if slaveIO == nil && slaveSql == nil {
        return true // isMaster = true
    }
    
    // 如果指标存在但值为 NA，也认为是主库
    if slaveIO != nil && slaveIO.IsNA {
        return true
    }
    
    return false // 是从库
}
```

- [ ] 5.4 处理双主模式的特殊情况（两个节点都既是主又是从）

**验收标准**:
- [ ] 单元测试覆盖主从模式采集: `go test ./internal/service/ -run TestMySQLCollector -v`
- [ ] 测试覆盖率 >= 80%

---

#### 6. 添加主从模式采集单元测试

**文件**: `internal/service/mysql_collector_test.go`

**操作**:
- [ ] 6.1 添加主从模式实例发现测试用例
- [ ] 6.2 添加 slave 状态指标采集测试用例
- [ ] 6.3 添加主库/从库角色判断测试用例
- [ ] 6.4 添加双主模式测试用例

**测试场景**:

```go
func TestMySQLCollector_MasterSlaveMode(t *testing.T) {
    tests := []struct {
        name           string
        clusterMode    string
        mockResponses  map[string]string
        expectedRole   string // "master" or "slave"
        expectedStatus bool   // slave replication healthy
    }{
        {
            name:        "master node without slave status",
            clusterMode: "master-slave",
            // master 节点查询 slave_status 返回空
            expectedRole: "master",
        },
        {
            name:        "slave node with healthy replication",
            clusterMode: "master-slave",
            // slave 节点返回 IO=1, SQL=1
            expectedRole:   "slave",
            expectedStatus: true,
        },
        {
            name:        "slave node with broken IO thread",
            clusterMode: "master-slave",
            // slave 节点返回 IO=0, SQL=1
            expectedRole:   "slave",
            expectedStatus: false,
        },
        {
            name:        "dual-master node A",
            clusterMode: "dual-master",
            // 双主节点既有 slave_status 也可能是 master
            expectedRole: "dual-master",
        },
    }
    // ...
}
```

**验收标准**:
- [ ] 所有测试用例通过
- [ ] 覆盖正常、异常、边界情况

---

### 阶段四：评估器扩展

#### 7. 扩展 MySQL 评估器支持主从告警

**文件**: `internal/service/mysql_evaluator.go`

**操作**:
- [ ] 7.1 添加复制状态评估方法

```go
// evaluateSlaveReplication evaluates slave replication health.
// Only applicable for master-slave and dual-master modes.
func (e *MySQLEvaluator) evaluateSlaveReplication(result *model.MySQLInspectionResult) {
    // 跳过主库节点
    if result.IsMaster {
        return
    }
    
    // 检查 IO 线程
    if !result.SlaveIORunning {
        alert := &model.MySQLAlert{
            Address:           result.GetAddress(),
            MetricName:        "slave_io_running",
            MetricDisplayName: "Slave IO 线程",
            CurrentValue:      0,
            Level:             model.AlertLevelCritical,
            Message:           fmt.Sprintf("Slave IO 线程停止: %s", result.LastIOError),
        }
        result.AddAlert(alert)
    }
    
    // 检查 SQL 线程
    if !result.SlaveSQLRunning {
        alert := &model.MySQLAlert{
            Address:           result.GetAddress(),
            MetricName:        "slave_sql_running",
            MetricDisplayName: "Slave SQL 线程",
            CurrentValue:      0,
            Level:             model.AlertLevelCritical,
            Message:           fmt.Sprintf("Slave SQL 线程停止: %s", result.LastSQLError),
        }
        result.AddAlert(alert)
    }
}
```

- [ ] 7.2 添加复制延迟评估方法

```go
// evaluateReplicationLag evaluates replication delay.
func (e *MySQLEvaluator) evaluateReplicationLag(result *model.MySQLInspectionResult) {
    // 跳过主库节点
    if result.IsMaster {
        return
    }
    
    // -1 表示无法获取延迟 (可能复制已断开)
    if result.SecondsBehindMaster < 0 {
        return // 已经由 slave replication 告警覆盖
    }
    
    lag := result.SecondsBehindMaster
    
    if lag > e.thresholds.ReplicationLagCritical {
        alert := &model.MySQLAlert{
            Address:           result.GetAddress(),
            MetricName:        "seconds_behind_master",
            MetricDisplayName: "复制延迟",
            CurrentValue:      float64(lag),
            FormattedValue:    formatDuration(lag),
            WarningThreshold:  float64(e.thresholds.ReplicationLagWarning),
            CriticalThreshold: float64(e.thresholds.ReplicationLagCritical),
            Level:             model.AlertLevelCritical,
            Message:           fmt.Sprintf("复制延迟 %d 秒，超过严重阈值 %d 秒", lag, e.thresholds.ReplicationLagCritical),
        }
        result.AddAlert(alert)
    } else if lag > e.thresholds.ReplicationLagWarning {
        alert := &model.MySQLAlert{
            Address:           result.GetAddress(),
            MetricName:        "seconds_behind_master",
            MetricDisplayName: "复制延迟",
            CurrentValue:      float64(lag),
            FormattedValue:    formatDuration(lag),
            WarningThreshold:  float64(e.thresholds.ReplicationLagWarning),
            CriticalThreshold: float64(e.thresholds.ReplicationLagCritical),
            Level:             model.AlertLevelWarning,
            Message:           fmt.Sprintf("复制延迟 %d 秒，超过警告阈值 %d 秒", lag, e.thresholds.ReplicationLagWarning),
        }
        result.AddAlert(alert)
    }
}
```

- [ ] 7.3 修改 `EvaluateAll` 方法，根据集群模式选择评估逻辑

```go
func (e *MySQLEvaluator) EvaluateAll(results map[string]*model.MySQLInspectionResult) []*MySQLEvaluationResult {
    // ... 现有逻辑 ...
    
    for _, result := range results {
        // 通用评估
        e.evaluateConnectionUsage(result)
        e.evaluateBinlogConfig(result)
        
        // 根据集群模式选择评估逻辑
        if result.Instance.ClusterMode.IsMGR() {
            e.evaluateMGRMemberCount(result)
            e.evaluateMGRStateOnline(result)
        } else {
            // 主从/双主模式
            e.evaluateSlaveReplication(result)
            e.evaluateReplicationLag(result)
        }
    }
    
    // ...
}
```

**验收标准**:
- [ ] 编译通过
- [ ] 单元测试通过

---

#### 8. 添加主从模式评估单元测试

**文件**: `internal/service/mysql_evaluator_test.go`

**操作**:
- [ ] 8.1 添加复制状态评估测试用例
- [ ] 8.2 添加复制延迟评估测试用例
- [ ] 8.3 添加边界条件测试

**测试用例**:

```go
func TestMySQLEvaluator_SlaveReplication(t *testing.T) {
    tests := []struct {
        name          string
        slaveIO       bool
        slaveSQL      bool
        isMaster      bool
        expectAlerts  int
        expectLevel   model.AlertLevel
    }{
        {"master node - no alerts", false, false, true, 0, ""},
        {"slave healthy", true, true, false, 0, ""},
        {"slave IO stopped", false, true, false, 1, model.AlertLevelCritical},
        {"slave SQL stopped", true, false, false, 1, model.AlertLevelCritical},
        {"slave both stopped", false, false, false, 2, model.AlertLevelCritical},
    }
    // ...
}

func TestMySQLEvaluator_ReplicationLag(t *testing.T) {
    tests := []struct {
        name        string
        lag         int
        isMaster    bool
        expectLevel model.AlertLevel
    }{
        {"master node - skip", 1000, true, ""},
        {"no lag", 0, false, ""},
        {"normal lag", 30, false, ""},           // < 60s warning
        {"warning lag", 100, false, model.AlertLevelWarning},  // 60-300s
        {"critical lag", 600, false, model.AlertLevelCritical}, // > 300s
        {"unknown lag", -1, false, ""},          // -1 means NULL
    }
    // ...
}
```

**验收标准**:
- [ ] 测试覆盖率 >= 90%
- [ ] 所有边界条件已覆盖

---

### 阶段五：报告生成扩展

#### 9. 扩展 Excel 报告支持主从模式

**文件**: `internal/report/excel/writer.go`

**操作**:
- [ ] 9.1 更新 MySQL 工作表表头，添加主从模式专属列
- [ ] 9.2 根据集群模式动态显示/隐藏列
- [ ] 9.3 添加复制延迟的条件格式

**表头调整**:

| 列 | MGR 模式 | 主从/双主模式 |
|----|----------|---------------|
| MGR 成员数 | 显示 | 隐藏或显示 N/A |
| MGR 角色 | 显示 | 隐藏或显示 N/A |
| MGR 状态 | 显示 | 隐藏或显示 N/A |
| 节点角色 | 隐藏 | 显示 (Master/Slave) |
| IO 线程 | 隐藏 | 显示 |
| SQL 线程 | 隐藏 | 显示 |
| 复制延迟 | 隐藏 | 显示 |
| 主库地址 | 隐藏 | 显示 |

**验收标准**:
- [ ] Excel 报告正确显示主从模式数据
- [ ] 条件格式正确应用于复制延迟列

---

#### 10. 扩展 HTML 报告支持主从模式

**文件**: `internal/report/html/writer.go`

**操作**:
- [ ] 10.1 更新 HTML 模板，支持主从模式字段
- [ ] 10.2 根据集群模式动态渲染表格列
- [ ] 10.3 添加复制延迟的颜色编码

**验收标准**:
- [ ] HTML 报告正确显示主从模式数据
- [ ] 表格排序功能正常

---

### 阶段六：测试与文档

#### 11. 端到端集成测试

**操作**:
- [ ] 11.1 准备主从模式测试环境（或 Mock 数据）
- [ ] 11.2 执行完整巡检流程
- [ ] 11.3 验证报告输出正确性

**测试命令**:

```bash
# 使用主从模式配置运行
./bin/inspect run -c config-master-slave.yaml

# 验证报告内容
# 1. 检查 MySQL 实例发现
# 2. 检查角色判断 (Master/Slave)
# 3. 检查复制状态
# 4. 检查复制延迟告警
```

**验收标准**:
- [ ] 主库正确识别
- [ ] 从库复制状态正确
- [ ] 复制延迟告警触发正确
- [ ] 报告格式正确

---

#### 12. 更新文档

**操作**:
- [ ] 12.1 更新 README.md 添加主从模式说明
- [ ] 12.2 添加 Categraf mysql.toml 主从模式配置示例
- [ ] 12.3 更新 `memory-bank/mysql-feature-implementation.md`
- [ ] 12.4 更新 `memory-bank/architecture.md`

**README 新增内容**:

```markdown
### MySQL 5.7 主从/双主模式配置

```yaml
mysql:
  enabled: true
  cluster_mode: "master-slave"  # 或 "dual-master"
  
  thresholds:
    connection_usage_warning: 70
    connection_usage_critical: 90
    replication_lag_warning: 60    # 秒
    replication_lag_critical: 300  # 秒
```

**Categraf 配置要求**:

```toml
[[instances]]
address = "127.0.0.1:3306"
gather_slave_status = true  # 必须启用
```
```

**验收标准**:
- [ ] 文档清晰完整
- [ ] 配置示例可直接使用

---

## 任务优先级

| 优先级 | 任务 | 预估工时 |
|--------|------|----------|
| P0 | 1-3: 指标定义与配置 | 2h |
| P0 | 4: 数据模型扩展 | 1h |
| P1 | 5-6: 采集器扩展 | 4h |
| P1 | 7-8: 评估器扩展 | 3h |
| P2 | 9-10: 报告生成扩展 | 3h |
| P2 | 11-12: 测试与文档 | 2h |

**总预估**: 15 小时 (约 2 个工作日)

---

## 进度追踪

| 任务 | 状态 | 负责人 | 完成日期 |
|------|------|--------|----------|
| 1. 扩展指标定义文件 | ⬜ 待开始 | | |
| 2. 扩展配置阈值定义 | ⬜ 待开始 | | |
| 3. 更新示例配置文件 | ⬜ 待开始 | | |
| 4. 扩展巡检结果模型 | ⬜ 待开始 | | |
| 5. 扩展采集器 | ⬜ 待开始 | | |
| 6. 采集器单元测试 | ⬜ 待开始 | | |
| 7. 扩展评估器 | ⬜ 待开始 | | |
| 8. 评估器单元测试 | ⬜ 待开始 | | |
| 9. 扩展 Excel 报告 | ⬜ 待开始 | | |
| 10. 扩展 HTML 报告 | ⬜ 待开始 | | |
| 11. 端到端测试 | ⬜ 待开始 | | |
| 12. 更新文档 | ⬜ 待开始 | | |

---

## 注意事项

### 1. 向后兼容性

- 现有 MGR 模式功能不受影响
- 配置文件新增字段都有默认值
- 报告格式保持一致

### 2. 测试环境

- 需要 MySQL 5.7 主从环境进行验证
- 或使用 Mock 数据进行单元测试

### 3. Categraf 依赖

- 必须确认 Categraf 的 `gather_slave_status` 功能正常
- 需要验证指标名称与预期一致

---

## 版本记录

| 版本 | 日期 | 说明 |
|------|------|------|
| v1.0 | 2026-01-09 | 初始版本，MySQL 5.7 主从/双主模式开发计划 |
