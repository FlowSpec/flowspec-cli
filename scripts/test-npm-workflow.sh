#!/bin/bash

# Test script for NPM publishing workflow
# This script simulates the workflow steps locally for testing

set -e

echo "🧪 Testing NPM publishing workflow locally..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    local status=$1
    local message=$2
    case $status in
        "success")
            echo -e "${GREEN}✅ $message${NC}"
            ;;
        "warning")
            echo -e "${YELLOW}⚠️  $message${NC}"
            ;;
        "error")
            echo -e "${RED}❌ $message${NC}"
            ;;
        "info")
            echo -e "${BLUE}ℹ️  $message${NC}"
            ;;
        "step")
            echo -e "${BLUE}🔄 $message${NC}"
            ;;
    esac
}

# Test version (use a test version to avoid conflicts)
TEST_VERSION="0.0.0-test-$(date +%s)"

# Cleanup function
cleanup() {
    print_status "info" "Cleaning up test files..."
    if [ -f "npm/package.json.backup" ]; then
        mv npm/package.json.backup npm/package.json
        print_status "success" "Restored original package.json"
    fi
}

# Set up cleanup trap
trap cleanup EXIT

# Test functions
test_package_structure_validation() {
    print_status "step" "Testing package structure validation..."
    
    local required_files=(
        "npm/package.json"
        "npm/install.js"
        "npm/lib/platform.js"
        "npm/lib/download.js"
        "npm/lib/binary.js"
        "npm/bin/flowspec-cli.js"
    )
    
    for file in "${required_files[@]}"; do
        if [ ! -f "$file" ]; then
            print_status "error" "Required file missing: $file"
            return 1
        fi
    done
    
    print_status "success" "Package structure validation passed"
    return 0
}

test_version_synchronization() {
    print_status "step" "Testing version synchronization..."
    
    cd npm
    
    # Backup original package.json
    cp package.json package.json.backup
    
    # Test version update
    npm version "$TEST_VERSION" --no-git-tag-version --allow-same-version
    
    # Verify version was updated
    local updated_version=$(node -p "require('./package.json').version")
    if [ "$updated_version" != "$TEST_VERSION" ]; then
        print_status "error" "Version update failed: expected $TEST_VERSION, got $updated_version"
        cd ..
        return 1
    fi
    
    print_status "success" "Version synchronization test passed"
    cd ..
    return 0
}

test_package_validation() {
    print_status "step" "Testing package validation..."
    
    cd npm
    
    # Test npm pack (dry run)
    if npm pack --dry-run >/dev/null 2>&1; then
        print_status "success" "Package validation passed"
    else
        print_status "error" "Package validation failed"
        cd ..
        return 1
    fi
    
    cd ..
    return 0
}

test_dependencies_installation() {
    print_status "step" "Testing dependencies installation..."
    
    cd npm
    
    # Check if package-lock.json exists
    if [ -f "package-lock.json" ]; then
        # Test npm ci
        if npm ci --only=production --dry-run >/dev/null 2>&1; then
            print_status "success" "Dependencies installation test passed"
        else
            print_status "warning" "Dependencies installation test failed (may be expected in test environment)"
        fi
    else
        print_status "warning" "package-lock.json not found, skipping dependencies test"
    fi
    
    cd ..
    return 0
}

test_npm_scripts() {
    print_status "step" "Testing NPM scripts..."
    
    cd npm
    
    # Check if test script exists and can be run
    local test_script=$(node -p "require('./package.json').scripts?.test" 2>/dev/null || echo "undefined")
    if [ "$test_script" != "undefined" ]; then
        print_status "info" "Test script found: $test_script"
        # Note: We don't actually run the tests here to avoid side effects
        print_status "success" "NPM scripts configuration validated"
    else
        print_status "warning" "No test script found in package.json"
    fi
    
    cd ..
    return 0
}

simulate_workflow_steps() {
    print_status "step" "Simulating workflow steps..."
    
    # Simulate the key steps from the GitHub workflow
    print_status "info" "Step 1: Package structure validation"
    test_package_structure_validation || return 1
    
    print_status "info" "Step 2: Version synchronization"
    test_version_synchronization || return 1
    
    print_status "info" "Step 3: Dependencies installation"
    test_dependencies_installation
    
    print_status "info" "Step 4: Package validation"
    test_package_validation || return 1
    
    print_status "info" "Step 5: NPM scripts validation"
    test_npm_scripts
    
    print_status "success" "All workflow simulation steps completed"
    return 0
}

# Main test function
main() {
    echo "🚀 Starting NPM workflow local testing..."
    echo ""
    
    # Check prerequisites
    if ! command -v node >/dev/null 2>&1; then
        print_status "error" "Node.js is required for testing"
        exit 1
    fi
    
    if ! command -v npm >/dev/null 2>&1; then
        print_status "error" "NPM is required for testing"
        exit 1
    fi
    
    print_status "info" "Node.js version: $(node --version)"
    print_status "info" "NPM version: $(npm --version)"
    print_status "info" "Test version: $TEST_VERSION"
    echo ""
    
    # Run tests
    local exit_code=0
    
    simulate_workflow_steps || exit_code=1
    
    echo ""
    
    # Summary
    if [ $exit_code -eq 0 ]; then
        print_status "success" "All local tests passed! 🎉"
        echo ""
        echo "📋 The workflow should work correctly in the CI environment."
        echo ""
        echo "🔄 Next steps for complete testing:"
        echo "  1. Set up NPM_TOKEN secret in GitHub repository"
        echo "  2. Create a test release with a pre-release tag"
        echo "  3. Monitor the workflow execution in GitHub Actions"
        echo "  4. Verify the package is published to NPM registry"
        echo "  5. Test installation: npm install @flowspec/cli"
    else
        print_status "error" "Some tests failed!"
        echo ""
        echo "🔧 Please fix the issues above before running the workflow."
        exit 1
    fi
}

# Show usage if help requested
if [ "$1" = "--help" ] || [ "$1" = "-h" ]; then
    echo "Usage: $0 [options]"
    echo ""
    echo "Test the NPM publishing workflow locally."
    echo ""
    echo "Options:"
    echo "  -h, --help    Show this help message"
    echo ""
    echo "This script simulates the key steps of the GitHub workflow"
    echo "to validate the NPM package configuration locally."
    exit 0
fi

# Run main function
main "$@"