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
