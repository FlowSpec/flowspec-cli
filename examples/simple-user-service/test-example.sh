#!/bin/bash

# 简单用户服务示例测试脚本

set -e

echo "🧪 测试简单用户服务示例"
echo "======================="

EXAMPLE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_BINARY="../../build/flowspec-cli"

# 检查 CLI 二进制文件是否存在
if [ ! -f "$CLI_BINARY" ]; then
    echo "❌ FlowSpec CLI 二进制文件不存在: $CLI_BINARY"
    echo "💡 请先运行: make build"
    exit 1
fi

echo "📍 示例目录: $EXAMPLE_DIR"
echo "🔧 CLI 二进制: $CLI_BINARY"

# 测试成功场景
echo ""
echo "🟢 测试成功场景..."
echo "命令: $CLI_BINARY align --path=$EXAMPLE_DIR/src --trace=$EXAMPLE_DIR/traces/success-scenario.json --output=human"
$CLI_BINARY align --path="$EXAMPLE_DIR/src" --trace="$EXAMPLE_DIR/traces/success-scenario.json" --output=human

EXIT_CODE=$?
if [ $EXIT_CODE -eq 0 ]; then
    echo "✅ 成功场景测试通过 (退出码: $EXIT_CODE)"
else
    echo "❌ 成功场景测试失败 (退出码: $EXIT_CODE)"
fi

echo ""
echo "🔴 测试前置条件失败场景..."
echo "命令: $CLI_BINARY align --path=$EXAMPLE_DIR/src --trace=$EXAMPLE_DIR/traces/precondition-failure.json --output=human"
$CLI_BINARY align --path="$EXAMPLE_DIR/src" --trace="$EXAMPLE_DIR/traces/precondition-failure.json" --output=human

EXIT_CODE=$?
if [ $EXIT_CODE -eq 1 ]; then
    echo "✅ 前置条件失败场景测试通过 (退出码: $EXIT_CODE)"
else
    echo "❌ 前置条件失败场景测试失败 (退出码: $EXIT_CODE)"
fi

echo ""
echo "🟡 测试后置条件失败场景..."
echo "命令: $CLI_BINARY align --path=$EXAMPLE_DIR/src --trace=$EXAMPLE_DIR/traces/postcondition-failure.json --output=human"
$CLI_BINARY align --path="$EXAMPLE_DIR/src" --trace="$EXAMPLE_DIR/traces/postcondition-failure.json" --output=human

EXIT_CODE=$?
if [ $EXIT_CODE -eq 1 ]; then
    echo "✅ 后置条件失败场景测试通过 (退出码: $EXIT_CODE)"
else
    echo "❌ 后置条件失败场景测试失败 (退出码: $EXIT_CODE)"
fi

echo ""
echo "📊 JSON 格式输出测试..."
echo "命令: $CLI_BINARY align --path=$EXAMPLE_DIR/src --trace=$EXAMPLE_DIR/traces/success-scenario.json --output=json"
JSON_OUTPUT=$($CLI_BINARY align --path="$EXAMPLE_DIR/src" --trace="$EXAMPLE_DIR/traces/success-scenario.json" --output=json)

# 验证 JSON 格式
if echo "$JSON_OUTPUT" | jq . > /dev/null 2>&1; then
    echo "✅ JSON 格式输出测试通过"
    echo "📋 JSON 输出摘要:"
    echo "$JSON_OUTPUT" | jq '.summary'
else
    echo "❌ JSON 格式输出测试失败"
fi

echo ""
echo "🎉 示例测试完成！"
