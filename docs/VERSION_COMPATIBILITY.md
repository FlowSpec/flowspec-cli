# FlowSpec Version Compatibility Guide

This document outlines the compatibility between different versions of FlowSpec and provides guidance on version management.

## Version Matrix

| FlowSpec Version | Release Date | Status | Support Level |
|------------------|--------------|--------|---------------|
| v2.1.x | 2025-08 | Current | Full Support |
| v2.0.x | 2025-07 | Stable | Full Support |
| v1.x.x | 2024-12 | Legacy | Security Updates Only |

## Feature Compatibility

### Data Model Compatibility

| Feature | v1.x | v2.0+ | Migration Required |
|---------|------|-------|-------------------|
| ServiceSpec Annotations | ✅ | ✅ | No |
| YAML Contracts (Legacy Format) | ✅ | ✅ | No |
| YAML Contracts (New Format) | ❌ | ✅ | Yes |
| Method-level Operations | ❌ | ✅ | Yes |
| Status Code Ranges | ❌ | ✅ | Yes |
| Field Requirements (Granular) | ❌ | ✅ | Yes |

### Command Compatibility

| Command | v1.x | v2.0+ | Notes |
|---------|------|-------|-------|
| `align` | ✅ | ✅ | Fully backward compatible |
| `verify` | ❌ | ✅ | New in v2.0, alias for align with YAML support |
| `explore` | ❌ | ✅ | New in v2.0, traffic exploration |

### API Compatibility

| API Feature | v1.x | v2.0+ | Breaking Change |
|-------------|------|-------|-----------------|
| CLI Exit Codes | Basic (0,1) | Extended (0,1,2,3,4,64) | No |
| JSON Output Format | v1 Schema | v2 Schema | Yes |
| Error Messages | Basic | Enhanced | No |
| Progress Reporting | Limited | Detailed | No |

## Upgrade Paths

### From v1.x to v2.0+

#### Recommended Upgrade Path
1. **Install v2.0+ alongside v1.x** (different package names)
2. **Test with existing workflows** using `align` command
3. **Gradually adopt new features** (`verify`, `explore`)
4. **Migrate contracts** to new YAML format
5. **Update CI/CD pipelines** to use new features
6. **Remove v1.x** when fully migrated

#### Breaking Changes to Consider

1. **JSON Output Schema Changes**
   ```json
   // v1.x
   {
     "results": [...],
     "summary": {
       "total": 5,
       "passed": 3,
       "failed": 2
     }
   }
   
   // v2.0+
   {
     "results": [...],
     "summary": {
       "total": 5,
       "success": 3,
       "failed": 2,
       "skipped": 0,
       "totalAssertions": 15,
       "failedAssertions": 3
     },
     "performanceInfo": {
       "memoryUsageMB": 45.2,
       "executionTimeMs": 1250
     }
   }
   ```

2. **YAML Contract Schema**
   - New contracts require `apiVersion` and `kind` fields
   - `operations` array replaces `methods` array
   - Granular field requirements (`required`/`optional` objects)

3. **Exit Code Expansion**
   - v1.x: 0 (success), 1 (failure)
   - v2.0+: 0 (success), 1 (validation failed), 2 (format error), 3 (parse error), 4 (system error), 64 (usage error)

### From v2.0 to v2.1+

#### Minor Version Updates
- Fully backward compatible
- New features added without breaking existing functionality
- Safe to upgrade without code changes

## Version Selection Guide

### When to Use v1.x (Legacy)

❌ **Not Recommended** - Only use if:
- You cannot upgrade due to organizational constraints
- You need a specific v1.x-only feature (none currently)
- You're in a maintenance-only mode

### When to Use v2.0+

✅ **Recommended** - Use for:
- All new projects
- Active development projects
- Projects requiring traffic exploration
- Projects using YAML contracts
- Enhanced CI/CD integration needs

## Compatibility Testing

### Test Matrix

Before upgrading, test these scenarios:

| Test Scenario | v1.x | v2.0+ | Expected Result |
|---------------|------|-------|-----------------|
| Basic align command | ✅ | ✅ | Identical output |
| JSON output parsing | ⚠️ | ✅ | Schema differences |
| CI/CD integration | ✅ | ✅ | Enhanced features in v2.0+ |
| Error handling | ✅ | ✅ | More detailed errors in v2.0+ |

### Compatibility Test Script

```bash
#!/bin/bash
# compatibility-test.sh

echo "FlowSpec Compatibility Test"
echo "=========================="

# Test basic functionality
echo "Testing basic align command..."
flowspec-cli align --path ./src --trace ./trace.json --output json > v2-output.json

# Compare with expected v1 behavior (if you have reference)
if [ -f "v1-reference.json" ]; then
    echo "Comparing with v1 reference..."
    # Note: Direct comparison will fail due to schema changes
    # Instead, compare key metrics
    
    v1_total=$(jq '.summary.total' v1-reference.json)
    v2_total=$(jq '.summary.total' v2-output.json)
    
    if [ "$v1_total" == "$v2_total" ]; then
        echo "✅ Total count matches"
    else
        echo "❌ Total count differs: v1=$v1_total, v2=$v2_total"
    fi
fi

# Test new features
echo "Testing new verify command..."
flowspec-cli verify --path ./src --trace ./trace.json --ci

echo "Testing YAML contract support..."
if [ -f "service-spec.yaml" ]; then
    flowspec-cli verify --path ./service-spec.yaml --trace ./trace.json
fi

echo "Compatibility test completed"
```

## Migration Timeline Recommendations

### Phase 1: Preparation (Week 1)
- Install FlowSpec v2.0+ in test environment
- Run compatibility tests
- Identify breaking changes impact
- Plan migration strategy

### Phase 2: Parallel Testing (Week 2-3)
- Run both v1.x and v2.0+ in parallel
- Compare outputs and behavior
- Update test suites
- Train team on new features

### Phase 3: Gradual Migration (Week 4-6)
- Start using `verify` command instead of `align`
- Introduce YAML contracts for new services
- Update CI/CD pipelines gradually
- Monitor for issues

### Phase 4: Full Migration (Week 7-8)
- Convert remaining services to YAML contracts
- Remove v1.x dependencies
- Update documentation
- Celebrate! 🎉

## Support Policy

### Long-Term Support (LTS)

| Version | LTS Status | Support Until | Security Updates |
|---------|------------|---------------|------------------|
| v2.0.x | Current LTS | 2026-07 | Yes |
| v1.x.x | Legacy | 2025-12 | Critical only |

### Update Recommendations

- **Patch Updates** (x.x.Z): Apply immediately (bug fixes, security)
- **Minor Updates** (x.Y.x): Apply within 1 month (new features, backward compatible)
- **Major Updates** (X.x.x): Plan migration (breaking changes possible)

## Version-Specific Documentation

### v2.1+ Features
- Enhanced GitHub Action integration
- Improved error messages
- Performance optimizations
- Additional trace format support

### v2.0 Features
- Traffic exploration (`explore` command)
- YAML contracts with new schema
- Enhanced CI/CD integration
- Improved error handling and exit codes

### v1.x Features (Legacy)
- Basic ServiceSpec annotation parsing
- Simple trace validation
- Basic reporting

## Troubleshooting Version Issues

### Common Version-Related Problems

1. **Command Not Found**
   ```bash
   # Check installed version
   flowspec-cli --version
   
   # Reinstall if needed
   npm install -g @flowspec/cli@latest
   ```

2. **Schema Validation Errors**
   ```bash
   # Check if using old YAML format
   grep -q "apiVersion" contract.yaml || echo "Missing apiVersion - old format detected"
   ```

3. **CI/CD Pipeline Failures**
   ```bash
   # Check exit code handling
   flowspec-cli verify --path ./contract.yaml --trace ./trace.json
   echo "Exit code: $?"
   ```

4. **JSON Output Parsing Issues**
   ```bash
   # Check JSON schema version
   jq '.version // "v1-format"' output.json
   ```

### Getting Version-Specific Help

- **v2.0+ Issues**: [GitHub Issues](https://github.com/FlowSpec/flowspec-cli/issues)
- **v1.x Issues**: [Legacy Support](mailto:legacy-support@flowspec.org)
- **Migration Help**: [Migration Support](https://github.com/FlowSpec/flowspec-cli/discussions/categories/migration)

## Best Practices

### Version Management

1. **Pin Versions in CI/CD**
   ```yaml
   # Good: Pin to specific version
   - run: npm install -g @flowspec/cli@2.1.0
   
   # Avoid: Using latest in production
   - run: npm install -g @flowspec/cli@latest
   ```

2. **Test Before Upgrading**
   ```bash
   # Test in staging first
   npm install -g @flowspec/cli@2.1.0
   ./run-tests.sh
   ```

3. **Document Version Requirements**
   ```markdown
   ## Requirements
   - FlowSpec CLI v2.0+ (for YAML contracts)
   - FlowSpec CLI v2.1+ (for enhanced GitHub Action)
   ```

### Compatibility Maintenance

1. **Keep Reference Outputs**
   - Save known-good outputs for regression testing
   - Update references when upgrading versions

2. **Monitor Breaking Changes**
   - Subscribe to FlowSpec release notifications
   - Review changelog before upgrading

3. **Gradual Adoption**
   - Don't adopt all new features immediately
   - Test thoroughly in non-production environments

## Conclusion

FlowSpec maintains strong backward compatibility while introducing powerful new features. By following this compatibility guide, you can safely upgrade and take advantage of new capabilities while minimizing disruption to existing workflows.

For the most up-to-date compatibility information, always refer to the [official documentation](https://github.com/FlowSpec/flowspec-cli) and [release notes](https://github.com/FlowSpec/flowspec-cli/releases).