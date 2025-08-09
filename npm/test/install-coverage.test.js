/**
 * Install Script Coverage Tests
 * 
 * These tests are designed to improve test coverage for the install.js script
 * by testing various installation scenarios and error conditions.
 */

const fs = require('fs');
const path = require('path');
const { PlatformDetector } = require('../lib/platform');
const { DownloadManager } = require('../lib/download');
const { BinaryManager } = require('../lib/binary');

// Mock all dependencies
jest.mock('fs');
jest.mock('path');
jest.mock('../lib/platform');
jest.mock('../lib/download');
jest.mock('../lib/binary');

// Import InstallationManager after mocks are set up
const { InstallationManager } = require('../install');

describe('Install Script Coverage Tests', () => {
  
  // Helper function to set up common mocks
  const setupCommonMocks = () => {
    // Mock path.join to return predictable paths
    path.join.mockImplementation((...args) => args.join('/'));
    
    // Mock filesystem operations
    fs.existsSync.mockImplementation((filePath) => {
      // Return true for package.json and directories
      return filePath.includes('package.json') || filePath.includes('bin') || filePath.includes('.tmp');
    });
    fs.mkdirSync.mockReturnValue();
    fs.readFileSync.mockImplementation((filePath, encoding) => {
      if (filePath.includes('package.json')) {
        return JSON.stringify({ version: '1.0.0' });
      }
      return '';
    });
    
    // Mock successful platform detection
    PlatformDetector.detectPlatform.mockReturnValue({ os: 'linux', arch: 'x64' });
    PlatformDetector.getBinaryName.mockReturnValue('flowspec-cli-linux-x64');
    PlatformDetector.getDownloadUrl.mockReturnValue('https://example.com/binary.tar.gz');
    PlatformDetector.isSupported.mockReturnValue(true);
    
    // Mock successful download and verification
    DownloadManager.downloadBinary.mockResolvedValue();
    const mockChecksums = new Map();
    mockChecksums.set('flowspec-cli-linux-x64', 'mock-checksum-value');
    DownloadManager.downloadChecksums.mockResolvedValue(mockChecksums);
    DownloadManager.verifyChecksum.mockResolvedValue(true);
    DownloadManager.extractTarGz.mockResolvedValue();
    DownloadManager.setBinaryPermissions.mockResolvedValue();
    DownloadManager.verifyBinary.mockResolvedValue(true);
    DownloadManager.cleanup.mockResolvedValue();
    
    // Mock BinaryManager
    BinaryManager.getBinaryPath.mockReturnValue('/mock/path/to/binary');
    BinaryManager.getBinaryInfo.mockReturnValue({
      name: 'flowspec-cli',
      version: '1.0.0',
      path: '/mock/path/to/binary'
    });
  };
  
  beforeEach(() => {
    // Clear all mocks
    jest.clearAllMocks();
    
    // Mock console methods to reduce noise (but keep log for debugging)
    jest.spyOn(console, 'log').mockImplementation(() => {});
    // console.error is mocked above to capture error messages
    jest.spyOn(console, 'warn').mockImplementation(() => {});
    
    // Mock process.exit to throw error instead of exiting
    // We'll capture the original error from console.error calls
    let lastError = null;
    jest.spyOn(console, 'error').mockImplementation((...args) => {
      // Capture the error message for later use
      const fullMessage = args.join(' ');
      if (fullMessage.includes('Installation failed:')) {
        lastError = fullMessage.replace(/.*Installation failed:\s*/, '');
      }
    });
    jest.spyOn(process, 'exit').mockImplementation((code) => {
      if (lastError) {
        throw new Error(lastError);
      }
      throw new Error(`Process exit with code ${code}`);
    });
    
    // Set up common mocks
    setupCommonMocks();
  });
  
  afterEach(() => {
    // Restore console methods
    console.log.mockRestore();
    console.error.mockRestore();
    console.warn.mockRestore();
    
    // Restore process.exit
    if (process.exit.mockRestore) {
      process.exit.mockRestore();
    }
  });
  
  test('should handle successful installation', async () => {
    const installer = new InstallationManager();
    
    await installer.install();
    
    expect(PlatformDetector.detectPlatform).toHaveBeenCalled();
    expect(DownloadManager.downloadBinary).toHaveBeenCalled();
    expect(console.log).toHaveBeenCalledWith(expect.stringContaining('installation completed successfully'));
  });
  
  test('should handle unsupported platform', async () => {
    // Override the common mock to simulate unsupported platform
    PlatformDetector.detectPlatform.mockReturnValue({ os: 'unsupported', arch: 'unknown' });
    PlatformDetector.isSupported.mockReturnValue(false);
    PlatformDetector.getSupportedPlatforms.mockReturnValue([
      { os: 'linux', arch: 'x64' },
      { os: 'darwin', arch: 'arm64' }
    ]);
    
    const installer = new InstallationManager();
    
    await expect(installer.install()).rejects.toThrow('Unsupported platform');
  });
  
  test('should handle download failure', async () => {
    // Override the common mock to simulate download failure
    DownloadManager.downloadBinary.mockRejectedValue(new Error('Download failed'));
    
    const installer = new InstallationManager();
    
    await expect(installer.install()).rejects.toThrow('Download failed');
  });
  
  test('should handle checksum verification failure', async () => {
    // Override the common mock to simulate checksum failure
    DownloadManager.verifyChecksum.mockResolvedValue(false);
    
    const installer = new InstallationManager();
    
    await expect(installer.install()).rejects.toThrow('SHA256 checksum verification failed');
  });
  
  test('should handle extraction failure', async () => {
    // Override the common mock to simulate extraction failure
    DownloadManager.extractTarGz.mockRejectedValue(new Error('Extraction failed'));
    
    const installer = new InstallationManager();
    
    await expect(installer.install()).rejects.toThrow('Extraction failed');
  });
  
  test('should handle cleanup failure during error', async () => {
    // Override the common mock to simulate download failure and cleanup failure
    DownloadManager.downloadBinary.mockRejectedValue(new Error('Download failed'));
    DownloadManager.cleanup.mockRejectedValue(new Error('Cleanup failed'));
    
    const installer = new InstallationManager();
    
    await expect(installer.install()).rejects.toThrow('Download failed');
  });
  
  test('should handle directory creation', async () => {
    // Override the common mock to simulate directories don't exist initially
    fs.existsSync.mockImplementation((filePath) => {
      // Return true for package.json and binary path (after creation), false for directories initially
      return filePath.includes('package.json') || filePath.includes('/mock/path/to/binary');
    });
    
    const installer = new InstallationManager();
    await installer.install();
    
    expect(fs.mkdirSync).toHaveBeenCalledWith(expect.any(String), { recursive: true });
  });
  
  test('should handle existing directory', async () => {
    // Override the common mock to simulate all directories exist
    fs.existsSync.mockReturnValue(true);
    
    const installer = new InstallationManager();
    await installer.install();
    
    expect(fs.mkdirSync).not.toHaveBeenCalled();
  });
  
  test('should handle version reading from package.json', async () => {
    // Override the common mock to use a different version
    fs.readFileSync.mockImplementation((filePath, encoding) => {
      if (filePath.includes('package.json')) {
        return JSON.stringify({ name: '@flowspec/cli', version: '2.1.0' });
      }
      return '';
    });
    
    const installer = new InstallationManager();
    await installer.install();
    
    expect(fs.readFileSync).toHaveBeenCalledWith(expect.stringContaining('package.json'), 'utf8');
  });
  
  test('should handle malformed package.json', async () => {
    // Override the common mock to simulate malformed package.json
    fs.readFileSync.mockImplementation((filePath, encoding) => {
      if (filePath.includes('package.json')) {
        return 'invalid json';
      }
      return '';
    });
    
    const installer = new InstallationManager();
    
    await expect(installer.install()).rejects.toThrow();
  });
});