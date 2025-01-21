package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/forrestdevs/moego/pkg/core"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/shared"
	"go.uber.org/zap"
)

type OpenAIAgent struct {
	id      string
	client  *openai.Client
	logger  *zap.Logger
	config  map[string]interface{}
	tools   []core.Tool
	history []openai.ChatCompletionMessageParamUnion
}

func NewOpenAIAgent(id string, apiKey string, logger *zap.Logger) Agent {
	client := openai.NewClient(
		option.WithAPIKey(apiKey),
	)

	return &OpenAIAgent{
		id:      id,
		client:  client,
		logger:  logger.With(zap.String("agent_id", id)),
		config:  make(map[string]interface{}),
		tools:   make([]core.Tool, 0),
		history: make([]openai.ChatCompletionMessageParamUnion, 0),
	}
}

func (a *OpenAIAgent) ID() string {
	return a.id
}

func (a *OpenAIAgent) Configure(config map[string]interface{}) error {
	if model, ok := config["model"].(string); !ok {
		return fmt.Errorf("model must be a string")
	} else {
		a.config["model"] = model
	}
	return nil
}

func (a *OpenAIAgent) AddTool(tool core.Tool) {
	a.tools = append(a.tools, tool)
}

func (a *OpenAIAgent) ProcessMessage(ctx context.Context, msg core.Message) ([]core.Message, error) {
	a.logger.Debug("Processing message", zap.String("content", msg.Content))

	// Add the incoming message to history
	a.history = append(a.history, openai.UserMessage(msg.Content))

	// Convert tools to OpenAI format
	toolParams := make([]openai.ChatCompletionToolParam, 0)
	for _, tool := range a.tools {
		schema := tool.JSONSchema()
		schemaJSON, err := json.Marshal(schema)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal tool schema: %w", err)
		}

		var params shared.FunctionParameters
		if err := json.Unmarshal(schemaJSON, &params); err != nil {
			return nil, fmt.Errorf("failed to unmarshal schema to function parameters: %w", err)
		}

		toolParams = append(toolParams, openai.ChatCompletionToolParam{
			Type: openai.F(openai.ChatCompletionToolTypeFunction),
			Function: openai.F(openai.FunctionDefinitionParam{
				Name:        openai.String(tool.Name()),
				Description: openai.String(tool.Description()),
				Parameters:  openai.F(params),
			}),
		})
	}

	// Get model from config
	model := a.config["model"].(string)

	// Create chat completion request
	params := openai.ChatCompletionNewParams{
		Messages: openai.F(a.history),
		Model:    openai.F(model),
	}

	// Add tools if available
	if len(toolParams) > 0 {
		params.Tools = openai.F(toolParams)
	}

	// Stream the response
	stream := a.client.Chat.Completions.NewStreaming(ctx, params)
	acc := openai.ChatCompletionAccumulator{}

	var toolResults []string
	for stream.Next() {
		chunk := stream.Current()
		acc.AddChunk(chunk)

		// Handle tool calls as they come in
		if tool, ok := acc.JustFinishedToolCall(); ok {
			a.logger.Debug("Tool call received",
				zap.String("tool", tool.Name),
				zap.String("args", tool.Arguments))

			// Find and execute the tool
			for _, t := range a.tools {
				if t.Name() == tool.Name {
					var args map[string]interface{}
					if err := json.Unmarshal([]byte(tool.Arguments), &args); err != nil {
						return nil, fmt.Errorf("failed to unmarshal tool arguments: %w", err)
					}

					result, err := t.Execute(ctx, args)
					if err != nil {
						return nil, fmt.Errorf("failed to execute tool: %w", err)
					}

					resultStr := fmt.Sprintf("%v", result)
					toolResults = append(toolResults, resultStr)
					a.logger.Debug("Tool executed",
						zap.String("tool", tool.Name),
						zap.String("result", resultStr))
				}
			}
		}

		// Handle content as it comes in
		if content, ok := acc.JustFinishedContent(); ok {
			a.logger.Debug("Content received", zap.String("content", content))
		}
	}

	if err := stream.Err(); err != nil {
		return nil, fmt.Errorf("stream error: %w", err)
	}

	// Create response message
	response := core.Message{
		Role:    core.RoleAssistant,
		Content: acc.Choices[0].Message.Content,
	}

	a.logger.Info("Message processed",
		zap.String("response", response.Content),
		zap.Strings("tool_results", toolResults))

	return []core.Message{response}, nil
}
