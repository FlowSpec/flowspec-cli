#!/bin/bash

# Demo script to show the monitoring system functionality

set -e

echo "🔍 FlowSpec Quality Monitoring System Demo"
echo "=========================================="

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

# Create demo coverage directory
DEMO_DIR="coverage-demo"
mkdir -p "$DEMO_DIR"

print_status "Created demo directory: $DEMO_DIR"

# Run basic tests to generate coverage data
print_status "Running tests to generate coverage data..."
go test -coverprofile="$DEMO_DIR/coverage.out" ./internal/monitor || true

if [ -f "$DEMO_DIR/coverage.out" ]; then
    print_success "Coverage data generated successfully"
    
    # Generate HTML report
    go tool cover -html="$DEMO_DIR/coverage.out" -o "$DEMO_DIR/coverage.html"
    print_success "HTML coverage report generated: $DEMO_DIR/coverage.html"
    
    # Show coverage summary
    echo ""
    print_status "Coverage Summary:"
    go tool cover -func="$DEMO_DIR/coverage.out" | tail -1
    
    # Generate function-level coverage
    echo ""
    print_status "Function-level Coverage:"
    go tool cover -func="$DEMO_DIR/coverage.out" | head -10
else
    print_warning "No coverage data generated"
fi

# Create sample monitoring configuration
print_status "Creating sample monitoring configuration..."
cat > "$DEMO_DIR/monitor_config.json" <<EOF
{
    "coverageThreshold": 80,
    "moduleThresholds": {
        "internal/engine": 90,
        "internal/models": 90,
        "internal/renderer": 85,
        "internal/parser": 80,
        "internal/ingestor": 80,
        "cmd/flowspec-cli": 80
    },
    "stabilityThreshold": 95,
    "performanceThreshold": "2m",
    "historyDays": 30,
    "alertsEnabled": true,
    "setupTimestamp": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
}
EOF

print_success "Monitoring configuration created: $DEMO_DIR/monitor_config.json"

# Create sample dashboard data
print_status "Creating sample dashboard data..."
cat > "$DEMO_DIR/dashboard_data.json" <<EOF
{
    "generatedAt": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")",
    "qualityOverview": {
        "overallScore": 82.5,
        "coverageScore": 78.0,
        "stabilityScore": 92.0,
        "performanceScore": 88.0,
        "qualityGrade": "B",
        "status": "good",
        "lastImprovement": "Coverage increased by 3.2%",
        "nextMilestone": "Achieve A grade (90% quality score)"
    },
    "alerts": [
        {
            "type": "coverage",
            "severity": "warning",
            "title": "Coverage Below Threshold",
            "message": "Module internal/engine coverage 75% below threshold 90%",
            "timestamp": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
        }
    ],
    "recommendations": [
        {
            "category": "coverage",
            "priority": "high",
            "title": "Improve Test Coverage",
            "description": "Current coverage 78% can be improved to meet 90% target",
            "impact": "Higher code quality and reduced bug risk"
        },
        {
            "category": "stability",
            "priority": "medium",
            "title": "Fix Flaky Tests",
            "description": "Address 2 flaky tests to improve test reliability",
            "impact": "More reliable CI/CD pipeline"
        }
    ]
}
EOF

print_success "Sample dashboard data created: $DEMO_DIR/dashboard_data.json"

# Create simple HTML dashboard
print_status "Creating demo HTML dashboard..."
cat > "$DEMO_DIR/index.html" <<EOF
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>FlowSpec Quality Dashboard - Demo</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; margin: 0; padding: 20px; background: #f5f5f5; }
        .container { max-width: 1200px; margin: 0 auto; }
        .header { background: white; padding: 20px; border-radius: 8px; margin-bottom: 20px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(300px, 1fr)); gap: 20px; }
        .card { background: white; padding: 20px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .metric { text-align: center; padding: 20px; }
        .metric-value { font-size: 2.5em; font-weight: bold; margin: 10px 0; color: #3b82f6; }
        .metric-label { color: #666; font-size: 0.9em; }
        .grade { font-size: 3em; font-weight: bold; padding: 20px; text-align: center; border-radius: 50%; width: 80px; height: 80px; margin: 0 auto; background: #3b82f6; color: white; }
        .alert { padding: 15px; margin: 10px 0; border-radius: 6px; border-left: 4px solid #f59e0b; background: #fffbeb; color: #92400e; }
        .recommendation { padding: 15px; margin: 10px 0; border-radius: 6px; background: #f8fafc; border: 1px solid #e2e8f0; border-left: 4px solid #ef4444; }
        .demo-note { background: #dbeafe; border: 1px solid #3b82f6; padding: 15px; border-radius: 6px; margin-bottom: 20px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="demo-note">
            <h3>🚀 Demo Mode</h3>
            <p>This is a demonstration of the FlowSpec Quality Monitoring Dashboard. In production, this would show real-time data from your test suite.</p>
        </div>
        
        <div class="header">
            <h1>🔍 FlowSpec Quality Dashboard</h1>
            <p>Generated: $(date)</p>
        </div>

        <div class="grid">
            <div class="card">
                <h2>📊 Quality Overview</h2>
                <div class="metric">
                    <div class="grade">B</div>
                    <div class="metric-value">82.5%</div>
                    <div class="metric-label">Overall Quality Score</div>
                </div>
                <p><strong>Status:</strong> Good</p>
                <p><strong>Next Milestone:</strong> Achieve A grade (90% quality score)</p>
            </div>

            <div class="card">
                <h2>🧪 Test Coverage</h2>
                <div class="metric">
                    <div class="metric-value">78.0%</div>
                    <div class="metric-label">Overall Coverage</div>
                </div>
                <p><strong>Trend:</strong> 📈 Improving (+3.2%)</p>
            </div>

            <div class="card">
                <h2>🔍 Test Stability</h2>
                <div class="metric">
                    <div class="metric-value">92.0%</div>
                    <div class="metric-label">Stability Score</div>
                </div>
                <p><strong>Flaky Tests:</strong> 2</p>
            </div>

            <div class="card">
                <h2>⏱️ Performance</h2>
                <div class="metric">
                    <div class="metric-value">1m 30s</div>
                    <div class="metric-label">Average Test Duration</div>
                </div>
                <p><strong>Performance Score:</strong> 88%</p>
            </div>
        </div>

        <div class="card">
            <h2>🚨 Active Alerts</h2>
            <div class="alert">
                <h3>Coverage Below Threshold</h3>
                <p>Module internal/engine coverage 75% below threshold 90%</p>
            </div>
        </div>

        <div class="card">
            <h2>💡 Recommendations</h2>
            <div class="recommendation">
                <h3>Improve Test Coverage <span style="font-size: 0.7em; color: #666;">[high priority]</span></h3>
                <p>Current coverage 78% can be improved to meet 90% target</p>
                <p><strong>Impact:</strong> Higher code quality and reduced bug risk</p>
            </div>
            <div class="recommendation">
                <h3>Fix Flaky Tests <span style="font-size: 0.7em; color: #666;">[medium priority]</span></h3>
                <p>Address 2 flaky tests to improve test reliability</p>
                <p><strong>Impact:</strong> More reliable CI/CD pipeline</p>
            </div>
        </div>

        <div class="card">
            <h2>📈 Features Implemented</h2>
            <ul>
                <li>✅ Test coverage analysis and reporting</li>
                <li>✅ Coverage trend analysis over time</li>
                <li>✅ Quality gate evaluation</li>
                <li>✅ HTML and JSON coverage reports</li>
                <li>✅ Test stability monitoring</li>
                <li>✅ Flaky test detection</li>
                <li>✅ Performance analysis</li>
                <li>✅ Quality dashboard generation</li>
                <li>✅ Automated alerts and recommendations</li>
                <li>✅ GitHub Actions integration</li>
                <li>✅ Continuous monitoring support</li>
            </ul>
        </div>
    </div>
</body>
</html>
EOF

print_success "Demo HTML dashboard created: $DEMO_DIR/index.html"

# Show summary
echo ""
print_status "Demo Summary"
echo "============"
echo "📁 Demo directory: $DEMO_DIR"
echo "📊 Dashboard: $DEMO_DIR/index.html"
echo "📄 Coverage report: $DEMO_DIR/coverage.html"
echo "⚙️  Configuration: $DEMO_DIR/monitor_config.json"
echo "📈 Dashboard data: $DEMO_DIR/dashboard_data.json"
echo ""
print_success "Quality monitoring system demo completed!"
echo ""
print_status "Key Features Implemented:"
echo "• Comprehensive coverage analysis with module-specific thresholds"
echo "• Test stability monitoring with flaky test detection"
echo "• Performance analysis and trend tracking"
echo "• Quality dashboard with alerts and recommendations"
echo "• GitHub Actions workflow for automated monitoring"
echo "• Continuous monitoring support with configurable intervals"
echo "• HTML and JSON reporting formats"
echo "• Historical trend analysis and predictions"
echo ""
print_status "To view the demo dashboard, open: $DEMO_DIR/index.html in your browser"