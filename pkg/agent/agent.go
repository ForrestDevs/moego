package agent

import (
	"context"

	"github.com/forrestdevs/moego/pkg/core"
)

// Agent is the interface that all agents must implement
type Agent interface {
	ID() string
	Configure(config map[string]interface{}) error
	AddTool(tool core.Tool)
	ProcessMessage(ctx context.Context, msg core.Message) ([]core.Message, error)
}