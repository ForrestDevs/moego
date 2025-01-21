package core

import (
	"encoding/json"
	"time"
)

// StreamMode represents different modes of streaming graph execution
type StreamMode string

const (
	// StreamValues streams the full state after each step
	StreamValues StreamMode = "values"

	// StreamUpdates streams state updates after each step
	StreamUpdates StreamMode = "updates"

	// StreamCustom streams custom data from nodes
	StreamCustom StreamMode = "custom"

	// StreamMessages streams LLM tokens and metadata
	StreamMessages StreamMode = "messages"

	// StreamDebug streams all possible information
	StreamDebug StreamMode = "debug"
)

// EventType represents different types of events that can be emitted
type EventType string

const (
	// EventChainStart emitted when a chain/node starts
	EventChainStart EventType = "on_chain_start"

	// EventChainEnd emitted when a chain/node ends
	EventChainEnd EventType = "on_chain_end"

	// EventChainStream emitted during chain/node execution
	EventChainStream EventType = "on_chain_stream"

	// EventChatModelStart emitted when chat model starts
	EventChatModelStart EventType = "on_chat_model_start"

	// EventChatModelStream emitted during chat model generation
	EventChatModelStream EventType = "on_chat_model_stream"

	// EventChatModelEnd emitted when chat model ends
	EventChatModelEnd EventType = "on_chat_model_end"

	// EventChannelWrite emitted when writing to a state channel
	EventChannelWrite EventType = "on_channel_write"
)

// Event represents a streaming event
type Event struct {
	// Type is the type of event
	Type EventType `json:"event"`

	// Name is the name of the component that emitted the event
	Name string `json:"name"`

	// RunID is a unique identifier for this run
	RunID string `json:"run_id"`

	// Tags are optional tags associated with the event
	Tags []string `json:"tags,omitempty"`

	// Metadata contains additional information about the event
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// Data contains the event payload
	Data json.RawMessage `json:"data,omitempty"`

	// ParentIDs contains IDs of parent events
	ParentIDs []string `json:"parent_ids,omitempty"`

	// Timestamp is when the event occurred
	Timestamp time.Time `json:"timestamp"`
}

// StreamEvent represents a stream event with its mode
type StreamEvent struct {
	Mode StreamMode
	Data interface{}
}

// Streamer manages streaming for a graph
type Streamer[T any] struct {
	// modes are the active streaming modes
	modes []StreamMode

	// eventCh is the channel for streaming events
	eventCh chan Event

	// streamCh is the channel for streaming data
	streamCh chan StreamEvent
}

// NewStreamer creates a new streamer with the specified modes
func NewStreamer[T any](modes []StreamMode) *Streamer[T] {
	return &Streamer[T]{
		modes:    modes,
		eventCh:  make(chan Event),
		streamCh: make(chan StreamEvent),
	}
}

// EmitEvent emits an event to the event stream
func (s *Streamer[T]) EmitEvent(evt Event) {
	if s.hasMode(StreamDebug) {
		s.eventCh <- evt
	}
}

// EmitValue emits a state value to the stream
func (s *Streamer[T]) EmitValue(state T) {
	if s.hasMode(StreamValues) {
		s.streamCh <- StreamEvent{
			Mode: StreamValues,
			Data: state,
		}
	}
}

// EmitUpdate emits a state update to the stream
func (s *Streamer[T]) EmitUpdate(update T) {
	if s.hasMode(StreamUpdates) {
		s.streamCh <- StreamEvent{
			Mode: StreamUpdates,
			Data: update,
		}
	}
}

// EmitCustom emits custom data to the stream
func (s *Streamer[T]) EmitCustom(data T) {
	if s.hasMode(StreamCustom) {
		s.streamCh <- StreamEvent{
			Mode: StreamCustom,
			Data: data,
		}
	}
}

// EmitMessage emits an LLM message to the stream
func (s *Streamer[T]) EmitMessage(msg T) {
	if s.hasMode(StreamMessages) {
		s.streamCh <- StreamEvent{
			Mode: StreamMessages,
			Data: msg,
		}
	}
}

// GetEventChannel returns the event channel
func (s *Streamer[T]) GetEventChannel() <-chan Event {
	return s.eventCh
}

// GetStreamChannel returns the stream channel
func (s *Streamer[T]) GetStreamChannel() <-chan StreamEvent {
	return s.streamCh
}

// hasMode checks if a mode is active
func (s *Streamer[T]) hasMode(mode StreamMode) bool {
	for _, m := range s.modes {
		if m == mode {
			return true
		}
	}
	return false
}

// Close closes all channels
func (s *Streamer[T]) Close() {
	close(s.eventCh)
	close(s.streamCh)
}

// StreamConfig contains configuration for streaming
type StreamConfig struct {
	// Modes are the active streaming modes
	Modes []StreamMode

	// BufferSize is the size of the stream channels
	BufferSize int
}

// DefaultStreamConfig returns the default streaming configuration
func DefaultStreamConfig() StreamConfig {
	return StreamConfig{
		Modes:      []StreamMode{StreamValues},
		BufferSize: 100,
	}
}
