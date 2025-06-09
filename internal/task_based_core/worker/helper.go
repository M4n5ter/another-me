package worker

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/m4n5ter/another-me/internal/task_based_core/communication"
	"github.com/m4n5ter/another-me/internal/task_based_core/state"
	. "github.com/m4n5ter/another-me/pkg/option"
	"github.com/m4n5ter/another-me/pkg/reactagent"
)

// ExecuteTaskParams 包含执行任务的参数
type ExecuteTaskParams struct {
	WorkerID     string
	WorkerType   string
	TaskID       string
	StateManager state.StateManagerInterface
	EventBus     *communication.MessageBus
	React        reactagent.ReAct
	Logger       *slog.Logger
	RunningMsg   string // Worker运行时状态消息
	CompletedMsg string // Worker完成时状态消息
}

// WorkerTaskHandler 处理任务状态和计数的接口
type WorkerTaskHandler interface {
	SetCurrentTask(taskID Option[string])
	IncrementTasksRun()
}

// executeTaskWithReAct 执行基于ReAct的任务的通用逻辑
func executeTaskWithReAct(ctx context.Context, params ExecuteTaskParams, handler WorkerTaskHandler) error {
	params.Logger.Info("start to execute task", "worker_type", params.WorkerType, "task_id", params.TaskID)

	task, err := params.StateManager.GetTask(params.TaskID)
	if err != nil {
		params.Logger.Error("failed to get task", "error", err)
		return fmt.Errorf("failed to get task: %w", err)
	}

	if task.State != state.TaskStateRunning {
		params.Logger.Error("task state is not correct", "task_id", params.TaskID, "task_name", task.Name, "state", task.State)
		return fmt.Errorf("task state is not correct")
	}

	// 设置当前任务
	handler.SetCurrentTask(Some(params.TaskID))

	// 更新Worker状态
	err = params.StateManager.UpdateWorkerState(params.WorkerID, state.WorkerStateRunning, params.RunningMsg)
	if err != nil {
		params.Logger.Error("failed to update worker state", "error", err)
	}

	params.Logger.Info("task is executing", "worker_type", params.WorkerType, "task_id", params.TaskID, "task_name", task.Name, "worker_id", params.WorkerID)
	ch := handleTaskByReAct(ctx, params.React, task)

	// 处理任务执行结果
	select {
	case result := <-ch:
		// 任务执行完成
		handler.IncrementTasksRun()
		handler.SetCurrentTask(None[string]())

		// 更新Worker状态
		err = params.StateManager.UpdateWorkerState(params.WorkerID, state.WorkerStateIdle, params.CompletedMsg)
		if err != nil {
			params.Logger.Error("failed to update worker state", "error", err)
		}

		if result.Error != nil {
			params.Logger.Error("task execution failed", "error", result.Error)

			// 发布任务更新进度事件
			taskEvent := communication.NewTaskEvent(
				communication.EventTypeTaskProgress,
				params.WorkerID,
				params.TaskID,
				task.Name,
			)
			taskEvent.WorkerID = Some(params.WorkerID)
			taskEvent.Progress = 100 // 失败时进度设为100
			taskEvent.ErrorMsg = Some(result.Error.Error())
			err = params.EventBus.Publish(taskEvent)
			if err != nil {
				params.Logger.Error("failed to publish task progress event", "error", err)
				return fmt.Errorf("failed to publish task progress event: %w", err)
			}

			return fmt.Errorf("task execution failed: %w", result.Error)
		}

		// 发布任务完成事件
		taskEvent := communication.NewTaskEvent(
			communication.EventTypeTaskCompleted,
			params.WorkerID,
			params.TaskID,
			task.Name,
		)
		taskEvent.WorkerID = Some(params.WorkerID)
		taskEvent.Result = Some(result.Result)
		err = params.EventBus.Publish(taskEvent)
		if err != nil {
			params.Logger.Error("failed to publish task completed event", "error", err)
			return fmt.Errorf("failed to publish task completed event: %w", err)
		}

		params.Logger.Info("task execution completed", "task_id", params.TaskID, "task_name", task.Name, "result", result.Result)

		// 发布更新任务进度事件
		taskEvent = communication.NewTaskEvent(
			communication.EventTypeTaskProgress,
			params.WorkerID,
			params.TaskID,
			task.Name,
		)
		taskEvent.WorkerID = Some(params.WorkerID)
		taskEvent.Progress = 100
		taskEvent.Result = Some(result.Result)
		err = params.EventBus.Publish(taskEvent)
		if err != nil {
			params.Logger.Error("failed to publish task progress event", "error", err)
			return fmt.Errorf("failed to publish task progress event: %w", err)
		}

		return nil

	case <-ctx.Done():
		// 任务被取消
		handler.SetCurrentTask(None[string]())
		params.Logger.Info("task is cancelled", "task_id", params.TaskID, "task_name", task.Name)

		// 发布任务取消事件
		taskEvent := communication.NewTaskEvent(
			communication.EventTypeTaskCancelled,
			params.WorkerID,
			params.TaskID,
			task.Name,
		)
		taskEvent.WorkerID = Some(params.WorkerID)
		err = params.EventBus.Publish(taskEvent)
		if err != nil {
			params.Logger.Error("failed to publish task cancelled event", "error", err)
			return fmt.Errorf("failed to publish task cancelled event: %w", err)
		}

		// 更新Worker状态
		err = params.StateManager.UpdateWorkerState(params.WorkerID, state.WorkerStateIdle, "task is cancelled")
		if err != nil {
			params.Logger.Error("failed to update worker state", "error", err)
		}

		return nil
	}
}

func handleTaskByReAct(ctx context.Context, react reactagent.ReAct, task *state.TaskInfo) <-chan TaskResult {
	ch := make(chan TaskResult)
	go func() {
		defer close(ch)
		chunks, err := react.Run(ctx, task.Description, task.ID)
		if err != nil {
			ch <- TaskResult{Result: "", Error: err}
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
