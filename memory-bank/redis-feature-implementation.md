# Redis 集群巡检功能 - 开发实施计划

> 本计划聚焦于 Redis 6.2 三主三从集群巡检功能的 MVP 实现。

---

## 功能概述

基于 Categraf 采集的 Redis 监控数据，实现 Redis 集群巡检功能。巡检项包括：

| 巡检项                  | 说明                      | 数据来源                                         | MVP 状态     |
| ----------------------- | ------------------------- | ------------------------------------------------ | ------------ |
| 巡检时间                | 巡检执行时间              | 系统时间                                         | ✅ 待实现    |
| IP 地址                 | Redis 实例 IP             | `address` 标签解析                               | ✅ 待实现    |
| 应用类型                | Redis                     | 固定值                                           | ✅ 待实现    |
| Redis 版本              | Redis 版本号              | Categraf command 扩展采集                        | ⏳ 显示 N/A |
| Redis 端口              | 实例端口                  | `address` 标签解析                               | ✅ 待实现    |
| 是否普通用户启动        | 非 root 用户启动          | -                                                | ⏳ 显示 N/A |
| 是否正常连接            | 连接状态                  | `redis_up`                                       | ✅ 待实现    |
| 是否启用集群            | 集群模式                  | `redis_cluster_enabled`                          | ✅ 待实现    |
| 集群状态                | 主从链接状态              | `redis_master_link_status`（slave 节点）         | ✅ 待实现    |
| 当前节点角色            | master/slave              | `replica_role` 标签 + `redis_connected_slaves`   | ✅ 待实现    |
| Master 节点偏移量       | master 的复制偏移量       | `redis_master_repl_offset`                       | ✅ 待实现    |
| 对应 Master 端口        | slave 对应的 master 端口  | `redis_master_port`                              | ✅ 待实现    |
| 对应复制节点偏移量      | slave 的复制偏移量        | `redis_slave_repl_offset`                        | ✅ 待实现    |
| 最大连接数              | maxclients                | `redis_maxclients`                               | ✅ 待实现    |

---

## 关键技术决策

### 1. Redis 集群模式支持策略

**MVP 支持 Redis 6.2 Cluster 模式**（3 台服务器 3 主 3 从），同时支持 3 主 6 从配置。

| 版本 | 集群模式                | 配置值 | 同步状态判断方式                             |
| ---- | ----------------------- | ------ | -------------------------------------------- |
| 6.2  | 3 主 3 从 Cluster       | `3m3s` | `master_link_status=1` 表示从节点链接正常    |
| 6.2  | 3 主 6 从 Cluster       | `3m6s` | 每个 master 有 2 个 slave，检查关联 slave 数 |

**配置说明**：

- 通过 `config.yaml` 中的 `cluster_mode` 字段指定模式
- `3m3s`：每个 master 期望 1 个 slave（默认）
- `3m6s`：每个 master 期望 2 个 slave
- 巡检逻辑根据模式检查 slave 连接数是否符合预期

### 2. 实例标识方式

- **主要标识**：`address` 标签（格式：`IP:Port`）
- **示例**：`192.18.102.2:7000`
- **解析**：分割冒号获取 IP 和端口
- **约束**：每台服务器运行 2 个 Redis 实例（端口 7000 和 7001）

### 3. 监控指标映射（基于 redis.txt 验证）

以下指标**已在生产环境验证存在**：

```yaml
# 已验证可用的指标
redis_up                                    # 连接状态 (1=正常) ✅
redis_cluster_enabled                       # 集群模式 (1=启用) ✅
redis_master_link_status                    # 主从链接状态 (1=正常，slave 节点) ✅
redis_connected_clients                     # 当前连接数 ✅
redis_maxclients                            # 最大连接数 ✅
redis_uptime_in_seconds                     # 运行时间 ✅
redis_master_repl_offset                    # 复制偏移量（master 和 slave 都有）✅
redis_slave_repl_offset                     # slave 复制偏移量 ✅
redis_master_port                           # slave 对应的 master 端口 ✅
redis_connected_slaves                      # 连接的 slave 数量 ✅

# 来自标签的信息
replica_role                                # 节点角色标签 (master/slave) ✅
address                                     # 实例地址 (IP:Port) ✅

# 不存在的指标/标签
master_host                                 # ❌ 不存在，无法直接获取 master IP
redis_version                               # ❌ 不存在，需要扩展采集
```

### 4. 节点角色判断逻辑

> **验证结论**：`replica_role` 标签**实际可靠**，在 redis.txt 中 slave 节点正确标记为 `replica_role=slave`。

**推荐判断方式**（双重验证）：

1. **主要方式**：`replica_role` 标签
   - `replica_role=master` → Master 节点
   - `replica_role=slave` → Slave 节点

2. **补充验证**：`redis_connected_slaves` 指标
   - `redis_connected_slaves > 0` → 确认为 Master（有 slave 连接）
   - `redis_connected_slaves = 0` 且有 `redis_master_link_status` 指标 → 确认为 Slave

**实现建议**：
```go
func determineRole(replicaRole string, connectedSlaves int, hasMasterLinkStatus bool) string {
    // 优先使用 replica_role 标签
    if replicaRole == "master" {
        return "master"
    }
    if replicaRole == "slave" {
        return "slave"
    }
    // 备用判断
    if connectedSlaves > 0 {
        return "master"
    }
    if hasMasterLinkStatus {
        return "slave"
    }
    return "unknown"
}
```

### 5. 对应 Master 信息获取

> **关键发现**：`master_host` 标签**不存在**，只有 `redis_master_port` 指标。

**MVP 方案**：
- **仅显示 Master 端口**：从 `redis_master_port` 指标获取
- **Master IP 显示 "N/A"**：无法直接从现有指标获取

**示例**：
- 192.18.102.2:7001 (slave) → Master 端口: 7000，Master IP: N/A

**后续扩展方案**（P2）：
1. **方案 A**：Categraf command 采集 `CLUSTER NODES` 输出，解析完整主从关系
2. **方案 B**：查询 VictoriaMetrics 中所有实例，通过端口匹配推算 Master IP

### 6. 复制延迟计算

**延迟计算公式**：
```
复制延迟 = slave 的 redis_master_repl_offset - slave 的 redis_slave_repl_offset
```

> **说明**：在 slave 节点上，`redis_master_repl_offset` 记录的是该 slave 已知的 master 偏移量，`redis_slave_repl_offset` 是 slave 自己的偏移量。两者差值即为复制延迟。

**验证数据**（来自 redis.txt）：
```
192.18.102.2:7001 (slave):
  redis_master_repl_offset = 650463763
  redis_slave_repl_offset  = 650463763
  延迟 = 0 字节（同步正常）
```

**MVP 方案**：对每个 slave 节点，直接计算其本地的两个偏移量差值。

### 7. 3 主 3 从集群实际拓扑

根据 `cluster nodes` 输出，**实际拓扑如下**：

| 实例地址 | 角色 | 负责 Slot | 对应 Master |
|----------|------|-----------|-------------|
| 192.18.102.3:7001 | **Master** | 0-5460 | - |
| 192.18.102.4:7001 | **Master** | 5461-10922 | - |
| 192.18.102.4:7000 | **Master** | 10923-16383 | - |
| 192.18.102.2:7000 | Slave | - | 192.18.102.3:7001 |
| 192.18.102.2:7001 | Slave | - | 192.18.102.4:7000 |
| 192.18.102.3:7000 | Slave | - | 192.18.102.4:7001 |

> **重要发现**：
> 1. **端口与角色无固定关系**：7000 和 7001 都可能是 master 或 slave
> 2. **同一服务器上的两个实例互不关联**：192.18.102.2:7000 的 master 是 192.18.102.3:7001，而不是本机的 7001
> 3. **Master-Slave 跨服务器分布**：这是典型的 Redis Cluster 高可用架构

### 8. 巡检状态判断规则

| 巡检项          | 正常条件                         | 警告条件                         | 严重条件                         |
| --------------- | -------------------------------- | -------------------------------- | -------------------------------- |
| 连接状态        | `redis_up=1`                     | -                                | `redis_up=0`                     |
| 主从链接（Slave）| `master_link_status=1`          | -                                | `master_link_status=0`           |
| 集群配置        | `cluster_enabled=1`              | -                                | `cluster_enabled=0`              |
| 连接使用率      | `<70%`                           | `70%-90%`                        | `>90%`                           |
| 复制延迟        | 偏移量差值 < 1MB                | 偏移量差值 1MB-10MB              | 偏移量差值 > 10MB                |

### 9. 集群健康状态判断

> **验证结论**：当前 Categraf Redis 插件**不采集 `CLUSTER INFO` 命令输出**，因此无法直接获取 `cluster_state`、`cluster_slots_ok` 等集群级别健康指标。

**已验证的集群相关指标**（redis.txt）：
- `redis_cluster_enabled`：是否启用集群模式（1=启用）
- `redis_cluster_connections`：集群内部连接数

**MVP 方案**：
- 仅通过各节点的 `redis_up` 和 `redis_master_link_status` 判断节点级别健康状态
- 集群整体健康状态由各节点状态汇总推断（所有节点正常 = 集群正常）

**后续扩展**（P3）：
- 扩展 Categraf 采集 `CLUSTER INFO` 命令输出
- 添加 `cluster_state`、`cluster_slots_ok`、`cluster_slots_fail` 等指标

### 10. 告警汇总逻辑

**告警按节点独立计数**，不汇总为集群级别告警：

| 维度 | 说明 |
|------|------|
| 告警粒度 | 每个 Redis 实例（IP:Port）独立生成告警 |
| 告警列表 | 所有实例的告警合并到一个列表 |
| 异常汇总表 | 按告警级别排序（严重优先），显示实例地址 |
| 统计摘要 | 统计正常/警告/严重/失败的实例数量 |

**实现方式**：
- 与 Host 巡检和 MySQL 巡检保持一致
- 单个实例触发多个告警时，该实例状态取最严重级别
- 报告中同时显示：汇总统计卡片 + 详细数据表 + 异常汇总表

### 11. 验证环境

**开发和功能验证使用陕西营销活动环境**：
- VictoriaMetrics 指向陕西营销活动监控系统（与 Host/MySQL 巡检共用）
- 已有 3 台服务器的 Redis 集群监控数据
- redis.txt 已验证包含所有必要指标

**Categraf 配置参考**（当前生产环境 redis.toml）：
```toml
interval = 15

[[instances]]
address = "192.18.102.2:7000"
password = "bg^PXTXfLa!j84qc"
pool_size = 2
gather_slowlog = true
slowlog_max_len = 100
slowlog_time_window = 30

[[instances]]
address = "192.18.102.2:7001"
password = "bg^PXTXfLa!j84qc"
pool_size = 2
gather_slowlog = true
slowlog_max_len = 100
slowlog_time_window = 30
```

---

## 阶段一：数据模型与配置扩展（步骤 1-4）

### 步骤 1：定义 Redis 实例模型

**操作**：

- 在 `internal/model/` 目录下创建 `redis.go` 文件
- 定义 `RedisInstance` 结构体，包含以下字段：
  - Address (string): 实例地址 (IP:Port)
  - IP (string): IP 地址
  - Port (int): 端口号
  - ApplicationType (string): 应用类型，固定为 "Redis"
  - Version (string): Redis 版本（MVP 阶段显示 N/A）
  - Role (string): 节点角色 (master/slave)
  - ClusterEnabled (bool): 是否启用集群
- 定义 `RedisInstanceStatus` 枚举（normal/warning/critical/failed）

**验证**：

- [ ] 执行 `go build ./internal/model/` 无编译错误
- [ ] 结构体字段覆盖所有巡检项

---

### 步骤 2：定义 Redis 巡检结果模型

**操作**：

- 在 `internal/model/redis.go` 中添加以下结构体：
  - `RedisInspectionResult`: 单个实例的巡检结果
    - Instance (*RedisInstance): 实例元信息
    - ConnectionStatus (bool): 连接状态 (redis_up=1)
    - ClusterEnabled (bool): 是否启用集群
    - ClusterState (string): 集群状态描述
    - MasterLinkStatus (bool): 主从链接状态（slave 节点）
    - MasterReplOffset (int64): 已知的 master 复制偏移量
    - SlaveReplOffset (int64): slave 复制偏移量
    - ReplicationLag (int64): 复制延迟（字节）
    - MasterPort (int): 对应的 master 端口（slave 节点，从 redis_master_port 获取）
    - MaxClients (int): 最大连接数
    - ConnectedClients (int): 当前连接数
    - ConnectedSlaves (int): 连接的 slave 数量（master 节点）
    - Uptime (int64): 运行时间（秒）
    - NonRootUser (string): 是否普通用户启动（MVP 固定 "N/A"）
    - Status (RedisInstanceStatus): 整体状态
    - Alerts ([]*RedisAlert): 告警列表
  - `RedisAlert`: 告警详情

**验证**：

- [ ] 执行 `go build ./internal/model/` 无编译错误
- [ ] 结构体字段能承载所有巡检数据

---

### 步骤 3：扩展配置结构体

**操作**：

- 在 `internal/config/config.go` 中添加 Redis 巡检配置：

  ```go
  type RedisInspectionConfig struct {
      Enabled           bool              `mapstructure:"enabled"`
      ClusterMode       string            `mapstructure:"cluster_mode"` // "3m3s", "3m6s"
      InstanceFilter    RedisFilter       `mapstructure:"instance_filter"`
      Thresholds        RedisThresholds   `mapstructure:"thresholds"`
  }

  type RedisFilter struct {
      AddressPatterns   []string          `mapstructure:"address_patterns"` // 地址匹配模式（glob）
      BusinessGroups    []string          `mapstructure:"business_groups"`  // 业务组（OR 关系）
      Tags              map[string]string `mapstructure:"tags"`             // 标签（AND 关系）
  }

  type RedisThresholds struct {
      ConnectionUsageWarning  float64 `mapstructure:"connection_usage_warning"`  // 默认 70%
      ConnectionUsageCritical float64 `mapstructure:"connection_usage_critical"` // 默认 90%
      ReplicationLagWarning   int64   `mapstructure:"replication_lag_warning"`   // 默认 1MB（保守值）
      ReplicationLagCritical  int64   `mapstructure:"replication_lag_critical"`  // 默认 10MB（保守值）
  }
  ```

**地址匹配模式说明**：
- 使用 glob 模式匹配（与 MySQL 实例筛选一致）
- 支持通配符 `*` 匹配任意字符
- 示例：`192.18.102.*` 匹配 `192.18.102.2:7000`、`192.18.102.3:7001` 等
- **注意**：不同项目 IP 段完全不同，需根据实际环境配置

- 在主配置结构体 `Config` 中添加 `Redis RedisInspectionConfig` 字段

**验证**：

- [ ] 执行 `go build ./internal/config/` 无编译错误
- [ ] 配置加载器能正确解析 Redis 配置节

---

### 步骤 4：创建 Redis 指标定义文件

**操作**：

- 在 `configs/` 目录下创建 `redis-metrics.yaml` 文件
- 定义所有 Redis 巡检指标的 PromQL 查询表达式
- 使用与 `metrics.yaml` 相同的格式

**指标定义内容**：

```yaml
redis_metrics:
  # 基础连接状态
  - name: redis_up
    display_name: "连接状态"
    query: "redis_up"
    category: connection
    note: "1=正常, 0=连接失败"

  # 集群配置
  - name: redis_cluster_enabled
    display_name: "集群模式"
    query: "redis_cluster_enabled"
    category: cluster
    note: "1=启用集群, 0=未启用"

  # 主从链接状态（仅 slave 节点有值）
  - name: redis_master_link_status
    display_name: "主从链接状态"
    query: "redis_master_link_status"
    category: replication
    note: "1=链接正常, 0=链接断开（仅 slave 节点）"

  # 连接相关
  - name: redis_connected_clients
    display_name: "当前连接数"
    query: "redis_connected_clients"
    category: connection

  - name: redis_maxclients
    display_name: "最大连接数"
    query: "redis_maxclients"
    category: connection

  # 复制偏移量
  - name: redis_master_repl_offset
    display_name: "Master 复制偏移量"
    query: "redis_master_repl_offset"
    category: replication
    note: "slave 节点上记录的是已知的 master 偏移量"

  - name: redis_slave_repl_offset
    display_name: "Slave 复制偏移量"
    query: "redis_slave_repl_offset"
    category: replication
    note: "slave 自己的复制偏移量"

  # 运行状态
  - name: redis_uptime_in_seconds
    display_name: "运行时间"
    query: "redis_uptime_in_seconds"
    category: status
    format: duration

  # 对应 master 端口（slave 节点）
  - name: redis_master_port
    display_name: "Master 端口"
    query: "redis_master_port"
    category: replication
    note: "slave 节点对应的 master 端口"

  # 连接的 slave 数量（用于角色判断）
  - name: redis_connected_slaves
    display_name: "连接的 Slave 数"
    query: "redis_connected_slaves"
    category: replication
    note: ">0 表示 master 节点"

  # 待定项 - MVP 阶段显示 N/A
  - name: redis_version
    display_name: "Redis 版本"
    query: ""
    category: info
    status: pending
    note: "需要扩展 Categraf command 配置采集"

  - name: non_root_user
    display_name: "非 root 用户启动"
    query: ""
    category: security
    status: pending
    note: "MVP 阶段显示 N/A"
```

**验证**：

- [ ] YAML 文件格式正确，可被解析
- [ ] 所有巡检项都有对应的指标定义
- [ ] 待定项正确标记 `status: pending`

---

## 阶段二：Redis 数据采集服务（步骤 5-8）

### 步骤 5：创建 Redis 采集器结构

**操作**：

- 在 `internal/service/` 目录下创建 `redis_collector.go` 文件
- 定义 `RedisCollector` 结构体：
  - vmClient (*vm.Client): VictoriaMetrics 客户端
  - config (*config.RedisInspectionConfig): Redis 配置
  - metrics ([]*model.RedisMetricDefinition): 指标定义列表
  - logger (zerolog.Logger): 日志器
- 实现构造函数 `NewRedisCollector`

**验证**：

- [ ] 执行 `go build ./internal/service/` 无编译错误

---

### 步骤 6：实现 Redis 实例发现

**操作**：

- 在 `RedisCollector` 中实现 `DiscoverInstances` 方法
- 查询 `redis_up` 指标获取所有 Redis 实例的 `address` 标签
- 解析 `address` 标签提取 IP 和端口
- 从 `replica_role` 标签获取角色信息
- 根据配置的 `InstanceFilter` 过滤实例
- 返回 `[]*model.RedisInstance` 列表

**实现逻辑**：

```go
func (c *RedisCollector) DiscoverInstances(ctx context.Context) ([]*model.RedisInstance, error) {
    // 1. 查询 redis_up 获取所有实例
    // 2. 从 address 标签解析 IP:Port
    // 3. 从 replica_role 标签获取角色（已验证可靠）
    // 4. 应用地址/业务组/标签过滤规则
    // 5. 构建实例列表
}
```

**验证**：

- [ ] 能够正确发现所有 Redis 实例（6 个）
- [ ] IP 和端口解析正确
- [ ] 过滤规则正确应用
- [ ] 执行 `go test ./internal/service/ -run TestRedisDiscoverInstances` 通过

---

### 步骤 7：实现 Redis 指标采集

**操作**：

- 在 `RedisCollector` 中实现 `CollectMetrics` 方法
- 按实例 `address` 标签进行过滤查询
- 采集所有配置的 Redis 指标
- 处理节点角色双重验证逻辑：
  - 主要：`replica_role` 标签
  - 补充：`redis_connected_slaves > 0` 表示 master
- 计算复制延迟：`redis_master_repl_offset - redis_slave_repl_offset`
- 返回 `map[string]*model.RedisMetrics` (key 为 address)

**关键处理**：

```go
// 复制延迟计算（仅 slave 节点）
func calculateReplicationLag(masterOffset, slaveOffset int64) int64 {
    if slaveOffset == 0 {
        return 0 // 非 slave 节点
    }
    lag := masterOffset - slaveOffset
    if lag < 0 {
        return 0 // 数据异常，返回 0
    }
    return lag
}
```

**验证**：

- [ ] 能够采集所有 Redis 指标
- [ ] 节点角色正确识别（3 master + 3 slave）
- [ ] 复制偏移量和延迟正确计算
- [ ] 执行 `go test ./internal/service/ -run TestRedisCollectMetrics` 通过

---

### 步骤 8：编写 Redis 采集器单元测试

**操作**：

- 在 `internal/service/redis_collector_test.go` 中编写测试
- 使用 httptest 模拟 VictoriaMetrics API 响应
- 测试用例覆盖：
  - 正常发现 Redis 实例
  - 正常采集所有指标
  - 地址解析（IP:Port 格式）
  - 节点角色判断（master/slave 双重验证）
  - 复制延迟计算
  - 实例过滤

**验证**：

- [ ] 执行 `go test ./internal/service/ -run TestRedis` 全部通过
- [ ] 测试覆盖率达到 70% 以上

---

## 阶段三：Redis 巡检评估服务（步骤 9-11）

### 步骤 9：实现 Redis 状态评估器

**操作**：

- 在 `internal/service/` 目录下创建 `redis_evaluator.go` 文件
- 定义 `RedisEvaluator` 结构体
- 实现 `Evaluate` 方法：
  - 根据阈值配置评估各指标状态
  - 生成告警信息
  - 确定实例整体状态（取最严重级别）

**评估规则实现**：

```go
func (e *RedisEvaluator) evaluateConnectionUsage(current, max int) AlertLevel {
    usage := float64(current) / float64(max) * 100
    if usage > e.thresholds.ConnectionUsageCritical {
        return AlertLevelCritical
    }
    if usage > e.thresholds.ConnectionUsageWarning {
        return AlertLevelWarning
    }
    return AlertLevelNormal
}

func (e *RedisEvaluator) evaluateMasterLinkStatus(status bool, role string) AlertLevel {
    // 仅对 slave 节点检查
    if role != "slave" {
        return AlertLevelNormal
    }
    if !status {
        return AlertLevelCritical
    }
    return AlertLevelNormal
}

func (e *RedisEvaluator) evaluateReplicationLag(lag int64) AlertLevel {
    if lag > e.thresholds.ReplicationLagCritical {
        return AlertLevelCritical
    }
    if lag > e.thresholds.ReplicationLagWarning {
        return AlertLevelWarning
    }
    return AlertLevelNormal
}
```

**验证**：

- [ ] 连接使用率 75% 被判定为警告
- [ ] 连接使用率 95% 被判定为严重
- [ ] 主从链接断开被判定为严重（slave 节点）
- [ ] 复制延迟超阈值被正确判定
- [ ] 执行 `go test ./internal/service/ -run TestRedisEvaluator` 通过

---

### 步骤 10：实现 Redis 巡检编排服务

**操作**：

- 在 `internal/service/` 目录下创建 `redis_inspector.go` 文件
- 定义 `RedisInspector` 结构体，整合采集器和评估器
- 实现 `Inspect` 方法，协调完整巡检流程：
  1. 发现实例
  2. 采集指标
  3. 评估状态
  4. 汇总结果

**验证**：

- [ ] 能够完成端到端的 Redis 巡检流程
- [ ] 单实例失败不影响其他实例
- [ ] 巡检摘要统计正确
- [ ] 执行 `go test ./internal/service/ -run TestRedisInspector` 通过

---

### 步骤 11：编写 Redis 巡检服务集成测试

**操作**：

- 在 `internal/service/redis_inspector_test.go` 中编写集成测试
- 模拟完整的巡检场景：
  - 正常 3 主 3 从集群（所有节点在线）
  - 主从链接断开场景（slave 节点离线）
  - 连接数过高场景
  - 复制延迟场景
  - 多实例巡检

**验证**：

- [ ] 执行 `go test ./internal/service/ -run TestRedisInspector` 全部通过
- [ ] 各场景告警逻辑正确

---

## 阶段四：报告生成扩展（步骤 12-15）

### 步骤 12：扩展 Excel 报告 - Redis 工作表

**操作**：

- 在 `internal/report/excel/writer.go` 中添加 Redis 报告功能
- 创建 "Redis 巡检" 工作表（独立于 Host 和 MySQL 工作表）
- 表头：巡检时间、IP 地址、端口、应用类型、版本、是否普通用户启动、连接状态、集群模式、集群状态、节点角色、Master 端口、复制延迟、最大连接数、整体状态
- 应用条件格式（与主机巡检一致）

**验证**：

- [ ] Excel 文件包含 "Redis 巡检" 工作表
- [ ] 表头完整正确
- [ ] 数据填充正确
- [ ] 条件格式正确应用

---

### 步骤 13：扩展 Excel 报告 - Redis 异常汇总

**操作**：

- 新增 "Redis 异常" 工作表（独立于 Host 和 MySQL 异常汇总）
- 仅包含有告警的 Redis 实例记录
- 显示告警类型、当前值、阈值

**验证**：

- [ ] 异常汇总表正确筛选告警记录
- [ ] 告警详情完整

---

### 步骤 14：扩展 HTML 报告 - Redis 区域

**操作**：

- 在 HTML 模板中添加 Redis 巡检区域（独立区域，位于 MySQL 区域之后）
- 使用与主机巡检一致的卡片和表格样式
- 添加 Redis 统计摘要卡片
- 实现表格排序功能

**验证**：

- [ ] HTML 报告正确显示 Redis 巡检区域
- [ ] 样式与主机巡检区域一致
- [ ] 排序功能正常

---

### 步骤 15：更新示例配置文件

**操作**：

- 在 `configs/config.example.yaml` 中添加 Redis 配置示例
- 添加详细注释说明各配置项

**配置示例**：

```yaml
# Redis 集群巡检配置
redis:
  enabled: true
  # 集群模式
  # 可选值: 3m3s (3主3从), 3m6s (3主6从)
  cluster_mode: "3m3s"

  # 实例筛选（可选）
  instance_filter:
    address_patterns:
      - "192.18.102.*"
    business_groups:
      - "生产Redis"
    tags:
      env: "prod"

  # 阈值配置
  thresholds:
    # 连接使用率阈值
    connection_usage_warning: 70   # 警告阈值 (%)
    connection_usage_critical: 90  # 严重阈值 (%)

    # 复制延迟阈值（字节偏移量差值）
    # 注意：目前无生产环境经验值，以下为保守默认值
    # 建议根据实际业务负载和网络情况调整
    # 正常情况下复制延迟应该很小（KB级）
    replication_lag_warning: 1048576    # 1MB - 保守警告阈值
    replication_lag_critical: 10485760  # 10MB - 保守严重阈值
```

**验证**：

- [ ] 配置文件格式正确
- [ ] 配置可被正确加载和验证

---

## 阶段五：CLI 扩展与测试（步骤 16-18）

### 步骤 16：扩展 run 命令支持 Redis 巡检

**操作**：

- 在 `cmd/inspect/cmd/run.go` 中集成 Redis 巡检
- 添加 `--redis-only` 标志（仅执行 Redis 巡检）
- 添加 `--skip-redis` 标志（跳过 Redis 巡检）
- 在巡检流程中加入 Redis 巡检步骤

**验证**：

- [ ] `./bin/inspect run -c config.yaml` 包含 Redis 巡检
- [ ] `--redis-only` 标志正确工作
- [ ] `--skip-redis` 标志正确工作

---

### 步骤 17：端到端测试

**操作**：

- 使用**陕西营销活动环境**进行验证测试
- 配置真实 VictoriaMetrics 地址进行测试
- 验证 Excel 报告中 Redis 数据正确性
- 验证 HTML 报告中 Redis 区域显示正确
- 测试 3 主 3 从集群场景

**验证**：

- [ ] Redis 实例正确发现（6 个实例）
- [ ] 节点角色正确识别（3 master + 3 slave）
- [ ] 主从链接状态正确判断
- [ ] 复制偏移量和延迟正确获取
- [ ] 报告生成完整

---

### 步骤 18：更新文档

**操作**：

- 更新 README.md 添加 Redis 巡检说明
- 添加 Categraf redis.toml 配置参考
- 添加 Redis 巡检项说明

**验证**：

- [ ] 文档完整描述 Redis 巡检功能
- [ ] 配置示例可直接使用

---

## 验证计划

### 自动化测试

执行以下命令验证所有测试通过：

```bash
# 运行所有 Redis 相关测试
go test ./internal/model/ -run TestRedis -v
go test ./internal/service/ -run TestRedis -v
go test ./internal/report/ -run TestRedis -v

# 运行完整测试套件
go test ./... -v

# 检查测试覆盖率
go test ./internal/service/ -coverprofile=coverage.out
go tool cover -func=coverage.out | grep redis
```

### 手动验证（使用陕西营销活动环境）

1. **配置验证**：

   ```bash
   ./bin/inspect validate -c config.yaml
   ```

   预期：配置验证通过，无错误信息

2. **Redis 巡检执行**：

   ```bash
   ./bin/inspect run -c config.yaml --redis-only
   ```

   预期：

   - 控制台显示发现的 Redis 实例数量（6 个）
   - 显示各实例的巡检状态
   - 生成包含 Redis 工作表的 Excel 报告

3. **报告内容验证**：
   - 打开生成的 Excel 文件
   - 确认 "Redis 巡检" 工作表存在
   - 确认数据与 VictoriaMetrics 查询结果一致

---

## 待定项说明

以下巡检项在 MVP 阶段显示 "N/A"，后续通过扩展 Categraf 采集配置实现：

| 巡检项           | MVP 处理                         | 后续实现方式               |
| ---------------- | -------------------------------- | -------------------------- |
| Redis 版本       | 显示 N/A                         | Categraf command 采集 INFO 命令 |
| 是否普通用户启动 | 显示 N/A                         | 扩展 Categraf 自定义脚本   |
| 对应 Master IP   | 仅显示端口                       | Categraf command 采集 CLUSTER NODES |

### Redis 版本采集扩展方案（P2）

通过 Categraf 的 `commands` 配置采集 INFO 命令获取版本：

```toml
[[instances]]
address = "192.18.102.2:7000"
password = "xxx"

# 自定义命令采集 Redis 版本
commands = [
    {command = ["info", "server"], metric = "redis_info_server"}
]
```

> **注意**：需要验证 Categraf Redis 插件是否支持此功能，以及如何从返回结果中提取 `redis_version` 字段。

---

## 后续扩展计划

### P1: 3 主 6 从支持

- 调整节点角色识别逻辑
- 每个 master 有 2 个 slave
- 更新巡检报告展示

### P2: Redis 版本 & Master IP 采集

- 扩展 Categraf 采集 INFO 命令中的 redis_version
- 扩展 Categraf 采集 CLUSTER NODES 输出，解析完整主从关系
- 添加版本标签到监控指标

### P3: 更多巡检项

- 内存使用率检查
- AOF/RDB 持久化状态
- 慢查询日志分析
- 命令统计分析

---

## 版本记录

| 版本 | 日期       | 说明                                                                 |
| ---- | ---------- | -------------------------------------------------------------------- |
| v1.0 | 2025-12-16 | 初始版本，聚焦 Redis 6.2 三主三从集群巡检                             |
| v1.1 | 2025-12-16 | 根据 redis.txt 验证更新：确认指标可用性、修正拓扑说明、明确角色判断逻辑、调整复制延迟计算方式、更新待定项说明 |
| v1.2 | 2025-12-16 | 根据用户澄清更新：明确 3m3s/3m6s 集群配置、glob 地址匹配模式、保守复制延迟阈值（无生产经验值）、集群健康状态判断（无 cluster_state 指标）、节点独立告警汇总逻辑 |
