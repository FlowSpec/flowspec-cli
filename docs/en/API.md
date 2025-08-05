# FlowSpec CLI API Documentation

This document describes the core API interfaces and data structures of the FlowSpec CLI.

## Table of Contents

- [Core Interfaces](#core-interfaces)
- [Data Models](#data-models)
- [Error Handling](#error-handling)
- [Usage Examples](#usage-examples)

## Core Interfaces

### SpecParser Interface

The ServiceSpec parser interface, used to extract and parse ServiceSpec annotations from source code.

```go
type SpecParser interface {
    // ParseFromSource parses ServiceSpec annotations from the specified source code directory.
    // sourcePath: The path to the source code directory.
    // Returns: The parsing result and an error.
    ParseFromSource(sourcePath string) (*ParseResult, error)
}

type FileParser interface {
    // CanParse checks if the specified file can be parsed.
    CanParse(filename string) bool
    
    // ParseFile parses ServiceSpec annotations from a single file.
    ParseFile(filepath string) ([]ServiceSpec, []ParseError)
}
```

#### Implementations

- `JavaParser`: Parses `@ServiceSpec` annotations from Java files.
- `TypeScriptParser`: Parses `@ServiceSpec` comments from TypeScript files.
- `GoParser`: Parses `@ServiceSpec` comments from Go files.

### TraceIngestor Interface

The OpenTelemetry trace data ingestor interface.

```go
type TraceIngestor interface {
    // IngestFromFile ingests trace data from a file.
    // tracePath: The path to the trace file.
    // Returns: A trace store object and an error.
    IngestFromFile(tracePath string) (*TraceStore, error)
}

type TraceQuery interface {
    // FindSpanByName finds a span by trace name and span name.
    FindSpanByName(traceName string, spanName string) (*Span, error)
    
    // FindSpansByOperationId finds all related spans by operation ID.
    FindSpansByOperationId(operationId string) ([]*Span, error)
    
    // GetTraceByID gets the full trace data by trace ID.
    GetTraceByID(traceId string) (*TraceData, error)
}
```

### AlignmentEngine Interface

The specification and trace alignment validation engine interface.

```go
type AlignmentEngine interface {
    // Align performs alignment validation between specifications and traces.
    // specs: A list of ServiceSpecs.
    // traceQuery: The trace query interface.
    // Returns: An alignment report and an error.
    Align(specs []ServiceSpec, traceQuery TraceQuery) (*AlignmentReport, error)
}

type AssertionEvaluator interface {
    // EvaluatePreconditions evaluates preconditions.
    EvaluatePreconditions(spec ServiceSpec, span *Span) ([]ValidationDetail, error)
    
    // EvaluatePostconditions evaluates postconditions.
    EvaluatePostconditions(spec ServiceSpec, span *Span) ([]ValidationDetail, error)
}
```

### ReportRenderer Interface

The report renderer interface.

```go
type ReportRenderer interface {
    // RenderHuman renders a report in human-readable format.
    RenderHuman(report *AlignmentReport) (string, error)
    
    // RenderJSON renders a report in JSON format.
    RenderJSON(report *AlignmentReport) (string, error)
    
    // GetExitCode gets the exit code based on the report result.
    GetExitCode(report *AlignmentReport) int
}
```

## Data Models

### ServiceSpec

`ServiceSpec` represents a service specification parsed from the source code.

```go
type ServiceSpec struct {
    // OperationID is the identifier for the operation, used to match with trace data.
    OperationID string `json:"operationId"`
    
    // Description is the description of the operation.
    Description string `json:"description"`
    
    // Preconditions are the preconditions in JSONLogic format.
    Preconditions map[string]interface{} `json:"preconditions"`
    
    // Postconditions are the postconditions in JSONLogic format.
    Postconditions map[string]interface{} `json:"postconditions"`
    
    // SourceFile is the path to the source file.
    SourceFile string `json:"sourceFile"`
    
    // LineNumber is the line number in the source file.
    LineNumber int `json:"lineNumber"`
}
```

### ParseResult

`ParseResult` contains successfully parsed ServiceSpecs and parsing errors.

```go
type ParseResult struct {
    // Specs is a list of successfully parsed ServiceSpecs.
    Specs []ServiceSpec `json:"specs"`
    
    // Errors is a list of errors encountered during parsing.
    Errors []ParseError `json:"errors"`
}

type ParseError struct {
    // File is the path to the file where the error occurred.
    File string `json:"file"`
    
    // Line is the line number where the error occurred.
    Line int `json:"line"`
    
    // Message is the error message.
    Message string `json:"message"`
}
```

### TraceData

`TraceData` represents a complete distributed trace.

```go
type TraceData struct {
    // TraceID is the unique identifier for the trace.
    TraceID string `json:"traceId"`
    
    // RootSpan is the root span.
    RootSpan *Span `json:"rootSpan"`
    
    // Spans is a map of all spans (SpanID -> Span).
    Spans map[string]*Span `json:"spans"`
    
    // SpanTree is the tree structure of the spans.
    SpanTree *SpanNode `json:"spanTree"`
}
```

### Span

`Span` represents an operational unit in a distributed trace.

```go
type Span struct {
    // SpanID is the unique identifier for the span.
    SpanID string `json:"spanId"`
    
    // TraceID is the identifier of the trace it belongs to.
    TraceID string `json:"traceId"`
    
    // ParentID is the identifier of the parent span.
    ParentID string `json:"parentSpanId,omitempty"`
    
    // Name is the name of the span.
    Name string `json:"name"`
    
    // StartTime is the start time.
    StartTime time.Time `json:"startTime"`
    
    // EndTime is the end time.
    EndTime time.Time `json:"endTime"`
    
    // Status is the status of the span.
    Status SpanStatus `json:"status"`
    
    // Attributes are the attributes of the span.
    Attributes map[string]interface{} `json:"attributes"`
    
    // Events is a list of events for the span.
    Events []SpanEvent `json:"events"`
}

type SpanStatus struct {
    // Code is the status code ("OK", "ERROR", "TIMEOUT").
    Code string `json:"code"`
    
    // Message is the status message.
    Message string `json:"message"`
}

type SpanEvent struct {
    // Name is the name of the event.
    Name string `json:"name"`
    
    // Timestamp is the timestamp of the event.
    Timestamp time.Time `json:"timestamp"`
    
    // Attributes are the attributes of the event.
    Attributes map[string]interface{} `json:"attributes"`
}
```

### AlignmentReport

`AlignmentReport` contains a summary and detailed information of the validation results.

```go
type AlignmentReport struct {
    // Summary is the summary statistics.
    Summary AlignmentSummary `json:"summary"`
    
    // Results is a list of detailed validation results.
    Results []AlignmentResult `json:"results"`
}

type AlignmentSummary struct {
    // Total is the total number of ServiceSpecs.
    Total int `json:"total"`
    
    // Success is the number of successful validations.
    Success int `json:"success"`
    
    // Failed is the number of failed validations.
    Failed int `json:"failed"`
    
    // Skipped is the number of skipped validations.
    Skipped int `json:"skipped"`
}

type AlignmentResult struct {
    // SpecOperationID is the operation ID of the ServiceSpec.
    SpecOperationID string `json:"specOperationId"`
    
    // Status is the validation status.
    Status AlignmentStatus `json:"status"`
    
    // Details is a list of validation details.
    Details []ValidationDetail `json:"details"`
    
    // ExecutionTime is the execution time.
    ExecutionTime time.Duration `json:"executionTime"`
}

type AlignmentStatus string

const (
    StatusSuccess AlignmentStatus = "SUCCESS"
    StatusFailed  AlignmentStatus = "FAILED"
    StatusSkipped AlignmentStatus = "SKIPPED"
)

type ValidationDetail struct {
    // Type is the validation type ("precondition" | "postcondition").
    Type string `json:"type"`
    
    // Expression is the assertion expression.
    Expression string `json:"expression"`
    
    // Expected is the expected value.
    Expected interface{} `json:"expected"`
    
    // Actual is the actual value.
    Actual interface{} `json:"actual"`
    
    // Message is the validation message.
    Message string `json:"message"`
    
    // SpanContext is the related span context.
    SpanContext *Span `json:"spanContext,omitempty"`
}
```

## Error Handling

### Error Types

The FlowSpec CLI defines the following error types:

```go
// ErrInvalidInput indicates invalid input parameters.
var ErrInvalidInput = errors.New("invalid input")

// ErrFileNotFound indicates that a file was not found.
var ErrFileNotFound = errors.New("file not found")

// ErrParseError indicates a parsing error.
var ErrParseError = errors.New("parse error")

// ErrValidationFailed indicates a validation failure.
var ErrValidationFailed = errors.New("validation failed")

// ErrResourceLimit indicates a resource limit was exceeded.
var ErrResourceLimit = errors.New("resource limit exceeded")
```

### Error Wrapping

Use `fmt.Errorf` to wrap errors to provide more context:

```go
func (p *SpecParser) parseFile(filepath string) error {
    content, err := os.ReadFile(filepath)
    if err != nil {
        return fmt.Errorf("failed to read file %s: %w", filepath, err)
    }
    
    // ... parsing logic
    
    return nil
}
```

### Exit Codes

The CLI tool uses the following exit codes:

- `0`: Successful execution
- `1`: Validation failed (assertions did not pass)
- `2`: System error (file not found, parse error, etc.)

## Usage Examples

### Conceptual Usage

This example demonstrates the conceptual workflow of using the core components. Note that direct import of `internal` packages is not possible from external code; this is for illustrative purposes only.

```go
package main

import (
    "fmt"
    "log"
    "os"
    
    // Note: These are internal packages and cannot be imported directly.
    // This example illustrates the interaction between components.
    "github.com/FlowSpec/flowspec-cli/internal/parser"
    "github.com/FlowSpec/flowspec-cli/internal/ingestor"
    "github.com/FlowSpec/flowspec-cli/internal/engine"
    "github.com/FlowSpec/flowspec-cli/internal/renderer"
)

func main() {
    // 1. Create a ServiceSpec parser and parse specifications from a directory.
    specParser := parser.NewSpecParser()
    parseResult, err := specParser.ParseFromSource("./my-project")
    if err != nil {
        log.Fatalf("Failed to parse ServiceSpecs: %v", err)
    }
    
    // 2. Ingest an OpenTelemetry trace file.
    traceIngestor := ingestor.NewTraceIngestor()
    traceStore, err := traceIngestor.IngestFromFile("./traces/run-1.json")
    if err != nil {
        log.Fatalf("Failed to ingest trace data: %v", err)
    }
    
    // 3. Create an alignment engine and perform validation.
    alignmentEngine := engine.NewAlignmentEngine()
    report, err := alignmentEngine.Align(parseResult.Specs, traceStore)
    if err != nil {
        log.Fatalf("Failed to perform alignment: %v", err)
    }
    
    // 4. Create a report renderer and display the results.
    reportRenderer := renderer.NewReportRenderer()
    humanReport, err := reportRenderer.RenderHuman(report)
    if err != nil {
        log.Fatalf("Failed to render report: %v", err)
    }
    
    fmt.Println(humanReport)
    
    // 5. Determine the exit code based on the report.
    exitCode := reportRenderer.GetExitCode(report)
    os.Exit(exitCode)
}
```

### Custom Parser

```go
type CustomParser struct {
    // Custom parser implementation
}

func (p *CustomParser) CanParse(filename string) bool {
    return strings.HasSuffix(filename, ".custom")
}

func (p *CustomParser) ParseFile(filepath string) ([]ServiceSpec, []ParseError) {
    // Custom parsing logic
    return specs, errors
}

// Register the custom parser
specParser := parser.NewSpecParser()
specParser.RegisterFileParser(&CustomParser{})
```

### JSONLogic Assertion Example

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

### Evaluation Context

The evaluation context for JSONLogic expressions contains the following variables:

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

## Performance Considerations

### Memory Optimization

- Use stream parsing to handle large files.
- Implement an object pool to reduce GC pressure.
- Provide memory usage monitoring and limits.

### Query Optimization

- Build an index on span names.
- Implement an operation ID map.
- Use a time range index to speed up queries.

### Concurrency Safety

All public interfaces are thread-safe and can be used safely in a multi-goroutine environment.

---

For more details, please refer to the comments and test cases in the source code.
