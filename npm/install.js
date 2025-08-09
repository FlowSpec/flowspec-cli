#!/usr/bin/env node

/**
 * Main installation script for flowspec-cli NPM wrapper
 * Orchestrates the complete installation process including platform detection,
 * download, checksum verification, and binary setup
 */

const fs = require('fs');
const path = require('path');
const { PlatformDetector } = require('./lib/platform');
const { DownloadManager } = require('./lib/download');
const { BinaryManager } = require('./lib/binary');

class InstallationManager {
  constructor() {
    this.config = {
      binaryDir: path.join(__dirname, 'bin'),
      packageJsonPath: path.join(__dirname, 'package.json'),
      tempDir: path.join(__dirname, '.tmp'),
      maxRetries: 3,
      retryDelay: 1000
    };
    
    this.cleanupPaths = [];
  }

  /**
   * Main installation entry point
   * @returns {Promise<void>}
   */
  async install() {
    console.log('🚀 Starting flowspec-cli installation...\n');
    
    // Skip binary download in test environment
    if (process.env.FLOWSPEC_CLI_SKIP_DOWNLOAD === 'true') {
      console.log('⚠️  Skipping binary download (test environment)');
      console.log('✅ flowspec-cli installation completed successfully!');
      return;
    }
    
    try {
      // Step 1: Read version from package.json
      const version = this.readVersion();
      console.log(`📦 Installing flowspec-cli version ${version}`);
      
      // Step 2: Detect and validate platform
      const platform = this.detectPlatform();
      console.log(`🖥️  Detected platform: ${platform.os}-${platform.arch}`);
      
      // Step 3: Prepare installation directories
      this.prepareDirectories();
      
      // Step 4: Download and install binary
      await this.downloadAndInstallBinary(version, platform);
      
      // Step 5: Verify installation
      await this.verifyInstallation();
      
      // Step 6: Cleanup temporary files
      await this.cleanup();
      
      console.log('\n✅ flowspec-cli installation completed successfully!');
      console.log('   You can now use flowspec-cli with:');
      console.log('   • npx flowspec-cli [command]');
      console.log('   • Add to package.json scripts and use with npm run');
      
    } catch (error) {
      console.error('\n❌ Installation failed:', error.message);
      
      // Attempt cleanup on failure
      try {
        await this.cleanup();
      } catch (cleanupError) {
        console.warn('⚠️  Cleanup failed:', cleanupError.message);
      }
      
      this.printTroubleshootingGuidance(error);
      process.exit(1);
    }
  }

  /**
   * Reads version from package.json as single source of truth
   * @returns {string} Version string
   */
  readVersion() {
    try {
      console.log('📋 Reading version from package.json...');
      
      if (!fs.existsSync(this.config.packageJsonPath)) {
        throw new Error(`package.json not found at ${this.config.packageJsonPath}`);
      }
      
      const packageJson = JSON.parse(fs.readFileSync(this.config.packageJsonPath, 'utf8'));
      
      if (!packageJson.version) {
        throw new Error('Version not found in package.json');
      }
      
      return packageJson.version;
    } catch (error) {
      throw new Error(`Failed to read version: ${error.message}`);
    }
  }

  /**
   * Detects and validates the current platform
   * @returns {Object} Platform information
   */
  detectPlatform() {
    try {
      console.log('🔍 Detecting platform and architecture...');
      
      const platform = PlatformDetector.detectPlatform();
      
      if (!PlatformDetector.isSupported(platform)) {
        const supportedPlatforms = PlatformDetector.getSupportedPlatforms()
          .map(p => `${p.os}-${p.arch}`)
          .join(', ');
        
        throw new Error(
          `Unsupported platform: ${platform.os}-${platform.arch}. ` +
          `Supported platforms: ${supportedPlatforms}`
        );
      }
      
      return platform;
    } catch (error) {
      throw new Error(`Platform detection failed: ${error.message}`);
    }
  }

  /**
   * Prepares necessary directories for installation
   */
  prepareDirectories() {
    console.log('📁 Preparing installation directories...');
    
    try {
      // Create binary directory
      if (!fs.existsSync(this.config.binaryDir)) {
        fs.mkdirSync(this.config.binaryDir, { recursive: true });
        console.log(`   Created binary directory: ${this.config.binaryDir}`);
      }
      
      // Create temporary directory
      if (!fs.existsSync(this.config.tempDir)) {
        fs.mkdirSync(this.config.tempDir, { recursive: true });
        console.log(`   Created temporary directory: ${this.config.tempDir}`);
        this.cleanupPaths.push(this.config.tempDir);
      }
    } catch (error) {
      throw new Error(`Failed to prepare directories: ${error.message}`);
    }
  }

  /**
   * Downloads and installs the binary for the detected platform
   * @param {string} version - Version to install
   * @param {Object} platform - Platform information
   * @returns {Promise<void>}
   */
  async downloadAndInstallBinary(version, platform) {
    const versionTag = version.startsWith('v') ? version : `v${version}`;
    const binaryName = PlatformDetector.getBinaryName(platform);
    const downloadUrl = PlatformDetector.getDownloadUrl(versionTag, platform);
    const archivePath = path.join(this.config.tempDir, binaryName);
    
    console.log(`\n⬇️  Downloading binary from GitHub releases...`);
    console.log(`   URL: ${downloadUrl}`);
    console.log(`   File: ${binaryName}`);
    
    try {
      // Download the binary archive with progress indication
      await DownloadManager.downloadBinary(downloadUrl, archivePath, {
        maxRetries: this.config.maxRetries,
        retryDelay: this.config.retryDelay,
        onProgress: this.createProgressCallback()
      });
      
      this.cleanupPaths.push(archivePath);
      
      // Download and verify checksums
      console.log('\n🔐 Verifying download integrity...');
      const checksums = await DownloadManager.downloadChecksums(versionTag);
      const expectedChecksum = checksums.get(binaryName);
      
      if (!expectedChecksum) {
        throw new Error(`No checksum found for ${binaryName} in checksums.txt`);
      }
      
      console.log(`   Expected SHA256: ${expectedChecksum}`);
      
      // Verify checksum
      const isValidChecksum = await DownloadManager.verifyChecksum(archivePath, expectedChecksum);
      if (!isValidChecksum) {
        throw new Error('SHA256 checksum verification failed - download may be corrupted');
      }
      
      console.log('   ✅ Checksum verification passed');
      
      // Extract the archive
      console.log('\n📦 Extracting binary archive...');
      await DownloadManager.extractTarGz(archivePath, this.config.binaryDir);
      
      // Set executable permissions
      const binaryPath = BinaryManager.getBinaryPath({ binaryDir: this.config.binaryDir });
      await DownloadManager.setBinaryPermissions(binaryPath);
      
      console.log(`   ✅ Binary extracted to: ${binaryPath}`);
      
    } catch (error) {
      throw new Error(`Binary installation failed: ${error.message}`);
    }
  }

  /**
   * Verifies that the installation was successful
   * @returns {Promise<void>}
   */
  async verifyInstallation() {
    console.log('\n🔍 Verifying installation...');
    
    try {
      const binaryPath = BinaryManager.getBinaryPath({ binaryDir: this.config.binaryDir });
      
      // Check if binary exists
      if (!fs.existsSync(binaryPath)) {
        throw new Error(`Binary not found at expected location: ${binaryPath}`);
      }
      
      console.log(`   ✅ Binary exists at: ${binaryPath}`);
      
      // Verify binary can execute
      const isExecutable = await DownloadManager.verifyBinary(binaryPath, { timeout: 15000 });
      if (!isExecutable) {
        throw new Error('Binary exists but cannot be executed or --version command failed');
      }
      
      console.log('   ✅ Binary is executable and functional');
      
      // Get binary info for final confirmation
      const binaryInfo = BinaryManager.getBinaryInfo({ 
        binaryDir: this.config.binaryDir,
        packageJsonPath: this.config.packageJsonPath
      });
      
      console.log(`   ✅ Installation verified for version ${binaryInfo.version}`);
      
    } catch (error) {
      throw new Error(`Installation verification failed: ${error.message}`);
    }
  }

  /**
   * Cleans up temporary files and directories
   * @returns {Promise<void>}
   */
  async cleanup() {
    if (this.cleanupPaths.length > 0) {
      console.log('\n🧹 Cleaning up temporary files...');
      
      try {
        await DownloadManager.cleanup(this.cleanupPaths);
        console.log('   ✅ Cleanup completed');
      } catch (error) {
        console.warn(`   ⚠️  Some cleanup operations failed: ${error.message}`);
      }
    }
  }

  /**
   * Creates a progress callback for download operations
   * @returns {Function} Progress callback function
   */
  createProgressCallback() {
    let lastProgress = -1;
    
    return (progress, downloaded, total) => {
      const roundedProgress = Math.floor(progress);
      
      // Only update every 10% to avoid spam
      if (roundedProgress >= lastProgress + 10) {
        lastProgress = roundedProgress;
        
        if (total) {
          const downloadedMB = (downloaded / 1024 / 1024).toFixed(1);
          const totalMB = (total / 1024 / 1024).toFixed(1);
          console.log(`   Progress: ${roundedProgress}% (${downloadedMB}MB / ${totalMB}MB)`);
        } else {
          console.log(`   Progress: ${roundedProgress}%`);
        }
      }
    };
  }

  /**
   * Prints troubleshooting guidance based on the error type
   * @param {Error} error - The error that occurred
   */
  printTroubleshootingGuidance(error) {
    console.log('\n🔧 Troubleshooting Guide:');
    
    const message = error.message.toLowerCase();
    
    if (message.includes('unsupported platform')) {
      console.log('   • Your platform is not supported by flowspec-cli');
      console.log('   • Check the GitHub releases page for available binaries');
      console.log('   • Consider building from source if your platform is not supported');
      
    } else if (message.includes('network') || message.includes('download') || message.includes('timeout')) {
      console.log('   • Check your internet connection');
      console.log('   • Verify you can access github.com');
      console.log('   • Try running the installation again (network issues are often temporary)');
      console.log('   • If behind a corporate firewall, check proxy settings');
      
    } else if (message.includes('checksum') || message.includes('verification')) {
      console.log('   • The downloaded file may be corrupted');
      console.log('   • Try running the installation again');
      console.log('   • Check if you have sufficient disk space');
      console.log('   • Verify the GitHub release contains valid checksums');
      
    } else if (message.includes('permission')) {
      console.log('   • Check file system permissions in the installation directory');
      console.log('   • On Unix systems, ensure you can create executable files');
      console.log('   • Try running with elevated permissions if necessary');
      
    } else if (message.includes('version')) {
      console.log('   • Check that package.json contains a valid version field');
      console.log('   • Verify the version exists in GitHub releases');
      console.log('   • Try updating to the latest version of this package');
      
    } else {
      console.log('   • Try running the installation again');
      console.log('   • Check the GitHub issues page for similar problems');
      console.log('   • Ensure you have sufficient disk space and permissions');
      console.log('   • Verify your Node.js version is supported (>=14.0.0)');
    }
    
    console.log('\n📚 Additional Resources:');
    console.log('   • GitHub Repository: https://github.com/flowspec/flowspec-cli');
    console.log('   • Issues: https://github.com/flowspec/flowspec-cli/issues');
    console.log('   • Manual Installation: https://github.com/flowspec/flowspec-cli#installation');
  }
}

// Main execution
async function main() {
  // Handle process termination gracefully
  const installer = new InstallationManager();
  
  const handleExit = async (signal) => {
    console.log(`\n⚠️  Installation interrupted by ${signal}`);
    try {
      await installer.cleanup();
    } catch (error) {
      console.warn('Cleanup failed:', error.message);
    }
    process.exit(1);
  };
  
  process.on('SIGINT', () => handleExit('SIGINT'));
  process.on('SIGTERM', () => handleExit('SIGTERM'));
  
  // Run installation
  await installer.install();
}

// Only run if this script is executed directly (not required as module)
if (require.main === module) {
  main().catch((error) => {
    console.error('Unexpected error:', error);
    process.exit(1);
  });
}

module.exports = { InstallationManager };