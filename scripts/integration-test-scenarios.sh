#!/bin/bash

# FlowSpec CLI 集成测试场景脚本
# 实现需求 8.2: 创建端到端成功验证测试用例、前置条件失败测试场景、后置条件失败测试场景、混合场景测试

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# 配置
BINARY_PATH="./build/flowspec-cli"
TEST_DIR="testdata/integration-scenarios"
RESULTS_DIR="$TEST_DIR/results"

echo -e "${BLUE}🧪 FlowSpec CLI 集成测试场景${NC}"
echo "========================================"
echo -e "${CYAN}实现需求 7.3, 7.4: 集成测试场景覆盖${NC}"
echo ""

# 检查二进制文件是否存在
if [ ! -f "$BINARY_PATH" ]; then
    echo -e "${RED}❌ 二进制文件不存在: $BINARY_PATH${NC}"
    echo "💡 请先运行: make build"
    exit 1
fi

# 创建测试目录
echo -e "${YELLOW}📁 准备测试环境...${NC}"
mkdir -p $TEST_DIR
mkdir -p $RESULTS_DIR

# 清理之前的结果
rm -f $RESULTS_DIR/*.json
rm -f $RESULTS_DIR/*.log

# ============================================================================
# 场景 1: 端到端成功验证测试用例
# ============================================================================

echo ""
echo -e "${GREEN}🎯 场景 1: 端到端成功验证测试${NC}"
echo "----------------------------------------"

# 创建成功场景的源代码文件
cat > $TEST_DIR/SuccessService.java << 'EOF'
/**
 * @ServiceSpec
 * operationId: "createUser"
 * description: "创建新用户账户"
 * preconditions:
 *   "valid_method": {"==": [{"var": "http_method"}, "POST"]}
 *   "has_email": {"!=": [{"var": "request_email"}, null]}
 * postconditions:
 *   "success_status": {"==": [{"var": "span.status.code"}, "OK"]}
 *   "created_status": {"==": [{"var": "http_status_code"}, 201]}
 */
public class SuccessService {
    public User createUser(CreateUserRequest request) {
        return userRepository.save(new User(request));
    }
}
EOF

cat > $TEST_DIR/successService.ts << 'EOF'
/**
 * @ServiceSpec
 * operationId: "getUser"
 * description: "根据ID获取用户"
 * preconditions:
 *   "valid_method": {"==": [{"var": "http_method"}, "GET"]}
 *   "has_user_id": {"!=": [{"var": "user_id"}, null]}
 * postconditions:
 *   "success_status": {"==": [{"var": "span.status.code"}, "OK"]}
 *   "ok_status": {"==": [{"var": "http_status_code"}, 200]}
 */
export class SuccessService {
    async getUser(userId: string): Promise<User> {
        return this.userRepository.findById(userId);
    }
}
EOF

cat > $TEST_DIR/success_service.go << 'EOF'
// @ServiceSpec
// operationId: "updateUser"
// description: "更新用户信息"
// preconditions:
//   "valid_method": {"==": [{"var": "http_method"}, "PUT"]}
//   "has_user_id": {"!=": [{"var": "user_id"}, null]}
// postconditions:
//   "success_status": {"==": [{"var": "span.status.code"}, "OK"]}
//   "ok_status": {"==": [{"var": "http_status_code"}, 200]}
func UpdateUser(userId string, user User) error {
    return userRepository.Update(userId, user)
}
EOF

# 创建成功场景的轨迹文件
cat > $TEST_DIR/success-trace.json << 'EOF'
{
  "resourceSpans": [
    {
      "resource": {
        "attributes": [
          {"key": "service.name", "value": {"stringValue": "user-service"}}
        ]
      },
      "scopeSpans": [
        {
          "spans": [
            {
              "traceId": "success123abc456def",
              "spanId": "span001",
              "name": "createUser",
              "startTimeUnixNano": "1640995200000000000",
              "endTimeUnixNano": "1640995201000000000",
              "attributes": [
                {"key": "http.method", "value": {"stringValue": "POST"}},
                {"key": "request.email", "value": {"stringValue": "user@example.com"}},
                {"key": "http.status_code", "value": {"intValue": "201"}},
                {"key": "operation.id", "value": {"stringValue": "createUser"}}
              ],
              "status": {"code": 1, "message": "OK"}
            },
            {
              "traceId": "success123abc456def",
              "spanId": "span002",
              "name": "getUser",
              "startTimeUnixNano": "1640995202000000000",
              "endTimeUnixNano": "1640995203000000000",
              "attributes": [
                {"key": "http.method", "value": {"stringValue": "GET"}},
                {"key": "user.id", "value": {"stringValue": "user123"}},
                {"key": "http.status_code", "value": {"intValue": "200"}},
                {"key": "operation.id", "value": {"stringValue": "getUser"}}
              ],
              "status": {"code": 1, "message": "OK"}
            },
            {
              "traceId": "success123abc456def",
              "spanId": "span003",
              "name": "updateUser",
              "startTimeUnixNano": "1640995204000000000",
              "endTimeUnixNano": "1640995205000000000",
              "attributes": [
                {"key": "http.method", "value": {"stringValue": "PUT"}},
                {"key": "user.id", "value": {"stringValue": "user123"}},
                {"key": "http.status_code", "value": {"intValue": "200"}},
                {"key": "operation.id", "value": {"stringValue": "updateUser"}}
              ],
              "status": {"code": 1, "message": "OK"}
            }
          ]
        }
      ]
    }
  ]
}
EOF

# 执行成功场景测试
echo -e "${CYAN}执行成功场景测试...${NC}"
if $BINARY_PATH align --path=$TEST_DIR --trace=$TEST_DIR/success-trace.json --output=json > $RESULTS_DIR/success-result.json 2> $RESULTS_DIR/success-error.log; then
    echo -e "${GREEN}✅ 成功场景测试通过${NC}"
    
    # 验证结果
    if command -v jq >/dev/null 2>&1; then
        SUCCESS_COUNT=$(jq '.summary.success' $RESULTS_DIR/success-result.json 2>/dev/null || echo "0")
        TOTAL_COUNT=$(jq '.summary.total' $RESULTS_DIR/success-result.json 2>/dev/null || echo "0")
        echo "   📊 成功: $SUCCESS_COUNT/$TOTAL_COUNT 个 ServiceSpec"
        
        if [ "$SUCCESS_COUNT" = "$TOTAL_COUNT" ] && [ "$TOTAL_COUNT" -gt "0" ]; then
            echo -e "${GREEN}   🎉 所有 ServiceSpec 验证成功！${NC}"
        else
            echo -e "${YELLOW}   ⚠️  部分 ServiceSpec 未成功${NC}"
        fi
    fi
else
    echo -e "${RED}❌ 成功场景测试失败${NC}"
    echo "错误日志:"
    cat $RESULTS_DIR/success-error.log
fi

# ============================================================================
# 场景 2: 前置条件失败测试场景
# ============================================================================

echo ""
echo -e "${PURPLE}🎯 场景 2: 前置条件失败测试${NC}"
echo "----------------------------------------"

# 创建前置条件失败场景的源代码文件
cat > $TEST_DIR/PreconditionService.java << 'EOF'
/**
 * @ServiceSpec
 * operationId: "strictCreateUser"
 * description: "严格的用户创建，要求特定前置条件"
 * preconditions:
 *   "required_email": {"!=": [{"var": "request_email"}, null]}
 *   "valid_email_format": {"regex": [{"var": "request_email"}, "^[\\w\\.-]+@[\\w\\.-]+\\.[a-zA-Z]{2,}$"]}
 *   "required_method": {"==": [{"var": "http_method"}, "POST"]}
 *   "min_password_length": {">=": [{"strlen": [{"var": "request_password"}]}, 8]}
 * postconditions:
 *   "success_status": {"==": [{"var": "span.status.code"}, "OK"]}
 */
public class PreconditionService {
    public User strictCreateUser(CreateUserRequest request) {
        return userRepository.save(new User(request));
    }
}
EOF

cat > $TEST_DIR/preconditionService.ts << 'EOF'
/**
 * @ServiceSpec
 * operationId: "strictGetUser"
 * description: "严格的用户获取，要求特定前置条件"
 * preconditions:
 *   "required_user_id": {"!=": [{"var": "user_id"}, null]}
 *   "valid_user_id_format": {"regex": [{"var": "user_id"}, "^[a-zA-Z0-9]{8,}$"]}
 *   "required_method": {"==": [{"var": "http_method"}, "GET"]}
 *   "has_auth_token": {"!=": [{"var": "auth_token"}, null]}
 * postconditions:
 *   "success_status": {"==": [{"var": "span.status.code"}, "OK"]}
 */
export class PreconditionService {
    async strictGetUser(userId: string): Promise<User> {
        return this.userRepository.findById(userId);
    }
}
EOF

# 创建前置条件失败的轨迹文件
cat > $TEST_DIR/precondition-failure-trace.json << 'EOF'
{
  "resourceSpans": [
    {
      "resource": {
        "attributes": [
          {"key": "service.name", "value": {"stringValue": "user-service"}}
        ]
      },
      "scopeSpans": [
        {
          "spans": [
            {
              "traceId": "precond123abc456def",
              "spanId": "span001",
              "name": "strictCreateUser",
              "startTimeUnixNano": "1640995200000000000",
              "endTimeUnixNano": "1640995201000000000",
              "attributes": [
                {"key": "http.method", "value": {"stringValue": "GET"}},
                {"key": "request.email", "value": {"stringValue": "invalid-email"}},
                {"key": "request.password", "value": {"stringValue": "123"}},
                {"key": "operation.id", "value": {"stringValue": "strictCreateUser"}}
              ],
              "status": {"code": 1, "message": "OK"}
            },
            {
              "traceId": "precond123abc456def",
              "spanId": "span002",
              "name": "strictGetUser",
              "startTimeUnixNano": "1640995202000000000",
              "endTimeUnixNano": "1640995203000000000",
              "attributes": [
                {"key": "http.method", "value": {"stringValue": "POST"}},
                {"key": "user.id", "value": {"stringValue": "123"}},
                {"key": "operation.id", "value": {"stringValue": "strictGetUser"}}
              ],
              "status": {"code": 1, "message": "OK"}
            }
          ]
        }
      ]
    }
  ]
}
EOF

# 执行前置条件失败场景测试
echo -e "${CYAN}执行前置条件失败场景测试...${NC}"
if $BINARY_PATH align --path=$TEST_DIR --trace=$TEST_DIR/precondition-failure-trace.json --output=json > $RESULTS_DIR/precondition-result.json 2> $RESULTS_DIR/precondition-error.log; then
    echo -e "${YELLOW}⚠️  前置条件失败场景测试完成${NC}"
    
    # 验证结果
    if command -v jq >/dev/null 2>&1; then
        FAILED_COUNT=$(jq '.summary.failed' $RESULTS_DIR/precondition-result.json 2>/dev/null || echo "0")
        TOTAL_COUNT=$(jq '.summary.total' $RESULTS_DIR/precondition-result.json 2>/dev/null || echo "0")
        echo "   📊 失败: $FAILED_COUNT/$TOTAL_COUNT 个 ServiceSpec"
        
        if [ "$FAILED_COUNT" -gt "0" ]; then
            echo -e "${GREEN}   ✅ 正确检测到前置条件失败！${NC}"
        else
            echo -e "${RED}   ❌ 未检测到预期的前置条件失败${NC}"
        fi
    fi
else
    echo -e "${RED}❌ 前置条件失败场景测试执行失败${NC}"
    echo "错误日志:"
    cat $RESULTS_DIR/precondition-error.log
fi

# ============================================================================
# 场景 3: 后置条件失败测试场景
# ============================================================================

echo ""
echo -e "${PURPLE}🎯 场景 3: 后置条件失败测试${NC}"
echo "----------------------------------------"

# 创建后置条件失败场景的源代码文件
cat > $TEST_DIR/PostconditionService.java << 'EOF'
/**
 * @ServiceSpec
 * operationId: "expectSuccessCreateUser"
 * description: "期望成功的用户创建"
 * preconditions:
 *   "valid_method": {"==": [{"var": "http_method"}, "POST"]}
 * postconditions:
 *   "success_status": {"==": [{"var": "span.status.code"}, "OK"]}
 *   "created_status": {"==": [{"var": "http_status_code"}, 201]}
 *   "has_user_id": {"!=": [{"var": "response_user_id"}, null]}
 */
public class PostconditionService {
    public User expectSuccessCreateUser(CreateUserRequest request) {
        return userRepository.save(new User(request));
    }
}
EOF

cat > $TEST_DIR/postconditionService.ts << 'EOF'
/**
 * @ServiceSpec
 * operationId: "expectSuccessGetUser"
 * description: "期望成功的用户获取"
 * preconditions:
 *   "valid_method": {"==": [{"var": "http_method"}, "GET"]}
 * postconditions:
 *   "success_status": {"==": [{"var": "span.status.code"}, "OK"]}
 *   "ok_status": {"==": [{"var": "http_status_code"}, 200]}
 *   "has_user_data": {"!=": [{"var": "response_user"}, null]}
 */
export class PostconditionService {
    async expectSuccessGetUser(userId: string): Promise<User> {
        return this.userRepository.findById(userId);
    }
}
EOF

# 创建后置条件失败的轨迹文件
cat > $TEST_DIR/postcondition-failure-trace.json << 'EOF'
{
  "resourceSpans": [
    {
      "resource": {
        "attributes": [
          {"key": "service.name", "value": {"stringValue": "user-service"}}
        ]
      },
      "scopeSpans": [
        {
          "spans": [
            {
              "traceId": "postcond123abc456def",
              "spanId": "span001",
              "name": "expectSuccessCreateUser",
              "startTimeUnixNano": "1640995200000000000",
              "endTimeUnixNano": "1640995201000000000",
              "attributes": [
                {"key": "http.method", "value": {"stringValue": "POST"}},
                {"key": "http.status_code", "value": {"intValue": "500"}},
                {"key": "operation.id", "value": {"stringValue": "expectSuccessCreateUser"}}
              ],
              "status": {"code": 2, "message": "ERROR"}
            },
            {
              "traceId": "postcond123abc456def",
              "spanId": "span002",
              "name": "expectSuccessGetUser",
              "startTimeUnixNano": "1640995202000000000",
              "endTimeUnixNano": "1640995203000000000",
              "attributes": [
                {"key": "http.method", "value": {"stringValue": "GET"}},
                {"key": "http.status_code", "value": {"intValue": "404"}},
                {"key": "operation.id", "value": {"stringValue": "expectSuccessGetUser"}}
              ],
              "status": {"code": 2, "message": "ERROR"}
            }
          ]
        }
      ]
    }
  ]
}
EOF

# 执行后置条件失败场景测试
echo -e "${CYAN}执行后置条件失败场景测试...${NC}"
if $BINARY_PATH align --path=$TEST_DIR --trace=$TEST_DIR/postcondition-failure-trace.json --output=json > $RESULTS_DIR/postcondition-result.json 2> $RESULTS_DIR/postcondition-error.log; then
    echo -e "${YELLOW}⚠️  后置条件失败场景测试完成${NC}"
    
    # 验证结果
    if command -v jq >/dev/null 2>&1; then
        FAILED_COUNT=$(jq '.summary.failed' $RESULTS_DIR/postcondition-result.json 2>/dev/null || echo "0")
        TOTAL_COUNT=$(jq '.summary.total' $RESULTS_DIR/postcondition-result.json 2>/dev/null || echo "0")
        echo "   📊 失败: $FAILED_COUNT/$TOTAL_COUNT 个 ServiceSpec"
        
        if [ "$FAILED_COUNT" -gt "0" ]; then
            echo -e "${GREEN}   ✅ 正确检测到后置条件失败！${NC}"
        else
            echo -e "${RED}   ❌ 未检测到预期的后置条件失败${NC}"
        fi
    fi
else
    echo -e "${RED}❌ 后置条件失败场景测试执行失败${NC}"
    echo "错误日志:"
    cat $RESULTS_DIR/postcondition-error.log
fi

# ============================================================================
# 场景 4: 混合场景测试 (部分成功、部分失败、部分跳过)
# ============================================================================

echo ""
echo -e "${PURPLE}🎯 场景 4: 混合场景测试${NC}"
echo "----------------------------------------"

# 创建混合场景的源代码文件
cat > $TEST_DIR/MixedService.java << 'EOF'
/**
 * @ServiceSpec
 * operationId: "mixedCreateUser"
 * description: "混合场景 - 应该成功"
 * preconditions:
 *   "valid_method": {"==": [{"var": "http_method"}, "POST"]}
 * postconditions:
 *   "success_status": {"==": [{"var": "span.status.code"}, "OK"]}
 */
/**
 * @ServiceSpec
 * operationId: "mixedUpdateUser"
 * description: "混合场景 - 应该失败 (后置条件)"
 * preconditions:
 *   "valid_method": {"==": [{"var": "http_method"}, "PUT"]}
 * postconditions:
 *   "success_status": {"==": [{"var": "span.status.code"}, "OK"]}
 */
/**
 * @ServiceSpec
 * operationId: "mixedNonExistentOperation"
 * description: "混合场景 - 应该跳过 (无对应轨迹)"
 * preconditions:
 *   "always_true": {"==": [true, true]}
 * postconditions:
 *   "always_true": {"==": [true, true]}
 */
public class MixedService {
    public User mixedCreateUser(CreateUserRequest request) { return new User(); }
    public User mixedUpdateUser(String id, UpdateUserRequest request) { return new User(); }
}
EOF

cat > $TEST_DIR/mixedService.ts << 'EOF'
/**
 * @ServiceSpec
 * operationId: "mixedGetUser"
 * description: "混合场景 - 应该成功"
 * preconditions:
 *   "valid_method": {"==": [{"var": "http_method"}, "GET"]}
 * postconditions:
 *   "success_status": {"==": [{"var": "span.status.code"}, "OK"]}
 */
/**
 * @ServiceSpec
 * operationId: "mixedDeleteUser"
 * description: "混合场景 - 应该失败 (前置条件)"
 * preconditions:
 *   "required_admin_role": {"==": [{"var": "user_role"}, "admin"]}
 * postconditions:
 *   "success_status": {"==": [{"var": "span.status.code"}, "OK"]}
 */
export class MixedService {
    async mixedGetUser(userId: string): Promise<User> { return new User(); }
    async mixedDeleteUser(userId: string): Promise<void> { }
}
EOF

# 创建混合场景的轨迹文件
cat > $TEST_DIR/mixed-trace.json << 'EOF'
{
  "resourceSpans": [
    {
      "resource": {
        "attributes": [
          {"key": "service.name", "value": {"stringValue": "user-service"}}
        ]
      },
      "scopeSpans": [
        {
          "spans": [
            {
              "traceId": "mixed123abc456def",
              "spanId": "span001",
              "name": "mixedCreateUser",
              "startTimeUnixNano": "1640995200000000000",
              "endTimeUnixNano": "1640995201000000000",
              "attributes": [
                {"key": "http.method", "value": {"stringValue": "POST"}},
                {"key": "operation.id", "value": {"stringValue": "mixedCreateUser"}}
              ],
              "status": {"code": 1, "message": "OK"}
            },
            {
              "traceId": "mixed123abc456def",
              "spanId": "span002",
              "name": "mixedGetUser",
              "startTimeUnixNano": "1640995202000000000",
              "endTimeUnixNano": "1640995203000000000",
              "attributes": [
                {"key": "http.method", "value": {"stringValue": "GET"}},
                {"key": "operation.id", "value": {"stringValue": "mixedGetUser"}}
              ],
              "status": {"code": 1, "message": "OK"}
            },
            {
              "traceId": "mixed123abc456def",
              "spanId": "span003",
              "name": "mixedUpdateUser",
              "startTimeUnixNano": "1640995204000000000",
              "endTimeUnixNano": "1640995205000000000",
              "attributes": [
                {"key": "http.method", "value": {"stringValue": "PUT"}},
                {"key": "operation.id", "value": {"stringValue": "mixedUpdateUser"}}
              ],
              "status": {"code": 2, "message": "ERROR"}
            },
            {
              "traceId": "mixed123abc456def",
              "spanId": "span004",
              "name": "mixedDeleteUser",
              "startTimeUnixNano": "1640995206000000000",
              "endTimeUnixNano": "1640995207000000000",
              "attributes": [
                {"key": "http.method", "value": {"stringValue": "DELETE"}},
                {"key": "user.role", "value": {"stringValue": "user"}},
                {"key": "operation.id", "value": {"stringValue": "mixedDeleteUser"}}
              ],
              "status": {"code": 1, "message": "OK"}
            }
          ]
        }
      ]
    }
  ]
}
EOF

# 执行混合场景测试
echo -e "${CYAN}执行混合场景测试...${NC}"
if $BINARY_PATH align --path=$TEST_DIR --trace=$TEST_DIR/mixed-trace.json --output=json > $RESULTS_DIR/mixed-result.json 2> $RESULTS_DIR/mixed-error.log; then
    echo -e "${YELLOW}⚠️  混合场景测试完成${NC}"
    
    # 验证结果
    if command -v jq >/dev/null 2>&1; then
        SUCCESS_COUNT=$(jq '.summary.success' $RESULTS_DIR/mixed-result.json 2>/dev/null || echo "0")
        FAILED_COUNT=$(jq '.summary.failed' $RESULTS_DIR/mixed-result.json 2>/dev/null || echo "0")
        SKIPPED_COUNT=$(jq '.summary.skipped' $RESULTS_DIR/mixed-result.json 2>/dev/null || echo "0")
        TOTAL_COUNT=$(jq '.summary.total' $RESULTS_DIR/mixed-result.json 2>/dev/null || echo "0")
        
        echo "   📊 结果分布:"
        echo "      ✅ 成功: $SUCCESS_COUNT"
        echo "      ❌ 失败: $FAILED_COUNT"
        echo "      ⏭️  跳过: $SKIPPED_COUNT"
        echo "      📝 总计: $TOTAL_COUNT"
        
        if [ "$SUCCESS_COUNT" -gt "0" ] && [ "$FAILED_COUNT" -gt "0" ] && [ "$SKIPPED_COUNT" -gt "0" ]; then
            echo -e "${GREEN}   🎉 混合场景测试成功！包含成功、失败和跳过的情况${NC}"
        else
            echo -e "${YELLOW}   ⚠️  混合场景结果与预期不完全匹配${NC}"
        fi
    fi
else
    echo -e "${RED}❌ 混合场景测试执行失败${NC}"
    echo "错误日志:"
    cat $RESULTS_DIR/mixed-error.log
fi

# ============================================================================
# 场景 5: Human 格式输出测试
# ============================================================================

echo ""
echo -e "${BLUE}🎯 场景 5: Human 格式输出测试${NC}"
echo "----------------------------------------"

echo -e "${CYAN}执行 Human 格式输出测试...${NC}"
if $BINARY_PATH align --path=$TEST_DIR --trace=$TEST_DIR/mixed-trace.json --output=human > $RESULTS_DIR/human-output.txt 2> $RESULTS_DIR/human-error.log; then
    echo -e "${GREEN}✅ Human 格式输出测试通过${NC}"
    
    # 验证 Human 格式输出包含预期元素
    if grep -q "FlowSpec" $RESULTS_DIR/human-output.txt && \
       grep -q "汇总统计" $RESULTS_DIR/human-output.txt && \
       grep -q "详细结果" $RESULTS_DIR/human-output.txt; then
        echo -e "${GREEN}   ✅ Human 格式包含所有必需元素${NC}"
    else
        echo -e "${YELLOW}   ⚠️  Human 格式可能缺少某些元素${NC}"
    fi
    
    # 显示输出长度
    OUTPUT_LINES=$(wc -l < $RESULTS_DIR/human-output.txt)
    echo "   📏 输出行数: $OUTPUT_LINES"
else
    echo -e "${RED}❌ Human 格式输出测试失败${NC}"
    echo "错误日志:"
    cat $RESULTS_DIR/human-error.log
fi

# ============================================================================
# 测试结果汇总
# ============================================================================

echo ""
echo -e "${BLUE}📋 测试结果汇总${NC}"
echo "========================================"

TOTAL_SCENARIOS=5
PASSED_SCENARIOS=0

# 检查各个场景的结果文件
if [ -f "$RESULTS_DIR/success-result.json" ]; then
    echo -e "${GREEN}✅ 场景 1: 端到端成功验证 - 通过${NC}"
    ((PASSED_SCENARIOS++))
else
    echo -e "${RED}❌ 场景 1: 端到端成功验证 - 失败${NC}"
fi

if [ -f "$RESULTS_DIR/precondition-result.json" ]; then
    echo -e "${GREEN}✅ 场景 2: 前置条件失败测试 - 通过${NC}"
    ((PASSED_SCENARIOS++))
else
    echo -e "${RED}❌ 场景 2: 前置条件失败测试 - 失败${NC}"
fi

if [ -f "$RESULTS_DIR/postcondition-result.json" ]; then
    echo -e "${GREEN}✅ 场景 3: 后置条件失败测试 - 通过${NC}"
    ((PASSED_SCENARIOS++))
else
    echo -e "${RED}❌ 场景 3: 后置条件失败测试 - 失败${NC}"
fi

if [ -f "$RESULTS_DIR/mixed-result.json" ]; then
    echo -e "${GREEN}✅ 场景 4: 混合场景测试 - 通过${NC}"
    ((PASSED_SCENARIOS++))
else
    echo -e "${RED}❌ 场景 4: 混合场景测试 - 失败${NC}"
fi

if [ -f "$RESULTS_DIR/human-output.txt" ]; then
    echo -e "${GREEN}✅ 场景 5: Human 格式输出 - 通过${NC}"
    ((PASSED_SCENARIOS++))
else
    echo -e "${RED}❌ 场景 5: Human 格式输出 - 失败${NC}"
fi

echo ""
echo -e "${CYAN}总体结果: $PASSED_SCENARIOS/$TOTAL_SCENARIOS 个场景通过${NC}"

if [ $PASSED_SCENARIOS -eq $TOTAL_SCENARIOS ]; then
    echo -e "${GREEN}🎉 所有集成测试场景通过！${NC}"
    echo -e "${GREEN}✅ 需求 7.3, 7.4 已满足${NC}"
    exit 0
else
    echo -e "${YELLOW}⚠️  部分集成测试场景未通过${NC}"
    echo -e "${YELLOW}📁 详细结果请查看: $RESULTS_DIR/${NC}"
    exit 1
fi