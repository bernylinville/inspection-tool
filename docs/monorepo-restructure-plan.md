# Monorepo 重构实施计划

> 版本：v1.0
> 创建日期：2026-01-12
> 目标：将现有巡检工具项目重构为 Monorepo 结构，支持巡检 CLI 和 CMDB 平台共存

---

## 1. 现状分析

### 1.1 当前项目结构

```
inspection-tool/                    # 单模块项目
├── go.mod                          # module inspection-tool
├── cmd/inspect/                    # CLI 入口
├── internal/                       # 私有代码
│   ├── client/                     # API 客户端（n9e/, vm/）
│   ├── config/                     # 配置管理
│   ├── model/                      # 数据模型
│   ├── service/                    # 业务逻辑
│   ├── report/                     # 报告生成
│   └── util/                       # 工具函数
├── configs/                        # 配置文件
├── templates/                      # HTML 模板
└── memory-bank/                    # 文档
```

### 1.2 可复用模块识别

| 模块 | 路径 | 复用价值 | 目标位置 |
|------|------|----------|----------|
| N9E Client | internal/client/n9e/ | ⭐⭐⭐⭐⭐ | pkg/n9e/ |
| VM Client | internal/client/vm/ | ⭐⭐⭐⭐⭐ | pkg/vm/ |
| 通用模型 | internal/model/ (部分) | ⭐⭐⭐⭐ | pkg/model/ |
| 工具函数 | internal/util/ | ⭐⭐⭐ | pkg/util/ |

### 1.3 目标 Monorepo 结构

```
my-internal-platform/               # Git Root
├── go.work                         # Go Workspace
├── go.work.sum
├── pkg/                            # 公共库（两个应用共享）
│   ├── go.mod                      # module github.com/xxx/platform/pkg
│   ├── n9e/                        # N9E Client
│   ├── vm/                         # VM Client
│   ├── model/                      # 通用模型
│   └── util/                       # 工具函数
├── apps/
│   ├── inspect-cli/                # 巡检工具 CLI
│   │   ├── go.mod
│   │   ├── main.go
│   │   ├── cmd/
│   │   └── internal/               # CLI 专用代码
│   └── cmdb-server/                # CMDB 后端（后续开发）
│       ├── go.mod
│       └── ...
├── web/                            # Vue 前端（后续开发）
├── configs/                        # 配置文件
├── docs/                           # 文档
└── Makefile
```

---

## 2. 重构原则

### 2.1 核心原则

1. **渐进式重构**：每步可验证，确保巡检功能不受影响
2. **最小改动**：优先移动文件，避免大规模代码修改
3. **向后兼容**：保持 CLI 命令和配置格式不变
4. **测试驱动**：每步完成后运行测试验证

### 2.2 Go Workspace 策略

使用 Go 1.18+ 的 Workspace 特性：
- 本地开发时，go.work 自动解析模块依赖
- 发布时，各模块独立版本管理
- 无需 replace 指令污染 go.mod
