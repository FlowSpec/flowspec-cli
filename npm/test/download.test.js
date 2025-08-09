/**
 * Unit tests for download and checksum verification functionality
 */

const fs = require('fs');
const path = require('path');
const crypto = require('crypto');
const { DownloadManager } = require('../lib/download');

// Mock modules
jest.mock('https');
jest.mock('http');
jest.mock('fs');
jest.mock('child_process');

const https = require('https');
const http = require('http');
const { spawn } = require('child_process');

describe('DownloadManager', () => {
  let mockRequest;
  let mockResponse;
  let mockFileStream;

  beforeEach(() => {
    jest.clearAllMocks();
    
    // Mock file stream
    mockFileStream = {
      close: jest.fn(),
      on: jest.fn(),
      write: jest.fn(),
      end: jest.fn()
    };

    // Mock response
    mockResponse = {
      statusCode: 200,
      statusMessage: 'OK',
      headers: { 'content-length': '1000' },
      on: jest.fn(),
      pipe: jest.fn(),
      setEncoding: jest.fn()
    };

    // Mock request
    mockRequest = {
      on: jest.fn(),
      destroy: jest.fn()
    };

    // Setup default mocks
    https.get = jest.fn().mockReturnValue(mockRequest);
    http.get = jest.fn().mockReturnValue(mockRequest);
    fs.createWriteStream = jest.fn().mockReturnValue(mockFileStream);
    fs.existsSync = jest.fn().mockReturnValue(true);
    fs.mkdirSync = jest.fn();
    fs.unlink = jest.fn();
    fs.createReadStream = jest.fn();
  });

  describe('downloadBinary', () => {
    it('should successfully download a binary file', async () => {
      // Setup successful download
      https.get.mockImplementation((url, options, callback) => {
        callback(mockResponse);
        return mockRequest;
      });

      mockFileStream.on.mockImplementation((event, callback) => {
        if (event === 'finish') {
          setTimeout(callback, 0);
        }
      });

      const targetPath = '/tmp/test-binary';
      const url = 'https://example.com/binary.tar.gz';

      await DownloadManager.downloadBinary(url, targetPath);

      expect(https.get).toHaveBeenCalledWith(
        url,
        expect.objectContaining({
          timeout: 30000,
          headers: expect.objectContaining({
            'User-Agent': 'flowspec-cli-npm-wrapper/1.0.0'
          })
        }),
        expect.any(Function)
      );
      expect(fs.createWriteStream).toHaveBeenCalledWith(targetPath);
      expect(mockResponse.pipe).toHaveBeenCalledWith(mockFileStream);
    });

    it('should handle HTTP errors', async () => {
      mockResponse.statusCode = 404;
      mockResponse.statusMessage = 'Not Found';

      https.get.mockImplementation((url, options, callback) => {
        callback(mockResponse);
        return mockRequest;
      });

      const targetPath = '/tmp/test-binary';
      const url = 'https://example.com/binary.tar.gz';

      await expect(DownloadManager.downloadBinary(url, targetPath))
        .rejects.toThrow('Failed to download binary after 3 attempts');
    });

    it('should handle network errors with retry logic', async () => {
      let attemptCount = 0;
      
      https.get.mockImplementation((url, options, callback) => {
        attemptCount++;
        
        // Setup fresh mock request for each attempt
        const freshMockRequest = {
          on: jest.fn(),
          destroy: jest.fn()
        };
        
        if (attemptCount < 3) {
          // Simulate network error for first 2 attempts
          setTimeout(() => {
            const errorHandler = freshMockRequest.on.mock.calls.find(call => call[0] === 'error');
            if (errorHandler) {
              errorHandler[1](new Error('Network error'));
            }
          }, 10);
        } else {
          // Success on third attempt
          callback(mockResponse);
          setTimeout(() => {
            const finishHandler = mockFileStream.on.mock.calls.find(call => call[0] === 'finish');
            if (finishHandler) {
              finishHandler[1]();
            }
          }, 10);
        }
        
        return freshMockRequest;
      });

      const targetPath = '/tmp/test-binary';
      const url = 'https://example.com/binary.tar.gz';

      await DownloadManager.downloadBinary(url, targetPath);

      expect(https.get).toHaveBeenCalledTimes(3);
    });

    it('should handle redirects', async () => {
      const redirectResponse = {
        statusCode: 302,
        headers: { location: 'https://redirect.example.com/binary.tar.gz' }
      };

      let callCount = 0;
      https.get.mockImplementation((url, options, callback) => {
        callCount++;
        if (callCount === 1) {
          callback(redirectResponse);
        } else {
          callback(mockResponse);
          setTimeout(() => {
            mockFileStream.on.mock.calls.find(call => call[0] === 'finish')[1]();
          }, 0);
        }
        return mockRequest;
      });

      const targetPath = '/tmp/test-binary';
      const url = 'https://example.com/binary.tar.gz';

      await DownloadManager.downloadBinary(url, targetPath);

      expect(https.get).toHaveBeenCalledTimes(2);
      expect(https.get).toHaveBeenLastCalledWith(
        'https://redirect.example.com/binary.tar.gz',
        expect.any(Object),
        expect.any(Function)
      );
    });

    it('should call progress callback when provided', async () => {
      const progressCallback = jest.fn();
      
      https.get.mockImplementation((url, options, callback) => {
        callback(mockResponse);
        
        // Simulate data chunks
        setTimeout(() => {
          const dataCallback = mockResponse.on.mock.calls.find(call => call[0] === 'data')[1];
          dataCallback(Buffer.alloc(500)); // First chunk
          dataCallback(Buffer.alloc(500)); // Second chunk
          
          // Finish download
          mockFileStream.on.mock.calls.find(call => call[0] === 'finish')[1]();
        }, 0);
        
        return mockRequest;
      });

      const targetPath = '/tmp/test-binary';
      const url = 'https://example.com/binary.tar.gz';

      await DownloadManager.downloadBinary(url, targetPath, {
        onProgress: progressCallback
      });

      expect(progressCallback).toHaveBeenCalledWith(50, 500, 1000);
      expect(progressCallback).toHaveBeenCalledWith(100, 1000, 1000);
    });

    it('should create target directory if it does not exist', async () => {
      fs.existsSync.mockReturnValue(false);
      
      https.get.mockImplementation((url, options, callback) => {
        callback(mockResponse);
        setTimeout(() => {
          mockFileStream.on.mock.calls.find(call => call[0] === 'finish')[1]();
        }, 0);
        return mockRequest;
      });

      const targetPath = '/tmp/new-dir/test-binary';
      const url = 'https://example.com/binary.tar.gz';

      await DownloadManager.downloadBinary(url, targetPath);

      expect(fs.mkdirSync).toHaveBeenCalledWith('/tmp/new-dir', { recursive: true });
    });
  });

  describe('downloadChecksums', () => {
    it('should successfully download and parse checksums', async () => {
      const checksumsContent = `# Checksums for flowspec-cli v1.0.0
abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890  flowspec-cli-linux-amd64.tar.gz
1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef  flowspec-cli-darwin-amd64.tar.gz`;

      https.get.mockImplementation((url, options, callback) => {
        callback({
          statusCode: 200,
          setEncoding: jest.fn(),
          on: jest.fn().mockImplementation((event, cb) => {
            if (event === 'data') {
              setTimeout(() => cb(checksumsContent), 0);
            } else if (event === 'end') {
              setTimeout(cb, 0);
            }
          })
        });
        return mockRequest;
      });

      const checksums = await DownloadManager.downloadChecksums('v1.0.0');

      expect(checksums).toBeInstanceOf(Map);
      expect(checksums.size).toBe(2);
      expect(checksums.get('flowspec-cli-linux-amd64.tar.gz'))
        .toBe('abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890');
      expect(checksums.get('flowspec-cli-darwin-amd64.tar.gz'))
        .toBe('1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef');
    });

    it('should handle HTTP errors when downloading checksums', async () => {
      https.get.mockImplementation((url, options, callback) => {
        callback({
          statusCode: 404,
          statusMessage: 'Not Found'
        });
        return mockRequest;
      });

      await expect(DownloadManager.downloadChecksums('v1.0.0'))
        .rejects.toThrow('Failed to download checksums: HTTP 404: Not Found');
    });

    it('should handle empty checksums file', async () => {
      https.get.mockImplementation((url, options, callback) => {
        callback({
          statusCode: 200,
          setEncoding: jest.fn(),
          on: jest.fn().mockImplementation((event, cb) => {
            if (event === 'data') {
              setTimeout(() => cb('# Only comments\n\n'), 0);
            } else if (event === 'end') {
              setTimeout(cb, 0);
            }
          })
        });
        return mockRequest;
      });

      await expect(DownloadManager.downloadChecksums('v1.0.0'))
        .rejects.toThrow('No valid checksums found in checksums.txt');
    });

    it('should skip invalid checksum lines', async () => {
      const checksumsContent = `# Valid checksums
abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890  flowspec-cli-linux-amd64.tar.gz
invalid-line-without-proper-format
1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef  flowspec-cli-darwin-amd64.tar.gz`;

      https.get.mockImplementation((url, options, callback) => {
        callback({
          statusCode: 200,
          setEncoding: jest.fn(),
          on: jest.fn().mockImplementation((event, cb) => {
            if (event === 'data') {
              setTimeout(() => cb(checksumsContent), 0);
            } else if (event === 'end') {
              setTimeout(cb, 0);
            }
          })
        });
        return mockRequest;
      });

      const checksums = await DownloadManager.downloadChecksums('v1.0.0');

      expect(checksums.size).toBe(2);
      expect(checksums.has('flowspec-cli-linux-amd64.tar.gz')).toBe(true);
      expect(checksums.has('flowspec-cli-darwin-amd64.tar.gz')).toBe(true);
    });
  });

  describe('verifyChecksum', () => {
    it('should successfully verify matching checksum', async () => {
      const testFilePath = '/tmp/test-file';
      const expectedChecksum = 'abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890';
      
      // Mock file stream for checksum calculation
      const mockReadStream = {
        on: jest.fn()
      };
      
      fs.createReadStream.mockReturnValue(mockReadStream);
      
      // Simulate successful checksum calculation
      mockReadStream.on.mockImplementation((event, callback) => {
        if (event === 'data') {
          // Simulate reading file data
          const hash = crypto.createHash('sha256');
          hash.update('test file content');
          setTimeout(() => callback(Buffer.from('test file content')), 0);
        } else if (event === 'end') {
          setTimeout(callback, 0);
        }
      });

      // Mock crypto.createHash to return expected checksum
      const mockHash = {
        update: jest.fn(),
        digest: jest.fn().mockReturnValue(expectedChecksum)
      };
      jest.spyOn(crypto, 'createHash').mockReturnValue(mockHash);

      const result = await DownloadManager.verifyChecksum(testFilePath, expectedChecksum);

      expect(result).toBe(true);
      expect(fs.existsSync).toHaveBeenCalledWith(testFilePath);
      expect(fs.createReadStream).toHaveBeenCalledWith(testFilePath);
    });

    it('should fail verification for mismatched checksum', async () => {
      const testFilePath = '/tmp/test-file';
      const expectedChecksum = 'abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890';
      const actualChecksum = '1111111111111111111111111111111111111111111111111111111111111111';
      
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
        digest: jest.fn().mockReturnValue(actualChecksum)
      };
      jest.spyOn(crypto, 'createHash').mockReturnValue(mockHash);

      const result = await DownloadManager.verifyChecksum(testFilePath, expectedChecksum);

      expect(result).toBe(false);
    });

    it('should handle file not found error', async () => {
      fs.existsSync.mockReturnValue(false);
      
      const testFilePath = '/tmp/nonexistent-file';
      const expectedChecksum = 'abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890';

      await expect(DownloadManager.verifyChecksum(testFilePath, expectedChecksum))
        .rejects.toThrow('Checksum verification failed: File not found: /tmp/nonexistent-file');
    });

    it('should handle file read errors', async () => {
      const testFilePath = '/tmp/test-file';
      const expectedChecksum = 'abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890';
      
      const mockReadStream = {
        on: jest.fn()
      };
      
      fs.createReadStream.mockReturnValue(mockReadStream);
      
      mockReadStream.on.mockImplementation((event, callback) => {
        if (event === 'error') {
          setTimeout(() => callback(new Error('File read error')), 0);
        }
      });

      await expect(DownloadManager.verifyChecksum(testFilePath, expectedChecksum))
        .rejects.toThrow('Checksum verification failed: File read error');
    });

    it('should normalize checksums for comparison', async () => {
      const testFilePath = '/tmp/test-file';
      const expectedChecksum = 'ABCDEF1234567890ABCDEF1234567890ABCDEF1234567890ABCDEF1234567890'; // Uppercase
      const actualChecksum = 'abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890'; // Lowercase
      
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
        digest: jest.fn().mockReturnValue(actualChecksum)
      };
      jest.spyOn(crypto, 'createHash').mockReturnValue(mockHash);

      const result = await DownloadManager.verifyChecksum(testFilePath, expectedChecksum);

      expect(result).toBe(true);
    });
  });

  describe('_parseChecksums', () => {
    it('should parse valid checksums format', () => {
      const content = `# Checksums for flowspec-cli v1.0.0
abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890  flowspec-cli-linux-amd64.tar.gz
1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef  flowspec-cli-darwin-amd64.tar.gz`;

      const checksums = DownloadManager._parseChecksums(content);

      expect(checksums.size).toBe(2);
      expect(checksums.get('flowspec-cli-linux-amd64.tar.gz'))
        .toBe('abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890');
    });

    it('should skip comments and empty lines', () => {
      const content = `# This is a comment

abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890  flowspec-cli-linux-amd64.tar.gz
# Another comment
1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef  flowspec-cli-darwin-amd64.tar.gz

`;

      const checksums = DownloadManager._parseChecksums(content);

      expect(checksums.size).toBe(2);
    });

    it('should throw error for empty checksums', () => {
      const content = `# Only comments
# No actual checksums`;

      expect(() => DownloadManager._parseChecksums(content))
        .toThrow('No valid checksums found in checksums.txt');
    });
  });

  describe('_calculateSHA256', () => {
    it('should calculate SHA256 hash correctly', async () => {
      const testFilePath = '/tmp/test-file';
      const expectedHash = 'abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890';
      
      const mockReadStream = {
        on: jest.fn()
      };
      
      fs.createReadStream.mockReturnValue(mockReadStream);
      
      // Mock the hash calculation
      const mockHash = {
        update: jest.fn(),
        digest: jest.fn().mockReturnValue(expectedHash)
      };
      jest.spyOn(crypto, 'createHash').mockReturnValue(mockHash);
      
      mockReadStream.on.mockImplementation((event, callback) => {
        if (event === 'data') {
          setTimeout(() => {
            callback(Buffer.from('test content'));
          }, 0);
        } else if (event === 'end') {
          setTimeout(callback, 0);
        }
      });

      const result = await DownloadManager._calculateSHA256(testFilePath);

      expect(result).toBe(expectedHash);
      expect(crypto.createHash).toHaveBeenCalledWith('sha256');
      expect(fs.createReadStream).toHaveBeenCalledWith(testFilePath);
    });
  });

  describe('error scenarios and retry logic', () => {
    it('should handle timeout errors', async () => {
      https.get.mockImplementation((url, options, callback) => {
        const freshMockRequest = {
          on: jest.fn(),
          destroy: jest.fn()
        };
        
        setTimeout(() => {
          const timeoutHandler = freshMockRequest.on.mock.calls.find(call => call[0] === 'timeout');
          if (timeoutHandler) {
            timeoutHandler[1]();
          }
        }, 10);
        
        return freshMockRequest;
      });

      const targetPath = '/tmp/test-binary';
      const url = 'https://example.com/binary.tar.gz';

      await expect(DownloadManager.downloadBinary(url, targetPath))
        .rejects.toThrow('Failed to download binary after 3 attempts');
    });

    it('should clean up partial files on error', async () => {
      https.get.mockImplementation((url, options, callback) => {
        callback(mockResponse);
        setTimeout(() => {
          const errorHandler = mockFileStream.on.mock.calls.find(call => call[0] === 'error');
          if (errorHandler) {
            errorHandler[1](new Error('Write error'));
          }
        }, 10);
        return mockRequest;
      });

      const targetPath = '/tmp/test-binary';
      const url = 'https://example.com/binary.tar.gz';

      await expect(DownloadManager.downloadBinary(url, targetPath, { retryDelay: 10, maxRetries: 1 }))
        .rejects.toThrow('Failed to download binary after 1 attempts');

      expect(fs.unlink).toHaveBeenCalledWith(targetPath, expect.any(Function));
    });

    it('should implement exponential backoff for retries', async () => {
      let attemptCount = 0;
      
      https.get.mockImplementation((url, options, callback) => {
        attemptCount++;
        const freshMockRequest = {
          on: jest.fn(),
          destroy: jest.fn()
        };
        
        setTimeout(() => {
          const errorHandler = freshMockRequest.on.mock.calls.find(call => call[0] === 'error');
          if (errorHandler) {
            errorHandler[1](new Error('Network error'));
          }
        }, 10);
        
        return freshMockRequest;
      });

      const targetPath = '/tmp/test-binary';
      const url = 'https://example.com/binary.tar.gz';

      await expect(DownloadManager.downloadBinary(url, targetPath, { 
        retryDelay: 10, 
        maxRetries: 2 
      })).rejects.toThrow('Failed to download binary after 2 attempts');

      expect(attemptCount).toBe(2);
    });

    it('should handle DNS resolution errors', async () => {
      https.get.mockImplementation((url, options, callback) => {
        const freshMockRequest = {
          on: jest.fn(),
          destroy: jest.fn()
        };
        
        setTimeout(() => {
          const errorHandler = freshMockRequest.on.mock.calls.find(call => call[0] === 'error');
          if (errorHandler) {
            const dnsError = new Error('getaddrinfo ENOTFOUND');
            dnsError.code = 'ENOTFOUND';
            errorHandler[1](dnsError);
          }
        }, 10);
        
        return freshMockRequest;
      });

      const targetPath = '/tmp/test-binary';
      const url = 'https://nonexistent.example.com/binary.tar.gz';

      await expect(DownloadManager.downloadBinary(url, targetPath, { maxRetries: 1 }))
        .rejects.toThrow('Failed to download binary after 1 attempts');
    });

    it('should handle connection refused errors', async () => {
      https.get.mockImplementation((url, options, callback) => {
        const freshMockRequest = {
          on: jest.fn(),
          destroy: jest.fn()
        };
        
        setTimeout(() => {
          const errorHandler = freshMockRequest.on.mock.calls.find(call => call[0] === 'error');
          if (errorHandler) {
            const connError = new Error('connect ECONNREFUSED');
            connError.code = 'ECONNREFUSED';
            errorHandler[1](connError);
          }
        }, 10);
        
        return freshMockRequest;
      });

      const targetPath = '/tmp/test-binary';
      const url = 'https://example.com/binary.tar.gz';

      await expect(DownloadManager.downloadBinary(url, targetPath, { maxRetries: 1 }))
        .rejects.toThrow('Failed to download binary after 1 attempts');
    });

    it('should handle SSL/TLS certificate errors', async () => {
      https.get.mockImplementation((url, options, callback) => {
        const freshMockRequest = {
          on: jest.fn(),
          destroy: jest.fn()
        };
        
        setTimeout(() => {
          const errorHandler = freshMockRequest.on.mock.calls.find(call => call[0] === 'error');
          if (errorHandler) {
            const sslError = new Error('certificate verify failed');
            sslError.code = 'CERT_UNTRUSTED';
            errorHandler[1](sslError);
          }
        }, 10);
        
        return freshMockRequest;
      });

      const targetPath = '/tmp/test-binary';
      const url = 'https://untrusted.example.com/binary.tar.gz';

      await expect(DownloadManager.downloadBinary(url, targetPath, { maxRetries: 1 }))
        .rejects.toThrow('Failed to download binary after 1 attempts');
    });

    it('should handle maximum redirect limit', async () => {
      let redirectCount = 0;
      
      https.get.mockImplementation((url, options, callback) => {
        redirectCount++;
        
        // Always return redirect to simulate infinite loop
        callback({
          statusCode: 302,
          headers: { location: `https://redirect${redirectCount}.example.com/binary.tar.gz` }
        });
        
        return mockRequest;
      });

      const targetPath = '/tmp/test-binary';
      const url = 'https://example.com/binary.tar.gz';

      await expect(DownloadManager.downloadBinary(url, targetPath, { maxRetries: 1 }))
        .rejects.toThrow('Failed to download binary after 1 attempts');
    });

    it('should handle corrupted download streams', async () => {
      https.get.mockImplementation((url, options, callback) => {
        const freshMockRequest = {
          on: jest.fn(),
          destroy: jest.fn()
        };
        
        setTimeout(() => {
          const errorHandler = freshMockRequest.on.mock.calls.find(call => call[0] === 'error');
          if (errorHandler) {
            errorHandler[1](new Error('Unexpected end of stream'));
          }
        }, 10);
        
        return freshMockRequest;
      });

      const targetPath = '/tmp/test-binary';
      const url = 'https://example.com/binary.tar.gz';

      await expect(DownloadManager.downloadBinary(url, targetPath, { maxRetries: 1 }))
        .rejects.toThrow('Failed to download binary after 1 attempts');
    }, 3000);
  });
});

describe('Additional Download Manager Coverage', () => {
    let mockResponse;
    let mockRequest;
    
    beforeEach(() => {
      mockResponse = {
        statusCode: 200,
        headers: {},
        on: jest.fn(),
        pipe: jest.fn(),
        emit: jest.fn()
      };
      
      mockRequest = {
        on: jest.fn(),
        destroy: jest.fn()
      };
    });
    
    test('should handle downloadChecksums with network error', async () => {
      https.get.mockImplementation((url, options, callback) => {
        const error = new Error('Network error');
        callback(error);
        return mockRequest;
      });
      
      await expect(DownloadManager.downloadChecksums('v1.0.0'))
        .rejects.toThrow('Failed to download checksums');
    });
    
    test('should handle downloadChecksums with HTTP error status', async () => {
      mockResponse.statusCode = 404;
      
      https.get.mockImplementation((url, options, callback) => {
        callback(mockResponse);
        return mockRequest;
      });
      
      await expect(DownloadManager.downloadChecksums('v1.0.0'))
        .rejects.toThrow('Failed to download checksums: HTTP 404');
    });
    
    test('should handle extractTarGz with extraction error', async () => {
      // Mock tar module directly
      const tar = require('tar');
      tar.extract = jest.fn().mockRejectedValue(new Error('Extraction failed'));
      
      await expect(DownloadManager.extractTarGz('/path/to/archive.tar.gz', '/path/to/dest'))
        .rejects.toThrow('Failed to extract archive');
    }, 5000);
    
    test('should handle setBinaryPermissions on Windows', async () => {
      // Mock Windows platform
      Object.defineProperty(process, 'platform', {
        value: 'win32',
        configurable: true
      });
      
      // Should not throw on Windows
      await expect(DownloadManager.setBinaryPermissions('/path/to/binary'))
        .resolves.not.toThrow();
    });
    
    test('should handle setBinaryPermissions with chmod error', async () => {
      // Mock Unix platform
      Object.defineProperty(process, 'platform', {
        value: 'linux',
        configurable: true
      });
      
      fs.chmodSync.mockImplementation(() => {
        throw new Error('Chmod failed');
      });
      
      await expect(DownloadManager.setBinaryPermissions('/path/to/binary'))
        .rejects.toThrow('Failed to set binary permissions');
    });
    
    test('should handle verifyBinary with execution error', async () => {
      const mockChild = {
        on: jest.fn(),
        kill: jest.fn(),
        killed: false
      };
      
      spawn.mockReturnValue(mockChild);
      
      mockChild.on.mockImplementation((event, callback) => {
        if (event === 'error') {
          setImmediate(() => callback(new Error('Execution failed')));
        }
      });
      
      const result = await DownloadManager.verifyBinary('/path/to/binary');
      expect(result).toBe(false);
    });
    
    test('should handle verifyBinary with non-zero exit code', async () => {
      const mockChild = {
        on: jest.fn(),
        kill: jest.fn(),
        killed: false
      };
      
      spawn.mockReturnValue(mockChild);
      
      mockChild.on.mockImplementation((event, callback) => {
        if (event === 'close') {
          setImmediate(() => callback(1, null));
        }
      });
      
      const result = await DownloadManager.verifyBinary('/path/to/binary');
      expect(result).toBe(false);
    });
    
    test('should handle verifyBinary with timeout', async () => {
      const mockChild = {
        on: jest.fn(),
        kill: jest.fn(),
        killed: false
      };
      
      spawn.mockReturnValue(mockChild);
      
      // Mock child that never responds
      mockChild.on.mockImplementation((event, callback) => {
        // Don't call any callbacks
      });
      
      const result = await DownloadManager.verifyBinary('/path/to/binary', { timeout: 100 });
      expect(result).toBe(false);
    });
    
    test('should handle _calculateSHA256 with file read error', async () => {
      fs.createReadStream.mockImplementation(() => {
        const stream = new (require('events').EventEmitter)();
        setImmediate(() => stream.emit('error', new Error('Read failed')));
        return stream;
      });
      
      await expect(DownloadManager._calculateSHA256('/path/to/file'))
        .rejects.toThrow('Read failed');
    });
    
    test('should handle downloadBinary with redirect loop', async () => {
      let redirectCount = 0;
      
      https.get.mockImplementation((url, options, callback) => {
        redirectCount++;
        
        if (redirectCount > 10) {
          // Simulate too many redirects
          const error = new Error('Too many redirects');
          callback(error);
          return mockRequest;
        }
        
        // Always redirect
        mockResponse.statusCode = 302;
        mockResponse.headers = { location: 'https://example.com/redirect' + redirectCount };
        callback(mockResponse);
        return mockRequest;
      });
      
      await expect(DownloadManager.downloadBinary('https://example.com/file.tar.gz', '/tmp/test', { maxRetries: 1 }))
        .rejects.toThrow();
    });
    
    test('should handle downloadBinary with invalid redirect location', async () => {
      mockResponse.statusCode = 302;
      mockResponse.headers = { location: 'invalid-url' };
      
      https.get.mockImplementation((url, options, callback) => {
        callback(mockResponse);
        return mockRequest;
      });
      
      await expect(DownloadManager.downloadBinary('https://example.com/file.tar.gz', '/tmp/test', { maxRetries: 1 }))
        .rejects.toThrow();
    });
    
    test('should handle downloadBinary with empty response', async () => {
      mockResponse.statusCode = 200;
      
      https.get.mockImplementation((url, options, callback) => {
        callback(mockResponse);
        // Don't emit any data, just end immediately
        setImmediate(() => {
          mockResponse.emit('end');
        });
        return mockRequest;
      });
      
      // Mock fs.createWriteStream to return a writable stream
      const mockWriteStream = {
        write: jest.fn(),
        end: jest.fn((callback) => {
          if (callback) callback();
        }),
        on: jest.fn(),
        once: jest.fn()
      };
      fs.createWriteStream.mockReturnValue(mockWriteStream);
      
      const result = await DownloadManager.downloadBinary('https://example.com/file.tar.gz', '/tmp/test');
      expect(result).toBeUndefined();
    }, 15000);
    
    test('should handle verifyChecksum with empty checksum', async () => {
      fs.existsSync.mockReturnValue(true);
      
      await expect(DownloadManager.verifyChecksum('/tmp/test-file', ''))
        .rejects.toThrow('Invalid checksum format');
    });
    
    test('should handle verifyChecksum with whitespace-only checksum', async () => {
      fs.existsSync.mockReturnValue(true);
      
      await expect(DownloadManager.verifyChecksum('/tmp/test-file', '   '))
        .rejects.toThrow('Invalid checksum format');
    });
  });