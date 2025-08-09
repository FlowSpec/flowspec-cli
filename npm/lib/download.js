/**
 * Download and checksum verification utilities for flowspec-cli NPM wrapper
 * Handles downloading binaries from GitHub releases with integrity verification
 */

const https = require('https');
const http = require('http');
const fs = require('fs');
const path = require('path');
const crypto = require('crypto');
const { URL } = require('url');
const tar = require('tar');
const { spawn } = require('child_process');

class DownloadManager {
  /**
   * Default configuration for downloads
   */
  static DEFAULT_CONFIG = {
    maxRetries: 3,
    retryDelay: 1000, // Base delay in ms, will be exponentially increased
    timeout: 30000, // 30 seconds
    userAgent: 'flowspec-cli-npm-wrapper/1.0.0'
  };

  /**
   * Downloads a binary file from the specified URL with progress indication and retry logic
   * @param {string} url - The URL to download from
   * @param {string} targetPath - The local path where the file should be saved
   * @param {Object} options - Optional configuration
   * @param {number} options.maxRetries - Maximum number of retry attempts
   * @param {number} options.retryDelay - Base delay between retries in milliseconds
   * @param {Function} options.onProgress - Progress callback function
   * @returns {Promise<void>}
   */
  static async downloadBinary(url, targetPath, options = {}) {
    const config = { ...this.DEFAULT_CONFIG, ...options };
    let lastError;

    for (let attempt = 1; attempt <= config.maxRetries; attempt++) {
      try {
        console.log(`Downloading binary from ${url} (attempt ${attempt}/${config.maxRetries})`);
        
        await this._downloadFile(url, targetPath, {
          timeout: config.timeout,
          userAgent: config.userAgent,
          onProgress: config.onProgress
        });
        
        console.log(`Binary downloaded successfully to ${targetPath}`);
        return;
      } catch (error) {
        lastError = error;
        console.warn(`Download attempt ${attempt} failed: ${error.message}`);
        
        if (attempt < config.maxRetries) {
          const delay = config.retryDelay * Math.pow(2, attempt - 1); // Exponential backoff
          console.log(`Retrying in ${delay}ms...`);
          await this._sleep(delay);
        }
      }
    }

    throw new Error(`Failed to download binary after ${config.maxRetries} attempts. Last error: ${lastError.message}`);
  }

  /**
   * Downloads and parses the checksums.txt file from GitHub releases
   * @param {string} version - Version string (e.g., "v1.0.0")
   * @param {Object} options - Optional configuration
   * @returns {Promise<Map<string, string>>} Map of filename to SHA256 checksum
   */
  static async downloadChecksums(version, options = {}) {
    const config = { ...this.DEFAULT_CONFIG, ...options };
    const baseUrl = options.baseUrl || 'https://github.com/flowspec/flowspec-cli/releases/download';
    const checksumsUrl = `${baseUrl}/${version}/checksums.txt`;
    
    console.log(`Downloading checksums from ${checksumsUrl}`);
    
    try {
      const checksumsContent = await this._downloadText(checksumsUrl, {
        timeout: config.timeout,
        userAgent: config.userAgent
      });
      
      return this._parseChecksums(checksumsContent);
    } catch (error) {
      throw new Error(`Failed to download checksums: ${error.message}`);
    }
  }

  /**
   * Verifies the SHA256 checksum of a file
   * @param {string} filePath - Path to the file to verify
   * @param {string} expectedChecksum - Expected SHA256 checksum (hex string)
   * @returns {Promise<boolean>} True if checksum matches, false otherwise
   */
  static async verifyChecksum(filePath, expectedChecksum) {
    try {
      console.log(`Verifying checksum for ${filePath}`);
      
      if (!fs.existsSync(filePath)) {
        throw new Error(`File not found: ${filePath}`);
      }

      if (!expectedChecksum || typeof expectedChecksum !== 'string' || expectedChecksum.trim() === '') {
        throw new Error(`Invalid checksum format: ${expectedChecksum}`);
      }

      const actualChecksum = await this._calculateSHA256(filePath);
      const normalizedExpected = expectedChecksum.toLowerCase().trim();
      const normalizedActual = actualChecksum.toLowerCase().trim();
      
      const isValid = normalizedExpected === normalizedActual;
      
      if (isValid) {
        console.log(`Checksum verification passed for ${filePath}`);
      } else {
        console.error(`Checksum verification failed for ${filePath}`);
        console.error(`Expected: ${normalizedExpected}`);
        console.error(`Actual:   ${normalizedActual}`);
      }
      
      return isValid;
    } catch (error) {
      throw new Error(`Checksum verification failed: ${error.message}`);
    }
  }

  /**
   * Extracts a tar.gz archive to the specified directory
   * @param {string} archivePath - Path to the tar.gz archive
   * @param {string} extractPath - Directory to extract to
   * @param {Object} options - Optional configuration
   * @param {boolean} options.strip - Number of path components to strip (default: 0)
   * @returns {Promise<void>}
   */
  static async extractTarGz(archivePath, extractPath, options = {}) {
    const config = { strip: 0, ...options };
    
    try {
      console.log(`Extracting ${archivePath} to ${extractPath}`);
      
      if (!fs.existsSync(archivePath)) {
        throw new Error(`Archive not found: ${archivePath}`);
      }

      // Ensure extract directory exists
      if (!fs.existsSync(extractPath)) {
        fs.mkdirSync(extractPath, { recursive: true });
      }

      await tar.extract({
        file: archivePath,
        cwd: extractPath,
        strip: config.strip,
        preservePaths: false,
        preserveOwner: false
      });
      
      console.log(`Successfully extracted archive to ${extractPath}`);
    } catch (error) {
      throw new Error(`Failed to extract archive: ${error.message}`);
    }
  }

  /**
   * Sets executable permissions on a binary file (Unix systems only)
   * @param {string} binaryPath - Path to the binary file
   * @returns {Promise<void>}
   */
  static async setBinaryPermissions(binaryPath) {
    try {
      console.log(`Setting executable permissions for ${binaryPath}`);
      
      if (!fs.existsSync(binaryPath)) {
        throw new Error(`Binary not found: ${binaryPath}`);
      }

      // On Windows, no need to set permissions
      if (process.platform === 'win32') {
        console.log('Skipping permission setting on Windows');
        return;
      }

      // Set executable permissions (755 - owner: rwx, group: rx, others: rx)
      fs.chmodSync(binaryPath, 0o755);
      
      console.log(`Successfully set executable permissions for ${binaryPath}`);
    } catch (error) {
      throw new Error(`Failed to set binary permissions: ${error.message}`);
    }
  }

  /**
   * Verifies that a binary can be executed by running it with --version flag
   * @param {string} binaryPath - Path to the binary file
   * @param {Object} options - Optional configuration
   * @param {number} options.timeout - Timeout in milliseconds (default: 10000)
   * @returns {Promise<boolean>} True if binary executes successfully, false otherwise
   */
  static async verifyBinary(binaryPath, options = {}) {
    const config = { timeout: 10000, ...options };
    
    try {
      console.log(`Verifying binary at ${binaryPath}`);
      
      if (!fs.existsSync(binaryPath)) {
        throw new Error(`Binary not found: ${binaryPath}`);
      }

      // Test binary execution with --version flag
      const result = await this._executeBinaryCommand(binaryPath, ['--version'], {
        timeout: config.timeout
      });
      
      if (result.exitCode === 0) {
        console.log(`Binary verification successful: ${result.stdout.trim()}`);
        return true;
      } else {
        console.error(`Binary verification failed with exit code ${result.exitCode}`);
        console.error(`stderr: ${result.stderr}`);
        return false;
      }
    } catch (error) {
      console.error(`Binary verification failed: ${error.message}`);
      return false;
    }
  }

  /**
   * Cleans up temporary files and directories
   * @param {string[]} paths - Array of file/directory paths to clean up
   * @returns {Promise<void>}
   */
  static async cleanup(paths) {
    if (!Array.isArray(paths)) {
      paths = [paths];
    }

    for (const filePath of paths) {
      try {
        if (fs.existsSync(filePath)) {
          const stats = fs.statSync(filePath);
          
          if (stats.isDirectory()) {
            fs.rmSync(filePath, { recursive: true, force: true });
            console.log(`Cleaned up directory: ${filePath}`);
          } else {
            fs.unlinkSync(filePath);
            console.log(`Cleaned up file: ${filePath}`);
          }
        }
      } catch (error) {
        console.warn(`Failed to clean up ${filePath}: ${error.message}`);
      }
    }
  }

  /**
   * Downloads a file from URL to local path
   * @private
   * @param {string} url - URL to download from
   * @param {string} targetPath - Local path to save to
   * @param {Object} options - Download options
   * @returns {Promise<void>}
   */
  static async _downloadFile(url, targetPath, options = {}) {
    return new Promise((resolve, reject) => {
      let parsedUrl;
      try {
        parsedUrl = new URL(url);
        // Only allow http and https protocols
        if (!['http:', 'https:'].includes(parsedUrl.protocol)) {
          return reject(new Error(`Unsupported protocol: ${parsedUrl.protocol}`));
        }
      } catch (error) {
        return reject(new Error(`Invalid URL: ${url}`));
      }
      
      const client = parsedUrl.protocol === 'https:' ? https : http;
      
      // Ensure target directory exists
      const targetDir = path.dirname(targetPath);
      if (!fs.existsSync(targetDir)) {
        fs.mkdirSync(targetDir, { recursive: true });
      }

      const request = client.get(url, {
        timeout: options.timeout || 30000,
        headers: {
          'User-Agent': options.userAgent || 'flowspec-cli-npm-wrapper'
        }
      }, (response) => {
        // Handle redirects
        if (response.statusCode >= 300 && response.statusCode < 400 && response.headers.location) {
          return this._downloadFile(response.headers.location, targetPath, options)
            .then(resolve)
            .catch(reject);
        }

        if (response.statusCode !== 200) {
          return reject(new Error(`HTTP ${response.statusCode}: ${response.statusMessage}`));
        }

        const totalSize = parseInt(response.headers['content-length'], 10);
        let downloadedSize = 0;
        
        const fileStream = fs.createWriteStream(targetPath);
        
        response.on('data', (chunk) => {
          downloadedSize += chunk.length;
          
          if (options.onProgress && totalSize) {
            const progress = (downloadedSize / totalSize) * 100;
            options.onProgress(progress, downloadedSize, totalSize);
          }
        });

        response.pipe(fileStream);
        
        fileStream.on('finish', () => {
          fileStream.close();
          resolve();
        });
        
        fileStream.on('error', (error) => {
          fs.unlink(targetPath, () => {}); // Clean up partial file
          reject(error);
        });
      });

      request.on('error', reject);
      request.on('timeout', () => {
        request.destroy();
        reject(new Error('Download timeout'));
      });
    });
  }

  /**
   * Downloads text content from URL
   * @private
   * @param {string} url - URL to download from
   * @param {Object} options - Download options
   * @returns {Promise<string>} Downloaded text content
   */
  static async _downloadText(url, options = {}) {
    return new Promise((resolve, reject) => {
      const parsedUrl = new URL(url);
      const client = parsedUrl.protocol === 'https:' ? https : http;

      const request = client.get(url, {
        timeout: options.timeout || 30000,
        headers: {
          'User-Agent': options.userAgent || 'flowspec-cli-npm-wrapper'
        }
      }, (response) => {
        // Handle redirects
        if (response.statusCode >= 300 && response.statusCode < 400 && response.headers.location) {
          return this._downloadText(response.headers.location, options)
            .then(resolve)
            .catch(reject);
        }

        if (response.statusCode !== 200) {
          return reject(new Error(`HTTP ${response.statusCode}: ${response.statusMessage}`));
        }

        let data = '';
        response.setEncoding('utf8');
        
        response.on('data', (chunk) => {
          data += chunk;
        });
        
        response.on('end', () => {
          resolve(data);
        });
      });

      request.on('error', reject);
      request.on('timeout', () => {
        request.destroy();
        reject(new Error('Download timeout'));
      });
    });
  }

  /**
   * Parses checksums.txt content into a Map
   * @private
   * @param {string} content - Content of checksums.txt file
   * @returns {Map<string, string>} Map of filename to checksum
   */
  static _parseChecksums(content) {
    const checksums = new Map();
    const lines = content.trim().split('\n');
    
    for (const line of lines) {
      const trimmedLine = line.trim();
      if (!trimmedLine || trimmedLine.startsWith('#')) {
        continue; // Skip empty lines and comments
      }
      
      // Expected format: "checksum  filename" (one or more spaces between checksum and filename)
      const match = trimmedLine.match(/^([a-fA-F0-9]{64})\s+(.+)$/);
      if (match) {
        const [, checksum, filename] = match;
        checksums.set(filename, checksum);
      } else {
        console.warn(`Skipping invalid checksum line: ${trimmedLine}`);
      }
    }
    
    if (checksums.size === 0) {
      throw new Error('No valid checksums found in checksums.txt');
    }
    
    console.log(`Parsed ${checksums.size} checksums`);
    return checksums;
  }

  /**
   * Calculates SHA256 checksum of a file
   * @private
   * @param {string} filePath - Path to the file
   * @returns {Promise<string>} SHA256 checksum as hex string
   */
  static async _calculateSHA256(filePath) {
    return new Promise((resolve, reject) => {
      const hash = crypto.createHash('sha256');
      const stream = fs.createReadStream(filePath);
      
      stream.on('data', (data) => {
        hash.update(data);
      });
      
      stream.on('end', () => {
        resolve(hash.digest('hex'));
      });
      
      stream.on('error', reject);
    });
  }

  /**
   * Sleep utility for retry delays
   * @private
   * @param {number} ms - Milliseconds to sleep
   * @returns {Promise<void>}
   */
  static async _sleep(ms) {
    return new Promise(resolve => setTimeout(resolve, ms));
  }

  /**
   * Executes a binary command and returns the result
   * @private
   * @param {string} binaryPath - Path to the binary
   * @param {string[]} args - Command arguments
   * @param {Object} options - Execution options
   * @param {number} options.timeout - Timeout in milliseconds
   * @returns {Promise<{exitCode: number, stdout: string, stderr: string}>}
   */
  static async _executeBinaryCommand(binaryPath, args = [], options = {}) {
    return new Promise((resolve, reject) => {
      const child = spawn(binaryPath, args, {
        stdio: ['pipe', 'pipe', 'pipe']
      });

      let stdout = '';
      let stderr = '';
      let isResolved = false;
      let timeoutId = null;

      const cleanup = () => {
        if (timeoutId) {
          clearTimeout(timeoutId);
          timeoutId = null;
        }
      };

      const resolveOnce = (result) => {
        if (!isResolved) {
          isResolved = true;
          cleanup();
          resolve(result);
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

      child.stdout.on('data', (data) => {
        stdout += data.toString();
      });

      child.stderr.on('data', (data) => {
        stderr += data.toString();
      });

      child.on('close', (exitCode) => {
        resolveOnce({
          exitCode: exitCode || 0,
          stdout: stdout.trim(),
          stderr: stderr.trim()
        });
      });

      child.on('error', (error) => {
        rejectOnce(new Error(`Failed to execute binary: ${error.message}`));
      });

      // Handle timeout
      if (options.timeout) {
        timeoutId = setTimeout(() => {
          rejectOnce(new Error(`Binary execution timed out after ${options.timeout}ms`));
        }, options.timeout);
      }
    });
  }
}

module.exports = { DownloadManager };