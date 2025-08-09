/**
 * Platform detection utilities for flowspec-cli NPM wrapper
 * Maps Node.js platform/arch combinations to Go build targets
 */

class PlatformDetector {
  /**
   * Platform mapping from Node.js to Go build targets
   */
  static PLATFORM_MAPPING = {
    'linux': {
      'x64': { os: 'linux', arch: 'amd64', extension: '.tar.gz' },
      'arm64': { os: 'linux', arch: 'arm64', extension: '.tar.gz' }
    },
    'darwin': {
      'x64': { os: 'darwin', arch: 'amd64', extension: '.tar.gz' },
      'arm64': { os: 'darwin', arch: 'arm64', extension: '.tar.gz' }
    },
    'win32': {
      'x64': { os: 'windows', arch: 'amd64', extension: '.tar.gz' }
    }
  };

  /**
   * Detects the current platform and architecture
   * @returns {Object} Platform information object
   */
  static detectPlatform() {
    const nodePlatform = process.platform;
    const nodeArch = process.arch;

    const platformMapping = this.PLATFORM_MAPPING[nodePlatform];
    if (!platformMapping) {
      throw new Error(`Unsupported platform: ${nodePlatform}`);
    }

    const archMapping = platformMapping[nodeArch];
    if (!archMapping) {
      throw new Error(`Unsupported architecture: ${nodeArch} on ${nodePlatform}`);
    }

    return {
      nodePlatform,
      nodeArch,
      ...archMapping
    };
  }

  /**
   * Generates the correct binary filename for the detected platform
   * @param {Object} platform - Platform information from detectPlatform()
   * @returns {string} Binary filename
   */
  static getBinaryName(platform) {
    if (!platform) {
      platform = this.detectPlatform();
    }

    return `flowspec-cli-${platform.os}-${platform.arch}${platform.extension}`;
  }

  /**
   * Constructs GitHub release download URL for the binary
   * @param {string} version - Version string (e.g., "v1.0.0")
   * @param {Object} platform - Platform information from detectPlatform()
   * @returns {string} Complete download URL
   */
  static getDownloadUrl(version, platform) {
    if (!platform) {
      platform = this.detectPlatform();
    }

    const binaryName = this.getBinaryName(platform);
    const baseUrl = 'https://github.com/flowspec/flowspec-cli/releases/download';
    
    return `${baseUrl}/${version}/${binaryName}`;
  }

  /**
   * Validates if the current platform is supported
   * @param {Object} platform - Platform information from detectPlatform()
   * @returns {boolean} True if platform is supported
   */
  static isSupported(platform) {
    if (!platform) {
      try {
        platform = this.detectPlatform();
        return true;
      } catch (error) {
        return false;
      }
    }

    const { nodePlatform, nodeArch } = platform;
    return !!(this.PLATFORM_MAPPING[nodePlatform] && 
              this.PLATFORM_MAPPING[nodePlatform][nodeArch]);
  }

  /**
   * Gets a list of all supported platforms
   * @returns {Array} Array of supported platform objects
   */
  static getSupportedPlatforms() {
    const platforms = [];
    
    for (const [nodePlatform, archMap] of Object.entries(this.PLATFORM_MAPPING)) {
      for (const [nodeArch, platformInfo] of Object.entries(archMap)) {
        platforms.push({
          nodePlatform,
          nodeArch,
          ...platformInfo
        });
      }
    }
    
    return platforms;
  }
}

module.exports = { PlatformDetector };