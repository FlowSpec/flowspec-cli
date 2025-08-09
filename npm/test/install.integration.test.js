/**
 * Integration tests for the complete installation flow
 * Tests the main install.js script and InstallationManager class
 */

const fs = require('fs');
const path = require('path');
const { spawn } = require('child_process');
const { InstallationManager } = require('../install');
const { PlatformDetector } = require('../lib/platform');
const { BinaryManager } = require('../lib/binary');

// Test configuration
const TEST_CONFIG = {
  testDir: path.join(__dirname, '.test-install'),
  mockPackageJson: {
    name: '@flowspec/cli',
    version: '1.0.0',
    description: 'Test package'
  }
};

describe('InstallationManager Integration Tests', () => {
  let originalCwd;
  let testInstaller;
  
  beforeAll(() => {
    originalCwd = process.cwd();
    
    // Create test directory
    if (fs.existsSync(TEST_CONFIG.testDir)) {
      fs.rmSync(TEST_CONFIG.testDir, { recursive: true, force: true });
    }
    fs.mkdirSync(TEST_CONFIG.testDir, { recursive: true });
  });
  
  afterAll(() => {
    // Cleanup test directory
    if (fs.existsSync(TEST_CONFIG.testDir)) {
      fs.rmSync(TEST_CONFIG.testDir, { recursive: true, force: true });
    }
    
    process.chdir(originalCwd);
  });
  
  beforeEach(() => {
    // Create fresh installer instance for each test
    testInstaller = new InstallationManager();
    
    // Override config for testing
    testInstaller.config = {
      binaryDir: path.join(TEST_CONFIG.testDir, 'bin'),
      packageJsonPath: path.join(TEST_CONFIG.testDir, 'package.json'),
      tempDir: path.join(TEST_CONFIG.testDir, '.tmp'),
      maxRetries: 2, // Reduced for faster tests
      retryDelay: 100 // Reduced for faster tests
    };
    
    // Create mock package.json
    fs.writeFileSync(
      testInstaller.config.packageJsonPath,
      JSON.stringify(TEST_CONFIG.mockPackageJson, null, 2)
    );
  });
  
  afterEach(() => {
    // Cleanup after each test
    if (testInstaller) {
      testInstaller.cleanup().catch(() => {});
    }
  });

  describe('readVersion', () => {
    test('should read version from package.json successfully', () => {
      const version = testInstaller.readVersion();
      expect(version).toBe('1.0.0');
    });
    
    test('should throw error when package.json is missing', () => {
      fs.unlinkSync(testInstaller.config.packageJsonPath);
      
      expect(() => {
        testInstaller.readVersion();
      }).toThrow('package.json not found');
    });
    
    test('should throw error when version is missing from package.json', () => {
      const invalidPackageJson = { name: '@flowspec/cli' };
      fs.writeFileSync(
        testInstaller.config.packageJsonPath,
        JSON.stringify(invalidPackageJson, null, 2)
      );
      
      expect(() => {
        testInstaller.readVersion();
      }).toThrow('Version not found in package.json');
    });
  });

  describe('detectPlatform', () => {
    test('should detect platform successfully', () => {
      const platform = testInstaller.detectPlatform();
      
      expect(platform).toHaveProperty('os');
      expect(platform).toHaveProperty('arch');
      expect(platform).toHaveProperty('extension');
      expect(platform).toHaveProperty('nodePlatform');
      expect(platform).toHaveProperty('nodeArch');
    });
    
    test('should validate platform is supported', () => {
      const platform = testInstaller.detectPlatform();
      expect(PlatformDetector.isSupported(platform)).toBe(true);
    });
  });

  describe('prepareDirectories', () => {
    test('should create necessary directories', () => {
      testInstaller.prepareDirectories();
      
      expect(fs.existsSync(testInstaller.config.binaryDir)).toBe(true);
      expect(fs.existsSync(testInstaller.config.tempDir)).toBe(true);
    });
    
    test('should handle existing directories gracefully', () => {
      // Create directories first
      fs.mkdirSync(testInstaller.config.binaryDir, { recursive: true });
      fs.mkdirSync(testInstaller.config.tempDir, { recursive: true });
      
      // Should not throw error
      expect(() => {
        testInstaller.prepareDirectories();
      }).not.toThrow();
    });
  });

  describe('createProgressCallback', () => {
    test('should create a functional progress callback', () => {
      const callback = testInstaller.createProgressCallback();
      
      expect(typeof callback).toBe('function');
      
      // Should not throw when called
      expect(() => {
        callback(50, 1024, 2048);
      }).not.toThrow();
    });
    
    test('should handle progress updates correctly', () => {
      const consoleSpy = jest.spyOn(console, 'log').mockImplementation();
      const callback = testInstaller.createProgressCallback();
      
      // First call should log
      callback(10, 1024, 10240);
      expect(consoleSpy).toHaveBeenCalled();
      
      // Small increment should not log
      consoleSpy.mockClear();
      callback(15, 1536, 10240);
      expect(consoleSpy).not.toHaveBeenCalled();
      
      // Large increment should log
      callback(25, 2560, 10240);
      expect(consoleSpy).toHaveBeenCalled();
      
      consoleSpy.mockRestore();
    });
  });

  describe('printTroubleshootingGuidance', () => {
    test('should print platform-specific guidance', () => {
      const consoleSpy = jest.spyOn(console, 'log').mockImplementation();
      
      const platformError = new Error('Unsupported platform: test-arch');
      testInstaller.printTroubleshootingGuidance(platformError);
      
      expect(consoleSpy).toHaveBeenCalledWith(expect.stringContaining('Troubleshooting Guide'));
      expect(consoleSpy).toHaveBeenCalledWith(expect.stringContaining('platform is not supported'));
      
      consoleSpy.mockRestore();
    });
    
    test('should print network-specific guidance', () => {
      const consoleSpy = jest.spyOn(console, 'log').mockImplementation();
      
      const networkError = new Error('Network timeout occurred');
      testInstaller.printTroubleshootingGuidance(networkError);
      
      expect(consoleSpy).toHaveBeenCalledWith(expect.stringContaining('internet connection'));
      
      consoleSpy.mockRestore();
    });
    
    test('should print checksum-specific guidance', () => {
      const consoleSpy = jest.spyOn(console, 'log').mockImplementation();
      
      const checksumError = new Error('Checksum verification failed');
      testInstaller.printTroubleshootingGuidance(checksumError);
      
      expect(consoleSpy).toHaveBeenCalledWith(expect.stringContaining('downloaded file may be corrupted'));
      
      consoleSpy.mockRestore();
    });
    
    test('should print general guidance for unknown errors', () => {
      const consoleSpy = jest.spyOn(console, 'log').mockImplementation();
      
      const unknownError = new Error('Some unknown error');
      testInstaller.printTroubleshootingGuidance(unknownError);
      
      expect(consoleSpy).toHaveBeenCalledWith(expect.stringContaining('Try running the installation again'));
      
      consoleSpy.mockRestore();
    });
  });

  describe('cleanup', () => {
    test('should clean up temporary files', async () => {
      // Create some temporary files
      const tempFile1 = path.join(testInstaller.config.tempDir, 'temp1.txt');
      const tempFile2 = path.join(testInstaller.config.tempDir, 'temp2.txt');
      
      fs.mkdirSync(testInstaller.config.tempDir, { recursive: true });
      fs.writeFileSync(tempFile1, 'test content');
      fs.writeFileSync(tempFile2, 'test content');
      
      testInstaller.cleanupPaths = [tempFile1, tempFile2];
      
      await testInstaller.cleanup();
      
      expect(fs.existsSync(tempFile1)).toBe(false);
      expect(fs.existsSync(tempFile2)).toBe(false);
    });
    
    test('should handle cleanup errors gracefully', async () => {
      const consoleSpy = jest.spyOn(console, 'warn').mockImplementation();
      
      // Add non-existent path to cleanup
      testInstaller.cleanupPaths = ['/non/existent/path'];
      
      // Should not throw
      await expect(testInstaller.cleanup()).resolves.not.toThrow();
      
      consoleSpy.mockRestore();
    });
  });
});

describe('install.js Script Integration Tests', () => {
  const testDir = path.join(__dirname, '.test-script');
  const mockPackageJsonPath = path.join(testDir, 'package.json');
  
  beforeAll(() => {
    // Create test directory and mock package.json
    if (fs.existsSync(testDir)) {
      fs.rmSync(testDir, { recursive: true, force: true });
    }
    fs.mkdirSync(testDir, { recursive: true });
    
    fs.writeFileSync(
      mockPackageJsonPath,
      JSON.stringify(TEST_CONFIG.mockPackageJson, null, 2)
    );
  });
  
  afterAll(() => {
    if (fs.existsSync(testDir)) {
      fs.rmSync(testDir, { recursive: true, force: true });
    }
  });

  test('should be executable as a Node.js script', (done) => {
    const installScriptPath = path.join(__dirname, '..', 'install.js');
    
    // Test that the script can be executed (will fail due to missing binary, but should start)
    const child = spawn('node', [installScriptPath], {
      cwd: testDir,
      stdio: ['pipe', 'pipe', 'pipe']
    });
    
    let stdout = '';
    let stderr = '';
    
    child.stdout.on('data', (data) => {
      stdout += data.toString();
    });
    
    child.stderr.on('data', (data) => {
      stderr += data.toString();
    });
    
    child.on('close', (exitCode) => {
      // Script should start and show installation message
      expect(stdout).toContain('Starting flowspec-cli installation');
      expect(stdout).toContain('Installing flowspec-cli version');
      
      // Will fail due to network/binary not available, but that's expected
      expect(exitCode).not.toBe(0);
      
      done();
    });
    
    // Kill the process after a short time to avoid long-running test
    setTimeout(() => {
      if (!child.killed) {
        child.kill('SIGTERM');
      }
    }, 5000);
  }, 10000);

  test('should handle SIGINT gracefully', (done) => {
    const installScriptPath = path.join(__dirname, '..', 'install.js');
    
    const child = spawn(process.execPath, [installScriptPath], {
      cwd: testDir,
      stdio: ['pipe', 'pipe', 'pipe']
    });
    
    let stdout = '';
    
    child.stdout.on('data', (data) => {
      stdout += data.toString();
    });
    
    child.on('close', (exitCode) => {
      expect(stdout).toContain('Installation interrupted');
      expect(exitCode).toBe(1);
      done();
    });
    
    // Send SIGINT after script starts
    setTimeout(() => {
      child.kill('SIGINT');
    }, 1000);
  }, 10000);
});

describe('Error Handling Integration Tests', () => {
  let testInstaller;
  
  beforeEach(() => {
    testInstaller = new InstallationManager();
    testInstaller.config = {
      binaryDir: path.join(TEST_CONFIG.testDir, 'bin'),
      packageJsonPath: path.join(TEST_CONFIG.testDir, 'package.json'),
      tempDir: path.join(TEST_CONFIG.testDir, '.tmp'),
      maxRetries: 1, // Single retry for faster tests
      retryDelay: 50
    };
  });

  test('should handle missing package.json gracefully', async () => {
    const consoleSpy = jest.spyOn(console, 'error').mockImplementation();
    const exitSpy = jest.spyOn(process, 'exit').mockImplementation();
    
    // Remove package.json
    if (fs.existsSync(testInstaller.config.packageJsonPath)) {
      fs.unlinkSync(testInstaller.config.packageJsonPath);
    }
    
    await testInstaller.install();
    
    expect(consoleSpy).toHaveBeenCalledWith(
      expect.stringContaining('Installation failed'),
      expect.stringContaining('package.json not found')
    );
    expect(exitSpy).toHaveBeenCalledWith(1);
    
    consoleSpy.mockRestore();
    exitSpy.mockRestore();
  });

  test('should handle directory creation failures', async () => {
    const consoleSpy = jest.spyOn(console, 'error').mockImplementation();
    const exitSpy = jest.spyOn(process, 'exit').mockImplementation();
    
    // Ensure test directory exists and create package.json
    fs.mkdirSync(path.dirname(testInstaller.config.packageJsonPath), { recursive: true });
    fs.writeFileSync(
      testInstaller.config.packageJsonPath,
      JSON.stringify(TEST_CONFIG.mockPackageJson, null, 2)
    );
    
    // Mock fs.mkdirSync to throw error
    const originalMkdirSync = fs.mkdirSync;
    fs.mkdirSync = jest.fn().mockImplementation(() => {
      throw new Error('Permission denied');
    });
    
    await testInstaller.install();
    
    expect(consoleSpy).toHaveBeenCalledWith(
      expect.stringContaining('Installation failed'),
      expect.stringContaining('Failed to prepare directories')
    );
    expect(exitSpy).toHaveBeenCalledWith(1);
    
    // Restore original function
    fs.mkdirSync = originalMkdirSync;
    consoleSpy.mockRestore();
    exitSpy.mockRestore();
  });
});

describe('Progress Reporting Integration Tests', () => {
  test('should report progress during installation steps', async () => {
    const consoleSpy = jest.spyOn(console, 'log').mockImplementation();
    
    const testInstaller = new InstallationManager();
    testInstaller.config = {
      binaryDir: path.join(TEST_CONFIG.testDir, 'bin'),
      packageJsonPath: path.join(TEST_CONFIG.testDir, 'package.json'),
      tempDir: path.join(TEST_CONFIG.testDir, '.tmp'),
      maxRetries: 1,
      retryDelay: 50
    };
    
    // Ensure test directory exists and create package.json
    fs.mkdirSync(path.dirname(testInstaller.config.packageJsonPath), { recursive: true });
    fs.writeFileSync(
      testInstaller.config.packageJsonPath,
      JSON.stringify(TEST_CONFIG.mockPackageJson, null, 2)
    );
    
    // Mock process.exit to prevent actual exit
    const exitSpy = jest.spyOn(process, 'exit').mockImplementation();
    
    await testInstaller.install();
    
    // Check that progress messages were logged
    expect(consoleSpy).toHaveBeenCalledWith(expect.stringContaining('Starting flowspec-cli installation'));
    expect(consoleSpy).toHaveBeenCalledWith(expect.stringContaining('Installing flowspec-cli version'));
    expect(consoleSpy).toHaveBeenCalledWith(expect.stringContaining('Detected platform'));
    expect(consoleSpy).toHaveBeenCalledWith(expect.stringContaining('Preparing installation directories'));
    
    consoleSpy.mockRestore();
    exitSpy.mockRestore();
  });
});