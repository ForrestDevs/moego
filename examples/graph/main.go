package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/forrestdevs/moego/pkg/agent"
	"github.com/forrestdevs/moego/pkg/core"
	"github.com/forrestdevs/moego/pkg/tools"
	dotenv "github.com/joho/godotenv"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// State represents our graph state
type State struct {
	Messages []core.Message `json:"messages"`
	Result   float64        `json:"result,omitempty"`
	Poem     string         `json:"poem,omitempty"`
}

func main() {
	// Load .env file
	if err := dotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found: %v", err)
	}

	// Initialize logger
	config := zap.NewDevelopmentConfig()
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
	config.Development = true

	logger, err := config.Build()
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Sync()

	// Get OpenAI API key from environment
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		logger.Fatal("OPENAI_API_KEY environment variable is required")
	}

	// Create agents
	mathExpert := agent.NewOpenAIAgent("math_expert", apiKey, logger)
	mathExpert.AddTool(tools.NewCalculator())
	mathExpert.Configure(map[string]interface{}{
		"model": "gpt-4o-mini",
		"system_message": "You are a math expert. Use the calculator tool to perform calculations. " +
			"Always explain your reasoning and show your work.",
	})

	poet := agent.NewOpenAIAgent("poet", apiKey, logger)
	poet.Configure(map[string]interface{}{
		"model": "gpt-4o-mini",
		"system_message": "You are a creative poet. When given a number, create a beautiful and " +
			"meaningful poem that incorporates that number in a creative way.",
	})

	// Create the graph
	graph := core.NewStateGraph[State]()

	// Configure streaming
	graph.SetStreamConfig(core.StreamConfig{
		Modes: []core.StreamMode{
			core.StreamValues,
			core.StreamMessages,
			core.StreamDebug,
		},
		BufferSize: 100,
	})

	// Add nodes
	graph.AddNode("calculate", func(ctx context.Context, state State) (State, error) {
		// Create a message for the math expert
		msg := core.Message{
			Role:    core.RoleUser,
			Content: "Calculate the sum of squares from 1 to 5 (1² + 2² + 3² + 4² + 5²)",
		}
		state.Messages = append(state.Messages, msg)

		// Get response from math expert
		responses, err := mathExpert.ProcessMessage(ctx, msg)
		if err != nil {
			return state, fmt.Errorf("math expert error: %w", err)
		}

		// Add responses to state
		state.Messages = append(state.Messages, responses...)

		// Extract the result from the last message
		lastMsg := responses[len(responses)-1]
		if lastMsg.Role == core.RoleAssistant {
			// Parse the result from the message
			// In a real implementation, you'd want to parse this more robustly
			state.Result = 55 // 1² + 2² + 3² + 4² + 5² = 1 + 4 + 9 + 16 + 25 = 55
		}

		return state, nil
	})

	graph.AddNode("write_poem", func(ctx context.Context, state State) (State, error) {
		// Create a message for the poet
		msg := core.Message{
			Role:    core.RoleUser,
			Content: fmt.Sprintf("Create a short, beautiful poem that incorporates the number %v", state.Result),
		}
		state.Messages = append(state.Messages, msg)

		// Get response from poet
		responses, err := poet.ProcessMessage(ctx, msg)
		if err != nil {
			return state, fmt.Errorf("poet error: %w", err)
		}

		// Add responses to state
		state.Messages = append(state.Messages, responses...)

		// Extract the poem from the last message
		lastMsg := responses[len(responses)-1]
		if lastMsg.Role == core.RoleAssistant {
			state.Poem = lastMsg.Content
		}

		return state, nil
	})

	// Add edges with simple routers
	graph.AddConditionalEdges(core.START, func(state State) ([]string, error) {
		return []string{"calculate"}, nil
	}, nil)
	graph.AddConditionalEdges("calculate", func(state State) ([]string, error) {
		return []string{"write_poem"}, nil
	}, nil)
	graph.AddConditionalEdges("write_poem", func(state State) ([]string, error) {
		return []string{core.END}, nil
	}, nil)

	// Set entry point
	graph.SetEntryPoint("calculate")

	// Compile the graph
	runnable, err := graph.Compile()
	if err != nil {
		logger.Fatal("Failed to compile graph", zap.Error(err))
	}

	// Create channels for streaming
	ctx := context.Background()
	streamCh, eventCh, err := runnable.Stream(ctx, State{
		Messages: []core.Message{},
	})
	if err != nil {
		logger.Fatal("Failed to start streaming", zap.Error(err))
	}

	// Handle streams and wait for completion
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			select {
			case evt, ok := <-eventCh:
				if !ok {
					return
				}
				logger.Info("Event",
					zap.String("type", string(evt.Type)),
					zap.String("name", evt.Name),
					zap.Any("metadata", evt.Metadata))

				// Check for completion
				if evt.Type == core.EventChainEnd && evt.Name == "LangGraph" {
					return
				}

			case stream, ok := <-streamCh:
				if !ok {
					return
				}
				switch stream.Mode {
				case core.StreamValues:
					if state, ok := stream.Data.(State); ok {
						if state.Result != 0 {
							logger.Info("Calculation result", zap.Float64("result", state.Result))
						}
						if state.Poem != "" {
							logger.Info("Generated poem", zap.String("poem", state.Poem))
						}
					}
				case core.StreamMessages:
					if msg, ok := stream.Data.(core.Message); ok {
						logger.Info("Message",
							zap.String("role", string(msg.Role)),
							zap.String("content", msg.Content))
					}
				}

			case <-ctx.Done():
				return
			}
		}
	}()

	// Wait for completion
	select {
	case <-ctx.Done():
		logger.Info("Context cancelled")
	case <-done:
		logger.Info("Graph execution completed")
	}
}
