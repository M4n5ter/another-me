package worker

import (
	"context"
	"fmt"

	"github.com/m4n5ter/another-me/internal/task_based_core/communication"
	"github.com/m4n5ter/another-me/internal/task_based_core/state"
	"github.com/m4n5ter/another-me/pkg/llminterface"
	. "github.com/m4n5ter/another-me/pkg/option"
	"github.com/m4n5ter/another-me/pkg/reactagent"
	"github.com/m4n5ter/another-me/pkg/toolcore"
)

type TemporaryWorker struct {
	*BaseWorker
	react reactagent.ReAct
}

func NewTemporaryWorker(id string, stateManager state.StateManagerInterface, eventBus *communication.MessageBus, registry *communication.ComponentRegistry, systemPrompt string, capabilities []string, chatAdapter llminterface.ChatAdapter, evaluator Option[llminterface.ChatAdapter]) (*TemporaryWorker, error) {
	builder := reactagent.NewToolCallingAgentBuilder().
		WithLLMAdapter(chatAdapter).
		WithToolRegistry(toolcore.NewRegistry()).
		WithSystemPrompt(systemPrompt)

	if evaluator.IsSome() {
		builder.WithTaskEvaluator(evaluator.Unwrap())
	}

	react, err := builder.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build react agent: %w", err)
	}

	return &TemporaryWorker{
		BaseWorker: NewBaseWorker(id, "temporary", capabilities, stateManager, eventBus, registry),
		react:      react,
	}, nil
}

func (w *TemporaryWorker) ExecuteTask(ctx context.Context, taskID string, taskData map[string]any) error {
	task, err := w.stateManager.GetTask(taskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	chunks, err := w.react.Run(ctx, taskID, task.Description)
	if err != nil {
		return fmt.Errorf("failed to execute task: %w", err)
	}

	for chunk := range chunks {
		switch chunk.Type {
		case reactagent.AgentChunkTypeError:
			w.logger.Error("temporary worker", "chunk", chunk.Error)
		case reactagent.AgentChunkTypeFinish, reactagent.AgentChunkTypeMaxIter:
			w.logger.Info("temporary worker", "chunk", chunk.FinalResponse)
		}
	}
	return nil
}
