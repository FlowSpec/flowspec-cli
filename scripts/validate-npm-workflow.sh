#!/bin/bash

# Validation script for NPM publishing workflow
# This script validates the workflow configuration and NPM package structure

set -e

echo "🔍 Validating NPM publishing workflow configuration..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
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
            echo -e "ℹ️  $message"
            ;;
    esac
}

# Validation functions
validate_workflow_file() {
    print_status "info" "Validating GitHub workflow file..."
    
    if [ ! -f ".github/workflows/release.yml" ]; then
        print_status "error" "Release workflow file not found"
        return 1
    fi
    
    # Check for required jobs
    if ! grep -q "publish-npm:" ".github/workflows/release.yml"; then
        print_status "error" "publish-npm job not found in workflow"
        return 1
    fi
    
    # Check for NPM_TOKEN usage
    if ! grep -q "NPM_TOKEN" ".github/workflows/release.yml"; then
        print_status "error" "NPM_TOKEN secret not referenced in workflow"
        return 1
    fi
    
    # Check for version synchronization
    if ! grep -q "npm version" ".github/workflows/release.yml"; then
        print_status "error" "NPM version synchronization not found"
        return 1
    fi
    
    # Check for error handling
    if ! grep -q "Rollback on failure" ".github/workflows/release.yml"; then
        print_status "error" "Rollback procedures not found"
        return 1
    fi
    
    print_status "success" "GitHub workflow file validation passed"
    return 0
}

validate_npm_package_structure() {
    print_status "info" "Validating NPM package structure..."
    
    # Check for npm directory
    if [ ! -d "npm" ]; then
        print_status "error" "npm directory not found"
        return 1
    fi
    
    # Check for package.json
    if [ ! -f "npm/package.json" ]; then
        print_status "error" "npm/package.json not found"
        return 1
    fi
    
    # Check for required scripts and files
    local required_files=(
        "npm/install.js"
        "npm/bin/flowspec-cli.js"
        "npm/lib/platform.js"
        "npm/lib/download.js"
        "npm/lib/binary.js"
    )
    
    for file in "${required_files[@]}"; do
        if [ ! -f "$file" ]; then
            print_status "warning" "$file not found (may not be implemented yet)"
        else
            print_status "success" "$file exists"
        fi
    done
    
    return 0
}

validate_package_json() {
    print_status "info" "Validating package.json configuration..."
    
    if [ ! -f "npm/package.json" ]; then
        print_status "warning" "package.json not found, skipping validation"
        return 0
    fi
    
    # Check for required fields using node if available
    if command -v node >/dev/null 2>&1; then
        cd npm
        
        # Check name field
        local name=$(node -p "require('./package.json').name" 2>/dev/null || echo "")
        if [ -z "$name" ]; then
            print_status "error" "package.json missing 'name' field"
            cd ..
            return 1
        fi
        print_status "success" "Package name: $name"
        
        # Check bin field
        local bin=$(node -p "JSON.stringify(require('./package.json').bin)" 2>/dev/null || echo "{}")
        if [ "$bin" = "{}" ]; then
            print_status "error" "package.json missing 'bin' field"
            cd ..
            return 1
        fi
        print_status "success" "Binary configuration found"
        
        # Check postinstall script
        local postinstall=$(node -p "require('./package.json').scripts?.postinstall" 2>/dev/null || echo "undefined")
        if [ "$postinstall" = "undefined" ]; then
            print_status "error" "package.json missing postinstall script"
            cd ..
            return 1
        fi
        print_status "success" "Postinstall script: $postinstall"
        
        cd ..
    else
        print_status "warning" "Node.js not available, skipping detailed package.json validation"
    fi
    
    return 0
}

validate_makefile_integration() {
    print_status "info" "Validating Makefile integration..."
    
    if [ ! -f "Makefile" ]; then
        print_status "warning" "Makefile not found"
        return 0
    fi
    
    # Check for package target
    if ! grep -q "package:" "Makefile"; then
        print_status "error" "package target not found in Makefile"
        return 1
    fi
    
    # Check for checksums generation
    if ! grep -q "sha256sum" "Makefile"; then
        print_status "warning" "SHA256 checksum generation not found in Makefile"
    else
        print_status "success" "Checksum generation found in Makefile"
    fi
    
    return 0
}

# Main validation
main() {
    echo "🚀 Starting NPM workflow validation..."
    echo ""
    
    local exit_code=0
    
    # Run all validations
    validate_workflow_file || exit_code=1
    echo ""
    
    validate_npm_package_structure || exit_code=1
    echo ""
    
    validate_package_json || exit_code=1
    echo ""
    
    validate_makefile_integration || exit_code=1
    echo ""
    
    # Summary
    if [ $exit_code -eq 0 ]; then
        print_status "success" "All validations passed!"
        echo ""
        echo "🎉 The NPM publishing workflow is properly configured."
        echo ""
        echo "📋 Next steps:"
        echo "  1. Ensure NPM_TOKEN secret is configured in GitHub repository settings"
        echo "  2. Complete implementation of NPM package files (if not done)"
        echo "  3. Test the workflow in a staging environment"
        echo "  4. Create a test release to validate the complete flow"
    else
        print_status "error" "Some validations failed!"
        echo ""
        echo "🔧 Please fix the issues above before proceeding."
        exit 1
    fi
}

# Run main function
main "$@"