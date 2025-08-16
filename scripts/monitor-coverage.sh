#!/bin/bash

# Enhanced Test Coverage Monitoring Script
# Integrates with the new monitoring system for comprehensive coverage analysis

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
COVERAGE_DIR=${COVERAGE_DIR:-"coverage"}
COVERAGE_THRESHOLD=${COVERAGE_THRESHOLD:-80}
MODULE_THRESHOLDS=${MODULE_THRESHOLDS:-"internal/engine:90,internal/models:90,internal/renderer:85"}
ALERT_WEBHOOK=${ALERT_WEBHOOK:-""}
HISTORY_DAYS=${HISTORY_DAYS:-30}
CONTINUOUS_MODE=${CONTINUOUS_MODE:-false}
MONITORING_INTERVAL=${MONITORING_INTERVAL:-300}

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to check if FlowSpec CLI is available
check_flowspec_cli() {
    if ! command -v ./flowspec-cli &> /dev/null && ! command -v flowspec-cli &> /dev/null; then
        print_error "FlowSpec CLI not found. Please build it first with 'make build'"
        return 1
    fi
    
    # Use local build if available, otherwise use installed version
    if [ -f "./flowspec-cli" ]; then
        FLOWSPEC_CLI="./flowspec-cli"
    else
        FLOWSPEC_CLI="flowspec-cli"
    fi
    
    print_status "Using FlowSpec CLI: $FLOWSPEC_CLI"
    return 0
}

# Function to run coverage analysis using the new monitoring system
run_coverage_analysis() {
    print_status "Running enhanced coverage analysis..."
    
    # Prepare arguments
    local args=(
        "monitor" "coverage"
        "--output-dir" "$COVERAGE_DIR"
        "--coverage-threshold" "$COVERAGE_THRESHOLD"
        "--history-days" "$HISTORY_DAYS"
    )
    
    # Add module thresholds if specified
    if [ -n "$MODULE_THRESHOLDS" ]; then
        # Convert comma-separated thresholds to flag format
        IFS=',' read -ra THRESHOLDS <<< "$MODULE_THRESHOLDS"
        for threshold in "${THRESHOLDS[@]}"; do
            args+=("--module-thresholds" "$threshold")
        done
    fi
    
    # Enable alerts if webhook is configured
    if [ -n "$ALERT_WEBHOOK" ]; then
        args+=("--alerts")
    fi
    
    # Run coverage analysis
    if $FLOWSPEC_CLI "${args[@]}"; then
        print_success "Coverage analysis completed successfully"
        return 0
    else
        print_error "Coverage analysis failed"
        return 1
    fi
}

# Function to run stability analysis
run_stability_analysis() {
    print_status "Running stability analysis..."
    
    local args=(
        "monitor" "stability"
        "--output-dir" "$COVERAGE_DIR"
        "--stability-threshold" "95.0"
        "--flaky-threshold" "3"
        "--performance-threshold" "2m"
        "--history-days" "$HISTORY_DAYS"
    )
    
    if $FLOWSPEC_CLI "${args[@]}"; then
        print_success "Stability analysis completed successfully"
        return 0
    else
        print_warning "Stability analysis completed with issues"
        return 1
    fi
}

# Function to generate comprehensive dashboard
generate_dashboard() {
    print_status "Generating quality monitoring dashboard..."
    
    local args=(
        "monitor" "dashboard"
        "--output-dir" "$COVERAGE_DIR"
        "--refresh-rate" "300"
        "--history-days" "$HISTORY_DAYS"
    )
    
    if $FLOWSPEC_CLI "${args[@]}"; then
        print_success "Quality dashboard generated successfully"
        print_status "Dashboard available at: $COVERAGE_DIR/index.html"
        return 0
    else
        print_error "Dashboard generation failed"
        return 1
    fi
}

# Function to send alerts via webhook
send_alert() {
    local message="$1"
    local severity="${2:-warning}"
    
    if [ -z "$ALERT_WEBHOOK" ]; then
        return 0
    fi
    
    print_status "Sending alert: $message"
    
    local payload=$(cat <<EOF
{
    "text": "FlowSpec Coverage Alert",
    "attachments": [
        {
            "color": "$( [ "$severity" = "error" ] && echo "danger" || echo "warning" )",
            "fields": [
                {
                    "title": "Alert",
                    "value": "$message",
                    "short": false
                },
                {
                    "title": "Timestamp",
                    "value": "$(date -u +"%Y-%m-%d %H:%M:%S UTC")",
                    "short": true
                },
                {
                    "title": "Project",
                    "value": "FlowSpec CLI",
                    "short": true
                }
            ]
        }
    ]
}
EOF
)
    
    if curl -X POST -H 'Content-type: application/json' --data "$payload" "$ALERT_WEBHOOK" &>/dev/null; then
        print_success "Alert sent successfully"
    else
        print_warning "Failed to send alert"
    fi
}

# Function to check coverage trends and send alerts
check_coverage_trends() {
    local coverage_file="$COVERAGE_DIR/coverage.json"
    local history_file="$COVERAGE_DIR/coverage_history.json"
    
    if [ ! -f "$coverage_file" ] || [ ! -f "$history_file" ]; then
        return 0
    fi
    
    # Extract current coverage
    local current_coverage=$(jq -r '.overallCoverage' "$coverage_file" 2>/dev/null || echo "0")
    
    # Check if coverage dropped significantly
    if [ -f "$history_file" ]; then
        local previous_coverage=$(jq -r '.[-2].overallCoverage // 0' "$history_file" 2>/dev/null || echo "0")
        local coverage_drop=$(echo "$previous_coverage - $current_coverage" | bc -l 2>/dev/null || echo "0")
        
        # Alert if coverage dropped by more than 5%
        if (( $(echo "$coverage_drop > 5" | bc -l) )); then
            send_alert "Coverage dropped by ${coverage_drop}% (from ${previous_coverage}% to ${current_coverage}%)" "error"
        fi
    fi
    
    # Alert if coverage is below threshold
    if (( $(echo "$current_coverage < $COVERAGE_THRESHOLD" | bc -l) )); then
        send_alert "Coverage ${current_coverage}% is below threshold ${COVERAGE_THRESHOLD}%" "warning"
    fi
}

# Function to run continuous monitoring
run_continuous_monitoring() {
    print_status "Starting continuous coverage monitoring (interval: ${MONITORING_INTERVAL}s)"
    
    while true; do
        print_status "Running monitoring cycle at $(date)"
        
        # Run full analysis
        if run_coverage_analysis && run_stability_analysis; then
            generate_dashboard
            check_coverage_trends
            print_success "Monitoring cycle completed successfully"
        else
            print_error "Monitoring cycle completed with errors"
            send_alert "Monitoring cycle failed" "error"
        fi
        
        print_status "Next monitoring cycle in ${MONITORING_INTERVAL} seconds..."
        sleep "$MONITORING_INTERVAL"
    done
}

# Function to setup monitoring infrastructure
setup_monitoring() {
    print_status "Setting up monitoring infrastructure..."
    
    # Create output directory
    mkdir -p "$COVERAGE_DIR"
    
    # Create monitoring configuration
    cat > "$COVERAGE_DIR/monitor_config.json" <<EOF
{
    "coverageThreshold": $COVERAGE_THRESHOLD,
    "moduleThresholds": {
        $(echo "$MODULE_THRESHOLDS" | sed 's/,/","/g' | sed 's/:/": /g' | sed 's/^/"/' | sed 's/$/"/')
    },
    "alertWebhook": "$ALERT_WEBHOOK",
    "historyDays": $HISTORY_DAYS,
    "monitoringInterval": $MONITORING_INTERVAL,
    "setupTimestamp": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
}
EOF
    
    print_success "Monitoring infrastructure setup complete"
}

# Function to display monitoring status
show_status() {
    print_status "Coverage Monitoring Status"
    echo "=========================="
    
    if [ -f "$COVERAGE_DIR/coverage.json" ]; then
        local coverage=$(jq -r '.overallCoverage' "$COVERAGE_DIR/coverage.json" 2>/dev/null || echo "N/A")
        local timestamp=$(jq -r '.timestamp' "$COVERAGE_DIR/coverage.json" 2>/dev/null || echo "N/A")
        echo "Last Coverage: $coverage% (at $timestamp)"
    else
        echo "Last Coverage: No data available"
    fi
    
    if [ -f "$COVERAGE_DIR/stability_report.json" ]; then
        local stability=$(jq -r '.stabilityMetrics.overallStability' "$COVERAGE_DIR/stability_report.json" 2>/dev/null || echo "N/A")
        echo "Last Stability: $stability%"
    else
        echo "Last Stability: No data available"
    fi
    
    if [ -f "$COVERAGE_DIR/index.html" ]; then
        echo "Dashboard: Available at $COVERAGE_DIR/index.html"
    else
        echo "Dashboard: Not generated"
    fi
    
    echo ""
    echo "Configuration:"
    echo "  Coverage Threshold: $COVERAGE_THRESHOLD%"
    echo "  Output Directory: $COVERAGE_DIR"
    echo "  History Days: $HISTORY_DAYS"
    echo "  Alert Webhook: $([ -n "$ALERT_WEBHOOK" ] && echo "Configured" || echo "Not configured")"
}

# Function to cleanup old monitoring data
cleanup_old_data() {
    print_status "Cleaning up old monitoring data..."
    
    # Remove files older than history days
    if [ -d "$COVERAGE_DIR" ]; then
        find "$COVERAGE_DIR" -name "*.json" -mtime +$HISTORY_DAYS -delete 2>/dev/null || true
        find "$COVERAGE_DIR" -name "*.html" -mtime +7 -delete 2>/dev/null || true  # Keep HTML for 7 days
        print_success "Cleanup completed"
    fi
}

# Main execution
main() {
    local command="${1:-run}"
    
    case "$command" in
        "run")
            check_flowspec_cli || exit 1
            setup_monitoring
            run_coverage_analysis
            run_stability_analysis
            generate_dashboard
            check_coverage_trends
            ;;
        "continuous")
            check_flowspec_cli || exit 1
            setup_monitoring
            run_continuous_monitoring
            ;;
        "status")
            show_status
            ;;
        "cleanup")
            cleanup_old_data
            ;;
        "setup")
            setup_monitoring
            ;;
        "dashboard")
            check_flowspec_cli || exit 1
            generate_dashboard
            ;;
        "help"|"-h"|"--help")
            echo "Enhanced Coverage Monitoring Script"
            echo "Usage: $0 [command]"
            echo ""
            echo "Commands:"
            echo "  run         Run complete monitoring analysis (default)"
            echo "  continuous  Run continuous monitoring"
            echo "  status      Show monitoring status"
            echo "  cleanup     Clean up old monitoring data"
            echo "  setup       Setup monitoring infrastructure"
            echo "  dashboard   Generate dashboard only"
            echo "  help        Show this help message"
            echo ""
            echo "Environment Variables:"
            echo "  COVERAGE_DIR           Output directory (default: coverage)"
            echo "  COVERAGE_THRESHOLD     Coverage threshold % (default: 80)"
            echo "  MODULE_THRESHOLDS      Module-specific thresholds (default: internal/engine:90,internal/models:90)"
            echo "  ALERT_WEBHOOK          Webhook URL for alerts"
            echo "  HISTORY_DAYS           Days to keep history (default: 30)"
            echo "  MONITORING_INTERVAL    Continuous monitoring interval in seconds (default: 300)"
            ;;
        *)
            print_error "Unknown command: $command"
            echo "Use '$0 help' for usage information"
            exit 1
            ;;
    esac
}

# Check dependencies
if ! command -v jq &> /dev/null; then
    print_warning "jq is not installed. Some features may not work properly."
fi

if ! command -v bc &> /dev/null; then
    print_error "bc is required but not installed. Please install bc."
    exit 1
fi

# Run main function
main "$@"