# FlowSpec v2 Migration Guide

This guide helps you migrate from FlowSpec v1 to v2, which introduces significant enhancements including traffic exploration, YAML contracts, and improved CI/CD integration.

## Overview of Changes

FlowSpec v2 introduces several major improvements:

- 🔍 **Traffic Exploration**: Automatically generate contracts from traffic logs
- 📄 **YAML Contracts**: Standalone contract files independent of source code
- 🎯 **Enhanced CI/CD**: Improved CI mode and GitHub Action support
- 🔧 **New Commands**: `explore` and `verify` commands
- 📊 **Better Reporting**: Enhanced error handling and exit codes
- 🌐 **Multi-format Support**: Support for various trace formats

## Breaking Changes

### 1. Data Model Changes

#### Old Format (v1)
```yaml
spec:
  endpoints:
    - path: /api/users/{id}
      methods: ["GET", "PUT", "DELETE"]
      statusCodes: [200, 400, 404]
      requiredHeaders: ["authorization"]
      optionalHeaders: ["accept-language"]
      requiredQuery: []
      optionalQuery: ["include"]
```

#### New Format (v2)
```yaml
apiVersion: flowspec/v1alpha1
kind: ServiceSpec
metadata:
  name: user-service
  version: v1.0.0
spec:
  endpoints:
    - path: /api/users/{id}
      operations:
        - method: GET
          responses:
            statusRanges: ["2xx", "4xx"]
            aggregation: "range"
          required:
            headers: ["authorization"]
            query: []
          optional:
            headers: ["accept-language"]
            query: ["include"]
        - method: PUT
          responses:
            statusCodes: [200, 400, 404]
            aggregation: "exact"
          required:
            headers: ["authorization", "content-type"]
            query: []
        - method: DELETE
          responses:
            statusCodes: [204, 404]
            aggregation: "exact"
          required:
            headers: ["authorization"]
            query: []
```

### 2. Command Changes

#### Old Commands (v1)
```bash
# Only align command available
flowspec-cli align --path ./src --trace ./trace.json
```

#### New Commands (v2)
```bash
# align command (unchanged, backward compatible)
flowspec-cli align --path ./src --trace ./trace.json

# verify command (new, alias for align with YAML support)
flowspec-cli verify --path ./contract.yaml --trace ./trace.json

# explore command (new, for traffic exploration)
flowspec-cli explore --traffic ./logs/ --out ./contract.yaml
```

## Migration Steps

### Step 1: Update FlowSpec CLI

#### NPM Installation
```bash
# Uninstall old version
npm uninstall -g @flowspec/cli

# Install new version
npm install -g @flowspec/cli@latest
```

#### Go Installation
```bash
# Install new version
go install github.com/FlowSpec/flowspec-cli/cmd/flowspec-cli@latest
```

#### Verify Installation
```bash
flowspec-cli --version
# Should show v2.x.x
```

### Step 2: Choose Migration Strategy

You have three migration strategies:

1. **Gradual Migration**: Keep existing annotations, add YAML contracts gradually
2. **Full Migration**: Convert all annotations to YAML contracts
3. **Hybrid Approach**: Use both annotations and YAML contracts

#### Strategy 1: Gradual Migration (Recommended)

Keep your existing ServiceSpec annotations and gradually introduce YAML contracts:

```bash
# Continue using existing annotations
flowspec-cli align --path ./src --trace ./trace.json

# Start using verify command (same behavior)
flowspec-cli verify --path ./src --trace ./trace.json

# Generate YAML contracts from traffic when ready
flowspec-cli explore --traffic ./logs/ --out ./contracts/service.yaml
```

#### Strategy 2: Full Migration

Convert all annotations to YAML contracts:

1. **Generate initial contracts from traffic**:
   ```bash
   flowspec-cli explore --traffic ./logs/ --out ./contracts/service.yaml
   ```

2. **Review and refine contracts**:
   - Compare generated contracts with your annotations
   - Adjust thresholds and field requirements
   - Add missing operations or endpoints

3. **Update CI/CD pipelines**:
   ```bash
   # Before
   flowspec-cli align --path ./src --trace ./trace.json
   
   # After
   flowspec-cli verify --path ./contracts/service.yaml --trace ./trace.json
   ```

4. **Remove source code annotations** (optional)

#### Strategy 3: Hybrid Approach

Use both annotations and YAML contracts for different services:

```bash
# Service A: Use annotations
flowspec-cli verify --path ./service-a/src --trace ./traces/service-a.json

# Service B: Use YAML contract
flowspec-cli verify --path ./contracts/service-b.yaml --trace ./traces/service-b.json
```

### Step 3: Update Data Models

#### Automatic Conversion Tool

Create a conversion script to transform old format to new format:

```bash
#!/bin/bash
# convert-contract.sh

# Convert old YAML format to new format
convert_contract() {
    local input_file="$1"
    local output_file="$2"
    
    # Add required headers
    cat > "$output_file" << EOF
apiVersion: flowspec/v1alpha1
kind: ServiceSpec
metadata:
  name: $(basename "$input_file" .yaml)
  version: v1.0.0
spec:
EOF
    
    # Convert endpoints
    yq eval '
      .spec.endpoints[] |= (
        .operations = (
          .methods[] as $method | {
            "method": $method,
            "responses": {
              "statusCodes": .statusCodes,
              "aggregation": "exact"
            },
            "required": {
              "headers": (.requiredHeaders // []),
              "query": (.requiredQuery // [])
            },
            "optional": {
              "headers": (.optionalHeaders // []),
              "query": (.optionalQuery // [])
            }
          }
        ) |
        del(.methods, .statusCodes, .requiredHeaders, .optionalHeaders, .requiredQuery, .optionalQuery)
      )
    ' "$input_file" >> "$output_file"
}

# Usage
convert_contract old-contract.yaml new-contract.yaml
```

#### Manual Conversion

For each endpoint in your old contract:

1. **Split methods into operations**:
   ```yaml
   # Old
   methods: ["GET", "POST"]
   
   # New
   operations:
     - method: GET
       # ... operation-specific config
     - method: POST
       # ... operation-specific config
   ```

2. **Convert field specifications**:
   ```yaml
   # Old
   requiredHeaders: ["authorization"]
   optionalHeaders: ["accept-language"]
   
   # New
   required:
     headers: ["authorization"]
     query: []
   optional:
     headers: ["accept-language"]
     query: []
   ```

3. **Update response specifications**:
   ```yaml
   # Old
   statusCodes: [200, 400, 404]
   
   # New
   responses:
     statusCodes: [200, 400, 404]
     aggregation: "exact"
   ```

### Step 4: Update CI/CD Pipelines

#### GitHub Actions

##### Before (v1)
```yaml
name: FlowSpec Validation
on: [push, pull_request]

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Install FlowSpec
        run: npm install -g @flowspec/cli@1.x
      - name: Validate
        run: flowspec-cli align --path ./src --trace ./trace.json
```

##### After (v2)
```yaml
name: FlowSpec Validation
on: [push, pull_request]

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      # Option 1: Use GitHub Action (recommended)
      - name: FlowSpec Verification
        uses: flowspec/flowspec-action@v1
        with:
          path: ./contracts/service.yaml
          trace: ./traces/integration.json
          ci: true
          
      # Option 2: Manual CLI installation
      - name: Install FlowSpec
        run: npm install -g @flowspec/cli@latest
      - name: Validate
        run: flowspec-cli verify --path ./contracts/service.yaml --trace ./trace.json --ci
```

#### NPM Scripts

##### Before (v1)
```json
{
  "scripts": {
    "validate": "flowspec-cli align --path ./src --trace ./traces/test.json"
  }
}
```

##### After (v2)
```json
{
  "scripts": {
    "validate": "flowspec-cli verify --path ./contracts/service.yaml --trace ./traces/test.json",
    "validate:src": "flowspec-cli verify --path ./src --trace ./traces/test.json",
    "generate:contract": "flowspec-cli explore --traffic ./logs/ --out ./contracts/service.yaml",
    "validate:ci": "flowspec-cli verify --path ./contracts/service.yaml --trace ./traces/test.json --ci"
  }
}
```

#### Makefile

##### Before (v1)
```makefile
validate:
	flowspec-cli align --path ./src --trace ./traces/test.json
```

##### After (v2)
```makefile
validate:
	flowspec-cli verify --path ./contracts/service.yaml --trace ./traces/test.json

validate-src:
	flowspec-cli verify --path ./src --trace ./traces/test.json

generate-contract:
	flowspec-cli explore --traffic ./logs/ --out ./contracts/service.yaml

validate-ci:
	flowspec-cli verify --path ./contracts/service.yaml --trace ./traces/test.json --ci
```

### Step 5: Update Documentation

#### README Updates

Add information about new features:

```markdown
## FlowSpec v2 Features

### Traffic Exploration
Generate contracts automatically from traffic logs:
```bash
flowspec-cli explore --traffic ./logs/nginx-access.log --out ./service-spec.yaml
```

### YAML Contracts
Use standalone YAML files for service specifications:
```bash
flowspec-cli verify --path ./service-spec.yaml --trace ./trace.json
```

### Enhanced CI/CD
Improved CI mode and GitHub Action support:
```bash
flowspec-cli verify --path ./service-spec.yaml --trace ./trace.json --ci
```
```

#### API Documentation

Update API documentation to reflect new data models and commands.

## Compatibility Matrix

| Feature | v1 | v2 | Notes |
|---------|----|----|-------|
| `align` command | ✅ | ✅ | Fully backward compatible |
| `verify` command | ❌ | ✅ | New alias for align with YAML support |
| `explore` command | ❌ | ✅ | New traffic exploration feature |
| Source code annotations | ✅ | ✅ | Fully supported |
| YAML contracts (old format) | ✅ | ✅ | Deprecated but supported |
| YAML contracts (new format) | ❌ | ✅ | Recommended format |
| CI mode | ❌ | ✅ | Enhanced CI/CD integration |
| GitHub Action | ❌ | ✅ | Official action available |

## Common Migration Issues

### Issue 1: Schema Validation Errors

**Problem**: YAML contracts fail schema validation

**Solution**: Ensure required fields are present:
```yaml
apiVersion: flowspec/v1alpha1  # Required
kind: ServiceSpec              # Required
metadata:                      # Required
  name: service-name
  version: v1.0.0
spec:                         # Required
  endpoints: []
```

### Issue 2: Path Parameter Format

**Problem**: Path parameters not recognized

**Old format**: `/api/users/:id`
**New format**: `/api/users/{id}`

**Solution**: Update path parameter syntax:
```bash
# Find and replace
sed -i 's/:([^/]*)/{\1}/g' contracts/*.yaml
```

### Issue 3: Status Code Aggregation

**Problem**: Status codes not matching as expected

**Solution**: Choose appropriate aggregation strategy:
```yaml
# For flexible matching
responses:
  statusRanges: ["2xx", "4xx"]
  aggregation: "range"

# For exact matching
responses:
  statusCodes: [200, 400, 404]
  aggregation: "exact"

# Let FlowSpec decide
responses:
  statusCodes: [200, 201, 400, 404]
  aggregation: "auto"
```

### Issue 4: Field Requirements

**Problem**: Required/optional fields not detected correctly

**Solution**: Adjust thresholds:
```bash
# More strict (higher threshold = more fields required)
flowspec-cli explore --traffic ./logs/ --out ./contract.yaml --required-threshold 0.99

# More lenient (lower threshold = fewer fields required)
flowspec-cli explore --traffic ./logs/ --out ./contract.yaml --required-threshold 0.8
```

### Issue 5: CI/CD Integration

**Problem**: CI pipelines failing with new exit codes

**Solution**: Update error handling:
```bash
# Handle different exit codes
case $? in
  0) echo "Success" ;;
  1) echo "Validation failed" ;;
  2) echo "Contract format error" ;;
  3) echo "Parse error" ;;
  *) echo "System error" ;;
esac
```

## Testing Migration

### Validation Checklist

- [ ] FlowSpec CLI v2 installed and working
- [ ] Existing `align` commands still work
- [ ] New `verify` command works with source code
- [ ] New `verify` command works with YAML contracts
- [ ] `explore` command generates valid contracts
- [ ] CI/CD pipelines updated and working
- [ ] All team members trained on new features

### Test Script

Create a test script to validate migration:

```bash
#!/bin/bash
# test-migration.sh

set -e

echo "Testing FlowSpec v2 migration..."

# Test 1: Backward compatibility
echo "Test 1: align command (backward compatibility)"
flowspec-cli align --path ./src --trace ./traces/test.json

# Test 2: verify with source code
echo "Test 2: verify command with source code"
flowspec-cli verify --path ./src --trace ./traces/test.json

# Test 3: verify with YAML contract
echo "Test 3: verify command with YAML contract"
flowspec-cli verify --path ./contracts/service.yaml --trace ./traces/test.json

# Test 4: explore command
echo "Test 4: explore command"
flowspec-cli explore --traffic ./logs/sample.log --out ./test-contract.yaml

# Test 5: CI mode
echo "Test 5: CI mode"
flowspec-cli verify --path ./contracts/service.yaml --trace ./traces/test.json --ci

echo "All tests passed! Migration successful."
```

## Rollback Plan

If you need to rollback to v1:

### 1. Reinstall v1
```bash
npm install -g @flowspec/cli@1.x
```

### 2. Revert CI/CD Changes
```bash
git checkout HEAD~1 -- .github/workflows/
```

### 3. Use Old Commands
```bash
flowspec-cli align --path ./src --trace ./trace.json
```

## Getting Help

### Resources

- 📖 [FlowSpec v2 Documentation](https://github.com/FlowSpec/flowspec-cli)
- 🔧 [GitHub Action](https://github.com/marketplace/actions/flowspec-verification)
- 💬 [GitHub Discussions](https://github.com/FlowSpec/flowspec-cli/discussions)
- 🐛 [Issue Tracker](https://github.com/FlowSpec/flowspec-cli/issues)

### Migration Support

If you encounter issues during migration:

1. Check the [troubleshooting guide](https://github.com/FlowSpec/flowspec-cli/blob/main/docs/TROUBLESHOOTING.md)
2. Search [existing issues](https://github.com/FlowSpec/flowspec-cli/issues)
3. Create a [new issue](https://github.com/FlowSpec/flowspec-cli/issues/new) with:
   - FlowSpec version
   - Migration step where issue occurred
   - Error messages
   - Sample configuration files

### Community

Join our community for migration help:

- 💬 [Discord](https://discord.gg/flowspec)
- 📧 [Mailing List](mailto:support@flowspec.org)
- 🐦 [Twitter](https://twitter.com/flowspec)

## Conclusion

FlowSpec v2 provides significant improvements while maintaining backward compatibility. The migration can be done gradually, allowing you to adopt new features at your own pace.

Key benefits of migrating:

- ✅ **Better CI/CD Integration**: Enhanced CI mode and GitHub Action
- ✅ **Traffic-Driven Contracts**: Generate contracts from actual traffic
- ✅ **Flexible Contract Management**: YAML contracts independent of source code
- ✅ **Improved Error Handling**: Better exit codes and error messages
- ✅ **Enhanced Reporting**: More detailed validation reports

Start with the gradual migration approach to minimize risk and gradually adopt new features as your team becomes comfortable with them.