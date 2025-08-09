/**
 * Unit tests for BinaryManager class
 * Tests binary management and execution scenarios
 */

const fs = require('fs');
const path = require('path');
const { spawn } = require('child_process');
const { BinaryManager } = require('../lib/binary');
const { PlatformDetector } = require('../lib/platform');
const { DownloadManager } = require('../lib/download');

// Mock dependencies
jest.mock('fs');
jest.mock('child_process');
jest.mock('../lib/platform');
jest.mock('../lib/download');

describe('BinaryManager', () => {
  const mockBinaryDir = '/mock/bin';
  const mockPackageJsonPath = '/mock/package.json';
  const mockBinaryPath = '/mock/bin/flowspec-cli';
  const mockWindowsBinaryPath = '/mock/bin/flowspec-cli.exe';
  
  const mockPlatform = {
    nodePlatform: 'linux',
    nodeArch: 'x64',
    os: 'linux',
    arch: 'amd64',
    extension: '.tar.gz'
  };
  
  const mockWindowsPlatform = {
    nodePlatform: 'win32',
    nodeArch: 'x64',
    os: 'windows',
    arch: 'amd64',
    extension: '.tar.gz'
  };
  
  const mockPackageJson = {
    name: '@flowspec/cli',
    version: '1.0.0'
  };

  beforeEach(() => {
    jest.clearAllMocks();
    
    // Default mocks
    PlatformDetector.detectPlatform.mockReturnValue(mockPlatform);
    PlatformDetector.getBinaryName.mockReturnValue('flowspec-cli-linux-amd64.tar.gz');
    PlatformDetector.getDownloadUrl.mockReturnValue('https://github.com/flowspec/flowspec-cli/releases/download/v1.0.0/flowspec-cli-linux-amd64.tar.gz');
    
    fs.existsSync.mockReturnValue(true);
    fs.readFileSync.mockReturnValue(JSON.stringify(mockPackageJson));
    fs.statSync.mockReturnValue({ mode: 0o755 }); // Executable permissions
    
    DownloadManager.verifyBinary.mockResolvedValue(true);
  });

  describe('getBinaryPath', () => {
    it('should return correct binary path for Linux/macOS', () => {
      PlatformDetector.detectPlatform.mockReturnValue(mockPlatform);
      
      const result = BinaryManager.getBinaryPath({
        binaryDir: mockBinaryDir
      });
      
      expect(result).toBe(mockBinaryPath);
      expect(PlatformDetector.detectPlatform).toHaveBeenCalled();
    });

    it('should return correct binary path for Windows with .exe extension', () => {
      PlatformDetector.detectPlatform.mockReturnValue(mockWindowsPlatform);
      
      const result = BinaryManager.getBinaryPath({
        binaryDir: mockBinaryDir
      });
      
      expect(result).toBe(mockWindowsBinaryPath);
    });

    it('should use default binary directory when not specified', () => {
      const result = BinaryManager.getBinaryPath();
      
      expect(result).toContain('flowspec-cli');
      expect(PlatformDetector.detectPlatform).toHaveBeenCalled();
    });

    it('should throw error when platform detection fails', () => {
      PlatformDetector.detectPlatform.mockImplementation(() => {
        throw new Error('Unsupported platform');
      });
      
      expect(() => {
        BinaryManager.getBinaryPath();
      }).toThrow('Failed to determine binary path: Unsupported platform');
    });
  });

  describe('ensureBinaryExists', () => {
    it('should do nothing when binary is already functional', async () => {
      fs.existsSync.mockReturnValue(true);
      DownloadManager.verifyBinary.mockResolvedValue(true);
      
      const consoleSpy = jest.spyOn(console, 'log').mockImplementation();
      
      await BinaryManager.ensureBinaryExists({
        binaryDir: mockBinaryDir,
        packageJsonPath: mockPackageJsonPath
      });
      
      expect(consoleSpy).toHaveBeenCalledWith(
        expect.stringContaining('Binary is already installed and functional')
      );
      expect(DownloadManager.downloadBinary).not.toHaveBeenCalled();
      
      consoleSpy.mockRestore();
    });

    it('should install binary when it does not exist', async () => {
      fs.existsSync
        .mockReturnValueOnce(false) // Binary doesn't exist initially
        .mockReturnValue(true); // But exists after installation
      
      fs.mkdirSync.mockImplementation();
      DownloadManager.downloadBinary.mockResolvedValue();
      DownloadManager.downloadChecksums.mockResolvedValue(new Map([
        ['flowspec-cli-linux-amd64.tar.gz', 'mock-checksum']
      ]));
      DownloadManager.verifyChecksum.mockResolvedValue(true);
      DownloadManager.extractTarGz.mockResolvedValue();
      DownloadManager.setBinaryPermissions.mockResolvedValue();
      DownloadManager.cleanup.mockResolvedValue();
      DownloadManager.verifyBinary.mockResolvedValue(true);
      
      const consoleSpy = jest.spyOn(console, 'log').mockImplementation();
      
      await BinaryManager.ensureBinaryExists({
        binaryDir: mockBinaryDir,
        packageJsonPath: mockPackageJsonPath
      });
      
      expect(DownloadManager.downloadBinary).toHaveBeenCalled();
      expect(DownloadManager.verifyChecksum).toHaveBeenCalled();
      expect(DownloadManager.extractTarGz).toHaveBeenCalled();
      expect(consoleSpy).toHaveBeenCalledWith('Binary installation completed successfully');
      
      consoleSpy.mockRestore();
    });

    it('should install binary when existing binary is not functional', async () => {
      fs.existsSync.mockReturnValue(true);
      DownloadManager.verifyBinary
        .mockResolvedValueOnce(false) // Not functional initially
        .mockResolvedValue(true); // Functional after installation
      
      DownloadManager.downloadBinary.mockResolvedValue();
      DownloadManager.downloadChecksums.mockResolvedValue(new Map([
        ['flowspec-cli-linux-amd64.tar.gz', 'mock-checksum']
      ]));
      DownloadManager.verifyChecksum.mockResolvedValue(true);
      DownloadManager.extractTarGz.mockResolvedValue();
      DownloadManager.setBinaryPermissions.mockResolvedValue();
      DownloadManager.cleanup.mockResolvedValue();
      
      await BinaryManager.ensureBinaryExists({
        binaryDir: mockBinaryDir,
        packageJsonPath: mockPackageJsonPath
      });
      
      expect(DownloadManager.downloadBinary).toHaveBeenCalled();
    });

    it('should throw error when reinstallOnFailure is disabled and binary is missing', async () => {
      fs.existsSync.mockReturnValue(false);
      
      await expect(
        BinaryManager.ensureBinaryExists({
          binaryDir: mockBinaryDir,
          reinstallOnFailure: false
        })
      ).rejects.toThrow('Binary not found or not functional');
    });

    it('should throw error when installation fails', async () => {
      fs.existsSync.mockReturnValue(false);
      DownloadManager.downloadBinary.mockRejectedValue(new Error('Download failed'));
      
      await expect(
        BinaryManager.ensureBinaryExists({
          binaryDir: mockBinaryDir,
          packageJsonPath: mockPackageJsonPath
        })
      ).rejects.toThrow('Failed to ensure binary exists');
    });

    it('should throw error when binary is still not functional after installation', async () => {
      fs.existsSync
        .mockReturnValueOnce(false) // Binary doesn't exist initially
        .mockReturnValue(true); // But exists after installation
      fs.mkdirSync.mockImplementation();
      DownloadManager.downloadBinary.mockResolvedValue();
      DownloadManager.downloadChecksums.mockResolvedValue(new Map([
        ['flowspec-cli-linux-amd64.tar.gz', 'mock-checksum']
      ]));
      DownloadManager.verifyChecksum.mockResolvedValue(true);
      DownloadManager.extractTarGz.mockResolvedValue();
      DownloadManager.setBinaryPermissions.mockResolvedValue();
      DownloadManager.cleanup.mockResolvedValue();
      DownloadManager.verifyBinary.mockResolvedValue(false); // Still not functional
      
      await expect(
        BinaryManager.ensureBinaryExists({
          binaryDir: mockBinaryDir,
          packageJsonPath: mockPackageJsonPath
        })
      ).rejects.toThrow('Binary installation completed but binary is still not functional');
    });
  });

  describe('executeBinary', () => {
    let mockChild;
    
    beforeEach(() => {
      mockChild = {
        on: jest.fn(),
        kill: jest.fn(),
        killed: false
      };
      spawn.mockReturnValue(mockChild);
    });

    it('should execute binary with correct arguments and return exit code', async () => {
      const args = ['--version'];
      const expectedExitCode = 0;
      
      // Mock successful execution
      mockChild.on.mockImplementation((event, callback) => {
        if (event === 'close') {
          setTimeout(() => callback(expectedExitCode), 0);
        }
      });
      
      const result = await BinaryManager.executeBinary(args, {
        binaryDir: mockBinaryDir,
        packageJsonPath: mockPackageJsonPath
      });
      
      expect(result).toBe(expectedExitCode);
      expect(spawn).toHaveBeenCalledWith(
        expect.stringContaining('flowspec-cli'),
        args,
        expect.objectContaining({
          env: process.env,
          cwd: process.cwd(),
          stdio: 'inherit'
        })
      );
    });

    it('should pass through custom environment variables', async () => {
      const args = ['test'];
      const customEnv = { ...process.env, CUSTOM_VAR: 'test-value' };
      
      mockChild.on.mockImplementation((event, callback) => {
        if (event === 'close') {
          setTimeout(() => callback(0), 0);
        }
      });
      
      await BinaryManager.executeBinary(args, {
        binaryDir: mockBinaryDir,
        packageJsonPath: mockPackageJsonPath,
        env: customEnv
      });
      
      expect(spawn).toHaveBeenCalledWith(
        expect.stringContaining('flowspec-cli'),
        args,
        expect.objectContaining({
          env: expect.objectContaining({
            CUSTOM_VAR: 'test-value'
          })
        })
      );
    });

    it('should use custom working directory', async () => {
      const args = ['test'];
      const customCwd = '/custom/working/dir';
      
      mockChild.on.mockImplementation((event, callback) => {
        if (event === 'close') {
          setTimeout(() => callback(0), 0);
        }
      });
      
      await BinaryManager.executeBinary(args, {
        binaryDir: mockBinaryDir,
        packageJsonPath: mockPackageJsonPath,
        cwd: customCwd
      });
      
      expect(spawn).toHaveBeenCalledWith(
        expect.stringContaining('flowspec-cli'),
        args,
        expect.objectContaining({
          cwd: customCwd
        })
      );
    });

    it('should handle process execution errors', async () => {
      const args = ['test'];
      const error = new Error('Process failed');
      
      mockChild.on.mockImplementation((event, callback) => {
        if (event === 'error') {
          setTimeout(() => callback(error), 0);
        }
      });
      
      await expect(
        BinaryManager.executeBinary(args, {
          binaryDir: mockBinaryDir,
          packageJsonPath: mockPackageJsonPath
        })
      ).rejects.toThrow('Failed to execute binary: Process execution failed: Process failed');
    });

    it('should handle execution timeout', async () => {
      const args = ['test'];
      const timeout = 1000;
      
      // Don't call any callbacks to simulate hanging process
      mockChild.on.mockImplementation(() => {});
      
      await expect(
        BinaryManager.executeBinary(args, {
          binaryDir: mockBinaryDir,
          packageJsonPath: mockPackageJsonPath,
          timeout
        })
      ).rejects.toThrow(`Process execution timed out after ${timeout}ms`);
      
      expect(mockChild.kill).toHaveBeenCalledWith('SIGTERM');
    });

    it('should ensure binary exists before execution', async () => {
      const args = ['test'];
      
      // Mock binary not existing initially
      fs.existsSync.mockReturnValueOnce(false);
      DownloadManager.downloadBinary.mockResolvedValue();
      DownloadManager.downloadChecksums.mockResolvedValue(new Map([
        ['flowspec-cli-linux-amd64.tar.gz', 'mock-checksum']
      ]));
      DownloadManager.verifyChecksum.mockResolvedValue(true);
      DownloadManager.extractTarGz.mockResolvedValue();
      DownloadManager.setBinaryPermissions.mockResolvedValue();
      DownloadManager.cleanup.mockResolvedValue();
      DownloadManager.verifyBinary.mockResolvedValue(true);
      
      mockChild.on.mockImplementation((event, callback) => {
        if (event === 'close') {
          setTimeout(() => callback(0), 0);
        }
      });
      
      await BinaryManager.executeBinary(args, {
        binaryDir: mockBinaryDir,
        packageJsonPath: mockPackageJsonPath
      });
      
      expect(DownloadManager.downloadBinary).toHaveBeenCalled();
      expect(spawn).toHaveBeenCalled();
    });
  });

  describe('getVersion', () => {
    it('should return version from package.json', () => {
      const result = BinaryManager.getVersion({
        packageJsonPath: mockPackageJsonPath
      });
      
      expect(result).toBe('1.0.0');
      expect(fs.readFileSync).toHaveBeenCalledWith(mockPackageJsonPath, 'utf8');
    });

    it('should throw error when package.json does not exist', () => {
      fs.existsSync.mockReturnValue(false);
      
      expect(() => {
        BinaryManager.getVersion({
          packageJsonPath: mockPackageJsonPath
        });
      }).toThrow('package.json not found');
    });

    it('should throw error when package.json is invalid JSON', () => {
      fs.readFileSync.mockReturnValue('invalid json');
      
      expect(() => {
        BinaryManager.getVersion({
          packageJsonPath: mockPackageJsonPath
        });
      }).toThrow('Failed to read version from package.json');
    });

    it('should throw error when version is missing from package.json', () => {
      fs.readFileSync.mockReturnValue(JSON.stringify({ name: '@flowspec/cli' }));
      
      expect(() => {
        BinaryManager.getVersion({
          packageJsonPath: mockPackageJsonPath
        });
      }).toThrow('Version not found in package.json');
    });
  });

  describe('getBinaryInfo', () => {
    it('should return complete binary information', () => {
      fs.existsSync.mockReturnValue(true);
      fs.statSync.mockReturnValue({ mode: 0o755 });
      
      const result = BinaryManager.getBinaryInfo({
        binaryDir: mockBinaryDir,
        packageJsonPath: mockPackageJsonPath
      });
      
      expect(result).toEqual({
        version: '1.0.0',
        binaryPath: mockBinaryPath,
        platform: mockPlatform,
        exists: true,
        isExecutable: true
      });
    });

    it('should indicate when binary does not exist', () => {
      fs.existsSync
        .mockReturnValueOnce(true) // package.json exists
        .mockReturnValue(false); // binary doesn't exist
      
      const result = BinaryManager.getBinaryInfo({
        binaryDir: mockBinaryDir,
        packageJsonPath: mockPackageJsonPath
      });
      
      expect(result.exists).toBe(false);
      expect(result.isExecutable).toBe(false);
    });
  });

  describe('removeBinary', () => {
    it('should remove existing binary', async () => {
      fs.existsSync.mockReturnValue(true);
      fs.unlinkSync.mockImplementation();
      
      const consoleSpy = jest.spyOn(console, 'log').mockImplementation();
      
      await BinaryManager.removeBinary({
        binaryDir: mockBinaryDir
      });
      
      expect(fs.unlinkSync).toHaveBeenCalledWith(mockBinaryPath);
      expect(consoleSpy).toHaveBeenCalledWith(`Removed binary at ${mockBinaryPath}`);
      
      consoleSpy.mockRestore();
    });

    it('should handle case when binary does not exist', async () => {
      fs.existsSync.mockReturnValue(false);
      
      const consoleSpy = jest.spyOn(console, 'log').mockImplementation();
      
      await BinaryManager.removeBinary({
        binaryDir: mockBinaryDir
      });
      
      expect(fs.unlinkSync).not.toHaveBeenCalled();
      expect(consoleSpy).toHaveBeenCalledWith(
        expect.stringContaining('Binary not found')
      );
      
      consoleSpy.mockRestore();
    });

    it('should throw error when removal fails', async () => {
      fs.existsSync.mockReturnValue(true);
      fs.unlinkSync.mockImplementation(() => {
        throw new Error('Permission denied');
      });
      
      await expect(
        BinaryManager.removeBinary({
          binaryDir: mockBinaryDir
        })
      ).rejects.toThrow('Failed to remove binary: Permission denied');
    });
  });

  describe('_isExecutable', () => {
    it('should return true for executable files on Unix', () => {
      fs.existsSync.mockReturnValue(true);
      fs.statSync.mockReturnValue({ mode: 0o755 }); // rwxr-xr-x
      
      const result = BinaryManager._isExecutable('/path/to/binary');
      
      expect(result).toBe(true);
    });

    it('should return false for non-executable files on Unix', () => {
      fs.existsSync.mockReturnValue(true);
      fs.statSync.mockReturnValue({ mode: 0o644 }); // rw-r--r--
      
      const result = BinaryManager._isExecutable('/path/to/binary');
      
      expect(result).toBe(false);
    });

    it('should return true for .exe files on Windows', () => {
      const originalPlatform = process.platform;
      Object.defineProperty(process, 'platform', { value: 'win32' });
      
      fs.existsSync.mockReturnValue(true);
      
      const result = BinaryManager._isExecutable('/path/to/binary.exe');
      
      expect(result).toBe(true);
      
      Object.defineProperty(process, 'platform', { value: originalPlatform });
    });

    it('should return false for non-existent files', () => {
      fs.existsSync.mockReturnValue(false);
      
      const result = BinaryManager._isExecutable('/path/to/nonexistent');
      
      expect(result).toBe(false);
    });

    it('should return false when stat fails', () => {
      fs.existsSync.mockReturnValue(true);
      fs.statSync.mockImplementation(() => {
        throw new Error('Stat failed');
      });
      
      const result = BinaryManager._isExecutable('/path/to/binary');
      
      expect(result).toBe(false);
    });
  });
});

describe('Additional Binary Manager Coverage', () => {
    beforeEach(() => {
      // Reset all mocks
      jest.clearAllMocks();
      
      // Set up default mocks
      fs.readFileSync.mockReturnValue(JSON.stringify({ name: 'test', version: '1.0.0' }));
    });
    
    test('should handle getBinaryInfo with non-existent binary', () => {
      // Mock package.json to exist but binary to not exist
      fs.existsSync.mockImplementation((filePath) => {
        return filePath.includes('package.json'); // package.json exists, binary doesn't
      });
      fs.readFileSync.mockReturnValue(JSON.stringify({ version: '1.0.0' }));
      
      const result = BinaryManager.getBinaryInfo();
      expect(result.exists).toBe(false);
      expect(result.version).toBe('1.0.0');
    });
    
    test('should handle getBinaryInfo with stat error', () => {
      fs.existsSync.mockReturnValue(true);
      fs.statSync.mockImplementation(() => {
        throw new Error('Stat failed');
      });
      
      expect(() => BinaryManager.getBinaryInfo()).toThrow('Failed to get binary info');
    });
    
    test('should handle removeBinary successfully', async () => {
      fs.existsSync.mockReturnValue(true);
      fs.unlinkSync.mockReturnValue();
      
      await expect(BinaryManager.removeBinary()).resolves.not.toThrow();
    });
    
    test('should handle removeBinary with non-existent binary', async () => {
      fs.existsSync.mockReturnValue(false);
      
      await expect(BinaryManager.removeBinary()).resolves.not.toThrow();
    });
    
    test('should handle removeBinary with unlink error', async () => {
      fs.existsSync.mockReturnValue(true);
      fs.unlinkSync.mockImplementation(() => {
        throw new Error('Unlink failed');
      });
      
      await expect(BinaryManager.removeBinary()).rejects.toThrow('Failed to remove binary');
    });
    
    test('should handle executeBinary with timeout', async () => {
      // Mock binary exists and is executable
      fs.existsSync.mockReturnValue(true);
      fs.statSync.mockReturnValue({ mode: 0o755 });
      
      const mockChild = {
        on: jest.fn(),
        kill: jest.fn(),
        killed: false
      };
      
      spawn.mockReturnValue(mockChild);
      
      // Mock child process that doesn't respond (simulates timeout)
      mockChild.on.mockImplementation((event, callback) => {
        // Don't call any callbacks to simulate hanging
      });
      
      await expect(BinaryManager.executeBinary(['--version'], { timeout: 100 }))
        .rejects.toThrow('Failed to execute binary');
    });
    
    test('should handle executeBinary with spawn error', async () => {
      spawn.mockImplementation(() => {
        throw new Error('Spawn failed');
      });
      
      fs.existsSync.mockReturnValue(true);
      
      await expect(BinaryManager.executeBinary(['--version']))
        .rejects.toThrow('Failed to execute binary');
    });
    
    test('should handle _isExecutable with permission check', () => {
      const binaryPath = '/path/to/binary';
      
      // Mock file exists and has executable permissions
      fs.existsSync.mockReturnValue(true);
      fs.statSync.mockReturnValue({ mode: 0o755 }); // Owner has execute permission
      
      const result = BinaryManager._isExecutable(binaryPath);
      expect(result).toBe(true);
      expect(fs.statSync).toHaveBeenCalledWith(binaryPath);
    });
    
    test('should handle _isExecutable with permission error', () => {
      const binaryPath = '/path/to/binary';
      
      fs.constants = { F_OK: 0, X_OK: 1 };
      fs.accessSync.mockImplementation(() => {
        throw new Error('Permission denied');
      });
      
      const result = BinaryManager._isExecutable(binaryPath);
      expect(result).toBe(false);
    });
    
    test('should handle ensureBinaryExists with installation failure', async () => {
      fs.existsSync.mockReturnValue(false);
      
      // Mock InstallManager to fail
      const mockInstallManager = {
        install: jest.fn().mockRejectedValue(new Error('Installation failed'))
      };
      
      jest.doMock('../install', () => ({
        InstallManager: jest.fn(() => mockInstallManager)
      }));
      
      await expect(BinaryManager.ensureBinaryExists())
        .rejects.toThrow('Failed to ensure binary exists');
    });
    
    test('should handle getVersion with corrupted package.json', () => {
      fs.readFileSync.mockReturnValue('invalid json');
      
      expect(() => BinaryManager.getVersion()).toThrow('Failed to read version');
    });
    
    test('should handle getVersion with missing version field', () => {
      fs.readFileSync.mockReturnValue(JSON.stringify({ name: 'test' }));
      
      expect(() => BinaryManager.getVersion()).toThrow('Failed to read version');
    });
    
    test('should handle executeBinary with stdio configuration', async () => {
      // Mock binary exists and is executable
      fs.existsSync.mockReturnValue(true);
      fs.statSync.mockReturnValue({ mode: 0o755 });
      
      const mockChild = {
        on: jest.fn(),
        kill: jest.fn(),
        killed: false
      };
      
      spawn.mockReturnValue(mockChild);
      
      mockChild.on.mockImplementation((event, callback) => {
        if (event === 'close') {
          setImmediate(() => callback(0, null));
        }
      });
      
      const result = await BinaryManager.executeBinary(['--version'], { stdio: true });
      expect(result).toBe(0);
      expect(spawn).toHaveBeenCalledWith(
        expect.any(String),
        ['--version'],
        expect.objectContaining({ stdio: 'inherit' })
      );
    });
    
    test('should handle executeBinary with custom environment', async () => {
      // Mock binary exists and is executable
      fs.existsSync.mockReturnValue(true);
      fs.statSync.mockReturnValue({ mode: 0o755 });
      
      const mockChild = {
        on: jest.fn(),
        kill: jest.fn(),
        killed: false
      };
      
      spawn.mockReturnValue(mockChild);
      
      mockChild.on.mockImplementation((event, callback) => {
        if (event === 'close') {
          setImmediate(() => callback(0, null));
        }
      });
      
      const customEnv = { CUSTOM_VAR: 'value' };
      const result = await BinaryManager.executeBinary(['--version'], { env: customEnv });
      
      expect(result).toBe(0);
      expect(spawn).toHaveBeenCalledWith(
        expect.any(String),
        ['--version'],
        expect.objectContaining({ env: customEnv })
      );
    });
  });