#!/bin/bash

# Test Coverage Script for FlowSpec CLI
# This script runs comprehensive tests and generates coverage reports

set -e

echo "🧪 Running comprehensive test suite with coverage analysis..."

# Create coverage directory
mkdir -p coverage

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

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

# Function to run tests for a specific module
run_module_tests() {
    local module=$1
    local min_coverage=${2:-80}
    
    print_status "Testing module: $module"
    
    # Run tests with coverage
    if go test -v -coverprofile=coverage/${module//\//_}.out -timeout=60s ./$module; then
        # Extract coverage percentage
        if [ -f "coverage/${module//\//_}.out" ]; then
            coverage=$(go tool cover -func=coverage/${module//\//_}.out | grep total | awk '{print $3}' | sed 's/%//')
            
            if (( $(echo "$coverage >= $min_coverage" | bc -l) )); then
                print_success "$module: ${coverage}% coverage (target: ${min_coverage}%)"
                return 0
            else
                print_warning "$module: ${coverage}% coverage (below target: ${min_coverage}%)"
                return 1
            fi
        else
            print_error "$module: No coverage file generated"
            return 1
        fi
    else
        print_error "$module: Tests failed"
        return 1
    fi
}

# Function to generate HTML coverage report
generate_html_report() {
    local module=$1
    if [ -f "coverage/${module//\//_}.out" ]; then
        go tool cover -html=coverage/${module//\//_}.out -o coverage/${module//\//_}.html
        print_status "HTML report generated: coverage/${module//\//_}.html"
    fi
}

# Main test execution
main() {
    print_status "Starting comprehensive test coverage analysis..."
    
    # List of modules to test with their minimum coverage requirements
    modules=(
        "cmd/flowspec-cli:80"
        "internal/engine:85"
        "internal/ingestor:80"
        "internal/models:90"
        "internal/parser:80"
        "internal/renderer:85"
    )
    
    # Track overall results
    total_modules=0
    passed_modules=0
    failed_modules=()
    
    # Test each module
    for module_spec in "${modules[@]}"; do
        module=$(echo "$module_spec" | cut -d: -f1)
        min_coverage=$(echo "$module_spec" | cut -d: -f2)
        
        total_modules=$((total_modules + 1))
        
        if run_module_tests "$module" "$min_coverage"; then
            passed_modules=$((passed_modules + 1))
            generate_html_report "$module"
        else
            failed_modules+=("$module")
        fi
        
        echo "" # Add spacing between modules
    done
    
    # Generate combined coverage report
    print_status "Generating combined coverage report..."
    
    # Combine all coverage files
    echo "mode: set" > coverage/combined.out
    for module_spec in "${modules[@]}"; do
        module=$(echo "$module_spec" | cut -d: -f1)
        if [ -f "coverage/${module//\//_}.out" ]; then
            tail -n +2 "coverage/${module//\//_}.out" >> coverage/combined.out
        fi
    done
    
    # Generate combined HTML report
    if [ -f "coverage/combined.out" ]; then
        go tool cover -html=coverage/combined.out -o coverage/combined.html
        
        # Calculate overall coverage
        overall_coverage=$(go tool cover -func=coverage/combined.out | grep total | awk '{print $3}' | sed 's/%//')
        
        print_status "Overall coverage: ${overall_coverage}%"
        print_status "Combined HTML report: coverage/combined.html"
    fi
    
    # Print summary
    echo ""
    print_status "=== TEST COVERAGE SUMMARY ==="
    print_status "Total modules tested: $total_modules"
    print_success "Modules passed: $passed_modules"
    
    if [ ${#failed_modules[@]} -gt 0 ]; then
        print_error "Modules failed: ${#failed_modules[@]}"
        for module in "${failed_modules[@]}"; do
            print_error "  - $module"
        done
    fi
    
    # Check if overall target is met
    if (( $(echo "$overall_coverage >= 80" | bc -l) )); then
        print_success "✅ Overall coverage target (80%) achieved: ${overall_coverage}%"
        
        # Run additional quality checks
        print_status "Running additional quality checks..."
        
        # Check for race conditions
        print_status "Checking for race conditions..."
        if go test -race -short ./...; then
            print_success "✅ No race conditions detected"
        else
            print_warning "⚠️  Race conditions detected"
        fi
        
        # Run vet
        print_status "Running go vet..."
        if go vet ./...; then
            print_success "✅ Go vet passed"
        else
            print_warning "⚠️  Go vet found issues"
        fi
        
        # Check for inefficient assignments
        if command -v ineffassign &> /dev/null; then
            print_status "Checking for inefficient assignments..."
            if ineffassign ./...; then
                print_success "✅ No inefficient assignments found"
            else
                print_warning "⚠️  Inefficient assignments found"
            fi
        fi
        
        return 0
    else
        print_error "❌ Overall coverage target (80%) not met: ${overall_coverage}%"
        return 1
    fi
}

# Cleanup function
cleanup() {
    print_status "Cleaning up temporary files..."
    # Keep coverage files for analysis
}

# Set trap for cleanup
trap cleanup EXIT

# Check dependencies
if ! command -v bc &> /dev/null; then
    print_error "bc is required but not installed. Please install bc."
    exit 1
fi

# Run main function
main "$@"