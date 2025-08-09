/**
 * Integration tests for DownloadManager extraction and setup functionality
 * Tests the complete flow of extracting archives and setting up binaries
 */

const fs = require('fs');
const path = require('path');
const tar = require('tar');
const { DownloadManager } = require('../lib/download');

describe('DownloadManager Integration Tests', () => {
  const testDir = path.join(__dirname, 'temp-integration');
  const testArchivePath = path.join(testDir, 'test-binary.tar.gz');
  const extractDir = path.join(testDir, 'extracted');
  const binaryName = process.platform === 'win32' ? 'test-binary.exe' : 'test-binary';
  const binaryPath = path.join(extractDir, binaryName);

  beforeEach(async () => {
    // Clean up and create test directory
    if (fs.existsSync(testDir)) {
      fs.rmSync(testDir, { recursive: true, force: true });
    }
    fs.mkdirSync(testDir, { recursive: true });
  });

  afterEach(async () => {
    // Clean up test directory
    if (fs.existsSync(testDir)) {
      fs.rmSync(testDir, { recursive: true, force: true });
    }
  });

  describe('extractTarGz', () => {
    beforeEach(async () => {
      // Create a test binary file
      const tempBinaryDir = path.join(testDir, 'temp-binary');
      fs.mkdirSync(tempBinaryDir, { recursive: true });
      
      const tempBinaryPath = path.join(tempBinaryDir, binaryName);
      
      // Create a simple shell script or batch file that acts as a binary
      if (process.platform === 'win32') {
        fs.writeFileSync(tempBinaryPath, '@echo off\necho test-binary version 1.0.0\n');
      } else {
        fs.writeFileSync(tempBinaryPath, '#!/bin/bash\necho "test-binary version 1.0.0"\n');
        fs.chmodSync(tempBinaryPath, 0o755);
      }

      // Create tar.gz archive
      await tar.create({
        gzip: true,
        file: testArchivePath,
        cwd: tempBinaryDir
      }, [binaryName]);

      // Clean up temp binary directory
      fs.rmSync(tempBinaryDir, { recursive: true, force: true });
    });

    test('should extract tar.gz archive successfully', async () => {
      await DownloadManager.extractTarGz(testArchivePath, extractDir);

      expect(fs.existsSync(binaryPath)).toBe(true);
      
      // Verify file content
      const content = fs.readFileSync(binaryPath, 'utf8');
      if (process.platform === 'win32') {
        expect(content).toContain('test-binary version 1.0.0');
      } else {
        expect(content).toContain('#!/bin/bash');
        expect(content).toContain('test-binary version 1.0.0');
      }
    });

    test('should handle custom strip option', async () => {
      // Create archive with nested directory structure
      const nestedDir = path.join(testDir, 'flowspec-cli-v1.0.0');
      fs.mkdirSync(nestedDir, { recursive: true });
      
      const nestedBinaryPath = path.join(nestedDir, binaryName);
      if (process.platform === 'win32') {
        fs.writeFileSync(nestedBinaryPath, '@echo off\necho nested-binary version 1.0.0\n');
      } else {
        fs.writeFileSync(nestedBinaryPath, '#!/bin/bash\necho "nested-binary version 1.0.0"\n');
        fs.chmodSync(nestedBinaryPath, 0o755);
      }

      const nestedArchivePath = path.join(testDir, 'nested-binary.tar.gz');
      await tar.create({
        gzip: true,
        file: nestedArchivePath,
        cwd: testDir
      }, ['flowspec-cli-v1.0.0']);

      await DownloadManager.extractTarGz(nestedArchivePath, extractDir, { strip: 1 });

      expect(fs.existsSync(binaryPath)).toBe(true);
    });

    test('should throw error for non-existent archive', async () => {
      const nonExistentPath = path.join(testDir, 'non-existent.tar.gz');
      
      await expect(DownloadManager.extractTarGz(nonExistentPath, extractDir))
        .rejects.toThrow('Archive not found');
    });

    test('should create extract directory if it does not exist', async () => {
      const newExtractDir = path.join(testDir, 'new-extract-dir');
      
      await DownloadManager.extractTarGz(testArchivePath, newExtractDir);
      
      expect(fs.existsSync(newExtractDir)).toBe(true);
      expect(fs.existsSync(path.join(newExtractDir, binaryName))).toBe(true);
    });
  });

  describe('setBinaryPermissions', () => {
    beforeEach(() => {
      // Create a test binary file
      fs.mkdirSync(extractDir, { recursive: true });
      fs.writeFileSync(binaryPath, process.platform === 'win32' ? 
        '@echo off\necho test-binary version 1.0.0\n' : 
        '#!/bin/bash\necho "test-binary version 1.0.0"\n'
      );
    });

    test('should set executable permissions on Unix systems', async () => {
      await DownloadManager.setBinaryPermissions(binaryPath);

      if (process.platform !== 'win32') {
        const stats = fs.statSync(binaryPath);
        const mode = stats.mode & parseInt('777', 8);
        expect(mode).toBe(parseInt('755', 8));
      }
    });

    test('should skip permission setting on Windows', async () => {
      const originalPlatform = process.platform;
      
      // Mock Windows platform
      Object.defineProperty(process, 'platform', {
        value: 'win32',
        configurable: true
      });

      await expect(DownloadManager.setBinaryPermissions(binaryPath))
        .resolves.not.toThrow();

      // Restore original platform
      Object.defineProperty(process, 'platform', {
        value: originalPlatform,
        configurable: true
      });
    });

    test('should throw error for non-existent binary', async () => {
      const nonExistentPath = path.join(testDir, 'non-existent-binary');
      
      await expect(DownloadManager.setBinaryPermissions(nonExistentPath))
        .rejects.toThrow('Binary not found');
    });
  });

  describe('verifyBinary', () => {
    beforeEach(() => {
      // Create a test binary file that responds to --version
      fs.mkdirSync(extractDir, { recursive: true });
      
      if (process.platform === 'win32') {
        fs.writeFileSync(binaryPath, `@echo off
if "%1"=="--version" (
  echo test-binary version 1.0.0
  exit /b 0
) else (
  echo Unknown command
  exit /b 1
)
`);
      } else {
        fs.writeFileSync(binaryPath, `#!/bin/bash
if [ "$1" = "--version" ]; then
  echo "test-binary version 1.0.0"
  exit 0
else
  echo "Unknown command"
  exit 1
fi
`);
        fs.chmodSync(binaryPath, 0o755);
      }
    });

    test('should verify binary successfully with --version flag', async () => {
      const result = await DownloadManager.verifyBinary(binaryPath);
      expect(result).toBe(true);
    });

    test('should return false for non-existent binary', async () => {
      const nonExistentPath = path.join(testDir, 'non-existent-binary');
      const result = await DownloadManager.verifyBinary(nonExistentPath);
      expect(result).toBe(false);
    });

    test('should handle binary execution timeout', async () => {
      // Create a binary that hangs
      if (process.platform === 'win32') {
        fs.writeFileSync(binaryPath, '@echo off\ntimeout /t 5 /nobreak > nul\n');
      } else {
        fs.writeFileSync(binaryPath, '#!/bin/bash\nsleep 5\n');
        fs.chmodSync(binaryPath, 0o755);
      }

      const result = await DownloadManager.verifyBinary(binaryPath, { timeout: 500 });
      expect(result).toBe(false);
    }, 5000);

    test('should handle binary that returns non-zero exit code', async () => {
      // Create a binary that always fails
      if (process.platform === 'win32') {
        fs.writeFileSync(binaryPath, '@echo off\nexit /b 1\n');
      } else {
        fs.writeFileSync(binaryPath, '#!/bin/bash\nexit 1\n');
        fs.chmodSync(binaryPath, 0o755);
      }

      const result = await DownloadManager.verifyBinary(binaryPath);
      expect(result).toBe(false);
    });
  });

  describe('cleanup', () => {
    test('should clean up single file', async () => {
      const testFile = path.join(testDir, 'test-file.txt');
      fs.writeFileSync(testFile, 'test content');
      
      expect(fs.existsSync(testFile)).toBe(true);
      
      await DownloadManager.cleanup(testFile);
      
      expect(fs.existsSync(testFile)).toBe(false);
    });

    test('should clean up directory recursively', async () => {
      const testSubDir = path.join(testDir, 'subdir');
      fs.mkdirSync(testSubDir, { recursive: true });
      fs.writeFileSync(path.join(testSubDir, 'file.txt'), 'content');
      
      expect(fs.existsSync(testSubDir)).toBe(true);
      
      await DownloadManager.cleanup(testSubDir);
      
      expect(fs.existsSync(testSubDir)).toBe(false);
    });

    test('should clean up multiple paths', async () => {
      const testFile1 = path.join(testDir, 'file1.txt');
      const testFile2 = path.join(testDir, 'file2.txt');
      const testSubDir = path.join(testDir, 'subdir');
      
      fs.writeFileSync(testFile1, 'content1');
      fs.writeFileSync(testFile2, 'content2');
      fs.mkdirSync(testSubDir);
      
      await DownloadManager.cleanup([testFile1, testFile2, testSubDir]);
      
      expect(fs.existsSync(testFile1)).toBe(false);
      expect(fs.existsSync(testFile2)).toBe(false);
      expect(fs.existsSync(testSubDir)).toBe(false);
    });

    test('should handle non-existent paths gracefully', async () => {
      const nonExistentPath = path.join(testDir, 'non-existent');
      
      await expect(DownloadManager.cleanup(nonExistentPath))
        .resolves.not.toThrow();
    });
  });

  describe('Complete extraction and setup flow', () => {
    test('should complete full extraction and setup process', async () => {
      // Create a realistic binary archive
      const tempBinaryDir = path.join(testDir, 'flowspec-cli-v1.0.0');
      fs.mkdirSync(tempBinaryDir, { recursive: true });
      
      const tempBinaryPath = path.join(tempBinaryDir, 'flowspec-cli' + (process.platform === 'win32' ? '.exe' : ''));
      
      if (process.platform === 'win32') {
        fs.writeFileSync(tempBinaryPath, `@echo off
if "%1"=="--version" (
  echo flowspec-cli version 1.0.0
  exit /b 0
) else (
  echo FlowSpec CLI - A powerful tool for ServiceSpec validation
  exit /b 0
)
`);
      } else {
        fs.writeFileSync(tempBinaryPath, `#!/bin/bash
if [ "$1" = "--version" ]; then
  echo "flowspec-cli version 1.0.0"
  exit 0
else
  echo "FlowSpec CLI - A powerful tool for ServiceSpec validation"
  exit 0
fi
`);
        fs.chmodSync(tempBinaryPath, 0o755);
      }

      // Create archive
      await tar.create({
        gzip: true,
        file: testArchivePath,
        cwd: testDir
      }, ['flowspec-cli-v1.0.0']);

      // Clean up temp directory
      fs.rmSync(tempBinaryDir, { recursive: true, force: true });

      // Test complete flow - extract with strip: 1 to remove the version directory
      await DownloadManager.extractTarGz(testArchivePath, extractDir, { strip: 1 });
      
      const finalBinaryPath = path.join(extractDir, 'flowspec-cli' + (process.platform === 'win32' ? '.exe' : ''));
      expect(fs.existsSync(finalBinaryPath)).toBe(true);

      await DownloadManager.setBinaryPermissions(finalBinaryPath);
      
      const verificationResult = await DownloadManager.verifyBinary(finalBinaryPath);
      expect(verificationResult).toBe(true);

      // Test cleanup
      await DownloadManager.cleanup([testArchivePath, extractDir]);
      
      expect(fs.existsSync(testArchivePath)).toBe(false);
      expect(fs.existsSync(extractDir)).toBe(false);
    });
  });
});