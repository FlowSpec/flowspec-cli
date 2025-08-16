package monitor

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCoverageMonitor(t *testing.T) {
	monitor := NewCoverageMonitor()
	
	assert.NotNil(t, monitor)
	assert.NotNil(t, monitor.config)
	assert.Equal(t, "coverage", monitor.config.OutputDir)
	assert.Equal(t, 80.0, monitor.config.GlobalThreshold)
	assert.Contains(t, monitor.config.ModuleThresholds, "internal/engine")
	assert.Equal(t, 90.0, monitor.config.ModuleThresholds["internal/engine"])
}

func TestCoverageMonitor_parseCoverageOutput(t *testing.T) {
	monitor := NewCoverageMonitor()
	
	// Sample coverage output
	coverageOutput := `github.com/flowspec/flowspec-cli/internal/engine/engine.go:45:	AlignSpecsWithTrace	85.7%
github.com/flowspec/flowspec-cli/internal/models/servicespec.go:23:	Validate	92.3%
github.com/flowspec/flowspec-cli/cmd/flowspec-cli/main.go:67:	executeAlignment	78.5%
total:					(statements)		82.1%`

	modules, overall, err := monitor.parseCoverageOutput(coverageOutput)
	
	require.NoError(t, err)
	assert.Equal(t, 82.1, overall)
	assert.Contains(t, modules, "internal/engine")
	assert.Contains(t, modules, "internal/models")
	assert.Contains(t, modules, "cmd/flowspec-cli")
	
	// Check module data
	engineModule := modules["internal/engine"]
	assert.Equal(t, "internal/engine", engineModule.Name)
	assert.Equal(t, 85.7, engineModule.Coverage)
	assert.Equal(t, 90.0, engineModule.Threshold) // From default config
	assert.Equal(t, "fail", engineModule.Status)  // Below threshold
	
	modelsModule := modules["internal/models"]
	assert.Equal(t, "internal/models", modelsModule.Name)
	assert.Equal(t, 92.3, modelsModule.Coverage)
	assert.Equal(t, 90.0, modelsModule.Threshold)
	assert.Equal(t, "pass", modelsModule.Status) // Above threshold
}

func TestCoverageMonitor_getModuleFromPath(t *testing.T) {
	monitor := NewCoverageMonitor()
	
	testCases := []struct {
		path     string
		expected string
	}{
		{"internal/engine/engine.go", "internal/engine"},
		{"internal/models/servicespec.go", "internal/models"},
		{"cmd/flowspec-cli/main.go", "cmd/flowspec-cli"},
		{"internal/parser/yaml/parser.go", "internal/parser"},
		{"pkg/utils/helper.go", "other"},
		{"main.go", "other"},
	}
	
	for _, tc := range testCases {
		t.Run(tc.path, func(t *testing.T) {
			result := monitor.getModuleFromPath(tc.path)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestCoverageMonitor_evaluateQualityGate(t *testing.T) {
	monitor := NewCoverageMonitor()
	
	// Test passing quality gate
	modules := map[string]ModuleCoverage{
		"internal/engine": {
			Name:      "internal/engine",
			Coverage:  92.0,
			Threshold: 90.0,
			Status:    "pass",
		},
		"internal/models": {
			Name:      "internal/models",
			Coverage:  95.0,
			Threshold: 90.0,
			Status:    "pass",
		},
	}
	
	result := monitor.evaluateQualityGate(modules, 85.0)
	
	assert.True(t, result.Passed)
	assert.Empty(t, result.Violations)
	assert.Equal(t, 100.0, result.Score)
	
	// Test failing quality gate
	modules["internal/engine"] = ModuleCoverage{
		Name:      "internal/engine",
		Coverage:  75.0,
		Threshold: 90.0,
		Status:    "fail",
	}
	
	result = monitor.evaluateQualityGate(modules, 70.0)
	
	assert.False(t, result.Passed)
	assert.NotEmpty(t, result.Violations)
	assert.Less(t, result.Score, 100.0)
	assert.Contains(t, result.Violations[0], "Overall coverage")
	assert.Contains(t, result.Violations[1], "internal/engine")
}

func TestCoverageMonitor_analyzeTrends(t *testing.T) {
	monitor := NewCoverageMonitor()
	
	// Test with no history
	trends := monitor.analyzeTrends(85.0)
	assert.Equal(t, "stable", trends.Direction)
	assert.Equal(t, 0.0, trends.ChangePercent)
	assert.Equal(t, 85.0, trends.PredictedCoverage)
	
	// Add some history
	monitor.history = []CoverageReport{
		{
			Timestamp:       time.Now().AddDate(0, 0, -10),
			OverallCoverage: 80.0,
		},
		{
			Timestamp:       time.Now().AddDate(0, 0, -5),
			OverallCoverage: 82.0,
		},
	}
	
	trends = monitor.analyzeTrends(85.0)
	assert.Equal(t, "improving", trends.Direction)
	assert.Greater(t, trends.ChangePercent, 0.0)
	assert.Equal(t, 2, trends.DaysAnalyzed)
}

func TestCoverageMonitor_CheckAlerts(t *testing.T) {
	monitor := NewCoverageMonitor()
	
	// Add history for comparison
	monitor.history = []CoverageReport{
		{
			Timestamp:       time.Now().AddDate(0, 0, -1),
			OverallCoverage: 90.0,
		},
	}
	
	// Test coverage drop alert
	report := &CoverageReport{
		OverallCoverage: 80.0, // 10% drop
		QualityGate: QualityGateResult{
			Passed: false,
		},
		TestResults: TestResults{
			RaceConditions: true,
			FailedTests:    2,
		},
	}
	
	alerts := monitor.CheckAlerts(report)
	
	assert.NotEmpty(t, alerts)
	assert.Contains(t, alerts[0], "Coverage dropped")
	assert.Contains(t, alerts[1], "Quality gate failed")
	assert.Contains(t, alerts[2], "Race conditions detected")
	assert.Contains(t, alerts[3], "test(s) failed")
}

func TestCoverageMonitor_saveAndLoadHistory(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "coverage_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)
	
	monitor := NewCoverageMonitor()
	monitor.config.OutputDir = tempDir
	
	// Add test history
	testReport := CoverageReport{
		Timestamp:       time.Now(),
		OverallCoverage: 85.0,
		ModuleCoverage: map[string]ModuleCoverage{
			"test": {
				Name:     "test",
				Coverage: 85.0,
			},
		},
	}
	
	monitor.history = []CoverageReport{testReport}
	
	// Save history
	err = monitor.saveHistory()
	require.NoError(t, err)
	
	// Verify file exists
	historyFile := filepath.Join(tempDir, monitor.config.HistoryFile)
	assert.FileExists(t, historyFile)
	
	// Load history in new monitor
	newMonitor := NewCoverageMonitor()
	newMonitor.config.OutputDir = tempDir
	
	err = newMonitor.loadHistory()
	require.NoError(t, err)
	
	assert.Len(t, newMonitor.history, 1)
	assert.Equal(t, 85.0, newMonitor.history[0].OverallCoverage)
}

func TestCoverageMonitor_calculatePerformanceScore(t *testing.T) {
	
	testCases := []struct {
		name        string
		performance PerformanceData
		expected    float64
	}{
		{
			name: "excellent_performance",
			performance: PerformanceData{
				AverageDuration:  30 * time.Second,
				SlowestTests:     []SlowTest{},
				PerformanceTrend: "improving",
			},
			expected: 110.0, // 100 + 10 bonus for improving
		},
		{
			name: "poor_performance",
			performance: PerformanceData{
				AverageDuration:  3 * time.Minute, // Over 2 minute threshold
				SlowestTests:     make([]SlowTest, 6), // More than 5 slow tests
				PerformanceTrend: "declining",
			},
			expected: 35.0, // 100 - 30 (duration) - 20 (slow tests) - 15 (declining)
		},
		{
			name: "average_performance",
			performance: PerformanceData{
				AverageDuration:  90 * time.Second,
				SlowestTests:     make([]SlowTest, 3),
				PerformanceTrend: "stable",
			},
			expected: 90.0, // 100 - 10 (slow tests)
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a dashboard to access the method
			dashboard := NewDashboard()
			score := dashboard.calculatePerformanceScore(tc.performance)
			assert.Equal(t, tc.expected, score)
		})
	}
}

func TestCoverageReport_Validation(t *testing.T) {
	report := &CoverageReport{
		Timestamp:       time.Now(),
		OverallCoverage: 85.5,
		ModuleCoverage: map[string]ModuleCoverage{
			"internal/engine": {
				Name:              "internal/engine",
				Coverage:          88.0,
				Threshold:         90.0,
				Status:            "warning",
				TotalStatements:   100,
				CoveredStatements: 88,
			},
		},
		TestResults: TestResults{
			TotalTests:     50,
			PassedTests:    48,
			FailedTests:    2,
			SkippedTests:   0,
			Duration:       2 * time.Minute,
			RaceConditions: false,
		},
		QualityGate: QualityGateResult{
			Passed:     true,
			Score:      85.0,
			Violations: []string{},
			Warnings:   []string{"Module internal/engine below threshold"},
		},
	}
	
	// Validate report structure
	assert.NotZero(t, report.Timestamp)
	assert.Greater(t, report.OverallCoverage, 0.0)
	assert.NotEmpty(t, report.ModuleCoverage)
	assert.Equal(t, 50, report.TestResults.TotalTests)
	assert.Equal(t, 48+2, report.TestResults.PassedTests+report.TestResults.FailedTests)
	assert.True(t, report.QualityGate.Passed)
	assert.NotEmpty(t, report.QualityGate.Warnings)
}

// Benchmark tests
func BenchmarkCoverageMonitor_parseCoverageOutput(b *testing.B) {
	monitor := NewCoverageMonitor()
	
	// Large coverage output simulation
	coverageOutput := `github.com/flowspec/flowspec-cli/internal/engine/engine.go:45:	AlignSpecsWithTrace	85.7%
github.com/flowspec/flowspec-cli/internal/models/servicespec.go:23:	Validate	92.3%
github.com/flowspec/flowspec-cli/cmd/flowspec-cli/main.go:67:	executeAlignment	78.5%
github.com/flowspec/flowspec-cli/internal/parser/parser.go:12:	ParseFile	88.9%
github.com/flowspec/flowspec-cli/internal/renderer/renderer.go:34:	RenderHuman	91.2%
total:					(statements)		82.1%`
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = monitor.parseCoverageOutput(coverageOutput)
	}
}

func BenchmarkCoverageMonitor_evaluateQualityGate(b *testing.B) {
	monitor := NewCoverageMonitor()
	
	modules := map[string]ModuleCoverage{
		"internal/engine": {Name: "internal/engine", Coverage: 92.0, Threshold: 90.0, Status: "pass"},
		"internal/models": {Name: "internal/models", Coverage: 95.0, Threshold: 90.0, Status: "pass"},
		"internal/parser": {Name: "internal/parser", Coverage: 88.0, Threshold: 80.0, Status: "pass"},
		"cmd/flowspec-cli": {Name: "cmd/flowspec-cli", Coverage: 75.0, Threshold: 80.0, Status: "fail"},
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = monitor.evaluateQualityGate(modules, 85.0)
	}
}