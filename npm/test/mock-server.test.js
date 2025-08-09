/**
 * Mock server tests for download scenarios without external dependencies
 * Tests download functionality using local HTTP servers
 */

const http = require('http');
const https = require('https');
const fs = require('fs');
const path = require('path');
const crypto = require('crypto');
const tar = require('tar');
const { DownloadManager } = require('../lib/download');

describe('DownloadManager Mock Server Tests', () => {
  let mockServer;
  let serverPort;
  let testDir;
  let mockBinaryContent;
  let mockChecksumContent;
  let mockBinaryChecksum;

  beforeAll(async () => {
    // Create test directory
    testDir = path.join(__dirname, '.mock-server-test');
    if (fs.existsSync(testDir)) {
      fs.rmSync(testDir, { recursive: true, force: true });
    }
    fs.mkdirSync(testDir, { recursive: true });

    // Create mock binary content
    mockBinaryContent = 'Mock binary content for testing';
    mockBinaryChecksum = crypto.createHash('sha256').update(mockBinaryContent).digest('hex');
    
    // Create mock checksums.txt content
    mockChecksumContent = `# Checksums for flowspec-cli v1.0.0
${mockBinaryChecksum}  flowspec-cli-linux-amd64.tar.gz
abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890  flowspec-cli-darwin-amd64.tar.gz
1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef  flowspec-cli-windows-amd64.tar.gz`;

    // Create mock tar.gz archive
    const tempBinaryPath = path.join(testDir, 'flowspec-cli');
    fs.writeFileSync(tempBinaryPath, mockBinaryContent);
    
    const archivePath = path.join(testDir, 'flowspec-cli-linux-amd64.tar.gz');
    await tar.create({
      gzip: true,
      file: archivePath,
      cwd: testDir
    }, ['flowspec-cli']);
    
    // Read the actual archive content for serving
    mockBinaryContent = fs.readFileSync(archivePath);
    mockBinaryChecksum = crypto.createHash('sha256').update(mockBinaryContent).digest('hex');
    
    // Update checksums content with actual checksum
    mockChecksumContent = `# Checksums for flowspec-cli v1.0.0
${mockBinaryChecksum}  flowspec-cli-linux-amd64.tar.gz
abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890  flowspec-cli-darwin-amd64.tar.gz
1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef  flowspec-cli-windows-amd64.tar.gz`;

  });

  afterAll(() => {
    if (fs.existsSync(testDir)) {
      fs.rmSync(testDir, { recursive: true, force: true });
    }
  });

  beforeEach((done) => {
    // Create a fresh mock server for each test
    mockServer = http.createServer((req, res) => {
      const url = req.url;
      
      if (url === '/checksums.txt') {
        res.writeHead(200, { 'Content-Type': 'text/plain' });
        res.end(mockChecksumContent);
      } else if (url === '/flowspec-cli-linux-amd64.tar.gz') {
        res.writeHead(200, { 
          'Content-Type': 'application/gzip',
          'Content-Length': mockBinaryContent.length
        });
        res.end(mockBinaryContent);
      } else if (url === '/slow-download') {
        // Simulate slow download with chunks
        res.writeHead(200, { 
          'Content-Type': 'application/gzip',
          'Content-Length': mockBinaryContent.length
        });
        
        let offset = 0;
        const chunkSize = 100;
        const sendChunk = () => {
          if (offset < mockBinaryContent.length) {
            const chunk = mockBinaryContent.slice(offset, offset + chunkSize);
            res.write(chunk);
            offset += chunkSize;
            setTimeout(sendChunk, 50); // 50ms delay between chunks
          } else {
            res.end();
          }
        };
        sendChunk();
      } else if (url === '/redirect-once') {
        res.writeHead(302, { 'Location': 'http://localhost:' + serverPort + '/flowspec-cli-linux-amd64.tar.gz' });
        res.end();
      } else if (url === '/redirect-loop') {
        res.writeHead(302, { 'Location': 'http://localhost:' + serverPort + '/redirect-loop' });
        res.end();
      } else if (url === '/not-found') {
        res.writeHead(404, { 'Content-Type': 'text/plain' });
        res.end('Not Found');
      } else if (url === '/server-error') {
        res.writeHead(500, { 'Content-Type': 'text/plain' });
        res.end('Internal Server Error');
      } else if (url === '/corrupted-binary') {
        res.writeHead(200, { 
          'Content-Type': 'application/gzip',
          'Content-Length': mockBinaryContent.length
        });
        // Send corrupted content
        res.end(Buffer.from('corrupted content'));
      } else if (url === '/timeout') {
        // Don't respond to simulate timeout
        return;
      } else if (url === '/partial-content') {
        res.writeHead(200, { 
          'Content-Type': 'application/gzip',
          'Content-Length': mockBinaryContent.length
        });
        // Send only partial content then close connection
        res.write(mockBinaryContent.slice(0, 50));
        res.destroy();
      } else {
        res.writeHead(404, { 'Content-Type': 'text/plain' });
        res.end('Not Found');
      }
    });

    mockServer.listen(0, 'localhost', () => {
      serverPort = mockServer.address().port;
      done();
    });
  });

  afterEach((done) => {
    if (mockServer) {
      mockServer.close(done);
    } else {
      done();
    }
  });

  describe('Successful download scenarios', () => {
    test('should download binary successfully from mock server', async () => {
      const targetPath = path.join(testDir, 'downloaded-binary.tar.gz');
      const url = `http://localhost:${serverPort}/flowspec-cli-linux-amd64.tar.gz`;

      await DownloadManager.downloadBinary(url, targetPath);

      expect(fs.existsSync(targetPath)).toBe(true);
      const downloadedContent = fs.readFileSync(targetPath);
      expect(downloadedContent.equals(mockBinaryContent)).toBe(true);
    });

    test('should download checksums successfully from mock server', async () => {
      const checksums = await DownloadManager.downloadChecksums('v1.0.0', {
        baseUrl: `http://localhost:${serverPort}`
      });

      expect(checksums).toBeInstanceOf(Map);
      expect(checksums.size).toBe(3);
      expect(checksums.get('flowspec-cli-linux-amd64.tar.gz')).toBe(mockBinaryChecksum);
    });

    test('should verify checksum successfully with downloaded binary', async () => {
      const targetPath = path.join(testDir, 'binary-for-checksum.tar.gz');
      const url = `http://localhost:${serverPort}/flowspec-cli-linux-amd64.tar.gz`;

      await DownloadManager.downloadBinary(url, targetPath);
      const result = await DownloadManager.verifyChecksum(targetPath, mockBinaryChecksum);

      expect(result).toBe(true);
    });

    test('should handle slow downloads with progress reporting', async () => {
      const targetPath = path.join(testDir, 'slow-download.tar.gz');
      const url = `http://localhost:${serverPort}/slow-download`;
      
      const progressUpdates = [];
      const progressCallback = (percent, downloaded, total) => {
        progressUpdates.push({ percent, downloaded, total });
      };

      await DownloadManager.downloadBinary(url, targetPath, {
        onProgress: progressCallback
      });

      expect(fs.existsSync(targetPath)).toBe(true);
      expect(progressUpdates.length).toBeGreaterThan(1);
      expect(progressUpdates[progressUpdates.length - 1].percent).toBe(100);
    }, 10000);

    test('should handle redirects correctly', async () => {
      const targetPath = path.join(testDir, 'redirected-binary.tar.gz');
      const url = `http://localhost:${serverPort}/redirect-once`;

      await DownloadManager.downloadBinary(url, targetPath);

      expect(fs.existsSync(targetPath)).toBe(true);
      const downloadedContent = fs.readFileSync(targetPath);
      expect(downloadedContent.equals(mockBinaryContent)).toBe(true);
    });
  });

  describe('Error scenarios', () => {
    test('should handle 404 errors', async () => {
      const targetPath = path.join(testDir, 'not-found.tar.gz');
      const url = `http://localhost:${serverPort}/not-found`;

      await expect(DownloadManager.downloadBinary(url, targetPath, { maxRetries: 1 }))
        .rejects.toThrow('Failed to download binary after 1 attempts');
    });

    test('should handle 500 server errors', async () => {
      const targetPath = path.join(testDir, 'server-error.tar.gz');
      const url = `http://localhost:${serverPort}/server-error`;

      await expect(DownloadManager.downloadBinary(url, targetPath, { maxRetries: 1 }))
        .rejects.toThrow('Failed to download binary after 1 attempts');
    });

    test('should handle checksum verification failures', async () => {
      const targetPath = path.join(testDir, 'corrupted-binary.tar.gz');
      const url = `http://localhost:${serverPort}/corrupted-binary`;

      await DownloadManager.downloadBinary(url, targetPath);
      
      const result = await DownloadManager.verifyChecksum(targetPath, mockBinaryChecksum);
      expect(result).toBe(false);
    });

    test('should handle connection timeouts', async () => {
      const targetPath = path.join(testDir, 'timeout-binary.tar.gz');
      const url = `http://localhost:${serverPort}/timeout`;

      await expect(DownloadManager.downloadBinary(url, targetPath, { 
        timeout: 1000,
        maxRetries: 1 
      })).rejects.toThrow('Failed to download binary after 1 attempts');
    }, 5000);

    test('should handle partial content errors', async () => {
      const targetPath = path.join(testDir, 'partial-binary.tar.gz');
      const url = `http://localhost:${serverPort}/partial-content`;

      await expect(DownloadManager.downloadBinary(url, targetPath, { maxRetries: 1 }))
        .rejects.toThrow('Failed to download binary after 1 attempts');
    });

    test('should handle infinite redirect loops', async () => {
      const targetPath = path.join(testDir, 'redirect-loop.tar.gz');
      const url = `http://localhost:${serverPort}/redirect-loop`;

      await expect(DownloadManager.downloadBinary(url, targetPath, { maxRetries: 1 }))
        .rejects.toThrow('Failed to download binary after 1 attempts');
    });
  });

  describe('Retry logic with mock server', () => {
    test('should retry on network errors and succeed', async () => {
      let attemptCount = 0;
      
      // Override the server handler for this test
      mockServer.removeAllListeners('request');
      mockServer.on('request', (req, res) => {
        attemptCount++;
        
        if (attemptCount < 3) {
          // Fail first 2 attempts
          res.destroy();
        } else {
          // Succeed on 3rd attempt
          res.writeHead(200, { 
            'Content-Type': 'application/gzip',
            'Content-Length': mockBinaryContent.length
          });
          res.end(mockBinaryContent);
        }
      });

      const targetPath = path.join(testDir, 'retry-success.tar.gz');
      const url = `http://localhost:${serverPort}/flowspec-cli-linux-amd64.tar.gz`;

      await DownloadManager.downloadBinary(url, targetPath, {
        maxRetries: 3,
        retryDelay: 100
      });

      expect(fs.existsSync(targetPath)).toBe(true);
      expect(attemptCount).toBe(3);
    });

    test('should respect maximum retry limit', async () => {
      let attemptCount = 0;
      
      // Override the server handler for this test
      mockServer.removeAllListeners('request');
      mockServer.on('request', (req, res) => {
        attemptCount++;
        res.destroy(); // Always fail
      });

      const targetPath = path.join(testDir, 'retry-fail.tar.gz');
      const url = `http://localhost:${serverPort}/flowspec-cli-linux-amd64.tar.gz`;

      await expect(DownloadManager.downloadBinary(url, targetPath, {
        maxRetries: 2,
        retryDelay: 50
      })).rejects.toThrow('Failed to download binary after 2 attempts');

      expect(attemptCount).toBe(2);
    });
  });

  describe('Performance and stress tests', () => {
    test('should handle large file downloads efficiently', async () => {
      // Create a larger mock binary (1MB)
      const largeBinaryContent = Buffer.alloc(1024 * 1024, 'A');
      
      // Override server for this test
      mockServer.removeAllListeners('request');
      mockServer.on('request', (req, res) => {
        res.writeHead(200, { 
          'Content-Type': 'application/gzip',
          'Content-Length': largeBinaryContent.length
        });
        res.end(largeBinaryContent);
      });

      const targetPath = path.join(testDir, 'large-binary.tar.gz');
      const url = `http://localhost:${serverPort}/large-binary`;
      
      const startTime = Date.now();
      await DownloadManager.downloadBinary(url, targetPath);
      const downloadTime = Date.now() - startTime;

      expect(fs.existsSync(targetPath)).toBe(true);
      expect(fs.statSync(targetPath).size).toBe(largeBinaryContent.length);
      
      // Should complete within reasonable time (adjust based on system performance)
      expect(downloadTime).toBeLessThan(5000);
    }, 10000);

    test('should handle concurrent downloads', async () => {
      const downloadPromises = [];
      
      for (let i = 0; i < 5; i++) {
        const targetPath = path.join(testDir, `concurrent-${i}.tar.gz`);
        const url = `http://localhost:${serverPort}/flowspec-cli-linux-amd64.tar.gz`;
        
        downloadPromises.push(DownloadManager.downloadBinary(url, targetPath));
      }

      await Promise.all(downloadPromises);

      // Verify all files were downloaded
      for (let i = 0; i < 5; i++) {
        const targetPath = path.join(testDir, `concurrent-${i}.tar.gz`);
        expect(fs.existsSync(targetPath)).toBe(true);
      }
    }, 10000);
  });

  describe('Edge cases with mock server', () => {
    test('should handle empty response body', async () => {
      // Override server for this test
      mockServer.removeAllListeners('request');
      mockServer.on('request', (req, res) => {
        res.writeHead(200, { 
          'Content-Type': 'application/gzip',
          'Content-Length': '0'
        });
        res.end();
      });

      const targetPath = path.join(testDir, 'empty-binary.tar.gz');
      const url = `http://localhost:${serverPort}/empty-binary`;

      await DownloadManager.downloadBinary(url, targetPath);

      expect(fs.existsSync(targetPath)).toBe(true);
      expect(fs.statSync(targetPath).size).toBe(0);
    });

    test('should handle missing Content-Length header', async () => {
      // Override server for this test
      mockServer.removeAllListeners('request');
      mockServer.on('request', (req, res) => {
        res.writeHead(200, { 'Content-Type': 'application/gzip' });
        res.end(mockBinaryContent);
      });

      const targetPath = path.join(testDir, 'no-content-length.tar.gz');
      const url = `http://localhost:${serverPort}/no-content-length`;

      await DownloadManager.downloadBinary(url, targetPath);

      expect(fs.existsSync(targetPath)).toBe(true);
    });

    test('should handle malformed checksums response', async () => {
      // Override server for this test
      mockServer.removeAllListeners('request');
      mockServer.on('request', (req, res) => {
        if (req.url === '/v1.0.0/checksums.txt') {
          res.writeHead(200, { 'Content-Type': 'text/plain' });
          res.end('malformed checksums content without proper format');
        } else {
          res.writeHead(404);
          res.end('Not Found');
        }
      });

      await expect(DownloadManager.downloadChecksums('v1.0.0', {
        baseUrl: `http://localhost:${serverPort}`
      })).rejects.toThrow('No valid checksums found in checksums.txt');
    });
  });
});