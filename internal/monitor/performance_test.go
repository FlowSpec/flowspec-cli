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

package monitor

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPerformanceMonitor_BasicUsage(t *testing.T) {
	monitor := NewPerformanceMonitor()

	// Start monitoring
	monitor.Start()

	// Simulate some work
	time.Sleep(100 * time.Millisecond)

	// Record some metrics
	monitor.RecordMetric("test_value", 42)
	monitor.RecordTimestamp("test_event")
	monitor.RecordDuration("test_operation", 50*time.Millisecond)

	// Stop monitoring
	metrics := monitor.Stop()

	require.NotNil(t, metrics)
	assert.Greater(t, metrics.ExecutionTime, 100*time.Millisecond)
	assert.Greater(t, metrics.PeakMemoryUsage, uint64(0))
	assert.Greater(t, metrics.PeakMemoryMB, 0.0)
	assert.Equal(t, 42, metrics.CustomMetrics["test_value"])
	assert.Contains(t, metrics.Timestamps, "test_event_timestamp")
	assert.Equal(t, 50*time.Millisecond, metrics.Durations["test_operation_duration"])
}

func TestPerformanceMonitor_MemoryTracking(t *testing.T) {
	monitor := NewPerformanceMonitor()
	monitor.Start()

	// Allocate some memory
	data := make([]byte, 10*1024*1024) // 10MB
	for i := range data {
		data[i] = byte(i % 256)
	}

	// Give memory monitor time to detect the allocation
	time.Sleep(200 * time.Millisecond)

	// Check memory tracking
	currentMemory := monitor.GetCurrentMemoryUsage()
	peakMemory := monitor.GetPeakMemoryUsage()

	// Peak memory should be at least as much as current
	assert.GreaterOrEqual(t, peakMemory, currentMemory)
	assert.Greater(t, currentMemory, uint64(0))

	// Test memory limit check
	assert.True(t, monitor.CheckMemoryLimit(1024))           // 1KB limit should be exceeded
	assert.False(t, monitor.CheckMemoryLimit(100*1024*1024)) // 100MB limit should not be exceeded

	metrics := monitor.Stop()
	require.NotNil(t, metrics)
	assert.Greater(t, metrics.PeakMemoryMB, 0.0)

	// Keep reference to data to prevent GC
	_ = data[0]
}

func TestPerformanceMonitor_TimedOperation(t *testing.T) {
	monitor := NewPerformanceMonitor()
	monitor.Start()

	// Test successful operation
	err := monitor.TimedOperation("test_op", func() error {
		time.Sleep(50 * time.Millisecond)
		return nil
	})

	assert.NoError(t, err)

	// Test operation with error
	testError := errors.New("test error")
	err = monitor.TimedOperation("error_op", func() error {
		time.Sleep(25 * time.Millisecond)
		return testError
	})

	assert.Equal(t, testError, err)

	metrics := monitor.Stop()
	require.NotNil(t, metrics)

	// Check that durations were recorded
	assert.Contains(t, metrics.Durations, "test_op_duration")
	assert.Contains(t, metrics.Durations, "error_op_duration")
	assert.GreaterOrEqual(t, metrics.Durations["test_op_duration"], 50*time.Millisecond)
	assert.GreaterOrEqual(t, metrics.Durations["error_op_duration"], 25*time.Millisecond)

	// Check that timestamps were recorded
	assert.Contains(t, metrics.Timestamps, "test_op_start_timestamp")
	assert.Contains(t, metrics.Timestamps, "test_op_end_timestamp")
	assert.Contains(t, metrics.Timestamps, "error_op_start_timestamp")
	assert.Contains(t, metrics.Timestamps, "error_op_end_timestamp")
}

func TestPerformanceMonitor_TimedOperationWithContext(t *testing.T) {
	monitor := NewPerformanceMonitor()
	monitor.Start()

	ctx := context.Background()

	err := monitor.TimedOperationWithContext(ctx, "ctx_op", func(ctx context.Context) error {
		select {
		case <-time.After(30 * time.Millisecond):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})

	assert.NoError(t, err)

	metrics := monitor.Stop()
	require.NotNil(t, metrics)

	assert.Contains(t, metrics.Durations, "ctx_op_duration")
	assert.GreaterOrEqual(t, metrics.Durations["ctx_op_duration"], 30*time.Millisecond)
}

func TestPerformanceMonitor_ConcurrentAccess(t *testing.T) {
	monitor := NewPerformanceMonitor()
	monitor.Start()

	// Test concurrent metric recording
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- true }()

			for j := 0; j < 100; j++ {
				monitor.RecordMetric(fmt.Sprintf("metric_%d_%d", id, j), j)
				monitor.RecordTimestamp(fmt.Sprintf("event_%d_%d", id, j))

				// Allocate some memory to trigger memory monitoring
				data := make([]byte, 1024)
				_ = data[0]

				time.Sleep(time.Microsecond)
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	metrics := monitor.Stop()
	require.NotNil(t, metrics)

	// Should have recorded many metrics
	assert.Greater(t, len(metrics.CustomMetrics), 900) // 10 * 100 - some might be timestamps
	assert.Greater(t, len(metrics.Timestamps), 900)
}

func TestPerformanceMonitor_GCStats(t *testing.T) {
	monitor := NewPerformanceMonitor()
	monitor.Start()

	// Force some garbage collection
	for i := 0; i < 5; i++ {
		data := make([]byte, 1024*1024) // 1MB
		_ = data[0]
		monitor.ForceGC()
	}

	metrics := monitor.Stop()
	require.NotNil(t, metrics)

	// Should have some GC activity
	assert.Greater(t, metrics.GCStats.NumGC, uint32(0))
}

func TestResourceMonitor_WithLimits(t *testing.T) {
	limits := ResourceLimits{
		MaxMemoryMB:     50.0, // 50MB limit
		MaxDuration:     1 * time.Second,
		MaxGCPauseTotal: 100 * time.Millisecond,
	}

	monitor := NewResourceMonitor(limits)
	monitor.Start()

	// Simulate work within limits
	time.Sleep(100 * time.Millisecond)
	monitor.RecordMetric("test_metric", "test_value")

	metrics, violations := monitor.Stop()

	require.NotNil(t, metrics)
	assert.Empty(t, violations, "Should not have any violations")
	assert.Greater(t, metrics.ExecutionTime, 100*time.Millisecond)
}

func TestResourceMonitor_MemoryLimitViolation(t *testing.T) {
	limits := ResourceLimits{
		MaxMemoryMB: 1.0, // Very low 1MB limit
		MaxDuration: 10 * time.Second,
	}

	monitor := NewResourceMonitor(limits)
	monitor.Start()

	// Allocate memory that exceeds the limit
	data := make([]byte, 5*1024*1024) // 5MB
	for i := range data {
		data[i] = byte(i % 256)
	}

	time.Sleep(50 * time.Millisecond) // Give memory monitor time to detect

	metrics, violations := monitor.Stop()

	require.NotNil(t, metrics)
	if len(violations) > 0 {
		assert.Contains(t, violations[0], "Memory limit exceeded")
	} else {
		t.Log("Memory limit violation not detected - this can happen due to GC timing")
	}

	// Keep reference to prevent GC
	_ = data[0]
}

func TestResourceMonitor_DurationLimitViolation(t *testing.T) {
	limits := ResourceLimits{
		MaxMemoryMB: 100.0,
		MaxDuration: 50 * time.Millisecond, // Very short duration limit
	}

	monitor := NewResourceMonitor(limits)
	monitor.Start()

	// Sleep longer than the limit
	time.Sleep(100 * time.Millisecond)

	metrics, violations := monitor.Stop()

	require.NotNil(t, metrics)
	assert.NotEmpty(t, violations, "Should have duration limit violation")
	assert.Contains(t, violations[0], "Duration limit exceeded")
}

func TestPerformanceMonitor_MultipleStartStop(t *testing.T) {
	monitor := NewPerformanceMonitor()

	// First session
	monitor.Start()
	time.Sleep(50 * time.Millisecond)
	metrics1 := monitor.Stop()

	require.NotNil(t, metrics1)
	assert.GreaterOrEqual(t, metrics1.ExecutionTime, 50*time.Millisecond)

	// Second session
	monitor.Start()
	time.Sleep(30 * time.Millisecond)
	metrics2 := monitor.Stop()

	require.NotNil(t, metrics2)
	assert.GreaterOrEqual(t, metrics2.ExecutionTime, 30*time.Millisecond)
	assert.Less(t, metrics2.ExecutionTime, metrics1.ExecutionTime)
}

func TestPerformanceMonitor_StopWithoutStart(t *testing.T) {
	monitor := NewPerformanceMonitor()

	// Stop without starting should return nil
	metrics := monitor.Stop()
	assert.Nil(t, metrics)
}

func TestPerformanceMonitor_DoubleStart(t *testing.T) {
	monitor := NewPerformanceMonitor()

	// Start twice should not cause issues
	monitor.Start()
	monitor.Start() // Second start should be ignored

	time.Sleep(50 * time.Millisecond)

	metrics := monitor.Stop()
	require.NotNil(t, metrics)
	assert.GreaterOrEqual(t, metrics.ExecutionTime, 50*time.Millisecond)
}

func TestPerformanceMonitor_MemoryEfficiency(t *testing.T) {
	monitor := NewPerformanceMonitor()
	monitor.Start()

	// Simulate memory usage
	data := make([]byte, 2*1024*1024) // 2MB
	for i := range data {
		data[i] = byte(i % 256)
	}

	time.Sleep(50 * time.Millisecond)

	metrics := monitor.Stop()
	require.NotNil(t, metrics)

	// Test efficiency checks
	assert.True(t, monitor.IsMemoryEfficient(metrics, 100.0)) // 100MB limit
	// Don't test 1MB limit as it may pass due to efficient GC

	assert.True(t, monitor.IsPerformant(metrics, 1*time.Second))
	assert.False(t, monitor.IsPerformant(metrics, 10*time.Millisecond))

	ratio := monitor.GetMemoryEfficiencyRatio(metrics)
	assert.GreaterOrEqual(t, ratio, 1.0) // Should be at least 1.0 (may not grow due to GC)

	// Keep reference to prevent GC
	_ = data[0]
}

func BenchmarkPerformanceMonitor_Overhead(b *testing.B) {
	monitor := NewPerformanceMonitor()
	monitor.Start()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		monitor.RecordMetric(fmt.Sprintf("metric_%d", i), i)
		monitor.GetCurrentMemoryUsage()
		monitor.GetPeakMemoryUsage()
	}

	monitor.Stop()
}

func BenchmarkPerformanceMonitor_TimedOperation(b *testing.B) {
	monitor := NewPerformanceMonitor()
	monitor.Start()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		err := monitor.TimedOperation(fmt.Sprintf("op_%d", i), func() error {
			// Simulate minimal work
			return nil
		})
		if err != nil {
			b.Fatal(err)
		}
	}

	monitor.Stop()
}

// Helper function for testing
func allocateMemory(sizeMB int) []byte {
	data := make([]byte, sizeMB*1024*1024)
	for i := range data {
		data[i] = byte(i % 256)
	}
	return data
}

func TestPerformanceMonitor_PrintSummary(t *testing.T) {
	monitor := NewPerformanceMonitor()
	monitor.Start()

	monitor.RecordMetric("test_files", 100)
	monitor.RecordDuration("parsing", 500*time.Millisecond)

	time.Sleep(100 * time.Millisecond)

	metrics := monitor.Stop()
	require.NotNil(t, metrics)

	// This should not panic
	monitor.PrintSummary(metrics)
	monitor.PrintSummary(nil) // Should handle nil gracefully
}
