// Copyright 2024-2025 FlowSpec
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package renderer

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/flowspec/flowspec-cli/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewReportRenderer(t *testing.T) {
	renderer := NewReportRenderer()

	assert.NotNil(t, renderer)
	assert.NotNil(t, renderer.config)
	assert.True(t, renderer.config.ShowTimestamps)
	assert.True(t, renderer.config.ShowPerformance)
	assert.True(t, renderer.config.ShowDetailedErrors)
	assert.True(t, renderer.config.ColorOutput)
}

func TestNewReportRendererWithConfig(t *testing.T) {
	config := &RendererConfig{
		ShowTimestamps:     false,
		ShowPerformance:    false,
		ShowDetailedErrors: false,
		ColorOutput:        false,
	}

	renderer := NewReportRendererWithConfig(config)

	assert.NotNil(t, renderer)
	assert.Equal(t, config, renderer.config)
	assert.False(t, renderer.config.ShowTimestamps)
	assert.False(t, renderer.config.ShowPerformance)
	assert.False(t, renderer.config.ShowDetailedErrors)
	assert.False(t, renderer.config.ColorOutput)
}

func TestRenderHuman_EmptyReport(t *testing.T) {
	// Use renderer without colors for easier testing
	config := DefaultRendererConfig()
	config.ColorOutput = false
	renderer := NewReportRendererWithConfig(config)
	report := models.NewAlignmentReport()

	output, err := renderer.RenderHuman(report)

	require.NoError(t, err)
	assert.Contains(t, output, "FlowSpec 验证报告")
	assert.Contains(t, output, "总计: 0 个 ServiceSpec")
	assert.Contains(t, output, "成功: 0 个")
	assert.Contains(t, output, "失败: 0 个")
	assert.Contains(t, output, "跳过: 0 个")
}

func TestRenderHuman_SuccessfulReport(t *testing.T) {
	// Use renderer without colors for easier testing
	config := DefaultRendererConfig()
	config.ColorOutput = false
	renderer := NewReportRendererWithConfig(config)
	report := createTestReport(t, []models.AlignmentStatus{
		models.StatusSuccess,
		models.StatusSuccess,
	})

	output, err := renderer.RenderHuman(report)

	require.NoError(t, err)
	assert.Contains(t, output, "总计: 2 个 ServiceSpec")
	assert.Contains(t, output, "成功: 2 个")
	assert.Contains(t, output, "失败: 0 个")
	assert.Contains(t, output, "验证结果: ✅ 成功")
	assert.Contains(t, output, "所有断言通过")
}

func TestRenderHuman_FailedReport(t *testing.T) {
	// Use renderer without colors for easier testing
	config := DefaultRendererConfig()
	config.ColorOutput = false
	renderer := NewReportRendererWithConfig(config)
	report := createTestReport(t, []models.AlignmentStatus{
		models.StatusSuccess,
		models.StatusFailed,
	})

	output, err := renderer.RenderHuman(report)

	require.NoError(t, err)
	assert.Contains(t, output, "总计: 2 个 ServiceSpec")
	assert.Contains(t, output, "成功: 1 个")
	assert.Contains(t, output, "失败: 1 个")
	assert.Contains(t, output, "验证结果: ❌ 失败")
}

func TestRenderHuman_MixedReport(t *testing.T) {
	// Use renderer without colors for easier testing
	config := DefaultRendererConfig()
	config.ColorOutput = false
	renderer := NewReportRendererWithConfig(config)
	report := createTestReport(t, []models.AlignmentStatus{
		models.StatusSuccess,
		models.StatusFailed,
		models.StatusSkipped,
	})

	output, err := renderer.RenderHuman(report)

	require.NoError(t, err)
	assert.Contains(t, output, "总计: 3 个 ServiceSpec")
	assert.Contains(t, output, "成功: 1 个")
	assert.Contains(t, output, "失败: 1 个")
	assert.Contains(t, output, "跳过: 1 个")
	assert.Contains(t, output, "失败的验证 (1 个)")
	assert.Contains(t, output, "成功的验证 (1 个)")
	assert.Contains(t, output, "跳过的验证 (1 个)")
}

func TestRenderHuman_WithPerformanceMetrics(t *testing.T) {
	// Use renderer without colors for easier testing
	config := DefaultRendererConfig()
	config.ColorOutput = false
	renderer := NewReportRendererWithConfig(config)
	report := createTestReport(t, []models.AlignmentStatus{models.StatusSuccess})

	// Add performance metrics
	report.PerformanceInfo = models.PerformanceInfo{
		SpecsProcessed:      1,
		ProcessingRate:      10.5,
		MemoryUsageMB:       25.3,
		ConcurrentWorkers:   4,
		AssertionsEvaluated: 1, // Match the test report which has 1 assertion
	}

	output, err := renderer.RenderHuman(report)

	require.NoError(t, err)
	assert.Contains(t, output, "性能指标")
	assert.Contains(t, output, "处理速度: 10.50 specs/秒")
	assert.Contains(t, output, "内存使用: 25.30 MB")
	assert.Contains(t, output, "并发工作线程: 4 个")
	assert.Contains(t, output, "断言评估: 1 个")
}

func TestRenderHuman_WithoutColors(t *testing.T) {
	config := DefaultRendererConfig()
	config.ColorOutput = false
	renderer := NewReportRendererWithConfig(config)

	report := createTestReport(t, []models.AlignmentStatus{models.StatusFailed})

	output, err := renderer.RenderHuman(report)

	require.NoError(t, err)
	// Should not contain ANSI color codes
	assert.NotContains(t, output, "\033[")
	assert.Contains(t, output, "FlowSpec 验证报告")
	assert.Contains(t, output, "❌ 失败")
}

func TestRenderHuman_DetailedErrors(t *testing.T) {
	// Use renderer without colors for easier testing
	config := DefaultRendererConfig()
	config.ColorOutput = false
	renderer := NewReportRendererWithConfig(config)
	report := models.NewAlignmentReport()

	// Create a result with detailed validation errors
	result := models.NewAlignmentResult("test-operation")
	result.Status = models.StatusFailed

	// Add a failed validation detail
	detail := models.ValidationDetail{
		Type:          "postcondition",
		Expression:    `{\"==\": [{\"var\": \"response.status\"}, 200]}`, // Corrected escaping for inner quotes
		Expected:      200,
		Actual:        500,
		Message:       "Response status check failed",
		FailureReason: "Expected HTTP 200 but got HTTP 500",
		Suggestions:   []string{"Check if the service is returning the correct status code", "Verify the service implementation"},
		ContextInfo: map[string]interface{}{
			"span": map[string]interface{}{
				"name":   "test-span",
				"id":     "span-123",
				"status": models.SpanStatus{Code: "ERROR", Message: "Internal server error"},
			},
		},
	}

	result.AddValidationDetail(detail)
	report.AddResult(*result)

	output, err := renderer.RenderHuman(report)

	require.NoError(t, err)
	assert.Contains(t, output, "Response status check failed")
	assert.Contains(t, output, "期望: 200")
	assert.Contains(t, output, "实际: 500")
	assert.Contains(t, output, "失败原因: Expected HTTP 200 but got HTTP 500")
	assert.Contains(t, output, "建议:")
	assert.Contains(t, output, "Check if the service is returning the correct status code")
	assert.Contains(t, output, "上下文信息:")
	assert.Contains(t, output, "Span 名称: test-span")
}

func TestRenderJSON(t *testing.T) {
	renderer := NewReportRenderer()
	report := createTestReport(t, []models.AlignmentStatus{
		models.StatusSuccess,
		models.StatusFailed,
	})

	output, err := renderer.RenderJSON(report)

	require.NoError(t, err)
	assert.Contains(t, output, "\"summary\"")
	assert.Contains(t, output, "\"results\"")
	assert.Contains(t, output, "\"total\": 2")
	assert.Contains(t, output, "\"success\": 1")
	assert.Contains(t, output, "\"failed\": 1")

	// Verify it's valid JSON by checking structure
	assert.True(t, strings.HasPrefix(strings.TrimSpace(output), "{"))
	assert.True(t, strings.HasSuffix(strings.TrimSpace(output), "}"))

	// Verify JSON can be unmarshaled back to a report
	var unmarshaledReport models.AlignmentReport
	err = json.Unmarshal([]byte(output), &unmarshaledReport)
	require.NoError(t, err)
	assert.Equal(t, report.Summary.Total, unmarshaledReport.Summary.Total)
	assert.Equal(t, report.Summary.Success, unmarshaledReport.Summary.Success)
	assert.Equal(t, report.Summary.Failed, unmarshaledReport.Summary.Failed)
}

func TestRenderJSON_NilReport(t *testing.T) {
	renderer := NewReportRenderer()

	output, err := renderer.RenderJSON(nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "report cannot be nil")
	assert.Empty(t, output)
}

func TestRenderJSON_InvalidReport(t *testing.T) {
	renderer := NewReportRenderer()
	report := models.NewAlignmentReport()

	// Create an invalid report with inconsistent summary
	result := models.NewAlignmentResult("test-op")
	result.Status = models.StatusSuccess
	report.AddResult(*result)

	// Manually corrupt the summary to make it inconsistent
	report.Summary.Total = 999 // Wrong total

	output, err := renderer.RenderJSON(report)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "report validation failed")
	assert.Empty(t, output)
}

func TestValidateJSONOutput(t *testing.T) {
	renderer := NewReportRenderer()

	// Create a valid report and render it to get properly formatted JSON
	validReport := models.NewAlignmentReport()
	validJSON, err := renderer.RenderJSON(validReport)
	require.NoError(t, err)

	tests := []struct {
		name        string
		jsonOutput  string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid JSON",
			jsonOutput:  validJSON,
			expectError: false,
		},
		{
			name:        "empty JSON",
			jsonOutput:  "",
			expectError: true,
			errorMsg:    "JSON output is empty",
		},
		{
			name:        "malformed JSON",
			jsonOutput:  `{\"invalid\": json}`, // Corrected escaping for inner quotes
			expectError: true,
			errorMsg:    "JSON is not well-formed",
		},
		{
			name:        "valid JSON but wrong structure",
			jsonOutput:  `{\"wrong\": \"structure\"}`, // Corrected escaping for inner quotes
			expectError: true,
			errorMsg:    "JSON structure validation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := renderer.ValidateJSONOutput(tt.jsonOutput)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetJSONSchema(t *testing.T) {
	renderer := NewReportRenderer()

	schema := renderer.GetJSONSchema()

	assert.NotEmpty(t, schema)
	assert.Contains(t, schema, `"$schema"`)
	assert.Contains(t, schema, "\"title\": \"FlowSpec Alignment Report\"")
	assert.Contains(t, schema, "\"properties\"")
	assert.Contains(t, schema, "\"summary\"")
	assert.Contains(t, schema, "\"results\"")

	// Verify it's valid JSON
	var schemaObj interface{}
	err := json.Unmarshal([]byte(schema), &schemaObj)
	require.NoError(t, err)
}

func TestRenderJSONWithSchema(t *testing.T) {
	renderer := NewReportRenderer()
	report := createTestReport(t, []models.AlignmentStatus{models.StatusSuccess})

	tests := []struct {
		name          string
		includeSchema bool
		expectSchema  bool
	}{
		{
			name:          "without schema",
			includeSchema: false,
			expectSchema:  false,
		},
		{
			name:          "with schema",
			includeSchema: true,
			expectSchema:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := renderer.RenderJSONWithSchema(report, tt.includeSchema)

			require.NoError(t, err)
			assert.NotEmpty(t, output)

			if tt.expectSchema {
				assert.Contains(t, output, `"$schema"`)
				assert.Contains(t, output, "\"report\"")
			} else {
				assert.NotContains(t, output, `"$schema"`)
				assert.Contains(t, output, "\"summary\"")
			}

			// Verify it's valid JSON
			var jsonObj interface{}
			err = json.Unmarshal([]byte(output), &jsonObj)
			require.NoError(t, err)
		})
	}
}

func TestJSONFormatConsistency(t *testing.T) {
	renderer := NewReportRenderer()
	report := createTestReport(t, []models.AlignmentStatus{
		models.StatusSuccess,
		models.StatusFailed,
		models.StatusSkipped,
	})

	output, err := renderer.RenderJSON(report)
	require.NoError(t, err)

	// Verify JSON is properly formatted (multi-line)
	lines := strings.Split(output, "\n")
	assert.True(t, len(lines) > 1, "JSON should be multi-line formatted")

	// Verify the JSON validates against our own validation
	err = renderer.ValidateJSONOutput(output)
	assert.NoError(t, err)

	// Verify JSON can be parsed and re-marshaled
	var testReport models.AlignmentReport
	err = json.Unmarshal([]byte(output), &testReport)
	require.NoError(t, err)

	// Verify the unmarshaled report has the same key data
	assert.Equal(t, report.Summary.Total, testReport.Summary.Total)
	assert.Equal(t, len(report.Results), len(testReport.Results))
}

func TestGetExitCode(t *testing.T) {
	renderer := NewReportRenderer()

	tests := []struct {
		name     string
		report   *models.AlignmentReport
		expected int
	}{
		{
			name:     "nil report",
			report:   nil,
			expected: 2,
		},
		{
			name:     "successful report",
			report:   createTestReport(t, []models.AlignmentStatus{models.StatusSuccess}),
			expected: 0,
		},
		{
			name:     "failed report",
			report:   createTestReport(t, []models.AlignmentStatus{models.StatusFailed}),
			expected: 1,
		},
		{
			name:     "mixed report with failures",
			report:   createTestReport(t, []models.AlignmentStatus{models.StatusSuccess, models.StatusFailed}),
			expected: 1,
		},
		{
			name:     "skipped only report",
			report:   createTestReport(t, []models.AlignmentStatus{models.StatusSkipped}),
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exitCode := renderer.GetExitCode(tt.report)
			assert.Equal(t, tt.expected, exitCode)
		})
	}
}

func TestGetColor(t *testing.T) {
	// Test with colors enabled
	config := DefaultRendererConfig()
	config.ColorOutput = true
	renderer := NewReportRendererWithConfig(config)

	assert.Equal(t, "\033[0m", renderer.getColor("reset"))
	assert.Equal(t, "\033[1m", renderer.getColor("bold"))
	assert.Equal(t, "\033[31m", renderer.getColor("red"))
	assert.Equal(t, "\033[32m", renderer.getColor("green"))
	assert.Equal(t, "", renderer.getColor("nonexistent"))

	// Test with colors disabled
	config.ColorOutput = false
	renderer = NewReportRendererWithConfig(config)

	assert.Equal(t, "", renderer.getColor("reset"))
	assert.Equal(t, "", renderer.getColor("red"))
	assert.Equal(t, "", renderer.getColor("green"))
}

func TestGetStatusColor(t *testing.T) {
	renderer := NewReportRenderer()

	assert.Equal(t, "\033[32m", renderer.getStatusColor(models.StatusSuccess))
	assert.Equal(t, "\033[31m", renderer.getStatusColor(models.StatusFailed))
	assert.Equal(t, "\033[33m", renderer.getStatusColor(models.StatusSkipped))
	assert.Equal(t, "\033[0m", renderer.getStatusColor("invalid"))
}

func TestGetStatusIcon(t *testing.T) {
	renderer := NewReportRenderer()

	assert.Equal(t, "✅", renderer.getStatusIcon(models.StatusSuccess))
	assert.Equal(t, "❌", renderer.getStatusIcon(models.StatusFailed))
	assert.Equal(t, "⏭️", renderer.getStatusIcon(models.StatusSkipped))
	assert.Equal(t, "❓", renderer.getStatusIcon("invalid"))
}

// Helper function to create test reports
func createTestReport(t *testing.T, statuses []models.AlignmentStatus) *models.AlignmentReport {
	report := models.NewAlignmentReport()
	report.ExecutionTime = int64(time.Second)
	report.StartTime = time.Now().UnixNano()
	report.EndTime = report.StartTime + report.ExecutionTime

	for i, status := range statuses {
		result := models.NewAlignmentResult(fmt.Sprintf("operation-%d", i+1))
		result.Status = status
		result.ExecutionTime = int64(100 * time.Millisecond)
		result.MatchedSpans = []string{fmt.Sprintf("span-%d", i+1)}

		// Add some validation details based on status
		switch status {
		case models.StatusSuccess:
			detail := models.ValidationDetail{
				Type:     "postcondition",
				Expected: true,
				Actual:   true,
				Message:  "Assertion passed successfully",
			}
			result.AddValidationDetail(detail)
		case models.StatusFailed:
			detail := models.ValidationDetail{
				Type:          "postcondition",
				Expected:      200,
				Actual:        500,
				Message:       "Status code assertion failed",
				FailureReason: "Expected 200 but got 500",
			}
			result.AddValidationDetail(detail)
		case models.StatusSkipped:
			// No validation details for skipped
		}

		report.AddResult(*result)
	}

	return report
}