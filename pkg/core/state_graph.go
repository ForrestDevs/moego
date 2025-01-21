package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

var (
	// ErrInvalidStateType is returned when the state type is invalid
	ErrInvalidStateType = errors.New("invalid state type")

	// ErrInvalidRouterOutput is returned when a router function returns an invalid output
	ErrInvalidRouterOutput = errors.New("invalid router output")
)

// StateNode represents a node in the state graph
type StateNode[T any] struct {
	// Name is the unique identifier for the node
	Name string

	// Function is the function associated with the node
	// It takes a context and state as input and returns an updated state and error
	Function func(ctx context.Context, state T) (T, error)
}

// Router is a function that determines which node(s) to execute next
type Router[T any] func(state T) ([]string, error)

// ConditionalEdge represents a conditional edge in the state graph
type ConditionalEdge[T any] struct {
	// From is the name of the node from which the edge originates
	From string

	// Router is the function that determines which nodes to execute next
	Router Router[T]

	// Mapping optionally maps router output values to node names
	Mapping map[string]string
}

// StateGraph represents a graph with typed state
type StateGraph[T any] struct {
	// nodes is a map of node names to their corresponding StateNode objects
	nodes map[string]StateNode[T]

	// edges is a slice of ConditionalEdge objects
	edges []ConditionalEdge[T]

	// entryPoint is the name of the entry point node
	entryPoint string

	// recursionLimit is the maximum number of steps the graph can execute
	recursionLimit int

	// interruptManager handles interrupts and breakpoints
	interruptManager *InterruptManager[T]

	// streamer handles streaming of events and data
	streamer *Streamer[T]

	// streamConfig contains streaming configuration
	streamConfig StreamConfig
}

// NewStateGraph creates a new instance of StateGraph
func NewStateGraph[T any]() *StateGraph[T] {
	config := DefaultStreamConfig()
	return &StateGraph[T]{
		nodes:            make(map[string]StateNode[T]),
		recursionLimit:   25, // Default recursion limit
		interruptManager: NewInterruptManager[T](),
		streamer:         NewStreamer[T](config.Modes),
		streamConfig:     config,
	}
}

// SetStreamConfig sets the streaming configuration
func (g *StateGraph[T]) SetStreamConfig(config StreamConfig) {
	g.streamConfig = config
	g.streamer = NewStreamer[T](config.Modes)
}

// GetEventChannel returns the channel for receiving events
func (g *StateGraph[T]) GetEventChannel() <-chan Event {
	return g.streamer.GetEventChannel()
}

// GetStreamChannel returns the channel for receiving stream data
func (g *StateGraph[T]) GetStreamChannel() <-chan StreamEvent {
	return g.streamer.GetStreamChannel()
}

// AddNode adds a new node to the state graph
func (g *StateGraph[T]) AddNode(name string, fn func(ctx context.Context, state T) (T, error)) {
	g.nodes[name] = StateNode[T]{
		Name:     name,
		Function: fn,
	}
}

// AddConditionalEdges adds conditional edges from a node using a router function
func (g *StateGraph[T]) AddConditionalEdges(from string, router Router[T], mapping map[string]string) {
	g.edges = append(g.edges, ConditionalEdge[T]{
		From:    from,
		Router:  router,
		Mapping: mapping,
	})
}

// SetEntryPoint sets the entry point node
func (g *StateGraph[T]) SetEntryPoint(name string) {
	g.entryPoint = name
}

// SetRecursionLimit sets the maximum number of steps the graph can execute
func (g *StateGraph[T]) SetRecursionLimit(limit int) {
	g.recursionLimit = limit
}

// AddBreakpoint adds a breakpoint at the specified node
func (g *StateGraph[T]) AddBreakpoint(nodeName string) {
	g.interruptManager.AddBreakpoint(nodeName)
}

// RemoveBreakpoint removes a breakpoint from the specified node
func (g *StateGraph[T]) RemoveBreakpoint(nodeName string) {
	g.interruptManager.RemoveBreakpoint(nodeName)
}

// GetInterruptChannel returns the channel for receiving interrupt info
func (g *StateGraph[T]) GetInterruptChannel() <-chan InterruptInfo {
	return g.interruptManager.GetInterruptChannel()
}

// Resume resumes graph execution with the provided state
func (g *StateGraph[T]) Resume(state T) error {
	return g.interruptManager.Resume(state)
}

// RunnableState represents a compiled state graph that can be invoked
type RunnableState[T any] struct {
	graph *StateGraph[T]
}

// Compile compiles the state graph and returns a RunnableState instance
func (g *StateGraph[T]) Compile() (*RunnableState[T], error) {
	if g.entryPoint == "" {
		return nil, ErrEntryPointNotSet
	}

	return &RunnableState[T]{
		graph: g,
	}, nil
}

// Send represents a message to be sent to a specific node with custom state
type Send[T any] struct {
	Node  string
	State T
}

// Command represents a combination of state update and routing instruction
type Command[T any] struct {
	Update T
	Goto   string
}

// Invoke executes the compiled state graph with the given input state
func (r *RunnableState[T]) Invoke(ctx context.Context, state T) (T, error) {
	currentNode := r.graph.entryPoint
	steps := 0

	// Emit initial state
	r.graph.streamer.EmitValue(state)
	r.graph.streamer.EmitEvent(Event{
		Type:      EventChainStart,
		Name:      "LangGraph",
		RunID:     "run-" + time.Now().Format("20060102150405"),
		Timestamp: time.Now(),
	})

	for {
		if steps >= r.graph.recursionLimit {
			var zero T
			return zero, fmt.Errorf("recursion limit (%d) exceeded", r.graph.recursionLimit)
		}

		if currentNode == END {
			break
		}

		// Check for breakpoints
		if r.graph.interruptManager.HasBreakpoint(currentNode) {
			if err := r.graph.interruptManager.Interrupt(currentNode, nil, state); err != nil {
				var zero T
				return zero, fmt.Errorf("error triggering breakpoint: %w", err)
			}

			var err error
			state, err = r.graph.interruptManager.WaitForResume(ctx)
			if err != nil {
				var zero T
				return zero, fmt.Errorf("error waiting for resume: %w", err)
			}
		}

		node, ok := r.graph.nodes[currentNode]
		if !ok {
			var zero T
			return zero, fmt.Errorf("%w: %s", ErrNodeNotFound, currentNode)
		}

		// Emit node start event
		r.graph.streamer.EmitEvent(Event{
			Type:      EventChainStart,
			Name:      currentNode,
			RunID:     "run-" + time.Now().Format("20060102150405"),
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"langgraph_step": steps,
				"langgraph_node": currentNode,
			},
		})

		var err error
		state, err = node.Function(ctx, state)
		if err != nil {
			// Check for interrupt requests
			if IsInterruptError(err) {
				data, _ := GetInterruptData(err)
				if err := r.graph.interruptManager.Interrupt(currentNode, data, state); err != nil {
					var zero T
					return zero, fmt.Errorf("error triggering interrupt: %w", err)
				}

				state, err = r.graph.interruptManager.WaitForResume(ctx)
				if err != nil {
					var zero T
					return zero, fmt.Errorf("error waiting for resume: %w", err)
				}
				continue
			}

			var zero T
			return zero, fmt.Errorf("error in node %s: %w", currentNode, err)
		}

		// Emit node end event and state update
		r.graph.streamer.EmitEvent(Event{
			Type:      EventChainEnd,
			Name:      currentNode,
			RunID:     "run-" + time.Now().Format("20060102150405"),
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"langgraph_step": steps,
				"langgraph_node": currentNode,
			},
		})
		r.graph.streamer.EmitUpdate(state)

		// Find and execute the router for the current node
		foundNext := false
		for _, edge := range r.graph.edges {
			if edge.From == currentNode {
				nextNodes, err := edge.Router(state)
				if err != nil {
					var zero T
					return zero, fmt.Errorf("error in router for node %s: %w", currentNode, err)
				}

				if len(nextNodes) == 0 {
					var zero T
					return zero, fmt.Errorf("%w: router returned no nodes", ErrInvalidRouterOutput)
				}

				// If mapping exists, translate the router output
				if edge.Mapping != nil {
					mappedNodes := make([]string, 0, len(nextNodes))
					for _, node := range nextNodes {
						if mapped, ok := edge.Mapping[node]; ok {
							mappedNodes = append(mappedNodes, mapped)
						} else {
							mappedNodes = append(mappedNodes, node)
						}
					}
					nextNodes = mappedNodes
				}

				// For now, just take the first node. In future we could support parallel execution
				currentNode = nextNodes[0]
				foundNext = true
				break
			}
		}

		if !foundNext {
			var zero T
			return zero, fmt.Errorf("%w: %s", ErrNoOutgoingEdge, currentNode)
		}

		steps++
	}

	// Emit final state and end event
	r.graph.streamer.EmitValue(state)
	r.graph.streamer.EmitEvent(Event{
		Type:      EventChainEnd,
		Name:      "LangGraph",
		RunID:     "run-" + time.Now().Format("20060102150405"),
		Timestamp: time.Now(),
	})

	return state, nil
}

// Stream executes the graph and returns channels for streaming results
func (r *RunnableState[T]) Stream(ctx context.Context, state T) (<-chan StreamEvent, <-chan Event, error) {
	// Create channels for streaming
	streamCh := make(chan StreamEvent, r.graph.streamConfig.BufferSize)
	eventCh := make(chan Event, r.graph.streamConfig.BufferSize)

	// Run the graph in a goroutine
	go func() {
		defer close(streamCh)
		defer close(eventCh)

		// Create a new context with cancellation
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		// Create a goroutine to forward events and stream data
		go func() {
			for {
				select {
				case evt, ok := <-r.graph.GetEventChannel():
					if !ok {
						return
					}
					select {
					case eventCh <- evt:
					case <-ctx.Done():
						return
					}
				case stream, ok := <-r.graph.GetStreamChannel():
					if !ok {
						return
					}
					select {
					case streamCh <- stream:
					case <-ctx.Done():
						return
					}
				case <-ctx.Done():
					return
				}
			}
		}()

		// Run the graph
		_, err := r.Invoke(ctx, state)
		if err != nil {
			// Handle error
			select {
			case eventCh <- Event{
				Type:      EventChainEnd,
				Name:      "LangGraph",
				RunID:     "run-" + time.Now().Format("20060102150405"),
				Timestamp: time.Now(),
				Metadata: map[string]interface{}{
					"error": err.Error(),
				},
			}:
			case <-ctx.Done():
			}
		}
	}()

	return streamCh, eventCh, nil
}

// MarshalState marshals a state object to JSON
func MarshalState[T any](state T) ([]byte, error) {
	return json.Marshal(state)
}

// UnmarshalState unmarshals JSON into a state object
func UnmarshalState[T any](data []byte) (T, error) {
	var state T
	err := json.Unmarshal(data, &state)
	if err != nil {
		var zero T
		return zero, err
	}
	return state, nil
}
