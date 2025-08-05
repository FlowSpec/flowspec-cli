#!/bin/bash

# FlowSpec CLI Performance Testing Script
# This script runs comprehensive performance tests and generates reports

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
PERFORMANCE_LOG="performance_results.log"
BENCHMARK_LOG="benchmark_results.log"
MEMORY_LOG="memory_usage.log"
REPORT_DIR="performance_reports"

echo -e "${BLUE}FlowSpec CLI Performance Testing Suite${NC}"
echo "========================================"

# Create report directory
mkdir -p "$REPORT_DIR"

# Function to log with timestamp
log_with_timestamp() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$PERFORMANCE_LOG"
}

# Function to run performance tests
run_performance_tests() {
    echo -e "${YELLOW}Running Performance Tests...${NC}"
    log_with_timestamp "Starting performance test suite"
    
    # Run large scale parsing performance test
    echo "Testing large scale parsing (1,000 files, 200 ServiceSpecs)..."
    if go test -v -run TestLargeScaleParsingPerformance ./cmd/flowspec-cli/ -timeout 60m > "$REPORT_DIR/large_scale_test.log" 2>&1; then
        echo -e "${GREEN}✓ Large scale parsing test passed${NC}"
        log_with_timestamp "Large scale parsing test: PASSED"
    else
        echo -e "${RED}✗ Large scale parsing test failed${NC}"
        log_with_timestamp "Large scale parsing test: FAILED"
        cat "$REPORT_DIR/large_scale_test.log" | tail -20
    fi
    
    # Run memory usage limits test
    echo "Testing memory usage limits (100MB file, <500MB memory)..."
    if go test -v -run TestMemoryUsageLimits ./cmd/flowspec-cli/ -timeout 30m > "$REPORT_DIR/memory_test.log" 2>&1; then
        echo -e "${GREEN}✓ Memory usage test passed${NC}"
        log_with_timestamp "Memory usage test: PASSED"
    else
        echo -e "${RED}✗ Memory usage test failed${NC}"
        log_with_timestamp "Memory usage test: FAILED"
        cat "$REPORT_DIR/memory_test.log" | tail -20
    fi
    
    # Run concurrency safety test
    echo "Testing concurrency safety..."
    if go test -v -run TestConcurrencySafety ./cmd/flowspec-cli/ -timeout 10m > "$REPORT_DIR/concurrency_test.log" 2>&1; then
        echo -e "${GREEN}✓ Concurrency safety test passed${NC}"
        log_with_timestamp "Concurrency safety test: PASSED"
    else
        echo -e "${RED}✗ Concurrency safety test failed${NC}"
        log_with_timestamp "Concurrency safety test: FAILED"
        cat "$REPORT_DIR/concurrency_test.log" | tail -20
    fi
    
    # Run performance regression test
    echo "Testing performance regression baselines..."
    if go test -v -run TestPerformanceRegression ./cmd/flowspec-cli/ -timeout 15m > "$REPORT_DIR/regression_test.log" 2>&1; then
        echo -e "${GREEN}✓ Performance regression test passed${NC}"
        log_with_timestamp "Performance regression test: PASSED"
    else
        echo -e "${RED}✗ Performance regression test failed${NC}"
        log_with_timestamp "Performance regression test: FAILED"
        cat "$REPORT_DIR/regression_test.log" | tail -20
    fi
}

# Function to run benchmarks
run_benchmarks() {
    echo -e "${YELLOW}Running Benchmarks...${NC}"
    log_with_timestamp "Starting benchmark suite"
    
    # Run CLI execution benchmark
    echo "Benchmarking CLI execution..."
    go test -bench=BenchmarkCLIExecution -benchmem -count=3 ./cmd/flowspec-cli/ > "$REPORT_DIR/cli_benchmark.log" 2>&1
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓ CLI execution benchmark completed${NC}"
        log_with_timestamp "CLI execution benchmark: COMPLETED"
    else
        echo -e "${RED}✗ CLI execution benchmark failed${NC}"
        log_with_timestamp "CLI execution benchmark: FAILED"
    fi
    
    # Run parsing-only benchmark
    echo "Benchmarking parsing performance..."
    go test -bench=BenchmarkParsingOnly -benchmem -count=3 ./cmd/flowspec-cli/ > "$REPORT_DIR/parsing_benchmark.log" 2>&1
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓ Parsing benchmark completed${NC}"
        log_with_timestamp "Parsing benchmark: COMPLETED"
    else
        echo -e "${RED}✗ Parsing benchmark failed${NC}"
        log_with_timestamp "Parsing benchmark: FAILED"
    fi
    
    # Run ingestor benchmarks
    echo "Benchmarking trace ingestion..."
    go test -bench=BenchmarkStreamingIngestor -benchmem -count=3 ./internal/ingestor/ > "$REPORT_DIR/ingestor_benchmark.log" 2>&1
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓ Ingestor benchmark completed${NC}"
        log_with_timestamp "Ingestor benchmark: COMPLETED"
    else
        echo -e "${RED}✗ Ingestor benchmark failed${NC}"
        log_with_timestamp "Ingestor benchmark: FAILED"
    fi
    
    # Run parser benchmarks
    echo "Benchmarking parser performance..."
    go test -bench=BenchmarkParser -benchmem -count=3 ./internal/parser/ > "$REPORT_DIR/parser_benchmark.log" 2>&1
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓ Parser benchmark completed${NC}"
        log_with_timestamp "Parser benchmark: COMPLETED"
    else
        echo -e "${RED}✗ Parser benchmark failed${NC}"
        log_with_timestamp "Parser benchmark: FAILED"
    fi
}

# Function to generate performance report
generate_report() {
    echo -e "${YELLOW}Generating Performance Report...${NC}"
    
    REPORT_FILE="$REPORT_DIR/performance_summary.md"
    
    cat > "$REPORT_FILE" << EOF
# FlowSpec CLI Performance Test Report

Generated on: $(date)
Go Version: $(go version)
System: $(uname -a)

## Test Results Summary

EOF
    
    # Add test results
    if grep -q "Large scale parsing test: PASSED" "$PERFORMANCE_LOG"; then
        echo "✅ **Large Scale Parsing Test**: PASSED" >> "$REPORT_FILE"
    else
        echo "❌ **Large Scale Parsing Test**: FAILED" >> "$REPORT_FILE"
    fi
    
    if grep -q "Memory usage test: PASSED" "$PERFORMANCE_LOG"; then
        echo "✅ **Memory Usage Test**: PASSED" >> "$REPORT_FILE"
    else
        echo "❌ **Memory Usage Test**: FAILED" >> "$REPORT_FILE"
    fi
    
    if grep -q "Concurrency safety test: PASSED" "$PERFORMANCE_LOG"; then
        echo "✅ **Concurrency Safety Test**: PASSED" >> "$REPORT_FILE"
    else
        echo "❌ **Concurrency Safety Test**: FAILED" >> "$REPORT_FILE"
    fi
    
    if grep -q "Performance regression test: PASSED" "$PERFORMANCE_LOG"; then
        echo "✅ **Performance Regression Test**: PASSED" >> "$REPORT_FILE"
    else
        echo "❌ **Performance Regression Test**: FAILED" >> "$REPORT_FILE"
    fi
    
    cat >> "$REPORT_FILE" << EOF

## Detailed Results

### Large Scale Parsing Performance
- **Requirement**: Process 1,000 files with 200 ServiceSpecs in <30 seconds
- **Test File**: [large_scale_test.log](large_scale_test.log)

### Memory Usage Limits
- **Requirement**: Process 100MB trace file with <500MB memory usage
- **Test File**: [memory_test.log](memory_test.log)

### Concurrency Safety
- **Test**: Multiple concurrent CLI executions
- **Test File**: [concurrency_test.log](concurrency_test.log)

### Performance Regression
- **Test**: Baseline performance measurements
- **Test File**: [regression_test.log](regression_test.log)

## Benchmark Results

### CLI Execution Benchmark
EOF
    
    if [ -f "$REPORT_DIR/cli_benchmark.log" ]; then
        echo '```' >> "$REPORT_FILE"
        cat "$REPORT_DIR/cli_benchmark.log" >> "$REPORT_FILE"
        echo '```' >> "$REPORT_FILE"
    fi
    
    cat >> "$REPORT_FILE" << EOF

### Parsing Benchmark
EOF
    
    if [ -f "$REPORT_DIR/parsing_benchmark.log" ]; then
        echo '```' >> "$REPORT_FILE"
        cat "$REPORT_DIR/parsing_benchmark.log" >> "$REPORT_FILE"
        echo '```' >> "$REPORT_FILE"
    fi
    
    cat >> "$REPORT_FILE" << EOF

## Performance Metrics

### Key Performance Indicators
EOF
    
    # Extract key metrics from test logs
    if [ -f "$REPORT_DIR/large_scale_test.log" ]; then
        echo "#### Large Scale Parsing" >> "$REPORT_FILE"
        grep -E "(Files per second|ServiceSpecs per second|Parse duration)" "$REPORT_DIR/large_scale_test.log" | sed 's/^/- /' >> "$REPORT_FILE"
    fi
    
    if [ -f "$REPORT_DIR/memory_test.log" ]; then
        echo "#### Memory Usage" >> "$REPORT_FILE"
        grep -E "(Max memory used|Memory efficiency|Processing time)" "$REPORT_DIR/memory_test.log" | sed 's/^/- /' >> "$REPORT_FILE"
    fi
    
    cat >> "$REPORT_FILE" << EOF

## Recommendations

Based on the test results:

1. **Performance**: All tests should pass within specified time limits
2. **Memory**: Memory usage should stay within 500MB for 100MB files
3. **Concurrency**: All concurrent operations should complete successfully
4. **Regression**: Performance should not degrade compared to baselines

## Files Generated

- Performance log: \`$PERFORMANCE_LOG\`
- Test reports: \`$REPORT_DIR/\`
- Summary report: \`$REPORT_FILE\`

EOF
    
    echo -e "${GREEN}Performance report generated: $REPORT_FILE${NC}"
}

# Function to monitor system resources
monitor_resources() {
    echo -e "${YELLOW}Monitoring system resources...${NC}"
    
    # Get system info
    echo "System Information:" > "$REPORT_DIR/system_info.log"
    echo "==================" >> "$REPORT_DIR/system_info.log"
    echo "Date: $(date)" >> "$REPORT_DIR/system_info.log"
    echo "Hostname: $(hostname)" >> "$REPORT_DIR/system_info.log"
    echo "OS: $(uname -a)" >> "$REPORT_DIR/system_info.log"
    echo "Go Version: $(go version)" >> "$REPORT_DIR/system_info.log"
    echo "" >> "$REPORT_DIR/system_info.log"
    
    # Memory info
    if command -v free >/dev/null 2>&1; then
        echo "Memory Information:" >> "$REPORT_DIR/system_info.log"
        free -h >> "$REPORT_DIR/system_info.log"
        echo "" >> "$REPORT_DIR/system_info.log"
    fi
    
    # CPU info
    if [ -f /proc/cpuinfo ]; then
        echo "CPU Information:" >> "$REPORT_DIR/system_info.log"
        grep -E "(model name|cpu cores|processor)" /proc/cpuinfo | head -10 >> "$REPORT_DIR/system_info.log"
        echo "" >> "$REPORT_DIR/system_info.log"
    fi
    
    # Disk space
    echo "Disk Space:" >> "$REPORT_DIR/system_info.log"
    df -h . >> "$REPORT_DIR/system_info.log"
}

# Main execution
main() {
    # Clean up previous results
    rm -f "$PERFORMANCE_LOG" "$BENCHMARK_LOG" "$MEMORY_LOG"
    rm -rf "$REPORT_DIR"
    mkdir -p "$REPORT_DIR"
    
    log_with_timestamp "Starting FlowSpec CLI performance test suite"
    
    # Monitor system resources
    monitor_resources
    
    # Run performance tests
    run_performance_tests
    
    # Run benchmarks
    run_benchmarks
    
    # Generate report
    generate_report
    
    echo ""
    echo -e "${BLUE}Performance Testing Complete!${NC}"
    echo "==============================="
    echo "Results available in: $REPORT_DIR/"
    echo "Summary report: $REPORT_DIR/performance_summary.md"
    echo "Performance log: $PERFORMANCE_LOG"
    
    log_with_timestamp "Performance test suite completed"
}

# Handle script arguments
case "${1:-}" in
    "tests")
        run_performance_tests
        ;;
    "benchmarks")
        run_benchmarks
        ;;
    "report")
        generate_report
        ;;
    "monitor")
        monitor_resources
        ;;
    *)
        main
        ;;
esac