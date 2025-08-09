/**
 * Binary management utilities for flowspec-cli NPM wrapper
 * Manages the lifecycle of the downloaded binary and provides execution utilities
 */

const fs = require('fs');
const path = require('path');
const { spawn } = require('child_process');
const { PlatformDetector } = require('./platform');
const { DownloadManager } = require('./download');

class BinaryManager {
  /**
   * Default configuration for binary management
   */
  static DEFAULT_CONFIG = {
    binaryDir: path.join(__dirname, '..', 'bin'),
    packageJsonPath: path.join(__dirname, '..', 'package.json'),
    executionTimeout: 0, // No timeout by default for CLI commands
    reinstallOnFailure: true
  };

  /**
   * Gets the path to the installed binary for the current platform
   * @param {Object} options - Optional configuration
   * @param {string} options.binaryDir - Directory where binaries are stored
   * @returns {string} Path to the binary executable
   */
  static getBinaryPath(options = {}) {
    const config = { ...this.DEFAULT_CONFIG, ...options };
    
    try {
      const platform = PlatformDetector.detectPlatform();
      const binaryName = this._getBinaryExecutableName(platform);
      const binaryPath = path.join(config.binaryDir, binaryName);
      
      return binaryPath;
    } catch (error) {
      throw new Error(`Failed to determine binary path: ${error.message}`);
    }
  }

  /**
   * Ensures the binary exists and is functional, with automatic re-installation capability
   * @param {Object} options - Optional configuration
   * @param {boolean} options.reinstallOnFailure - Whether to reinstall if binary is missing or broken
   * @param {Function} options.onProgress - Progress callback for download operations
   * @returns {Promise<void>}
   */
  static async ensureBinaryExists(options = {}) {
    const config = { ...this.DEFAULT_CONFIG, ...options };
    
    try {
      const binaryPath = this.getBinaryPath(config);
      
      // Check if binary exists and is functional
      if (await this._isBinaryFunctional(binaryPath)) {
        console.log(`Binary is already installed and functional at ${binaryPath}`);
        return;
      }
      
      if (!config.reinstallOnFailure) {
        throw new Error(`Binary not found or not functional at ${binaryPath} and reinstallOnFailure is disabled`);
      }
      
      console.log('Binary not found or not functional, attempting installation...');
      await this._installBinary(config);
      
      // Verify installation was successful
      if (!(await this._isBinaryFunctional(this.getBinaryPath(config)))) {
        throw new Error('Binary installation completed but binary is still not functional');
      }
      
      console.log('Binary installation completed successfully');
    } catch (error) {
      throw new Error(`Failed to ensure binary exists: ${error.message}`);
    }
  }

  /**
   * Executes the binary with transparent command forwarding
   * @param {string[]} args - Command line arguments to pass to the binary
   * @param {Object} options - Optional configuration
   * @param {number} options.timeout - Execution timeout in milliseconds (0 = no timeout)
   * @param {Object} options.env - Environment variables to pass to the binary
   * @param {string} options.cwd - Working directory for the binary execution
   * @param {boolean} options.stdio - Whether to inherit stdio from parent process
   * @returns {Promise<number>} Exit code from the binary execution
   */
  static async executeBinary(args = [], options = {}) {
    const config = { 
      timeout: this.DEFAULT_CONFIG.executionTimeout,
      env: process.env,
      cwd: process.cwd(),
      stdio: true,
      ...options 
    };
    
    try {
      // Ensure binary exists before attempting execution
      await this.ensureBinaryExists();
      
      const binaryPath = this.getBinaryPath();
      
      console.log(`Executing binary: ${binaryPath} ${args.join(' ')}`);
      
      return await this._executeBinaryProcess(binaryPath, args, config);
    } catch (error) {
      throw new Error(`Failed to execute binary: ${error.message}`);
    }
  }

  /**
   * Gets the version from package.json as the single source of truth
   * @param {Object} options - Optional configuration
   * @param {string} options.packageJsonPath - Path to package.json file
   * @returns {string} Version string from package.json
   */
  static getVersion(options = {}) {
    const config = { ...this.DEFAULT_CONFIG, ...options };
    
    try {
      if (!fs.existsSync(config.packageJsonPath)) {
        throw new Error(`package.json not found at ${config.packageJsonPath}`);
      }
      
      const packageJson = JSON.parse(fs.readFileSync(config.packageJsonPath, 'utf8'));
      
      if (!packageJson.version) {
        throw new Error('Version not found in package.json');
      }
      
      return packageJson.version;
    } catch (error) {
      throw new Error(`Failed to read version from package.json: ${error.message}`);
    }
  }

  /**
   * Gets information about the current binary installation
   * @param {Object} options - Optional configuration
   * @returns {Object} Binary installation information
   */
  static getBinaryInfo(options = {}) {
    try {
      const binaryPath = this.getBinaryPath(options);
      const version = this.getVersion(options);
      const platform = PlatformDetector.detectPlatform();
      
      return {
        version,
        binaryPath,
        platform,
        exists: fs.existsSync(binaryPath),
        isExecutable: this._isExecutable(binaryPath)
      };
    } catch (error) {
      throw new Error(`Failed to get binary info: ${error.message}`);
    }
  }

  /**
   * Removes the installed binary (useful for cleanup or forced reinstallation)
   * @param {Object} options - Optional configuration
   * @returns {Promise<void>}
   */
  static async removeBinary(options = {}) {
    try {
      const binaryPath = this.getBinaryPath(options);
      
      if (fs.existsSync(binaryPath)) {
        fs.unlinkSync(binaryPath);
        console.log(`Removed binary at ${binaryPath}`);
      } else {
        console.log(`Binary not found at ${binaryPath}, nothing to remove`);
      }
    } catch (error) {
      throw new Error(`Failed to remove binary: ${error.message}`);
    }
  }

  /**
   * Gets the executable name for the current platform (without path)
   * @private
   * @param {Object} platform - Platform information from PlatformDetector
   * @returns {string} Executable filename
   */
  static _getBinaryExecutableName(platform) {
    const baseName = 'flowspec-cli';
    
    // On Windows, add .exe extension
    if (platform.os === 'windows') {
      return `${baseName}.exe`;
    }
    
    return baseName;
  }

  /**
   * Checks if a binary exists and is functional
   * @private
   * @param {string} binaryPath - Path to the binary
   * @returns {Promise<boolean>} True if binary is functional
   */
  static async _isBinaryFunctional(binaryPath) {
    try {
      // Check if file exists
      if (!fs.existsSync(binaryPath)) {
        return false;
      }
      
      // Check if file is executable
      if (!this._isExecutable(binaryPath)) {
        return false;
      }
      
      // Test binary execution with --version flag
      return await DownloadManager.verifyBinary(binaryPath, { timeout: 10000 });
    } catch (error) {
      console.warn(`Binary functionality check failed: ${error.message}`);
      return false;
    }
  }

  /**
   * Checks if a file has executable permissions
   * @private
   * @param {string} filePath - Path to the file
   * @returns {boolean} True if file is executable
   */
  static _isExecutable(filePath) {
    try {
      if (!fs.existsSync(filePath)) {
        return false;
      }
      
      // On Windows, assume .exe files are executable
      if (process.platform === 'win32') {
        return path.extname(filePath).toLowerCase() === '.exe' || 
               path.extname(filePath) === '';
      }
      
      // On Unix systems, check file permissions
      const stats = fs.statSync(filePath);
      const mode = stats.mode;
      
      // Check if owner has execute permission (0o100)
      return (mode & 0o100) !== 0;
    } catch (error) {
      return false;
    }
  }

  /**
   * Installs the binary by downloading and extracting it
   * @private
   * @param {Object} config - Configuration options
   * @returns {Promise<void>}
   */
  static async _installBinary(config) {
    const platform = PlatformDetector.detectPlatform();
    const version = this.getVersion(config);
    const versionTag = version.startsWith('v') ? version : `v${version}`;
    
    // Create binary directory if it doesn't exist
    if (!fs.existsSync(config.binaryDir)) {
      fs.mkdirSync(config.binaryDir, { recursive: true });
    }
    
    // Download and extract binary
    const downloadUrl = PlatformDetector.getDownloadUrl(versionTag, platform);
    const binaryName = PlatformDetector.getBinaryName(platform);
    const archivePath = path.join(config.binaryDir, binaryName);
    const extractPath = config.binaryDir;
    
    try {
      // Download the binary archive
      await DownloadManager.downloadBinary(downloadUrl, archivePath, {
        onProgress: config.onProgress
      });
      
      // Download and verify checksums
      const checksums = await DownloadManager.downloadChecksums(versionTag);
      const expectedChecksum = checksums.get(binaryName);
      
      if (!expectedChecksum) {
        throw new Error(`No checksum found for ${binaryName}`);
      }
      
      // Verify checksum
      const isValidChecksum = await DownloadManager.verifyChecksum(archivePath, expectedChecksum);
      if (!isValidChecksum) {
        throw new Error('Checksum verification failed');
      }
      
      // Extract the archive
      await DownloadManager.extractTarGz(archivePath, extractPath);
      
      // Set executable permissions
      const binaryPath = this.getBinaryPath(config);
      await DownloadManager.setBinaryPermissions(binaryPath);
      
      // Clean up the archive
      await DownloadManager.cleanup([archivePath]);
      
      console.log(`Binary installed successfully at ${binaryPath}`);
    } catch (error) {
      // Clean up on failure
      await DownloadManager.cleanup([archivePath]);
      throw error;
    }
  }

  /**
   * Executes a binary process with proper stdio handling and exit code preservation
   * @private
   * @param {string} binaryPath - Path to the binary
   * @param {string[]} args - Command arguments
   * @param {Object} config - Execution configuration
   * @returns {Promise<number>} Exit code from the process
   */
  static async _executeBinaryProcess(binaryPath, args, config) {
    return new Promise((resolve, reject) => {
      const spawnOptions = {
        env: config.env,
        cwd: config.cwd,
        stdio: config.stdio ? 'inherit' : ['pipe', 'pipe', 'pipe']
      };
      
      const child = spawn(binaryPath, args, spawnOptions);
      
      let isResolved = false;
      let timeoutId = null;
      
      const cleanup = () => {
        if (timeoutId) {
          clearTimeout(timeoutId);
          timeoutId = null;
        }
      };
      
      const resolveOnce = (exitCode) => {
        if (!isResolved) {
          isResolved = true;
          cleanup();
          resolve(exitCode);
        }
      };
      
      const rejectOnce = (error) => {
        if (!isResolved) {
          isResolved = true;
          cleanup();
          if (!child.killed) {
            child.kill('SIGTERM');
          }
          reject(error);
        }
      };
      
      child.on('close', (exitCode) => {
        resolveOnce(exitCode || 0);
      });
      
      child.on('error', (error) => {
        rejectOnce(new Error(`Process execution failed: ${error.message}`));
      });
      
      // Handle timeout if specified
      if (config.timeout > 0) {
        timeoutId = setTimeout(() => {
          rejectOnce(new Error(`Process execution timed out after ${config.timeout}ms`));
        }, config.timeout);
      }
      
      // Handle process termination signals
      const handleSignal = (signal) => {
        if (!child.killed) {
          child.kill(signal);
        }
      };
      
      process.on('SIGINT', () => handleSignal('SIGINT'));
      process.on('SIGTERM', () => handleSignal('SIGTERM'));
    });
  }
}

module.exports = { BinaryManager };