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

package ingestor

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewSliceIterator(t *testing.T) {
	data := []string{"a", "b", "c"}
	iterator := NewSliceIterator(data)

	assert.NotNil(t, iterator)
	
	// Test iteration
	var results []string
	for iterator.Next() {
		results = append(results, iterator.Value())
	}
	
	assert.NoError(t, iterator.Err())
	assert.Equal(t, data, results)
	
	// Test Close
	assert.NoError(t, iterator.Close())
}

func TestSliceIterator_EmptySlice(t *testing.T) {
	var data []string
	iterator := NewSliceIterator(data)

	assert.NotNil(t, iterator)
	assert.False(t, iterator.Next())
	assert.NoError(t, iterator.Err())
	assert.NoError(t, iterator.Close())
}

func TestSliceIterator_SingleElement(t *testing.T) {
	data := []string{"single"}
	iterator := NewSliceIterator(data)

	// First call should return true
	assert.True(t, iterator.Next())
	assert.Equal(t, "single", iterator.Value())
	
	// Second call should return false
	assert.False(t, iterator.Next())
	assert.NoError(t, iterator.Err())
}

func TestSliceIterator_MultipleIterations(t *testing.T) {
	data := []int{1, 2, 3, 4, 5}
	iterator := NewSliceIterator(data)

	// Iterate through all elements
	for i, expected := range data {
		assert.True(t, iterator.Next(), "Next() should return true for element %d", i)
		assert.Equal(t, expected, iterator.Value(), "Value should match for element %d", i)
	}
	
	// Should be exhausted now
	assert.False(t, iterator.Next())
	assert.NoError(t, iterator.Err())
}

func TestNewChannelIterator(t *testing.T) {
	bufferSize := 10
	iterator, dataCh, errCh := NewChannelIterator[string](bufferSize)

	assert.NotNil(t, iterator)
	assert.NotNil(t, dataCh)
	assert.NotNil(t, errCh)

	// Test sending data
	go func() {
		dataCh <- "item1"
		dataCh <- "item2"
		dataCh <- "item3"
		close(dataCh)
	}()

	// Test iteration
	var results []string
	for iterator.Next() {
		results = append(results, iterator.Value())
	}

	assert.NoError(t, iterator.Err())
	assert.Equal(t, []string{"item1", "item2", "item3"}, results)
	assert.NoError(t, iterator.Close())
}

func TestChannelIterator_WithError(t *testing.T) {
	iterator, dataCh, errCh := NewChannelIterator[string](10)

	testError := errors.New("test error")

	// Send error
	go func() {
		dataCh <- "item1"
		time.Sleep(10 * time.Millisecond) // Small delay to ensure order
		errCh <- testError
		close(dataCh)
	}()

	// Should get first item
	assert.True(t, iterator.Next())
	assert.Equal(t, "item1", iterator.Value())

	// Should stop on error
	assert.False(t, iterator.Next())
	assert.Error(t, iterator.Err())
	assert.Equal(t, testError, iterator.Err())
}

func TestChannelIterator_EmptyChannel(t *testing.T) {
	iterator, dataCh, _ := NewChannelIterator[string](10)

	// Close channel immediately
	go func() {
		close(dataCh)
	}()

	// Should not iterate
	assert.False(t, iterator.Next())
	assert.NoError(t, iterator.Err())
	assert.NoError(t, iterator.Close())

	// Note: errCh is send-only, so we can't test receiving from it directly
	// This is expected behavior for the channel iterator design
}

func TestChannelIterator_BackpressureControl(t *testing.T) {
	bufferSize := 2
	iterator, dataCh, _ := NewChannelIterator[int](bufferSize)

	// Send more items than buffer size
	go func() {
		defer close(dataCh)
		for i := 0; i < 5; i++ {
			select {
			case dataCh <- i:
				// Successfully sent
			case <-time.After(100 * time.Millisecond):
				// This should happen due to backpressure
				return
			}
		}
	}()

	// Read items slowly to test backpressure
	var results []int
	for iterator.Next() {
		results = append(results, iterator.Value())
		time.Sleep(10 * time.Millisecond) // Slow consumption
	}

	assert.NoError(t, iterator.Err())
	// Should have received at least the buffered items
	assert.GreaterOrEqual(t, len(results), bufferSize)
	assert.NoError(t, iterator.Close())

	// Note: errCh is send-only, so we can't clean it up directly
}

func TestChannelIterator_ConcurrentAccess(t *testing.T) {
	iterator, dataCh, _ := NewChannelIterator[int](100)

	// Producer goroutine
	go func() {
		defer close(dataCh)
		for i := 0; i < 100; i++ {
			dataCh <- i
		}
	}()

	// Consumer
	var results []int
	for iterator.Next() {
		results = append(results, iterator.Value())
	}

	assert.NoError(t, iterator.Err())
	assert.Len(t, results, 100)
	
	// Verify all numbers are present (order should be preserved)
	for i, value := range results {
		assert.Equal(t, i, value)
	}

	assert.NoError(t, iterator.Close())

	// Note: errCh is send-only, so we can't clean it up directly
}

func TestChannelIterator_ErrorAfterData(t *testing.T) {
	iterator, dataCh, errCh := NewChannelIterator[string](10)

	testError := errors.New("error after data")

	// Send data then error
	go func() {
		dataCh <- "item1"
		dataCh <- "item2"
		errCh <- testError
		close(dataCh)
	}()

	// Should get both items
	assert.True(t, iterator.Next())
	assert.Equal(t, "item1", iterator.Value())
	
	assert.True(t, iterator.Next())
	assert.Equal(t, "item2", iterator.Value())

	// Should stop on error
	assert.False(t, iterator.Next())
	assert.Error(t, iterator.Err())
	assert.Equal(t, testError, iterator.Err())
}

func TestChannelIterator_MultipleErrors(t *testing.T) {
	iterator, dataCh, errCh := NewChannelIterator[string](10)

	firstError := errors.New("first error")
	secondError := errors.New("second error")

	// Send multiple errors
	go func() {
		errCh <- firstError
		errCh <- secondError // This should be ignored
		close(dataCh)
	}()

	// Should stop on first error
	assert.False(t, iterator.Next())
	assert.Error(t, iterator.Err())
	assert.Equal(t, firstError, iterator.Err()) // Should be the first error
}

func TestChannelIterator_CloseWhileIterating(t *testing.T) {
	iterator, dataCh, _ := NewChannelIterator[int](10)

	// Send some data
	go func() {
		dataCh <- 1
		dataCh <- 2
		// Don't close the channel, simulate ongoing operation
		time.Sleep(100 * time.Millisecond)
		dataCh <- 3
		close(dataCh)
	}()

	// Get first item
	assert.True(t, iterator.Next())
	assert.Equal(t, 1, iterator.Value())

	// Close iterator while it's still receiving data
	assert.NoError(t, iterator.Close())

	// Further iterations should not work (implementation dependent)
	// The exact behavior may vary, but it shouldn't panic

	// Note: errCh is send-only, so we can't clean it up directly
}

func TestIterator_InterfaceCompliance(t *testing.T) {
	// Test that both implementations satisfy the Iterator interface
	var _ Iterator[string] = NewSliceIterator([]string{"test"})
	
	iterator, dataCh, _ := NewChannelIterator[string](10)
	var _ Iterator[string] = iterator
	
	// Clean up
	close(dataCh)
	iterator.Close()
}

func TestChannelIterator_LargeDataSet(t *testing.T) {
	const dataSize = 10000
	iterator, dataCh, _ := NewChannelIterator[int](100)

	// Producer goroutine
	go func() {
		defer close(dataCh)
		for i := 0; i < dataSize; i++ {
			dataCh <- i
		}
	}()

	// Consumer
	count := 0
	for iterator.Next() {
		value := iterator.Value()
		assert.Equal(t, count, value)
		count++
	}

	assert.NoError(t, iterator.Err())
	assert.Equal(t, dataSize, count)
	assert.NoError(t, iterator.Close())

	// Note: errCh is send-only, so we can't clean it up directly
}

func TestSliceIterator_ValueBeforeNext(t *testing.T) {
	data := []string{"a", "b", "c"}
	iterator := NewSliceIterator(data)

	// Calling Value() before Next() should return zero value
	// This behavior is implementation-dependent, but shouldn't panic
	value := iterator.Value()
	assert.Equal(t, "", value) // Zero value for string

	// Normal iteration should still work
	assert.True(t, iterator.Next())
	assert.Equal(t, "a", iterator.Value())
}

func TestChannelIterator_ValueBeforeNext(t *testing.T) {
	iterator, dataCh, _ := NewChannelIterator[string](10)

	// Calling Value() before Next() should return zero value
	value := iterator.Value()
	assert.Equal(t, "", value) // Zero value for string

	// Send data and test normal iteration
	go func() {
		dataCh <- "test"
		close(dataCh)
	}()

	assert.True(t, iterator.Next())
	assert.Equal(t, "test", iterator.Value())
	assert.NoError(t, iterator.Close())
}