#!/bin/bash

# 生成示例轨迹数据脚本

set -e

echo "📊 生成 FlowSpec CLI 示例轨迹数据"
echo "================================="

EXAMPLES_DIR="examples"
TRACE_DIR="traces"

# 创建示例轨迹数据目录
mkdir -p "$EXAMPLES_DIR/simple-user-service/$TRACE_DIR"

echo "🔄 生成简单用户服务示例轨迹..."

# 成功场景轨迹
cat > "$EXAMPLES_DIR/simple-user-service/$TRACE_DIR/success-scenario.json" << 'EOF'
{
  "resourceSpans": [{
    "resource": {
      "attributes": [
        {"key": "service.name", "value": {"stringValue": "user-service"}},
        {"key": "service.version", "value": {"stringValue": "1.0.0"}}
      ]
    },
    "scopeSpans": [{
      "scope": {
        "name": "user-service-tracer",
        "version": "1.0.0"
      },
      "spans": [
        {
          "traceId": "1234567890abcdef1234567890abcdef",
          "spanId": "abcdef1234567890",
          "name": "createUser",
          "kind": "SPAN_KIND_SERVER",
          "startTimeUnixNano": "1640995200000000000",
          "endTimeUnixNano": "1640995201000000000",
          "status": {
            "code": "STATUS_CODE_OK"
          },
          "attributes": [
            {"key": "http.method", "value": {"stringValue": "POST"}},
            {"key": "http.url", "value": {"stringValue": "/api/users"}},
            {"key": "http.status_code", "value": {"intValue": 201}},
            {"key": "request.body.email", "value": {"stringValue": "user@example.com"}},
            {"key": "request.body.name", "value": {"stringValue": "John Doe"}},
            {"key": "request.body.password.length", "value": {"intValue": 12}},
            {"key": "response.body.userId", "value": {"stringValue": "12345"}},
            {"key": "response.body.email", "value": {"stringValue": "user@example.com"}},
            {"key": "response.body.name", "value": {"stringValue": "John Doe"}}
          ],
          "events": [
            {
              "timeUnixNano": "1640995200500000000",
              "name": "user.validation.completed",
              "attributes": [
                {"key": "validation.result", "value": {"stringValue": "success"}}
              ]
            },
            {
              "timeUnixNano": "1640995200800000000",
              "name": "user.created",
              "attributes": [
                {"key": "user.id", "value": {"stringValue": "12345"}}
              ]
            }
          ]
        },
        {
          "traceId": "1234567890abcdef1234567890abcdef",
          "spanId": "1234567890abcdef",
          "name": "getUser",
          "kind": "SPAN_KIND_SERVER",
          "startTimeUnixNano": "1640995202000000000",
          "endTimeUnixNano": "1640995202500000000",
          "status": {
            "code": "STATUS_CODE_OK"
          },
          "attributes": [
            {"key": "http.method", "value": {"stringValue": "GET"}},
            {"key": "http.url", "value": {"stringValue": "/api/users/12345"}},
            {"key": "http.status_code", "value": {"intValue": 200}},
            {"key": "request.params.userId", "value": {"stringValue": "12345"}},
            {"key": "response.body.userId", "value": {"stringValue": "12345"}},
            {"key": "response.body.email", "value": {"stringValue": "user@example.com"}},
            {"key": "response.body.name", "value": {"stringValue": "John Doe"}}
          ]
        },
        {
          "traceId": "1234567890abcdef1234567890abcdef",
          "spanId": "fedcba0987654321",
          "name": "updateUser",
          "kind": "SPAN_KIND_SERVER",
          "startTimeUnixNano": "1640995203000000000",
          "endTimeUnixNano": "1640995203800000000",
          "status": {
            "code": "STATUS_CODE_OK"
          },
          "attributes": [
            {"key": "http.method", "value": {"stringValue": "PUT"}},
            {"key": "http.url", "value": {"stringValue": "/api/users/12345"}},
            {"key": "http.status_code", "value": {"intValue": 200}},
            {"key": "request.params.userId", "value": {"stringValue": "12345"}},
            {"key": "request.body.name", "value": {"stringValue": "John Smith"}},
            {"key": "request.body.email", "value": {"stringValue": "john.smith@example.com"}},
            {"key": "response.body.userId", "value": {"stringValue": "12345"}},
            {"key": "response.body.email", "value": {"stringValue": "john.smith@example.com"}},
            {"key": "response.body.name", "value": {"stringValue": "John Smith"}}
          ]
        },
        {
          "traceId": "1234567890abcdef1234567890abcdef",
          "spanId": "0987654321fedcba",
          "name": "deleteUser",
          "kind": "SPAN_KIND_SERVER",
          "startTimeUnixNano": "1640995204000000000",
          "endTimeUnixNano": "1640995204200000000",
          "status": {
            "code": "STATUS_CODE_OK"
          },
          "attributes": [
            {"key": "http.method", "value": {"stringValue": "DELETE"}},
            {"key": "http.url", "value": {"stringValue": "/api/users/12345"}},
            {"key": "http.status_code", "value": {"intValue": 204}},
            {"key": "request.params.userId", "value": {"stringValue": "12345"}},
            {"key": "response.body", "value": {"stringValue": ""}}
          ]
        }
      ]
    }]
  }]
}
EOF

# 前置条件失败场景轨迹
cat > "$EXAMPLES_DIR/simple-user-service/$TRACE_DIR/precondition-failure.json" << 'EOF'
{
  "resourceSpans": [{
    "resource": {
      "attributes": [
        {"key": "service.name", "value": {"stringValue": "user-service"}},
        {"key": "service.version", "value": {"stringValue": "1.0.0"}}
      ]
    },
    "scopeSpans": [{
      "scope": {
        "name": "user-service-tracer",
        "version": "1.0.0"
      },
      "spans": [
        {
          "traceId": "abcdef1234567890abcdef1234567890",
          "spanId": "1111222233334444",
          "name": "createUser",
          "kind": "SPAN_KIND_SERVER",
          "startTimeUnixNano": "1640995300000000000",
          "endTimeUnixNano": "1640995300100000000",
          "status": {
            "code": "STATUS_CODE_ERROR",
            "message": "Invalid input: password too short"
          },
          "attributes": [
            {"key": "http.method", "value": {"stringValue": "POST"}},
            {"key": "http.url", "value": {"stringValue": "/api/users"}},
            {"key": "http.status_code", "value": {"intValue": 400}},
            {"key": "request.body.email", "value": {"stringValue": "invalid-email"}},
            {"key": "request.body.name", "value": {"stringValue": "Test User"}},
            {"key": "request.body.password.length", "value": {"intValue": 5}},
            {"key": "response.body.error", "value": {"stringValue": "Password must be at least 8 characters"}}
          ],
          "events": [
            {
              "timeUnixNano": "1640995300050000000",
              "name": "validation.failed",
              "attributes": [
                {"key": "validation.error", "value": {"stringValue": "password_too_short"}}
              ]
            }
          ]
        }
      ]
    }]
  }]
}
EOF

# 后置条件失败场景轨迹
cat > "$EXAMPLES_DIR/simple-user-service/$TRACE_DIR/postcondition-failure.json" << 'EOF'
{
  "resourceSpans": [{
    "resource": {
      "attributes": [
        {"key": "service.name", "value": {"stringValue": "user-service"}},
        {"key": "service.version", "value": {"stringValue": "1.0.0"}}
      ]
    },
    "scopeSpans": [{
      "scope": {
        "name": "user-service-tracer",
        "version": "1.0.0"
      },
      "spans": [
        {
          "traceId": "fedcba0987654321fedcba0987654321",
          "spanId": "5555666677778888",
          "name": "createUser",
          "kind": "SPAN_KIND_SERVER",
          "startTimeUnixNano": "1640995400000000000",
          "endTimeUnixNano": "1640995405000000000",
          "status": {
            "code": "STATUS_CODE_ERROR",
            "message": "Internal server error"
          },
          "attributes": [
            {"key": "http.method", "value": {"stringValue": "POST"}},
            {"key": "http.url", "value": {"stringValue": "/api/users"}},
            {"key": "http.status_code", "value": {"intValue": 500}},
            {"key": "request.body.email", "value": {"stringValue": "user@example.com"}},
            {"key": "request.body.name", "value": {"stringValue": "Test User"}},
            {"key": "request.body.password.length", "value": {"intValue": 12}},
            {"key": "response.body.error", "value": {"stringValue": "Database connection failed"}}
          ],
          "events": [
            {
              "timeUnixNano": "1640995404000000000",
              "name": "database.error",
              "attributes": [
                {"key": "error.type", "value": {"stringValue": "connection_timeout"}}
              ]
            }
          ]
        }
      ]
    }]
  }]
}
EOF

echo "✅ 简单用户服务示例轨迹生成完成"

# 创建预期结果文件
mkdir -p "$EXAMPLES_DIR/simple-user-service/expected-results"

echo "📋 生成预期验证结果..."

# 成功场景预期结果
cat > "$EXAMPLES_DIR/simple-user-service/expected-results/success-report.json" << 'EOF'
{
  "summary": {
    "total": 4,
    "success": 4,
    "failed": 0,
    "skipped": 0
  },
  "results": [
    {
      "specOperationId": "createUser",
      "status": "SUCCESS",
      "details": [],
      "executionTime": "15ms"
    },
    {
      "specOperationId": "getUser",
      "status": "SUCCESS",
      "details": [],
      "executionTime": "8ms"
    },
    {
      "specOperationId": "updateUser",
      "status": "SUCCESS",
      "details": [],
      "executionTime": "12ms"
    },
    {
      "specOperationId": "deleteUser",
      "status": "SUCCESS",
      "details": [],
      "executionTime": "5ms"
    }
  ]
}
EOF

echo "✅ 预期结果文件生成完成"

# 创建测试脚本
cat > "$EXAMPLES_DIR/simple-user-service/test-example.sh" << 'EOF'
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
EOF

chmod +x "$EXAMPLES_DIR/simple-user-service/test-example.sh"

echo "✅ 测试脚本生成完成"

echo ""
echo "🎉 示例轨迹数据生成完成！"
echo "========================="
echo ""
echo "📁 生成的文件:"
echo "  $EXAMPLES_DIR/simple-user-service/$TRACE_DIR/success-scenario.json"
echo "  $EXAMPLES_DIR/simple-user-service/$TRACE_DIR/precondition-failure.json"
echo "  $EXAMPLES_DIR/simple-user-service/$TRACE_DIR/postcondition-failure.json"
echo "  $EXAMPLES_DIR/simple-user-service/expected-results/success-report.json"
echo "  $EXAMPLES_DIR/simple-user-service/test-example.sh"
echo ""
echo "🧪 运行示例测试:"
echo "  cd $EXAMPLES_DIR/simple-user-service"
echo "  ./test-example.sh"
echo ""
echo "💡 手动测试命令:"
echo "  flowspec-cli align --path=$EXAMPLES_DIR/simple-user-service/src --trace=$EXAMPLES_DIR/simple-user-service/$TRACE_DIR/success-scenario.json --output=human"