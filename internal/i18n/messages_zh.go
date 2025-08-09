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

package i18n

// chineseMessages contains all Chinese (Simplified) translations
var chineseMessages = map[string]string{
	// Report headers
	"report.title":                    "FlowSpec 验证报告",
	"report.summary":                  "📊 汇总统计",
	"report.details":                  "🔍 详细结果",
	"report.performance":              "⚡ 性能指标",
	
	// Summary statistics
	"summary.total":                   "总计: %d 个 ServiceSpec",
	"summary.success":                 "成功: %d 个",
	"summary.failed":                  "失败: %d 个", 
	"summary.skipped":                 "跳过: %d 个",
	"summary.success_rate":            "(%.1f%%)",
	
	// Performance metrics
	"performance.processing_rate":     "处理速度: %.2f specs/秒",
	"performance.memory_usage":        "内存使用: %.2f MB",
	"performance.concurrent_workers":  "并发工作线程: %d 个",
	"performance.assertions":          "断言评估: %d 个",
	"performance.execution_time":      "⏱️ 执行时间: %v",
	"performance.average_time":        "平均处理时间: %v/spec",
	
	// Result sections
	"results.failed":                  "❌ 失败的验证 (%d 个)",
	"results.success":                 "✅ 成功的验证 (%d 个)",
	"results.skipped":                 "⏭️ 跳过的验证 (%d 个)",
	
	// Result details
	"result.execution_time":           "⏱️ 执行时间: %v",
	"result.matched_span":             "🎯 匹配的 Span: %s",
	"result.assertion_stats":          "📊 断言统计: %d 总计, %d 通过, %d 失败",
	"result.preconditions":            "✅ 前置条件: (%d/%d 通过)",
	"result.postconditions":           "✅ 后置条件: (%d/%d 通过)",
	"result.no_matching_spans":        "🔍 未找到匹配的 Span",
	"result.span_matching":            "🔗 Span 匹配:",
	"result.no_matching_spans_for_op": "✅ No matching spans found for operation: %s",
	
	// Final status messages
	"status.success":                  "验证结果: ✅ 成功 (所有断言通过)",
	"status.failed":                   "验证结果: ❌ 失败 (%d 个断言失败)",
	"status.congratulations":          "🎉 恭喜！ 所有 %d 个 ServiceSpec 都符合预期规约。",
	"status.suggestions":              "💡 建议:",
	"status.suggestion.check_failed":  "• 检查失败的断言是否反映了实际的服务行为变化",
	"status.suggestion.verify_trace":  "• 验证轨迹数据是否包含预期的 span 属性和状态",
	"status.suggestion.update_specs":  "• 考虑更新 ServiceSpec 规约以匹配新的服务行为",
	
	// CLI messages
	"cli.starting_validation":         "开始执行对齐验证",
	"cli.parsing_specs":               "[1/4] 解析代码库中的 ServiceSpec 注解...",
	"cli.specs_parsed":                "✅ 成功解析 %d 个 ServiceSpec",
	"cli.parsing_warnings":            "⚠️ 解析过程中跳过了 %d 个错误的注解",
	"cli.ingesting_traces":            "[2/4] 摄取 OpenTelemetry 轨迹数据...",
	"cli.traces_ingested":             "✅ 成功摄取轨迹数据，包含 %d 个 span (TraceID: %s)",
	"cli.executing_alignment":         "[3/4] 执行规约与轨迹对齐验证...",
	"cli.processing_specs":            "处理 %d 个 ServiceSpec，预计需要 %ds...",
	"cli.alignment_completed":         "✅ 对齐验证完成，处理了 %d 个 ServiceSpec (耗时: %v)",
	"cli.validation_results":          "验证结果: 成功 %d, 失败 %d, 跳过 %d",
	"cli.generating_report":           "[4/4] 生成验证报告...",
	"cli.execution_completed":         "✅ 流程执行完成，总耗时: %v",
	"cli.all_validations_passed":      "🎉 所有验证通过，服务行为符合预期规约",
	"cli.validations_summary":         "成功验证了 %d 个 ServiceSpec，共 %d 个断言全部通过",
	"cli.validation_failed":           "验证失败：存在不符合规约的服务行为",
	"cli.failure_details":             "失败详情: %d 个 ServiceSpec 中有 %d 个断言失败",
	"cli.skipped_validations":         "跳过的验证: %d 个 ServiceSpec 因为找不到对应的轨迹数据",
	"cli.suggestion":                  "💡 提示：检查失败的断言，确认是服务行为变化还是规约需要更新",
	
	// Error messages
	"error.no_specs_found":            "未找到任何 ServiceSpec 注解",
	"error.parsing_errors":            "解析过程中发现的错误:",
	"error.validation_failure":        "验证失败：存在不符合规约的服务行为",
	
	// Assertion messages
	"assertion.passed":                "✅ %s assertion passed: %s",
	"assertion.failed":                "❌ %s assertion failed: %s",
	"assertion.precondition":          "Precondition",
	"assertion.postcondition":         "Postcondition",
	
	// Common terms
	"term.success":                    "SUCCESS",
	"term.failed":                     "FAILED", 
	"term.skipped":                    "SKIPPED",
	"term.passed":                     "通过",
	"term.failed_lower":               "失败",
}