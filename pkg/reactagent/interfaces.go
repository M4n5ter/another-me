package reactagent

import (
	"context"

	"github.com/m4n5ter/another-me/pkg/llminterface"
)

// ReAct 代表一个 ReAct 智能体抽象
type ReAct interface {
	Run(ctx context.Context, userInput, conversationID string) (<-chan AgentOutputChunk, error)
	ContinueFromCheckpoint(ctx context.Context, userInput string, checkpoint *llminterface.ChatInput) (<-chan AgentOutputChunk, error)
}
