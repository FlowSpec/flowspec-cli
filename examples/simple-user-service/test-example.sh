#!/bin/bash

# Simple User Service Example Test Script

set -e

echo "🧪 Testing Simple User Service Example"
echo "======================="

EXAMPLE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_BINARY="$EXAMPLE_DIR/../../build/flowspec-cli"

# Check if CLI binary exists
if [ ! -f "$CLI_BINARY" ]; then
    echo "❌ FlowSpec CLI binary not found: $CLI_BINARY"
    echo "💡 Please run: make build first"
    exit 1
fi

echo "📍 Example Directory: $EXAMPLE_DIR"
echo "🔧 CLI Binary: $CLI_BINARY"

# Test success scenario
echo ""
echo "🟢 Testing Success Scenario..."
echo "Command: $CLI_BINARY align --path=$EXAMPLE_DIR/src --trace=$EXAMPLE_DIR/traces/success-scenario.json --output=human"
$CLI_BINARY align --path="$EXAMPLE_DIR/src" --trace="$EXAMPLE_DIR/traces/success-scenario.json" --output=human

EXIT_CODE=$?
if [ $EXIT_CODE -eq 0 ]; then
    echo "✅ Success scenario test passed (Exit Code: $EXIT_CODE)"
else
    echo "❌ Success scenario test failed (Exit Code: $EXIT_CODE)"
fi

echo ""
echo "🔴 Testing Precondition Failure Scenario..."
echo "Command: $CLI_BINARY align --path=$EXAMPLE_DIR/src --trace=$EXAMPLE_DIR/traces/precondition-failure.json --output=human"
$CLI_BINARY align --path="$EXAMPLE_DIR/src" --trace="$EXAMPLE_DIR/traces/precondition-failure.json" --output=human

EXIT_CODE=$?
if [ $EXIT_CODE -eq 1 ]; then
    echo "✅ Precondition failure scenario test passed (Exit Code: $EXIT_CODE)"
else
    echo "❌ Precondition failure scenario test failed (Exit Code: $EXIT_CODE)"
fi

echo ""
echo "🟡 Testing Postcondition Failure Scenario..."
echo "Command: $CLI_BINARY align --path=$EXAMPLE_DIR/src --trace=$EXAMPLE_DIR/traces/postcondition-failure.json --output=human"
$CLI_BINARY align --path="$EXAMPLE_DIR/src" --trace="$EXAMPLE_DIR/traces/postcondition-failure.json" --output=human

EXIT_CODE=$?
if [ $EXIT_CODE -eq 1 ]; then
    echo "✅ Postcondition failure scenario test passed (Exit Code: $EXIT_CODE)"
else
    echo "❌ Postcondition failure scenario test failed (Exit Code: $EXIT_CODE)"
fi

echo ""
echo "📊 Testing JSON Format Output..."
echo "Command: $CLI_BINARY align --path=$EXAMPLE_DIR/src --trace=$EXAMPLE_DIR/traces/success-scenario.json --output=json"
JSON_OUTPUT=$($CLI_BINARY align --path="$EXAMPLE_DIR/src" --trace="$EXAMPLE_DIR/traces/success-scenario.json" --output=json)

# Validate JSON format
if echo "$JSON_OUTPUT" | jq . > /dev/null 2>&1; then
    echo "✅ JSON format output test passed"
    echo "📋 JSON Output Summary:"
    echo "$JSON_OUTPUT" | jq '.summary'
else
    echo "❌ JSON format output test failed"
fi

echo ""
echo "🎉 Example tests completed!"
