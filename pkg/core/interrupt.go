package core

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
)

var (
	// ErrInterrupted is returned when graph execution is interrupted
	ErrInterrupted = errors.New("graph execution interrupted")

	// ErrInvalidResume is returned when trying to resume with invalid data
	ErrInvalidResume = errors.New("invalid resume data")
)

// InterruptInfo contains information about an interrupt
type InterruptInfo struct {
	// NodeName is the name of the node that triggered the interrupt
	NodeName string `json:"node_name"`

	// Data is arbitrary data passed to the client
	Data json.RawMessage `json:"data"`

	// State is the current state of the graph
	State json.RawMessage `json:"state"`
}

// InterruptManager manages interrupts and breakpoints
type InterruptManager[T any] struct {
	mu sync.Mutex

	// interrupted indicates if execution is currently interrupted
	interrupted bool

	// interruptCh is used to send interrupt info to clients
	interruptCh chan InterruptInfo

	// resumeCh is used to receive resume data from clients
	resumeCh chan T

	// breakpoints is a set of node names where execution should pause
	breakpoints map[string]struct{}
}

// NewInterruptManager creates a new interrupt manager
func NewInterruptManager[T any]() *InterruptManager[T] {
	return &InterruptManager[T]{
		interruptCh: make(chan InterruptInfo),
		resumeCh:    make(chan T),
		breakpoints: make(map[string]struct{}),
	}
}

// AddBreakpoint adds a breakpoint at the specified node
func (m *InterruptManager[T]) AddBreakpoint(nodeName string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.breakpoints[nodeName] = struct{}{}
}

// RemoveBreakpoint removes a breakpoint from the specified node
func (m *InterruptManager[T]) RemoveBreakpoint(nodeName string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.breakpoints, nodeName)
}

// HasBreakpoint checks if a node has a breakpoint
func (m *InterruptManager[T]) HasBreakpoint(nodeName string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.breakpoints[nodeName]
	return ok
}

// Interrupt pauses graph execution and sends interrupt info to clients
func (m *InterruptManager[T]) Interrupt(nodeName string, data interface{}, state T) error {
	m.mu.Lock()
	if m.interrupted {
		m.mu.Unlock()
		return errors.New("already interrupted")
	}
	m.interrupted = true
	m.mu.Unlock()

	dataBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}

	stateBytes, err := json.Marshal(state)
	if err != nil {
		return err
	}

	info := InterruptInfo{
		NodeName: nodeName,
		Data:     dataBytes,
		State:    stateBytes,
	}

	m.interruptCh <- info
	return nil
}

// Resume resumes graph execution with the provided state
func (m *InterruptManager[T]) Resume(state T) error {
	m.mu.Lock()
	if !m.interrupted {
		m.mu.Unlock()
		return errors.New("not interrupted")
	}
	m.interrupted = false
	m.mu.Unlock()

	m.resumeCh <- state
	return nil
}

// WaitForResume waits for the client to resume execution
func (m *InterruptManager[T]) WaitForResume(ctx context.Context) (T, error) {
	select {
	case state := <-m.resumeCh:
		return state, nil
	case <-ctx.Done():
		var zero T
		return zero, ctx.Err()
	}
}

// GetInterruptChannel returns the channel for receiving interrupt info
func (m *InterruptManager[T]) GetInterruptChannel() <-chan InterruptInfo {
	return m.interruptCh
}

// Interrupt is a helper function that can be used in node functions to trigger an interrupt
func Interrupt[T any](ctx context.Context, data interface{}) (T, error) {
	var zero T
	return zero, &InterruptError{Data: data}
}

// InterruptError is returned when a node triggers an interrupt
type InterruptError struct {
	Data interface{} `json:"data"`
}

func (e *InterruptError) Error() string {
	return "interrupt requested"
}

// IsInterruptError checks if an error is an InterruptError
func IsInterruptError(err error) bool {
	_, ok := err.(*InterruptError)
	return ok
}

// GetInterruptData extracts data from an InterruptError
func GetInterruptData(err error) (interface{}, bool) {
	if ierr, ok := err.(*InterruptError); ok {
		return ierr.Data, true
	}
	return nil, false
}
