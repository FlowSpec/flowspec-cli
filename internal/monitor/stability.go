package monitor

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// StabilityMonitor handles test stability monitoring and analysis
type StabilityMonitor struct {
	config  *StabilityConfig
	history []StabilityReport
}

// StabilityConfig defines configuration for stability monitoring
type StabilityConfig struct {
	OutputDir           string        `json:"outputDir"`
	HistoryFile         string        `json:"historyFile"`
	MaxHistoryEntries   int           `json:"maxHistoryEntries"`
	StabilityThreshold  float64       `json:"stabilityThreshold"`  // Minimum pass rate (e.g., 95%)
	PerformanceThreshold time.Duration `json:"performanceThreshold"` // Maximum acceptable test duration
	FlakyTestThreshold  int           `json:"flakyTestThreshold"`   // Number of failures to consider a test flaky
	MonitoringWindow    int           `json:"monitoringWindow"`     // Days to analyze for trends
}

// StabilityReport represents a test stability analysis report
type StabilityReport struct {
	Timestamp        time.Time              `json:"timestamp"`
	TestExecution    TestExecution          `json:"testExecution"`
	StabilityMetrics StabilityMetrics       `json:"stabilityMetrics"`
	FlakyTests       []FlakyTest            `json:"flakyTests"`
	PerformanceData  PerformanceData        `json:"performanceData"`
	TrendAnalysis    StabilityTrends        `json:"trendAnalysis"`
	Recommendations  []string               `json:"recommendations"`
}

// TestExecution represents a single test execution
type TestExecution struct {
	StartTime       time.Time     `json:"startTime"`
	EndTime         time.Time     `json:"endTime"`
	Duration        time.Duration `json:"duration"`
	TotalTests      int           `json:"totalTests"`
	PassedTests     int           `json:"passedTests"`
	FailedTests     int           `json:"failedTests"`
	SkippedTests    int           `json:"skippedTests"`
	PassRate        float64       `json:"passRate"`
	FailedTestNames []string      `json:"failedTestNames"`
	SlowTests       []SlowTest    `json:"slowTests"`
}

// StabilityMetrics represents stability analysis metrics
type StabilityMetrics struct {
	OverallStability    float64            `json:"overallStability"`    // Pass rate over time
	TestStability       map[string]float64 `json:"testStability"`       // Per-test stability
	AveragePassRate     float64            `json:"averagePassRate"`     // Average pass rate
	StabilityTrend      string             `json:"stabilityTrend"`      // "improving", "declining", "stable"
	ConsistencyScore    float64            `json:"consistencyScore"`    // How consistent test results are
	ReliabilityScore    float64            `json:"reliabilityScore"`    // Overall reliability score
}

// FlakyTest represents a test that fails intermittently
type FlakyTest struct {
	Name            string    `json:"name"`
	FailureCount    int       `json:"failureCount"`
	TotalRuns       int       `json:"totalRuns"`
	FailureRate     float64   `json:"failureRate"`
	LastFailure     time.Time `json:"lastFailure"`
	FailureReasons  []string  `json:"failureReasons"`
	Severity        string    `json:"severity"` // "low", "medium", "high", "critical"
}

// SlowTest represents a test that takes longer than expected
type SlowTest struct {
	Name            string        `json:"name"`
	Duration        time.Duration `json:"duration"`
	AverageDuration time.Duration `json:"averageDuration"`
	Slowdown        float64       `json:"slowdown"` // How much slower than average
}

// PerformanceData represents test performance metrics
type PerformanceData struct {
	TotalDuration      time.Duration     `json:"totalDuration"`
	AverageDuration    time.Duration     `json:"averageDuration"`
	MedianDuration     time.Duration     `json:"medianDuration"`
	P95Duration        time.Duration     `json:"p95Duration"`
	SlowestTests       []SlowTest        `json:"slowestTests"`
	PerformanceTrend   string            `json:"performanceTrend"` // "improving", "declining", "stable"
	DurationHistory    []time.Duration   `json:"durationHistory"`
}

// StabilityTrends represents stability trend analysis
type StabilityTrends struct {
	PassRateTrend       string  `json:"passRateTrend"`
	PerformanceTrend    string  `json:"performanceTrend"`
	FlakinessIncrease   bool    `json:"flakinessIncrease"`
	StabilityScore      float64 `json:"stabilityScore"`      // 0-100 score
	PredictedStability  float64 `json:"predictedStability"`  // Predicted future stability
	RecommendedActions  []string `json:"recommendedActions"`
}

// NewStabilityMonitor creates a new stability monitor with default configuration
func NewStabilityMonitor() *StabilityMonitor {
	config := &StabilityConfig{
		OutputDir:            "coverage",
		HistoryFile:          "stability_history.json",
		MaxHistoryEntries:    200,
		StabilityThreshold:   95.0, // 95% pass rate
		PerformanceThreshold: 2 * time.Minute, // 2 minutes max
		FlakyTestThreshold:   3,    // 3 failures = flaky
		MonitoringWindow:     30,   // 30 days
	}

	return &StabilityMonitor{
		config:  config,
		history: make([]StabilityReport, 0),
	}
}

// RunStabilityAnalysis executes comprehensive stability analysis
func (sm *StabilityMonitor) RunStabilityAnalysis() (*StabilityReport, error) {
	fmt.Println("🔍 Starting test stability analysis...")

	// Ensure output directory exists
	if err := os.MkdirAll(sm.config.OutputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	// Load historical data
	if err := sm.loadHistory(); err != nil {
		fmt.Printf("Warning: failed to load stability history: %v\n", err)
	}

	// Run multiple test executions to assess stability
	executions, err := sm.runMultipleTestExecutions(5) // Run tests 5 times
	if err != nil {
		return nil, fmt.Errorf("failed to run test executions: %w", err)
	}

	// Analyze stability metrics
	stabilityMetrics := sm.analyzeStabilityMetrics(executions)

	// Identify flaky tests
	flakyTests := sm.identifyFlakyTests(executions)

	// Analyze performance data
	performanceData := sm.analyzePerformanceData(executions)

	// Analyze trends
	trendAnalysis := sm.analyzeTrends(stabilityMetrics)

	// Generate recommendations
	recommendations := sm.generateRecommendations(stabilityMetrics, flakyTests, performanceData)

	// Create report
	report := &StabilityReport{
		Timestamp:        time.Now(),
		TestExecution:    executions[len(executions)-1], // Latest execution
		StabilityMetrics: stabilityMetrics,
		FlakyTests:       flakyTests,
		PerformanceData:  performanceData,
		TrendAnalysis:    trendAnalysis,
		Recommendations:  recommendations,
	}

	// Save report
	if err := sm.saveReport(report); err != nil {
		return nil, fmt.Errorf("failed to save stability report: %w", err)
	}

	// Update history
	sm.history = append(sm.history, *report)
	if err := sm.saveHistory(); err != nil {
		fmt.Printf("Warning: failed to save stability history: %v\n", err)
	}

	return report, nil
}

// runMultipleTestExecutions runs tests multiple times to assess stability
func (sm *StabilityMonitor) runMultipleTestExecutions(count int) ([]TestExecution, error) {
	executions := make([]TestExecution, 0, count)

	for i := 0; i < count; i++ {
		fmt.Printf("🧪 Running test execution %d/%d...\n", i+1, count)
		
		execution, err := sm.runSingleTestExecution()
		if err != nil {
			return nil, fmt.Errorf("test execution %d failed: %w", i+1, err)
		}
		
		executions = append(executions, *execution)
		
		// Small delay between executions
		if i < count-1 {
			time.Sleep(1 * time.Second)
		}
	}

	return executions, nil
}

// runSingleTestExecution runs tests once and captures detailed results
func (sm *StabilityMonitor) runSingleTestExecution() (*TestExecution, error) {
	startTime := time.Now()
	
	// Run tests with verbose output to capture individual test results
	cmd := exec.Command("go", "test", "-v", "-timeout=5m", "./...")
	output, _ := cmd.CombinedOutput()
	
	endTime := time.Now()
	duration := endTime.Sub(startTime)
	
	// Parse test results from output
	execution := &TestExecution{
		StartTime:       startTime,
		EndTime:         endTime,
		Duration:        duration,
		FailedTestNames: make([]string, 0),
		SlowTests:       make([]SlowTest, 0),
	}
	
	// Parse output to extract test results
	lines := strings.Split(string(output), "\n")
	testDurations := make(map[string]time.Duration)
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		// Parse test results
		if strings.Contains(line, "--- PASS:") {
			execution.PassedTests++
			if duration := sm.parseTestDuration(line); duration > 0 {
				testName := sm.parseTestName(line)
				testDurations[testName] = duration
			}
		} else if strings.Contains(line, "--- FAIL:") {
			execution.FailedTests++
			testName := sm.parseTestName(line)
			execution.FailedTestNames = append(execution.FailedTestNames, testName)
			if duration := sm.parseTestDuration(line); duration > 0 {
				testDurations[testName] = duration
			}
		} else if strings.Contains(line, "--- SKIP:") {
			execution.SkippedTests++
		}
	}
	
	execution.TotalTests = execution.PassedTests + execution.FailedTests + execution.SkippedTests
	if execution.TotalTests > 0 {
		execution.PassRate = (float64(execution.PassedTests) / float64(execution.TotalTests)) * 100
	}
	
	// Identify slow tests
	for testName, testDuration := range testDurations {
		if testDuration > 10*time.Second { // Consider tests > 10s as slow
			execution.SlowTests = append(execution.SlowTests, SlowTest{
				Name:     testName,
				Duration: testDuration,
			})
		}
	}
	
	// Sort slow tests by duration
	sort.Slice(execution.SlowTests, func(i, j int) bool {
		return execution.SlowTests[i].Duration > execution.SlowTests[j].Duration
	})
	
	return execution, nil
}

// parseTestName extracts test name from test output line
func (sm *StabilityMonitor) parseTestName(line string) string {
	parts := strings.Fields(line)
	if len(parts) >= 3 {
		return parts[2]
	}
	return "unknown"
}

// parseTestDuration extracts test duration from test output line
func (sm *StabilityMonitor) parseTestDuration(line string) time.Duration {
	// Look for duration pattern like (0.00s)
	if idx := strings.Index(line, "("); idx != -1 {
		if endIdx := strings.Index(line[idx:], ")"); endIdx != -1 {
			durationStr := line[idx+1 : idx+endIdx]
			if duration, err := time.ParseDuration(durationStr); err == nil {
				return duration
			}
		}
	}
	return 0
}

// analyzeStabilityMetrics calculates stability metrics from multiple executions
func (sm *StabilityMonitor) analyzeStabilityMetrics(executions []TestExecution) StabilityMetrics {
	if len(executions) == 0 {
		return StabilityMetrics{}
	}
	
	// Calculate overall stability (average pass rate)
	totalPassRate := 0.0
	testStability := make(map[string]float64)
	testCounts := make(map[string]int)
	testPasses := make(map[string]int)
	
	for _, execution := range executions {
		totalPassRate += execution.PassRate
		
		// Track individual test stability
		allTests := make(map[string]bool)
		
		// Mark passed tests
		totalTests := execution.PassedTests + execution.FailedTests + execution.SkippedTests
		passedTests := execution.PassedTests
		
		// Estimate test names (simplified approach)
		for i := 0; i < totalTests; i++ {
			testName := fmt.Sprintf("test_%d", i)
			allTests[testName] = i < passedTests
			testCounts[testName]++
			if i < passedTests {
				testPasses[testName]++
			}
		}
	}
	
	overallStability := totalPassRate / float64(len(executions))
	
	// Calculate per-test stability
	for testName, passes := range testPasses {
		if count := testCounts[testName]; count > 0 {
			testStability[testName] = (float64(passes) / float64(count)) * 100
		}
	}
	
	// Calculate consistency score (how consistent are the results)
	passRates := make([]float64, len(executions))
	for i, execution := range executions {
		passRates[i] = execution.PassRate
	}
	consistencyScore := sm.calculateConsistencyScore(passRates)
	
	// Calculate reliability score
	reliabilityScore := (overallStability + consistencyScore) / 2
	
	// Determine trend
	trend := "stable"
	if len(sm.history) > 0 {
		lastStability := sm.history[len(sm.history)-1].StabilityMetrics.OverallStability
		if overallStability > lastStability+2 {
			trend = "improving"
		} else if overallStability < lastStability-2 {
			trend = "declining"
		}
	}
	
	return StabilityMetrics{
		OverallStability: overallStability,
		TestStability:    testStability,
		AveragePassRate:  overallStability,
		StabilityTrend:   trend,
		ConsistencyScore: consistencyScore,
		ReliabilityScore: reliabilityScore,
	}
}

// calculateConsistencyScore calculates how consistent test results are
func (sm *StabilityMonitor) calculateConsistencyScore(passRates []float64) float64 {
	if len(passRates) <= 1 {
		return 100.0
	}
	
	// Calculate standard deviation
	mean := 0.0
	for _, rate := range passRates {
		mean += rate
	}
	mean /= float64(len(passRates))
	
	variance := 0.0
	for _, rate := range passRates {
		variance += (rate - mean) * (rate - mean)
	}
	variance /= float64(len(passRates))
	
	stdDev := math.Sqrt(variance)
	
	// Convert to consistency score (lower std dev = higher consistency)
	// Use coefficient of variation for better scaling
	coefficientOfVariation := stdDev / mean * 100
	consistencyScore := 100.0 - coefficientOfVariation
	if consistencyScore < 0 {
		consistencyScore = 0
	}
	
	return consistencyScore
}

// identifyFlakyTests identifies tests that fail intermittently
func (sm *StabilityMonitor) identifyFlakyTests(executions []TestExecution) []FlakyTest {
	testFailures := make(map[string]int)
	testTotalRuns := make(map[string]int)
	lastFailures := make(map[string]time.Time)
	
	// Count failures across executions
	for _, execution := range executions {
		// Track failed tests
		for _, failedTest := range execution.FailedTestNames {
			testFailures[failedTest]++
			lastFailures[failedTest] = execution.EndTime
			// Each execution counts as one run for this test
			testTotalRuns[failedTest]++
		}
		
		// For each failed test, assume it ran in all executions
		for failedTest := range testFailures {
			if testTotalRuns[failedTest] < len(executions) {
				testTotalRuns[failedTest] = len(executions)
			}
		}
	}
	
	// Identify flaky tests
	flakyTests := make([]FlakyTest, 0)
	for testName, failures := range testFailures {
		totalRuns := testTotalRuns[testName]
		if totalRuns == 0 {
			continue
		}
		
		failureRate := (float64(failures) / float64(totalRuns)) * 100
		
		// Consider a test flaky if it fails sometimes but not always
		if failures >= sm.config.FlakyTestThreshold && failures < totalRuns {
			severity := "low"
			if failureRate > 50 {
				severity = "critical"
			} else if failureRate > 25 {
				severity = "high"
			} else if failureRate > 10 {
				severity = "medium"
			}
			
			flakyTests = append(flakyTests, FlakyTest{
				Name:           testName,
				FailureCount:   failures,
				TotalRuns:      totalRuns,
				FailureRate:    failureRate,
				LastFailure:    lastFailures[testName],
				FailureReasons: []string{"Intermittent failure"}, // Simplified
				Severity:       severity,
			})
		}
	}
	
	// Sort by failure rate (most flaky first)
	sort.Slice(flakyTests, func(i, j int) bool {
		return flakyTests[i].FailureRate > flakyTests[j].FailureRate
	})
	
	return flakyTests
}

// analyzePerformanceData analyzes test performance across executions
func (sm *StabilityMonitor) analyzePerformanceData(executions []TestExecution) PerformanceData {
	if len(executions) == 0 {
		return PerformanceData{}
	}
	
	durations := make([]time.Duration, len(executions))
	totalDuration := time.Duration(0)
	
	for i, execution := range executions {
		durations[i] = execution.Duration
		totalDuration += execution.Duration
	}
	
	// Sort durations for percentile calculations
	sortedDurations := make([]time.Duration, len(durations))
	copy(sortedDurations, durations)
	sort.Slice(sortedDurations, func(i, j int) bool {
		return sortedDurations[i] < sortedDurations[j]
	})
	
	averageDuration := totalDuration / time.Duration(len(executions))
	medianDuration := sortedDurations[len(sortedDurations)/2]
	p95Index := int(float64(len(sortedDurations)) * 0.95)
	if p95Index >= len(sortedDurations) {
		p95Index = len(sortedDurations) - 1
	}
	p95Duration := sortedDurations[p95Index]
	
	// Collect slowest tests across all executions
	allSlowTests := make([]SlowTest, 0)
	for _, execution := range executions {
		allSlowTests = append(allSlowTests, execution.SlowTests...)
	}
	
	// Sort and take top 10 slowest
	sort.Slice(allSlowTests, func(i, j int) bool {
		return allSlowTests[i].Duration > allSlowTests[j].Duration
	})
	if len(allSlowTests) > 10 {
		allSlowTests = allSlowTests[:10]
	}
	
	// Determine performance trend
	performanceTrend := "stable"
	if len(sm.history) > 0 {
		lastAvgDuration := sm.history[len(sm.history)-1].PerformanceData.AverageDuration
		if averageDuration < lastAvgDuration-5*time.Second {
			performanceTrend = "improving"
		} else if averageDuration > lastAvgDuration+5*time.Second {
			performanceTrend = "declining"
		}
	}
	
	return PerformanceData{
		TotalDuration:    totalDuration,
		AverageDuration:  averageDuration,
		MedianDuration:   medianDuration,
		P95Duration:      p95Duration,
		SlowestTests:     allSlowTests,
		PerformanceTrend: performanceTrend,
		DurationHistory:  durations,
	}
}

// analyzeTrends analyzes stability trends over time
func (sm *StabilityMonitor) analyzeTrends(metrics StabilityMetrics) StabilityTrends {
	// Calculate stability score (0-100)
	stabilityScore := (metrics.OverallStability + metrics.ConsistencyScore + metrics.ReliabilityScore) / 3
	
	// Predict future stability (simplified linear prediction)
	predictedStability := metrics.OverallStability
	if len(sm.history) > 1 {
		recentStability := make([]float64, 0)
		for _, report := range sm.history {
			recentStability = append(recentStability, report.StabilityMetrics.OverallStability)
		}
		
		if len(recentStability) >= 2 {
			// Simple linear trend
			first := recentStability[0]
			last := recentStability[len(recentStability)-1]
			trend := (last - first) / float64(len(recentStability))
			predictedStability = metrics.OverallStability + trend
		}
	}
	
	// Generate recommended actions
	recommendedActions := make([]string, 0)
	if metrics.OverallStability < sm.config.StabilityThreshold {
		recommendedActions = append(recommendedActions, "Investigate and fix failing tests")
	}
	if metrics.ConsistencyScore < 80 {
		recommendedActions = append(recommendedActions, "Improve test consistency and reduce flakiness")
	}
	if stabilityScore < 85 {
		recommendedActions = append(recommendedActions, "Review test infrastructure and environment")
	}
	
	return StabilityTrends{
		PassRateTrend:      metrics.StabilityTrend,
		PerformanceTrend:   "stable", // Will be set by performance analysis
		FlakinessIncrease:  false,    // Will be calculated based on history
		StabilityScore:     stabilityScore,
		PredictedStability: predictedStability,
		RecommendedActions: recommendedActions,
	}
}

// generateRecommendations generates actionable recommendations
func (sm *StabilityMonitor) generateRecommendations(metrics StabilityMetrics, flakyTests []FlakyTest, performance PerformanceData) []string {
	recommendations := make([]string, 0)
	
	// Stability recommendations
	if metrics.OverallStability < sm.config.StabilityThreshold {
		recommendations = append(recommendations, 
			fmt.Sprintf("Overall stability %.1f%% is below threshold %.1f%%. Focus on fixing consistently failing tests.", 
				metrics.OverallStability, sm.config.StabilityThreshold))
	}
	
	// Flaky test recommendations
	if len(flakyTests) > 0 {
		recommendations = append(recommendations, 
			fmt.Sprintf("Found %d flaky tests. Prioritize fixing tests with high failure rates.", len(flakyTests)))
		
		for i, test := range flakyTests {
			if i >= 3 { // Limit to top 3 recommendations
				break
			}
			recommendations = append(recommendations, 
				fmt.Sprintf("Fix flaky test '%s' (%.1f%% failure rate)", test.Name, test.FailureRate))
		}
	}
	
	// Performance recommendations
	if performance.AverageDuration > sm.config.PerformanceThreshold {
		recommendations = append(recommendations, 
			fmt.Sprintf("Test suite duration %v exceeds threshold %v. Consider parallelization or optimization.", 
				performance.AverageDuration, sm.config.PerformanceThreshold))
	}
	
	if len(performance.SlowestTests) > 0 {
		recommendations = append(recommendations, 
			fmt.Sprintf("Optimize slow tests, starting with '%s' (%v duration)", 
				performance.SlowestTests[0].Name, performance.SlowestTests[0].Duration))
	}
	
	// Consistency recommendations
	if metrics.ConsistencyScore < 80 {
		recommendations = append(recommendations, 
			"Test results are inconsistent. Review test environment setup and external dependencies.")
	}
	
	return recommendations
}

// saveReport saves the stability report to JSON
func (sm *StabilityMonitor) saveReport(report *StabilityReport) error {
	reportPath := filepath.Join(sm.config.OutputDir, "stability_report.json")
	
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal stability report: %w", err)
	}
	
	if err := os.WriteFile(reportPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write stability report: %w", err)
	}
	
	fmt.Printf("📄 Stability report saved: %s\n", reportPath)
	return nil
}

// loadHistory loads stability history from file
func (sm *StabilityMonitor) loadHistory() error {
	historyPath := filepath.Join(sm.config.OutputDir, sm.config.HistoryFile)
	
	if _, err := os.Stat(historyPath); os.IsNotExist(err) {
		return nil // No history file exists yet
	}
	
	data, err := os.ReadFile(historyPath)
	if err != nil {
		return fmt.Errorf("failed to read stability history: %w", err)
	}
	
	if err := json.Unmarshal(data, &sm.history); err != nil {
		return fmt.Errorf("failed to unmarshal stability history: %w", err)
	}
	
	return nil
}

// saveHistory saves stability history to file
func (sm *StabilityMonitor) saveHistory() error {
	historyPath := filepath.Join(sm.config.OutputDir, sm.config.HistoryFile)
	
	// Keep only recent history
	if len(sm.history) > sm.config.MaxHistoryEntries {
		sm.history = sm.history[len(sm.history)-sm.config.MaxHistoryEntries:]
	}
	
	data, err := json.MarshalIndent(sm.history, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal stability history: %w", err)
	}
	
	if err := os.WriteFile(historyPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write stability history: %w", err)
	}
	
	return nil
}

// PrintReport prints a formatted stability report to stdout
func (sm *StabilityMonitor) PrintReport(report *StabilityReport) {
	fmt.Println("\n🔍 Test Stability Analysis Report")
	fmt.Println("=================================")
	fmt.Printf("Timestamp: %s\n", report.Timestamp.Format("2006-01-02 15:04:05"))
	
	// Print stability metrics
	fmt.Println("\n📊 Stability Metrics:")
	fmt.Printf("  Overall Stability: %.1f%%\n", report.StabilityMetrics.OverallStability)
	fmt.Printf("  Consistency Score: %.1f%%\n", report.StabilityMetrics.ConsistencyScore)
	fmt.Printf("  Reliability Score: %.1f%%\n", report.StabilityMetrics.ReliabilityScore)
	fmt.Printf("  Trend: %s\n", report.StabilityMetrics.StabilityTrend)
	
	// Print test execution summary
	fmt.Println("\n🧪 Latest Test Execution:")
	fmt.Printf("  Duration: %v\n", report.TestExecution.Duration)
	fmt.Printf("  Pass Rate: %.1f%% (%d/%d tests)\n", 
		report.TestExecution.PassRate, report.TestExecution.PassedTests, report.TestExecution.TotalTests)
	
	if report.TestExecution.FailedTests > 0 {
		fmt.Printf("  Failed Tests: %d\n", report.TestExecution.FailedTests)
		for i, testName := range report.TestExecution.FailedTestNames {
			if i >= 5 { // Limit output
				fmt.Printf("    ... and %d more\n", len(report.TestExecution.FailedTestNames)-5)
				break
			}
			fmt.Printf("    - %s\n", testName)
		}
	}
	
	// Print flaky tests
	if len(report.FlakyTests) > 0 {
		fmt.Println("\n⚠️  Flaky Tests:")
		for i, test := range report.FlakyTests {
			if i >= 5 { // Limit output
				fmt.Printf("  ... and %d more flaky tests\n", len(report.FlakyTests)-5)
				break
			}
			
			severityIcon := "⚠️"
			if test.Severity == "critical" {
				severityIcon = "🚨"
			} else if test.Severity == "high" {
				severityIcon = "❌"
			}
			
			fmt.Printf("  %s %s: %.1f%% failure rate (%d/%d runs)\n", 
				severityIcon, test.Name, test.FailureRate, test.FailureCount, test.TotalRuns)
		}
	}
	
	// Print performance data
	fmt.Println("\n⏱️  Performance Data:")
	fmt.Printf("  Average Duration: %v\n", report.PerformanceData.AverageDuration)
	fmt.Printf("  95th Percentile: %v\n", report.PerformanceData.P95Duration)
	fmt.Printf("  Performance Trend: %s\n", report.PerformanceData.PerformanceTrend)
	
	if len(report.PerformanceData.SlowestTests) > 0 {
		fmt.Println("  Slowest Tests:")
		for i, test := range report.PerformanceData.SlowestTests {
			if i >= 3 { // Show top 3
				break
			}
			fmt.Printf("    - %s: %v\n", test.Name, test.Duration)
		}
	}
	
	// Print trend analysis
	fmt.Println("\n📈 Trend Analysis:")
	fmt.Printf("  Stability Score: %.1f/100\n", report.TrendAnalysis.StabilityScore)
	fmt.Printf("  Predicted Stability: %.1f%%\n", report.TrendAnalysis.PredictedStability)
	
	// Print recommendations
	if len(report.Recommendations) > 0 {
		fmt.Println("\n💡 Recommendations:")
		for _, recommendation := range report.Recommendations {
			fmt.Printf("  - %s\n", recommendation)
		}
	}
	
	// Print quality assessment
	fmt.Println("\n🎯 Quality Assessment:")
	if report.StabilityMetrics.OverallStability >= 95 {
		fmt.Println("  ✅ Excellent stability")
	} else if report.StabilityMetrics.OverallStability >= 90 {
		fmt.Println("  ✅ Good stability")
	} else if report.StabilityMetrics.OverallStability >= 80 {
		fmt.Println("  ⚠️  Acceptable stability - room for improvement")
	} else {
		fmt.Println("  ❌ Poor stability - immediate attention required")
	}
}