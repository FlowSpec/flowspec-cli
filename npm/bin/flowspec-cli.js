#!/usr/bin/env node

/**
 * FlowSpec CLI Wrapper
 * 
 * This script acts as a transparent wrapper around the flowspec-cli binary,
 * providing seamless integration with npm/npx and package.json scripts.
 * 
 * Features:
 * - Transparent argument and environment variable forwarding
 * - Proper exit code preservation
 * - Automatic binary installation if missing
 * - Clear error messages and troubleshooting guidance
 */

const { BinaryManager } = require('../lib/binary');

/**
 * Main execution function
 */
async function main() {
  try {
    // Get command line arguments (excluding node and script path)
    const args = process.argv.slice(2);
    
    // Execute the binary with transparent forwarding
    const exitCode = await BinaryManager.executeBinary(args, {
      env: process.env,
      cwd: process.cwd(),
      stdio: true
    });
    
    // Exit with the same code as the underlying binary
    process.exit(exitCode);
    
  } catch (error) {
    // Handle different types of errors with helpful messages
    handleError(error);
    process.exit(1);
  }
}

/**
 * Handles errors with user-friendly messages and troubleshooting guidance
 * @param {Error} error - The error that occurred
 */
function handleError(error) {
  const errorMessage = error.message || 'Unknown error occurred';
  
  console.error('❌ FlowSpec CLI Error:');
  console.error(`   ${errorMessage}`);
  console.error('');
  
  // Provide specific guidance based on error type
  if (errorMessage.includes('Binary not found') || 
      errorMessage.includes('not functional') ||
      errorMessage.includes('Failed to ensure binary exists')) {
    
    console.error('🔧 Troubleshooting Steps:');
    console.error('   1. Try reinstalling the package:');
    console.error('      npm uninstall @flowspec/cli && npm install @flowspec/cli');
    console.error('');
    console.error('   2. Check if your platform is supported:');
    console.error('      - Linux (x64, arm64)');
    console.error('      - macOS (x64, arm64)');
    console.error('      - Windows (x64)');
    console.error('');
    console.error('   3. Ensure you have proper network connectivity to download the binary');
    console.error('');
    console.error('   4. If the issue persists, please report it at:');
    console.error('      https://github.com/flowspec/flowspec-cli/issues');
    
  } else if (errorMessage.includes('Unsupported platform') || 
             errorMessage.includes('Unsupported architecture')) {
    
    console.error('🚫 Platform Not Supported:');
    console.error('   Your platform is not currently supported by FlowSpec CLI.');
    console.error('');
    console.error('   Supported platforms:');
    console.error('   - Linux (x64, arm64)');
    console.error('   - macOS (x64, arm64)');
    console.error('   - Windows (x64)');
    console.error('');
    console.error('   For manual installation instructions, visit:');
    console.error('   https://github.com/flowspec/flowspec-cli#installation');
    
  } else if (errorMessage.includes('network') || 
             errorMessage.includes('download') ||
             errorMessage.includes('ENOTFOUND') ||
             errorMessage.includes('ECONNREFUSED')) {
    
    console.error('🌐 Network Error:');
    console.error('   Failed to download the FlowSpec CLI binary.');
    console.error('');
    console.error('   Please check:');
    console.error('   - Your internet connection');
    console.error('   - Corporate firewall/proxy settings');
    console.error('   - GitHub.com accessibility');
    console.error('');
    console.error('   You can also try downloading manually from:');
    console.error('   https://github.com/flowspec/flowspec-cli/releases');
    
  } else if (errorMessage.includes('permission') || 
             errorMessage.includes('EACCES') ||
             errorMessage.includes('EPERM')) {
    
    console.error('🔒 Permission Error:');
    console.error('   Insufficient permissions to install or execute the binary.');
    console.error('');
    console.error('   Try:');
    console.error('   - Running with appropriate permissions');
    console.error('   - Installing in a user-writable directory');
    console.error('   - Checking file system permissions');
    
  } else {
    console.error('ℹ️  For more help:');
    console.error('   - Documentation: https://github.com/flowspec/flowspec-cli#readme');
    console.error('   - Issues: https://github.com/flowspec/flowspec-cli/issues');
  }
  
  console.error('');
}

/**
 * Handle process signals gracefully
 */
function setupSignalHandlers() {
  // Handle SIGINT (Ctrl+C)
  process.on('SIGINT', () => {
    console.log('\n⚠️  FlowSpec CLI interrupted by user');
    process.exit(130); // Standard exit code for SIGINT
  });
  
  // Handle SIGTERM
  process.on('SIGTERM', () => {
    console.log('\n⚠️  FlowSpec CLI terminated');
    process.exit(143); // Standard exit code for SIGTERM
  });
  
  // Handle uncaught exceptions
  process.on('uncaughtException', (error) => {
    console.error('\n💥 Unexpected error occurred:');
    console.error(error.stack || error.message);
    console.error('\nPlease report this issue at: https://github.com/flowspec/flowspec-cli/issues');
    process.exit(1);
  });
  
  // Handle unhandled promise rejections
  process.on('unhandledRejection', (reason, promise) => {
    console.error('\n💥 Unhandled promise rejection:');
    console.error(reason);
    console.error('\nPlease report this issue at: https://github.com/flowspec/flowspec-cli/issues');
    process.exit(1);
  });
}

// Set up signal handlers
setupSignalHandlers();

// Run the main function
main().catch((error) => {
  handleError(error);
  process.exit(1);
});