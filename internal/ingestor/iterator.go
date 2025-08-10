package ingestor

import (
	"context"
	"sync"
)

// Iterator provides a generic interface for streaming data processing
type Iterator[T any] interface {
	// Next advances the iterator to the next item
	Next() bool
	// Value returns the current item
	Value() T
	// Err returns any error that occurred during iteration
	Err() error
	// Close releases any resources held by the iterator
	Close() error
}

// ChannelIterator implements Iterator using a channel-based approach with backpressure control
type ChannelIterator[T any] struct {
	ch       <-chan T
	errCh    <-chan error
	current  T
	err      error
	closed   bool
	ctx      context.Context
	cancel   context.CancelFunc
	mu       sync.RWMutex
}

// NewChannelIterator creates a new channel-based iterator with backpressure control
func NewChannelIterator[T any](bufferSize int) (*ChannelIterator[T], chan<- T, chan<- error) {
	ctx, cancel := context.WithCancel(context.Background())
	
	dataCh := make(chan T, bufferSize)
	errCh := make(chan error, 1)
	
	iterator := &ChannelIterator[T]{
		ch:     dataCh,
		errCh:  errCh,
		ctx:    ctx,
		cancel: cancel,
	}
	
	return iterator, dataCh, errCh
}

// Next advances the iterator to the next item
func (c *ChannelIterator[T]) Next() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if c.closed || c.err != nil {
		return false
	}
	
	select {
	case item, ok := <-c.ch:
		if !ok {
			c.closed = true
			return false
		}
		c.current = item
		return true
	case err := <-c.errCh:
		c.err = err
		return false
	case <-c.ctx.Done():
		c.err = c.ctx.Err()
		return false
	}
}

// Value returns the current item
func (c *ChannelIterator[T]) Value() T {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.current
}

// Err returns any error that occurred during iteration
func (c *ChannelIterator[T]) Err() error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.err
}

// Close releases any resources held by the iterator
func (c *ChannelIterator[T]) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if !c.closed {
		c.cancel()
		c.closed = true
	}
	return nil
}

// SliceIterator implements Iterator for in-memory slices
type SliceIterator[T any] struct {
	items   []T
	index   int
	current T
}

// NewSliceIterator creates a new slice-based iterator
func NewSliceIterator[T any](items []T) *SliceIterator[T] {
	return &SliceIterator[T]{
		items: items,
		index: -1,
	}
}

// Next advances the iterator to the next item
func (s *SliceIterator[T]) Next() bool {
	s.index++
	if s.index >= len(s.items) {
		return false
	}
	s.current = s.items[s.index]
	return true
}

// Value returns the current item
func (s *SliceIterator[T]) Value() T {
	return s.current
}

// Err returns any error that occurred during iteration
func (s *SliceIterator[T]) Err() error {
	return nil
}

// Close releases any resources held by the iterator
func (s *SliceIterator[T]) Close() error {
	return nil
}