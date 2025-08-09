# NPM Publishing Workflow

This document describes the automated NPM publishing workflow that publishes the FlowSpec CLI as an NPM package whenever a new release is created.

## Overview

The NPM publishing workflow is integrated into the existing GitHub release workflow and automatically:

1. **Validates** the NPM package structure and configuration
2. **Synchronizes** the package version with the Git tag
3. **Tests** the package before publishing
4. **Publishes** to the NPM registry
5. **Verifies** the published package
6. **Provides** detailed feedback and error handling

## Workflow Structure

### Jobs

1. **`release`** - The existing job that builds and releases Go binaries
2. **`publish-npm`** - New job that publishes the NPM package (runs after `release`)

### Key Features

- **Automatic Version Sync**: Package version is automatically updated to match Git tags
- **Comprehensive Validation**: Package structure, dependencies, and configuration are validated
- **Error Handling**: Detailed error messages and rollback procedures
- **Security**: Uses NPM_TOKEN secret for authentication
- **Verification**: Confirms package is available on NPM registry after publishing

## Configuration Requirements

### GitHub Secrets

The workflow requires the following secret to be configured in the GitHub repository:

- **`NPM_TOKEN`**: NPM authentication token with publish permissions

#### Setting up NPM_TOKEN

1. Create an NPM account and login
2. Generate an access token:
   ```bash
   npm login
   npm token create --type=automation
   ```
3. Add the token to GitHub repository secrets:
   - Go to repository Settings → Secrets and variables → Actions
   - Click "New repository secret"
   - Name: `NPM_TOKEN`
   - Value: Your NPM token

### Package Structure

The workflow expects the following NPM package structure:

```
npm/
├── package.json          # NPM package metadata
├── install.js           # Installation script
├── bin/
│   └── flowspec-cli.js  # CLI wrapper script
├── lib/
│   ├── platform.js      # Platform detection
│   ├── download.js      # Download manager
│   └── binary.js        # Binary management
├── README.md            # Package documentation
└── .npmignore          # NPM ignore file
```

## Workflow Steps

### 1. Package Structure Validation

Validates that all required files and directories exist:
- `npm/package.json`
- `npm/install.js`
- `npm/lib/` directory with required modules
- `npm/bin/` directory with CLI wrapper

### 2. Version Synchronization

- Extracts version from Git tag (e.g., `v1.2.3` → `1.2.3`)
- Updates `package.json` version using `npm version`
- Validates version was updated correctly

### 3. Dependencies and Testing

- Installs production dependencies with `npm ci`
- Runs NPM package tests with `npm test`
- Validates package with `npm pack --dry-run`

### 4. Publishing

- Publishes package to NPM registry with `npm publish`
- Uses `--access public` for public packages
- Provides detailed logging for debugging

### 5. Verification

- Waits for registry propagation
- Verifies package is available on NPM
- Creates detailed summary report

### 6. Error Handling

If publishing fails:
- Restores original `package.json`
- Provides detailed error messages
- Creates troubleshooting guide
- Exits with error code to fail the workflow

## Testing

### Local Testing

Use the provided test scripts to validate the workflow locally:

```bash
# Validate workflow configuration
./scripts/validate-npm-workflow.sh

# Test workflow steps locally
./scripts/test-npm-workflow.sh
```

### Staging Testing

1. **Create a test release**:
   ```bash
   git tag v0.0.1-test
   git push origin v0.0.1-test
   ```

2. **Monitor workflow execution** in GitHub Actions

3. **Verify package publication** on NPM registry

4. **Test installation**:
   ```bash
   npm install @flowspec/cli@0.0.1-test
   npx flowspec-cli --version
   ```

## Troubleshooting

### Common Issues

#### NPM_TOKEN Authentication Failed
- **Cause**: Invalid or expired NPM token
- **Solution**: Regenerate NPM token and update GitHub secret

#### Version Already Exists
- **Cause**: Attempting to publish a version that already exists on NPM
- **Solution**: Use a different version number or unpublish the existing version

#### Package Validation Failed
- **Cause**: Invalid `package.json` or missing files
- **Solution**: Fix package structure and configuration

#### Network Issues
- **Cause**: Connectivity problems with NPM registry
- **Solution**: Retry the workflow or check NPM status

### Debugging Steps

1. **Check workflow logs** in GitHub Actions for detailed error messages
2. **Run local tests** to validate package configuration
3. **Verify NPM token** has correct permissions
4. **Check NPM registry status** at https://status.npmjs.org/
5. **Validate package.json** syntax and required fields

## Monitoring

### Success Indicators

- ✅ Workflow completes without errors
- ✅ Package appears on NPM registry
- ✅ Installation works: `npm install @flowspec/cli`
- ✅ CLI execution works: `npx flowspec-cli --version`

### Failure Indicators

- ❌ Workflow fails with error messages
- ❌ Package not found on NPM registry
- ❌ Installation fails
- ❌ CLI wrapper doesn't execute properly

## Maintenance

### Regular Tasks

1. **Monitor NPM token expiration** and renew as needed
2. **Update dependencies** in package.json
3. **Review and update** workflow configuration
4. **Test workflow** with pre-release versions

### Updates

When updating the workflow:

1. **Test changes locally** using the test scripts
2. **Use pre-release tags** for testing (e.g., `v1.2.3-beta.1`)
3. **Monitor workflow execution** carefully
4. **Validate published packages** before promoting to stable

## Security Considerations

- **NPM_TOKEN** is stored as a GitHub secret and not exposed in logs
- **Package validation** prevents publishing malformed packages
- **Version verification** ensures consistency between Git tags and NPM versions
- **Rollback procedures** minimize impact of failed publications

## Integration with Existing Workflow

The NPM publishing workflow is designed to integrate seamlessly with the existing release workflow:

- **Runs after** successful binary release
- **Uses same version** as Git tag
- **Inherits permissions** and environment
- **Provides independent** success/failure status
- **Doesn't affect** existing release process if NPM publishing fails