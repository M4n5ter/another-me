package worker

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/m4n5ter/another-me/pkg/i18n"
	"github.com/m4n5ter/another-me/pkg/llminterface"
	. "github.com/m4n5ter/another-me/pkg/option"
	"github.com/m4n5ter/another-me/pkg/reactagent"
	"github.com/m4n5ter/another-me/pkg/toolcore"
	"github.com/m4n5ter/another-me/pkg/tools/admintool"
)

type FileSystemWorker struct {
	*BaseWorker
	logger *slog.Logger
	react  reactagent.ReAct
}

func NewFileSystemWorker(chatAdapter llminterface.ChatAdapter, evaluator Option[llminterface.ChatAdapter]) (*FileSystemWorker, error) {
	ctx := context.Background()
	registry := toolcore.NewRegistry()
	tools := admintool.NewAdminTools(i18n.GlobalManager)
	for _, tool := range tools {
		err := registry.Register(ctx, tool)
		if err != nil {
			slog.Error("failed to register tool", "error", err)
			return nil, fmt.Errorf("failed to register tool: %w", err)
		}
	}

	builder := reactagent.NewToolCallingAgentBuilder().
		WithLLMAdapter(chatAdapter).
		WithToolRegistry(registry).
		WithMaxIterations(25)
	if evaluator.IsSome() {
		builder.WithTaskEvaluator(evaluator.Unwrap())
	}

	react, err := builder.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to create react agent: %w", err)
	}

	workerID := fmt.Sprintf("file_system_worker_%d", time.Now().UnixNano())

	return &FileSystemWorker{
		BaseWorker: NewBaseWorker(workerID, WorkerTypeFileSystem, "admin_system", "admin_filesystem", "syslog", "sysstat", "proc_info", "file", "archive", "time", "search_filesystem"),
		react:      react,
		logger:     slog.Default().WithGroup("file_system_worker").With("worker_id", workerID),
	}, nil
}

var _ Worker = (*FileSystemWorker)(nil)

func (w *FileSystemWorker) Execute(ctx context.Context, task *Task) (*TaskResult, error) {
	return nil, nil
}

func (w *FileSystemWorker) Shutdown(ctx context.Context) error {
	return nil
}
