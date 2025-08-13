# FlowSpec CLI Issue Fixes

This document describes the fixes implemented for the reported issues.

## Issue 1: Fix broken examples and improve out-of-the-box experience

### Problem
- The `verify` command failed with "unsupported trace format" errors
- The `explore` command examples were missing required files
- Examples were not usable out-of-the-box

### Root Cause
1. **Trace file format**: The example trace files were in Jaeger format instead of OTLP JSON format
2. **Missing attributes**: Trace files lacked the required span attributes for assertions
3. **Incorrect span names**: Span names didn't match the expected operation names

### Solution
1. **Fixed trace file format**: Converted all trace files to OTLP JSON format
2. **Added required attributes**: Added request/response body attributes and proper HTTP attributes
3. **Corrected span names**: Updated span names to match operation names for source code matching
4. **Updated scripts**: Fixed the verify-contracts.sh script to use the correct CLI path

### Files Modified
- `examples/yaml-contracts/test-traces/user-service-trace.json` - Fixed format and attributes
- `examples/yaml-contracts/test-traces/order-service-trace.json` - Fixed format and attributes  
- `examples/yaml-contracts/verify-contracts.sh` - Fixed CLI path references
- `examples/yaml-contracts/TRACE_FORMAT_GUIDE.md` - Added comprehensive format guide

### Verification
```bash
# Now works correctly
./build/flowspec-cli verify --path examples/yaml-contracts/user-service.yaml --trace examples/yaml-contracts/test-traces/user-service-trace.json

# Also works with source code annotations
./build/flowspec-cli verify --path examples/simple-user-service/src --trace examples/simple-user-service/traces/success-scenario.json
```

## Issue 2: Improve explore command UX for low-sample data

### Problem
- The `explore` command would silently generate empty contracts when no endpoints met the minimum sample threshold
- Users received no feedback about why the contract was empty
- No guidance on how to fix the issue

### Root Cause
The tool filtered out endpoints that didn't meet the `--min-samples` threshold (default: 5) but provided no user feedback about this filtering.

### Solution
Added intelligent warning messages when no endpoints are generated:

1. **Clear warning**: Explains why no endpoints were generated
2. **Actionable suggestion**: Recommends using the `--min-samples` flag
3. **Smart threshold calculation**: Suggests an appropriate threshold based on the actual data
4. **Context explanation**: Explains this commonly happens with small log files

### Code Changes
Modified `cmd/flowspec-cli/main.go`:
- Added warning logic after contract generation
- Implemented smart threshold suggestion algorithm
- Added helper functions `max()` and `min()`

### Before (Silent Failure)
```bash
$ ./build/flowspec-cli explore --traffic small.log --out contract.yaml
INFO[...] ✅ Contract generation completed successfully
INFO[...] Processed 3 log lines, 3 parsed successfully (100.0% success rate)
# Contract file contains: endpoints: []
```

### After (Clear Guidance)
```bash
$ ./build/flowspec-cli explore --traffic small.log --out contract.yaml
INFO[...] ✅ Contract generation completed successfully
INFO[...] Processed 3 log lines, 3 parsed successfully (100.0% success rate)
WARN[...] ⚠️  No endpoints were generated because none met the minimum sample threshold of 5
WARN[...] 💡 To include endpoints with fewer samples, consider lowering the threshold with the '--min-samples' flag
WARN[...]    Example: --min-samples 1
WARN[...]    This often happens with small log files or when testing with limited traffic data
```

### Smart Threshold Suggestions
The tool now suggests intelligent thresholds based on the data:
- **1 sample**: Suggests `--min-samples 1`
- **3 samples**: Suggests `--min-samples 1` 
- **8 samples**: Suggests `--min-samples 2`
- **21 samples**: Suggests `--min-samples 2`

### Verification
```bash
# Test with small log file
echo '127.0.0.1 - - [10/Oct/2025:14:00:00 +0000] "GET /api/health HTTP/1.1" 200 10 "-" "curl/7.88.1"' > small.log

# Shows helpful warning
./build/flowspec-cli explore --traffic small.log --out contract.yaml

# Follow the suggestion
./build/flowspec-cli explore --traffic small.log --out contract.yaml --min-samples 1
# Now generates endpoints successfully
```

## Testing the Fixes

### Test Issue 1 Fix
```bash
# Test YAML contract verification
./build/flowspec-cli verify --path examples/yaml-contracts/user-service.yaml --trace examples/yaml-contracts/test-traces/user-service-trace.json

# Test source code annotation verification  
./build/flowspec-cli verify --path examples/simple-user-service/src --trace examples/simple-user-service/traces/success-scenario.json

# Run the verification script
cd examples/yaml-contracts && ./verify-contracts.sh
```

### Test Issue 2 Fix
```bash
# Test with small log file (shows warning)
./build/flowspec-cli explore --traffic examples/test-small-access.log --out test.yaml

# Test with suggested threshold (works)
./build/flowspec-cli explore --traffic examples/test-small-access.log --out test.yaml --min-samples 2

# Test with larger log file (shows different suggestion)
./build/flowspec-cli explore --traffic examples/test-medium-access.log --out test.yaml
```

## Impact

### Issue 1 Impact
- ✅ All examples now work out-of-the-box
- ✅ New users can successfully run verification commands
- ✅ Clear documentation on trace format requirements
- ✅ Reduced barrier to adoption

### Issue 2 Impact  
- ✅ Users receive clear feedback when no endpoints are generated
- ✅ Actionable suggestions help users fix the issue immediately
- ✅ Smart threshold recommendations based on actual data
- ✅ Improved user experience for small datasets and testing scenarios

Both fixes significantly improve the out-of-the-box experience and reduce user confusion.