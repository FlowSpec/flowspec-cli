#!/bin/bash

# FlowSpec CLI 开发环境设置脚本

set -e

echo "🚀 FlowSpec CLI 开发环境设置"
echo "=============================="

# 检查 Go 版本
echo "📋 检查 Go 版本..."
if ! command -v go &> /dev/null; then
    echo "❌ Go 未安装，请先安装 Go 1.21 或更高版本"
    echo "💡 安装指南: https://golang.org/doc/install"
    exit 1
fi

GO_VERSION=$(go version | cut -d' ' -f3 | sed 's/go//')
REQUIRED_VERSION="1.21"

if [ "$(printf '%s\n' "$REQUIRED_VERSION" "$GO_VERSION" | sort -V | head -n1)" != "$REQUIRED_VERSION" ]; then
    echo "❌ Go 版本过低: $GO_VERSION，需要 $REQUIRED_VERSION 或更高版本"
    exit 1
fi

echo "✅ Go 版本: $GO_VERSION"

# 检查 Make
echo "📋 检查 Make..."
if ! command -v make &> /dev/null; then
    echo "⚠️  Make 未安装，某些构建脚本可能无法使用"
    echo "💡 macOS: brew install make"
    echo "💡 Ubuntu: sudo apt-get install make"
else
    echo "✅ Make 可用"
fi

# 检查 Git
echo "📋 检查 Git..."
if ! command -v git &> /dev/null; then
    echo "❌ Git 未安装，请先安装 Git"
    exit 1
fi
echo "✅ Git 可用"

# 安装 golangci-lint
echo "📋 检查 golangci-lint..."
if ! command -v golangci-lint &> /dev/null; then
    echo "⚠️  golangci-lint 未安装，正在安装..."
    
    # 检测操作系统
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)
    
    case $ARCH in
        x86_64) ARCH="amd64" ;;
        arm64|aarch64) ARCH="arm64" ;;
        *) echo "❌ 不支持的架构: $ARCH"; exit 1 ;;
    esac
    
    if [[ "$OS" == "darwin" ]]; then
        if command -v brew &> /dev/null; then
            echo "使用 Homebrew 安装 golangci-lint..."
            brew install golangci-lint
        else
            echo "使用 curl 安装 golangci-lint..."
            curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.54.2
        fi
    elif [[ "$OS" == "linux" ]]; then
        echo "使用 curl 安装 golangci-lint..."
        curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.54.2
    else
        echo "⚠️  请手动安装 golangci-lint: https://golangci-lint.run/usage/install/"
    fi
else
    echo "✅ golangci-lint 可用"
fi

# 下载依赖
echo "📦 下载 Go 模块依赖..."
go mod download
go mod tidy
echo "✅ 依赖下载完成"

# 运行初始构建
echo "🔨 运行初始构建..."
if command -v make &> /dev/null; then
    make build
else
    echo "使用 go build 构建..."
    mkdir -p build
    go build -o build/flowspec-cli ./cmd/flowspec-cli
fi
echo "✅ 构建完成"

# 运行测试
echo "🧪 运行测试..."
if command -v make &> /dev/null; then
    make test
else
    go test ./...
fi
echo "✅ 测试通过"

# 设置 Git hooks (可选)
echo "🔧 设置 Git hooks..."
if [ -d ".git" ]; then
    cat > .git/hooks/pre-commit << 'EOF'
#!/bin/bash
# FlowSpec CLI pre-commit hook

echo "🔍 运行 pre-commit 检查..."

# 格式化代码
echo "📝 格式化代码..."
make fmt

# 运行代码检查
echo "🔍 运行代码检查..."
make vet

# 运行测试
echo "🧪 运行测试..."
make test

echo "✅ Pre-commit 检查通过"
EOF
    chmod +x .git/hooks/pre-commit
    echo "✅ Git pre-commit hook 设置完成"
else
    echo "⚠️  不是 Git 仓库，跳过 Git hooks 设置"
fi

# 显示开发信息
echo ""
echo "🎉 开发环境设置完成！"
echo "====================="
echo ""
echo "📋 可用的开发命令:"
echo "  make build      - 构建项目"
echo "  make test       - 运行测试"
echo "  make coverage   - 生成覆盖率报告"
echo "  make lint       - 运行代码检查"
echo "  make ci         - 运行完整 CI 检查"
echo "  make help       - 显示所有可用命令"
echo ""
echo "🚀 快速开始:"
echo "  ./build/flowspec-cli --help"
echo "  ./build/flowspec-cli align --path=./examples/simple-user-service/src --trace=./examples/simple-user-service/traces/success-scenario.json"
echo ""
echo "📖 文档:"
echo "  README.md           - 项目介绍"
echo "  CONTRIBUTING.md     - 贡献指南"
echo "  docs/API.md         - API 文档"
echo "  docs/ARCHITECTURE.md - 架构文档"
echo ""
echo "💡 提示:"
echo "  - 使用 'make help' 查看所有可用命令"
echo "  - 查看 examples/ 目录了解使用示例"
echo "  - 提交代码前会自动运行 pre-commit 检查"
echo ""
echo "Happy coding! 🎯"