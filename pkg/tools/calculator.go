package tools

import (
	"context"
	"fmt"
	"strconv"

	"github.com/forrestdevs/moego/pkg/core"
)

// Calculator is a tool for performing basic math operations
type Calculator struct {
	core.BaseTool
}

// NewCalculator creates a new calculator tool
func NewCalculator() *Calculator {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"operation": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"add", "subtract", "multiply", "divide", "square"},
				"description": "The mathematical operation to perform",
			},
			"a": map[string]interface{}{
				"type":        "number",
				"description": "The first number",
			},
			"b": map[string]interface{}{
				"type":        "number",
				"description": "The second number (not used for square operation)",
			},
		},
		"required": []string{"operation", "a"},
	}

	return &Calculator{
		BaseTool: *core.NewBaseTool(
			"calculator",
			"A tool for performing basic math operations (add, subtract, multiply, divide, square)",
			schema,
		),
	}
}

// Execute runs the calculator with the given arguments
func (c *Calculator) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	operation, ok := args["operation"].(string)
	if !ok {
		return nil, fmt.Errorf("operation must be a string")
	}

	a, err := getNumber(args["a"])
	if err != nil {
		return nil, fmt.Errorf("invalid first number: %w", err)
	}

	switch operation {
	case "square":
		return a * a, nil
	}

	b, err := getNumber(args["b"])
	if err != nil {
		return nil, fmt.Errorf("invalid second number: %w", err)
	}

	switch operation {
	case "add":
		return a + b, nil
	case "subtract":
		return a - b, nil
	case "multiply":
		return a * b, nil
	case "divide":
		if b == 0 {
			return nil, fmt.Errorf("division by zero")
		}
		return a / b, nil
	default:
		return nil, fmt.Errorf("unknown operation: %s", operation)
	}
}

// getNumber converts an interface{} to a float64
func getNumber(v interface{}) (float64, error) {
	switch v := v.(type) {
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int:
		return float64(v), nil
	case string:
		return strconv.ParseFloat(v, 64)
	default:
		return 0, fmt.Errorf("unsupported number type: %T", v)
	}
}
