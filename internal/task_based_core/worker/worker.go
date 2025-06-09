package worker

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/m4n5ter/another-me/internal/task_based_core/communication"
	"github.com/m4n5ter/another-me/internal/task_based_core/state"
	. "github.com/m4n5ter/another-me/pkg/option"
)

// Worker 工作器接口 - 执行具体任务的智能体
type Worker interface {
	// Start 启动Worker
	Start(ctx context.Context) error

	// Stop 停止Worker
	Stop() error

	// GetID 获取Worker ID
	GetID() string

	// GetType 获取Worker类型
	GetType() string

	// GetCapabilities 获取Worker能力列表
	GetCapabilities() []string

	// ExecuteTask 执行任务
	ExecuteTask(ctx context.Context, taskID string, taskData map[string]any) error

	// GetStatus 获取Worker状态
	GetStatus() WorkerStatus
}

// WorkerStatus Worker状态信息
type WorkerStatus struct {
	ID          string            `json:"id"`
	Type        string            `json:"type"`
	State       state.WorkerState `json:"state"`
	CurrentTask Option[string]    `json:"current_task"`
	Uptime      time.Duration     `json:"uptime"`
	TasksRun    int               `json:"tasks_run"`
	LastTask    Option[time.Time] `json:"last_task"`
}

type TaskResult struct {
	Result any
	Error  error
}

// BaseWorker Worker基础实现
type BaseWorker struct {
	id           string
	workerType   string
	capabilities []string
	logger       *slog.Logger

	// 核心组件引用
	stateManager state.StateManagerInterface
	eventBus     *communication.MessageBus
	registry     *communication.ComponentRegistry

	// 运行状态
	ctx         context.Context
	cancel      context.CancelFunc
	startTime   time.Time
	tasksRun    int
	currentTask Option[string]
}

// NewBaseWorker 创建基础Worker
func NewBaseWorker(
	id string,
	workerType string,
	capabilities []string,
	stateManager state.StateManagerInterface,
	eventBus *communication.MessageBus,
	registry *communication.ComponentRegistry,
) *BaseWorker {
	return &BaseWorker{
		id:           id,
		workerType:   workerType,
		capabilities: capabilities,
		logger:       slog.Default().WithGroup("worker").With("id", id, "type", workerType),
		stateManager: stateManager,
		eventBus:     eventBus,
		registry:     registry,
		currentTask:  None[string](),
	}
}

var _ Worker = (*BaseWorker)(nil)

// Start 启动Worker
func (w *BaseWorker) Start(ctx context.Context) error {
	w.logger.Info("启动Worker")

	// 创建取消上下文
	w.ctx, w.cancel = context.WithCancel(ctx)
	w.startTime = time.Now()

	// 注册到组件注册表
	if err := w.registerSelf(); err != nil {
		return err
	}

	// 订阅相关事件
	w.subscribeToEvents()

	// 启动主处理循环
	go w.mainLoop()

	w.logger.Info("Worker启动完成")
	return nil
}

// Stop 停止Worker
func (w *BaseWorker) Stop() error {
	w.logger.Info("停止Worker")

	if w.cancel != nil {
		w.cancel()
	}

	// 注销组件
	if err := w.registry.UnregisterComponent(w.id); err != nil {
		w.logger.Error("注销Worker失败", "error", err)
	}

	w.logger.Info("Worker已停止")
	return nil
}

// GetID 获取Worker ID
func (w *BaseWorker) GetID() string {
	return w.id
}

// GetType 获取Worker类型
func (w *BaseWorker) GetType() string {
	return w.workerType
}

// GetCapabilities 获取Worker能力列表
func (w *BaseWorker) GetCapabilities() []string {
	return w.capabilities
}

// ExecuteTask 执行任务（基础实现，子类可重写）
func (w *BaseWorker) ExecuteTask(ctx context.Context, taskID string, taskData map[string]any) error {
	w.logger.Info("开始执行任务", "task_id", taskID)

	// 设置当前任务
	w.currentTask = Some(taskID)

	// 更新Worker状态
	if err := w.stateManager.UpdateWorkerState(w.id, state.WorkerStateRunning, "开始执行任务"); err != nil {
		w.logger.Error("更新Worker状态失败", "error", err)
	}

	// 模拟任务执行时间
	select {
	case <-time.After(5 * time.Second):
		// 任务执行完成
		w.tasksRun++
		w.currentTask = None[string]()

		// 更新Worker状态
		if err := w.stateManager.UpdateWorkerState(w.id, state.WorkerStateIdle, "任务执行完成"); err != nil {
			w.logger.Error("更新Worker状态失败", "error", err)
		}

		// 发布任务完成事件
		taskEvent := communication.NewTaskEvent(
			communication.EventTypeTaskCompleted,
			w.id,
			taskID,
			"基础任务",
		)
		taskEvent.WorkerID = Some(w.id)
		err := w.eventBus.Publish(taskEvent)
		if err != nil {
			w.logger.Error("发布任务完成事件失败", "error", err)
			return fmt.Errorf("发布任务完成事件失败: %w", err)
		}

		w.logger.Info("任务执行完成", "task_id", taskID)
		return nil

	case <-ctx.Done():
		// 任务被取消
		w.currentTask = None[string]()
		w.logger.Info("任务被取消", "task_id", taskID)
		return fmt.Errorf("任务被取消: %w", ctx.Err())
	}
}

// GetStatus 获取Worker状态
func (w *BaseWorker) GetStatus() WorkerStatus {
	var lastTask Option[time.Time]
	if w.tasksRun > 0 {
		lastTask = Some(time.Now()) // 简化版本
	}

	return WorkerStatus{
		ID:          w.id,
		Type:        w.workerType,
		State:       state.WorkerStateIdle, // 简化版本，实际应从状态管理器获取
		CurrentTask: w.currentTask,
		Uptime:      time.Since(w.startTime),
		TasksRun:    w.tasksRun,
		LastTask:    lastTask,
	}
}

// 内部方法

// registerSelf 注册自己到组件注册表
func (w *BaseWorker) registerSelf() error {
	component := &communication.ComponentInfo{
		ID:           w.id,
		Type:         communication.ComponentTypeWorker,
		Name:         w.workerType + " Worker",
		Version:      "1.0.0",
		Capabilities: w.capabilities,
		Config: map[string]any{
			"worker_type": w.workerType,
			"max_tasks":   100,
		},
	}

	// 添加Worker实例到元数据中，可以方便在其他组件中获取Worker实例
	component.Metadata = map[string]any{
		"instance": w,
	}

	err := w.registry.RegisterComponent(component)
	if err != nil {
		return fmt.Errorf("注册Worker失败: %w", err)
	}
	return nil
}

// subscribeToEvents 订阅相关事件
func (w *BaseWorker) subscribeToEvents() {
	// 订阅任务开始事件
	w.eventBus.Subscribe(communication.EventTypeTaskStarted, func(event communication.Event) {
		if taskEvent, ok := event.(*communication.TaskEvent); ok {
			// 检查任务是否分配给此Worker
			if taskEvent.WorkerID.IsSome() && taskEvent.WorkerID.Unwrap() == w.id {
				// 执行任务
				go func() {
					if err := w.ExecuteTask(w.ctx, taskEvent.TaskID, make(map[string]any)); err != nil {
						w.logger.Error("任务执行失败", "task_id", taskEvent.TaskID, "error", err)

						// 发布任务失败事件
						failedEvent := communication.NewTaskEvent(
							communication.EventTypeTaskFailed,
							w.id,
							taskEvent.TaskID,
							taskEvent.TaskName,
						)
						failedEvent.WorkerID = Some(w.id)
						failedEvent.ErrorMsg = Some(err.Error())
						err := w.eventBus.Publish(failedEvent)
						if err != nil {
							w.logger.Error("发布任务失败事件失败", "error", err)
						}

						// 更新任务状态
						if err := w.stateManager.UpdateTaskState(taskEvent.TaskID, state.TaskStateFailed, "任务执行失败"); err != nil {
							w.logger.Error("更新任务状态失败", "error", err)
						}
					}
				}()
			}
		}
	})
}

// mainLoop 主处理循环
func (w *BaseWorker) mainLoop() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-w.ctx.Done():
			w.logger.Info("Worker主循环退出")
			return
		case <-ticker.C:
			w.performPeriodicTasks()
		}
	}
}

// performPeriodicTasks 执行周期性任务
func (w *BaseWorker) performPeriodicTasks() {
	// 发送心跳
	w.sendHeartbeat()

	// 检查健康状态
	w.checkHealth()
}

// sendHeartbeat 发送心跳
func (w *BaseWorker) sendHeartbeat() {
	if err := w.registry.Heartbeat(w.id); err != nil {
		w.logger.Error("发送心跳失败", "error", err)
	}
}

// checkHealth 检查健康状态
func (w *BaseWorker) checkHealth() {
	// 基础实现：检查是否正常运行
	w.logger.Debug("Worker健康检查", "uptime", time.Since(w.startTime), "tasks_run", w.tasksRun)
}
