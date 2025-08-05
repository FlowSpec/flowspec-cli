#!/bin/bash

# FlowSpec CLI 最终集成测试和验收脚本

set -e

echo "🔍 FlowSpec CLI 最终集成测试和验收"
echo "================================="

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 测试结果统计
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# 日志函数
log_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

log_success() {
    echo -e "${GREEN}✅ $1${NC}"
    ((PASSED_TESTS++))
}

log_error() {
    echo -e "${RED}❌ $1${NC}"
    ((FAILED_TESTS++))
}

log_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

# 运行测试并检查结果
run_test() {
    local test_name="$1"
    local command="$2"
    local expected_exit_code="${3:-0}"
    
    ((TOTAL_TESTS++))
    log_info "运行测试: $test_name"
    
    # 使用 eval 来正确处理带管道和重定向的复杂命令
    if eval "$command" > /dev/null 2>&1; then
        actual_exit_code=$?
    else
        actual_exit_code=$?
    fi
    
    if [ $actual_exit_code -eq $expected_exit_code ]; then
        log_success "$test_name"
        return 0
    else
        log_error "$test_name (期望退出码: $expected_exit_code, 实际: $actual_exit_code)"
        return 1
    fi
}

# 检查必要的工具和文件
check_prerequisites() {
    log_info "检查前置条件..."
    
    # 检查 Go
    if ! command -v go &> /dev/null; then
        log_error "Go 未安装"
        exit 1
    fi
    log_success "Go 可用"

    # 检查 jq
    if ! command -v jq &> /dev/null; then
        log_warning "jq 未安装，JSON 对比功能将受限。建议安装 (e.g., brew install jq)"
    else
        log_success "jq 可用"
    fi
    
    # 检查 Make
    if command -v make &> /dev/null; then
        log_success "Make 可用"
    else
        log_warning "Make 不可用，将使用 go 命令"
    fi
}

# 代码质量检查
code_quality_checks() {
    log_info "执行代码质量检查..."
    run_test "代码格式化检查" "make fmt && git diff --exit-code"
    run_test "Go vet 检查" "make vet"
    if command -v golangci-lint &> /dev/null; then
        run_test "Golangci-lint 检查" "make lint"
    else
        log_warning "golangci-lint 不可用，跳过 lint 检查"
    fi
}

# 构建测试
build_tests() {
    log_info "执行构建测试..."
    run_test "清理构建文件" "make clean"
    run_test "下载依赖" "make deps"
    run_test "基本构建" "make build"
    
    if [ -f "build/flowspec-cli" ]; then
        log_success "二进制文件构建成功"
    else
        log_error "二进制文件不存在"
        exit 1
    fi
}

# 单元测试
unit_tests() {
    log_info "执行单元测试..."
    run_test "单元测试执行" "make test"
    run_test "测试覆盖率生成" "make coverage"
}

# 集成测试
integration_tests() {
    log_info "执行集成测试..."
    
    # 测试 CLI 基本功能
    run_test "CLI 帮助信息" "./build/flowspec-cli --help"
    run_test "CLI 版本信息" "./build/flowspec-cli --version"
    
    # 定义测试用例
    declare -A scenarios
    scenarios=(
        ["成功场景"]="success-scenario.json 0 examples/simple-user-service/expected-results/success-report.json"
        ["前置条件失败场景"]="precondition-failure.json 1"
        ["后置条件失败场景"]="postcondition-failure.json 1"
    )

    for name in "${!scenarios[@]}"; do
        params=(${scenarios[$name]})
        trace_file=${params[0]}
        expected_code=${params[1]}
        expected_json_file=${params[2]}

        ((TOTAL_TESTS++))
        log_info "运行集成测试: $name"
        
        actual_output_file=$(mktemp)
        
        # 执行命令并捕获输出和退出码
        set +e
        ./build/flowspec-cli align \
            --path=examples/simple-user-service/src \
            --trace=examples/simple-user-service/traces/$trace_file \
            --output=json > "$actual_output_file" 2>/dev/null
        actual_code=$?
        set -e

        # 检查退出码
        if [ "$actual_code" -ne "$expected_code" ]; then
            log_error "$name - 退出码不匹配 (期望: $expected_code, 实际: $actual_code)"
            rm "$actual_output_file"
            continue
        fi

        # 如果有预期的 JSON 文件，进行内容对比
        if [ -n "$expected_json_file" ]; then
            if ! command -v jq &> /dev/null; then
                log_warning "$name - jq 未安装，跳过 JSON 内容对比"
            else
                # 使用 jq 对比，忽略键顺序和格式
                if jq -e '. == input' "$actual_output_file" "$expected_json_file" > /dev/null; then
                    log_success "$name - 退出码和 JSON 输出均匹配"
                else
                    log_error "$name - JSON 输出不匹配"
                    # diff <(jq . "$expected_json_file") <(jq . "$actual_output_file") || true
                fi
            fi
        else
            log_success "$name - 退出码匹配"
        fi
        
        rm "$actual_output_file"
    done
}

# 主函数
main() {
    check_prerequisites
    echo ""
    code_quality_checks
    echo ""
    build_tests
    echo ""
    unit_tests
    echo ""
    integration_tests
    echo ""
    
    # 最终结果
    echo "🏁 集成测试最终结果"
    echo "========================="
    echo "总测试数: $TOTAL_TESTS"
    echo "通过测试: $PASSED_TESTS"
    echo "失败测试: $FAILED_TESTS"
    
    if [ $FAILED_TESTS -eq 0 ]; then
        log_success "🎉 所有测试通过！"
        exit 0
    else
        log_error "❌ 有 $FAILED_TESTS 个测试失败，请检查日志"
        exit 1
    fi
}

# 运行主函数
main "$@"
