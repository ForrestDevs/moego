package main

import (
	// "context"
	"log"
	"os"
	// "time"

	dotenv "github.com/joho/godotenv"

	"github.com/forrestdevs/moego/pkg/agent"
	// "github.com/forrestdevs/moego/pkg/core"
	// "github.com/forrestdevs/moego/pkg/router"
	"github.com/forrestdevs/moego/pkg/tools"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

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


	// Create router
	// r := router.NewSimpleRouter(logger)

	// Create and configure agents
	mathExpert := agent.NewOpenAIAgent("math_expert", apiKey, logger)
	mathExpert.AddTool(tools.NewCalculator())
	mathExpert.Configure(map[string]interface{}{
		"model": "gpt-4o-mini",
	})

	assistant := agent.NewOpenAIAgent("assistant", apiKey, logger)
	assistant.Configure(map[string]interface{}{
		"model": "gpt-3.5-turbo",
	})

	// Register agents
	// if err := r.RegisterAgent(mathExpert); err != nil {
	// 	logger.Fatal("Failed to register math expert", zap.Error(err))
	// }
	// if err := r.RegisterAgent(assistant); err != nil {
	// 	logger.Fatal("Failed to register assistant", zap.Error(err))
	// }

	// Create a message
	// msg := core.Message{
	// 	ID:      "msg1",
	// 	From:    assistant.ID(),
	// 	To:      mathExpert.ID(),
	// 	Content: "I need help with a calculation. What is 42 multiplied by 8?",
	// 	Metadata: map[string]interface{}{
	// 		"operation": "multiply",
	// 		"a":         42,
	// 		"b":         8,
	// 	},
	// 	Timestamp: time.Now().Unix(),
	// }

	// Route the message
	// ctx := context.Background()
	// if err := r.Route(ctx, msg); err != nil {
	// 	logger.Error("Failed to route message", zap.Error(err))
	// 	return
	// }

	// Create a follow-up message
	// followUp := core.Message{
	// 	ID:        "msg2",
	// 	From:      assistant.ID(),
	// 	To:        mathExpert.ID(),
	// 	Content:   "Now, can you divide that result by 2?",
	// 	Timestamp: time.Now().Unix(),
	// }

	// Route the follow-up message
	// if err := r.Route(ctx, followUp); err != nil {
	// 	logger.Error("Failed to route follow-up message", zap.Error(err))
	// }
}
