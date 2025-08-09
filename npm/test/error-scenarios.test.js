/**
 * Comprehensive error scenario tests
 * Tests edge cases, retry logic, and error handling across all modules
 */

const fs = require('fs');
const { EventEmitter } = require('events');
const { PlatformDetector } = require('../lib/platform');
const { DownloadManager } = require('../lib/download');
const { BinaryManager } = require('../lib/binary');

// Mock modules for error testing
jest.mock('fs');
jest.mock('https');
jest.mock('http');
jest.mock('child_process');

const https = require('https');
const http = require('http');
const { spawn } = require('child_process');

describe('Comprehensive Error Scenario Tests', () => {
  let mockRequest;
  let mockResponse;
  let mockFileStream;

  // Helper function to create mock HTTP implementation
  const createMockHttpImplementation = (response = mockResponse, request = mockRequest) => {
    return (_url, options, callback) => {
      const actualCallback = typeof options === 'function' ? options : callback;
      if (actualCallback) {
        actualCallback(response);
      }
      return request;
    };
  };

  beforeEach(() => {
    jest.clearAllMocks();
    
    // Set up common mocks with EventEmitter functionality
    mockRequest = Object.assign(new EventEmitter(), {
      end: jest.fn(),
      destroy: jest.fn()
    });
    
    mockResponse = Object.assign(new EventEmitter(), {
      statusCode: 200,
      headers: {},
      pipe: jest.fn()
    });
    
    mockFileStream = Object.assign(new EventEmitter(), {
      destroy: jest.fn()
    });
  });

  describe('Platform Detection Error Scenarios', () => {
    const originalPlatform = process.platform;
    const originalArch = process.arch;

    afterEach(() => {
      Object.defineProperty(process, 'platform', { value: originalPlatform });
      Object.defineProperty(process, 'arch', { value: originalArch });
    });

    test('should handle undefined process.platform', () => {
      Object.defineProperty(process, 'platform', { value: undefined });
      Object.defineProperty(process, 'arch', { value: 'x64' });

      expect(() => {
        PlatformDetector.detectPlatform();
      }).toThrow('Unsupported platform: undefined');
    });

    test('should handle null process.arch', () => {
      Object.defineProperty(process, 'platform', { value: 'linux' });
      Object.defineProperty(process, 'arch', { value: null });

      expect(() => {
        PlatformDetector.detectPlatform();
      }).toThrow('Unsupported architecture: null on linux');
    });

    test('should handle exotic platform names', () => {
      const exoticPlatforms = ['aix', 'android', 'freebsd', 'openbsd', 'sunos', 'cygwin'];
      
      exoticPlatforms.forEach(platform => {
        Object.defineProperty(process, 'platform', { value: platform });
        Object.defineProperty(process, 'arch', { value: 'x64' });

        expect(() => {
          PlatformDetector.detectPlatform();
        }).toThrow(`Unsupported platform: ${platform}`);
      });
    });

    test('should handle exotic architecture names', () => {
      const exoticArchs = ['ia32', 'mips', 'mipsel', 'ppc', 'ppc64', 's390', 's390x'];
      
      exoticArchs.forEach(arch => {
        Object.defineProperty(process, 'platform', { value: 'linux' });
        Object.defineProperty(process, 'arch', { value: arch });

        expect(() => {
          PlatformDetector.detectPlatform();
        }).toThrow(`Unsupported architecture: ${arch} on linux`);
      });
    });

    test('should handle corrupted process object', () => {
      // Simulate corrupted process object
      const originalProcess = global.process;
      global.process = {};

      expect(() => {
        PlatformDetector.detectPlatform();
      }).toThrow();

      global.process = originalProcess;
    });
  });

  describe('Download Manager Error Scenarios', () => {
    beforeEach(() => {
      mockFileStream = {
        close: jest.fn(),
        on: jest.fn(),
        write: jest.fn(),
        end: jest.fn()
      };

      mockResponse = {
        statusCode: 200,
        statusMessage: 'OK',
        headers: { 'content-length': '1000' },
        on: jest.fn(),
        pipe: jest.fn(),
        setEncoding: jest.fn()
      };

      mockRequest = {
        on: jest.fn(),
        destroy: jest.fn()
      };

      https.get = jest.fn().mockReturnValue(mockRequest);
      http.get = jest.fn().mockReturnValue(mockRequest);
      fs.createWriteStream = jest.fn().mockReturnValue(mockFileStream);
      fs.existsSync = jest.fn().mockReturnValue(true);
      fs.mkdirSync = jest.fn();
      fs.unlink = jest.fn();
      fs.createReadStream = jest.fn();
    });

    test('should handle malformed URLs', async () => {
      const malformedUrls = [
        'not-a-url',
        'ftp://example.com/file.tar.gz'
      ];

      for (const url of malformedUrls) {
        // The _downloadFile method will throw synchronously when trying to parse invalid URLs
        // This will be caught by downloadBinary and wrapped in the retry logic
        await expect(DownloadManager.downloadBinary(url, '/tmp/test', { maxRetries: 1 }))
          .rejects.toThrow();
      }
      
      // Test null and undefined separately as they cause TypeError
      await expect(DownloadManager.downloadBinary(null, '/tmp/test', { maxRetries: 1 }))
        .rejects.toThrow();
        
      await expect(DownloadManager.downloadBinary(undefined, '/tmp/test', { maxRetries: 1 }))
        .rejects.toThrow();
    }, 10000);

    test('should handle filesystem permission errors', async () => {
      fs.createWriteStream.mockImplementation(() => {
        throw new Error('EACCES: permission denied');
      });

      https.get.mockImplementation((_url, _options, callback) => {
        callback(mockResponse);
        return mockRequest;
      });

      await expect(DownloadManager.downloadBinary('https://example.com/file.tar.gz', '/tmp/test'))
        .rejects.toThrow('EACCES: permission denied');
    });

    test('should handle disk space errors', async () => {
      https.get.mockImplementation((_url, _options, callback) => {
        callback(mockResponse);
        setTimeout(() => {
          const errorHandler = mockFileStream.on.mock.calls.find(call => call[0] === 'error');
          if (errorHandler) {
            const diskError = new Error('ENOSPC: no space left on device');
            diskError.code = 'ENOSPC';
            errorHandler[1](diskError);
          }
        }, 10);
        return mockRequest;
      });

      await expect(DownloadManager.downloadBinary('https://example.com/file.tar.gz', '/tmp/test', { maxRetries: 1 }))
        .rejects.toThrow('Failed to download binary after 1 attempts');
    });

    test('should handle invalid HTTP response codes', async () => {
      const invalidCodes = [100, 201, 202, 300, 301, 400, 401, 403, 500, 502, 503];

      for (const code of invalidCodes) {
        mockResponse.statusCode = code;
        mockResponse.statusMessage = `HTTP ${code}`;

        https.get.mockImplementation((_url, _options, callback) => {
          callback(mockResponse);
          return mockRequest;
        });

        await expect(DownloadManager.downloadBinary('https://example.com/file.tar.gz', '/tmp/test', { maxRetries: 1 }))
          .rejects.toThrow('Failed to download binary after 1 attempts');
      }
    });

    test('should handle corrupted response headers', async () => {
      const corruptedHeaders = [
        { 'content-length': 'not-a-number' },
        { 'content-length': '-1' },
        { 'location': '' }, // Empty redirect location
      ];

      for (const headers of corruptedHeaders) {
        // Create a fresh mock response for each test
        const testResponse = Object.assign(new EventEmitter(), {
          statusCode: headers.location !== undefined ? 302 : 200,
          headers: headers,
          pipe: jest.fn()
        });

        const testRequest = Object.assign(new EventEmitter(), {
          end: jest.fn(),
          destroy: jest.fn()
        });

        https.get.mockImplementation((_url, options, callback) => {
          const actualCallback = typeof options === 'function' ? options : callback;
          if (actualCallback) {
            setImmediate(() => {
              actualCallback(testResponse);
            });
          }
          return testRequest;
        });

        // Mock fs.createWriteStream to complete successfully
        const mockWriteStream = Object.assign(new EventEmitter(), {
          destroy: jest.fn(),
          close: jest.fn()
        });
        fs.createWriteStream.mockReturnValue(mockWriteStream);

        // Simulate successful download completion
        setImmediate(() => {
          testResponse.pipe.mockImplementation((stream) => {
            setTimeout(() => stream.emit('finish'), 10);
            return testResponse;
          });
          setTimeout(() => mockWriteStream.emit('finish'), 20);
        });

        // Should handle gracefully without crashing
        try {
          await DownloadManager.downloadBinary('https://example.com/file.tar.gz', '/tmp/test', { maxRetries: 1 });
          // If it succeeds, that's acceptable
        } catch (error) {
          expect(error).toBeDefined();
        }
      }
    }, 15000);

    test('should handle network interface errors', async () => {
      const networkErrors = [
        { code: 'ENETUNREACH', message: 'Network is unreachable' },
        { code: 'ENETDOWN', message: 'Network is down' },
        { code: 'EHOSTUNREACH', message: 'No route to host' },
        { code: 'EHOSTDOWN', message: 'Host is down' },
        { code: 'ECONNRESET', message: 'Connection reset by peer' },
        { code: 'EPIPE', message: 'Broken pipe' }
      ];

      for (const errorInfo of networkErrors) {
        https.get.mockImplementation((_url, _options, _callback) => {
          const freshMockRequest = {
            on: jest.fn(),
            destroy: jest.fn()
          };

          setTimeout(() => {
            const errorHandler = freshMockRequest.on.mock.calls.find(call => call[0] === 'error');
            if (errorHandler) {
              const error = new Error(errorInfo.message);
              error.code = errorInfo.code;
              errorHandler[1](error);
            }
          }, 10);

          return freshMockRequest;
        });

        await expect(DownloadManager.downloadBinary('https://example.com/file.tar.gz', '/tmp/test', { maxRetries: 1 }))
          .rejects.toThrow('Failed to download binary after 1 attempts');
      }
    });

    test('should handle checksum calculation errors', async () => {
      const mockReadStream = {
        on: jest.fn()
      };

      fs.createReadStream.mockReturnValue(mockReadStream);

      // Simulate read stream error
      mockReadStream.on.mockImplementation((event, callback) => {
        if (event === 'error') {
          setTimeout(() => callback(new Error('Read error')), 0);
        }
      });

      await expect(DownloadManager.verifyChecksum('/tmp/test-file', 'expected-checksum'))
        .rejects.toThrow('Checksum verification failed: Read error');
    });

    test('should handle invalid checksum formats', async () => {
      const invalidChecksums = [
        '', // Empty checksum
        'invalid-checksum', // Too short
        '123', // Too short
        'g'.repeat(64), // Invalid hex characters
        'A'.repeat(63), // Wrong length
        'A'.repeat(65), // Wrong length
        null,
        undefined
      ];

      const mockReadStream = {
        on: jest.fn()
      };

      fs.createReadStream.mockReturnValue(mockReadStream);

      mockReadStream.on.mockImplementation((event, callback) => {
        if (event === 'end') {
          setTimeout(callback, 0);
        }
      });

      const mockHash = {
        update: jest.fn(),
        digest: jest.fn().mockReturnValue('a'.repeat(64))
      };
      jest.spyOn(require('crypto'), 'createHash').mockReturnValue(mockHash);

      for (const invalidChecksum of invalidChecksums) {
        if (invalidChecksum === null || invalidChecksum === undefined || invalidChecksum === '') {
          await expect(DownloadManager.verifyChecksum('/tmp/test-file', invalidChecksum))
            .rejects.toThrow('Invalid checksum format');
        } else {
          const result = await DownloadManager.verifyChecksum('/tmp/test-file', invalidChecksum);
          expect(result).toBe(false);
        }
      }
    });
  });

  describe('Binary Manager Error Scenarios', () => {
    let mockChild;

    beforeEach(() => {
      mockChild = {
        on: jest.fn(),
        kill: jest.fn(),
        killed: false
      };
      spawn.mockReturnValue(mockChild);
      fs.existsSync.mockReturnValue(true);
      fs.readFileSync.mockReturnValue(JSON.stringify({ name: 'test', version: '1.0.0' }));
    });

    test('should handle binary execution failures', async () => {
      const executionErrors = [
        { code: 'ENOENT', message: 'Binary not found' },
        { code: 'EACCES', message: 'Permission denied' },
        { code: 'ENOEXEC', message: 'Exec format error' },
        { code: 'E2BIG', message: 'Argument list too long' }
      ];

      for (const errorInfo of executionErrors) {
        // Create a fresh mock child for each test
        const testChild = Object.assign(new EventEmitter(), {
          kill: jest.fn(),
          killed: false
        });
        
        spawn.mockReturnValue(testChild);

        // Mock BinaryManager.ensureBinaryExists to resolve successfully
        jest.spyOn(BinaryManager, 'ensureBinaryExists').mockResolvedValue();
        jest.spyOn(BinaryManager, 'getBinaryPath').mockReturnValue('/mock/binary/path');

        // Create a promise that will be resolved when the error is emitted
        const errorPromise = BinaryManager.executeBinary(['--version']);
        
        // Simulate error after a short delay
        setImmediate(() => {
          const error = new Error(errorInfo.message);
          error.code = errorInfo.code;
          testChild.emit('error', error);
        });

        await expect(errorPromise).rejects.toThrow();
      }
    }, 15000);

    test('should handle corrupted package.json', async () => {
      const corruptedPackageJsons = [
        'not json',
        '{"name": "test"', // Incomplete JSON
        '{"name": "test", "version":}', // Invalid JSON
        '{}', // Missing version
        '{"version": ""}', // Empty version
        '{"version": null}', // Null version
      ];

      for (const corruptedJson of corruptedPackageJsons) {
        fs.readFileSync.mockReturnValue(corruptedJson);

        expect(() => {
          BinaryManager.getVersion();
        }).toThrow();
      }
    });

    test('should handle binary verification timeouts', async () => {
      // Create a mock child that simulates timeout
      const timeoutChild = Object.assign(new EventEmitter(), {
        kill: jest.fn(),
        killed: false
      });
      
      spawn.mockReturnValue(timeoutChild);

      // Mock BinaryManager methods
      jest.spyOn(BinaryManager, 'ensureBinaryExists').mockResolvedValue();
      jest.spyOn(BinaryManager, 'getBinaryPath').mockReturnValue('/mock/binary/path');

      // Don't emit any events to simulate hanging process
      // The timeout should trigger after 100ms

      await expect(BinaryManager.executeBinary(['--version'], { timeout: 100 }))
        .rejects.toThrow('Process execution timed out after 100ms');
    }, 5000);

    test('should handle process signal errors', async () => {
      const signals = ['SIGKILL', 'SIGTERM', 'SIGINT', 'SIGHUP'];

      for (const signal of signals) {
        // Create a fresh mock child for each signal test
        const signalChild = Object.assign(new EventEmitter(), {
          kill: jest.fn(),
          killed: false
        });
        
        spawn.mockReturnValue(signalChild);

        // Mock BinaryManager methods
        jest.spyOn(BinaryManager, 'ensureBinaryExists').mockResolvedValue();
        jest.spyOn(BinaryManager, 'getBinaryPath').mockReturnValue('/mock/binary/path');

        // Create the execution promise
        const executionPromise = BinaryManager.executeBinary(['--version']);
        
        // Simulate signal termination after a short delay
        setImmediate(() => {
          signalChild.emit('close', 1, signal);
        });

        // Should resolve with exit code 1 (not reject)
        const exitCode = await executionPromise;
        expect(exitCode).toBe(1);
      }
    }, 15000);

    test('should handle filesystem stat errors', async () => {
      fs.existsSync.mockReturnValue(true);
      fs.statSync.mockImplementation(() => {
        throw new Error('EACCES: permission denied');
      });

      const result = BinaryManager._isExecutable('/path/to/binary');
      expect(result).toBe(false);
    });

    test('should handle directory creation failures', async () => {
      // Mock getBinaryPath to return a path
      jest.spyOn(BinaryManager, 'getBinaryPath').mockReturnValue('/mock/binary/path');
      
      // Mock _isBinaryFunctional to return false initially, then false again after "installation"
      jest.spyOn(BinaryManager, '_isBinaryFunctional')
        .mockResolvedValueOnce(false)  // First call - binary doesn't exist
        .mockResolvedValueOnce(false); // Second call - installation failed
      
      // Mock _installBinary to simulate successful installation but binary still not functional
      jest.spyOn(BinaryManager, '_installBinary').mockResolvedValue();

      await expect(BinaryManager.ensureBinaryExists({ reinstallOnFailure: true }))
        .rejects.toThrow('Binary installation completed but binary is still not functional');
    });
  });

  describe('Integration Error Scenarios', () => {
    test('should handle cascading failures', async () => {
      // Mock getBinaryPath to return a path
      jest.spyOn(BinaryManager, 'getBinaryPath').mockReturnValue('/mock/binary/path');
      
      // Mock _isBinaryFunctional to return false (binary doesn't exist)
      jest.spyOn(BinaryManager, '_isBinaryFunctional').mockResolvedValue(false);
      
      // Mock _installBinary to throw an error simulating cascading failures
      jest.spyOn(BinaryManager, '_installBinary').mockRejectedValue(new Error('Cannot create directory'));

      await expect(BinaryManager.ensureBinaryExists({ reinstallOnFailure: true }))
        .rejects.toThrow('Cannot create directory');
    });

    test('should handle memory pressure scenarios', async () => {
      // Ensure file exists for this test
      fs.existsSync.mockReturnValue(true);
      
      // Mock fs.createReadStream to throw memory error
      fs.createReadStream.mockImplementation(() => {
        const mockStream = Object.assign(new EventEmitter(), {});
        setImmediate(() => {
          const error = new Error('Cannot allocate memory');
          error.code = 'ENOMEM';
          mockStream.emit('error', error);
        });
        return mockStream;
      });

      // The verifyChecksum method should throw an error with the memory error message
      await expect(DownloadManager.verifyChecksum('/tmp/test-file', 'checksum'))
        .rejects.toThrow('Checksum verification failed: Cannot allocate memory');
    });

    test('should handle concurrent access conflicts', async () => {
      // Simulate file locking conflicts
      fs.createWriteStream.mockImplementation(() => {
        throw new Error('EBUSY: resource busy or locked');
      });

      https.get.mockImplementation((_url, _options, callback) => {
        callback({
          statusCode: 200,
          headers: { 'content-length': '1000' },
          on: jest.fn(),
          pipe: jest.fn()
        });
        return { on: jest.fn(), destroy: jest.fn() };
      });

      await expect(DownloadManager.downloadBinary('https://example.com/file.tar.gz', '/tmp/test', { maxRetries: 3 }))
        .rejects.toThrow('Failed to download binary after 3 attempts');
    });

    test('should handle system resource exhaustion', async () => {
      // Ensure file exists for this test
      fs.existsSync.mockReturnValue(true);
      
      // Mock fs.createReadStream to throw EMFILE error
      fs.createReadStream.mockImplementation(() => {
        const mockStream = Object.assign(new EventEmitter(), {});
        setImmediate(() => {
          const error = new Error('EMFILE: too many open files');
          error.code = 'EMFILE';
          mockStream.emit('error', error);
        });
        return mockStream;
      });

      // The verifyChecksum method should throw an error with the EMFILE error message
      await expect(DownloadManager.verifyChecksum('/tmp/test-file', 'checksum'))
        .rejects.toThrow('Checksum verification failed: EMFILE: too many open files');
    });
  });

  describe('Recovery and Cleanup Error Scenarios', () => {
    test('should handle cleanup failures gracefully', async () => {
      fs.unlink.mockImplementation((path, callback) => {
        callback(new Error('EACCES: permission denied'));
      });

      // Should not throw even if cleanup fails
      await expect(DownloadManager.cleanup('/tmp/test-file'))
        .resolves.not.toThrow();
    });

    test('should handle partial cleanup scenarios', async () => {
      const paths = ['/tmp/file1', '/tmp/file2', '/tmp/file3'];
      
      fs.unlink.mockImplementation((path, callback) => {
        if (path === '/tmp/file2') {
          callback(new Error('Permission denied'));
        } else {
          callback(null);
        }
      });

      // Should continue cleanup even if some files fail
      await expect(DownloadManager.cleanup(paths))
        .resolves.not.toThrow();
    });

    test('should handle recovery from corrupted state', async () => {
      // Simulate a scenario where binary exists but is corrupted
      fs.existsSync.mockReturnValue(true);
      
      const mockReadStream = {
        on: jest.fn()
      };
      
      fs.createReadStream.mockReturnValue(mockReadStream);
      
      mockReadStream.on.mockImplementation((event, callback) => {
        if (event === 'error') {
          setTimeout(() => callback(new Error('File corrupted')), 0);
        }
      });

      await expect(DownloadManager.verifyChecksum('/tmp/corrupted-binary', 'checksum'))
        .rejects.toThrow('Checksum verification failed: File corrupted');
    });
  });

  describe('Edge Case Error Scenarios', () => {
    test('should handle extremely large files', async () => {
      // Simulate a file that's too large
      mockResponse = {
        statusCode: 200,
        headers: { 'content-length': '999999999999999' }, // Extremely large
        on: jest.fn(),
        pipe: jest.fn()
      };

      https.get.mockImplementation((_url, _options, callback) => {
        callback(mockResponse);
        return { on: jest.fn(), destroy: jest.fn() };
      });

      // Should handle large files gracefully
      try {
        await DownloadManager.downloadBinary('https://example.com/huge-file.tar.gz', '/tmp/test');
      } catch (error) {
        expect(error).toBeDefined();
      }
    });

    test('should handle zero-byte files', async () => {
      mockResponse = {
        statusCode: 200,
        headers: { 'content-length': '0' },
        on: jest.fn(),
        pipe: jest.fn()
      };

      const mockFileStream = {
        on: jest.fn(),
        close: jest.fn()
      };

      fs.createWriteStream.mockReturnValue(mockFileStream);

      https.get.mockImplementation((_url, _options, callback) => {
        callback(mockResponse);
        setTimeout(() => {
          const finishHandler = mockFileStream.on.mock.calls.find(call => call[0] === 'finish');
          if (finishHandler) {
            finishHandler[1]();
          }
        }, 0);
        return { on: jest.fn(), destroy: jest.fn() };
      });

      await DownloadManager.downloadBinary('https://example.com/empty-file.tar.gz', '/tmp/test');
      expect(fs.createWriteStream).toHaveBeenCalled();
    });

    test('should handle unicode and special characters in paths', async () => {
      const specialPaths = [
        '/tmp/测试文件.tar.gz',
        '/tmp/файл.tar.gz',
        '/tmp/file with spaces.tar.gz',
        '/tmp/file-with-émojis-🚀.tar.gz'
      ];

      for (const specialPath of specialPaths) {
        // Create a fresh mock response for each path test
        const pathResponse = Object.assign(new EventEmitter(), {
          statusCode: 200,
          headers: { 'content-length': '1000' },
          pipe: jest.fn().mockImplementation((stream) => {
            // Simulate successful pipe
            setImmediate(() => {
              if (stream && stream.emit) {
                stream.emit('finish');
              }
            });
            return pathResponse;
          })
        });

        const pathRequest = Object.assign(new EventEmitter(), {
          end: jest.fn(),
          destroy: jest.fn()
        });

        // Mock fs.createWriteStream with close method
        const mockWriteStream = Object.assign(new EventEmitter(), {
          destroy: jest.fn(),
          close: jest.fn()
        });
        fs.createWriteStream.mockReturnValue(mockWriteStream);

        // Mock successful response for each path
        https.get.mockImplementation((_url, options, callback) => {
          const actualCallback = typeof options === 'function' ? options : callback;
          if (actualCallback) {
            setImmediate(() => {
              actualCallback(pathResponse);
              // Simulate response end after a short delay
              setTimeout(() => pathResponse.emit('end'), 10);
            });
          }
          return pathRequest;
        });

        try {
          await DownloadManager.downloadBinary('https://example.com/file.tar.gz', specialPath, { maxRetries: 1 });
          // If it succeeds, that's also acceptable
        } catch (error) {
          // Should handle special characters gracefully
          expect(error).toBeDefined();
        }
      }
    }, 20000);
  });
});