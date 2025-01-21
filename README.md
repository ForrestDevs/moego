# MoeGo - Mixture of Experts LLM Agent Framework

MoeGo is a clean, extensible framework for building Mixture of Experts (MoE) systems with LLM-powered agents in Go. It provides a modular architecture for defining agents, tools, and communication patterns.

## Features

- Clean architecture with dependency injection
- Modular agent system with base implementations
- Flexible tool/function calling system
- Thread-safe message routing
- Configurable agents and tools
- Structured logging with zap

## Installation

```bash
go get github.com/forrestdevs/moego
```

## Quick Start

```go
package main

import (
    "github.com/forrestdevs/moego/pkg/agent"
    "github.com/forrestdevs/moego/pkg/router"
    "github.com/forrestdevs/moego/pkg/tools"
    "go.uber.org/zap"
)

func main() {
    // Initialize logger
    logger, _ := zap.NewDevelopment()
    
    // Create router
    router := router.NewSimpleRouter(logger)
    
    // Create an agent with a calculator tool
    agent := agent.NewLLMAgent("math_expert", logger)
    agent.AddTool(tools.NewCalculator())
    
    // Register the agent
    router.RegisterAgent(agent)
    
    // Your application logic here...
}
```

## Project Structure

```
moego/
├── cmd/
│   └── example/          # Example applications
├── pkg/
│   ├── agent/           # Agent implementations
│   ├── core/            # Core interfaces and types
│   ├── router/          # Message routing
│   └── tools/           # Tool implementations
└── README.md
```

## Creating Custom Agents

Implement the `core.Agent` interface or embed `agent.BaseAgent`:

```go
type MyAgent struct {
    *agent.BaseAgent
    // Custom fields
}

func NewMyAgent(name string, logger *zap.Logger) *MyAgent {
    return &MyAgent{
        BaseAgent: agent.NewBaseAgent(name, logger),
    }
}

// Implement additional methods...
```

## Creating Custom Tools

Implement the `core.Tool` interface:

```go
type MyTool struct {
    name        string
    description string
}

func (t *MyTool) Name() string { return t.name }
func (t *MyTool) Description() string { return t.description }
func (t *MyTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
    // Tool implementation
}
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the LICENSE file for details. 
