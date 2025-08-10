package ingestor

import (
	"context"
	"testing"
	"time"
)

func TestSliceIterator(t *testing.T) {
	items := []string{"a", "b", "c"}
	iter := NewSliceIterator(items)
	defer iter.Close()

	var result []string
	for iter.Next() {
		result = append(result, iter.Value())
	}

	if len(result) != 3 {
		t.Errorf("Expected 3 items, got %d", len(result))
	}

	expected := []string{"a", "b", "c"}
	for i, v := range result {
		if v != expected[i] {
			t.Errorf("Expected %s at index %d, got %s", expected[i], i, v)
		}
	}

	if iter.Err() != nil {
		t.Errorf("Expected no error, got %v", iter.Err())
	}
}

func TestChannelIterator(t *testing.T) {
	iter, dataCh, errCh := NewChannelIterator[string](10)
	defer iter.Close()

	// Send data in a goroutine
	go func() {
		defer close(dataCh)
		dataCh <- "item1"
		dataCh <- "item2"
		dataCh <- "item3"
	}()

	var result []string
	for iter.Next() {
		result = append(result, iter.Value())
	}

	if len(result) != 3 {
		t.Errorf("Expected 3 items, got %d", len(result))
	}

	expected := []string{"item1", "item2", "item3"}
	for i, v := range result {
		if v != expected[i] {
			t.Errorf("Expected %s at index %d, got %s", expected[i], i, v)
		}
	}

	if iter.Err() != nil {
		t.Errorf("Expected no error, got %v", iter.Err())
	}

	// Ensure errCh is closed to prevent goroutine leak
	close(errCh)
}

func TestChannelIteratorWithError(t *testing.T) {
	iter, dataCh, errCh := NewChannelIterator[string](10)
	defer iter.Close()

	// Send error in a goroutine
	go func() {
		defer close(dataCh)
		defer close(errCh)
		dataCh <- "item1"
		errCh <- context.DeadlineExceeded
	}()

	var result []string
	for iter.Next() {
		result = append(result, iter.Value())
	}

	if len(result) != 1 {
		t.Errorf("Expected 1 item before error, got %d", len(result))
	}

	if iter.Err() == nil {
		t.Error("Expected error, got nil")
	}
}

func TestChannelIteratorBackpressure(t *testing.T) {
	// Small buffer to test backpressure
	iter, dataCh, errCh := NewChannelIterator[int](2)
	defer iter.Close()

	// Send more data than buffer can hold
	go func() {
		defer close(dataCh)
		defer close(errCh)
		for i := 0; i < 5; i++ {
			select {
			case dataCh <- i:
				// Successfully sent
			case <-time.After(100 * time.Millisecond):
				// This tests that backpressure is working
				return
			}
		}
	}()

	// Read with delay to test backpressure
	var result []int
	for iter.Next() {
		result = append(result, iter.Value())
		time.Sleep(50 * time.Millisecond) // Simulate slow consumer
	}

	// Should have received some items
	if len(result) == 0 {
		t.Error("Expected to receive some items")
	}

	if iter.Err() != nil {
		t.Errorf("Expected no error, got %v", iter.Err())
	}
}

func TestChannelIteratorClose(t *testing.T) {
	iter, dataCh, errCh := NewChannelIterator[string](10)

	// Close immediately
	err := iter.Close()
	if err != nil {
		t.Errorf("Expected no error on close, got %v", err)
	}

	// Should not advance after close
	if iter.Next() {
		t.Error("Expected Next() to return false after close")
	}

	// Clean up channels
	close(dataCh)
	close(errCh)
}