// Copyright 2024-2025 FlowSpec
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package traffic

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultIngestOptions(t *testing.T) {
	options := DefaultIngestOptions()

	assert.NotNil(t, options)
	assert.Equal(t, "combined", options.LogFormat)
	assert.Equal(t, 1.0, options.SampleRate)
	assert.Equal(t, "drop", options.RedactionPolicy)
	assert.Equal(t, 10, options.MaxErrorSamples)
	
	// Check default sensitive keys
	expectedSensitiveKeys := []string{"authorization", "cookie", "set-cookie", "token", "password", "api_key"}
	assert.Equal(t, expectedSensitiveKeys, options.SensitiveKeys)
}

func TestNewIngestMetrics(t *testing.T) {
	metrics := NewIngestMetrics()

	assert.NotNil(t, metrics)
	assert.Equal(t, int64(0), metrics.TotalLines)
	assert.Equal(t, int64(0), metrics.ParsedLines)
	assert.Equal(t, int64(0), metrics.ErrorLines)
	assert.Equal(t, time.Duration(0), metrics.Duration)
	assert.NotNil(t, metrics.ErrorSamples)
	assert.Len(t, metrics.ErrorSamples, 0)
}

func TestIngestMetrics_AddError(t *testing.T) {
	metrics := NewIngestMetrics()
	maxSamples := 3

	// Add errors up to the limit
	metrics.AddError("error line 1", maxSamples)
	metrics.AddError("error line 2", maxSamples)
	metrics.AddError("error line 3", maxSamples)

	assert.Equal(t, int64(3), metrics.ErrorLines)
	assert.Len(t, metrics.ErrorSamples, 3)
	assert.Contains(t, metrics.ErrorSamples, "error line 1")
	assert.Contains(t, metrics.ErrorSamples, "error line 2")
	assert.Contains(t, metrics.ErrorSamples, "error line 3")

	// Add more errors beyond the limit
	metrics.AddError("error line 4", maxSamples)
	metrics.AddError("error line 5", maxSamples)

	// Error count should increase but samples should remain at limit
	assert.Equal(t, int64(5), metrics.ErrorLines)
	assert.Len(t, metrics.ErrorSamples, 3) // Still at limit
	assert.NotContains(t, metrics.ErrorSamples, "error line 4")
	assert.NotContains(t, metrics.ErrorSamples, "error line 5")
}

func TestIngestMetrics_AddParsed(t *testing.T) {
	metrics := NewIngestMetrics()

	metrics.AddParsed()
	metrics.AddParsed()
	metrics.AddParsed()

	assert.Equal(t, int64(3), metrics.ParsedLines)
}

func TestIngestMetrics_AddTotal(t *testing.T) {
	metrics := NewIngestMetrics()

	metrics.AddTotal()
	metrics.AddTotal()
	metrics.AddTotal()
	metrics.AddTotal()

	assert.Equal(t, int64(4), metrics.TotalLines)
}

func TestIngestMetrics_SetDuration(t *testing.T) {
	metrics := NewIngestMetrics()
	duration := 5 * time.Second

	metrics.SetDuration(duration)

	assert.Equal(t, duration, metrics.Duration)
}

func TestIngestMetrics_ErrorRate(t *testing.T) {
	metrics := NewIngestMetrics()

	// Test with no lines
	assert.Equal(t, 0.0, metrics.ErrorRate())

	// Test with some errors
	metrics.TotalLines = 100
	metrics.ErrorLines = 15

	expectedRate := 15.0 / 100.0
	assert.Equal(t, expectedRate, metrics.ErrorRate())

	// Test with no errors
	metrics.ErrorLines = 0
	assert.Equal(t, 0.0, metrics.ErrorRate())
}

func TestIngestMetrics_IsIncomplete(t *testing.T) {
	metrics := NewIngestMetrics()

	// Test with no lines (should not be incomplete)
	assert.False(t, metrics.IsIncomplete())

	// Test with error rate below threshold (10%)
	metrics.TotalLines = 100
	metrics.ErrorLines = 5 // 5% error rate
	assert.False(t, metrics.IsIncomplete())

	// Test with error rate at threshold (10%)
	metrics.ErrorLines = 10 // 10% error rate
	assert.False(t, metrics.IsIncomplete())

	// Test with error rate above threshold (>10%)
	metrics.ErrorLines = 15 // 15% error rate
	assert.True(t, metrics.IsIncomplete())

	// Test with very high error rate
	metrics.ErrorLines = 50 // 50% error rate
	assert.True(t, metrics.IsIncomplete())
}

func TestNormalizedRecord_Structure(t *testing.T) {
	timestamp := time.Now()
	
	record := &NormalizedRecord{
		Method:    "GET",
		Path:      "/api/users/123",
		RawPath:   "/api/users/123?include=profile",
		Status:    200,
		Timestamp: timestamp,
		Query: map[string][]string{
			"include": {"profile"},
			"format":  {"json"},
		},
		Headers: map[string][]string{
			"authorization": {"Bearer token123"},
			"accept":        {"application/json"},
		},
		Host:      "api.example.com",
		Scheme:    "https",
		BodyBytes: 1024,
	}

	// Verify all fields are set correctly
	assert.Equal(t, "GET", record.Method)
	assert.Equal(t, "/api/users/123", record.Path)
	assert.Equal(t, "/api/users/123?include=profile", record.RawPath)
	assert.Equal(t, 200, record.Status)
	assert.Equal(t, timestamp, record.Timestamp)
	assert.Equal(t, "api.example.com", record.Host)
	assert.Equal(t, "https", record.Scheme)
	assert.Equal(t, int64(1024), record.BodyBytes)

	// Verify query parameters
	assert.Len(t, record.Query, 2)
	assert.Equal(t, []string{"profile"}, record.Query["include"])
	assert.Equal(t, []string{"json"}, record.Query["format"])

	// Verify headers
	assert.Len(t, record.Headers, 2)
	assert.Equal(t, []string{"Bearer token123"}, record.Headers["authorization"])
	assert.Equal(t, []string{"application/json"}, record.Headers["accept"])
}

func TestTimeRange_Structure(t *testing.T) {
	since := time.Now().Add(-24 * time.Hour)
	until := time.Now()

	timeRange := &TimeRange{
		Since: &since,
		Until: &until,
	}

	assert.NotNil(t, timeRange.Since)
	assert.NotNil(t, timeRange.Until)
	assert.Equal(t, since, *timeRange.Since)
	assert.Equal(t, until, *timeRange.Until)

	// Test with nil values
	emptyRange := &TimeRange{}
	assert.Nil(t, emptyRange.Since)
	assert.Nil(t, emptyRange.Until)
}

func TestIngestOptions_Structure(t *testing.T) {
	since := time.Now().Add(-24 * time.Hour)
	until := time.Now()

	options := &IngestOptions{
		LogFormat:       "common",
		CustomRegex:     `^(\S+) - (\S+) \[([^\]]+)\]`,
		SampleRate:      0.5,
		TimeFilter: &TimeRange{
			Since: &since,
			Until: &until,
		},
		SensitiveKeys:   []string{"password", "token"},
		RedactionPolicy: "mask",
		MaxErrorSamples: 20,
	}

	assert.Equal(t, "common", options.LogFormat)
	assert.Equal(t, `^(\S+) - (\S+) \[([^\]]+)\]`, options.CustomRegex)
	assert.Equal(t, 0.5, options.SampleRate)
	assert.NotNil(t, options.TimeFilter)
	assert.Equal(t, since, *options.TimeFilter.Since)
	assert.Equal(t, until, *options.TimeFilter.Until)
	assert.Equal(t, []string{"password", "token"}, options.SensitiveKeys)
	assert.Equal(t, "mask", options.RedactionPolicy)
	assert.Equal(t, 20, options.MaxErrorSamples)
}