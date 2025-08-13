#!/bin/bash

# FlowSpec Nginx Exploration Example
# This script demonstrates the complete workflow of traffic exploration and contract verification

set -e

echo "🚀 FlowSpec Nginx Exploration Example"
echo "======================================"

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

# Step 1: Generate contract from access logs
echo -e "${YELLOW}📊 Step 1: Generating contract from Nginx access logs...${NC}"
echo "Command: flowspec-cli explore --traffic access.log --out generated-contract.yaml --service-name nginx-example"

if flowspec-cli explore --traffic access.log --out generated-contract.yaml --service-name nginx-example --service-version v1.0.0; then
    echo -e "${GREEN}✅ Contract generated successfully!${NC}"
    echo
    
    # Show generated contract summary
    echo -e "${BLUE}📄 Generated contract summary:${NC}"
    if command -v yq &> /dev/null; then
        echo "Service: $(yq '.metadata.name' generated-contract.yaml) v$(yq '.metadata.version' generated-contract.yaml)"
        echo "Endpoints: $(yq '.spec.endpoints | length' generated-contract.yaml)"
    else
        echo "Generated contract saved to: generated-contract.yaml"
        echo "First few lines:"
        head -10 generated-contract.yaml
    fi
    echo
else
    echo -e "${RED}❌ Failed to generate contract${NC}"
    exit 1
fi

# Step 2: Compare with expected contract (if available)
if [ -f "expected-contract.yaml" ]; then
    echo -e "${YELLOW}🔍 Step 2: Comparing with expected contract...${NC}"
    
    if command -v diff &> /dev/null; then
        if diff -u expected-contract.yaml generated-contract.yaml > contract-diff.txt; then
            echo -e "${GREEN}✅ Generated contract matches expected contract!${NC}"
            rm -f contract-diff.txt
        else
            echo -e "${YELLOW}⚠️  Generated contract differs from expected:${NC}"
            echo "Differences saved to: contract-diff.txt"
            echo "First few differences:"
            head -20 contract-diff.txt
        fi
    else
        echo -e "${BLUE}ℹ️  diff command not available, skipping comparison${NC}"
    fi
    echo
fi

# Step 3: Verify contract against trace data
echo -e "${YELLOW}✅ Step 3: Verifying contract against trace data...${NC}"
echo "Command: flowspec-cli verify --path generated-contract.yaml --trace test-trace.json --ci"

if flowspec-cli verify --path generated-contract.yaml --trace test-trace.json --ci; then
    echo -e "${GREEN}✅ Contract verification passed!${NC}"
else
    echo -e "${RED}❌ Contract verification failed${NC}"
    echo "This might be expected if the trace data doesn't match all generated endpoints"
fi
echo

# Step 4: Show detailed verification (non-CI mode)
echo -e "${YELLOW}📋 Step 4: Detailed verification report...${NC}"
echo "Command: flowspec-cli verify --path generated-contract.yaml --trace test-trace.json --output human"

flowspec-cli verify --path generated-contract.yaml --trace test-trace.json --output human || true
echo

# Step 5: Generate JSON report
echo -e "${YELLOW}📊 Step 5: Generating JSON report...${NC}"
echo "Command: flowspec-cli verify --path generated-contract.yaml --trace test-trace.json --output json"

if flowspec-cli verify --path generated-contract.yaml --trace test-trace.json --output json > verification-report.json 2>/dev/null; then
    echo -e "${GREEN}✅ JSON report generated: verification-report.json${NC}"
    
    if command -v jq &> /dev/null; then
        echo -e "${BLUE}📊 Report summary:${NC}"
        jq -r '.summary | "Total: \(.total), Success: \(.success), Failed: \(.failed), Skipped: \(.skipped)"' verification-report.json
    fi
else
    echo -e "${YELLOW}⚠️  JSON report generation completed with warnings${NC}"
fi
echo

# Step 6: Advanced exploration examples
echo -e "${YELLOW}🔧 Step 6: Advanced exploration examples...${NC}"

echo -e "${BLUE}Example 1: Custom thresholds${NC}"
echo "flowspec-cli explore --traffic access.log --out custom-contract.yaml \\"
echo "  --path-clustering-threshold 0.7 --required-threshold 0.9 --min-samples 2"
echo

echo -e "${BLUE}Example 2: Time filtering${NC}"
echo "flowspec-cli explore --traffic access.log --out filtered-contract.yaml \\"
echo "  --since \"2025-08-01T10:30:20Z\" --until \"2025-08-01T10:30:30Z\""
echo

echo -e "${BLUE}Example 3: Custom service metadata${NC}"
echo "flowspec-cli explore --traffic access.log --out api-contract.yaml \\"
echo "  --service-name \"user-api\" --service-version \"v2.1.0\""
echo

# Summary
echo -e "${GREEN}🎉 Example completed successfully!${NC}"
echo
echo -e "${BLUE}📁 Generated files:${NC}"
echo "  - generated-contract.yaml (main output)"
echo "  - verification-report.json (JSON report)"
if [ -f "contract-diff.txt" ]; then
    echo "  - contract-diff.txt (differences from expected)"
fi
echo
echo -e "${BLUE}💡 Next steps:${NC}"
echo "  1. Review the generated contract"
echo "  2. Refine thresholds if needed"
echo "  3. Integrate into your CI/CD pipeline"
echo "  4. Use the contract for ongoing validation"
echo
echo -e "${BLUE}🔗 Learn more:${NC}"
echo "  - FlowSpec Documentation: https://github.com/FlowSpec/flowspec-cli"
echo "  - GitHub Action: https://github.com/marketplace/actions/flowspec-verification"

# Cleanup option
echo
read -p "🗑️  Clean up generated files? (y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    rm -f generated-contract.yaml verification-report.json contract-diff.txt custom-contract.yaml filtered-contract.yaml api-contract.yaml
    echo -e "${GREEN}✅ Cleanup completed${NC}"
fi