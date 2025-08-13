#!/bin/bash

# FlowSpec CLI Installation Script
# This script downloads and installs FlowSpec CLI for GitHub Actions
# Supports Linux, macOS, and Windows (via Git Bash/WSL)

set -euo pipefail

# Configuration
REPO_OWNER="FlowSpec"
REPO_NAME="flowspec-cli"
GITHUB_API_BASE="https://api.github.com/repos/${REPO_OWNER}/${REPO_NAME}"
GITHUB_RELEASES_BASE="https://github.com/${REPO_OWNER}/${REPO_NAME}/releases/download"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
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

# Platform detection
detect_platform() {
    local os arch
    
    # Detect OS
    case "$(uname -s)" in
        Linux*)
            os="linux"
            ;;
        Darwin*)
            os="darwin"
            ;;
        CYGWIN*|MINGW*|MSYS*)
            os="windows"
            ;;
        *)
            log_error "Unsupported operating system: $(uname -s)"
            exit 1
            ;;
    esac
    
    # Detect architecture
    case "$(uname -m)" in
        x86_64|amd64)
            arch="amd64"
            ;;
        arm64|aarch64)
            arch="arm64"
            ;;
        i386|i686)
            arch="386"
            ;;
        *)
            log_error "Unsupported architecture: $(uname -m)"
            exit 1
            ;;
    esac
    
    echo "${os}-${arch}"
}

# Get file extension based on OS
get_file_extension() {
    local platform="$1"
    
    if [[ "$platform" == *"windows"* ]]; then
        echo "zip"
    else
        echo "tar.gz"
    fi
}

# Resolve version (latest or specific)
resolve_version() {
    local version="$1"
    
    if [ "$version" = "latest" ]; then
        log_info "Resolving latest version..."
        local latest_version
        latest_version=$(curl -fsSL "${GITHUB_API_BASE}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
        
        if [ -z "$latest_version" ]; then
            log_error "Failed to resolve latest version"
            exit 1
        fi
        
        log_info "Latest version: $latest_version"
        echo "$latest_version"
    else
        # Validate version format (should start with 'v')
        if [[ ! "$version" =~ ^v[0-9]+\.[0-9]+\.[0-9]+ ]]; then
            log_warning "Version should start with 'v' (e.g., v1.0.0). Adding 'v' prefix."
            version="v${version}"
        fi
        echo "$version"
    fi
}

# Download file with retry logic
download_with_retry() {
    local url="$1"
    local output="$2"
    local max_retries=3
    local retry_count=0
    
    while [ $retry_count -lt $max_retries ]; do
        if curl -fsSL --connect-timeout 10 --max-time 300 -o "$output" "$url"; then
            return 0
        fi
        
        retry_count=$((retry_count + 1))
        if [ $retry_count -lt $max_retries ]; then
            log_warning "Download failed, retrying in 5 seconds... (attempt $retry_count/$max_retries)"
            sleep 5
        fi
    done
    
    log_error "Failed to download after $max_retries attempts: $url"
    return 1
}

# Verify checksum
verify_checksum() {
    local filename="$1"
    local checksums_file="$2"
    
    log_info "Verifying checksum..."
    
    # Check if checksum tools are available
    if command -v sha256sum >/dev/null 2>&1; then
        if grep "$filename" "$checksums_file" | sha256sum -c - >/dev/null 2>&1; then
            log_success "Checksum verification passed"
            return 0
        fi
    elif command -v shasum >/dev/null 2>&1; then
        if grep "$filename" "$checksums_file" | shasum -a 256 -c - >/dev/null 2>&1; then
            log_success "Checksum verification passed"
            return 0
        fi
    else
        log_warning "No checksum verification tool available (sha256sum or shasum)"
        log_warning "Skipping checksum verification"
        return 0
    fi
    
    log_error "Checksum verification failed"
    return 1
}

# Extract archive
extract_archive() {
    local filename="$1"
    local extension="$2"
    
    log_info "Extracting archive..."
    
    case "$extension" in
        "zip")
            if command -v unzip >/dev/null 2>&1; then
                unzip -q "$filename"
            else
                log_error "unzip command not found"
                return 1
            fi
            ;;
        "tar.gz")
            if command -v tar >/dev/null 2>&1; then
                tar -xzf "$filename"
            else
                log_error "tar command not found"
                return 1
            fi
            ;;
        *)
            log_error "Unsupported archive format: $extension"
            return 1
            ;;
    esac
    
    log_success "Archive extracted successfully"
}

# Find binary after extraction
find_binary() {
    local binary_name="flowspec-cli"
    
    # Look for binary in current directory first
    if [ -f "$binary_name" ]; then
        echo "$binary_name"
        return 0
    fi
    
    # Look for binary with .exe extension (Windows)
    if [ -f "${binary_name}.exe" ]; then
        echo "${binary_name}.exe"
        return 0
    fi
    
    # Look for binary in subdirectories
    local found_binary
    found_binary=$(find . -name "${binary_name}*" -type f -executable 2>/dev/null | head -1)
    
    if [ -n "$found_binary" ]; then
        echo "$found_binary"
        return 0
    fi
    
    log_error "Could not find $binary_name binary after extraction"
    return 1
}

# Install binary to system PATH
install_binary() {
    local binary_path="$1"
    local platform="$2"
    
    log_info "Installing binary to system PATH..."
    
    # Make binary executable
    chmod +x "$binary_path"
    
    # Determine installation directory
    local install_dir
    if [[ "$platform" == *"windows"* ]]; then
        # For Windows (Git Bash/WSL), use a directory that will be added to PATH
        install_dir="/c/flowspec-cli"
        mkdir -p "$install_dir"
        cp "$binary_path" "$install_dir/"
        
        # Add to PATH for current session
        export PATH="$install_dir:$PATH"
        
        # Output for GitHub Actions to add to PATH
        if [ -n "${GITHUB_PATH:-}" ]; then
            echo "$install_dir" >> "$GITHUB_PATH"
        fi
    else
        # For Linux/macOS, install to /usr/local/bin
        if [ -w "/usr/local/bin" ]; then
            cp "$binary_path" "/usr/local/bin/"
        elif command -v sudo >/dev/null 2>&1; then
            sudo cp "$binary_path" "/usr/local/bin/"
        else
            log_error "Cannot install to /usr/local/bin (no write permission and sudo not available)"
            log_info "Please manually copy the binary to a directory in your PATH:"
            log_info "  cp $binary_path /usr/local/bin/"
            return 1
        fi
    fi
    
    log_success "Binary installed successfully"
}

# Verify installation
verify_installation() {
    log_info "Verifying installation..."
    
    if command -v flowspec-cli >/dev/null 2>&1; then
        local version
        version=$(flowspec-cli --version 2>/dev/null || echo "unknown")
        log_success "FlowSpec CLI installed successfully: $version"
        return 0
    else
        log_error "FlowSpec CLI not found in PATH after installation"
        return 1
    fi
}

# Main installation function
install_flowspec_cli() {
    local version="${1:-latest}"
    local temp_dir
    
    log_info "Starting FlowSpec CLI installation..."
    log_info "Requested version: $version"
    
    # Detect platform
    local platform
    platform=$(detect_platform)
    log_info "Detected platform: $platform"
    
    # Get file extension
    local extension
    extension=$(get_file_extension "$platform")
    
    # Resolve version
    version=$(resolve_version "$version")
    
    # Create temporary directory
    temp_dir=$(mktemp -d)
    cd "$temp_dir"
    
    # Construct URLs and filenames
    local filename="flowspec-cli-${platform}.${extension}"
    local download_url="${GITHUB_RELEASES_BASE}/${version}/${filename}"
    local checksums_url="${GITHUB_RELEASES_BASE}/${version}/checksums.txt"
    
    log_info "Download URL: $download_url"
    
    # Download binary and checksums
    log_info "Downloading FlowSpec CLI binary..."
    if ! download_with_retry "$download_url" "$filename"; then
        log_error "Failed to download FlowSpec CLI binary"
        cd / && rm -rf "$temp_dir"
        exit 1
    fi
    
    log_info "Downloading checksums..."
    if ! download_with_retry "$checksums_url" "checksums.txt"; then
        log_warning "Failed to download checksums, skipping verification"
    else
        # Verify checksum
        if ! verify_checksum "$filename" "checksums.txt"; then
            log_error "Checksum verification failed"
            cd / && rm -rf "$temp_dir"
            exit 1
        fi
    fi
    
    # Extract archive
    if ! extract_archive "$filename" "$extension"; then
        log_error "Failed to extract archive"
        cd / && rm -rf "$temp_dir"
        exit 1
    fi
    
    # Find binary
    local binary_path
    if ! binary_path=$(find_binary); then
        cd / && rm -rf "$temp_dir"
        exit 1
    fi
    
    log_info "Found binary: $binary_path"
    
    # Install binary
    if ! install_binary "$binary_path" "$platform"; then
        cd / && rm -rf "$temp_dir"
        exit 1
    fi
    
    # Cleanup
    cd /
    rm -rf "$temp_dir"
    
    # Verify installation
    if ! verify_installation; then
        exit 1
    fi
    
    log_success "FlowSpec CLI installation completed successfully!"
}

# Script entry point
main() {
    local version="${1:-latest}"
    
    # Check if running in GitHub Actions
    if [ -n "${GITHUB_ACTIONS:-}" ]; then
        log_info "Running in GitHub Actions environment"
    fi
    
    # Install FlowSpec CLI
    install_flowspec_cli "$version"
}

# Run main function if script is executed directly
if [ "${BASH_SOURCE[0]}" = "${0}" ]; then
    main "$@"
fi