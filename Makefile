# GoPM2 Makefile

# 变量定义
BINARY_NAME=gopm2
BINARY_WINDOWS=$(BINARY_NAME).exe
GO_FILES=$(shell find . -name "*.go" -not -path "./examples/*")

# 默认目标
.PHONY: all
all: build

# 构建
.PHONY: build
build:
	@echo "构建 $(BINARY_NAME)..."
	go build -o $(BINARY_NAME) .

# Windows构建
.PHONY: build-windows
build-windows:
	@echo "构建 Windows 版本..."
	GOOS=windows GOARCH=amd64 go build -o $(BINARY_WINDOWS) .

# 跨平台构建 (Linux/macOS)
.PHONY: build-all
build-all:
	@echo "构建所有平台版本..."
	GOOS=linux GOARCH=amd64 go build -o $(BINARY_NAME)-linux .
	GOOS=darwin GOARCH=amd64 go build -o $(BINARY_NAME)-darwin .
	GOOS=windows GOARCH=amd64 go build -o $(BINARY_WINDOWS) .

# Windows PowerShell 跨平台构建
.PHONY: build-all-ps
build-all-ps:
	@echo "Windows PowerShell 构建所有平台版本..."
	powershell -Command "$$env:GOOS='linux'; $$env:GOARCH='amd64'; go build -o $(BINARY_NAME)-linux ."
	powershell -Command "$$env:GOOS='darwin'; $$env:GOARCH='amd64'; go build -o $(BINARY_NAME)-darwin ."  
	powershell -Command "$$env:GOOS='windows'; $$env:GOARCH='amd64'; go build -o $(BINARY_WINDOWS) ."

# 清理
.PHONY: clean
clean:
	@echo "清理构建文件..."
	rm -f $(BINARY_NAME) $(BINARY_WINDOWS)
	rm -f $(BINARY_NAME)-linux $(BINARY_NAME)-darwin
	rm -f *.log
	rm -rf logs/
	rm -rf temp/
	rm -rf tmp/

# 安装依赖
.PHONY: deps
deps:
	@echo "安装依赖..."
	go mod tidy
	go mod download

# 运行测试
.PHONY: test
test:
	@echo "运行测试..."
	go test -v ./...

# 代码格式化
.PHONY: fmt
fmt:
	@echo "格式化代码..."
	go fmt ./...

# 代码检查
.PHONY: vet
vet:
	@echo "代码检查..."
	go vet ./...

# 安装
.PHONY: install
install: build
	@echo "安装到 GOPATH/bin..."
	go install .

# 运行示例
.PHONY: example
example: build
	@echo "运行示例配置..."
	./$(BINARY_NAME) start examples/ecosystem.config.json

# 查看版本信息
.PHONY: version
version:
	@echo "Go 版本: $(shell go version)"
	@echo "项目信息:"
	@echo "  二进制文件: $(BINARY_NAME)"
	@echo "  源文件数量: $(shell echo $(GO_FILES) | wc -w)"

# 帮助信息
.PHONY: help
help:
	@echo "可用命令:"
	@echo "  build        - 构建项目"
	@echo "  build-windows - 构建Windows版本"
	@echo "  build-all    - 构建所有平台版本"
	@echo "  clean        - 清理构建文件"
	@echo "  deps         - 安装依赖"
	@echo "  test         - 运行测试"
	@echo "  fmt          - 格式化代码"
	@echo "  vet          - 代码检查"
	@echo "  install      - 安装到GOPATH/bin"
	@echo "  example      - 运行示例配置"
	@echo "  version      - 显示版本信息"
	@echo "  help         - 显示此帮助信息" 