package monitor

import (
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// Dashboard generates comprehensive monitoring dashboards
type Dashboard struct {
	coverageMonitor  *CoverageMonitor
	stabilityMonitor *StabilityMonitor
	config           *DashboardConfig
}

// DashboardConfig defines dashboard configuration
type DashboardConfig struct {
	OutputDir     string `json:"outputDir"`
	TemplateDir   string `json:"templateDir"`
	StaticDir     string `json:"staticDir"`
	RefreshRate   int    `json:"refreshRate"` // seconds
	HistoryDays   int    `json:"historyDays"`
	AlertsEnabled bool   `json:"alertsEnabled"`
}

// DashboardData represents all data for the monitoring dashboard
type DashboardData struct {
	GeneratedAt      time.Time         `json:"generatedAt"`
	CoverageReport   *CoverageReport   `json:"coverageReport"`
	StabilityReport  *StabilityReport  `json:"stabilityReport"`
	QualityOverview  QualityOverview   `json:"qualityOverview"`
	TrendCharts      TrendCharts       `json:"trendCharts"`
	Alerts           []Alert           `json:"alerts"`
	Recommendations  []Recommendation  `json:"recommendations"`
}

// QualityOverview provides a high-level quality summary
type QualityOverview struct {
	OverallScore        float64 `json:"overallScore"`        // 0-100
	CoverageScore       float64 `json:"coverageScore"`       // 0-100
	StabilityScore      float64 `json:"stabilityScore"`      // 0-100
	PerformanceScore    float64 `json:"performanceScore"`    // 0-100
	QualityGrade        string  `json:"qualityGrade"`        // A, B, C, D, F
	Status              string  `json:"status"`              // "excellent", "good", "warning", "critical"
	LastImprovement     string  `json:"lastImprovement"`     // Description of last improvement
	NextMilestone       string  `json:"nextMilestone"`       // Next quality milestone
}

// TrendCharts contains data for trend visualization
type TrendCharts struct {
	CoverageTrend    []TrendPoint `json:"coverageTrend"`
	StabilityTrend   []TrendPoint `json:"stabilityTrend"`
	PerformanceTrend []TrendPoint `json:"performanceTrend"`
	TestCountTrend   []TrendPoint `json:"testCountTrend"`
}

// TrendPoint represents a single point in a trend chart
type TrendPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
	Label     string    `json:"label"`
}

// Alert represents a monitoring alert
type Alert struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`        // "coverage", "stability", "performance"
	Severity    string    `json:"severity"`    // "info", "warning", "error", "critical"
	Title       string    `json:"title"`
	Message     string    `json:"message"`
	Timestamp   time.Time `json:"timestamp"`
	Resolved    bool      `json:"resolved"`
	ActionItems []string  `json:"actionItems"`
}

// Recommendation represents an actionable recommendation
type Recommendation struct {
	ID          string    `json:"id"`
	Category    string    `json:"category"`    // "coverage", "stability", "performance", "quality"
	Priority    string    `json:"priority"`    // "low", "medium", "high", "critical"
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Impact      string    `json:"impact"`      // Expected impact of implementing
	Effort      string    `json:"effort"`      // Estimated effort required
	Steps       []string  `json:"steps"`       // Implementation steps
	CreatedAt   time.Time `json:"createdAt"`
}

// NewDashboard creates a new monitoring dashboard
func NewDashboard() *Dashboard {
	config := &DashboardConfig{
		OutputDir:     "coverage/dashboard",
		TemplateDir:   "templates",
		StaticDir:     "static",
		RefreshRate:   300, // 5 minutes
		HistoryDays:   30,
		AlertsEnabled: true,
	}

	return &Dashboard{
		coverageMonitor:  NewCoverageMonitor(),
		stabilityMonitor: NewStabilityMonitor(),
		config:           config,
	}
}

// GenerateDashboard creates a comprehensive monitoring dashboard
func (d *Dashboard) GenerateDashboard() error {
	fmt.Println("📊 Generating monitoring dashboard...")

	// Ensure output directory exists
	if err := os.MkdirAll(d.config.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create dashboard directory: %w", err)
	}

	// Run coverage analysis
	coverageReport, err := d.coverageMonitor.RunCoverageAnalysis()
	if err != nil {
		return fmt.Errorf("failed to run coverage analysis: %w", err)
	}

	// Run stability analysis
	stabilityReport, err := d.stabilityMonitor.RunStabilityAnalysis()
	if err != nil {
		return fmt.Errorf("failed to run stability analysis: %w", err)
	}

	// Generate quality overview
	qualityOverview := d.generateQualityOverview(coverageReport, stabilityReport)

	// Generate trend charts
	trendCharts := d.generateTrendCharts()

	// Generate alerts
	alerts := d.generateAlerts(coverageReport, stabilityReport)

	// Generate recommendations
	recommendations := d.generateRecommendations(coverageReport, stabilityReport, qualityOverview)

	// Create dashboard data
	dashboardData := &DashboardData{
		GeneratedAt:      time.Now(),
		CoverageReport:   coverageReport,
		StabilityReport:  stabilityReport,
		QualityOverview:  qualityOverview,
		TrendCharts:      trendCharts,
		Alerts:           alerts,
		Recommendations:  recommendations,
	}

	// Save dashboard data as JSON
	if err := d.saveDashboardData(dashboardData); err != nil {
		return fmt.Errorf("failed to save dashboard data: %w", err)
	}

	// Generate HTML dashboard
	if err := d.generateHTMLDashboard(dashboardData); err != nil {
		return fmt.Errorf("failed to generate HTML dashboard: %w", err)
	}

	// Generate static files
	if err := d.generateStaticFiles(); err != nil {
		return fmt.Errorf("failed to generate static files: %w", err)
	}

	fmt.Printf("✅ Dashboard generated successfully: %s/index.html\n", d.config.OutputDir)
	return nil
}

// generateQualityOverview creates a high-level quality summary
func (d *Dashboard) generateQualityOverview(coverage *CoverageReport, stability *StabilityReport) QualityOverview {
	// Calculate individual scores
	coverageScore := coverage.OverallCoverage
	stabilityScore := stability.StabilityMetrics.OverallStability
	performanceScore := d.calculatePerformanceScore(stability.PerformanceData)

	// Calculate overall score (weighted average)
	overallScore := (coverageScore*0.4 + stabilityScore*0.4 + performanceScore*0.2)

	// Determine quality grade
	grade := "F"
	status := "critical"
	if overallScore >= 90 {
		grade = "A"
		status = "excellent"
	} else if overallScore >= 80 {
		grade = "B"
		status = "good"
	} else if overallScore >= 70 {
		grade = "C"
		status = "warning"
	} else if overallScore >= 60 {
		grade = "D"
		status = "warning"
	}

	// Generate improvement and milestone messages
	lastImprovement := d.getLastImprovement(coverage, stability)
	nextMilestone := d.getNextMilestone(overallScore)

	return QualityOverview{
		OverallScore:     overallScore,
		CoverageScore:    coverageScore,
		StabilityScore:   stabilityScore,
		PerformanceScore: performanceScore,
		QualityGrade:     grade,
		Status:           status,
		LastImprovement:  lastImprovement,
		NextMilestone:    nextMilestone,
	}
}

// calculatePerformanceScore calculates a performance score from 0-100
func (d *Dashboard) calculatePerformanceScore(performance PerformanceData) float64 {
	// Base score starts at 100
	score := 100.0

	// Penalize long test durations
	if performance.AverageDuration > 2*time.Minute {
		score -= 30
	} else if performance.AverageDuration > 1*time.Minute {
		score -= 15
	}

	// Penalize slow tests
	if len(performance.SlowestTests) > 5 {
		score -= 20
	} else if len(performance.SlowestTests) > 2 {
		score -= 10
	}

	// Bonus for improving performance
	if performance.PerformanceTrend == "improving" {
		score += 10
	} else if performance.PerformanceTrend == "declining" {
		score -= 15
	}

	if score < 0 {
		score = 0
	}

	return score
}

// getLastImprovement generates a description of the last improvement
func (d *Dashboard) getLastImprovement(coverage *CoverageReport, stability *StabilityReport) string {
	improvements := make([]string, 0)

	if coverage.Trends.Direction == "improving" {
		improvements = append(improvements, fmt.Sprintf("Coverage increased by %.1f%%", coverage.Trends.ChangePercent))
	}

	if stability.StabilityMetrics.StabilityTrend == "improving" {
		improvements = append(improvements, "Test stability improved")
	}

	if len(improvements) == 0 {
		return "No recent improvements detected"
	}

	return improvements[0] // Return the first improvement
}

// getNextMilestone determines the next quality milestone
func (d *Dashboard) getNextMilestone(overallScore float64) string {
	if overallScore < 60 {
		return "Reach 60% overall quality score"
	} else if overallScore < 70 {
		return "Achieve C grade (70% quality score)"
	} else if overallScore < 80 {
		return "Achieve B grade (80% quality score)"
	} else if overallScore < 90 {
		return "Achieve A grade (90% quality score)"
	} else {
		return "Maintain excellent quality standards"
	}
}

// generateTrendCharts creates trend data for visualization
func (d *Dashboard) generateTrendCharts() TrendCharts {
	// Get historical data
	coverageHistory := d.coverageMonitor.history
	stabilityHistory := d.stabilityMonitor.history

	// Generate coverage trend
	coverageTrend := make([]TrendPoint, 0)
	for _, report := range coverageHistory {
		coverageTrend = append(coverageTrend, TrendPoint{
			Timestamp: report.Timestamp,
			Value:     report.OverallCoverage,
			Label:     fmt.Sprintf("%.1f%%", report.OverallCoverage),
		})
	}

	// Generate stability trend
	stabilityTrend := make([]TrendPoint, 0)
	for _, report := range stabilityHistory {
		stabilityTrend = append(stabilityTrend, TrendPoint{
			Timestamp: report.Timestamp,
			Value:     report.StabilityMetrics.OverallStability,
			Label:     fmt.Sprintf("%.1f%%", report.StabilityMetrics.OverallStability),
		})
	}

	// Generate performance trend (using average duration in seconds)
	performanceTrend := make([]TrendPoint, 0)
	for _, report := range stabilityHistory {
		durationSeconds := report.PerformanceData.AverageDuration.Seconds()
		performanceTrend = append(performanceTrend, TrendPoint{
			Timestamp: report.Timestamp,
			Value:     durationSeconds,
			Label:     report.PerformanceData.AverageDuration.String(),
		})
	}

	// Generate test count trend
	testCountTrend := make([]TrendPoint, 0)
	for _, report := range stabilityHistory {
		testCountTrend = append(testCountTrend, TrendPoint{
			Timestamp: report.Timestamp,
			Value:     float64(report.TestExecution.TotalTests),
			Label:     fmt.Sprintf("%d tests", report.TestExecution.TotalTests),
		})
	}

	return TrendCharts{
		CoverageTrend:    coverageTrend,
		StabilityTrend:   stabilityTrend,
		PerformanceTrend: performanceTrend,
		TestCountTrend:   testCountTrend,
	}
}

// generateAlerts creates monitoring alerts based on current state
func (d *Dashboard) generateAlerts(coverage *CoverageReport, stability *StabilityReport) []Alert {
	alerts := make([]Alert, 0)

	// Coverage alerts
	if !coverage.QualityGate.Passed {
		alerts = append(alerts, Alert{
			ID:        fmt.Sprintf("coverage-gate-%d", time.Now().Unix()),
			Type:      "coverage",
			Severity:  "error",
			Title:     "Coverage Quality Gate Failed",
			Message:   fmt.Sprintf("Overall coverage %.1f%% below threshold", coverage.OverallCoverage),
			Timestamp: time.Now(),
			Resolved:  false,
			ActionItems: []string{
				"Review uncovered code areas",
				"Add tests for critical functions",
				"Update coverage thresholds if appropriate",
			},
		})
	}

	// Stability alerts
	if stability.StabilityMetrics.OverallStability < 95 {
		severity := "warning"
		if stability.StabilityMetrics.OverallStability < 80 {
			severity = "error"
		}

		alerts = append(alerts, Alert{
			ID:        fmt.Sprintf("stability-%d", time.Now().Unix()),
			Type:      "stability",
			Severity:  severity,
			Title:     "Test Stability Below Threshold",
			Message:   fmt.Sprintf("Test stability %.1f%% below 95%% threshold", stability.StabilityMetrics.OverallStability),
			Timestamp: time.Now(),
			Resolved:  false,
			ActionItems: []string{
				"Investigate failing tests",
				"Fix flaky tests",
				"Review test environment setup",
			},
		})
	}

	// Flaky test alerts
	if len(stability.FlakyTests) > 0 {
		criticalFlaky := 0
		for _, test := range stability.FlakyTests {
			if test.Severity == "critical" || test.Severity == "high" {
				criticalFlaky++
			}
		}

		if criticalFlaky > 0 {
			alerts = append(alerts, Alert{
				ID:        fmt.Sprintf("flaky-tests-%d", time.Now().Unix()),
				Type:      "stability",
				Severity:  "warning",
				Title:     "Flaky Tests Detected",
				Message:   fmt.Sprintf("%d flaky tests found (%d critical/high severity)", len(stability.FlakyTests), criticalFlaky),
				Timestamp: time.Now(),
				Resolved:  false,
				ActionItems: []string{
					"Fix high-severity flaky tests first",
					"Investigate root causes of flakiness",
					"Consider test isolation improvements",
				},
			})
		}
	}

	// Performance alerts
	if stability.PerformanceData.AverageDuration > 2*time.Minute {
		alerts = append(alerts, Alert{
			ID:        fmt.Sprintf("performance-%d", time.Now().Unix()),
			Type:      "performance",
			Severity:  "warning",
			Title:     "Test Suite Performance Degraded",
			Message:   fmt.Sprintf("Average test duration %v exceeds 2 minute threshold", stability.PerformanceData.AverageDuration),
			Timestamp: time.Now(),
			Resolved:  false,
			ActionItems: []string{
				"Optimize slow tests",
				"Consider test parallelization",
				"Review test setup/teardown efficiency",
			},
		})
	}

	return alerts
}

// generateRecommendations creates actionable recommendations
func (d *Dashboard) generateRecommendations(coverage *CoverageReport, stability *StabilityReport, quality QualityOverview) []Recommendation {
	recommendations := make([]Recommendation, 0)

	// Coverage recommendations
	if coverage.OverallCoverage < 90 {
		priority := "medium"
		if coverage.OverallCoverage < 70 {
			priority = "high"
		}

		recommendations = append(recommendations, Recommendation{
			ID:          fmt.Sprintf("coverage-improve-%d", time.Now().Unix()),
			Category:    "coverage",
			Priority:    priority,
			Title:       "Improve Test Coverage",
			Description: fmt.Sprintf("Current coverage %.1f%% can be improved to meet 90%% target", coverage.OverallCoverage),
			Impact:      "Higher code quality and reduced bug risk",
			Effort:      "Medium - requires writing additional tests",
			Steps: []string{
				"Identify uncovered code areas using coverage report",
				"Prioritize critical business logic for testing",
				"Write unit tests for uncovered functions",
				"Add integration tests for complex workflows",
			},
			CreatedAt: time.Now(),
		})
	}

	// Stability recommendations
	if len(stability.FlakyTests) > 0 {
		recommendations = append(recommendations, Recommendation{
			ID:          fmt.Sprintf("flaky-tests-%d", time.Now().Unix()),
			Category:    "stability",
			Priority:    "high",
			Title:       "Fix Flaky Tests",
			Description: fmt.Sprintf("Address %d flaky tests to improve test reliability", len(stability.FlakyTests)),
			Impact:      "More reliable CI/CD pipeline and developer confidence",
			Effort:      "High - requires investigation and debugging",
			Steps: []string{
				"Analyze flaky test failure patterns",
				"Identify common causes (timing, dependencies, environment)",
				"Implement proper test isolation",
				"Add retry mechanisms where appropriate",
				"Consider test environment improvements",
			},
			CreatedAt: time.Now(),
		})
	}

	// Performance recommendations
	if len(stability.PerformanceData.SlowestTests) > 0 {
		recommendations = append(recommendations, Recommendation{
			ID:          fmt.Sprintf("performance-optimize-%d", time.Now().Unix()),
			Category:    "performance",
			Priority:    "medium",
			Title:       "Optimize Test Performance",
			Description: fmt.Sprintf("Optimize %d slow tests to improve overall test suite performance", len(stability.PerformanceData.SlowestTests)),
			Impact:      "Faster feedback loop and improved developer productivity",
			Effort:      "Medium - requires profiling and optimization",
			Steps: []string{
				"Profile slowest tests to identify bottlenecks",
				"Optimize test setup and teardown",
				"Consider test data optimization",
				"Implement test parallelization where possible",
				"Review external dependencies in tests",
			},
			CreatedAt: time.Now(),
		})
	}

	// Quality improvement recommendations
	if quality.OverallScore < 80 {
		recommendations = append(recommendations, Recommendation{
			ID:          fmt.Sprintf("quality-improve-%d", time.Now().Unix()),
			Category:    "quality",
			Priority:    "high",
			Title:       "Improve Overall Quality Score",
			Description: fmt.Sprintf("Current quality score %.1f%% needs improvement to reach B grade (80%%)", quality.OverallScore),
			Impact:      "Better code maintainability and reduced technical debt",
			Effort:      "High - requires comprehensive quality improvements",
			Steps: []string{
				"Focus on coverage improvements (40% weight)",
				"Address test stability issues (40% weight)",
				"Optimize test performance (20% weight)",
				"Implement quality monitoring automation",
				"Establish quality gates in CI/CD pipeline",
			},
			CreatedAt: time.Now(),
		})
	}

	// Sort recommendations by priority
	sort.Slice(recommendations, func(i, j int) bool {
		priorityOrder := map[string]int{"critical": 4, "high": 3, "medium": 2, "low": 1}
		return priorityOrder[recommendations[i].Priority] > priorityOrder[recommendations[j].Priority]
	})

	return recommendations
}

// saveDashboardData saves dashboard data as JSON
func (d *Dashboard) saveDashboardData(data *DashboardData) error {
	dataPath := filepath.Join(d.config.OutputDir, "dashboard_data.json")

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal dashboard data: %w", err)
	}

	if err := os.WriteFile(dataPath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write dashboard data: %w", err)
	}

	return nil
}

// generateHTMLDashboard creates an HTML dashboard
func (d *Dashboard) generateHTMLDashboard(data *DashboardData) error {
	htmlTemplate := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>FlowSpec Quality Dashboard</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; margin: 0; padding: 20px; background: #f5f5f5; }
        .container { max-width: 1200px; margin: 0 auto; }
        .header { background: white; padding: 20px; border-radius: 8px; margin-bottom: 20px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(300px, 1fr)); gap: 20px; }
        .card { background: white; padding: 20px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .metric { text-align: center; padding: 20px; }
        .metric-value { font-size: 2.5em; font-weight: bold; margin: 10px 0; }
        .metric-label { color: #666; font-size: 0.9em; }
        .status-excellent { color: #22c55e; }
        .status-good { color: #3b82f6; }
        .status-warning { color: #f59e0b; }
        .status-critical { color: #ef4444; }
        .alert { padding: 15px; margin: 10px 0; border-radius: 6px; border-left: 4px solid; }
        .alert-error { background: #fef2f2; border-color: #ef4444; color: #991b1b; }
        .alert-warning { background: #fffbeb; border-color: #f59e0b; color: #92400e; }
        .alert-info { background: #eff6ff; border-color: #3b82f6; color: #1e40af; }
        .recommendation { padding: 15px; margin: 10px 0; border-radius: 6px; background: #f8fafc; border: 1px solid #e2e8f0; }
        .priority-high { border-left: 4px solid #ef4444; }
        .priority-medium { border-left: 4px solid #f59e0b; }
        .priority-low { border-left: 4px solid #3b82f6; }
        .timestamp { color: #666; font-size: 0.8em; }
        .grade { font-size: 3em; font-weight: bold; padding: 20px; text-align: center; border-radius: 50%; width: 80px; height: 80px; margin: 0 auto; }
        .grade-A { background: #22c55e; color: white; }
        .grade-B { background: #3b82f6; color: white; }
        .grade-C { background: #f59e0b; color: white; }
        .grade-D { background: #f97316; color: white; }
        .grade-F { background: #ef4444; color: white; }
        .trend-up { color: #22c55e; }
        .trend-down { color: #ef4444; }
        .trend-stable { color: #6b7280; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🔍 FlowSpec Quality Dashboard</h1>
            <p class="timestamp">Generated: {{.GeneratedAt.Format "2006-01-02 15:04:05"}}</p>
        </div>

        <div class="grid">
            <div class="card">
                <h2>📊 Quality Overview</h2>
                <div class="metric">
                    <div class="grade grade-{{.QualityOverview.QualityGrade}}">{{.QualityOverview.QualityGrade}}</div>
                    <div class="metric-value status-{{.QualityOverview.Status}}">{{printf "%.1f" .QualityOverview.OverallScore}}%</div>
                    <div class="metric-label">Overall Quality Score</div>
                </div>
                <p><strong>Status:</strong> {{.QualityOverview.Status}}</p>
                <p><strong>Next Milestone:</strong> {{.QualityOverview.NextMilestone}}</p>
            </div>

            <div class="card">
                <h2>🧪 Test Coverage</h2>
                <div class="metric">
                    <div class="metric-value">{{printf "%.1f" .CoverageReport.OverallCoverage}}%</div>
                    <div class="metric-label">Overall Coverage</div>
                </div>
                <p><strong>Quality Gate:</strong> {{if .CoverageReport.QualityGate.Passed}}✅ Passed{{else}}❌ Failed{{end}}</p>
                <p><strong>Trend:</strong> 
                    {{if eq .CoverageReport.Trends.Direction "improving"}}📈 Improving{{else if eq .CoverageReport.Trends.Direction "declining"}}📉 Declining{{else}}📊 Stable{{end}}
                </p>
            </div>

            <div class="card">
                <h2>🔍 Test Stability</h2>
                <div class="metric">
                    <div class="metric-value">{{printf "%.1f" .StabilityReport.StabilityMetrics.OverallStability}}%</div>
                    <div class="metric-label">Stability Score</div>
                </div>
                <p><strong>Pass Rate:</strong> {{printf "%.1f" .StabilityReport.TestExecution.PassRate}}%</p>
                <p><strong>Flaky Tests:</strong> {{len .StabilityReport.FlakyTests}}</p>
            </div>

            <div class="card">
                <h2>⏱️ Performance</h2>
                <div class="metric">
                    <div class="metric-value">{{.StabilityReport.PerformanceData.AverageDuration}}</div>
                    <div class="metric-label">Average Test Duration</div>
                </div>
                <p><strong>Total Tests:</strong> {{.StabilityReport.TestExecution.TotalTests}}</p>
                <p><strong>Slow Tests:</strong> {{len .StabilityReport.PerformanceData.SlowestTests}}</p>
            </div>
        </div>

        {{if .Alerts}}
        <div class="card">
            <h2>🚨 Active Alerts</h2>
            {{range .Alerts}}
            <div class="alert alert-{{.Severity}}">
                <h3>{{.Title}}</h3>
                <p>{{.Message}}</p>
                <p class="timestamp">{{.Timestamp.Format "2006-01-02 15:04:05"}}</p>
            </div>
            {{end}}
        </div>
        {{end}}

        {{if .Recommendations}}
        <div class="card">
            <h2>💡 Recommendations</h2>
            {{range .Recommendations}}
            <div class="recommendation priority-{{.Priority}}">
                <h3>{{.Title}} <span style="font-size: 0.7em; color: #666;">[{{.Priority}} priority]</span></h3>
                <p>{{.Description}}</p>
                <p><strong>Impact:</strong> {{.Impact}}</p>
                <p><strong>Effort:</strong> {{.Effort}}</p>
            </div>
            {{end}}
        </div>
        {{end}}

        <div class="card">
            <h2>📈 Module Coverage Details</h2>
            {{range $name, $module := .CoverageReport.ModuleCoverage}}
            <div style="display: flex; justify-content: space-between; align-items: center; padding: 10px 0; border-bottom: 1px solid #eee;">
                <span>{{$module.Name}}</span>
                <span>
                    {{if eq $module.Status "pass"}}✅{{else if eq $module.Status "warning"}}⚠️{{else}}❌{{end}}
                    {{printf "%.1f" $module.Coverage}}%
                </span>
            </div>
            {{end}}
        </div>
    </div>

    <script>
        // Auto-refresh every 5 minutes
        setTimeout(function() {
            location.reload();
        }, 300000);
    </script>
</body>
</html>`

	tmpl, err := template.New("dashboard").Parse(htmlTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse HTML template: %w", err)
	}

	htmlPath := filepath.Join(d.config.OutputDir, "index.html")
	file, err := os.Create(htmlPath)
	if err != nil {
		return fmt.Errorf("failed to create HTML file: %w", err)
	}
	defer file.Close()

	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("failed to execute HTML template: %w", err)
	}

	return nil
}

// generateStaticFiles creates CSS and JS files for the dashboard
func (d *Dashboard) generateStaticFiles() error {
	// Create static directory
	staticDir := filepath.Join(d.config.OutputDir, "static")
	if err := os.MkdirAll(staticDir, 0755); err != nil {
		return fmt.Errorf("failed to create static directory: %w", err)
	}

	// Create a simple refresh script
	refreshScript := `
// Auto-refresh dashboard
setInterval(function() {
    fetch('/dashboard_data.json')
        .then(response => response.json())
        .then(data => {
            // Update timestamp
            document.querySelector('.timestamp').textContent = 'Generated: ' + new Date(data.generatedAt).toLocaleString();
        })
        .catch(error => console.log('Refresh failed:', error));
}, 60000); // Refresh every minute
`

	jsPath := filepath.Join(staticDir, "dashboard.js")
	if err := os.WriteFile(jsPath, []byte(refreshScript), 0644); err != nil {
		return fmt.Errorf("failed to write JavaScript file: %w", err)
	}

	return nil
}

// PrintSummary prints a summary of the dashboard generation
func (d *Dashboard) PrintSummary(data *DashboardData) {
	fmt.Println("\n📊 Quality Dashboard Summary")
	fmt.Println("============================")
	fmt.Printf("Overall Quality Score: %.1f%% (Grade: %s)\n", 
		data.QualityOverview.OverallScore, data.QualityOverview.QualityGrade)
	fmt.Printf("Coverage: %.1f%% | Stability: %.1f%% | Performance: %.1f%%\n",
		data.QualityOverview.CoverageScore, 
		data.QualityOverview.StabilityScore, 
		data.QualityOverview.PerformanceScore)
	
	if len(data.Alerts) > 0 {
		fmt.Printf("\n🚨 Active Alerts: %d\n", len(data.Alerts))
		for _, alert := range data.Alerts {
			fmt.Printf("  - %s: %s\n", alert.Severity, alert.Title)
		}
	}
	
	if len(data.Recommendations) > 0 {
		fmt.Printf("\n💡 Top Recommendations:\n")
		for i, rec := range data.Recommendations {
			if i >= 3 { // Show top 3
				break
			}
			fmt.Printf("  - [%s] %s\n", rec.Priority, rec.Title)
		}
	}
	
	fmt.Printf("\n🌐 Dashboard available at: %s/index.html\n", d.config.OutputDir)
}