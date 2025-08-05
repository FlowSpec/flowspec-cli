# FlowSpec CLI API 文档

本文档描述了 FlowSpec CLI 的核心 API 接口和数据结构。

## 目录

- [核心接口](#核心接口)
- [数据模型](#数据模型)
- [错误处理](#错误处理)
- [使用示例](#使用示例)

## 核心接口

### SpecParser 接口

ServiceSpec 解析器接口，用于从源代码中提取和解析 ServiceSpec 注解。

```go
type SpecParser interface {
    // ParseFromSource 从指定的源代码目录解析 ServiceSpec 注解
    // sourcePath: 源代码目录路径
    // 返回: 解析结果和错误信息
    ParseFromSource(sourcePath string) (*ParseResult, error)
}

type FileParser interface {
    // CanParse 检查是否可以解析指定文件
    CanParse(filename string) bool
    
    // ParseFile 解析单个文件中的 ServiceSpec 注解
    ParseFile(filepath string) ([]ServiceSpec, []ParseError)
}
```

#### 实现类

- `JavaParser`: 解析 Java 文件中的 `@ServiceSpec` 注解
- `TypeScriptParser`: 解析 TypeScript 文件中的 `@ServiceSpec` 注释
- `GoParser`: 解析 Go 文件中的 `@ServiceSpec` 注释

### TraceIngestor 接口

OpenTelemetry 轨迹数据摄取器接口。

```go
type TraceIngestor interface {
    // IngestFromFile 从文件中摄取轨迹数据
    // tracePath: 轨迹文件路径
    // 返回: 轨迹存储对象和错误信息
    IngestFromFile(tracePath string) (*TraceStore, error)
}

type TraceQuery interface {
    // FindSpanByName 根据轨迹名称和 Span 名称查找 Span
    FindSpanByName(traceName string, spanName string) (*Span, error)
    
    // FindSpansByOperationId 根据操作 ID 查找所有相关 Span
    FindSpansByOperationId(operationId string) ([]*Span, error)
    
    // GetTraceByID 根据轨迹 ID 获取完整轨迹数据
    GetTraceByID(traceId string) (*TraceData, error)
}
```

### AlignmentEngine 接口

规约与轨迹对齐验证引擎接口。

```go
type AlignmentEngine interface {
    // Align 执行规约与轨迹的对齐验证
    // specs: ServiceSpec 列表
    // traceQuery: 轨迹查询接口
    // 返回: 对齐报告和错误信息
    Align(specs []ServiceSpec, traceQuery TraceQuery) (*AlignmentReport, error)
}

type AssertionEvaluator interface {
    // EvaluatePreconditions 评估前置条件
    EvaluatePreconditions(spec ServiceSpec, span *Span) ([]ValidationDetail, error)
    
    // EvaluatePostconditions 评估后置条件
    EvaluatePostconditions(spec ServiceSpec, span *Span) ([]ValidationDetail, error)
}
```

### ReportRenderer 接口

报告渲染器接口。

```go
type ReportRenderer interface {
    // RenderHuman 渲染人类可读格式的报告
    RenderHuman(report *AlignmentReport) (string, error)
    
    // RenderJSON 渲染 JSON 格式的报告
    RenderJSON(report *AlignmentReport) (string, error)
    
    // GetExitCode 根据报告结果获取退出码
    GetExitCode(report *AlignmentReport) int
}
```

## 数据模型

### ServiceSpec

ServiceSpec 表示从源代码中解析出的服务规约。

```go
type ServiceSpec struct {
    // OperationID 操作标识符，用于与轨迹数据匹配
    OperationID string `json:"operationId"`
    
    // Description 操作描述
    Description string `json:"description"`
    
    // Preconditions 前置条件，使用 JSONLogic 格式
    Preconditions map[string]interface{} `json:"preconditions"`
    
    // Postconditions 后置条件，使用 JSONLogic 格式
    Postconditions map[string]interface{} `json:"postconditions"`
    
    // SourceFile 源文件路径
    SourceFile string `json:"sourceFile"`
    
    // LineNumber 在源文件中的行号
    LineNumber int `json:"lineNumber"`
}
```

### ParseResult

解析结果包含成功解析的 ServiceSpec 和解析错误。

```go
type ParseResult struct {
    // Specs 成功解析的 ServiceSpec 列表
    Specs []ServiceSpec `json:"specs"`
    
    // Errors 解析过程中遇到的错误列表
    Errors []ParseError `json:"errors"`
}

type ParseError struct {
    // File 出错的文件路径
    File string `json:"file"`
    
    // Line 出错的行号
    Line int `json:"line"`
    
    // Message 错误信息
    Message string `json:"message"`
}
```

### TraceData

轨迹数据表示一个完整的分布式轨迹。

```go
type TraceData struct {
    // TraceID 轨迹唯一标识符
    TraceID string `json:"traceId"`
    
    // RootSpan 根 Span
    RootSpan *Span `json:"rootSpan"`
    
    // Spans 所有 Span 的映射表 (SpanID -> Span)
    Spans map[string]*Span `json:"spans"`
    
    // SpanTree Span 的树形结构
    SpanTree *SpanNode `json:"spanTree"`
}
```

### Span

Span 表示分布式轨迹中的一个操作单元。

```go
type Span struct {
    // SpanID Span 唯一标识符
    SpanID string `json:"spanId"`
    
    // TraceID 所属轨迹的标识符
    TraceID string `json:"traceId"`
    
    // ParentID 父 Span 的标识符
    ParentID string `json:"parentSpanId,omitempty"`
    
    // Name Span 名称
    Name string `json:"name"`
    
    // StartTime 开始时间
    StartTime time.Time `json:"startTime"`
    
    // EndTime 结束时间
    EndTime time.Time `json:"endTime"`
    
    // Status Span 状态
    Status SpanStatus `json:"status"`
    
    // Attributes Span 属性
    Attributes map[string]interface{} `json:"attributes"`
    
    // Events Span 事件列表
    Events []SpanEvent `json:"events"`
}

type SpanStatus struct {
    // Code 状态码 ("OK", "ERROR", "TIMEOUT")
    Code string `json:"code"`
    
    // Message 状态消息
    Message string `json:"message"`
}

type SpanEvent struct {
    // Name 事件名称
    Name string `json:"name"`
    
    // Timestamp 事件时间戳
    Timestamp time.Time `json:"timestamp"`
    
    // Attributes 事件属性
    Attributes map[string]interface{} `json:"attributes"`
}
```

### AlignmentReport

对齐报告包含验证结果的汇总和详细信息。

```go
type AlignmentReport struct {
    // Summary 汇总统计信息
    Summary AlignmentSummary `json:"summary"`
    
    // Results 详细验证结果列表
    Results []AlignmentResult `json:"results"`
}

type AlignmentSummary struct {
    // Total 总的 ServiceSpec 数量
    Total int `json:"total"`
    
    // Success 验证成功的数量
    Success int `json:"success"`
    
    // Failed 验证失败的数量
    Failed int `json:"failed"`
    
    // Skipped 跳过验证的数量
    Skipped int `json:"skipped"`
}

type AlignmentResult struct {
    // SpecOperationID ServiceSpec 的操作 ID
    SpecOperationID string `json:"specOperationId"`
    
    // Status 验证状态
    Status AlignmentStatus `json:"status"`
    
    // Details 验证详情列表
    Details []ValidationDetail `json:"details"`
    
    // ExecutionTime 执行时间
    ExecutionTime time.Duration `json:"executionTime"`
}

type AlignmentStatus string

const (
    StatusSuccess AlignmentStatus = "SUCCESS"
    StatusFailed  AlignmentStatus = "FAILED"
    StatusSkipped AlignmentStatus = "SKIPPED"
)

type ValidationDetail struct {
    // Type 验证类型 ("precondition" | "postcondition")
    Type string `json:"type"`
    
    // Expression 断言表达式
    Expression string `json:"expression"`
    
    // Expected 期望值
    Expected interface{} `json:"expected"`
    
    // Actual 实际值
    Actual interface{} `json:"actual"`
    
    // Message 验证消息
    Message string `json:"message"`
    
    // SpanContext 相关的 Span 上下文
    SpanContext *Span `json:"spanContext,omitempty"`
}
```

## 错误处理

### 错误类型

FlowSpec CLI 定义了以下错误类型：

```go
// ErrInvalidInput 输入参数无效
var ErrInvalidInput = errors.New("invalid input")

// ErrFileNotFound 文件未找到
var ErrFileNotFound = errors.New("file not found")

// ErrParseError 解析错误
var ErrParseError = errors.New("parse error")

// ErrValidationFailed 验证失败
var ErrValidationFailed = errors.New("validation failed")

// ErrResourceLimit 资源限制
var ErrResourceLimit = errors.New("resource limit exceeded")
```

### 错误包装

使用 `fmt.Errorf` 包装错误以提供更多上下文：

```go
func (p *SpecParser) parseFile(filepath string) error {
    content, err := os.ReadFile(filepath)
    if err != nil {
        return fmt.Errorf("failed to read file %s: %w", filepath, err)
    }
    
    // ... 解析逻辑
    
    return nil
}
```

### 退出码

CLI 工具使用以下退出码：

- `0`: 成功执行
- `1`: 验证失败（断言不通过）
- `2`: 系统错误（文件不存在、解析错误等）

## 使用示例

### 概念性用法

此示例演示了使用核心组件的概念性工作流程。请注意，无法从外部代码直接导入 `internal` 包；这仅用于说明目的。

```go
package main

import (
    "fmt"
    "log"
    "os"
    
    // 注意：这些是内部包，不能被外部直接导入。
    // 此示例仅用于说明组件之间的交互方式。
    "github.com/FlowSpec/flowspec-cli/internal/parser"
    "github.com/FlowSpec/flowspec-cli/internal/ingestor"
    "github.com/FlowSpec/flowspec-cli/internal/engine"
    "github.com/FlowSpec/flowspec-cli/internal/renderer"
)

func main() {
    // 1. 创建 ServiceSpec 解析器并从目录中解析规约
    specParser := parser.NewSpecParser()
    parseResult, err := specParser.ParseFromSource("./my-project")
    if err != nil {
        log.Fatalf("解析 ServiceSpec 失败: %v", err)
    }
    
    // 2. 摄取 OpenTelemetry 轨迹文件
    traceIngestor := ingestor.NewTraceIngestor()
    traceStore, err := traceIngestor.IngestFromFile("./traces/run-1.json")
    if err != nil {
        log.Fatalf("摄取轨迹数据失败: %v", err)
    }
    
    // 3. 创建对齐引擎并执行验证
    alignmentEngine := engine.NewAlignmentEngine()
    report, err := alignmentEngine.Align(parseResult.Specs, traceStore)
    if err != nil {
        log.Fatalf("执行对齐失败: %v", err)
    }
    
    // 4. 创建报告渲染器并显示结果
    reportRenderer := renderer.NewReportRenderer()
    humanReport, err := reportRenderer.RenderHuman(report)
    if err != nil {
        log.Fatalf("渲染报告失败: %v", err)
    }
    
    fmt.Println(humanReport)
    
    // 5. 根据报告确定退出码
    exitCode := reportRenderer.GetExitCode(report)
    os.Exit(exitCode)
}
```

### 自定义解析器

```go
type CustomParser struct {
    // 自定义解析器实现
}

func (p *CustomParser) CanParse(filename string) bool {
    return strings.HasSuffix(filename, ".custom")
}

func (p *CustomParser) ParseFile(filepath string) ([]ServiceSpec, []ParseError) {
    // 自定义解析逻辑
    return specs, errors
}

// 注册自定义解析器
specParser := parser.NewSpecParser()
specParser.RegisterFileParser(&CustomParser{})
```

### JSONLogic 断言示例

```json
{
  "preconditions": {
    "request_validation": {
      "and": [
        {"!=": [{"var": "span.attributes.http.method"}, null]},
        {"==": [{"var": "span.attributes.http.method"}, "POST"]},
        {">=": [{"var": "span.attributes.request.body.password.length"}, 8]}
      ]
    }
  },
  "postconditions": {
    "response_validation": {
      "and": [
        {"==": [{"var": "status.code"}, "OK"]},
        {">=": [{"var": "span.attributes.http.status_code"}, 200]},
        {"<": [{"var": "span.attributes.http.status_code"}, 300]},
        {"!=": [{"var": "span.attributes.response.body.userId"}, null]}
      ]
    }
  }
}
```

### 评估上下文

JSONLogic 表达式的评估上下文包含以下变量：

```json
{
  "span": {
    "attributes": {
      "http.method": "POST",
      "http.status_code": 201,
      "request.body.email": "user@example.com",
      "response.body.userId": "12345"
    },
    "startTime": "2023-10-01T10:00:00Z",
    "name": "createUser"
  },
  "endTime": "2023-10-01T10:00:05Z",
  "status": {
    "code": "OK",
    "message": ""
  },
  "events": [
    {
      "name": "user.created",
      "timestamp": "2023-10-01T10:00:04Z",
      "attributes": {
        "userId": "12345"
      }
    }
  ]
}
```

## 性能考虑

### 内存优化

- 使用流式解析处理大文件
- 实现对象池复用减少 GC 压力
- 提供内存使用监控和限制

### 查询优化

- 构建 Span 名称索引
- 实现操作 ID 映射表
- 使用时间范围索引加速查询

### 并发安全

所有公共接口都是线程安全的，可以在多 goroutine 环境中安全使用。

---

更多详细信息请参考源代码中的注释和测试用例。
