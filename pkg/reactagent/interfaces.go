package reactagent

import (
	"context"

	"github.com/m4n5ter/another-me/pkg/llminterface"
)

type ReAct interface {
	Run(ctx context.Context, userInput, conversationID string) (<-chan AgentOutputChunk, error)
	ContinueFromCheckpoint(ctx context.Context, userInput string, checkpoint *llminterface.ChatInput) (<-chan AgentOutputChunk, error)
}
