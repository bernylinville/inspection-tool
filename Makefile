# 系统巡检工具 - Makefile
# ===========================
# 版本信息（通过 -ldflags 注入到 cmd 包）
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
CMD_PKG := inspection-tool/cmd/inspect/cmd
LDFLAGS := -X $(CMD_PKG).Version=$(VERSION) -X $(CMD_PKG).BuildTime=$(BUILD_TIME) -X $(CMD_PKG).GitCommit=$(GIT_COMMIT)

# 构建目标配置
BINARY_NAME := inspect
BUILD_DIR := bin
COVERAGE_DIR := coverage

# Go 工具链参数
GO := go
GOTEST := $(GO) test
GOBUILD := $(GO) build

# 工具链定义 (锁定版本以保证团队一致性)
# 假设 $GOPATH/bin 在系统 PATH 中
GOLANGCI_LINT_VER := v1.63.4
MODERNIZE_PKG := golang.org/x/tools/go/analysis/passes/modernize/cmd/modernize@latest
MODERNIZE_BIN := modernize

# 全局默认禁用 CGO，生成纯静态二进制文件
# 注意：test 目标会显式覆盖此设置以支持 -race
export CGO_ENABLED=0

.PHONY: all build build-all test lint clean coverage help modernize analyze fix check deps tools

# 默认目标
all: build

# 依赖管理
deps:
	@echo "==> 整理依赖..."
	$(GO) mod tidy
	$(GO) mod verify

# 安装构建工具 (安装到 $GOPATH/bin，提升后续运行速度)
tools:
	@echo "==> 安装构建工具..."
	$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VER)
	$(GO) install $(MODERNIZE_PKG)

# 构建本地二进制
build: deps
	@echo "==> 构建本地二进制..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -trimpath -ldflags "-s -w $(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/inspect
	@echo "==> 构建完成: $(BUILD_DIR)/$(BINARY_NAME)"

# 交叉编译多平台
build-all: deps
	@echo "==> 交叉编译多平台..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GOBUILD) -trimpath -ldflags "-s -w $(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/inspect
	GOOS=darwin GOARCH=amd64 $(GOBUILD) -trimpath -ldflags "-s -w $(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/inspect
	GOOS=darwin GOARCH=arm64 $(GOBUILD) -trimpath -ldflags "-s -w $(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/inspect
	GOOS=windows GOARCH=amd64 $(GOBUILD) -trimpath -ldflags "-s -w $(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/inspect
	@echo "==> 交叉编译完成:"
	@ls -lh $(BUILD_DIR)/

# 运行测试（关键修复: 显式开启 CGO 以支持 -race）
test:
	@echo "==> 运行测试 (Race Detector Enabled)..."
	CGO_ENABLED=1 $(GOTEST) -v -race ./...

# 代码检查（需要安装 golangci-lint）
lint: analyze
	@echo "==> 运行代码检查..."
	golangci-lint run ./...

# 清理构建产物
clean:
	@echo "==> 清理构建产物..."
	rm -rf $(BUILD_DIR)
	rm -rf $(COVERAGE_DIR)
	rm -f *.xlsx *.html
	$(GO) clean -cache -testcache
	@echo "==> 清理完成"

# 生成测试覆盖率报告
coverage:
	@echo "==> 生成测试覆盖率报告..."
	@mkdir -p $(COVERAGE_DIR)
	$(GOTEST) -coverprofile=$(COVERAGE_DIR)/coverage.out ./...
	$(GO) tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html
	@echo "==> 覆盖率摘要:"
	$(GO) tool cover -func=$(COVERAGE_DIR)/coverage.out | tail -1
	@echo "==> 报告已生成: $(COVERAGE_DIR)/coverage.html"

# ===========================
# Go 代码现代化
# ===========================
# 分析代码，显示可改进的地方 (优先使用本地二进制，避免联网)
analyze:
	@echo "==> 分析代码现代化建议..."
	@if command -v $(MODERNIZE_BIN) >/dev/null 2>&1; then \
		$(MODERNIZE_BIN) ./... || true; \
	else \
		echo "Warning: modernize binary not found in PATH, using go run (slower)..."; \
		$(GO) run $(MODERNIZE_PKG) ./... || true; \
	fi

# 自动修复代码 (优先使用本地二进制，避免联网)
modernize:
	@echo "==> 应用代码现代化..."
	@if command -v $(MODERNIZE_BIN) >/dev/null 2>&1; then \
		$(MODERNIZE_BIN) -fix ./...; \
	else \
		echo "Warning: modernize binary not found in PATH, using go run (slower)..."; \
		$(GO) run $(MODERNIZE_PKG) -fix ./...; \
	fi
	@echo "==> 格式化代码..."
	$(GO) fmt ./...
	@echo "✓ 现代化完成"

# 完整修复流程：现代化 + 测试
fix: modernize test
	@echo "✓ 修复完成，测试通过"

# CI 检查：lint + test
check: lint test
	@echo "✓ 所有检查通过"

# 帮助信息
help:
	@echo "系统巡检工具 - 可用目标:"
	@echo ""
	@echo "  构建:"
	@echo "    build      - 构建本地二进制文件（纯静态，-trimpath -s -w）"
	@echo "    build-all  - 交叉编译多平台（linux/darwin/windows）"
	@echo ""
	@echo "  依赖:"
	@echo "    deps       - 整理依赖（go mod tidy + verify）"
	@echo "    tools      - 安装构建工具（锁定版本）"
	@echo ""
	@echo "  测试:"
	@echo "    test       - 运行测试（带竞态检测，自动开启 CGO）"
	@echo "    coverage   - 生成测试覆盖率报告"
	@echo ""
	@echo "  代码质量:"
	@echo "    lint       - 运行代码检查（golangci-lint + modernize）"
	@echo "    analyze    - 仅分析代码现代化建议（优先使用本地缓存）"
	@echo "    modernize  - 自动应用代码现代化（优先使用本地缓存）"
	@echo "    fix        - 现代化 + 测试（推荐）"
	@echo "    check      - CI 检查（lint + test）"
	@echo ""
	@echo "  其他:"
	@echo "    clean      - 清理构建产物、缓存及报告文件"
	@echo "    help       - 显示此帮助信息"
	@echo ""
	@echo "推荐工作流:"
	@echo "  make tools       # 首次运行安装工具"
	@echo "  make deps        # 整理依赖"
	@echo "  make fix         # 自动修复 + 测试"
