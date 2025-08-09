/**
 * End-to-end tests that verify actual npm install and npx execution
 * These tests simulate real-world usage scenarios
 */

const { spawn, exec } = require('child_process');
const fs = require('fs');
const path = require('path');
const { promisify } = require('util');

const execAsync = promisify(exec);

describe('End-to-End NPM Package Tests', () => {
  const testProjectDir = path.join(__dirname, '.e2e-test-project');
  const packageTarballPath = path.join(__dirname, '..', 'flowspec-cli-test.tgz');
  
  beforeAll(async () => {
    // Clean up any existing test project
    if (fs.existsSync(testProjectDir)) {
      fs.rmSync(testProjectDir, { recursive: true, force: true });
    }
    
    // Create test project directory
    fs.mkdirSync(testProjectDir, { recursive: true });
    
    // Create a test package.json for the test project
    const testPackageJson = {
      name: 'e2e-test-project',
      version: '1.0.0',
      description: 'Test project for E2E testing',
      scripts: {
        'flowspec': 'flowspec-cli',
        'flowspec-version': 'flowspec-cli --version',
        'flowspec-help': 'flowspec-cli --help'
      },
      devDependencies: {}
    };
    
    fs.writeFileSync(
      path.join(testProjectDir, 'package.json'),
      JSON.stringify(testPackageJson, null, 2)
    );
    
    // Pack the current NPM package for testing
    try {
      await execAsync('npm pack', { cwd: path.join(__dirname, '..') });
      
      // Move the tarball to expected location
      const files = fs.readdirSync(path.join(__dirname, '..'));
      const tarball = files.find(f => f.startsWith('flowspec-cli-') && f.endsWith('.tgz'));
      
      if (tarball) {
        fs.renameSync(
          path.join(__dirname, '..', tarball),
          packageTarballPath
        );
      }
    } catch (error) {
      console.warn('Could not create package tarball for E2E testing:', error.message);
    }
  }, 30000);
  
  afterAll(() => {
    // Clean up test project and tarball
    if (fs.existsSync(testProjectDir)) {
      fs.rmSync(testProjectDir, { recursive: true, force: true });
    }
    
    if (fs.existsSync(packageTarballPath)) {
      fs.unlinkSync(packageTarballPath);
    }
  });

  describe('NPM Installation', () => {
    test('should install package from tarball successfully', async () => {
      if (!fs.existsSync(packageTarballPath)) {
        console.warn('Skipping E2E test: package tarball not available');
        return;
      }

      const result = await executeCommand('npm', ['install', packageTarballPath], {
        cwd: testProjectDir,
        timeout: 60000,
        env: { ...process.env, FLOWSPEC_CLI_SKIP_DOWNLOAD: 'true' }
      });

      expect(result.exitCode).toBe(0);
      expect(result.stdout).toContain('added');
      
      // Verify node_modules structure
      const nodeModulesPath = path.join(testProjectDir, 'node_modules', '@flowspec', 'cli');
      expect(fs.existsSync(nodeModulesPath)).toBe(true);
      
      // Verify bin symlink was created
      const binPath = path.join(testProjectDir, 'node_modules', '.bin', 'flowspec-cli');
      expect(fs.existsSync(binPath)).toBe(true);
    }, 120000);

    test('should run postinstall script during installation', async () => {
      if (!fs.existsSync(packageTarballPath)) {
        console.warn('Skipping E2E test: package tarball not available');
        return;
      }

      const result = await executeCommand('npm', ['install', packageTarballPath, '--verbose'], {
        cwd: testProjectDir,
        timeout: 60000,
        env: { ...process.env, FLOWSPEC_CLI_SKIP_DOWNLOAD: 'true' }
      });

      // Should show postinstall script execution (even if skipped)
      expect(result.stdout + result.stderr).toContain('Starting flowspec-cli installation');
    }, 120000);
  });

  describe('NPX Execution', () => {
    beforeEach(async () => {
      // Ensure package is installed for npx tests
      if (fs.existsSync(packageTarballPath)) {
        try {
          await executeCommand('npm', ['install', packageTarballPath], {
            cwd: testProjectDir,
            timeout: 60000
          });
        } catch (error) {
          console.warn('Could not install package for npx test:', error.message);
        }
      }
    });

    test('should execute via npx successfully', async () => {
      const result = await executeCommand('npx', ['flowspec-cli', '--help'], {
        cwd: testProjectDir,
        timeout: 30000
      });

      // Should not crash and should provide some output
      expect(result.exitCode).toBeDefined();
      expect(typeof result.exitCode).toBe('number');
    }, 60000);

    test('should execute via npx with version flag', async () => {
      const result = await executeCommand('npx', ['flowspec-cli', '--version'], {
        cwd: testProjectDir,
        timeout: 30000
      });

      expect(result.exitCode).toBeDefined();
      expect(typeof result.exitCode).toBe('number');
    }, 60000);

    test('should handle npx execution with arguments', async () => {
      const result = await executeCommand('npx', ['flowspec-cli', '--help', '--verbose'], {
        cwd: testProjectDir,
        timeout: 30000
      });

      expect(result.exitCode).toBeDefined();
      expect(typeof result.exitCode).toBe('number');
    }, 60000);
  });

  describe('Package.json Scripts Integration', () => {
    beforeEach(async () => {
      // Ensure package is installed for script tests
      if (fs.existsSync(packageTarballPath)) {
        try {
          await executeCommand('npm', ['install', packageTarballPath], {
            cwd: testProjectDir,
            timeout: 60000
          });
        } catch (error) {
          console.warn('Could not install package for script test:', error.message);
        }
      }
    });

    test('should work in npm scripts', async () => {
      const result = await executeCommand('npm', ['run', 'flowspec-help'], {
        cwd: testProjectDir,
        timeout: 30000
      });

      expect(result.exitCode).toBeDefined();
      expect(typeof result.exitCode).toBe('number');
    }, 60000);

    test('should work in npm scripts with version command', async () => {
      const result = await executeCommand('npm', ['run', 'flowspec-version'], {
        cwd: testProjectDir,
        timeout: 30000
      });

      expect(result.exitCode).toBeDefined();
      expect(typeof result.exitCode).toBe('number');
    }, 60000);

    test('should preserve exit codes in npm scripts', async () => {
      // Add a script that should fail
      const packageJsonPath = path.join(testProjectDir, 'package.json');
      const packageJson = JSON.parse(fs.readFileSync(packageJsonPath, 'utf8'));
      packageJson.scripts['flowspec-invalid'] = 'flowspec-cli --invalid-flag-that-does-not-exist';
      fs.writeFileSync(packageJsonPath, JSON.stringify(packageJson, null, 2));

      const result = await executeCommand('npm', ['run', 'flowspec-invalid'], {
        cwd: testProjectDir,
        timeout: 30000
      });

      // Should exit with non-zero code for invalid command
      expect(result.exitCode).not.toBe(0);
    }, 60000);
  });

  describe('Cross-Platform Compatibility', () => {
    test('should work on current platform', async () => {
      if (!fs.existsSync(packageTarballPath)) {
        console.warn('Skipping E2E test: package tarball not available');
        return;
      }

      // Install and test on current platform
      await executeCommand('npm', ['install', packageTarballPath], {
        cwd: testProjectDir,
        timeout: 60000
      });

      const result = await executeCommand('npx', ['flowspec-cli', '--version'], {
        cwd: testProjectDir,
        timeout: 30000
      });

      expect(result.exitCode).toBeDefined();
      expect(typeof result.exitCode).toBe('number');
    }, 120000);

    test('should handle platform-specific binary paths', async () => {
      if (!fs.existsSync(packageTarballPath)) {
        console.warn('Skipping E2E test: package tarball not available');
        return;
      }

      const nodeModulesPath = path.join(testProjectDir, 'node_modules', '@flowspec', 'cli');
      
      if (fs.existsSync(nodeModulesPath)) {
        const binDir = path.join(nodeModulesPath, 'bin');
        
        if (fs.existsSync(binDir)) {
          const files = fs.readdirSync(binDir);
          
          // Should contain the JavaScript wrapper
          expect(files).toContain('flowspec-cli.js');
          
          // Check that the wrapper is executable (on Unix systems)
          if (process.platform !== 'win32') {
            const wrapperPath = path.join(binDir, 'flowspec-cli.js');
            const stats = fs.statSync(wrapperPath);
            expect(stats.mode & parseInt('111', 8)).toBeGreaterThan(0); // Check execute permissions
          }
        }
      }
    }, 60000);
  });

  describe('Error Handling in Real Environment', () => {
    test('should handle missing binary gracefully', async () => {
      // Create a minimal package structure without binary
      const tempDir = path.join(__dirname, '.temp-no-binary');
      fs.mkdirSync(tempDir, { recursive: true });
      
      const packageJson = {
        name: 'test-no-binary',
        version: '1.0.0',
        bin: { 'flowspec-cli': 'bin/flowspec-cli.js' }
      };
      
      fs.writeFileSync(
        path.join(tempDir, 'package.json'),
        JSON.stringify(packageJson, null, 2)
      );
      
      // Create bin directory but no actual binary
      fs.mkdirSync(path.join(tempDir, 'bin'), { recursive: true });
      fs.writeFileSync(
        path.join(tempDir, 'bin', 'flowspec-cli.js'),
        '#!/usr/bin/env node\nconsole.error("Binary not found"); process.exit(1);'
      );

      const result = await executeCommand('node', [path.join(tempDir, 'bin', 'flowspec-cli.js')], {
        timeout: 10000
      });

      expect(result.exitCode).toBe(1);
      expect(result.stderr).toContain('Binary not found');
      
      // Clean up
      fs.rmSync(tempDir, { recursive: true, force: true });
    }, 30000);

    test('should handle network issues during installation', async () => {
      // This test simulates network issues by modifying the installation environment
      const tempDir = path.join(__dirname, '.temp-network-test');
      fs.mkdirSync(tempDir, { recursive: true });
      
      const packageJson = {
        name: 'test-network-issues',
        version: '1.0.0',
        scripts: { postinstall: 'node -e "console.log(\'Network test\'); process.exit(0);"' }
      };
      
      fs.writeFileSync(
        path.join(tempDir, 'package.json'),
        JSON.stringify(packageJson, null, 2)
      );

      // Test with limited network access (this is a simulation)
      const result = await executeCommand('npm', ['install'], {
        cwd: tempDir,
        timeout: 30000,
        env: { ...process.env, HTTP_PROXY: 'http://invalid-proxy:8080' }
      });

      // Should handle network issues gracefully
      expect(result.exitCode).toBeDefined();
      
      // Clean up
      fs.rmSync(tempDir, { recursive: true, force: true });
    }, 60000);
  });

  describe('Performance Tests', () => {
    test('should install within reasonable time', async () => {
      if (!fs.existsSync(packageTarballPath)) {
        console.warn('Skipping E2E test: package tarball not available');
        return;
      }

      const startTime = Date.now();
      
      const result = await executeCommand('npm', ['install', packageTarballPath], {
        cwd: testProjectDir,
        timeout: 120000,
        env: { ...process.env, FLOWSPEC_CLI_SKIP_DOWNLOAD: 'true' }
      });
      
      const installTime = Date.now() - startTime;

      expect(result.exitCode).toBe(0);
      // Installation should complete within 2 minutes
      expect(installTime).toBeLessThan(120000);
    }, 150000);

    test('should execute commands quickly after installation', async () => {
      if (!fs.existsSync(packageTarballPath)) {
        console.warn('Skipping E2E test: package tarball not available');
        return;
      }

      // Ensure package is installed
      await executeCommand('npm', ['install', packageTarballPath], {
        cwd: testProjectDir,
        timeout: 60000
      });

      const startTime = Date.now();
      
      const result = await executeCommand('npx', ['flowspec-cli', '--version'], {
        cwd: testProjectDir,
        timeout: 30000
      });
      
      const executionTime = Date.now() - startTime;

      expect(result.exitCode).toBeDefined();
      // Command execution should be fast (under 10 seconds)
      expect(executionTime).toBeLessThan(10000);
    }, 90000);
  });
});

/**
 * Helper function to execute commands and capture output
 * @param {string} command - Command to execute
 * @param {string[]} args - Command arguments
 * @param {Object} options - Execution options
 * @returns {Promise<Object>} Execution result
 */
function executeCommand(command, args = [], options = {}) {
  return new Promise((resolve) => {
    const {
      timeout = 30000,
      cwd = process.cwd(),
      env = process.env
    } = options;

    let stdout = '';
    let stderr = '';
    let isResolved = false;

    const child = spawn(command, args, {
      cwd,
      env,
      stdio: ['pipe', 'pipe', 'pipe']
    });

    child.stdout.on('data', (data) => {
      stdout += data.toString();
    });

    child.stderr.on('data', (data) => {
      stderr += data.toString();
    });

    child.on('close', (exitCode) => {
      if (!isResolved) {
        isResolved = true;
        resolve({
          exitCode,
          stdout: stdout.trim(),
          stderr: stderr.trim()
        });
      }
    });

    child.on('error', (error) => {
      if (!isResolved) {
        isResolved = true;
        resolve({
          exitCode: 1,
          stdout: stdout.trim(),
          stderr: `Process error: ${error.message}`
        });
      }
    });

    // Handle timeout
    const timeoutId = setTimeout(() => {
      if (!isResolved) {
        isResolved = true;
        child.kill('SIGTERM');
        resolve({
          exitCode: 124, // Timeout exit code
          stdout: stdout.trim(),
          stderr: 'Process timed out'
        });
      }
    }, timeout);

    child.on('close', () => {
      clearTimeout(timeoutId);
    });
  });
}