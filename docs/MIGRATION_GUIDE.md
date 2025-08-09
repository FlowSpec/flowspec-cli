# FlowSpec CLI Migration Guide

This guide helps you migrate from manual FlowSpec CLI installation to the NPM package for easier management and integration.

## Overview

The FlowSpec CLI NPM package (`@flowspec/cli`) provides the same functionality as the manually installed binary, but with several advantages:

- **Automatic platform detection** - No need to manually select the correct binary
- **Version management** - Easy to pin and update versions
- **Integrated workflow** - Seamless integration with Node.js projects
- **Dependency management** - Managed alongside other development dependencies
- **Security** - Automatic SHA256 checksum verification
- **CI/CD friendly** - Simplified installation in automated environments

## Migration Scenarios

### Scenario 1: Node.js Project with Manual Installation

**Current setup:**
- FlowSpec CLI installed manually or via `go install`
- Used in npm scripts or CI/CD pipelines
- Binary managed separately from project dependencies

**Migration steps:**

1. **Install NPM package**:
   ```bash
   npm install @flowspec/cli --save-dev
   ```

2. **Update package.json scripts** (if any):
   ```json
   {
     "scripts": {
       "validate:specs": "flowspec-cli align --path=./src --trace=./traces/integration.json --output=json",
       "test:integration": "flowspec-cli align --path=./services --trace=./traces/e2e.json --verbose"
     }
   }
   ```

3. **Test the installation**:
   ```bash
   npx flowspec-cli --version
   npm run validate:specs
   ```

4. **Remove manual installation** (optional):
   ```bash
   # Remove from PATH
   which flowspec-cli  # Find current location
   rm /usr/local/bin/flowspec-cli  # Remove (adjust path as needed)
   
   # Or if installed via go install
   rm $(go env GOPATH)/bin/flowspec-cli
   ```

### Scenario 2: CI/CD Pipeline Migration

**Current CI/CD setup:**
```yaml
# GitHub Actions example
- name: Install FlowSpec CLI
  run: |
    curl -L https://github.com/flowspec/flowspec-cli/releases/latest/download/flowspec-cli-linux-amd64.tar.gz | tar xz
    sudo mv flowspec-cli /usr/local/bin/
    
- name: Validate ServiceSpecs
  run: flowspec-cli align --path=./src --trace=./traces/ci.json --output=json
```

**Migrated CI/CD setup:**
```yaml
# GitHub Actions example
- name: Setup Node.js
  uses: actions/setup-node@v3
  with:
    node-version: '18'
    
- name: Install dependencies
  run: npm ci
  
- name: Validate ServiceSpecs
  run: npm run validate:specs
```

Or for global installation:
```yaml
- name: Setup Node.js
  uses: actions/setup-node@v3
  with:
    node-version: '18'
    
- name: Install FlowSpec CLI
  run: npm install -g @flowspec/cli
  
- name: Validate ServiceSpecs
  run: flowspec-cli align --path=./src --trace=./traces/ci.json --output=json
```

### Scenario 3: Docker Container Migration

**Current Dockerfile:**
```dockerfile
FROM node:18-alpine

# Manual FlowSpec CLI installation
RUN apk add --no-cache curl tar
RUN curl -L https://github.com/flowspec/flowspec-cli/releases/latest/download/flowspec-cli-linux-amd64.tar.gz | tar xz -C /usr/local/bin/

COPY package*.json ./
RUN npm ci

COPY . .
RUN flowspec-cli align --path=./src --trace=./traces/build.json --output=json
```

**Migrated Dockerfile:**
```dockerfile
FROM node:18-alpine

COPY package*.json ./
RUN npm ci

COPY . .
RUN npx @flowspec/cli align --path=./src --trace=./traces/build.json --output=json
```

### Scenario 4: Global Tool Migration

**Current setup:**
- FlowSpec CLI installed globally via `go install` or manual installation
- Used across multiple projects
- Not tied to specific project dependencies

**Migration options:**

**Option A: Global NPM installation**
```bash
# Remove current installation
rm $(which flowspec-cli)

# Install globally via NPM
npm install -g @flowspec/cli

# Verify installation
flowspec-cli --version
```

**Option B: Per-project installation (recommended)**
```bash
# For each project, install as dev dependency
cd /path/to/project1
npm install @flowspec/cli --save-dev

cd /path/to/project2
npm install @flowspec/cli --save-dev

# Use via npx or npm scripts
npx @flowspec/cli --help
```

## Version Management

### Pinning Specific Versions

With NPM, you can easily pin specific versions:

```json
{
  "devDependencies": {
    "@flowspec/cli": "1.2.3"
  }
}
```

### Updating Versions

```bash
# Check current version
npm list @flowspec/cli

# Update to latest version
npm update @flowspec/cli

# Install specific version
npm install @flowspec/cli@1.2.4 --save-dev
```

## Troubleshooting Migration Issues

### Binary Not Found After Migration

If you get "command not found" errors:

1. **Verify installation**:
   ```bash
   npm list @flowspec/cli
   ```

2. **Use npx instead**:
   ```bash
   npx @flowspec/cli --help
   ```

3. **Check PATH** (for global installations):
   ```bash
   npm config get prefix
   echo $PATH
   ```

### Version Conflicts

If you have both manual and NPM installations:

1. **Check which binary is being used**:
   ```bash
   which flowspec-cli
   ```

2. **Remove manual installation**:
   ```bash
   rm $(which flowspec-cli)
   ```

3. **Verify NPM version is used**:
   ```bash
   npx @flowspec/cli --version
   ```

### Platform Detection Issues

If the wrong binary is downloaded:

1. **Check platform detection**:
   ```bash
   node -e "console.log(process.platform, process.arch)"
   ```

2. **Reinstall package**:
   ```bash
   npm uninstall @flowspec/cli
   npm install @flowspec/cli --save-dev
   ```

## Best Practices After Migration

### 1. Use Package.json Scripts

Instead of running commands directly, define them in package.json:

```json
{
  "scripts": {
    "validate": "flowspec-cli align --path=./src --trace=./traces/test.json --output=json",
    "validate:verbose": "flowspec-cli align --path=./src --trace=./traces/test.json --output=human --verbose",
    "ci:validate": "flowspec-cli align --path=. --trace=./traces/ci.json --output=json > validation-report.json"
  }
}
```

### 2. Version Consistency

Ensure all team members use the same version by committing package-lock.json:

```bash
git add package-lock.json
git commit -m "Lock FlowSpec CLI version"
```

### 3. CI/CD Integration

Use npm scripts in CI/CD for consistency:

```yaml
- name: Install dependencies
  run: npm ci
  
- name: Validate ServiceSpecs
  run: npm run validate
```

### 4. Documentation Updates

Update your project documentation to reflect the new installation method:

```markdown
## Development Setup

1. Install dependencies:
   ```bash
   npm install
   ```

2. Validate ServiceSpecs:
   ```bash
   npm run validate
   ```
```

## Rollback Plan

If you need to rollback to manual installation:

1. **Uninstall NPM package**:
   ```bash
   npm uninstall @flowspec/cli
   ```

2. **Reinstall manually**:
   ```bash
   # Via go install
   go install github.com/FlowSpec/flowspec-cli/cmd/flowspec-cli@latest
   
   # Or download binary
   curl -L https://github.com/flowspec/flowspec-cli/releases/latest/download/flowspec-cli-linux-amd64.tar.gz | tar xz
   sudo mv flowspec-cli /usr/local/bin/
   ```

3. **Update scripts** to use direct binary calls instead of npm scripts

## Support

If you encounter issues during migration:

1. Check the [troubleshooting guide](../npm/README.md#troubleshooting)
2. Search [existing issues](https://github.com/flowspec/flowspec-cli/issues)
3. Create a [new issue](https://github.com/flowspec/flowspec-cli/issues/new) with:
   - Your current installation method
   - Target installation method
   - Operating system and Node.js version
   - Complete error messages
   - Steps you've already tried

## Benefits Summary

After migration, you'll enjoy:

- ✅ **Simplified installation** - Single `npm install` command
- ✅ **Automatic updates** - Easy version management with npm
- ✅ **Platform independence** - No need to worry about architecture
- ✅ **Better integration** - Works seamlessly with Node.js toolchain
- ✅ **Enhanced security** - Automatic checksum verification
- ✅ **Consistent environments** - Same version across dev/CI/prod
- ✅ **Reduced maintenance** - No manual binary management