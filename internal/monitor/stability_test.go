package monitor

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStabilityMonitor(t *testing.T) {
	monitor := NewStabilityMonitor()
	
	assert.NotNil(t, monitor)
	assert.NotNil(t, monitor.config)
	assert.Equal(t, "coverage", monitor.config.OutputDir)
	assert.Equal(t, 95.0, monitor.config.StabilityThreshold)
	assert.Equal(t, 3, monitor.config.FlakyTestThreshold)
	assert.Equal(t, 2*time.Minute, monitor.config.PerformanceThreshold)
}

func TestStabilityMonitor_parseTestName(t *testing.T) {
	monitor := NewStabilityMonitor()
	
	testCases := []struct {
		line     string
		expected string
	}{
		{"--- PASS: TestAlignSpecsWithTrace (0.05s)", "TestAlignSpecsWithTrace"},
		{"--- FAIL: TestExecuteAlignment (0.12s)", "TestExecuteAlignment"},
		{"--- SKIP: TestIntegration (0.00s)", "TestIntegration"},
		{"=== RUN   TestCoverageAnalysis", "TestCoverageAnalysis"},
		{"invalid line", "unknown"},
		{"", "unknown"},
	}
	
	for _, tc := range testCases {
		t.Run(tc.line, func(t *testing.T) {
			result := monitor.parseTestName(tc.line)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestStabilityMonitor_parseTestDuration(t *testing.T) {
	monitor := NewStabilityMonitor()
	
	testCases := []struct {
		line     string
		expected time.Duration
	}{
		{"--- PASS: TestAlignSpecsWithTrace (0.05s)", 50 * time.Millisecond},
		{"--- FAIL: TestExecuteAlignment (1.23s)", 1230 * time.Millisecond},
		{"--- SKIP: TestIntegration (0.00s)", 0},
		{"no duration info", 0},
		{"", 0},
	}
	
	for _, tc := range testCases {
		t.Run(tc.line, func(t *testing.T) {
			result := monitor.parseTestDuration(tc.line)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestStabilityMonitor_analyzeStabilityMetrics(t *testing.T) {
	monitor := NewStabilityMonitor()
	
	// Create test executions
	executions := []TestExecution{
		{
			PassRate:     95.0,
			PassedTests:  19,
			FailedTests:  1,
			SkippedTests: 0,
		},
		{
			PassRate:     90.0,
			PassedTests:  18,
			FailedTests:  2,
			SkippedTests: 0,
		},
		{
			PassRate:     100.0,
			PassedTests:  20,
			FailedTests:  0,
			SkippedTests: 0,
		},
	}
	
	metrics := monitor.analyzeStabilityMetrics(executions)
	
	// Check overall stability (average pass rate)
	expectedStability := (95.0 + 90.0 + 100.0) / 3
	assert.Equal(t, expectedStability, metrics.OverallStability)
	assert.Equal(t, expectedStability, metrics.AveragePassRate)
	
	// Check consistency score (should be high for similar results)
	assert.Greater(t, metrics.ConsistencyScore, 80.0)
	
	// Check reliability score
	assert.Greater(t, metrics.ReliabilityScore, 80.0)
	
	// Check trend (should be stable with no history)
	assert.Equal(t, "stable", metrics.StabilityTrend)
}

func TestStabilityMonitor_calculateConsistencyScore(t *testing.T) {
	monitor := NewStabilityMonitor()
	
	testCases := []struct {
		name      string
		passRates []float64
		expected  float64
	}{
		{
			name:      "perfect_consistency",
			passRates: []float64{95.0, 95.0, 95.0, 95.0},
			expected:  100.0,
		},
		{
			name:      "good_consistency",
			passRates: []float64{94.0, 95.0, 96.0, 95.0},
			expected:  98.0, // Low variance
		},
		{
			name:      "poor_consistency",
			passRates: []float64{70.0, 95.0, 80.0, 90.0},
			expected:  50.0, // High variance
		},
		{
			name:      "single_value",
			passRates: []float64{95.0},
			expected:  100.0,
		},
		{
			name:      "empty",
			passRates: []float64{},
			expected:  100.0,
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			score := monitor.calculateConsistencyScore(tc.passRates)
			assert.InDelta(t, tc.expected, score, 10.0) // Allow some tolerance
		})
	}
}

func TestStabilityMonitor_identifyFlakyTests(t *testing.T) {
	monitor := NewStabilityMonitor()
	
	// Create test executions with some flaky tests
	executions := []TestExecution{
		{
			PassedTests:     8,
			FailedTests:     2,
			SkippedTests:    0,
			FailedTestNames: []string{"TestFlaky1", "TestFlaky2"},
			EndTime:         time.Now(),
		},
		{
			PassedTests:     9,
			FailedTests:     1,
			SkippedTests:    0,
			FailedTestNames: []string{"TestFlaky1"},
			EndTime:         time.Now(),
		},
		{
			PassedTests:     7,
			FailedTests:     3,
			SkippedTests:    0,
			FailedTestNames: []string{"TestFlaky1", "TestFlaky2", "TestAlwaysFails"},
			EndTime:         time.Now(),
		},
		{
			PassedTests:     6,
			FailedTests:     4,
			SkippedTests:    0,
			FailedTestNames: []string{"TestFlaky1", "TestFlaky2", "TestAlwaysFails", "TestFlaky3"},
			EndTime:         time.Now(),
		},
	}
	
	flakyTests := monitor.identifyFlakyTests(executions)
	
	// Should identify TestFlaky1 and TestFlaky2 as flaky (fail sometimes but not always)
	assert.NotEmpty(t, flakyTests)
	
	// Find TestFlaky1 in results
	var flaky1 *FlakyTest
	for i, test := range flakyTests {
		if test.Name == "TestFlaky1" {
			flaky1 = &flakyTests[i]
			break
		}
	}
	
	require.NotNil(t, flaky1, "TestFlaky1 should be identified as flaky")
	assert.Equal(t, 4, flaky1.FailureCount) // Failed in all 4 executions
	assert.Greater(t, flaky1.FailureRate, 0.0)
	assert.Less(t, flaky1.FailureRate, 100.0) // Not always failing
}

func TestStabilityMonitor_analyzePerformanceData(t *testing.T) {
	monitor := NewStabilityMonitor()
	
	// Create test executions with different durations
	executions := []TestExecution{
		{
			Duration: 30 * time.Second,
			SlowTests: []SlowTest{
				{Name: "TestSlow1", Duration: 15 * time.Second},
			},
		},
		{
			Duration: 45 * time.Second,
			SlowTests: []SlowTest{
				{Name: "TestSlow1", Duration: 20 * time.Second},
				{Name: "TestSlow2", Duration: 12 * time.Second},
			},
		},
		{
			Duration: 35 * time.Second,
			SlowTests: []SlowTest{
				{Name: "TestSlow1", Duration: 18 * time.Second},
			},
		},
	}
	
	performance := monitor.analyzePerformanceData(executions)
	
	// Check total and average duration
	expectedTotal := 30*time.Second + 45*time.Second + 35*time.Second
	expectedAverage := expectedTotal / 3
	
	assert.Equal(t, expectedTotal, performance.TotalDuration)
	assert.Equal(t, expectedAverage, performance.AverageDuration)
	
	// Check median (should be 35s)
	assert.Equal(t, 35*time.Second, performance.MedianDuration)
	
	// Check P95 (should be 45s for this small dataset)
	assert.Equal(t, 45*time.Second, performance.P95Duration)
	
	// Check slowest tests (should include TestSlow1 and TestSlow2)
	assert.NotEmpty(t, performance.SlowestTests)
	assert.LessOrEqual(t, len(performance.SlowestTests), 10) // Limited to top 10
	
	// Should be sorted by duration (slowest first)
	if len(performance.SlowestTests) > 1 {
		assert.GreaterOrEqual(t, performance.SlowestTests[0].Duration, performance.SlowestTests[1].Duration)
	}
}

func TestStabilityMonitor_analyzeTrends(t *testing.T) {
	monitor := NewStabilityMonitor()
	
	metrics := StabilityMetrics{
		OverallStability: 92.0,
		ConsistencyScore: 88.0,
		ReliabilityScore: 90.0,
		StabilityTrend:   "improving",
	}
	
	trends := monitor.analyzeTrends(metrics)
	
	// Check stability score calculation (weighted average)
	expectedScore := (92.0 + 88.0 + 90.0) / 3
	assert.Equal(t, expectedScore, trends.StabilityScore)
	
	// Check predicted stability (should be current stability with no history)
	assert.Equal(t, 92.0, trends.PredictedStability)
	
	// Check recommended actions
	assert.NotEmpty(t, trends.RecommendedActions)
	
	// Should recommend improving consistency since it's below 80%
	found := false
	for _, action := range trends.RecommendedActions {
		if contains(action, "consistency") {
			found = true
			break
		}
	}
	assert.True(t, found, "Should recommend improving consistency")
}

func TestStabilityMonitor_generateRecommendations(t *testing.T) {
	monitor := NewStabilityMonitor()
	
	// Create test data with various issues
	metrics := StabilityMetrics{
		OverallStability: 85.0, // Below 95% threshold
		ConsistencyScore: 75.0, // Below 80%
	}
	
	flakyTests := []FlakyTest{
		{
			Name:        "TestFlaky1",
			FailureRate: 30.0,
			Severity:    "high",
		},
		{
			Name:        "TestFlaky2",
			FailureRate: 15.0,
			Severity:    "medium",
		},
	}
	
	performance := PerformanceData{
		AverageDuration: 3 * time.Minute, // Above 2 minute threshold
		SlowestTests: []SlowTest{
			{Name: "TestSlow1", Duration: 45 * time.Second},
			{Name: "TestSlow2", Duration: 30 * time.Second},
		},
	}
	
	recommendations := monitor.generateRecommendations(metrics, flakyTests, performance)
	
	assert.NotEmpty(t, recommendations)
	
	// Should have recommendations for each issue
	recommendationTypes := make(map[string]bool)
	for _, rec := range recommendations {
		if contains(rec, "stability") && contains(rec, "threshold") {
			recommendationTypes["stability"] = true
		}
		if contains(rec, "flaky") {
			recommendationTypes["flaky"] = true
		}
		if contains(rec, "duration") || contains(rec, "performance") {
			recommendationTypes["performance"] = true
		}
		if contains(rec, "consistency") {
			recommendationTypes["consistency"] = true
		}
	}
	
	assert.True(t, recommendationTypes["stability"], "Should recommend stability improvement")
	assert.True(t, recommendationTypes["flaky"], "Should recommend fixing flaky tests")
	assert.True(t, recommendationTypes["performance"], "Should recommend performance improvement")
	assert.True(t, recommendationTypes["consistency"], "Should recommend consistency improvement")
}

func TestStabilityReport_Validation(t *testing.T) {
	report := &StabilityReport{
		Timestamp: time.Now(),
		TestExecution: TestExecution{
			StartTime:    time.Now().Add(-2 * time.Minute),
			EndTime:      time.Now(),
			Duration:     2 * time.Minute,
			TotalTests:   50,
			PassedTests:  47,
			FailedTests:  3,
			SkippedTests: 0,
			PassRate:     94.0,
		},
		StabilityMetrics: StabilityMetrics{
			OverallStability: 94.0,
			ConsistencyScore: 88.0,
			ReliabilityScore: 91.0,
			StabilityTrend:   "stable",
		},
		FlakyTests: []FlakyTest{
			{
				Name:        "TestFlaky1",
				FailureRate: 25.0,
				Severity:    "medium",
			},
		},
		PerformanceData: PerformanceData{
			AverageDuration: 90 * time.Second,
			P95Duration:     2 * time.Minute,
		},
		Recommendations: []string{
			"Fix flaky test TestFlaky1",
			"Improve test consistency",
		},
	}
	
	// Validate report structure
	assert.NotZero(t, report.Timestamp)
	assert.Equal(t, 50, report.TestExecution.TotalTests)
	assert.Equal(t, 47+3, report.TestExecution.PassedTests+report.TestExecution.FailedTests)
	assert.Equal(t, 94.0, report.TestExecution.PassRate)
	assert.Equal(t, 94.0, report.StabilityMetrics.OverallStability)
	assert.NotEmpty(t, report.FlakyTests)
	assert.NotEmpty(t, report.Recommendations)
	assert.Greater(t, report.PerformanceData.AverageDuration, 0*time.Second)
}

// Helper function to check if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && 
		   (s == substr || 
		    len(s) > len(substr) && 
		    (s[:len(substr)] == substr || 
		     s[len(s)-len(substr):] == substr || 
		     containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Benchmark tests
func BenchmarkStabilityMonitor_analyzeStabilityMetrics(b *testing.B) {
	monitor := NewStabilityMonitor()
	
	// Create large dataset
	executions := make([]TestExecution, 100)
	for i := 0; i < 100; i++ {
		executions[i] = TestExecution{
			PassRate:     90.0 + float64(i%10), // Vary between 90-99%
			PassedTests:  90 + i%10,
			FailedTests:  10 - i%10,
			SkippedTests: 0,
		}
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = monitor.analyzeStabilityMetrics(executions)
	}
}

func BenchmarkStabilityMonitor_identifyFlakyTests(b *testing.B) {
	monitor := NewStabilityMonitor()
	
	// Create executions with many flaky tests
	executions := make([]TestExecution, 50)
	for i := 0; i < 50; i++ {
		failedTests := make([]string, 0)
		for j := 0; j < 10; j++ {
			if (i+j)%3 == 0 { // Make some tests flaky
				failedTests = append(failedTests, fmt.Sprintf("TestFlaky%d", j))
			}
		}
		
		executions[i] = TestExecution{
			PassedTests:     40,
			FailedTests:     len(failedTests),
			FailedTestNames: failedTests,
			EndTime:         time.Now(),
		}
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = monitor.identifyFlakyTests(executions)
	}
}