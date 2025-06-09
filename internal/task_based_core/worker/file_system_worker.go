package worker

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/m4n5ter/another-me/internal/task_based_core/communication"
	"github.com/m4n5ter/another-me/internal/task_based_core/state"
	"github.com/m4n5ter/another-me/pkg/i18n"
	"github.com/m4n5ter/another-me/pkg/llminterface"
	. "github.com/m4n5ter/another-me/pkg/option"
	"github.com/m4n5ter/another-me/pkg/reactagent"
	"github.com/m4n5ter/another-me/pkg/toolcore"
	"github.com/m4n5ter/another-me/pkg/tools/admintool"
)

type FileSystemOperationWorker struct {
	*BaseWorker
	react reactagent.ReAct
}

func NewFileSystemOperationWorker(
	id string,
	stateManager state.StateManagerInterface,
	eventBus *communication.MessageBus,
	registry *communication.ComponentRegistry,
	chatAdapter llminterface.ChatAdapter,
	evaluator Option[llminterface.ChatAdapter],
) (*FileSystemOperationWorker, error) {
	ctx := context.Background()
	toolRegistry := toolcore.NewRegistry()
	tools := admintool.NewAdminTools(i18n.GlobalManager)
	for _, tool := range tools {
		err := toolRegistry.Register(ctx, tool)
		if err != nil {
			slog.Error("failed to register tool", "error", err)
			return nil, fmt.Errorf("failed to register tool: %w", err)
		}
	}

	builder := reactagent.NewToolCallingAgentBuilder().
		WithLLMAdapter(chatAdapter).
		WithToolRegistry(toolRegistry).
		WithMaxIterations(25).
		WithSystemPrompt(`你是一个专业的文件系统操作助手，能够执行各种文件和系统相关的操作。
你可以使用系统管理工具、文件系统操作、系统日志查看、系统状态监控等工具来完成任务。
请根据任务要求，选择合适的工具，确保操作的安全性和准确性。`)

	if evaluator.IsSome() {
		builder.WithTaskEvaluator(evaluator.Unwrap())
	}

	react, err := builder.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to create react agent: %w", err)
	}

	capabilities := []string{
		"syslog", "sysstat",
		"proc_info", "file", "archive", "time", "search_filesystem",
	}

	base := NewBaseWorker(id, "file_system_operation", capabilities, stateManager, eventBus, registry)
	return &FileSystemOperationWorker{BaseWorker: base, react: react}, nil
}

func (w *FileSystemOperationWorker) ExecuteTask(ctx context.Context, taskID string, taskData map[string]any) error {
	params := ExecuteTaskParams{
		WorkerID:     w.id,
		WorkerType:   "FileSystemOperationWorker",
		TaskID:       taskID,
		StateManager: w.stateManager,
		EventBus:     w.eventBus,
		React:        w.react,
		Logger:       w.logger,
		RunningMsg:   "文件系统操作任务执行中",
		CompletedMsg: "file system operation task completed",
	}

	return executeTaskWithReAct(ctx, params, w)
}
