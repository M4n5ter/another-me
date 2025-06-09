package system

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"google.golang.org/genai"

	"github.com/m4n5ter/another-me/internal/task_based_core/communication"
	"github.com/m4n5ter/another-me/internal/task_based_core/orchestrator"
	"github.com/m4n5ter/another-me/internal/task_based_core/state"
	"github.com/m4n5ter/another-me/internal/task_based_core/worker"
	"github.com/m4n5ter/another-me/pkg/llminterface"
	"github.com/m4n5ter/another-me/pkg/llminterface/google"
	. "github.com/m4n5ter/another-me/pkg/option"
	"github.com/m4n5ter/another-me/pkg/toolcore"
)

// MultiAgentSystem 多智能体系统集成
type MultiAgentSystem struct {
	logger *slog.Logger

	// 核心组件
	stateManager *state.StateManager
	eventBus     *communication.MessageBus
	registry     *communication.ComponentRegistry
	taskDAG      *communication.TaskDAG

	// 集成控制器
	stateController *StateController
	eventMapper     *EventMapper
	stateBridge     *StateBridge

	// 智能体组件
	orchestrator *orchestrator.Orchestrator
	workers      []worker.Worker

	// LLM适配器
	chatAdapter llminterface.ChatAdapter

	// 系统控制
	ctx    context.Context
	cancel context.CancelFunc
}

// StateController 状态控制器 - 统一管理状态和事件的双向绑定
type StateController struct {
	logger       *slog.Logger
	stateManager *state.StateManager
	eventBus     *communication.MessageBus
}

// EventMapper 事件映射器 - 负责状态和事件之间的映射转换
type EventMapper struct {
	logger *slog.Logger
}

// StateBridge 状态桥接器 - 连接状态管理器和通信组件
type StateBridge struct {
	logger       *slog.Logger
	stateManager *state.StateManager
	registry     *communication.ComponentRegistry
	taskDAG      *communication.TaskDAG
}

// NewMultiAgentSystem 创建新的多智能体系统
func NewMultiAgentSystem(ctx context.Context, logger *slog.Logger) (*MultiAgentSystem, error) {
	systemLogger := logger.WithGroup("multi_agent_system")

	// 创建LLM适配器
	chatAdapter, err := createLLMAdapter(ctx, logger)
	if err != nil {
		return nil, err
	}

	// 创建系统context
	sysCtx, cancel := context.WithCancel(ctx)

	// 创建核心组件
	stateManager := state.NewStateManager()
	eventBus := communication.NewMessageBus(1000, 4)
	registry := communication.NewComponentRegistry(eventBus)
	taskDAG := communication.NewTaskDAG(eventBus, 3)

	// 创建集成控制器
	stateController := NewStateController(stateManager, eventBus, logger)
	eventMapper := NewEventMapper(logger)
	stateBridge := NewStateBridge(stateManager, registry, taskDAG, logger)

	system := &MultiAgentSystem{
		logger:          systemLogger,
		stateManager:    stateManager,
		eventBus:        eventBus,
		registry:        registry,
		taskDAG:         taskDAG,
		stateController: stateController,
		eventMapper:     eventMapper,
		stateBridge:     stateBridge,
		chatAdapter:     chatAdapter,
		ctx:             sysCtx,
		cancel:          cancel,
	}

	return system, nil
}

// Start 启动多智能体系统
func (mas *MultiAgentSystem) Start(ctx context.Context) error {
	mas.logger.Info("start multi agent system")

	// 设置系统状态
	err := mas.stateManager.SetSystemState(state.SystemStateAnalyzing, "系统启动初始化")
	if err != nil {
		return fmt.Errorf("failed to set system state: %w", err)
	}

	// 初始化集成控制器
	err = mas.stateController.Initialize()
	if err != nil {
		return fmt.Errorf("failed to initialize state controller: %w", err)
	}

	err = mas.stateBridge.Initialize()
	if err != nil {
		return fmt.Errorf("failed to initialize state bridge: %w", err)
	}

	// 创建并启动Orchestrator
	orchestratorComp := orchestrator.NewOrchestrator(
		"main-orchestrator",
		mas.stateManager,
		mas.eventBus,
		mas.registry,
		mas.taskDAG,
		mas.chatAdapter,
		toolcore.NewRegistry(), // TODO: 需要一个工具注册表
	)
	mas.orchestrator = orchestratorComp

	err = mas.orchestrator.Start(mas.ctx)
	if err != nil {
		return fmt.Errorf("failed to start orchestrator: %w", err)
	}

	// 创建并启动Workers
	err = mas.createAndStartWorkers()
	if err != nil {
		return err
	}

	// 设置系统为执行状态
	err = mas.stateManager.SetSystemState(state.SystemStateExecuting, "系统组件启动完成")
	if err != nil {
		return fmt.Errorf("failed to set system state: %w", err)
	}

	mas.logger.Info("multi agent system started")
	return nil
}

// Stop 停止多智能体系统
func (mas *MultiAgentSystem) Stop() error {
	mas.logger.Info("stop multi agent system")

	// 设置系统状态为关闭中
	err := mas.stateManager.SetSystemState(state.SystemStateShuttingDown, "系统准备关闭")
	if err != nil {
		return fmt.Errorf("failed to set system state: %w", err)
	}

	// 取消context
	mas.cancel()

	// 停止所有workers
	for _, w := range mas.workers {
		err := w.Stop()
		if err != nil {
			mas.logger.Error("failed to stop worker", "error", err)
		}
	}

	// 停止orchestrator
	if mas.orchestrator != nil {
		err := mas.orchestrator.Stop()
		if err != nil {
			mas.logger.Error("failed to stop orchestrator", "error", err)
		}
	}

	// 关闭核心组件
	mas.registry.Close()
	mas.eventBus.Close()

	mas.logger.Info("multi agent system stopped")
	return nil
}

// ProcessUserRequest 处理用户请求
func (mas *MultiAgentSystem) ProcessUserRequest(request string) error {
	if mas.orchestrator == nil {
		return ErrOrchestratorNotReady
	}
	err := mas.orchestrator.ProcessUserRequest(request)
	if err != nil {
		return fmt.Errorf("failed to process user request: %w", err)
	}
	return nil
}

// GetSystemStatus 获取系统状态
func (mas *MultiAgentSystem) GetSystemStatus() SystemStatus {
	systemInfo := mas.stateManager.GetSystemInfo()

	workerStatuses := make([]worker.WorkerStatus, 0, len(mas.workers))
	for _, w := range mas.workers {
		workerStatuses = append(workerStatuses, w.GetStatus())
	}

	var orchestratorStatus *orchestrator.OrchestratorStatus
	if mas.orchestrator != nil {
		status := mas.orchestrator.GetStatus()
		orchestratorStatus = &status
	}

	return SystemStatus{
		SystemInfo:         systemInfo,
		OrchestratorStatus: orchestratorStatus,
		WorkerStatuses:     workerStatuses,
		EventBusStats:      mas.eventBus.GetStats(),
		RegistryStats:      mas.registry.GetRegistryStats(),
		DAGStats:           mas.taskDAG.GetDAGStats(),
	}
}

// SystemStatus 系统完整状态
type SystemStatus struct {
	SystemInfo         state.SystemInfo                 `json:"system_info"`
	OrchestratorStatus *orchestrator.OrchestratorStatus `json:"orchestrator_status"`
	WorkerStatuses     []worker.WorkerStatus            `json:"worker_statuses"`
	EventBusStats      communication.BusStats           `json:"event_bus_stats"`
	RegistryStats      map[string]any                   `json:"registry_stats"`
	DAGStats           communication.DAGStats           `json:"dag_stats"`
}

// NewStateController 创建状态控制器
func NewStateController(sm *state.StateManager, eventBus *communication.MessageBus, logger *slog.Logger) *StateController {
	return &StateController{
		logger:       logger.WithGroup("state_controller"),
		stateManager: sm,
		eventBus:     eventBus,
	}
}

// Initialize 初始化状态控制器
func (sc *StateController) Initialize() error {
	sc.logger.Info("initialize state controller")

	// 订阅任务相关事件，自动更新状态
	sc.eventBus.Subscribe(communication.EventTypeTaskStarted, func(event communication.Event) {
		if taskEvent, ok := event.(*communication.TaskEvent); ok {
			err := sc.stateManager.UpdateTaskState(taskEvent.TaskID, state.TaskStateRunning, "任务开始执行")
			if err != nil {
				sc.logger.Error("failed to update task state", "error", err)
			}
		}
	})

	sc.eventBus.Subscribe(communication.EventTypeTaskCompleted, func(event communication.Event) {
		if taskEvent, ok := event.(*communication.TaskEvent); ok {
			err := sc.stateManager.UpdateTaskState(taskEvent.TaskID, state.TaskStateCompleted, "任务执行完成")
			if err != nil {
				sc.logger.Error("failed to update task state", "error", err)
			}
		}
	})

	sc.eventBus.Subscribe(communication.EventTypeTaskFailed, func(event communication.Event) {
		if taskEvent, ok := event.(*communication.TaskEvent); ok {
			err := sc.stateManager.UpdateTaskState(taskEvent.TaskID, state.TaskStateFailed, "任务执行失败")
			if err != nil {
				sc.logger.Error("failed to update task state", "error", err)
			}
		}
	})

	sc.eventBus.Subscribe(communication.EventTypeTaskCancelled, func(event communication.Event) {
		if taskEvent, ok := event.(*communication.TaskEvent); ok {
			err := sc.stateManager.UpdateTaskState(taskEvent.TaskID, state.TaskStateCancelled, "任务被取消")
			if err != nil {
				sc.logger.Error("failed to update task state", "error", err)
			}
		}
	})

	sc.eventBus.Subscribe(communication.EventTypeTaskRetry, func(event communication.Event) {
		if taskEvent, ok := event.(*communication.TaskEvent); ok {
			err := sc.stateManager.UpdateTaskState(taskEvent.TaskID, state.TaskStateRetrying, "任务重试")
			if err != nil {
				sc.logger.Error("failed to update task state", "error", err)
			}
		}
	})

	sc.eventBus.Subscribe(communication.EventTypeTaskProgress, func(event communication.Event) {
		if taskEvent, ok := event.(*communication.TaskEvent); ok {
			err := sc.stateManager.UpdateTaskProgress(taskEvent.TaskID, taskEvent.Progress, taskEvent.Result, taskEvent.ErrorMsg)
			if err != nil {
				sc.logger.Error("failed to update task progress", "error", err)
			}
		}
	})

	// 订阅 Worker 注册事件，自动更新状态
	sc.eventBus.Subscribe(communication.EventTypeComponentRegistered, func(event communication.Event) {
		if componentEvent, ok := event.(*communication.ComponentEvent); ok {
			if componentEvent.ComponentType == communication.ComponentTypeWorker {
				workerInfo := &state.WorkerInfo{
					ID:    componentEvent.ComponentID,
					Type:  "worker", // 简化版本，后续可以考虑使用更复杂的类型
					State: state.WorkerStateIdle,
					Tools: componentEvent.Capabilities,
				}
				err := sc.stateManager.RegisterWorker(workerInfo)
				if err != nil {
					sc.logger.Error("failed to register worker", "error", err)
				}
			}
		}
	})

	// 订阅 Worker 注销事件，自动更新状态
	sc.eventBus.Subscribe(communication.EventTypeComponentUnregistered, func(event communication.Event) {
		if componentEvent, ok := event.(*communication.ComponentEvent); ok {
			if componentEvent.ComponentType == communication.ComponentTypeWorker {
				err := sc.stateManager.UnregisterWorker(componentEvent.ComponentID, "Worker 注销事件触发")
				if err != nil {
					sc.logger.Error("failed to unregister worker", "error", err)
				}
			}
		}
	})

	// TODO: 心跳事件要处理吗？

	return nil
}

// NewEventMapper 创建事件映射器
func NewEventMapper(logger *slog.Logger) *EventMapper {
	return &EventMapper{
		logger: logger.WithGroup("event_mapper"),
	}
}

// NewStateBridge 创建状态桥接器
func NewStateBridge(sm *state.StateManager, registry *communication.ComponentRegistry, taskDAG *communication.TaskDAG, logger *slog.Logger) *StateBridge {
	return &StateBridge{
		logger:       logger.WithGroup("state_bridge"),
		stateManager: sm,
		registry:     registry,
		taskDAG:      taskDAG,
	}
}

// Initialize 初始化状态桥接器
func (sb *StateBridge) Initialize() error {
	sb.logger.Info("initialize state bridge")

	// 同步任务状态
	go sb.syncTaskStates()

	return nil
}

// syncTaskStates 同步任务状态
func (sb *StateBridge) syncTaskStates() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// 从TaskDAG获取任务状态，同步到StateManager
		// 这里可以实现更复杂的同步逻辑
	}
}

// createLLMAdapter 创建LLM适配器
func createLLMAdapter(ctx context.Context, logger *slog.Logger) (llminterface.ChatAdapter, error) {
	// 创建 google genai 客户端
	apiKey := os.Getenv("GEMINI_API_KEY")
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
		HTTPOptions: genai.HTTPOptions{
			BaseURL: "https://gateway.ai.cloudflare.com/v1/ef2319bf182b2b327281a937e203cf85/another-me/google-ai-studio",
		},
	})
	if err != nil {
		logger.Error("NewClient of gemini failed", "error", err)
		return nil, fmt.Errorf("failed to create google genai client: %w", err)
	}

	// 设置 google genai 适配器
	chatAdapter, err := google.NewGeminiAdapter(ctx, client, nil, &google.GeminiAdapterConfig{
		Model:       "gemini-2.5-flash-preview-05-20",
		Temperature: Some(float32(0.1)),
		ThinkingConfig: Some(google.GeminiThinkingConfig{
			IncludeThoughts: true,
			ThinkingBudget:  Some(int32(1000)),
		}),
	})
	if err != nil {
		logger.Error("Failed to create google genai adapter", "error", err)
		return nil, fmt.Errorf("failed to create google genai adapter: %w", err)
	}

	return chatAdapter, nil
}

// createAndStartWorkers 创建并启动Workers
func (mas *MultiAgentSystem) createAndStartWorkers() error {
	mas.logger.Info("create and start workers")

	// 创建 FileSystem Worker
	fileSystemWorker, err := worker.NewFileSystemOperationWorker(
		"file-system-worker-01",
		mas.stateManager,
		mas.eventBus,
		mas.registry,
		mas.chatAdapter,
		Some(mas.chatAdapter),
	)
	if err != nil {
		return fmt.Errorf("failed to create file system worker: %w", err)
	}
	err = fileSystemWorker.Start(mas.ctx)
	if err != nil {
		return fmt.Errorf("failed to start file system worker: %w", err)
	}
	mas.workers = append(mas.workers, fileSystemWorker)

	mas.logger.Info("workers created", "count", len(mas.workers))
	return nil
}

// 错误定义
var (
	ErrOrchestratorNotReady = &SystemError{
		Type:    ErrorTypeOrchestratorNotReady,
		Message: "orchestrator is not ready",
	}
)

// SystemError 系统错误
type SystemError struct {
	Type    ErrorType `json:"type"`
	Message string    `json:"message"`
}

// Error 实现error接口
func (e *SystemError) Error() string {
	return e.Message
}

// ErrorType 错误类型
type ErrorType int

const (
	ErrorTypeOrchestratorNotReady ErrorType = iota
	ErrorTypeSystemNotStarted
	ErrorTypeInvalidRequest
)

// String 返回错误类型的字符串表示
func (e ErrorType) String() string {
	switch e {
	case ErrorTypeOrchestratorNotReady:
		return "orchestrator is not ready"
	case ErrorTypeSystemNotStarted:
		return "system is not started"
	case ErrorTypeInvalidRequest:
		return "invalid request"
	default:
		return "unknown error"
	}
}
