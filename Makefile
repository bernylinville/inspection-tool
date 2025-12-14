# 系统巡检工具 - Makefile
# ===========================

# 版本信息（通过 -ldflags 注入到 cmd 包）
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
CMD_PKG := inspection-tool/cmd/inspect/cmd
LDFLAGS := -X $(CMD_PKG).Version=$(VERSION) -X $(CMD_PKG).BuildTime=$(BUILD_TIME) -X $(CMD_PKG).GitCommit=$(GIT_COMMIT)

# 构建目标
BINARY_NAME := inspect
BUILD_DIR := bin
COVERAGE_DIR := coverage

# Go 参数
GO := go
GOTEST := $(GO) test
GOBUILD := $(GO) build

.PHONY: all build build-all test lint clean coverage help

# 默认目标
all: build

# 构建本地二进制
build:
	@echo "==> 构建本地二进制..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/inspect
	@echo "==> 构建完成: $(BUILD_DIR)/$(BINARY_NAME)"

# 交叉编译多平台
build-all:
	@echo "==> 交叉编译多平台..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GOBUILD) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/inspect
	GOOS=darwin GOARCH=amd64 $(GOBUILD) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/inspect
	GOOS=darwin GOARCH=arm64 $(GOBUILD) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/inspect
	GOOS=windows GOARCH=amd64 $(GOBUILD) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/inspect
	@echo "==> 交叉编译完成:"
	@ls -lh $(BUILD_DIR)/

# 运行测试（带竞态检测）
test:
	@echo "==> 运行测试..."
	$(GOTEST) -v -race ./...

# 代码检查（需要安装 golangci-lint）
lint:
	@echo "==> 运行代码检查..."
	golangci-lint run ./...

# 清理构建产物
clean:
	@echo "==> 清理构建产物..."
	rm -rf $(BUILD_DIR)
	rm -rf $(COVERAGE_DIR)
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

# 帮助信息
help:
	@echo "系统巡检工具 - 可用目标:"
	@echo ""
	@echo "  build      - 构建本地二进制文件"
	@echo "  build-all  - 交叉编译多平台（linux/darwin/windows）"
	@echo "  test       - 运行测试（带竞态检测）"
	@echo "  lint       - 运行代码检查（需要 golangci-lint）"
	@echo "  clean      - 清理构建产物"
	@echo "  coverage   - 生成测试覆盖率报告"
	@echo "  help       - 显示此帮助信息"
	@echo ""
	@echo "示例:"
	@echo "  make build              # 构建本地二进制"
	@echo "  make test               # 运行所有测试"
	@echo "  make coverage           # 生成覆盖率报告"
