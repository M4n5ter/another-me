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
	w.logger.Info("FileSystemOperationWorker开始执行任务", "task_id", taskID)

	task, err := w.stateManager.GetTask(taskID)
	if err != nil {
		w.logger.Error("获取任务失败", "error", err)
		return fmt.Errorf("获取任务失败: %w", err)
	}

	if task.State != state.TaskStateRunning {
		w.logger.Error("任务状态不正确", "task_id", taskID, "task_name", task.Name, "state", task.State)
		return fmt.Errorf("任务状态不正确: %w", err)
	}

	// 设置当前任务
	w.currentTask = Some(taskID)

	// 更新Worker状态
	err = w.stateManager.UpdateWorkerState(w.id, state.WorkerStateRunning, "文件系统操作任务执行中")
	if err != nil {
		w.logger.Error("更新Worker状态失败", "error", err)
	}

	w.logger.Info("文件系统操作任务执行中", "task_id", taskID, "task_name", task.Name, "worker_id", w.id)
	ch := w.handleTask(ctx, task)

	// 这里应该实现具体的数据分析逻辑
	select {
	case result := <-ch:
		// 任务执行完成
		w.tasksRun++
		w.currentTask = None[string]()

		// 更新Worker状态
		err = w.stateManager.UpdateWorkerState(w.id, state.WorkerStateIdle, "数据分析任务完成")
		if err != nil {
			w.logger.Error("更新Worker状态失败", "error", err)
		}

		if result.Error != nil {
			w.logger.Error("任务执行失败", "error", result.Error)

			// 发布任务更新进度事件
			taskEvent := communication.NewTaskEvent(
				communication.EventTypeTaskProgress,
				w.id,
				taskID,
				task.Name,
			)
			taskEvent.WorkerID = Some(w.id)
			taskEvent.Progress = 100 // TODO: 进度应该定多少？
			taskEvent.ErrorMsg = Some(result.Error.Error())
			err = w.eventBus.Publish(taskEvent)
			if err != nil {
				w.logger.Error("发布任务更新进度事件失败", "error", err)
				return fmt.Errorf("发布任务更新进度事件失败: %w", err)
			}

			return fmt.Errorf("任务执行失败: %w", result.Error)
		}

		// 发布任务完成事件
		taskEvent := communication.NewTaskEvent(
			communication.EventTypeTaskCompleted,
			w.id,
			taskID,
			task.Name,
		)
		taskEvent.WorkerID = Some(w.id)
		taskEvent.Result = Some(result.Result)
		err = w.eventBus.Publish(taskEvent)
		if err != nil {
			w.logger.Error("发布任务完成事件失败", "error", err)
			return fmt.Errorf("发布任务完成事件失败: %w", err)
		}

		w.logger.Info("任务执行完成", "task_id", taskID, "task_name", task.Name, "result", result.Result)

		// 发布更新任务进度事件
		taskEvent = communication.NewTaskEvent(
			communication.EventTypeTaskProgress,
			w.id,
			taskID,
			task.Name,
		)
		taskEvent.WorkerID = Some(w.id)
		taskEvent.Progress = 100
		taskEvent.Result = Some(result.Result)
		err = w.eventBus.Publish(taskEvent)
		if err != nil {
			w.logger.Error("发布更新任务进度事件失败", "error", err)
			return fmt.Errorf("发布更新任务进度事件失败: %w", err)
		}

		return nil

	case <-ctx.Done():
		// 任务被取消
		w.currentTask = None[string]()
		w.logger.Info("任务被取消", "task_id", taskID, "task_name", task.Name)

		// 发布任务取消事件
		taskEvent := communication.NewTaskEvent(
			communication.EventTypeTaskCancelled,
			w.id,
			taskID,
			task.Name,
		)
		taskEvent.WorkerID = Some(w.id)
		err = w.eventBus.Publish(taskEvent)
		if err != nil {
			w.logger.Error("发布任务取消事件失败", "error", err)
			return fmt.Errorf("发布任务取消事件失败: %w", err)
		}

		// 更新Worker状态
		err = w.stateManager.UpdateWorkerState(w.id, state.WorkerStateIdle, "任务被取消")
		if err != nil {
			w.logger.Error("更新Worker状态失败", "error", err)
		}

		return nil
	}
}

func (w *FileSystemOperationWorker) handleTask(ctx context.Context, task *state.TaskInfo) <-chan TaskResult {
	ch := make(chan TaskResult)
	go func() {
		defer close(ch)
		chunks, err := w.react.Run(ctx, task.Description, task.ID)
		if err != nil {
			w.logger.Error("执行任务失败", "error", err)
			return
		}

		for chunk := range chunks {
			if chunk.IsLast {
				switch chunk.Type {
				case reactagent.AgentChunkTypeFinish, reactagent.AgentChunkTypeMaxIter:
					ch <- TaskResult{Result: chunk.FinalResponse, Error: nil}
				case reactagent.AgentChunkTypeError:
					ch <- TaskResult{Result: chunk.Error, Error: nil}
				default:
					ch <- TaskResult{Result: chunk.FinalResponse, Error: nil}
				}
			}
		}
	}()

	return ch
}
