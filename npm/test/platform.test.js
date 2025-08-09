/**
 * Unit tests for platform detection utilities
 */

const { PlatformDetector } = require('../lib/platform');

describe('PlatformDetector', () => {
  // Store original process values to restore after tests
  const originalPlatform = process.platform;
  const originalArch = process.arch;

  afterEach(() => {
    // Restore original values after each test
    Object.defineProperty(process, 'platform', {
      value: originalPlatform,
      writable: true
    });
    Object.defineProperty(process, 'arch', {
      value: originalArch,
      writable: true
    });
  });

  describe('detectPlatform()', () => {
    test('should detect Linux x64 platform correctly', () => {
      Object.defineProperty(process, 'platform', { value: 'linux', writable: true });
      Object.defineProperty(process, 'arch', { value: 'x64', writable: true });

      const result = PlatformDetector.detectPlatform();

      expect(result).toEqual({
        nodePlatform: 'linux',
        nodeArch: 'x64',
        os: 'linux',
        arch: 'amd64',
        extension: '.tar.gz'
      });
    });

    test('should detect Linux arm64 platform correctly', () => {
      Object.defineProperty(process, 'platform', { value: 'linux', writable: true });
      Object.defineProperty(process, 'arch', { value: 'arm64', writable: true });

      const result = PlatformDetector.detectPlatform();

      expect(result).toEqual({
        nodePlatform: 'linux',
        nodeArch: 'arm64',
        os: 'linux',
        arch: 'arm64',
        extension: '.tar.gz'
      });
    });

    test('should detect macOS x64 platform correctly', () => {
      Object.defineProperty(process, 'platform', { value: 'darwin', writable: true });
      Object.defineProperty(process, 'arch', { value: 'x64', writable: true });

      const result = PlatformDetector.detectPlatform();

      expect(result).toEqual({
        nodePlatform: 'darwin',
        nodeArch: 'x64',
        os: 'darwin',
        arch: 'amd64',
        extension: '.tar.gz'
      });
    });

    test('should detect macOS arm64 platform correctly', () => {
      Object.defineProperty(process, 'platform', { value: 'darwin', writable: true });
      Object.defineProperty(process, 'arch', { value: 'arm64', writable: true });

      const result = PlatformDetector.detectPlatform();

      expect(result).toEqual({
        nodePlatform: 'darwin',
        nodeArch: 'arm64',
        os: 'darwin',
        arch: 'arm64',
        extension: '.tar.gz'
      });
    });

    test('should detect Windows x64 platform correctly', () => {
      Object.defineProperty(process, 'platform', { value: 'win32', writable: true });
      Object.defineProperty(process, 'arch', { value: 'x64', writable: true });

      const result = PlatformDetector.detectPlatform();

      expect(result).toEqual({
        nodePlatform: 'win32',
        nodeArch: 'x64',
        os: 'windows',
        arch: 'amd64',
        extension: '.tar.gz'
      });
    });

    test('should throw error for unsupported platform', () => {
      Object.defineProperty(process, 'platform', { value: 'freebsd', writable: true });
      Object.defineProperty(process, 'arch', { value: 'x64', writable: true });

      expect(() => {
        PlatformDetector.detectPlatform();
      }).toThrow('Unsupported platform: freebsd');
    });

    test('should throw error for unsupported architecture', () => {
      Object.defineProperty(process, 'platform', { value: 'linux', writable: true });
      Object.defineProperty(process, 'arch', { value: 'ia32', writable: true });

      expect(() => {
        PlatformDetector.detectPlatform();
      }).toThrow('Unsupported architecture: ia32 on linux');
    });

    test('should throw error for Windows arm64 (unsupported)', () => {
      Object.defineProperty(process, 'platform', { value: 'win32', writable: true });
      Object.defineProperty(process, 'arch', { value: 'arm64', writable: true });

      expect(() => {
        PlatformDetector.detectPlatform();
      }).toThrow('Unsupported architecture: arm64 on win32');
    });
  });

  describe('getBinaryName()', () => {
    test('should generate correct binary name for Linux x64', () => {
      const platform = {
        nodePlatform: 'linux',
        nodeArch: 'x64',
        os: 'linux',
        arch: 'amd64',
        extension: '.tar.gz'
      };

      const result = PlatformDetector.getBinaryName(platform);
      expect(result).toBe('flowspec-cli-linux-amd64.tar.gz');
    });

    test('should generate correct binary name for macOS arm64', () => {
      const platform = {
        nodePlatform: 'darwin',
        nodeArch: 'arm64',
        os: 'darwin',
        arch: 'arm64',
        extension: '.tar.gz'
      };

      const result = PlatformDetector.getBinaryName(platform);
      expect(result).toBe('flowspec-cli-darwin-arm64.tar.gz');
    });

    test('should generate correct binary name for Windows x64', () => {
      const platform = {
        nodePlatform: 'win32',
        nodeArch: 'x64',
        os: 'windows',
        arch: 'amd64',
        extension: '.tar.gz'
      };

      const result = PlatformDetector.getBinaryName(platform);
      expect(result).toBe('flowspec-cli-windows-amd64.tar.gz');
    });

    test('should auto-detect platform when no platform provided', () => {
      Object.defineProperty(process, 'platform', { value: 'linux', writable: true });
      Object.defineProperty(process, 'arch', { value: 'x64', writable: true });

      const result = PlatformDetector.getBinaryName();
      expect(result).toBe('flowspec-cli-linux-amd64.tar.gz');
    });
  });

  describe('getDownloadUrl()', () => {
    test('should construct correct download URL for Linux x64', () => {
      const platform = {
        nodePlatform: 'linux',
        nodeArch: 'x64',
        os: 'linux',
        arch: 'amd64',
        extension: '.tar.gz'
      };

      const result = PlatformDetector.getDownloadUrl('v1.0.0', platform);
      expect(result).toBe('https://github.com/flowspec/flowspec-cli/releases/download/v1.0.0/flowspec-cli-linux-amd64.tar.gz');
    });

    test('should construct correct download URL for macOS arm64', () => {
      const platform = {
        nodePlatform: 'darwin',
        nodeArch: 'arm64',
        os: 'darwin',
        arch: 'arm64',
        extension: '.tar.gz'
      };

      const result = PlatformDetector.getDownloadUrl('v2.1.3', platform);
      expect(result).toBe('https://github.com/flowspec/flowspec-cli/releases/download/v2.1.3/flowspec-cli-darwin-arm64.tar.gz');
    });

    test('should construct correct download URL for Windows x64', () => {
      const platform = {
        nodePlatform: 'win32',
        nodeArch: 'x64',
        os: 'windows',
        arch: 'amd64',
        extension: '.tar.gz'
      };

      const result = PlatformDetector.getDownloadUrl('v0.9.1', platform);
      expect(result).toBe('https://github.com/flowspec/flowspec-cli/releases/download/v0.9.1/flowspec-cli-windows-amd64.tar.gz');
    });

    test('should auto-detect platform when no platform provided', () => {
      Object.defineProperty(process, 'platform', { value: 'darwin', writable: true });
      Object.defineProperty(process, 'arch', { value: 'arm64', writable: true });

      const result = PlatformDetector.getDownloadUrl('v1.5.0');
      expect(result).toBe('https://github.com/flowspec/flowspec-cli/releases/download/v1.5.0/flowspec-cli-darwin-arm64.tar.gz');
    });
  });

  describe('isSupported()', () => {
    test('should return true for supported Linux x64', () => {
      const platform = {
        nodePlatform: 'linux',
        nodeArch: 'x64',
        os: 'linux',
        arch: 'amd64',
        extension: '.tar.gz'
      };

      const result = PlatformDetector.isSupported(platform);
      expect(result).toBe(true);
    });

    test('should return true for supported macOS arm64', () => {
      const platform = {
        nodePlatform: 'darwin',
        nodeArch: 'arm64',
        os: 'darwin',
        arch: 'arm64',
        extension: '.tar.gz'
      };

      const result = PlatformDetector.isSupported(platform);
      expect(result).toBe(true);
    });

    test('should return false for unsupported platform', () => {
      const platform = {
        nodePlatform: 'freebsd',
        nodeArch: 'x64'
      };

      const result = PlatformDetector.isSupported(platform);
      expect(result).toBe(false);
    });

    test('should return false for unsupported architecture', () => {
      const platform = {
        nodePlatform: 'linux',
        nodeArch: 'ia32'
      };

      const result = PlatformDetector.isSupported(platform);
      expect(result).toBe(false);
    });

    test('should auto-detect and return true for current supported platform', () => {
      Object.defineProperty(process, 'platform', { value: 'linux', writable: true });
      Object.defineProperty(process, 'arch', { value: 'x64', writable: true });

      const result = PlatformDetector.isSupported();
      expect(result).toBe(true);
    });

    test('should auto-detect and return false for current unsupported platform', () => {
      Object.defineProperty(process, 'platform', { value: 'freebsd', writable: true });
      Object.defineProperty(process, 'arch', { value: 'x64', writable: true });

      const result = PlatformDetector.isSupported();
      expect(result).toBe(false);
    });
  });

  describe('getSupportedPlatforms()', () => {
    test('should return all supported platforms', () => {
      const result = PlatformDetector.getSupportedPlatforms();

      expect(result).toHaveLength(5); // 2 Linux + 2 macOS + 1 Windows
      expect(result).toContainEqual({
        nodePlatform: 'linux',
        nodeArch: 'x64',
        os: 'linux',
        arch: 'amd64',
        extension: '.tar.gz'
      });
      expect(result).toContainEqual({
        nodePlatform: 'linux',
        nodeArch: 'arm64',
        os: 'linux',
        arch: 'arm64',
        extension: '.tar.gz'
      });
      expect(result).toContainEqual({
        nodePlatform: 'darwin',
        nodeArch: 'x64',
        os: 'darwin',
        arch: 'amd64',
        extension: '.tar.gz'
      });
      expect(result).toContainEqual({
        nodePlatform: 'darwin',
        nodeArch: 'arm64',
        os: 'darwin',
        arch: 'arm64',
        extension: '.tar.gz'
      });
      expect(result).toContainEqual({
        nodePlatform: 'win32',
        nodeArch: 'x64',
        os: 'windows',
        arch: 'amd64',
        extension: '.tar.gz'
      });
    });
  });

  describe('Edge cases and error scenarios', () => {
    test('should handle null platform gracefully in getBinaryName', () => {
      Object.defineProperty(process, 'platform', { value: 'linux', writable: true });
      Object.defineProperty(process, 'arch', { value: 'x64', writable: true });

      const result = PlatformDetector.getBinaryName(null);
      expect(result).toBe('flowspec-cli-linux-amd64.tar.gz');
    });

    test('should handle undefined platform gracefully in getDownloadUrl', () => {
      Object.defineProperty(process, 'platform', { value: 'darwin', writable: true });
      Object.defineProperty(process, 'arch', { value: 'arm64', writable: true });

      const result = PlatformDetector.getDownloadUrl('v1.0.0', undefined);
      expect(result).toBe('https://github.com/flowspec/flowspec-cli/releases/download/v1.0.0/flowspec-cli-darwin-arm64.tar.gz');
    });

    test('should handle empty version string in getDownloadUrl', () => {
      const platform = {
        nodePlatform: 'linux',
        nodeArch: 'x64',
        os: 'linux',
        arch: 'amd64',
        extension: '.tar.gz'
      };

      const result = PlatformDetector.getDownloadUrl('', platform);
      expect(result).toBe('https://github.com/flowspec/flowspec-cli/releases/download//flowspec-cli-linux-amd64.tar.gz');
    });

    test('should handle version without v prefix', () => {
      const platform = {
        nodePlatform: 'linux',
        nodeArch: 'x64',
        os: 'linux',
        arch: 'amd64',
        extension: '.tar.gz'
      };

      const result = PlatformDetector.getDownloadUrl('1.0.0', platform);
      expect(result).toBe('https://github.com/flowspec/flowspec-cli/releases/download/1.0.0/flowspec-cli-linux-amd64.tar.gz');
    });

    test('should handle malformed process.arch values', () => {
      Object.defineProperty(process, 'platform', { value: 'linux', writable: true });
      Object.defineProperty(process, 'arch', { value: 'x86_64', writable: true });

      expect(() => {
        PlatformDetector.detectPlatform();
      }).toThrow('Unsupported architecture: x86_64 on linux');
    });

    test('should handle case sensitivity in platform detection', () => {
      Object.defineProperty(process, 'platform', { value: 'Linux', writable: true });
      Object.defineProperty(process, 'arch', { value: 'X64', writable: true });

      expect(() => {
        PlatformDetector.detectPlatform();
      }).toThrow('Unsupported platform: Linux');
    });
  });
});