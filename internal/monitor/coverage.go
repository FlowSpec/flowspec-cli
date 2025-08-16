package monitor

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// CoverageMonitor handles test coverage monitoring and reporting
type CoverageMonitor struct {
	config     *CoverageConfig
	thresholds map[string]float64
	history    []CoverageReport
}

// CoverageConfig defines configuration for coverage monitoring
type CoverageConfig struct {
	OutputDir         string            `json:"outputDir"`
	CoverageFile      string            `json:"coverageFile"`
	HTMLFile          string            `json:"htmlFile"`
	JSONFile          string            `json:"jsonFile"`
	GlobalThreshold   float64           `json:"globalThreshold"`
	ModuleThresholds  map[string]float64 `json:"moduleThresholds"`
	HistoryFile       string            `json:"historyFile"`
	AlertThreshold    float64           `json:"alertThreshold"`
	TrendWindowDays   int               `json:"trendWindowDays"`
}

// CoverageReport represents a coverage analysis report
type CoverageReport struct {
	Timestamp       time.Time                `json:"timestamp"`
	OverallCoverage float64                  `json:"overallCoverage"`
	ModuleCoverage  map[string]ModuleCoverage `json:"moduleCoverage"`
	TestResults     TestResults              `json:"testResults"`
	QualityGate     QualityGateResult        `json:"qualityGate"`
	Trends          CoverageTrends           `json:"trends"`
}

// ModuleCoverage represents coverage data for a specific module
type ModuleCoverage struct {
	Name            string  `json:"name"`
	Coverage        float64 `json:"coverage"`
	Threshold       float64 `json:"threshold"`
	Status          string  `json:"status"` // "pass", "fail", "warning"
	TotalStatements int     `json:"totalStatements"`
	CoveredStatements int   `json:"coveredStatements"`
	UncoveredFiles  []string `json:"uncoveredFiles"`
}

// TestResults represents test execution results
type TestResults struct {
	TotalTests   int           `json:"totalTests"`
	PassedTests  int           `json:"passedTests"`
	FailedTests  int           `json:"failedTests"`
	SkippedTests int           `json:"skippedTests"`
	Duration     time.Duration `json:"duration"`
	RaceConditions bool        `json:"raceConditions"`
}

// QualityGateResult represents the result of quality gate checks
type QualityGateResult struct {
	Passed      bool     `json:"passed"`
	Violations  []string `json:"violations"`
	Warnings    []string `json:"warnings"`
	Score       float64  `json:"score"`
}

// CoverageTrends represents coverage trend analysis
type CoverageTrends struct {
	Direction       string  `json:"direction"` // "improving", "declining", "stable"
	ChangePercent   float64 `json:"changePercent"`
	DaysAnalyzed    int     `json:"daysAnalyzed"`
	PredictedCoverage float64 `json:"predictedCoverage"`
}

// NewCoverageMonitor creates a new coverage monitor with default configuration
func NewCoverageMonitor() *CoverageMonitor {
	config := &CoverageConfig{
		OutputDir:       "coverage",
		CoverageFile:    "coverage.out",
		HTMLFile:        "coverage.html",
		JSONFile:        "coverage.json",
		GlobalThreshold: 80.0,
		ModuleThresholds: map[string]float64{
			"internal/engine":   90.0,
			"internal/models":   90.0,
			"internal/renderer": 85.0,
			"internal/parser":   80.0,
			"internal/ingestor": 80.0,
			"cmd/flowspec-cli":  80.0,
		},
		HistoryFile:     "coverage_history.json",
		AlertThreshold:  5.0, // Alert if coverage drops by 5%
		TrendWindowDays: 30,
	}

	return &CoverageMonitor{
		config:     config,
		thresholds: config.ModuleThresholds,
		history:    make([]CoverageReport, 0),
	}
}

// RunCoverageAnalysis executes comprehensive coverage analysis
func (cm *CoverageMonitor) RunCoverageAnalysis() (*CoverageReport, error) {
	fmt.Println("🧪 Starting comprehensive coverage analysis...")

	// Ensure output directory exists
	if err := os.MkdirAll(cm.config.OutputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	// Run tests with coverage
	testResults, err := cm.runTestsWithCoverage()
	if err != nil {
		return nil, fmt.Errorf("failed to run tests: %w", err)
	}

	// Parse coverage data
	moduleCoverage, overallCoverage, err := cm.parseCoverageData()
	if err != nil {
		return nil, fmt.Errorf("failed to parse coverage data: %w", err)
	}

	// Generate HTML report
	if err := cm.generateHTMLReport(); err != nil {
		return nil, fmt.Errorf("failed to generate HTML report: %w", err)
	}

	// Load coverage history
	if err := cm.loadHistory(); err != nil {
		fmt.Printf("Warning: failed to load coverage history: %v\n", err)
	}

	// Analyze trends
	trends := cm.analyzeTrends(overallCoverage)

	// Evaluate quality gate
	qualityGate := cm.evaluateQualityGate(moduleCoverage, overallCoverage)

	// Create report
	report := &CoverageReport{
		Timestamp:       time.Now(),
		OverallCoverage: overallCoverage,
		ModuleCoverage:  moduleCoverage,
		TestResults:     *testResults,
		QualityGate:     qualityGate,
		Trends:          trends,
	}

	// Save report
	if err := cm.saveReport(report); err != nil {
		return nil, fmt.Errorf("failed to save report: %w", err)
	}

	// Update history
	cm.history = append(cm.history, *report)
	if err := cm.saveHistory(); err != nil {
		fmt.Printf("Warning: failed to save coverage history: %v\n", err)
	}

	return report, nil
}

// runTestsWithCoverage executes tests and generates coverage data
func (cm *CoverageMonitor) runTestsWithCoverage() (*TestResults, error) {
	coveragePath := filepath.Join(cm.config.OutputDir, cm.config.CoverageFile)
	
	startTime := time.Now()
	
	// Run tests with coverage and race detection
	cmd := exec.Command("go", "test", "-v", "-race", "-coverprofile="+coveragePath, "./...")
	output, err := cmd.CombinedOutput()
	
	duration := time.Since(startTime)
	
	// Parse test results from output
	results := &TestResults{
		Duration: duration,
		RaceConditions: strings.Contains(string(output), "WARNING: DATA RACE"),
	}
	
	// Parse test counts from output
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "PASS") {
			results.PassedTests++
		} else if strings.Contains(line, "FAIL") {
			results.FailedTests++
		} else if strings.Contains(line, "SKIP") {
			results.SkippedTests++
		}
	}
	
	results.TotalTests = results.PassedTests + results.FailedTests + results.SkippedTests
	
	if err != nil && results.FailedTests == 0 {
		return nil, fmt.Errorf("test execution failed: %w\nOutput: %s", err, string(output))
	}
	
	return results, nil
}

// parseCoverageData parses the coverage output file
func (cm *CoverageMonitor) parseCoverageData() (map[string]ModuleCoverage, float64, error) {
	coveragePath := filepath.Join(cm.config.OutputDir, cm.config.CoverageFile)
	
	// Generate function-level coverage data
	cmd := exec.Command("go", "tool", "cover", "-func="+coveragePath)
	output, err := cmd.Output()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to generate coverage data: %w", err)
	}
	
	return cm.parseCoverageOutput(string(output))
}

// parseCoverageOutput parses the go tool cover output
func (cm *CoverageMonitor) parseCoverageOutput(output string) (map[string]ModuleCoverage, float64, error) {
	lines := strings.Split(output, "\n")
	moduleData := make(map[string]ModuleCoverage)
	var overallCoverage float64
	
	// Regex to parse coverage lines
	// Format: filepath:line: functionName coverage%
	coverageRegex := regexp.MustCompile(`^(.+?):\d+:\s*(\w+)\s+(\d+\.\d+)%$`)
	totalRegex := regexp.MustCompile(`^total:\s+\(statements\)\s+(\d+\.\d+)%$`)
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		
		// Check for total coverage
		if matches := totalRegex.FindStringSubmatch(line); matches != nil {
			coverage, _ := strconv.ParseFloat(matches[1], 64)
			overallCoverage = coverage
			continue
		}
		
		// Parse individual file coverage
		if matches := coverageRegex.FindStringSubmatch(line); matches != nil {
			filePath := matches[1]
			coverage, _ := strconv.ParseFloat(matches[3], 64)
			
			// Determine module from file path
			module := cm.getModuleFromPath(filePath)
			
			if existing, ok := moduleData[module]; ok {
				// Update existing module data
				existing.TotalStatements++
				if coverage > 0 {
					existing.CoveredStatements++
				} else {
					existing.UncoveredFiles = append(existing.UncoveredFiles, filePath)
				}
				// Recalculate average coverage
				existing.Coverage = (existing.Coverage + coverage) / 2
				moduleData[module] = existing
			} else {
				// Create new module data
				threshold := cm.config.GlobalThreshold
				if moduleThreshold, exists := cm.thresholds[module]; exists {
					threshold = moduleThreshold
				}
				
				status := "pass"
				if coverage < threshold {
					status = "fail"
				}
				
				uncoveredFiles := make([]string, 0)
				if coverage == 0 {
					uncoveredFiles = append(uncoveredFiles, filePath)
				}
				
				moduleData[module] = ModuleCoverage{
					Name:              module,
					Coverage:          coverage,
					Threshold:         threshold,
					Status:            status,
					TotalStatements:   1,
					CoveredStatements: func() int { if coverage > 0 { return 1 }; return 0 }(),
					UncoveredFiles:    uncoveredFiles,
				}
			}
		}
	}
	
	return moduleData, overallCoverage, nil
}

// getModuleFromPath determines the module name from a file path
func (cm *CoverageMonitor) getModuleFromPath(filePath string) string {
	// Handle full package paths (e.g., github.com/flowspec/flowspec-cli/internal/engine/engine.go)
	// Extract the relative path after the package name
	if strings.Contains(filePath, "github.com/flowspec/flowspec-cli/") {
		parts := strings.Split(filePath, "github.com/flowspec/flowspec-cli/")
		if len(parts) > 1 {
			filePath = parts[1]
		}
	}
	
	// Remove file extension and get directory
	dir := filepath.Dir(filePath)
	
	// Map common patterns to modules
	if strings.HasPrefix(dir, "cmd/") {
		return "cmd/flowspec-cli"
	} else if strings.HasPrefix(dir, "internal/") {
		parts := strings.Split(dir, "/")
		if len(parts) >= 2 {
			return strings.Join(parts[:2], "/")
		}
	}
	
	return "other"
}

// generateHTMLReport generates an HTML coverage report
func (cm *CoverageMonitor) generateHTMLReport() error {
	coveragePath := filepath.Join(cm.config.OutputDir, cm.config.CoverageFile)
	htmlPath := filepath.Join(cm.config.OutputDir, cm.config.HTMLFile)
	
	cmd := exec.Command("go", "tool", "cover", "-html="+coveragePath, "-o", htmlPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to generate HTML report: %w", err)
	}
	
	fmt.Printf("📊 HTML coverage report generated: %s\n", htmlPath)
	return nil
}

// evaluateQualityGate checks if coverage meets quality standards
func (cm *CoverageMonitor) evaluateQualityGate(modules map[string]ModuleCoverage, overall float64) QualityGateResult {
	violations := make([]string, 0)
	warnings := make([]string, 0)
	score := 100.0
	
	// Check overall coverage
	if overall < cm.config.GlobalThreshold {
		violations = append(violations, fmt.Sprintf("Overall coverage %.1f%% below threshold %.1f%%", overall, cm.config.GlobalThreshold))
		score -= 20
	}
	
	// Check module-specific thresholds
	for _, module := range modules {
		if module.Status == "fail" {
			violations = append(violations, fmt.Sprintf("Module %s coverage %.1f%% below threshold %.1f%%", module.Name, module.Coverage, module.Threshold))
			score -= 10
		} else if module.Status == "warning" {
			warnings = append(warnings, fmt.Sprintf("Module %s coverage %.1f%% close to threshold %.1f%%", module.Name, module.Coverage, module.Threshold))
			score -= 5
		}
	}
	
	return QualityGateResult{
		Passed:     len(violations) == 0,
		Violations: violations,
		Warnings:   warnings,
		Score:      score,
	}
}

// analyzeTrends analyzes coverage trends over time
func (cm *CoverageMonitor) analyzeTrends(currentCoverage float64) CoverageTrends {
	if len(cm.history) < 2 {
		return CoverageTrends{
			Direction:         "stable",
			ChangePercent:     0,
			DaysAnalyzed:      0,
			PredictedCoverage: currentCoverage,
		}
	}
	
	// Get recent history within trend window
	cutoff := time.Now().AddDate(0, 0, -cm.config.TrendWindowDays)
	recentHistory := make([]CoverageReport, 0)
	
	for _, report := range cm.history {
		if report.Timestamp.After(cutoff) {
			recentHistory = append(recentHistory, report)
		}
	}
	
	if len(recentHistory) < 2 {
		return CoverageTrends{
			Direction:         "stable",
			ChangePercent:     0,
			DaysAnalyzed:      len(recentHistory),
			PredictedCoverage: currentCoverage,
		}
	}
	
	// Calculate trend
	firstCoverage := recentHistory[0].OverallCoverage
	changePercent := ((currentCoverage - firstCoverage) / firstCoverage) * 100
	
	direction := "stable"
	if changePercent > 1 {
		direction = "improving"
	} else if changePercent < -1 {
		direction = "declining"
	}
	
	// Simple linear prediction
	avgChange := changePercent / float64(len(recentHistory))
	predictedCoverage := currentCoverage + avgChange
	
	return CoverageTrends{
		Direction:         direction,
		ChangePercent:     changePercent,
		DaysAnalyzed:      len(recentHistory),
		PredictedCoverage: predictedCoverage,
	}
}

// saveReport saves the coverage report to JSON
func (cm *CoverageMonitor) saveReport(report *CoverageReport) error {
	jsonPath := filepath.Join(cm.config.OutputDir, cm.config.JSONFile)
	
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal report: %w", err)
	}
	
	if err := os.WriteFile(jsonPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write report file: %w", err)
	}
	
	fmt.Printf("📄 JSON coverage report saved: %s\n", jsonPath)
	return nil
}

// loadHistory loads coverage history from file
func (cm *CoverageMonitor) loadHistory() error {
	historyPath := filepath.Join(cm.config.OutputDir, cm.config.HistoryFile)
	
	if _, err := os.Stat(historyPath); os.IsNotExist(err) {
		return nil // No history file exists yet
	}
	
	data, err := os.ReadFile(historyPath)
	if err != nil {
		return fmt.Errorf("failed to read history file: %w", err)
	}
	
	if err := json.Unmarshal(data, &cm.history); err != nil {
		return fmt.Errorf("failed to unmarshal history: %w", err)
	}
	
	return nil
}

// saveHistory saves coverage history to file
func (cm *CoverageMonitor) saveHistory() error {
	historyPath := filepath.Join(cm.config.OutputDir, cm.config.HistoryFile)
	
	// Keep only recent history to prevent file from growing too large
	maxHistoryEntries := 100
	if len(cm.history) > maxHistoryEntries {
		cm.history = cm.history[len(cm.history)-maxHistoryEntries:]
	}
	
	data, err := json.MarshalIndent(cm.history, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal history: %w", err)
	}
	
	if err := os.WriteFile(historyPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write history file: %w", err)
	}
	
	return nil
}

// PrintReport prints a formatted coverage report to stdout
func (cm *CoverageMonitor) PrintReport(report *CoverageReport) {
	fmt.Println("\n📊 Coverage Analysis Report")
	fmt.Println("==========================")
	fmt.Printf("Timestamp: %s\n", report.Timestamp.Format("2006-01-02 15:04:05"))
	fmt.Printf("Overall Coverage: %.1f%%\n", report.OverallCoverage)
	
	// Print module coverage
	fmt.Println("\n📦 Module Coverage:")
	
	// Sort modules by name for consistent output
	modules := make([]ModuleCoverage, 0, len(report.ModuleCoverage))
	for _, module := range report.ModuleCoverage {
		modules = append(modules, module)
	}
	sort.Slice(modules, func(i, j int) bool {
		return modules[i].Name < modules[j].Name
	})
	
	for _, module := range modules {
		status := "✅"
		if module.Status == "fail" {
			status = "❌"
		} else if module.Status == "warning" {
			status = "⚠️"
		}
		
		fmt.Printf("  %s %s: %.1f%% (threshold: %.1f%%)\n", 
			status, module.Name, module.Coverage, module.Threshold)
		
		if len(module.UncoveredFiles) > 0 && len(module.UncoveredFiles) <= 3 {
			for _, file := range module.UncoveredFiles {
				fmt.Printf("    - Uncovered: %s\n", file)
			}
		} else if len(module.UncoveredFiles) > 3 {
			fmt.Printf("    - %d uncovered files\n", len(module.UncoveredFiles))
		}
	}
	
	// Print test results
	fmt.Println("\n🧪 Test Results:")
	fmt.Printf("  Total: %d, Passed: %d, Failed: %d, Skipped: %d\n",
		report.TestResults.TotalTests, report.TestResults.PassedTests,
		report.TestResults.FailedTests, report.TestResults.SkippedTests)
	fmt.Printf("  Duration: %v\n", report.TestResults.Duration)
	if report.TestResults.RaceConditions {
		fmt.Println("  ⚠️  Race conditions detected")
	}
	
	// Print quality gate results
	fmt.Println("\n🚪 Quality Gate:")
	if report.QualityGate.Passed {
		fmt.Printf("  ✅ PASSED (Score: %.1f/100)\n", report.QualityGate.Score)
	} else {
		fmt.Printf("  ❌ FAILED (Score: %.1f/100)\n", report.QualityGate.Score)
	}
	
	if len(report.QualityGate.Violations) > 0 {
		fmt.Println("  Violations:")
		for _, violation := range report.QualityGate.Violations {
			fmt.Printf("    - %s\n", violation)
		}
	}
	
	if len(report.QualityGate.Warnings) > 0 {
		fmt.Println("  Warnings:")
		for _, warning := range report.QualityGate.Warnings {
			fmt.Printf("    - %s\n", warning)
		}
	}
	
	// Print trends
	fmt.Println("\n📈 Coverage Trends:")
	trendIcon := "📊"
	if report.Trends.Direction == "improving" {
		trendIcon = "📈"
	} else if report.Trends.Direction == "declining" {
		trendIcon = "📉"
	}
	
	fmt.Printf("  %s Direction: %s (%.1f%% change over %d days)\n",
		trendIcon, report.Trends.Direction, report.Trends.ChangePercent, report.Trends.DaysAnalyzed)
	fmt.Printf("  🔮 Predicted coverage: %.1f%%\n", report.Trends.PredictedCoverage)
}

// CheckAlerts checks if any coverage alerts should be triggered
func (cm *CoverageMonitor) CheckAlerts(report *CoverageReport) []string {
	alerts := make([]string, 0)
	
	// Check for significant coverage drops
	if len(cm.history) > 0 {
		lastReport := cm.history[len(cm.history)-1]
		coverageDrop := lastReport.OverallCoverage - report.OverallCoverage
		
		if coverageDrop > cm.config.AlertThreshold {
			alerts = append(alerts, fmt.Sprintf("Coverage dropped by %.1f%% (from %.1f%% to %.1f%%)",
				coverageDrop, lastReport.OverallCoverage, report.OverallCoverage))
		}
	}
	
	// Check for failing quality gate
	if !report.QualityGate.Passed {
		alerts = append(alerts, "Quality gate failed")
	}
	
	// Check for race conditions
	if report.TestResults.RaceConditions {
		alerts = append(alerts, "Race conditions detected in tests")
	}
	
	// Check for test failures
	if report.TestResults.FailedTests > 0 {
		alerts = append(alerts, fmt.Sprintf("%d test(s) failed", report.TestResults.FailedTests))
	}
	
	return alerts
}