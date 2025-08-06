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
	"fmt"
	"runtime"
	"sync"
	"time"
)

// PerformanceMonitor tracks performance metrics during CLI execution
type PerformanceMonitor struct {
	startTime       time.Time
	metrics         map[string]interface{}
	memoryPeakUsage uint64
	memoryMutex     sync.RWMutex
	stopChan        chan struct{}
	isMonitoring    bool
	monitorMutex    sync.Mutex
}

// PerformanceMetrics contains collected performance data
type PerformanceMetrics struct {
	ExecutionTime   time.Duration            `json:"execution_time"`
	PeakMemoryUsage uint64                   `json:"peak_memory_usage_bytes"`
	PeakMemoryMB    float64                  `json:"peak_memory_usage_mb"`
	InitialMemory   uint64                   `json:"initial_memory_bytes"`
	FinalMemory     uint64                   `json:"final_memory_bytes"`
	GCStats         runtime.MemStats         `json:"gc_stats"`
	CustomMetrics   map[string]interface{}   `json:"custom_metrics"`
	Timestamps      map[string]time.Time     `json:"timestamps"`
	Durations       map[string]time.Duration `json:"durations"`
}

// NewPerformanceMonitor creates a new performance monitor
func NewPerformanceMonitor() *PerformanceMonitor {
	return &PerformanceMonitor{
		metrics:      make(map[string]interface{}),
		isMonitoring: false,
	}
}

// Start begins performance monitoring
func (pm *PerformanceMonitor) Start() {
	pm.monitorMutex.Lock()
	defer pm.monitorMutex.Unlock()

	if pm.isMonitoring {
		return
	}

	pm.startTime = time.Now()
	pm.isMonitoring = true
	pm.stopChan = make(chan struct{})

	// Record initial memory usage
	var initialMem runtime.MemStats
	runtime.ReadMemStats(&initialMem)
	pm.memoryMutex.Lock()
	pm.memoryPeakUsage = initialMem.Alloc
	pm.memoryMutex.Unlock()

	pm.metrics["initial_memory"] = initialMem.Alloc
	pm.metrics["start_time"] = pm.startTime

	// Start memory monitoring goroutine
	go pm.monitorMemory()
}

// Stop ends performance monitoring and returns collected metrics
func (pm *PerformanceMonitor) Stop() *PerformanceMetrics {
	pm.monitorMutex.Lock()
	defer pm.monitorMutex.Unlock()

	if !pm.isMonitoring {
		return nil
	}

	// Stop monitoring
	if pm.stopChan != nil {
		close(pm.stopChan)
		pm.stopChan = nil
	}
	pm.isMonitoring = false

	// Record final metrics
	endTime := time.Now()
	executionTime := endTime.Sub(pm.startTime)

	var finalMem runtime.MemStats
	runtime.ReadMemStats(&finalMem)

	var gcStats runtime.MemStats
	runtime.ReadMemStats(&gcStats)

	pm.memoryMutex.RLock()
	peakMemory := pm.memoryPeakUsage
	pm.memoryMutex.RUnlock()

	// Create metrics object
	metrics := &PerformanceMetrics{
		ExecutionTime:   executionTime,
		PeakMemoryUsage: peakMemory,
		PeakMemoryMB:    float64(peakMemory) / (1024 * 1024),
		InitialMemory:   pm.metrics["initial_memory"].(uint64),
		FinalMemory:     finalMem.Alloc,
		GCStats:         gcStats,
		CustomMetrics:   make(map[string]interface{}),
		Timestamps:      make(map[string]time.Time),
		Durations:       make(map[string]time.Duration),
	}

	// Copy custom metrics
	for k, v := range pm.metrics {
		if k != "initial_memory" && k != "start_time" {
			switch val := v.(type) {
			case time.Time:
				metrics.Timestamps[k] = val
			case time.Duration:
				metrics.Durations[k] = val
			default:
				metrics.CustomMetrics[k] = val
			}
		}
	}

	return metrics
}

// RecordMetric records a custom metric
func (pm *PerformanceMonitor) RecordMetric(key string, value interface{}) {
	pm.monitorMutex.Lock()
	defer pm.monitorMutex.Unlock()

	pm.metrics[key] = value
}

// RecordTimestamp records a timestamp for a specific event
func (pm *PerformanceMonitor) RecordTimestamp(event string) {
	pm.RecordMetric(event+"_timestamp", time.Now())
}

// RecordDuration records the duration of an operation
func (pm *PerformanceMonitor) RecordDuration(operation string, duration time.Duration) {
	pm.RecordMetric(operation+"_duration", duration)
}

// GetCurrentMemoryUsage returns current memory usage in bytes
func (pm *PerformanceMonitor) GetCurrentMemoryUsage() uint64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m.Alloc
}

// GetPeakMemoryUsage returns peak memory usage in bytes
func (pm *PerformanceMonitor) GetPeakMemoryUsage() uint64 {
	pm.memoryMutex.RLock()
	defer pm.memoryMutex.RUnlock()
	return pm.memoryPeakUsage
}

// CheckMemoryLimit checks if memory usage exceeds the specified limit
func (pm *PerformanceMonitor) CheckMemoryLimit(limitBytes uint64) bool {
	current := pm.GetCurrentMemoryUsage()
	return current > limitBytes
}

// ForceGC triggers garbage collection and waits for it to complete
func (pm *PerformanceMonitor) ForceGC() {
	runtime.GC()
	runtime.GC() // Run twice to ensure cleanup
}

// monitorMemory continuously monitors memory usage in a separate goroutine
func (pm *PerformanceMonitor) monitorMemory() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-pm.stopChan:
			return
		case <-ticker.C:
			var m runtime.MemStats
			runtime.ReadMemStats(&m)

			pm.memoryMutex.Lock()
			if m.Alloc > pm.memoryPeakUsage {
				pm.memoryPeakUsage = m.Alloc
			}
			pm.memoryMutex.Unlock()
		}
	}
}

// TimedOperation executes a function and records its execution time
func (pm *PerformanceMonitor) TimedOperation(name string, operation func() error) error {
	start := time.Now()
	pm.RecordTimestamp(name + "_start")

	err := operation()

	duration := time.Since(start)
	pm.RecordTimestamp(name + "_end")
	pm.RecordDuration(name, duration)

	return err
}

// TimedOperationWithContext executes a function with context and records timing
func (pm *PerformanceMonitor) TimedOperationWithContext(ctx context.Context, name string, operation func(context.Context) error) error {
	start := time.Now()
	pm.RecordTimestamp(name + "_start")

	err := operation(ctx)

	duration := time.Since(start)
	pm.RecordTimestamp(name + "_end")
	pm.RecordDuration(name, duration)

	return err
}

// GetExecutionTime returns the current execution time since Start() was called
func (pm *PerformanceMonitor) GetExecutionTime() time.Duration {
	if pm.startTime.IsZero() {
		return 0
	}
	return time.Since(pm.startTime)
}

// PrintSummary prints a summary of performance metrics
func (pm *PerformanceMonitor) PrintSummary(metrics *PerformanceMetrics) {
	if metrics == nil {
		fmt.Println("No performance metrics available")
		return
	}

	fmt.Println("Performance Summary:")
	fmt.Println("===================")
	fmt.Printf("Execution Time: %s\n", metrics.ExecutionTime)
	fmt.Printf("Peak Memory Usage: %.2f MB\n", metrics.PeakMemoryMB)
	fmt.Printf("Initial Memory: %.2f MB\n", float64(metrics.InitialMemory)/(1024*1024))
	fmt.Printf("Final Memory: %.2f MB\n", float64(metrics.FinalMemory)/(1024*1024))
	fmt.Printf("Memory Growth: %.2f MB\n", float64(metrics.FinalMemory-metrics.InitialMemory)/(1024*1024))
	fmt.Printf("GC Runs: %d\n", metrics.GCStats.NumGC)
	fmt.Printf("GC Pause Total: %s\n", time.Duration(metrics.GCStats.PauseTotalNs))

	if len(metrics.Durations) > 0 {
		fmt.Println("\nOperation Durations:")
		for name, duration := range metrics.Durations {
			fmt.Printf("  %s: %s\n", name, duration)
		}
	}

	if len(metrics.CustomMetrics) > 0 {
		fmt.Println("\nCustom Metrics:")
		for name, value := range metrics.CustomMetrics {
			fmt.Printf("  %s: %v\n", name, value)
		}
	}
}

// IsMemoryEfficient checks if memory usage is within acceptable limits
func (pm *PerformanceMonitor) IsMemoryEfficient(metrics *PerformanceMetrics, maxMemoryMB float64) bool {
	return metrics.PeakMemoryMB <= maxMemoryMB
}

// IsPerformant checks if execution time is within acceptable limits
func (pm *PerformanceMonitor) IsPerformant(metrics *PerformanceMetrics, maxDuration time.Duration) bool {
	return metrics.ExecutionTime <= maxDuration
}

// GetMemoryEfficiencyRatio returns the ratio of peak memory to initial memory
func (pm *PerformanceMonitor) GetMemoryEfficiencyRatio(metrics *PerformanceMetrics) float64 {
	if metrics.InitialMemory == 0 {
		return 0
	}
	return float64(metrics.PeakMemoryUsage) / float64(metrics.InitialMemory)
}

// ResourceMonitor provides system resource monitoring capabilities
type ResourceMonitor struct {
	monitor *PerformanceMonitor
	limits  ResourceLimits
}

// ResourceLimits defines resource usage limits
type ResourceLimits struct {
	MaxMemoryMB     float64       `json:"max_memory_mb"`
	MaxDuration     time.Duration `json:"max_duration"`
	MaxGCPauseTotal time.Duration `json:"max_gc_pause_total"`
}

// NewResourceMonitor creates a new resource monitor with limits
func NewResourceMonitor(limits ResourceLimits) *ResourceMonitor {
	return &ResourceMonitor{
		monitor: NewPerformanceMonitor(),
		limits:  limits,
	}
}

// Start begins resource monitoring
func (rm *ResourceMonitor) Start() {
	rm.monitor.Start()
}

// Stop ends monitoring and validates against limits
func (rm *ResourceMonitor) Stop() (*PerformanceMetrics, []string) {
	metrics := rm.monitor.Stop()
	if metrics == nil {
		return nil, []string{"Failed to collect performance metrics"}
	}

	var violations []string

	// Check memory limit
	if rm.limits.MaxMemoryMB > 0 && metrics.PeakMemoryMB > rm.limits.MaxMemoryMB {
		violations = append(violations, fmt.Sprintf(
			"Memory limit exceeded: %.2f MB > %.2f MB",
			metrics.PeakMemoryMB, rm.limits.MaxMemoryMB))
	}

	// Check duration limit
	if rm.limits.MaxDuration > 0 && metrics.ExecutionTime > rm.limits.MaxDuration {
		violations = append(violations, fmt.Sprintf(
			"Duration limit exceeded: %s > %s",
			metrics.ExecutionTime, rm.limits.MaxDuration))
	}

	// Check GC pause limit
	gcPauseTotal := time.Duration(metrics.GCStats.PauseTotalNs)
	if rm.limits.MaxGCPauseTotal > 0 && gcPauseTotal > rm.limits.MaxGCPauseTotal {
		violations = append(violations, fmt.Sprintf(
			"GC pause limit exceeded: %s > %s",
			gcPauseTotal, rm.limits.MaxGCPauseTotal))
	}

	return metrics, violations
}

// RecordMetric records a custom metric
func (rm *ResourceMonitor) RecordMetric(key string, value interface{}) {
	rm.monitor.RecordMetric(key, value)
}

// TimedOperation executes a timed operation
func (rm *ResourceMonitor) TimedOperation(name string, operation func() error) error {
	return rm.monitor.TimedOperation(name, operation)
}
