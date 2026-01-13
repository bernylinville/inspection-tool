# 项目结构调整实施计划

> 版本：v1.0
> 创建日期：2026-01-12
> 目标：将 inspection-tool 重构为 Monorepo 结构，支持巡检功能和 CMDB 平台共存

---

## 1. 概述

### 1.1 当前状态

```
inspection-tool/                    # 单模块 Go 项目
├── cmd/inspect/                    # CLI 入口
├── internal/                       # 全部业务代码
│   ├── client/n9e/                 # N9E API 客户端
│   ├── client/vm/                  # VictoriaMetrics 客户端
│   ├── config/                     # 配置管理
│   ├── model/                      # 数据模型
│   ├── service/                    # 巡检业务逻辑
│   ├── report/                     # 报告生成
│   └── util/                       # 工具函数
├── configs/                        # 配置文件
├── templates/                      # HTML 模板
├── go.mod                          # module inspection-tool
└── Makefile
```

### 1.2 目标状态

```
inspection-tool/                    # Monorepo（Git Root 保持不变）
├── go.work                         # Go Workspace
├── go.work.sum
│
├── pkg/                            # 公共库（可被多应用引用）
│   ├── go.mod                      # module inspection-tool/pkg
│   ├── n9e/                        # N9E Client
│   ├── vm/                         # VM Client
│   └── model/                      # 通用模型（Host, Alert 等）
│
├── apps/
│   ├── inspect-cli/                # 巡检工具 CLI
│   │   ├── go.mod                  # module inspection-tool/apps/inspect-cli
│   │   ├── main.go
│   │   ├── cmd/                    # Cobra 命令
│   │   └── internal/               # CLI 专用代码
│   │       ├── config/             # 配置加载
│   │       ├── service/            # 巡检服务
│   │       └── report/             # 报告生成
│   │
│   └── cmdb-server/                # CMDB 后端（后续开发）
│       ├── go.mod                  # module inspection-tool/apps/cmdb-server
│       └── internal/
│
├── web/                            # Vue 前端（后续开发）
│
├── configs/                        # 共享配置文件
├── templates/                      # 共享模板
├── migrations/                     # 数据库迁移（CMDB 用）
├── scripts/                        # 脚本工具
└── Makefile                        # 统一构建脚本
```

### 1.3 核心原则

| 原则 | 说明 |
|------|------|
| **渐进式重构** | 每个步骤后验证功能完整性 |
| **最小改动** | 优先移动文件，保持代码逻辑不变 |
| **保持兼容** | 重构期间 CLI 功能不中断 |
| **先 pkg 后 apps** | 先抽取公共库，再迁移应用代码 |

---

## 2. 阶段概览

| 阶段 | 描述 | 预计时间 | 验证点 |
|------|------|----------|--------|
| **Phase 1** | 准备工作与备份 | 0.5h | Git 状态干净 |
| **Phase 2** | 创建 Monorepo 目录结构 | 0.5h | 目录存在 |
| **Phase 3** | 抽取 pkg/ 公共库 | 2h | `go build ./pkg/...` |
| **Phase 4** | 迁移 apps/inspect-cli | 3h | `make build` + 测试通过 |
| **Phase 5** | 配置 Go Workspace | 0.5h | `go work sync` |
| **Phase 6** | 更新构建系统 | 1h | 完整构建验证 |
| **Phase 7** | 最终验证 | 1h | 全量测试 + 巡检功能验证 |
| **Phase 8** | CMDB 骨架搭建 | 1h | 后端项目初始化 |

**总预计时间**: 9-10 小时

---

## 3. 详细实施步骤

### Phase 1: 准备工作与备份

#### Step 1.1: 确保代码库干净
```bash
git status
git add -A && git commit -m "chore: 重构前状态保存"
```

#### Step 1.2: 创建重构分支
```bash
git checkout -b refactor/monorepo-structure
```

#### Step 1.3: 运行完整测试确认基线
```bash
make test
```

#### Step 1.4: 记录当前构建产物
```bash
make build
./bin/inspect version
```

**验证检查点**:
- [ ] Git 状态干净
- [ ] 所有测试通过
- [ ] 可正常构建

---

### Phase 2: 创建 Monorepo 目录结构

#### Step 2.1: 创建 pkg/ 目录
```bash
mkdir -p pkg
```

#### Step 2.2: 创建 apps/ 目录
```bash
mkdir -p apps/inspect-cli
mkdir -p apps/cmdb-server
```

#### Step 2.3: 创建 web/ 占位目录
```bash
mkdir -p web
touch web/.gitkeep
```

#### Step 2.4: 创建 migrations/ 占位目录
```bash
mkdir -p migrations
touch migrations/.gitkeep
```

#### Step 2.5: 创建 scripts/ 目录（如不存在）
```bash
mkdir -p scripts
```

**验证检查点**:
- [ ] `pkg/` 目录存在
- [ ] `apps/inspect-cli/` 目录存在
- [ ] `apps/cmdb-server/` 目录存在

---

### Phase 3: 抽取 pkg/ 公共库

#### Step 3.1: 创建 pkg/go.mod
```bash
cat > pkg/go.mod << 'GOMOD'
module inspection-tool/pkg

go 1.25.5

require (
    github.com/go-resty/resty/v2 v2.17.0
    github.com/rs/zerolog v1.34.0
)
GOMOD
```

#### Step 3.2: 移动 N9E Client 到 pkg/
```bash
cp -r internal/client/n9e pkg/n9e
```

#### Step 3.3: 更新 pkg/n9e 的 import 路径
编辑 `pkg/n9e/client.go` 和 `pkg/n9e/types.go`:
- 将 `inspection-tool/internal/model` 改为 `inspection-tool/pkg/model`

#### Step 3.4: 移动 VM Client 到 pkg/
```bash
cp -r internal/client/vm pkg/vm
```

#### Step 3.5: 更新 pkg/vm 的 import 路径
编辑 `pkg/vm/client.go` 和 `pkg/vm/types.go`:
- 更新 import 路径指向 `inspection-tool/pkg/model`

#### Step 3.6: 抽取通用 model 到 pkg/
```bash
mkdir -p pkg/model
cp internal/model/host.go pkg/model/
cp internal/model/metric.go pkg/model/
cp internal/model/alert.go pkg/model/
```

#### Step 3.7: 运行 pkg/ 编译验证
```bash
cd pkg && go mod tidy && go build ./...
cd ..
```

**验证检查点**:
- [ ] `pkg/go.mod` 存在
- [ ] `pkg/n9e/` 代码存在
- [ ] `pkg/vm/` 代码存在
- [ ] `pkg/model/` 包含通用模型
- [ ] `cd pkg && go build ./...` 成功

---

### Phase 4: 迁移 apps/inspect-cli

#### Step 4.1: 创建 apps/inspect-cli/go.mod
```bash
cat > apps/inspect-cli/go.mod << 'GOMOD'
module inspection-tool/apps/inspect-cli

go 1.25.5

require (
    inspection-tool/pkg v0.0.0
    github.com/go-playground/validator/v10 v10.29.0
    github.com/go-resty/resty/v2 v2.17.0
    github.com/rs/zerolog v1.34.0
    github.com/spf13/cobra v1.10.2
    github.com/spf13/viper v1.21.0
    github.com/xuri/excelize/v2 v2.10.0
    golang.org/x/sync v0.19.0
    gopkg.in/yaml.v3 v3.0.1
)

replace inspection-tool/pkg => ../../pkg
GOMOD
```

#### Step 4.2: 移动 main.go 入口
```bash
cp cmd/inspect/main.go apps/inspect-cli/main.go
```

#### Step 4.3: 移动 cmd/ 命令定义
```bash
cp -r cmd/inspect/cmd apps/inspect-cli/cmd
```

#### Step 4.4: 创建 apps/inspect-cli/internal/
```bash
mkdir -p apps/inspect-cli/internal
```

#### Step 4.5: 移动 config 到 CLI internal
```bash
cp -r internal/config apps/inspect-cli/internal/config
```

#### Step 4.6: 移动 service 到 CLI internal
```bash
cp -r internal/service apps/inspect-cli/internal/service
```

#### Step 4.7: 移动 report 到 CLI internal
```bash
cp -r internal/report apps/inspect-cli/internal/report
```

#### Step 4.8: 移动 util 到 CLI internal
```bash
cp -r internal/util apps/inspect-cli/internal/util
```

#### Step 4.9: 移动巡检专用 model
```bash
mkdir -p apps/inspect-cli/internal/model
cp internal/model/mysql.go apps/inspect-cli/internal/model/
cp internal/model/redis.go apps/inspect-cli/internal/model/
cp internal/model/nginx.go apps/inspect-cli/internal/model/
cp internal/model/tomcat.go apps/inspect-cli/internal/model/
cp internal/model/tomcat_metric.go apps/inspect-cli/internal/model/
cp internal/model/elasticsearch.go apps/inspect-cli/internal/model/
cp internal/model/inspection.go apps/inspect-cli/internal/model/
cp internal/model/*_test.go apps/inspect-cli/internal/model/
```

#### Step 4.10: 批量更新 import 路径
在 `apps/inspect-cli/` 下所有 `.go` 文件中:
- `inspection-tool/internal/client/n9e` → `inspection-tool/pkg/n9e`
- `inspection-tool/internal/client/vm` → `inspection-tool/pkg/vm`
- `inspection-tool/internal/model` → 分两种情况:
  - 通用模型 (Host, Alert, Metric) → `inspection-tool/pkg/model`
  - 巡检专用模型 → `inspection-tool/apps/inspect-cli/internal/model`
- `inspection-tool/internal/config` → `inspection-tool/apps/inspect-cli/internal/config`
- `inspection-tool/internal/service` → `inspection-tool/apps/inspect-cli/internal/service`
- `inspection-tool/internal/report` → `inspection-tool/apps/inspect-cli/internal/report`
- `inspection-tool/cmd/inspect/cmd` → `inspection-tool/apps/inspect-cli/cmd`

#### Step 4.11: 更新 main.go import 路径
编辑 `apps/inspect-cli/main.go`:
```go
import (
    "inspection-tool/apps/inspect-cli/cmd"
)
```

#### Step 4.12: 运行 go mod tidy
```bash
cd apps/inspect-cli && go mod tidy
cd ../..
```

#### Step 4.13: 编译验证
```bash
cd apps/inspect-cli && go build -o ../../bin/inspect .
cd ../..
```

#### Step 4.14: 运行测试
```bash
cd apps/inspect-cli && go test -v ./...
cd ../..
```

**验证检查点**:
- [ ] `apps/inspect-cli/go.mod` 存在
- [ ] `apps/inspect-cli/main.go` 存在
- [ ] 编译成功，生成 `bin/inspect`
- [ ] 所有测试通过

---

### Phase 5: 配置 Go Workspace

#### Step 5.1: 创建 go.work 文件
```bash
cat > go.work << 'GOWORK'
go 1.25.5

use (
    ./pkg
    ./apps/inspect-cli
    ./apps/cmdb-server
)
GOWORK
```

#### Step 5.2: 同步 workspace
```bash
go work sync
```

#### Step 5.3: 验证 workspace
```bash
go build ./pkg/...
go build ./apps/inspect-cli/...
```

**验证检查点**:
- [ ] `go.work` 存在
- [ ] `go work sync` 无错误
- [ ] workspace 下所有模块可构建

---

### Phase 6: 更新构建系统

#### Step 6.1: 更新 Makefile
创建新的 Makefile，支持 Monorepo 结构（见附录 A）

#### Step 6.2: 验证构建
```bash
make clean
make build
./bin/inspect version
```

**验证检查点**:
- [ ] `make build` 成功
- [ ] `./bin/inspect version` 显示正确版本

---

### Phase 7: 最终验证

#### Step 7.1: 运行完整测试套件
```bash
make test
```

#### Step 7.2: 验证巡检功能
```bash
./bin/inspect validate -c config.yaml
./bin/inspect run -c config.yaml --dry-run
```

#### Step 7.3: 清理旧文件（确认无误后）
```bash
rm -rf cmd/
rm -rf internal/
rm go.mod go.sum
```

#### Step 7.4: 提交重构
```bash
git add -A
git commit -m "refactor: 重构为 Monorepo 结构，支持多应用"
```

**验证检查点**:
- [ ] 所有测试通过
- [ ] CLI 功能正常
- [ ] 旧文件已清理
- [ ] Git 提交成功

---

### Phase 8: CMDB 骨架搭建

#### Step 8.1: 创建 cmdb-server go.mod
```bash
cat > apps/cmdb-server/go.mod << 'GOMOD'
module inspection-tool/apps/cmdb-server

go 1.25.5

require (
    inspection-tool/pkg v0.0.0
    github.com/gin-gonic/gin v1.11.0
    gorm.io/gorm v1.31.1
    gorm.io/driver/postgres v1.6.0
)

replace inspection-tool/pkg => ../../pkg
GOMOD
```

#### Step 8.2: 创建 cmdb-server main.go 骨架
```bash
cat > apps/cmdb-server/main.go << 'GOMAIN'
package main

import "fmt"

func main() {
    fmt.Println("CMDB Server - Coming Soon")
}
GOMAIN
```

#### Step 8.3: 创建 cmdb-server internal 目录结构
```bash
mkdir -p apps/cmdb-server/internal/{handler,service,repository,proxy}
```

#### Step 8.4: 同步 workspace
```bash
go work sync
```

#### Step 8.5: 验证 cmdb-server 可编译
```bash
cd apps/cmdb-server && go build .
cd ../..
```

**验证检查点**:
- [ ] `apps/cmdb-server/go.mod` 存在
- [ ] `apps/cmdb-server/main.go` 存在
- [ ] cmdb-server 可编译

---

## 4. 文件移动清单

### 4.1 移动到 pkg/（公共库）

| 源路径 | 目标路径 | 说明 |
|--------|----------|------|
| `internal/client/n9e/*` | `pkg/n9e/*` | N9E API 客户端 |
| `internal/client/vm/*` | `pkg/vm/*` | VM API 客户端 |
| `internal/model/host.go` | `pkg/model/host.go` | 主机通用模型 |
| `internal/model/metric.go` | `pkg/model/metric.go` | 指标通用模型 |
| `internal/model/alert.go` | `pkg/model/alert.go` | 告警通用模型 |

### 4.2 移动到 apps/inspect-cli/（CLI 专用）

| 源路径 | 目标路径 | 说明 |
|--------|----------|------|
| `cmd/inspect/main.go` | `apps/inspect-cli/main.go` | 程序入口 |
| `cmd/inspect/cmd/*` | `apps/inspect-cli/cmd/*` | Cobra 命令 |
| `internal/config/*` | `apps/inspect-cli/internal/config/*` | 配置管理 |
| `internal/service/*` | `apps/inspect-cli/internal/service/*` | 巡检服务 |
| `internal/report/*` | `apps/inspect-cli/internal/report/*` | 报告生成 |
| `internal/util/*` | `apps/inspect-cli/internal/util/*` | 工具函数 |
| `internal/model/mysql.go` | `apps/inspect-cli/internal/model/mysql.go` | MySQL 模型 |
| `internal/model/redis.go` | `apps/inspect-cli/internal/model/redis.go` | Redis 模型 |
| `internal/model/nginx.go` | `apps/inspect-cli/internal/model/nginx.go` | Nginx 模型 |
| `internal/model/tomcat*.go` | `apps/inspect-cli/internal/model/tomcat*.go` | Tomcat 模型 |
| `internal/model/elasticsearch.go` | `apps/inspect-cli/internal/model/elasticsearch.go` | ES 模型 |
| `internal/model/inspection.go` | `apps/inspect-cli/internal/model/inspection.go` | 巡检结果模型 |

### 4.3 保持原位置（共享资源）

| 路径 | 说明 |
|------|------|
| `configs/*` | 配置文件 |
| `templates/*` | HTML 模板 |
| `Makefile` | 构建脚本（需更新） |
| `README.md` | 项目文档 |

---

## 5. Import 路径映射

### 5.1 需要替换的 import 路径

| 旧路径 | 新路径 |
|--------|--------|
| `inspection-tool/internal/client/n9e` | `inspection-tool/pkg/n9e` |
| `inspection-tool/internal/client/vm` | `inspection-tool/pkg/vm` |
| `inspection-tool/internal/config` | `inspection-tool/apps/inspect-cli/internal/config` |
| `inspection-tool/internal/service` | `inspection-tool/apps/inspect-cli/internal/service` |
| `inspection-tool/internal/report` | `inspection-tool/apps/inspect-cli/internal/report` |
| `inspection-tool/internal/util` | `inspection-tool/apps/inspect-cli/internal/util` |
| `inspection-tool/cmd/inspect/cmd` | `inspection-tool/apps/inspect-cli/cmd` |

### 5.2 model 包特殊处理

`inspection-tool/internal/model` 需要拆分:
- **通用模型** → `inspection-tool/pkg/model`
  - Host, HostMeta, DiskMountInfo
  - MetricDefinition, MetricValue, HostMetrics
  - Alert, AlertLevel, AlertSummary
- **巡检专用模型** → `inspection-tool/apps/inspect-cli/internal/model`
  - MySQL*, Redis*, Nginx*, Tomcat*, Elasticsearch*
  - InspectionResult, InspectionSummary, HostResult

---

## 6. 回滚方案

如果重构过程中出现问题：

```bash
# 方案 1: 回退到重构前的提交
git checkout main

# 方案 2: 重置当前分支
git reset --hard HEAD~1

# 方案 3: 删除重构分支重新开始
git checkout main
git branch -D refactor/monorepo-structure
git checkout -b refactor/monorepo-structure
```

---

## 7. 验证清单

### 重构完成后必须验证

- [ ] `go work sync` 无错误
- [ ] `make build` 成功
- [ ] `make test` 所有测试通过
- [ ] `./bin/inspect version` 显示正确版本
- [ ] `./bin/inspect validate -c config.yaml` 配置验证通过
- [ ] 巡检功能正常（如有测试环境）
- [ ] Git 状态干净，无遗漏文件

### 代码质量验证

- [ ] `make lint` 无错误
- [ ] `make coverage` 覆盖率保持 85%+
- [ ] 无循环依赖警告

---

## 8. 附录 A: 新 Makefile

```makefile
# 系统巡检工具 - Makefile (Monorepo 版)

VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

CLI_PKG := inspection-tool/apps/inspect-cli/cmd
CLI_LDFLAGS := -X $(CLI_PKG).Version=$(VERSION) -X $(CLI_PKG).BuildTime=$(BUILD_TIME) -X $(CLI_PKG).GitCommit=$(GIT_COMMIT)

BINARY_NAME := inspect
BUILD_DIR := bin
COVERAGE_DIR := coverage

GO := go
GOTEST := $(GO) test
GOBUILD := $(GO) build

.PHONY: all build build-cli build-all test test-pkg test-cli lint clean coverage help

all: build

build: build-cli

build-cli:
	@echo "==> 构建 CLI 工具..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -ldflags "$(CLI_LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) ./apps/inspect-cli
	@echo "==> 构建完成: $(BUILD_DIR)/$(BINARY_NAME)"

build-all:
	@echo "==> 交叉编译多平台..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GOBUILD) -ldflags "$(CLI_LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./apps/inspect-cli
	GOOS=darwin GOARCH=amd64 $(GOBUILD) -ldflags "$(CLI_LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./apps/inspect-cli
	GOOS=darwin GOARCH=arm64 $(GOBUILD) -ldflags "$(CLI_LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./apps/inspect-cli
	GOOS=windows GOARCH=amd64 $(GOBUILD) -ldflags "$(CLI_LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./apps/inspect-cli

test: test-pkg test-cli

test-pkg:
	@echo "==> 测试 pkg/..."
	$(GOTEST) -v -race ./pkg/...

test-cli:
	@echo "==> 测试 CLI..."
	$(GOTEST) -v -race ./apps/inspect-cli/...

lint:
	@echo "==> 运行代码检查..."
	golangci-lint run ./pkg/... ./apps/inspect-cli/...

clean:
	@echo "==> 清理构建产物..."
	rm -rf $(BUILD_DIR)
	rm -rf $(COVERAGE_DIR)

coverage:
	@echo "==> 生成测试覆盖率报告..."
	@mkdir -p $(COVERAGE_DIR)
	$(GOTEST) -coverprofile=$(COVERAGE_DIR)/coverage.out ./pkg/... ./apps/inspect-cli/...
	$(GO) tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html

help:
	@echo "可用目标:"
	@echo "  build      - 构建 CLI 工具"
	@echo "  build-all  - 交叉编译多平台"
	@echo "  test       - 运行所有测试"
	@echo "  test-pkg   - 仅测试 pkg/"
	@echo "  test-cli   - 仅测试 CLI"
	@echo "  lint       - 运行代码检查"
	@echo "  clean      - 清理构建产物"
	@echo "  coverage   - 生成覆盖率报告"
```
