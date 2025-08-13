#!/bin/bash

# FlowSpec YAML Contract Verification Example
# This script demonstrates how to verify multiple YAML contracts against trace data

set -e

echo "🔍 FlowSpec YAML Contract Verification"
echo "====================================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Check if flowspec-cli is available
if ! command -v flowspec-cli &> /dev/null; then
    echo -e "${RED}❌ flowspec-cli not found. Please install it first.${NC}"
    echo "Installation options:"
    echo "  npm install -g @flowspec/cli"
    echo "  go install github.com/FlowSpec/flowspec-cli/cmd/flowspec-cli@latest"
    exit 1
fi

echo -e "${BLUE}📋 FlowSpec CLI version:${NC}"
flowspec-cli --version
echo

# Contract and trace pairs
declare -A contracts=(
    ["user-service.yaml"]="test-traces/user-service-trace.json"
    ["order-service.yaml"]="test-traces/order-service-trace.json"
    ["minimal-service.yaml"]="test-traces/user-service-trace.json"  # Use user trace for minimal example
)

# Counters
total_contracts=0
passed_contracts=0
failed_contracts=0

echo -e "${YELLOW}🚀 Starting contract verification...${NC}"
echo

# Verify each contract
for contract in "${!contracts[@]}"; do
    trace="${contracts[$contract]}"
    total_contracts=$((total_contracts + 1))
    
    echo -e "${BLUE}📄 Verifying: $contract${NC}"
    echo "   Trace: $trace"
    
    if [ ! -f "$contract" ]; then
        echo -e "${RED}   ❌ Contract file not found: $contract${NC}"
        failed_contracts=$((failed_contracts + 1))
        echo
        continue
    fi
    
    if [ ! -f "$trace" ]; then
        echo -e "${RED}   ❌ Trace file not found: $trace${NC}"
        failed_contracts=$((failed_contracts + 1))
        echo
        continue
    fi
    
    # Run verification in CI mode for concise output
    if flowspec-cli verify --path "$contract" --trace "$trace" --ci 2>/dev/null; then
        echo -e "${GREEN}   ✅ Verification passed${NC}"
        passed_contracts=$((passed_contracts + 1))
    else
        echo -e "${RED}   ❌ Verification failed${NC}"
        failed_contracts=$((failed_contracts + 1))
        
        # Show detailed output for failed contracts
        echo -e "${YELLOW}   📋 Detailed failure report:${NC}"
        flowspec-cli verify --path "$contract" --trace "$trace" --output human 2>/dev/null || true
    fi
    echo
done

# Verify legacy format (should still work)
echo -e "${BLUE}📄 Verifying legacy format: legacy-format.yaml${NC}"
echo "   Note: This uses the deprecated format but should still work"

if flowspec-cli verify --path legacy-format.yaml --trace test-traces/user-service-trace.json --ci 2>/dev/null; then
    echo -e "${GREEN}   ✅ Legacy format verification passed${NC}"
    passed_contracts=$((passed_contracts + 1))
else
    echo -e "${YELLOW}   ⚠️  Legacy format verification failed (expected)${NC}"
    failed_contracts=$((failed_contracts + 1))
fi
total_contracts=$((total_contracts + 1))
echo

# Summary
echo -e "${YELLOW}📊 Verification Summary${NC}"
echo "======================"
echo "Total contracts: $total_contracts"
echo -e "Passed: ${GREEN}$passed_contracts${NC}"
echo -e "Failed: ${RED}$failed_contracts${NC}"
echo

if [ $failed_contracts -eq 0 ]; then
    echo -e "${GREEN}🎉 All contracts verified successfully!${NC}"
    exit_code=0
else
    echo -e "${YELLOW}⚠️  Some contracts failed verification${NC}"
    echo "This might be expected if trace data doesn't match all contract specifications"
    exit_code=1
fi

# Advanced examples
echo -e "${YELLOW}🔧 Advanced Usage Examples${NC}"
echo "=========================="
echo

echo -e "${BLUE}Example 1: JSON output${NC}"
echo "flowspec-cli verify --path user-service.yaml --trace test-traces/user-service-trace.json --output json"
echo

echo -e "${BLUE}Example 2: Debug mode${NC}"
echo "flowspec-cli verify --path user-service.yaml --trace test-traces/user-service-trace.json --debug"
echo

echo -e "${BLUE}Example 3: Strict mode${NC}"
echo "flowspec-cli verify --path user-service.yaml --trace test-traces/user-service-trace.json --strict"
echo

echo -e "${BLUE}Example 4: Custom timeout${NC}"
echo "flowspec-cli verify --path user-service.yaml --trace test-traces/user-service-trace.json --timeout 60s"
echo

# Generate sample JSON report
echo -e "${YELLOW}📊 Generating sample JSON report...${NC}"
if flowspec-cli verify --path user-service.yaml --trace test-traces/user-service-trace.json --output json > sample-report.json 2>/dev/null; then
    echo -e "${GREEN}✅ JSON report generated: sample-report.json${NC}"
    
    if command -v jq &> /dev/null; then
        echo -e "${BLUE}📋 Report summary:${NC}"
        jq -r '.summary | "Total: \(.total), Success: \(.success), Failed: \(.failed), Skipped: \(.skipped)"' sample-report.json
    fi
else
    echo -e "${YELLOW}⚠️  JSON report generation completed with warnings${NC}"
fi
echo

# CI/CD Integration examples
echo -e "${YELLOW}🚀 CI/CD Integration Examples${NC}"
echo "============================="
echo

echo -e "${BLUE}GitHub Actions:${NC}"
cat << 'EOF'
name: Contract Validation
on: [push, pull_request]
jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Validate Contracts
        run: |
          flowspec-cli verify --path contracts/user-service.yaml --trace traces/user-service.json --ci
          flowspec-cli verify --path contracts/order-service.yaml --trace traces/order-service.json --ci
EOF
echo

echo -e "${BLUE}NPM Scripts:${NC}"
cat << 'EOF'
{
  "scripts": {
    "validate:contracts": "./verify-contracts.sh",
    "validate:user": "flowspec-cli verify --path user-service.yaml --trace test-traces/user-service-trace.json",
    "validate:order": "flowspec-cli verify --path order-service.yaml --trace test-traces/order-service-trace.json"
  }
}
EOF
echo

echo -e "${BLUE}Makefile:${NC}"
cat << 'EOF'
.PHONY: validate-contracts
validate-contracts:
	@./verify-contracts.sh

.PHONY: validate-user-service
validate-user-service:
	flowspec-cli verify --path user-service.yaml --trace test-traces/user-service-trace.json --output human
EOF
echo

# Cleanup option
echo
read -p "🗑️  Clean up generated files? (y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    rm -f sample-report.json
    echo -e "${GREEN}✅ Cleanup completed${NC}"
fi

echo -e "${BLUE}🔗 Learn more:${NC}"
echo "  - FlowSpec Documentation: https://github.com/FlowSpec/flowspec-cli"
echo "  - YAML Contract Guide: https://github.com/FlowSpec/flowspec-cli/blob/main/examples/yaml-contracts/README.md"

exit $exit_code