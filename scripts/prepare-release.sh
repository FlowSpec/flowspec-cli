#!/bin/bash

# FlowSpec CLI Release Preparation Script
# This script orchestrates the release process by calling targets defined in the Makefile.

set -e

# --- Color Definitions ---
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# --- Logging Functions ---
log_info() { echo -e "${BLUE}ℹ️  $1${NC}"; }
log_success() { echo -e "${GREEN}✅ $1${NC}"; }
log_error() { echo -e "${RED}❌ $1${NC}"; }
log_warning() { echo -e "${YELLOW}⚠️  $1${NC}"; }

# --- Help Display ---
show_help() {
    echo "FlowSpec CLI Release Preparation Script"
    echo ""
    echo "Usage: $0 [OPTIONS] VERSION"
    echo ""
    echo "This script prepares a new release by running checks, updating versions,"
    echo "building binaries, and creating release assets."
    echo ""
    echo "Options:"
    echo "  -h, --help     Display this help message"
    echo "  -d, --dry-run  Run in dry-run mode without executing commands"
    echo "  -f, --force    Force execution, skipping checks like git status"
    echo ""
    echo "Arguments:"
    echo "  VERSION        The release version (e.g., 1.0.0)"
    echo ""
    echo "Example:"
    echo "  $0 1.0.1"
}

# --- Argument Parsing ---
DRY_RUN=false
FORCE=false
VERSION=""

while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help) show_help; exit 0 ;;
        -d|--dry-run) DRY_RUN=true; shift ;;
        -f|--force) FORCE=true; shift ;;
        -*) log_error "Unknown option: $1"; show_help; exit 1 ;;
        *)
            if [ -z "$VERSION" ]; then
                VERSION="$1"
            else
                log_error "Redundant argument: $1"; show_help; exit 1
            fi
            shift
            ;; 
    esac
done

# --- Version Validation ---
if [ -z "$VERSION" ]; then
    log_error "Missing VERSION argument"; show_help; exit 1
fi
if ! [[ "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.-]+)?$ ]]; then
    log_error "Invalid version format: $VERSION"; exit 1
fi

# --- Script Execution ---
echo "🚀 FlowSpec CLI Release Preparation"
echo "=================================="
echo "Version: $VERSION"
echo "Dry Run: $DRY_RUN"
echo "Force Mode: $FORCE"
echo ""

# Command execution wrapper
execute() {
    local cmd="$1"
    log_info "Executing: $cmd"
    if [ "$DRY_RUN" = true ]; then
        echo "  [DRY RUN] $cmd"
    else
        if eval "$cmd"; then
            log_success "Command successful."
        else
            log_error "Command failed."
            exit 1
        fi
    fi
}

# --- Main Workflow Functions ---

check_prerequisites() {
    log_info "Checking prerequisites..."
    if [ ! -d ".git" ]; then log_error "Not a Git repository."; exit 1; fi

    if [ "$FORCE" != true ]; then
        if [ -n "$(git status --porcelain)" ]; then
            log_error "Git working directory is not clean. Commit changes or use --force."
            git status --short
            exit 1
        fi
        local current_branch=$(git branch --show-current)
        if [ "$current_branch" != "main" ] && [ "$current_branch" != "master" ]; then
            log_warning "Not on main/master branch (current: $current_branch)."
        fi
    fi

    if git tag -l | grep -q "^v$VERSION$"; then
        log_error "Tag v$VERSION already exists."; exit 1
    fi
    log_success "Prerequisites check passed."
}

update_version_files() {
    log_info "Updating version files..."
    if [ "$DRY_RUN" = true ]; then
        log_info "[DRY RUN] Skipping file modifications."
        return
    fi

    # Update version.go
    if [ -f "version.go" ]; then
        sed -i.bak "s/Version = ".*"/Version = \"$VERSION\"/" version.go
        rm -f version.go.bak
        log_success "Updated version.go to $VERSION"
    else
        log_warning "version.go not found, skipping."
    fi

    # Update CHANGELOG.md (simple placeholder update)
    if [ -f "CHANGELOG.md" ]; then
        # This is a placeholder; a more robust solution would use a changelog tool
        sed -i.bak "s/## \[Unreleased\]/## [Unreleased]\n\n## \[$VERSION\] - $(date +%Y-%m-%d)/" CHANGELOG.md
        rm -f CHANGELOG.md.bak
        log_success "Updated CHANGELOG.md for v$VERSION"
    else
        log_warning "CHANGELOG.md not found, skipping."
    fi
}

run_full_ci_checks() {
    log_info "Running full quality and test checks via Makefile..."
    execute "make ci"
}

build_and_package_release() {
    log_info "Building all release assets via Makefile..."
    execute "make clean"
    execute "make build-all VERSION=$VERSION"
    execute "make package VERSION=$VERSION"
}

create_git_commit_and_tag() {
    log_info "Creating Git commit and tag..."
    if [ "$DRY_RUN" = true ]; then
        log_info "[DRY RUN] Skipping git commit and tag."
        return
    fi
    git add version.go CHANGELOG.md
    git commit -m "chore: Prepare release v$VERSION"
    git tag -a "v$VERSION" -m "Release v$VERSION"
    log_success "Created commit and tag for v$VERSION."
}

show_next_steps() {
    echo ""
    log_success "Release preparation for v$VERSION complete!"
    echo "========================================="
    echo ""
    if [ "$DRY_RUN" = true ]; then
        echo "Dry run finished. To execute for real, run:"
        echo "  ./scripts/prepare-release.sh $VERSION"
    else
        echo "Next steps:"
        echo "1. Push the commit and tag to the remote repository:"
        echo "   git push origin main"
        echo "   git push origin v$VERSION"
        echo ""
        echo "2. Go to GitHub and create a new release from the v$VERSION tag."
        echo "3. Upload the assets from the 'build/packages/' directory."
    fi
    echo ""
}

# --- Main Execution ---
main() {
    check_prerequisites
    update_version_files
    run_full_ci_checks
    build_and_package_release
    create_git_commit_and_tag
    show_next_steps
}

main
