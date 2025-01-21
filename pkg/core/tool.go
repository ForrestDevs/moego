package core

import (
	"context"
	"fmt"
	"reflect"
)

// Tool represents a function that can be called by an agent
type Tool interface {
	// Name returns the name of the tool
	Name() string

	// Description returns a description of what the tool does
	Description() string

	// JSONSchema returns the JSON schema for the tool's parameters
	JSONSchema() map[string]interface{}

	// Execute runs the tool with the given arguments
	Execute(ctx context.Context, args map[string]interface{}) (interface{}, error)

	// Validate validates the tool's arguments against its schema
	Validate(args map[string]interface{}) error
}

// BaseTool provides common functionality for tools
type BaseTool struct {
	name        string
	description string
	schema      map[string]interface{}
}

// NewBaseTool creates a new base tool
func NewBaseTool(name, description string, schema map[string]interface{}) *BaseTool {
	return &BaseTool{
		name:        name,
		description: description,
		schema:      schema,
	}
}

func (t *BaseTool) Name() string {
	return t.name
}

func (t *BaseTool) Description() string {
	return t.description
}

func (t *BaseTool) JSONSchema() map[string]interface{} {
	return t.schema
}

// validateType checks if the value matches the expected type
func validateType(value interface{}, expectedType string) bool {
	switch expectedType {
	case "string":
		_, ok := value.(string)
		return ok
	case "number":
		_, ok := value.(float64)
		return ok
	case "integer":
		_, ok := value.(int)
		return ok
	case "boolean":
		_, ok := value.(bool)
		return ok
	case "object":
		_, ok := value.(map[string]interface{})
		return ok
	case "array":
		return reflect.TypeOf(value).Kind() == reflect.Slice
	default:
		return true
	}
}

// Validate checks if the arguments match the tool's schema
func (t *BaseTool) Validate(args map[string]interface{}) error {
	properties, ok := t.schema["properties"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid schema: missing or invalid properties")
	}

	required := make(map[string]bool)
	if req, ok := t.schema["required"].([]string); ok {
		for _, field := range req {
			required[field] = true
		}
	}

	// Check required fields
	for field := range required {
		if _, exists := args[field]; !exists {
			return fmt.Errorf("missing required field: %s", field)
		}
	}

	// Validate each field
	for name, value := range args {
		prop, ok := properties[name].(map[string]interface{})
		if !ok {
			return fmt.Errorf("unknown field: %s", name)
		}

		expectedType, ok := prop["type"].(string)
		if !ok {
			continue // Skip type validation if type is not specified
		}

		if !validateType(value, expectedType) {
			return fmt.Errorf("invalid type for field %s: expected %s", name, expectedType)
		}

		// Validate enum if present
		if enum, ok := prop["enum"].([]interface{}); ok {
			valid := false
			for _, enumValue := range enum {
				if reflect.DeepEqual(value, enumValue) {
					valid = true
					break
				}
			}
			if !valid {
				return fmt.Errorf("invalid value for field %s: must be one of %v", name, enum)
			}
		}
	}

	return nil
}
