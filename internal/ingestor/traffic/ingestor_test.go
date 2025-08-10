package traffic

import (
	"testing"
	"time"
)

func TestDefaultIngestOptions(t *testing.T) {
	opts := DefaultIngestOptions()
	
	if opts.LogFormat != "combined" {
		t.Errorf("Expected LogFormat to be 'combined', got %s", opts.LogFormat)
	}
	
	if opts.SampleRate != 1.0 {
		t.Errorf("Expected SampleRate to be 1.0, got %f", opts.SampleRate)
	}
	
	if opts.RedactionPolicy != "drop" {
		t.Errorf("Expected RedactionPolicy to be 'drop', got %s", opts.RedactionPolicy)
	}
	
	if opts.MaxErrorSamples != 10 {
		t.Errorf("Expected MaxErrorSamples to be 10, got %d", opts.MaxErrorSamples)
	}
	
	expectedSensitiveKeys := []string{"authorization", "cookie", "set-cookie", "token", "password", "api_key"}
	if len(opts.SensitiveKeys) != len(expectedSensitiveKeys) {
		t.Errorf("Expected %d sensitive keys, got %d", len(expectedSensitiveKeys), len(opts.SensitiveKeys))
	}
}

func TestNewIngestMetrics(t *testing.T) {
	metrics := NewIngestMetrics()
	
	if metrics.TotalLines != 0 {
		t.Errorf("Expected TotalLines to be 0, got %d", metrics.TotalLines)
	}
	
	if metrics.ParsedLines != 0 {
		t.Errorf("Expected ParsedLines to be 0, got %d", metrics.ParsedLines)
	}
	
	if metrics.ErrorLines != 0 {
		t.Errorf("Expected ErrorLines to be 0, got %d", metrics.ErrorLines)
	}
	
	if len(metrics.ErrorSamples) != 0 {
		t.Errorf("Expected ErrorSamples to be empty, got %d items", len(metrics.ErrorSamples))
	}
}

func TestIngestMetrics_AddError(t *testing.T) {
	metrics := NewIngestMetrics()
	maxSamples := 3
	
	// Add errors up to the limit
	for i := 0; i < 5; i++ {
		errorLine := "error line " + string(rune('1'+i))
		metrics.AddError(errorLine, maxSamples)
	}
	
	if metrics.ErrorLines != 5 {
		t.Errorf("Expected ErrorLines to be 5, got %d", metrics.ErrorLines)
	}
	
	if len(metrics.ErrorSamples) != maxSamples {
		t.Errorf("Expected ErrorSamples to have %d items, got %d", maxSamples, len(metrics.ErrorSamples))
	}
	
	// Verify the first 3 error samples are collected
	expectedSamples := []string{"error line 1", "error line 2", "error line 3"}
	for i, expected := range expectedSamples {
		if metrics.ErrorSamples[i] != expected {
			t.Errorf("Expected ErrorSamples[%d] to be '%s', got '%s'", i, expected, metrics.ErrorSamples[i])
		}
	}
}

func TestIngestMetrics_AddParsed(t *testing.T) {
	metrics := NewIngestMetrics()
	
	metrics.AddParsed()
	metrics.AddParsed()
	
	if metrics.ParsedLines != 2 {
		t.Errorf("Expected ParsedLines to be 2, got %d", metrics.ParsedLines)
	}
}

func TestIngestMetrics_AddTotal(t *testing.T) {
	metrics := NewIngestMetrics()
	
	metrics.AddTotal()
	metrics.AddTotal()
	metrics.AddTotal()
	
	if metrics.TotalLines != 3 {
		t.Errorf("Expected TotalLines to be 3, got %d", metrics.TotalLines)
	}
}

func TestIngestMetrics_SetDuration(t *testing.T) {
	metrics := NewIngestMetrics()
	duration := 5 * time.Second
	
	metrics.SetDuration(duration)
	
	if metrics.Duration != duration {
		t.Errorf("Expected Duration to be %v, got %v", duration, metrics.Duration)
	}
}

func TestIngestMetrics_ErrorRate(t *testing.T) {
	metrics := NewIngestMetrics()
	
	// Test with no data
	if metrics.ErrorRate() != 0.0 {
		t.Errorf("Expected ErrorRate to be 0.0 with no data, got %f", metrics.ErrorRate())
	}
	
	// Add some data
	metrics.AddTotal()
	metrics.AddTotal()
	metrics.AddTotal()
	metrics.AddTotal()
	metrics.AddError("error1", 10)
	metrics.AddError("error2", 10)
	
	expectedRate := 2.0 / 4.0 // 2 errors out of 4 total
	if metrics.ErrorRate() != expectedRate {
		t.Errorf("Expected ErrorRate to be %f, got %f", expectedRate, metrics.ErrorRate())
	}
}

func TestIngestMetrics_IsIncomplete(t *testing.T) {
	metrics := NewIngestMetrics()
	
	// Test with low error rate (5%)
	for i := 0; i < 20; i++ {
		metrics.AddTotal()
	}
	metrics.AddError("error1", 10)
	
	if metrics.IsIncomplete() {
		t.Error("Expected IsIncomplete to be false with 5% error rate")
	}
	
	// Test with high error rate (15%)
	metrics.AddError("error2", 10)
	metrics.AddError("error3", 10)
	
	if !metrics.IsIncomplete() {
		t.Error("Expected IsIncomplete to be true with 15% error rate")
	}
}

func TestTimeRange(t *testing.T) {
	now := time.Now()
	since := now.Add(-1 * time.Hour)
	until := now.Add(1 * time.Hour)
	
	timeRange := &TimeRange{
		Since: &since,
		Until: &until,
	}
	
	if timeRange.Since == nil || !timeRange.Since.Equal(since) {
		t.Errorf("Expected Since to be %v, got %v", since, timeRange.Since)
	}
	
	if timeRange.Until == nil || !timeRange.Until.Equal(until) {
		t.Errorf("Expected Until to be %v, got %v", until, timeRange.Until)
	}
}

func TestNormalizedRecord(t *testing.T) {
	now := time.Now()
	record := &NormalizedRecord{
		Method:    "GET",
		Path:      "/api/users/123",
		RawPath:   "/api/users/123?include=profile",
		Status:    200,
		Timestamp: now,
		Query: map[string][]string{
			"include": {"profile"},
		},
		Headers: map[string][]string{
			"authorization": {"Bearer token123"},
			"accept":        {"application/json"},
		},
		Host:      "api.example.com",
		Scheme:    "https",
		BodyBytes: 1024,
	}
	
	if record.Method != "GET" {
		t.Errorf("Expected Method to be 'GET', got %s", record.Method)
	}
	
	if record.Path != "/api/users/123" {
		t.Errorf("Expected Path to be '/api/users/123', got %s", record.Path)
	}
	
	if record.Status != 200 {
		t.Errorf("Expected Status to be 200, got %d", record.Status)
	}
	
	if !record.Timestamp.Equal(now) {
		t.Errorf("Expected Timestamp to be %v, got %v", now, record.Timestamp)
	}
	
	if len(record.Query["include"]) != 1 || record.Query["include"][0] != "profile" {
		t.Errorf("Expected Query include to be ['profile'], got %v", record.Query["include"])
	}
	
	if len(record.Headers["authorization"]) != 1 || record.Headers["authorization"][0] != "Bearer token123" {
		t.Errorf("Expected Headers authorization to be ['Bearer token123'], got %v", record.Headers["authorization"])
	}
}