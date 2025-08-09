/**
 * Jest setup file for common test utilities and configurations
 */

// Increase timeout for integration tests
jest.setTimeout(30000);

// Mock console methods to reduce noise during tests
const originalConsoleLog = console.log;
const originalConsoleError = console.error;
const originalConsoleWarn = console.warn;

// Store original methods for restoration
global.originalConsole = {
  log: originalConsoleLog,
  error: originalConsoleError,
  warn: originalConsoleWarn
};

// Global test utilities
global.testUtils = {
  /**
   * Suppress console output during tests
   */
  suppressConsole: () => {
    console.log = jest.fn();
    console.error = jest.fn();
    console.warn = jest.fn();
  },

  /**
   * Restore console output
   */
  restoreConsole: () => {
    console.log = originalConsoleLog;
    console.error = originalConsoleError;
    console.warn = originalConsoleWarn;
  },

  /**
   * Create a temporary directory for testing
   */
  createTempDir: (baseName = 'test') => {
    const path = require('path');
    const fs = require('fs');
    
    const tempDir = path.join(__dirname, `.${baseName}-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`);
    fs.mkdirSync(tempDir, { recursive: true });
    
    return tempDir;
  },

  /**
   * Clean up temporary directory
   */
  cleanupTempDir: (tempDir) => {
    const fs = require('fs');
    
    if (fs.existsSync(tempDir)) {
      fs.rmSync(tempDir, { recursive: true, force: true });
    }
  },

  /**
   * Wait for a specified amount of time
   */
  sleep: (ms) => {
    return new Promise(resolve => setTimeout(resolve, ms));
  },

  /**
   * Create a mock HTTP server for testing
   */
  createMockServer: (port = 0) => {
    const http = require('http');
    
    return new Promise((resolve, reject) => {
      const server = http.createServer();
      
      server.listen(port, 'localhost', (err) => {
        if (err) {
          reject(err);
        } else {
          resolve({
            server,
            port: server.address().port,
            close: () => new Promise(resolve => server.close(resolve))
          });
        }
      });
    });
  },

  /**
   * Generate a random string for testing
   */
  randomString: (length = 10) => {
    return Math.random().toString(36).substring(2, 2 + length);
  },

  /**
   * Generate a mock checksum
   */
  generateMockChecksum: (content = 'mock content') => {
    const crypto = require('crypto');
    return crypto.createHash('sha256').update(content).digest('hex');
  }
};

// Global test constants
global.TEST_CONSTANTS = {
  SUPPORTED_PLATFORMS: [
    { nodePlatform: 'linux', nodeArch: 'x64', os: 'linux', arch: 'amd64', extension: '.tar.gz' },
    { nodePlatform: 'linux', nodeArch: 'arm64', os: 'linux', arch: 'arm64', extension: '.tar.gz' },
    { nodePlatform: 'darwin', nodeArch: 'x64', os: 'darwin', arch: 'amd64', extension: '.tar.gz' },
    { nodePlatform: 'darwin', nodeArch: 'arm64', os: 'darwin', arch: 'arm64', extension: '.tar.gz' },
    { nodePlatform: 'win32', nodeArch: 'x64', os: 'windows', arch: 'amd64', extension: '.tar.gz' }
  ],
  
  UNSUPPORTED_PLATFORMS: [
    { nodePlatform: 'freebsd', nodeArch: 'x64' },
    { nodePlatform: 'linux', nodeArch: 'ia32' },
    { nodePlatform: 'win32', nodeArch: 'arm64' }
  ],

  MOCK_PACKAGE_JSON: {
    name: '@flowspec/cli',
    version: '1.0.0',
    description: 'Test package'
  },

  MOCK_CHECKSUMS: {
    'flowspec-cli-linux-amd64.tar.gz': 'abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890',
    'flowspec-cli-darwin-amd64.tar.gz': '1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef',
    'flowspec-cli-windows-amd64.tar.gz': 'fedcba0987654321fedcba0987654321fedcba0987654321fedcba0987654321'
  }
};

// Global error matchers for consistent error testing
expect.extend({
  toBeNetworkError(received) {
    const networkErrorCodes = ['ENOTFOUND', 'ECONNREFUSED', 'ETIMEDOUT', 'ECONNRESET'];
    const pass = received instanceof Error && networkErrorCodes.includes(received.code);
    
    if (pass) {
      return {
        message: () => `expected ${received} not to be a network error`,
        pass: true,
      };
    } else {
      return {
        message: () => `expected ${received} to be a network error`,
        pass: false,
      };
    }
  },

  toBeFileSystemError(received) {
    const fsErrorCodes = ['ENOENT', 'EACCES', 'EPERM', 'ENOSPC', 'EMFILE'];
    const pass = received instanceof Error && fsErrorCodes.includes(received.code);
    
    if (pass) {
      return {
        message: () => `expected ${received} not to be a filesystem error`,
        pass: true,
      };
    } else {
      return {
        message: () => `expected ${received} to be a filesystem error`,
        pass: false,
      };
    }
  },

  toBeValidChecksum(received) {
    const pass = typeof received === 'string' && /^[a-fA-F0-9]{64}$/.test(received);
    
    if (pass) {
      return {
        message: () => `expected ${received} not to be a valid SHA256 checksum`,
        pass: true,
      };
    } else {
      return {
        message: () => `expected ${received} to be a valid SHA256 checksum`,
        pass: false,
      };
    }
  }
});

// Clean up any existing test artifacts before starting
const fs = require('fs');
const path = require('path');

const testDir = path.join(__dirname);
const testArtifacts = fs.readdirSync(testDir).filter(file => 
  file.startsWith('.test-') || file.startsWith('.temp-') || file.startsWith('.mock-')
);

testArtifacts.forEach(artifact => {
  const artifactPath = path.join(testDir, artifact);
  try {
    if (fs.statSync(artifactPath).isDirectory()) {
      fs.rmSync(artifactPath, { recursive: true, force: true });
    } else {
      fs.unlinkSync(artifactPath);
    }
  } catch (error) {
    // Ignore cleanup errors
  }
});

// Set up global error handlers for unhandled promises
process.on('unhandledRejection', (reason, promise) => {
  console.error('Unhandled Rejection at:', promise, 'reason:', reason);
});

process.on('uncaughtException', (error) => {
  console.error('Uncaught Exception:', error);
});