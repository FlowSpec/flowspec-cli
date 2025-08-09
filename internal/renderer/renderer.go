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
	"time"

	"github.com/flowspec/flowspec-cli/internal/i18n"
	"github.com/flowspec/flowspec-cli/internal/models"
)

// Icon constants for consistent display
const (
	IconSuccess = "✅"
	IconFailed  = "❌"
	IconSkipped = "⏭️"
)

// ReportRenderer defines the interface for rendering alignment reports
type ReportRenderer interface {
	RenderHuman(report *models.AlignmentReport) (string, error)
	RenderJSON(report *models.AlignmentReport) (string, error)
	GetExitCode(report *models.AlignmentReport) int
	SetLanguage(lang i18n.SupportedLanguage)
	GetLanguage() i18n.SupportedLanguage
}

// DefaultReportRenderer implements the ReportRenderer interface
type DefaultReportRenderer struct {
	config    *RendererConfig
	localizer *i18n.Localizer
}

// RendererConfig holds configuration for the report renderer
type RendererConfig struct {
	ShowTimestamps     bool
	ShowPerformance    bool
	ShowDetailedErrors bool
	ColorOutput        bool
}

// DefaultRendererConfig returns a default renderer configuration
func DefaultRendererConfig() *RendererConfig {
	return &RendererConfig{
		ShowTimestamps:     true,
		ShowPerformance:    true,
		ShowDetailedErrors: true,
		ColorOutput:        true,
	}
}

// NewReportRenderer creates a new report renderer with default configuration
func NewReportRenderer() *DefaultReportRenderer {
	return &DefaultReportRenderer{
		config:    DefaultRendererConfig(),
		localizer: i18n.NewLocalizerFromEnv(),
	}
}

// NewReportRendererWithConfig creates a new report renderer with custom configuration
func NewReportRendererWithConfig(config *RendererConfig) *DefaultReportRenderer {
	return &DefaultReportRenderer{
		config:    config,
		localizer: i18n.NewLocalizerFromEnv(),
	}
}

// SetLanguage sets the language for the renderer
func (r *DefaultReportRenderer) SetLanguage(lang i18n.SupportedLanguage) {
	r.localizer.SetLanguage(lang)
}

// GetLanguage returns the current language
func (r *DefaultReportRenderer) GetLanguage() i18n.SupportedLanguage {
	return r.localizer.GetLanguage()
}

// RenderHuman implements the ReportRenderer interface with enhanced formatting and color support
func (r *DefaultReportRenderer) RenderHuman(report *models.AlignmentReport) (string, error) {
	var output strings.Builder

	// Header with enhanced styling
	r.writeColoredHeader(&output, "FlowSpec 验证报告")
	output.WriteString("==================================================\n\n")

	// Summary statistics with color coding
	r.writeColoredSection(&output, "📊 汇总统计")
	output.WriteString(fmt.Sprintf("  总计: %s%d%s 个 ServiceSpec\n",
		r.getColor("bold"), report.Summary.Total, r.getColor("reset")))

	// Success count with green color
	output.WriteString(fmt.Sprintf("  %s%s 成功: %s%d%s 个%s",
		r.getColor("green"), IconSuccess, r.getColor("bold"), report.Summary.Success, r.getColor("reset"), r.getColor("reset")))
	if report.Summary.Total > 0 {
		successRate := float64(report.Summary.Success) / float64(report.Summary.Total) * 100
		output.WriteString(fmt.Sprintf(" (%.1f%%)", successRate))
	}
	output.WriteString("\n")

	// Failed count with red color
	if report.Summary.Failed > 0 {
		output.WriteString(fmt.Sprintf("  %s❌ 失败: %s%d%s 个%s",
			r.getColor("red"), r.getColor("bold"), report.Summary.Failed, r.getColor("reset"), r.getColor("reset")))
		if report.Summary.Total > 0 {
			failureRate := float64(report.Summary.Failed) / float64(report.Summary.Total) * 100
			output.WriteString(fmt.Sprintf(" (%.1f%%)", failureRate))
		}
		output.WriteString("\n")
	} else {
		output.WriteString(fmt.Sprintf("  %s❌ 失败: %s0%s 个%s\n",
			r.getColor("dim"), r.getColor("dim"), r.getColor("reset"), r.getColor("reset")))
	}

	// Skipped count with yellow color
	if report.Summary.Skipped > 0 {
		output.WriteString(fmt.Sprintf("  %s⏭️  跳过: %s%d%s 个%s",
			r.getColor("yellow"), r.getColor("bold"), report.Summary.Skipped, r.getColor("reset"), r.getColor("reset")))
		if report.Summary.Total > 0 {
			skipRate := float64(report.Summary.Skipped) / float64(report.Summary.Total) * 100
			output.WriteString(fmt.Sprintf(" (%.1f%%)", skipRate))
		}
		output.WriteString("\n")
	} else {
		output.WriteString(fmt.Sprintf("  %s⏭️  跳过: %s0%s 个%s\n",
			r.getColor("dim"), r.getColor("dim"), r.getColor("reset"), r.getColor("reset")))
	}

	// Performance metrics with enhanced formatting
	if r.config.ShowPerformance && report.PerformanceInfo.SpecsProcessed > 0 {
		output.WriteString("\n")
		r.writeColoredSubsection(&output, "⚡ 性能指标")
		output.WriteString(fmt.Sprintf("  处理速度: %s%.2f%s specs/秒\n",
			r.getColor("cyan"), report.PerformanceInfo.ProcessingRate, r.getColor("reset")))
		output.WriteString(fmt.Sprintf("  内存使用: %s%.2f%s MB\n",
			r.getColor("cyan"), report.PerformanceInfo.MemoryUsageMB, r.getColor("reset")))
		if report.PerformanceInfo.ConcurrentWorkers > 0 {
			output.WriteString(fmt.Sprintf("  并发工作线程: %s%d%s 个\n",
				r.getColor("cyan"), report.PerformanceInfo.ConcurrentWorkers, r.getColor("reset")))
		}
		if report.Summary.TotalAssertions > 0 {
			output.WriteString(fmt.Sprintf("  断言评估: %s%d%s 个\n",
				r.getColor("cyan"), report.Summary.TotalAssertions, r.getColor("reset")))
		}
	}

	// Execution time with enhanced formatting
	if r.config.ShowTimestamps {
		executionTime := time.Duration(report.ExecutionTime)
		output.WriteString(fmt.Sprintf("  ⏱️  执行时间: %s%v%s\n",
			r.getColor("magenta"), executionTime, r.getColor("reset")))

		// Show average time per spec if meaningful
		if report.Summary.Total > 0 {
			avgTime := time.Duration(report.Summary.AverageExecutionTime)
			output.WriteString(fmt.Sprintf("  平均处理时间: %s%v%s/spec\n",
				r.getColor("magenta"), avgTime, r.getColor("reset")))
		}
	}

	output.WriteString("\n")
	r.writeColoredSection(&output, "🔍 详细结果")
	output.WriteString("──────────────────────────────────────────────────\n\n")

	// Group results by status for better readability
	successResults := []models.AlignmentResult{}
	failedResults := []models.AlignmentResult{}
	skippedResults := []models.AlignmentResult{}

	for _, result := range report.Results {
		switch result.Status {
		case models.StatusSuccess:
			successResults = append(successResults, result)
		case models.StatusFailed:
			failedResults = append(failedResults, result)
		case models.StatusSkipped:
			skippedResults = append(skippedResults, result)
		}
	}

	// Render failed results first (most important)
	if len(failedResults) > 0 {
		r.writeColoredSubsection(&output, fmt.Sprintf("❌ 失败的验证 (%d 个)", len(failedResults)))
		for i, result := range failedResults {
			r.renderResultHuman(&output, result, i+1, len(failedResults))
			if i < len(failedResults)-1 {
				output.WriteString("\n")
			}
		}
		output.WriteString("\n")
	}

	// Render successful results
	if len(successResults) > 0 {
		r.writeColoredSubsection(&output, fmt.Sprintf("%s 成功的验证 (%d 个)", IconSuccess, len(successResults)))
		for i, result := range successResults {
			r.renderResultHuman(&output, result, i+1, len(successResults))
			if i < len(successResults)-1 {
				output.WriteString("\n")
			}
		}
		output.WriteString("\n")
	}

	// Render skipped results last
	if len(skippedResults) > 0 {
		r.writeColoredSubsection(&output, fmt.Sprintf("⏭️  跳过的验证 (%d 个)", len(skippedResults)))
		for i, result := range skippedResults {
			r.renderResultHuman(&output, result, i+1, len(skippedResults))
			if i < len(skippedResults)-1 {
				output.WriteString("\n")
			}
		}
		output.WriteString("\n")
	}

	// Final summary with enhanced styling
	output.WriteString("==================================================\n")
	if report.HasFailures() {
		output.WriteString(fmt.Sprintf("%s验证结果: ❌ 失败%s (%s%d%s 个断言失败)\n",
			r.getColor("red"), r.getColor("reset"),
			r.getColor("bold"), report.Summary.FailedAssertions, r.getColor("reset")))

		// Provide actionable summary for failures
		if report.Summary.FailedAssertions > 0 {
			output.WriteString(fmt.Sprintf("\n%s💡 建议:%s\n", r.getColor("yellow"), r.getColor("reset")))
			output.WriteString("  • 检查失败的断言是否反映了实际的服务行为变化\n")
			output.WriteString("  • 验证轨迹数据是否包含预期的 span 属性和状态\n")
			output.WriteString("  • 考虑更新 ServiceSpec 规约以匹配新的服务行为\n")
		}
	} else {
		output.WriteString(fmt.Sprintf("%s验证结果: %s 成功%s (所有断言通过)\n",
			r.getColor("green"), IconSuccess, r.getColor("reset")))

		if report.Summary.Total > 0 {
			output.WriteString(fmt.Sprintf("\n%s🎉 恭喜！%s 所有 %d 个 ServiceSpec 都符合预期规约。\n",
				r.getColor("green"), r.getColor("reset"), report.Summary.Total))
		}
	}

	return output.String(), nil
}

// renderResultHuman renders a single alignment result in human format with enhanced styling
func (r *DefaultReportRenderer) renderResultHuman(output *strings.Builder, result models.AlignmentResult, index, total int) {
	// Status icon and operation ID with color coding
	statusIcon := r.getStatusIcon(result.Status)
	statusColor := r.getStatusColor(result.Status)

	output.WriteString(fmt.Sprintf("%s[%d/%d]%s %s %s%s%s (%s%s%s)\n",
		r.getColor("dim"), index, total, r.getColor("reset"),
		statusIcon,
		r.getColor("bold"), result.SpecOperationID, r.getColor("reset"),
		statusColor, result.Status, r.getColor("reset")))

	// Execution time with formatting
	if r.config.ShowTimestamps {
		executionTime := time.Duration(result.ExecutionTime)
		output.WriteString(fmt.Sprintf("   ⏱️  执行时间: %s%v%s\n",
			r.getColor("dim"), executionTime, r.getColor("reset")))
	}

	// Matched spans with enhanced formatting
	if len(result.MatchedSpans) > 0 {
		output.WriteString(fmt.Sprintf("   🎯 匹配的 Span: %s%s%s\n",
			r.getColor("cyan"), strings.Join(result.MatchedSpans, ", "), r.getColor("reset")))
	} else if result.Status == models.StatusSkipped {
		output.WriteString(fmt.Sprintf("   %s🔍 未找到匹配的 Span%s\n",
			r.getColor("yellow"), r.getColor("reset")))
	}

	// Assertion summary with color coding
	if result.AssertionsTotal > 0 {
		passedColor := r.getColor("green")
		failedColor := r.getColor("red")
		if result.AssertionsPassed == 0 {
			passedColor = r.getColor("dim")
		}
		if result.AssertionsFailed == 0 {
			failedColor = r.getColor("dim")
		}

		output.WriteString(fmt.Sprintf("   📊 断言统计: %s%d%s 总计, %s%d%s 通过, %s%d%s 失败\n",
			r.getColor("bold"), result.AssertionsTotal, r.getColor("reset"),
			passedColor, result.AssertionsPassed, r.getColor("reset"),
			failedColor, result.AssertionsFailed, r.getColor("reset")))
	}

	// Error message for failed results with enhanced formatting
	if result.Status == models.StatusFailed && result.ErrorMessage != "" {
		output.WriteString(fmt.Sprintf("   %s⚠️  错误信息:%s %s\n",
			r.getColor("red"), r.getColor("reset"), result.ErrorMessage))
	}

	// Detailed validation results with improved readability
	if r.config.ShowDetailedErrors && len(result.Details) > 0 {
		r.renderValidationDetailsHuman(output, result.Details)
	}
}

// renderValidationDetailsHuman renders validation details in human format with enhanced styling
func (r *DefaultReportRenderer) renderValidationDetailsHuman(output *strings.Builder, details []models.ValidationDetail) {
	preconditions := []models.ValidationDetail{}
	postconditions := []models.ValidationDetail{}
	matchingDetails := []models.ValidationDetail{}

	for _, detail := range details {
		switch detail.Type {
		case "precondition":
			preconditions = append(preconditions, detail)
		case "postcondition":
			postconditions = append(postconditions, detail)
		case "matching":
			matchingDetails = append(matchingDetails, detail)
		}
	}

	// Render matching details first (if any)
	if len(matchingDetails) > 0 {
		output.WriteString(fmt.Sprintf("   %s🔗 Span 匹配:%s\n",
			r.getColor("cyan"), r.getColor("reset")))
		for _, detail := range matchingDetails {
			r.renderValidationDetailHuman(output, detail, "     ")
		}
	}

	// Render preconditions
	if len(preconditions) > 0 {
		passedCount := 0
		for _, detail := range preconditions {
			if detail.IsPassed() {
				passedCount++
			}
		}

		statusIcon := IconSuccess
		statusColor := r.getColor("green")
		if passedCount < len(preconditions) {
			statusIcon = IconFailed
			statusColor = r.getColor("red")
		}

		output.WriteString(fmt.Sprintf("   %s%s 前置条件:%s %s(%d/%d 通过)%s\n",
			statusColor, statusIcon, r.getColor("reset"),
			r.getColor("dim"), passedCount, len(preconditions), r.getColor("reset")))

		for _, detail := range preconditions {
			r.renderValidationDetailHuman(output, detail, "     ")
		}
	}

	// Render postconditions
	if len(postconditions) > 0 {
		passedCount := 0
		for _, detail := range postconditions {
			if detail.IsPassed() {
				passedCount++
			}
		}

		statusIcon := IconSuccess
		statusColor := r.getColor("green")
		if passedCount < len(postconditions) {
			statusIcon = IconFailed
			statusColor = r.getColor("red")
		}

		output.WriteString(fmt.Sprintf("   %s%s 后置条件:%s %s(%d/%d 通过)%s\n",
			statusColor, statusIcon, r.getColor("reset"),
			r.getColor("dim"), passedCount, len(postconditions), r.getColor("reset")))

		for _, detail := range postconditions {
			r.renderValidationDetailHuman(output, detail, "     ")
		}
	}
}

// renderValidationDetailHuman renders a single validation detail in human format with enhanced styling
func (r *DefaultReportRenderer) renderValidationDetailHuman(output *strings.Builder, detail models.ValidationDetail, indent string) {
	icon := IconSuccess
	iconColor := r.getColor("green")
	if !detail.IsPassed() {
		icon = IconFailed
		iconColor = r.getColor("red")
	}

	// Render the main message with color coding
	output.WriteString(fmt.Sprintf("%s%s%s%s %s\n",
		indent, iconColor, icon, r.getColor("reset"), detail.Message))

	// Show detailed information for failed assertions
	if !detail.IsPassed() && r.config.ShowDetailedErrors {
		// Expression details
		if detail.Expression != "" {
			output.WriteString(fmt.Sprintf("%s   %s表达式:%s %s%s%s\n",
				indent, r.getColor("dim"), r.getColor("reset"),
				r.getColor("cyan"), detail.Expression, r.getColor("reset")))
		}

		// Expected vs Actual with enhanced formatting
		output.WriteString(fmt.Sprintf("%s   %s期望:%s %s%v%s %s(%T)%s\n",
			indent, r.getColor("green"), r.getColor("reset"),
			r.getColor("bold"), detail.Expected, r.getColor("reset"),
			r.getColor("dim"), detail.Expected, r.getColor("reset")))

		output.WriteString(fmt.Sprintf("%s   %s实际:%s %s%v%s %s(%T)%s\n",
			indent, r.getColor("red"), r.getColor("reset"),
			r.getColor("bold"), detail.Actual, r.getColor("reset"),
			r.getColor("dim"), detail.Actual, r.getColor("reset")))

		// Failure reason with enhanced formatting
		if detail.FailureReason != "" {
			output.WriteString(fmt.Sprintf("%s   %s💡 失败原因:%s %s\n",
				indent, r.getColor("yellow"), r.getColor("reset"), detail.FailureReason))
		}

		// Context information (if available)
		if len(detail.ContextInfo) > 0 {
			output.WriteString(fmt.Sprintf("%s   %s🔍 上下文信息:%s\n",
				indent, r.getColor("cyan"), r.getColor("reset")))

			// Show relevant span information
			if spanInfo, ok := detail.ContextInfo["span"].(map[string]interface{}); ok {
				if spanName, ok := spanInfo["name"].(string); ok {
					output.WriteString(fmt.Sprintf("%s     Span 名称: %s%s%s\n",
						indent, r.getColor("cyan"), spanName, r.getColor("reset")))
				}
				if spanID, ok := spanInfo["id"].(string); ok {
					output.WriteString(fmt.Sprintf("%s     Span ID: %s%s%s\n",
						indent, r.getColor("dim"), spanID, r.getColor("reset")))
				}
				if status, ok := spanInfo["status"].(models.SpanStatus); ok {
					statusColor := r.getColor("green")
					if status.Code == "ERROR" {
						statusColor = r.getColor("red")
					}
					output.WriteString(fmt.Sprintf("%s     状态: %s%s%s",
						indent, statusColor, status.Code, r.getColor("reset")))
					if status.Message != "" {
						output.WriteString(fmt.Sprintf(" - %s", status.Message))
					}
					output.WriteString("\n")
				}
			}
		}

		// Actionable suggestions with enhanced formatting
		if len(detail.Suggestions) > 0 {
			output.WriteString(fmt.Sprintf("%s   %s💡 建议:%s\n",
				indent, r.getColor("yellow"), r.getColor("reset")))
			for i, suggestion := range detail.Suggestions {
				output.WriteString(fmt.Sprintf("%s     %s%d.%s %s\n",
					indent, r.getColor("dim"), i+1, r.getColor("reset"), suggestion))
			}
		}

		// Add separator for readability
		if detail.FailureReason != "" || len(detail.Suggestions) > 0 {
			output.WriteString(fmt.Sprintf("%s   %s%s%s\n",
				indent, r.getColor("dim"), strings.Repeat("─", 40), r.getColor("reset")))
		}
	}
}

// getStatusIcon returns an icon for the given alignment status
func (r *DefaultReportRenderer) getStatusIcon(status models.AlignmentStatus) string {
	switch status {
	case models.StatusSuccess:
		return IconSuccess
	case models.StatusFailed:
		return IconFailed
	case models.StatusSkipped:
		return IconSkipped
	default:
		return "❓"
	}
}

// RenderJSON implements the ReportRenderer interface with enhanced JSON formatting and validation
func (r *DefaultReportRenderer) RenderJSON(report *models.AlignmentReport) (string, error) {
	if report == nil {
		return "", fmt.Errorf("report cannot be nil")
	}

	// Validate report completeness before rendering
	if err := r.validateReportCompleteness(report); err != nil {
		return "", fmt.Errorf("report validation failed: %w", err)
	}

	// Create a structured JSON output with consistent formatting
	jsonData, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal report to JSON: %w", err)
	}

	// Validate that the generated JSON is well-formed
	var testUnmarshal interface{}
	if err := json.Unmarshal(jsonData, &testUnmarshal); err != nil {
		return "", fmt.Errorf("generated JSON is malformed: %w", err)
	}

	return string(jsonData), nil
}

// validateReportCompleteness validates that the report has all required fields
func (r *DefaultReportRenderer) validateReportCompleteness(report *models.AlignmentReport) error {
	// Check if this looks like a valid AlignmentReport structure
	// This is a basic structural check - if it unmarshaled successfully but doesn't have
	// the expected fields, it's likely the wrong structure

	// Check for required top-level fields by checking if they have reasonable values
	// An empty/default AlignmentReport is still valid, but a completely wrong structure should fail

	// If Results is nil but Summary.Total > 0, that's inconsistent
	if report.Results == nil && report.Summary.Total > 0 {
		return fmt.Errorf("results is nil but summary indicates %d total specs", report.Summary.Total)
	}

	// Validate summary exists and has consistent data
	if report.Summary.Total != len(report.Results) {
		return fmt.Errorf("summary total (%d) doesn't match results count (%d)",
			report.Summary.Total, len(report.Results))
	}

	// Count actual statuses to verify summary accuracy
	actualSuccess := 0
	actualFailed := 0
	actualSkipped := 0

	for i, result := range report.Results {
		// Validate each result has required fields
		if result.SpecOperationID == "" {
			return fmt.Errorf("result[%d] missing specOperationId", i)
		}

		if !result.Status.IsValid() {
			return fmt.Errorf("result[%d] has invalid status: %s", i, result.Status)
		}

		// Count statuses
		switch result.Status {
		case models.StatusSuccess:
			actualSuccess++
		case models.StatusFailed:
			actualFailed++
		case models.StatusSkipped:
			actualSkipped++
		}

		// Validate assertion counts are consistent
		if result.AssertionsTotal != result.AssertionsPassed+result.AssertionsFailed {
			return fmt.Errorf("result[%d] assertion counts inconsistent: total=%d, passed=%d, failed=%d",
				i, result.AssertionsTotal, result.AssertionsPassed, result.AssertionsFailed)
		}
	}

	// Validate summary counts match actual counts
	if report.Summary.Success != actualSuccess {
		return fmt.Errorf("summary success count (%d) doesn't match actual (%d)",
			report.Summary.Success, actualSuccess)
	}
	if report.Summary.Failed != actualFailed {
		return fmt.Errorf("summary failed count (%d) doesn't match actual (%d)",
			report.Summary.Failed, actualFailed)
	}
	if report.Summary.Skipped != actualSkipped {
		return fmt.Errorf("summary skipped count (%d) doesn't match actual (%d)",
			report.Summary.Skipped, actualSkipped)
	}

	// Validate timing information
	if report.ExecutionTime < 0 {
		return fmt.Errorf("execution time cannot be negative: %d", report.ExecutionTime)
	}

	if report.StartTime > 0 && report.EndTime > 0 && report.EndTime < report.StartTime {
		return fmt.Errorf("end time (%d) cannot be before start time (%d)",
			report.EndTime, report.StartTime)
	}

	return nil
}

// GetJSONSchema returns the JSON schema for the alignment report
func (r *DefaultReportRenderer) GetJSONSchema() string {
	return `{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "$id": "https://flowspec.dev/schemas/alignment-report.json",
  "title": "FlowSpec Alignment Report",
  "description": "Schema for FlowSpec alignment verification reports",
  "type": "object",
  "required": ["summary", "results", "executionTime", "startTime", "endTime"],
  "properties": {
    "summary": {
      "type": "object",
      "required": ["total", "success", "failed", "skipped", "successRate", "failureRate", "skipRate"],
      "properties": {
        "total": {"type": "integer", "minimum": 0},
        "success": {"type": "integer", "minimum": 0},
        "failed": {"type": "integer", "minimum": 0},
        "skipped": {"type": "integer", "minimum": 0},
        "successRate": {"type": "number", "minimum": 0, "maximum": 1},
        "failureRate": {"type": "number", "minimum": 0, "maximum": 1},
        "skipRate": {"type": "number", "minimum": 0, "maximum": 1},
        "averageExecutionTime": {"type": "integer", "minimum": 0},
        "totalAssertions": {"type": "integer", "minimum": 0},
        "failedAssertions": {"type": "integer", "minimum": 0}
      }
    },
    "results": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["specOperationId", "status", "details", "executionTime"],
        "properties": {
          "specOperationId": {"type": "string", "minLength": 1},
          "status": {"type": "string", "enum": ["SUCCESS", "FAILED", "SKIPPED"]},
          "details": {
            "type": "array",
            "items": {
              "type": "object",
              "required": ["type", "expression", "expected", "actual", "message"],
              "properties": {
                "type": {"type": "string", "enum": ["precondition", "postcondition", "matching"]},
                "expression": {"type": "string"},
                "expected": {},
                "actual": {},
                "message": {"type": "string"},
                "failureReason": {"type": "string"},
                "suggestions": {"type": "array", "items": {"type": "string"}},
                "contextInfo": {"type": "object"}
              }
            }
          },
          "executionTime": {"type": "integer", "minimum": 0},
          "startTime": {"type": "integer", "minimum": 0},
          "endTime": {"type": "integer", "minimum": 0},
          "matchedSpans": {"type": "array", "items": {"type": "string"}},
          "assertionsTotal": {"type": "integer", "minimum": 0},
          "assertionsPassed": {"type": "integer", "minimum": 0},
          "assertionsFailed": {"type": "integer", "minimum": 0},
          "errorMessage": {"type": "string"}
        }
      }
    },
    "executionTime": {"type": "integer", "minimum": 0},
    "startTime": {"type": "integer", "minimum": 0},
    "endTime": {"type": "integer", "minimum": 0},
    "performanceInfo": {
      "type": "object",
      "properties": {
        "specsProcessed": {"type": "integer", "minimum": 0},
        "spansMatched": {"type": "integer", "minimum": 0},
        "assertionsEvaluated": {"type": "integer", "minimum": 0},
        "concurrentWorkers": {"type": "integer", "minimum": 1},
        "memoryUsageMB": {"type": "number", "minimum": 0},
        "processingRate": {"type": "number", "minimum": 0}
      }
    }
  }
}`
}

// ValidateJSONOutput validates that the JSON output conforms to the schema
func (r *DefaultReportRenderer) ValidateJSONOutput(jsonOutput string) error {
	// Additional JSON-specific validations
	if len(jsonOutput) == 0 {
		return fmt.Errorf("JSON output is empty")
	}

	// First, ensure it's valid JSON
	var genericJSON map[string]interface{}
	if err := json.Unmarshal([]byte(jsonOutput), &genericJSON); err != nil {
		return fmt.Errorf("JSON is not well-formed: %w", err)
	}

	// Check for required top-level fields that should exist in an AlignmentReport
	requiredFields := []string{"summary", "results", "executionTime", "startTime", "endTime"}
	for _, field := range requiredFields {
		if _, exists := genericJSON[field]; !exists {
			return fmt.Errorf("JSON structure validation failed: missing required field '%s'", field)
		}
	}

	// Try to parse as an AlignmentReport
	var report models.AlignmentReport
	if err := json.Unmarshal([]byte(jsonOutput), &report); err != nil {
		return fmt.Errorf("JSON structure validation failed: cannot unmarshal as AlignmentReport: %w", err)
	}

	// Validate the structure matches our expectations
	if err := r.validateReportCompleteness(&report); err != nil {
		return fmt.Errorf("JSON structure validation failed: %w", err)
	}

	// Just verify that it can be re-marshaled (structure is valid)
	_, err := json.MarshalIndent(genericJSON, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to format JSON for validation: %w", err)
	}

	return nil
}

// RenderJSONWithSchema renders JSON output with optional schema inclusion
func (r *DefaultReportRenderer) RenderJSONWithSchema(report *models.AlignmentReport, includeSchema bool) (string, error) {
	jsonOutput, err := r.RenderJSON(report)
	if err != nil {
		return "", err
	}

	if !includeSchema {
		return jsonOutput, nil
	}

	// Create a wrapper object that includes both the schema and the report
	wrapper := map[string]interface{}{
		"$schema": "https://flowspec.dev/schemas/alignment-report.json",
		"report":  report,
	}

	wrapperJSON, err := json.MarshalIndent(wrapper, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal wrapper JSON: %w", err)
	}

	return string(wrapperJSON), nil
}

// GetExitCode implements the ReportRenderer interface
func (r *DefaultReportRenderer) GetExitCode(report *models.AlignmentReport) int {
	if report == nil {
		return 2 // System error
	}

	if report.HasFailures() {
		return 1 // Validation failures
	}

	return 0 // Success
}

// Color support methods

// getColor returns ANSI color codes if color output is enabled
func (r *DefaultReportRenderer) getColor(colorName string) string {
	if !r.config.ColorOutput {
		return ""
	}

	colors := map[string]string{
		"reset":   "\033[0m",
		"bold":    "\033[1m",
		"dim":     "\033[2m",
		"red":     "\033[31m",
		"green":   "\033[32m",
		"yellow":  "\033[33m",
		"blue":    "\033[34m",
		"magenta": "\033[35m",
		"cyan":    "\033[36m",
		"white":   "\033[37m",
	}

	if color, exists := colors[colorName]; exists {
		return color
	}
	return ""
}

// getStatusColor returns the appropriate color for a given status
func (r *DefaultReportRenderer) getStatusColor(status models.AlignmentStatus) string {
	switch status {
	case models.StatusSuccess:
		return r.getColor("green")
	case models.StatusFailed:
		return r.getColor("red")
	case models.StatusSkipped:
		return r.getColor("yellow")
	default:
		return r.getColor("reset")
	}
}

// writeColoredHeader writes a colored header section
func (r *DefaultReportRenderer) writeColoredHeader(output *strings.Builder, text string) {
	output.WriteString(fmt.Sprintf("%s%s%s%s\n",
		r.getColor("bold"), r.getColor("blue"), text, r.getColor("reset")))
}

// writeColoredSection writes a colored section header
func (r *DefaultReportRenderer) writeColoredSection(output *strings.Builder, text string) {
	output.WriteString(fmt.Sprintf("%s%s%s\n",
		r.getColor("bold"), text, r.getColor("reset")))
}

// writeColoredSubsection writes a colored subsection header
func (r *DefaultReportRenderer) writeColoredSubsection(output *strings.Builder, text string) {
	output.WriteString(fmt.Sprintf("%s%s%s\n",
		r.getColor("cyan"), text, r.getColor("reset")))
}
