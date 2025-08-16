# FlowSpec 测试代码规范指南

## 概述

本指南定义了 FlowSpec 项目测试代码的编写规范，包括命名规范、结构规范、断言规范等。遵循这些规范可以提高测试代码的可读性、可维护性和一致性。

---

## 📝 命名规范

### 测试函数命名

#### 基本格式
```go
// 格式: Test<FunctionName>_<Scenario>
func TestExecuteAlignment_SuccessfulValidation(t *testing.T) {}
func TestExecuteAlignment_InvalidSourcePath(t *testing.T) {}
func TestExecuteAlignment_TimeoutError(t *testing.T) {}
```

#### 命名最佳实践
```go
// ✅ 好的命名 - 清晰描述测试场景
func TestParseServiceSpec_ValidYAMLFormat(t *testing.T) {}
func TestParseServiceSpec_MalformedJSON(t *testing.T) {}
func TestParseServiceSpec_EmptyFile(t *testing.T) {}

// ❌ 避免的命名 - 模糊不清
func TestParseServiceSpec_Test1(t *testing.T) {}
func TestParseServiceSpec_Good(t *testing.T) {}
func TestParseServiceSpec_Bad(t *testing.T) {}
```

#### 场景命名约定
```go
// 成功场景
func TestFunction_Success(t *testing.T) {}
func TestFunction_ValidInput(t *testing.T) {}
func TestFunction_ExpectedBehavior(t *testing.T) {}

// 错误场景
func TestFunction_InvalidInput(t *testing.T) {}
func TestFunction_MissingParameter(t *testing.T) {}
func TestFunction_NetworkError(t *testing.T) {}

// 边界场景
func TestFunction_EmptyInput(t *testing.T) {}
func TestFunction_MaximumSize(t *testing.T) {}
func TestFunction_BoundaryValue(t *testing.T) {}
```

### 表驱动测试命名

```go
func TestValidateConfig(t *testing.T) {
    testCases := []struct {
        name        string  // 使用 snake_case
        config      Config
        expectError bool
        errorMsg    string
    }{
        {
            name: "valid_config_with_all_fields",
            config: Config{
                SourcePath: "valid/path",
                TracePath:  "valid/trace.json",
            },
            expectError: false,
        },
        {
            name: "missing_source_path",
            config: Config{
                TracePath: "valid/trace.json",
            },
            expectError: true,
            errorMsg:    "source path is required",
        },
        {
            name: "invalid_trace_format",
            config: Config{
                SourcePath: "valid/path",
                TracePath:  "invalid.txt",
            },
            expectError: true,
            errorMsg:    "unsupported trace format",
        },
    }
    
    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            // 测试逻辑
        })
    }
}
```

### 测试文件命名

```go
// 单元测试文件
engine.go -> engine_test.go
parser.go -> parser_test.go
renderer.go -> renderer_test.go

// 集成测试文件
integration_test.go
end_to_end_test.go

// 性能测试文件
performance_test.go
benchmark_test.go

// 特定功能测试文件
alignment_integration_test.go
yaml_parsing_test.go
```

---

## 🏗️ 结构规范

### AAA 模式 (Arrange, Act, Assert)

```go
func TestExecuteAlignment_SuccessfulValidation(t *testing.T) {
    // Arrange - 准备测试数据和环境
    config := AlignConfig{
        SourcePath:   "testdata/valid",
        TracePath:    "testdata/trace.json",
        OutputFormat: "human",
        Timeout:      30 * time.Second,
    }
    
    // Act - 执行被测试的操作
    result, err := executeAlignment(config)
    
    // Assert - 验证结果
    assert.NoError(t, err)
    assert.NotNil(t, result)
    assert.Equal(t, "success", result.Status)
    assert.Greater(t, len(result.Details), 0)
}
```

### 表驱动测试结构

```go
func TestValidateInput(t *testing.T) {
    testCases := []struct {
        name        string
        input       string
        expected    bool
        expectError bool
        errorMsg    string
    }{
        {
            name:        "valid_input",
            input:       "valid-data",
            expected:    true,
            expectError: false,
        },
        {
            name:        "empty_input",
            input:       "",
            expected:    false,
            expectError: true,
            errorMsg:    "input cannot be empty",
        },
    }
    
    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            // Arrange
            // (测试数据已在 testCases 中定义)
            
            // Act
            result, err := validateInput(tc.input)
            
            // Assert
            if tc.expectError {
                assert.Error(t, err)
                if tc.errorMsg != "" {
                    assert.Contains(t, err.Error(), tc.errorMsg)
                }
            } else {
                assert.NoError(t, err)
                assert.Equal(t, tc.expected, result)
            }
        })
    }
}
```

### 复杂测试的组织

```go
func TestComplexWorkflow(t *testing.T) {
    // 使用子测试组织复杂的测试场景
    t.Run("setup_phase", func(t *testing.T) {
        // 设置阶段的测试
    })
    
    t.Run("execution_phase", func(t *testing.T) {
        // 执行阶段的测试
        t.Run("normal_execution", func(t *testing.T) {
            // 正常执行测试
        })
        
        t.Run("error_handling", func(t *testing.T) {
            // 错误处理测试
        })
    })
    
    t.Run("cleanup_phase", func(t *testing.T) {
        // 清理阶段的测试
    })
}
```

---

## ✅ 断言规范

### 推荐的断言库

```go
import (
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)
```

### 断言选择指南

```go
// 使用 assert 进行一般断言 (失败时继续执行)
assert.Equal(t, expected, actual)
assert.NoError(t, err)
assert.True(t, condition)

// 使用 require 进行关键断言 (失败时停止测试)
require.NoError(t, err)  // 如果有错误，后续断言无意义
require.NotNil(t, result)  // 如果为 nil，后续访问会 panic
```

### 具体断言最佳实践

#### 相等性断言
```go
// ✅ 好的断言 - 具体和清晰
assert.Equal(t, "expected-value", result.Value, "Value should match expected")
assert.Equal(t, 42, result.Count, "Count should be 42")

// ❌ 避免的断言 - 模糊和不清晰
assert.True(t, result.Value == "expected-value")
assert.True(t, result.Count == 42)
```

#### 错误断言
```go
// ✅ 好的错误断言
assert.NoError(t, err, "Should not return error for valid input")
assert.Error(t, err, "Should return error for invalid input")
assert.EqualError(t, err, "expected error message")
assert.Contains(t, err.Error(), "partial error message")

// ❌ 避免的错误断言
assert.True(t, err == nil)
assert.True(t, err != nil)
```

#### 集合断言
```go
// ✅ 好的集合断言
assert.Len(t, result.Items, 3, "Should have exactly 3 items")
assert.Contains(t, result.Items, expectedItem, "Should contain expected item")
assert.ElementsMatch(t, expected, actual, "Should contain same elements")

// ❌ 避免的集合断言
assert.True(t, len(result.Items) == 3)
assert.True(t, len(result.Items) > 0)
```

#### 类型断言
```go
// ✅ 好的类型断言
assert.IsType(t, &AlignmentResult{}, result)
assert.Implements(t, (*Renderer)(nil), renderer)

// ❌ 避免的类型断言
assert.True(t, reflect.TypeOf(result) == reflect.TypeOf(&AlignmentResult{}))
```

### 自定义断言消息

```go
// ✅ 提供有用的断言消息
assert.Equal(t, expected, actual, "Alignment result should match expected output")
assert.NoError(t, err, "File parsing should succeed for valid YAML")
assert.Greater(t, result.ProcessingTime, 0, "Processing time should be positive")

// ✅ 使用格式化消息
assert.Equal(t, expected, actual, "Expected %s but got %s", expected, actual)
assert.Len(t, items, expectedCount, "Expected %d items but got %d", expectedCount, len(items))
```

---

## 🧪 测试数据管理

### 测试数据组织

```
testdata/
├── valid/                    # 有效测试数据
│   ├── service-spec.yaml
│   ├── trace.json
│   └── expected-result.json
├── invalid/                  # 无效测试数据
│   ├── malformed.yaml
│   ├── empty-trace.json
│   └── invalid-format.json
├── edge-cases/              # 边界情况数据
│   ├── large-trace.json
│   ├── unicode-data.yaml
│   └── boundary-values.json
└── fixtures/                # 测试固件
    ├── mock-responses/
    └── sample-configs/
```

### 测试数据生成函数

```go
// 使用工厂函数生成测试数据
func createValidAlignConfig() AlignConfig {
    return AlignConfig{
        SourcePath:   "testdata/valid",
        TracePath:    "testdata/trace.json",
        OutputFormat: "human",
        Timeout:      30 * time.Second,
        MaxWorkers:   4,
    }
}

func createInvalidAlignConfig() AlignConfig {
    return AlignConfig{
        SourcePath: "", // 无效的空路径
        TracePath:  "nonexistent.json",
    }
}

// 使用构建器模式创建复杂测试数据
type AlignConfigBuilder struct {
    config AlignConfig
}

func NewAlignConfigBuilder() *AlignConfigBuilder {
    return &AlignConfigBuilder{
        config: AlignConfig{
            OutputFormat: "human",
            Timeout:      30 * time.Second,
            MaxWorkers:   4,
        },
    }
}

func (b *AlignConfigBuilder) WithSourcePath(path string) *AlignConfigBuilder {
    b.config.SourcePath = path
    return b
}

func (b *AlignConfigBuilder) WithTracePath(path string) *AlignConfigBuilder {
    b.config.TracePath = path
    return b
}

func (b *AlignConfigBuilder) Build() AlignConfig {
    return b.config
}

// 使用示例
func TestExecuteAlignment_CustomConfig(t *testing.T) {
    config := NewAlignConfigBuilder().
        WithSourcePath("testdata/custom").
        WithTracePath("testdata/custom-trace.json").
        Build()
    
    result, err := executeAlignment(config)
    
    assert.NoError(t, err)
    assert.NotNil(t, result)
}
```

### 测试数据清理

```go
func TestWithTempFile(t *testing.T) {
    // 创建临时文件
    tmpFile, err := os.CreateTemp("", "test-*.json")
    require.NoError(t, err)
    defer os.Remove(tmpFile.Name()) // 确保清理
    
    // 写入测试数据
    testData := `{"test": "data"}`
    _, err = tmpFile.WriteString(testData)
    require.NoError(t, err)
    require.NoError(t, tmpFile.Close())
    
    // 执行测试
    result, err := parseFile(tmpFile.Name())
    
    assert.NoError(t, err)
    assert.Equal(t, "data", result.Test)
}

func TestWithTempDir(t *testing.T) {
    // 创建临时目录
    tmpDir, err := os.MkdirTemp("", "test-*")
    require.NoError(t, err)
    defer os.RemoveAll(tmpDir) // 确保清理
    
    // 在临时目录中创建测试文件
    testFile := filepath.Join(tmpDir, "test.yaml")
    err = os.WriteFile(testFile, []byte("test: data"), 0644)
    require.NoError(t, err)
    
    // 执行测试
    result, err := parseDirectory(tmpDir)
    
    assert.NoError(t, err)
    assert.Len(t, result, 1)
}
```

---

## 🎭 Mock 和 Stub 规范

### Mock 使用指南

```go
// 使用 testify/mock 创建 Mock
type MockRenderer struct {
    mock.Mock
}

func (m *MockRenderer) RenderHuman(report *AlignmentReport) (string, error) {
    args := m.Called(report)
    return args.String(0), args.Error(1)
}

func TestWithMockRenderer(t *testing.T) {
    // Arrange
    mockRenderer := new(MockRenderer)
    mockRenderer.On("RenderHuman", mock.AnythingOfType("*AlignmentReport")).
        Return("mocked output", nil)
    
    service := NewReportService(mockRenderer)
    
    // Act
    output, err := service.GenerateReport(&AlignmentReport{})
    
    // Assert
    assert.NoError(t, err)
    assert.Equal(t, "mocked output", output)
    mockRenderer.AssertExpectations(t) // 验证 Mock 调用
}
```

### 接口隔离

```go
// 定义最小接口用于测试
type FileReader interface {
    ReadFile(filename string) ([]byte, error)
}

type Parser struct {
    reader FileReader
}

// 在测试中使用简单的 Mock 实现
type MockFileReader struct {
    files map[string][]byte
    err   error
}

func (m *MockFileReader) ReadFile(filename string) ([]byte, error) {
    if m.err != nil {
        return nil, m.err
    }
    if content, exists := m.files[filename]; exists {
        return content, nil
    }
    return nil, os.ErrNotExist
}

func TestParser_WithMockReader(t *testing.T) {
    // Arrange
    mockReader := &MockFileReader{
        files: map[string][]byte{
            "test.yaml": []byte("test: data"),
        },
    }
    parser := &Parser{reader: mockReader}
    
    // Act
    result, err := parser.ParseFile("test.yaml")
    
    // Assert
    assert.NoError(t, err)
    assert.Equal(t, "data", result.Test)
}
```

---

## 📊 性能和基准测试

### 基准测试规范

```go
func BenchmarkExecuteAlignment(b *testing.B) {
    config := createValidAlignConfig()
    
    b.ResetTimer() // 重置计时器，排除准备时间
    
    for i := 0; i < b.N; i++ {
        _, err := executeAlignment(config)
        if err != nil {
            b.Fatal(err)
        }
    }
}

func BenchmarkExecuteAlignment_WithSizes(b *testing.B) {
    sizes := []int{100, 1000, 10000}
    
    for _, size := range sizes {
        b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
            config := createConfigWithSize(size)
            
            b.ResetTimer()
            for i := 0; i < b.N; i++ {
                _, err := executeAlignment(config)
                if err != nil {
                    b.Fatal(err)
                }
            }
        })
    }
}
```

### 内存分配测试

```go
func BenchmarkExecuteAlignment_Memory(b *testing.B) {
    config := createValidAlignConfig()
    
    b.ReportAllocs() // 报告内存分配
    b.ResetTimer()
    
    for i := 0; i < b.N; i++ {
        _, err := executeAlignment(config)
        if err != nil {
            b.Fatal(err)
        }
    }
}
```

---

## 🔧 测试工具和辅助函数

### 常用测试辅助函数

```go
// 测试辅助函数应该以 helper 开头或放在 helper.go 文件中
func helperCreateTempFile(t *testing.T, content string) string {
    t.Helper() // 标记为辅助函数，错误时显示调用者位置
    
    tmpFile, err := os.CreateTemp("", "test-*.json")
    require.NoError(t, err)
    
    _, err = tmpFile.WriteString(content)
    require.NoError(t, err)
    require.NoError(t, tmpFile.Close())
    
    t.Cleanup(func() {
        os.Remove(tmpFile.Name())
    })
    
    return tmpFile.Name()
}

func helperAssertValidationResult(t *testing.T, result *ValidationResult, expectedStatus string) {
    t.Helper()
    
    assert.NotNil(t, result, "Validation result should not be nil")
    assert.Equal(t, expectedStatus, result.Status, "Status should match expected")
    assert.NotEmpty(t, result.Details, "Details should not be empty")
}
```

### 测试套件组织

```go
// 使用 testify/suite 组织相关测试
type AlignmentTestSuite struct {
    suite.Suite
    tempDir string
    config  AlignConfig
}

func (suite *AlignmentTestSuite) SetupSuite() {
    // 整个测试套件的设置
    var err error
    suite.tempDir, err = os.MkdirTemp("", "alignment-test-*")
    suite.Require().NoError(err)
}

func (suite *AlignmentTestSuite) TearDownSuite() {
    // 整个测试套件的清理
    os.RemoveAll(suite.tempDir)
}

func (suite *AlignmentTestSuite) SetupTest() {
    // 每个测试的设置
    suite.config = createValidAlignConfig()
}

func (suite *AlignmentTestSuite) TestExecuteAlignment_Success() {
    result, err := executeAlignment(suite.config)
    
    suite.NoError(err)
    suite.NotNil(result)
}

func TestAlignmentTestSuite(t *testing.T) {
    suite.Run(t, new(AlignmentTestSuite))
}
```

---

## 📚 文档和注释规范

### 测试文档

```go
// TestExecuteAlignment_SuccessfulValidation 测试成功的对齐验证场景
// 
// 此测试验证当提供有效的配置时，executeAlignment 函数能够：
// 1. 正确解析源文件和轨迹文件
// 2. 执行对齐验证逻辑
// 3. 返回预期的验证结果
//
// 测试数据：
// - 源路径：testdata/valid (包含有效的 ServiceSpec)
// - 轨迹路径：testdata/trace.json (包含匹配的轨迹数据)
func TestExecuteAlignment_SuccessfulValidation(t *testing.T) {
    // 测试实现
}
```

### 复杂测试逻辑注释

```go
func TestComplexAlignmentLogic(t *testing.T) {
    // 准备测试数据：创建包含多个端点的复杂 ServiceSpec
    spec := &ServiceSpec{
        Operations: []Operation{
            {ID: "op1", Path: "/users", Method: "GET"},
            {ID: "op2", Path: "/users/{id}", Method: "POST"},
        },
    }
    
    // 创建包含嵌套 span 的复杂轨迹数据
    // 这模拟了真实场景中的微服务调用链
    trace := &TraceData{
        Spans: []Span{
            {ID: "span1", OperationName: "GET /users", ParentID: ""},
            {ID: "span2", OperationName: "POST /users/123", ParentID: "span1"},
        },
    }
    
    // 执行对齐：这里测试的是复杂的匹配算法
    // 算法需要正确处理路径参数化和层级关系
    result, err := AlignSpecsWithTrace([]*ServiceSpec{spec}, trace)
    
    // 验证结果：确保所有操作都被正确匹配
    assert.NoError(t, err)
    assert.Len(t, result.MatchedOperations, 2, "Both operations should be matched")
    
    // 验证路径参数化是否正确处理
    // "/users/{id}" 应该匹配 "/users/123"
    userDetailMatch := findMatchByOperationID(result.MatchedOperations, "op2")
    assert.NotNil(t, userDetailMatch, "User detail operation should be matched")
    assert.Equal(t, "123", userDetailMatch.ExtractedParams["id"], "ID parameter should be extracted")
}
```

---

## 🚀 最佳实践总结

### DO - 应该做的

1. **使用清晰的命名**：测试函数名应该清楚描述测试场景
2. **遵循 AAA 模式**：Arrange, Act, Assert 结构清晰
3. **使用表驱动测试**：处理多个相似的测试用例
4. **提供有意义的断言消息**：帮助快速定位问题
5. **适当使用 Mock**：隔离外部依赖
6. **清理测试资源**：使用 defer 或 t.Cleanup
7. **测试边界条件**：空值、最大值、异常情况
8. **保持测试独立**：测试之间不应该有依赖关系

### DON'T - 不应该做的

1. **不要使用模糊的测试名称**：避免 Test1, TestGood 等
2. **不要在测试中使用随机数据**：除非专门测试随机性
3. **不要忽略错误处理**：测试中也要正确处理错误
4. **不要写过于复杂的测试**：一个测试应该只验证一个场景
5. **不要在测试中使用 panic**：使用适当的断言
6. **不要忽略测试清理**：避免测试间的相互影响
7. **不要过度使用 Mock**：只在必要时使用
8. **不要写不稳定的测试**：避免时间依赖和竞争条件

---

## 📋 代码审查检查清单

在代码审查时，请检查以下项目：

- [ ] 测试函数命名遵循规范
- [ ] 测试结构清晰 (AAA 模式)
- [ ] 断言使用恰当且有意义的消息
- [ ] 测试数据管理合理
- [ ] Mock 使用适当
- [ ] 测试覆盖了主要场景和边界条件
- [ ] 测试代码可读性好
- [ ] 测试执行稳定且快速
- [ ] 测试资源得到正确清理
- [ ] 复杂逻辑有适当注释

---

*此规范指南应该与团队一起持续改进和完善，确保测试代码质量不断提升。*