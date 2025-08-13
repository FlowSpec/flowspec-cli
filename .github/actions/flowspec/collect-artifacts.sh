#!/bin/bash

# FlowSpec Artifact Collection Script
# This script collects all relevant artifacts and debug information
# for upload in GitHub Actions, especially useful for failure scenarios

set -euo pipefail

# Configuration
ARTIFACTS_DIR="${1:-artifacts}"
EXIT_CODE="${2:-0}"
VERIFICATION_LOG="${3:-verification.log}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

# Create artifacts directory
create_artifacts_dir() {
    log_info "Creating artifacts directory: $ARTIFACTS_DIR"
    mkdir -p "$ARTIFACTS_DIR"
}

# Collect system information
collect_system_info() {
    log_info "Collecting system information..."
    
    cat > "$ARTIFACTS_DIR/system-info.txt" << EOF
FlowSpec CLI Verification - System Information
==============================================

Date: $(date -u '+%Y-%m-%d %H:%M:%S UTC')
Exit Code: $EXIT_CODE

System Information:
- OS: $(uname -s)
- Architecture: $(uname -m)
- Kernel: $(uname -r)
- Hostname: $(hostname)

Environment Variables:
- GITHUB_ACTIONS: ${GITHUB_ACTIONS:-}
- GITHUB_WORKFLOW: ${GITHUB_WORKFLOW:-}
- GITHUB_RUN_ID: ${GITHUB_RUN_ID:-}
- GITHUB_RUN_NUMBER: ${GITHUB_RUN_NUMBER:-}
- GITHUB_JOB: ${GITHUB_JOB:-}
- GITHUB_ACTOR: ${GITHUB_ACTOR:-}
- GITHUB_REPOSITORY: ${GITHUB_REPOSITORY:-}
- GITHUB_REF: ${GITHUB_REF:-}
- GITHUB_SHA: ${GITHUB_SHA:-}
- RUNNER_OS: ${RUNNER_OS:-}
- RUNNER_ARCH: ${RUNNER_ARCH:-}

FlowSpec CLI Version:
EOF

    # Try to get FlowSpec CLI version
    if command -v flowspec-cli >/dev/null 2>&1; then
        flowspec-cli --version >> "$ARTIFACTS_DIR/system-info.txt" 2>&1 || echo "Failed to get version" >> "$ARTIFACTS_DIR/system-info.txt"
    else
        echo "FlowSpec CLI not found in PATH" >> "$ARTIFACTS_DIR/system-info.txt"
    fi
    
    log_success "System information collected"
}

# Collect verification logs
collect_verification_logs() {
    log_info "Collecting verification logs..."
    
    # Copy main verification log
    if [ -f "$VERIFICATION_LOG" ]; then
        cp "$VERIFICATION_LOG" "$ARTIFACTS_DIR/"
        log_success "Verification log copied"
    else
        log_warning "Verification log not found: $VERIFICATION_LOG"
        echo "Verification log not found: $VERIFICATION_LOG" > "$ARTIFACTS_DIR/verification-log-missing.txt"
    fi
    
    # Collect any additional log files
    for log_file in *.log flowspec*.log; do
        if [ -f "$log_file" ] && [ "$log_file" != "$VERIFICATION_LOG" ]; then
            cp "$log_file" "$ARTIFACTS_DIR/"
            log_info "Additional log file copied: $log_file"
        fi
    done
}

# Collect FlowSpec generated artifacts
collect_flowspec_artifacts() {
    log_info "Collecting FlowSpec generated artifacts..."
    
    # Standard FlowSpec artifacts
    local artifacts_found=false
    
    # Summary JSON
    if [ -f "artifacts/flowspec-summary.json" ]; then
        # Already in artifacts directory
        log_success "Found summary JSON"
        artifacts_found=true
    elif [ -f "flowspec-summary.json" ]; then
        cp "flowspec-summary.json" "$ARTIFACTS_DIR/"
        log_success "Summary JSON copied"
        artifacts_found=true
    fi
    
    # JUnit XML
    if [ -f "artifacts/flowspec-report.xml" ]; then
        # Already in artifacts directory
        log_success "Found JUnit XML report"
        artifacts_found=true
    elif [ -f "flowspec-report.xml" ]; then
        cp "flowspec-report.xml" "$ARTIFACTS_DIR/"
        log_success "JUnit XML report copied"
        artifacts_found=true
    fi
    
    # Human readable report
    if [ -f "flowspec-report.txt" ]; then
        cp "flowspec-report.txt" "$ARTIFACTS_DIR/"
        log_success "Human readable report copied"
        artifacts_found=true
    fi
    
    # Any other FlowSpec generated files
    for artifact in flowspec-*.json flowspec-*.xml flowspec-*.txt flowspec-*.yaml flowspec-*.yml; do
        if [ -f "$artifact" ]; then
            cp "$artifact" "$ARTIFACTS_DIR/"
            log_info "Additional FlowSpec artifact copied: $artifact"
            artifacts_found=true
        fi
    done
    
    if [ "$artifacts_found" = false ]; then
        log_warning "No FlowSpec artifacts found"
        echo "No FlowSpec artifacts were generated" > "$ARTIFACTS_DIR/no-artifacts.txt"
    fi
}

# Collect debug information for failures
collect_debug_info() {
    if [ "$EXIT_CODE" -ne 0 ]; then
        log_info "Collecting debug information for failure (exit code: $EXIT_CODE)..."
        
        # Create debug info file
        cat > "$ARTIFACTS_DIR/debug-info.txt" << EOF
FlowSpec CLI Verification - Debug Information
============================================

Exit Code: $EXIT_CODE
Failure Category: $(get_failure_category "$EXIT_CODE")

Working Directory Contents:
EOF
        
        # List current directory contents
        ls -la >> "$ARTIFACTS_DIR/debug-info.txt" 2>&1 || echo "Failed to list directory" >> "$ARTIFACTS_DIR/debug-info.txt"
        
        echo "" >> "$ARTIFACTS_DIR/debug-info.txt"
        echo "Environment Variables:" >> "$ARTIFACTS_DIR/debug-info.txt"
        env | grep -E "(FLOWSPEC|GITHUB|CI)" | sort >> "$ARTIFACTS_DIR/debug-info.txt" 2>&1 || echo "Failed to get environment" >> "$ARTIFACTS_DIR/debug-info.txt"
        
        # Collect any error files
        for error_file in error.log stderr.log *.err; do
            if [ -f "$error_file" ]; then
                cp "$error_file" "$ARTIFACTS_DIR/"
                log_info "Error file copied: $error_file"
            fi
        done
        
        log_success "Debug information collected"
    fi
}

# Get failure category based on exit code
get_failure_category() {
    local exit_code="$1"
    
    case "$exit_code" in
        0)
            echo "Success"
            ;;
        1)
            echo "Validation Failed - Specifications do not match traces"
            ;;
        2)
            echo "Contract Format Error - Invalid YAML or specification format"
            ;;
        3)
            echo "Parse Error - Unable to parse trace or log files"
            ;;
        4)
            echo "System Error - Runtime or environment issue"
            ;;
        64)
            echo "Usage Error - Invalid command line arguments"
            ;;
        *)
            echo "Unknown Error"
            ;;
    esac
}

# Create artifact manifest
create_artifact_manifest() {
    log_info "Creating artifact manifest..."
    
    cat > "$ARTIFACTS_DIR/manifest.json" << EOF
{
  "timestamp": "$(date -u '+%Y-%m-%dT%H:%M:%SZ')",
  "exit_code": $EXIT_CODE,
  "failure_category": "$(get_failure_category "$EXIT_CODE")",
  "github": {
    "workflow": "${GITHUB_WORKFLOW:-}",
    "run_id": "${GITHUB_RUN_ID:-}",
    "run_number": "${GITHUB_RUN_NUMBER:-}",
    "job": "${GITHUB_JOB:-}",
    "actor": "${GITHUB_ACTOR:-}",
    "repository": "${GITHUB_REPOSITORY:-}",
    "ref": "${GITHUB_REF:-}",
    "sha": "${GITHUB_SHA:-}"
  },
  "system": {
    "os": "$(uname -s)",
    "arch": "$(uname -m)",
    "runner_os": "${RUNNER_OS:-}",
    "runner_arch": "${RUNNER_ARCH:-}"
  },
  "artifacts": [
EOF

    # List all files in artifacts directory
    local first=true
    for file in "$ARTIFACTS_DIR"/*; do
        if [ -f "$file" ] && [ "$(basename "$file")" != "manifest.json" ]; then
            if [ "$first" = true ]; then
                first=false
            else
                echo "," >> "$ARTIFACTS_DIR/manifest.json"
            fi
            
            local filename=$(basename "$file")
            local size=$(stat -f%z "$file" 2>/dev/null || stat -c%s "$file" 2>/dev/null || echo "unknown")
            
            cat >> "$ARTIFACTS_DIR/manifest.json" << EOF
    {
      "name": "$filename",
      "size": $size,
      "type": "$(get_file_type "$filename")"
    }EOF
        fi
    done
    
    echo "" >> "$ARTIFACTS_DIR/manifest.json"
    echo "  ]" >> "$ARTIFACTS_DIR/manifest.json"
    echo "}" >> "$ARTIFACTS_DIR/manifest.json"
    
    log_success "Artifact manifest created"
}

# Get file type based on extension
get_file_type() {
    local filename="$1"
    
    case "$filename" in
        *.json)
            echo "json"
            ;;
        *.xml)
            echo "xml"
            ;;
        *.log|*.txt)
            echo "text"
            ;;
        *.yaml|*.yml)
            echo "yaml"
            ;;
        *)
            echo "unknown"
            ;;
    esac
}

# Generate summary for GitHub Actions
generate_github_summary() {
    log_info "Generating GitHub Actions summary..."
    
    if [ -n "${GITHUB_STEP_SUMMARY:-}" ]; then
        cat >> "$GITHUB_STEP_SUMMARY" << EOF

## 📁 Artifacts Collected

The following artifacts have been collected and will be uploaded:

EOF
        
        # List artifacts
        for file in "$ARTIFACTS_DIR"/*; do
            if [ -f "$file" ]; then
                local filename=$(basename "$file")
                local size=$(stat -f%z "$file" 2>/dev/null || stat -c%s "$file" 2>/dev/null || echo "unknown")
                echo "- **$filename** (${size} bytes)" >> "$GITHUB_STEP_SUMMARY"
            fi
        done
        
        if [ "$EXIT_CODE" -ne 0 ]; then
            cat >> "$GITHUB_STEP_SUMMARY" << EOF

### 🔍 Debug Information

Exit code: \`$EXIT_CODE\` - $(get_failure_category "$EXIT_CODE")

Additional debug information has been collected to help diagnose the issue.
EOF
        fi
        
        log_success "GitHub Actions summary updated"
    else
        log_warning "GITHUB_STEP_SUMMARY not available"
    fi
}

# Main function
main() {
    log_info "Starting artifact collection..."
    log_info "Artifacts directory: $ARTIFACTS_DIR"
    log_info "Exit code: $EXIT_CODE"
    log_info "Verification log: $VERIFICATION_LOG"
    
    create_artifacts_dir
    collect_system_info
    collect_verification_logs
    collect_flowspec_artifacts
    collect_debug_info
    create_artifact_manifest
    generate_github_summary
    
    # Final summary
    local artifact_count=$(find "$ARTIFACTS_DIR" -type f | wc -l)
    log_success "Artifact collection completed successfully!"
    log_info "Total artifacts collected: $artifact_count"
    
    # List all collected artifacts
    log_info "Collected artifacts:"
    for file in "$ARTIFACTS_DIR"/*; do
        if [ -f "$file" ]; then
            local filename=$(basename "$file")
            local size=$(stat -f%z "$file" 2>/dev/null || stat -c%s "$file" 2>/dev/null || echo "unknown")
            echo "  - $filename (${size} bytes)"
        fi
    done
}

# Run main function if script is executed directly
if [ "${BASH_SOURCE[0]}" = "${0}" ]; then
    main "$@"
fi