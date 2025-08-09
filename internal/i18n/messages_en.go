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

// englishMessages contains all English translations
var englishMessages = map[string]string{
	// Report headers
	"report.title":       "FlowSpec Validation Report",
	"report.summary":     "📊 Summary Statistics",
	"report.details":     "🔍 Detailed Results",
	"report.performance": "⚡ Performance Metrics",

	// Summary statistics
	"summary.total":        "Total: %d ServiceSpecs",
	"summary.success":      "Success: %d",
	"summary.failed":       "Failed: %d",
	"summary.skipped":      "Skipped: %d",
	"summary.success_rate": "(%.1f%%)",

	// Performance metrics
	"performance.processing_rate":    "Processing Rate: %.2f specs/sec",
	"performance.memory_usage":       "Memory Usage: %.2f MB",
	"performance.concurrent_workers": "Concurrent Workers: %d",
	"performance.assertions":         "Assertions Evaluated: %d",
	"performance.execution_time":     "Execution Time: %v",
	"performance.average_time":       "Average Processing Time: %v/spec",

	// Result sections
	"results.failed":  "❌ Failed Validations (%d)",
	"results.success": "✅ Successful Validations (%d)",
	"results.skipped": "⏭️ Skipped Validations (%d)",

	// Result details
	"result.execution_time":           "Execution Time: %v",
	"result.matched_span":             "Matched Span: %s",
	"result.assertion_stats":          "Assertion Statistics: %d total, %d passed, %d failed",
	"result.preconditions":            "Preconditions: (%d/%d passed)",
	"result.postconditions":           "Postconditions: (%d/%d passed)",
	"result.no_matching_spans":        "No matching spans found",
	"result.span_matching":            "Span Matching:",
	"result.no_matching_spans_for_op": "No matching spans found for operation: %s",

	// Final status messages
	"status.success":                 "✅ Success (All assertions passed)",
	"status.failed":                  "❌ Failed (%d assertions failed)",
	"status.congratulations":         "🎉 Congratulations! All %d ServiceSpecs comply with expected specifications.",
	"status.suggestions":             "💡 Suggestions:",
	"status.suggestion.check_failed": "• Check failed assertions to see if they reflect actual service behavior changes",
	"status.suggestion.verify_trace": "• Verify trace data contains expected span attributes and status",
	"status.suggestion.update_specs": "• Consider updating ServiceSpec specifications to match new service behavior",

	// CLI messages
	"cli.starting_validation":    "Starting alignment validation",
	"cli.parsing_specs":          "[1/4] Parsing ServiceSpec annotations from codebase...",
	"cli.specs_parsed":           "✅ Successfully parsed %d ServiceSpecs",
	"cli.parsing_warnings":       "⚠️ Skipped %d invalid annotations during parsing",
	"cli.ingesting_traces":       "[2/4] Ingesting OpenTelemetry trace data...",
	"cli.traces_ingested":        "✅ Successfully ingested trace data with %d spans (TraceID: %s)",
	"cli.executing_alignment":    "[3/4] Executing specification-trace alignment validation...",
	"cli.processing_specs":       "Processing %d ServiceSpecs, estimated time: %ds...",
	"cli.alignment_completed":    "✅ Alignment validation completed, processed %d ServiceSpecs (took: %v)",
	"cli.validation_results":     "Validation results: success %d, failed %d, skipped %d",
	"cli.generating_report":      "[4/4] Generating validation report...",
	"cli.execution_completed":    "✅ Process execution completed, total time: %v",
	"cli.all_validations_passed": "🎉 All validations passed, service behavior meets expected specifications",
	"cli.validations_summary":    "Successfully validated %d ServiceSpecs with %d assertions all passed",
	"cli.validation_failed":      "Validation failed: service behavior does not comply with specifications",
	"cli.failure_details":        "Failure details: %d assertions failed out of %d ServiceSpecs",
	"cli.skipped_validations":    "Skipped validations: %d ServiceSpecs because corresponding trace data was not found",
	"cli.suggestion":             "💡 Tip: Check failed assertions to confirm whether it's a service behavior change or specification needs updating",

	// Error messages
	"error.no_specs_found":     "no ServiceSpecs found in source path: %s",
	"error.parsing_errors":     "Parsing errors encountered:",
	"error.validation_failure": "Validation failure: non-compliant service behavior exists",

	// Assertion messages
	"assertion.passed":        "✅ %s assertion passed: %s",
	"assertion.failed":        "❌ %s assertion failed: %s",
	"assertion.precondition":  "Precondition",
	"assertion.postcondition": "Postcondition",

	// Common terms
	"term.success":      "SUCCESS",
	"term.failed":       "FAILED",
	"term.skipped":      "SKIPPED",
	"term.passed":       "passed",
	"term.failed_lower": "failed",
}
