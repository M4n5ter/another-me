package orchestrator

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/m4n5ter/another-me/internal/task_based_core/communication"
	"github.com/m4n5ter/another-me/internal/task_based_core/state"
	"github.com/m4n5ter/another-me/pkg/llminterface"
	. "github.com/m4n5ter/another-me/pkg/option"
)

// Orchestrator 主脑编排器 - 负责任务分析、规划和调度
type Orchestrator struct {
	id     string
	logger *slog.Logger

	// 核心组件引用
	stateManager *state.StateManager
	eventBus     *communication.MessageBus
	registry     *communication.ComponentRegistry
	taskDAG      *communication.TaskDAG

	// LLM适配器
	llmAdapter llminterface.ChatAdapter

	// 运行状态
	ctx    context.Context
	cancel context.CancelFunc

	// 任务管理
	currentRequest Option[string] // 当前处理的用户请求
}

// NewOrchestrator 创建新的Orchestrator
func NewOrchestrator(
	id string,
	stateManager *state.StateManager,
	eventBus *communication.MessageBus,
	registry *communication.ComponentRegistry,
	taskDAG *communication.TaskDAG,
	llmAdapter llminterface.ChatAdapter,
) *Orchestrator {
	return &Orchestrator{
		id:           id,
		logger:       slog.Default().WithGroup("orchestrator").With("id", id),
		stateManager: stateManager,
		eventBus:     eventBus,
		registry:     registry,
		taskDAG:      taskDAG,
		llmAdapter:   llmAdapter,
	}
}

var _ OrchestratorInterface = (*Orchestrator)(nil)

// OrchestratorInterface Orchestrator接口
type OrchestratorInterface interface {
	Start(ctx context.Context) error
	Stop() error
	ProcessUserRequest(request string) error
	GetStatus() OrchestratorStatus
}

// OrchestratorStatus Orchestrator状态信息
type OrchestratorStatus struct {
	ID             string            `json:"id"`
	State          state.SystemState `json:"state"`
	CurrentRequest Option[string]    `json:"current_request"`
	ActiveTasks    int               `json:"active_tasks"`
	ProcessedTasks int               `json:"processed_tasks"`
	Uptime         time.Duration     `json:"uptime"`
	LastActivity   time.Time         `json:"last_activity"`
}

// Start 启动Orchestrator
func (o *Orchestrator) Start(ctx context.Context) error {
	o.logger.Info("启动Orchestrator")

	// 创建取消上下文
	o.ctx, o.cancel = context.WithCancel(ctx)

	// 注册到组件注册表
	if err := o.registerSelf(); err != nil {
		return err
	}

	// 订阅相关事件
	o.subscribeToEvents()

	// 启动主处理循环
	go o.mainLoop()

	o.logger.Info("Orchestrator启动完成")
	return nil
}

// Stop 停止Orchestrator
func (o *Orchestrator) Stop() error {
	o.logger.Info("停止Orchestrator")

	if o.cancel != nil {
		o.cancel()
	}

	// 注销组件
	if err := o.registry.UnregisterComponent(o.id); err != nil {
		o.logger.Error("注销Orchestrator失败", "error", err)
	}

	o.logger.Info("Orchestrator已停止")
	return nil
}

// ProcessUserRequest 处理用户请求
func (o *Orchestrator) ProcessUserRequest(request string) error {
	o.logger.Info("处理用户请求", "request", request)

	// 设置当前请求
	o.currentRequest = Some(request)

	// 分析阶段
	if err := o.stateManager.SetSystemState(state.SystemStateAnalyzing, "开始分析用户请求"); err != nil {
		return fmt.Errorf("设置系统状态失败: %w", err)
	}

	// 使用LLM分析请求
	analysisResult, err := o.analyzeRequest(request)
	if err != nil {
		o.logger.Error("请求分析失败", "error", err)
		err = o.stateManager.SetSystemState(state.SystemStateError, "请求分析失败")
		if err != nil {
			return fmt.Errorf("设置系统状态失败: %w", err)
		}
		return fmt.Errorf("请求分析失败: %w", err)
	}

	// 规划阶段
	if err := o.stateManager.SetSystemState(state.SystemStatePlanning, "开始制定执行计划"); err != nil {
		return fmt.Errorf("设置系统状态失败: %w", err)
	}

	// 生成任务计划
	tasks, err := o.planTasks(analysisResult)
	if err != nil {
		o.logger.Error("任务规划失败", "error", err)
		err = o.stateManager.SetSystemState(state.SystemStateError, "任务规划失败")
		if err != nil {
			return fmt.Errorf("设置系统状态失败: %w", err)
		}
		return fmt.Errorf("任务规划失败: %w", err)
	}

	// 执行阶段
	if err := o.stateManager.SetSystemState(state.SystemStateExecuting, "开始执行任务"); err != nil {
		return fmt.Errorf("设置系统状态失败: %w", err)
	}

	// 创建并调度任务
	if err := o.scheduleTaskExecution(tasks); err != nil {
		o.logger.Error("任务调度失败", "error", err)
		err = o.stateManager.SetSystemState(state.SystemStateError, "任务调度失败")
		if err != nil {
			return fmt.Errorf("设置系统状态失败: %w", err)
		}
		return fmt.Errorf("任务调度失败: %w", err)
	}

	return nil
}

// GetStatus 获取Orchestrator状态
func (o *Orchestrator) GetStatus() OrchestratorStatus {
	systemInfo := o.stateManager.GetSystemInfo()

	return OrchestratorStatus{
		ID:             o.id,
		State:          systemInfo.State,
		CurrentRequest: o.currentRequest,
		ActiveTasks:    systemInfo.ActiveTasks,
		ProcessedTasks: systemInfo.CompletedTasks,
		LastActivity:   time.Now(),
	}
}

// 内部方法

// registerSelf 注册自己到组件注册表
func (o *Orchestrator) registerSelf() error {
	component := &communication.ComponentInfo{
		ID:      o.id,
		Type:    communication.ComponentTypeOrchestrator,
		Name:    "主脑编排器",
		Version: "1.0.0",
		Capabilities: []string{
			"task_analysis",
			"task_planning",
			"task_scheduling",
			"resource_allocation",
			"system_coordination",
		},
		Config: map[string]any{
			"max_concurrent_tasks": 10,
			"planning_timeout":     "30s",
			"llm_model":            "gemini-2.5-flash",
		},
	}

	err := o.registry.RegisterComponent(component)
	if err != nil {
		return fmt.Errorf("注册Orchestrator失败: %w", err)
	}
	return nil
}

// subscribeToEvents 订阅相关事件
func (o *Orchestrator) subscribeToEvents() {
	// 订阅任务完成事件
	o.eventBus.Subscribe(communication.EventTypeTaskCompleted, func(event communication.Event) {
		if taskEvent, ok := event.(*communication.TaskEvent); ok {
			o.handleTaskCompleted(taskEvent)
		}
	})

	// 订阅任务失败事件
	o.eventBus.Subscribe(communication.EventTypeTaskFailed, func(event communication.Event) {
		if taskEvent, ok := event.(*communication.TaskEvent); ok {
			o.handleTaskFailed(taskEvent)
		}
	})

	// 订阅Worker注册事件
	o.eventBus.Subscribe(communication.EventTypeComponentRegistered, func(event communication.Event) {
		if componentEvent, ok := event.(*communication.ComponentEvent); ok {
			o.handleWorkerRegistered(componentEvent)
		}
	})
}

// mainLoop 主处理循环
func (o *Orchestrator) mainLoop() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-o.ctx.Done():
			o.logger.Info("Orchestrator主循环退出")
			return
		case <-ticker.C:
			o.performPeriodicTasks()
		}
	}
}

// performPeriodicTasks 执行周期性任务
func (o *Orchestrator) performPeriodicTasks() {
	// 检查系统状态
	o.checkSystemHealth()

	// 检查任务执行情况
	o.monitorTaskExecution()

	// 发送心跳
	o.sendHeartbeat()
}

// analyzeRequest 分析用户请求
func (o *Orchestrator) analyzeRequest(request string) (AnalysisResult, error) {
	o.logger.Info("开始分析用户请求")

	// 构建分析提示
	prompt := o.buildAnalysisPrompt(request)

	// 调用LLM进行分析
	input := llminterface.ChatInput{
		Messages: []llminterface.InputMessage{
			{
				Role: llminterface.RoleUser,
				Content: []llminterface.ContentPart{
					{
						Type: llminterface.PartTypeText,
						Text: prompt,
					},
				},
			},
		},
	}

	responseChan, err := o.llmAdapter.Chat(o.ctx, input)
	if err != nil {
		return AnalysisResult{}, fmt.Errorf("调用LLM失败: %w", err)
	}

	// 收集所有响应块
	var responseText strings.Builder
	for chunk := range responseChan {
		if chunk.Error != nil {
			return AnalysisResult{}, chunk.Error
		}
		for _, part := range chunk.ContentParts {
			if part.Type == llminterface.PartTypeText {
				responseText.WriteString(part.Text)
			}
		}
	}

	// 解析LLM响应
	analysisResult := o.parseAnalysisResponse(responseText.String())

	o.logger.Info("请求分析完成", "result", analysisResult)
	return analysisResult, nil
}

// planTasks 规划任务
func (o *Orchestrator) planTasks(analysis AnalysisResult) ([]TaskPlan, error) {
	o.logger.Info("开始规划任务")

	// 根据分析结果生成任务计划
	var tasks []TaskPlan

	// 这里可以实现更复杂的规划逻辑
	// 暂时创建一个示例任务
	task := TaskPlan{
		ID:                   "task-" + time.Now().Format("20060102-150405"),
		Name:                 "示例任务",
		Type:                 "general",
		Priority:             state.PriorityNormal,
		Description:          analysis.Summary,
		RequiredCapabilities: []string{"basic_operation"},
		EstimatedDuration:    30 * time.Second,
	}

	tasks = append(tasks, task)

	o.logger.Info("任务规划完成", "task_count", len(tasks))
	return tasks, nil
}

// scheduleTaskExecution 调度任务执行
func (o *Orchestrator) scheduleTaskExecution(tasks []TaskPlan) error {
	o.logger.Info("开始调度任务执行", "task_count", len(tasks))

	for _, taskPlan := range tasks {
		// 创建任务信息
		taskInfo := &state.TaskInfo{
			ID:          taskPlan.ID,
			Name:        taskPlan.Name,
			Description: taskPlan.Description,
			State:       state.TaskStatePending,
			Priority:    taskPlan.Priority,
			CreatedAt:   time.Now(),
			Metadata: map[string]any{
				"type":                  taskPlan.Type,
				"required_capabilities": taskPlan.RequiredCapabilities,
				"estimated_duration":    taskPlan.EstimatedDuration,
			},
		}

		// 创建任务到状态管理器
		if err := o.stateManager.CreateTask(taskInfo); err != nil {
			o.logger.Error("创建任务失败", "task_id", taskPlan.ID, "error", err)
			continue
		}

		// 创建任务节点到DAG
		taskNode := &communication.TaskNode{
			ID:            taskPlan.ID,
			Name:          taskPlan.Name,
			Type:          taskPlan.Type,
			Priority:      int(taskPlan.Priority),
			EstimatedTime: taskPlan.EstimatedDuration,
			Status:        communication.TaskStatusPending,
			Metadata: map[string]any{
				"required_capabilities": taskPlan.RequiredCapabilities,
			},
		}

		if err := o.taskDAG.AddTask(taskNode); err != nil {
			o.logger.Error("添加任务到DAG失败", "task_id", taskPlan.ID, "error", err)
			continue
		}

		// 查找并分配合适的Worker
		if err := o.assignTaskToWorker(taskPlan); err != nil {
			o.logger.Error("分配任务失败", "task_id", taskPlan.ID, "error", err)
		}
	}

	return nil
}

// assignTaskToWorker 分配任务给Worker
func (o *Orchestrator) assignTaskToWorker(taskPlan TaskPlan) error {
	// 查找合适的Worker
	workers := o.registry.ListComponentsByType(communication.ComponentTypeWorker)

	for _, worker := range workers {
		if worker.Status == communication.ComponentStatusActive ||
			worker.Status == communication.ComponentStatusIdle {

			// 检查Worker能力是否匹配
			if o.workerCanHandleTask(worker, taskPlan) {
				// 分配任务
				if err := o.stateManager.AssignTaskToWorker(worker.ID, taskPlan.ID); err != nil {
					continue
				}

				// 标记任务开始
				if err := o.taskDAG.MarkTaskStarted(taskPlan.ID, worker.ID); err != nil {
					continue
				}

				// 发布任务开始事件
				taskEvent := communication.NewTaskEvent(
					communication.EventTypeTaskStarted,
					o.id,
					taskPlan.ID,
					taskPlan.Name,
				)
				taskEvent.WorkerID = Some(worker.ID)
				err := o.eventBus.Publish(taskEvent)
				if err != nil {
					o.logger.Error("发布任务开始事件失败", "error", err)
					return fmt.Errorf("发布任务开始事件失败: %w", err)
				}

				o.logger.Info("任务分配成功",
					"task_id", taskPlan.ID,
					"worker_id", worker.ID)
				return nil
			}
		}
	}

	return communication.ErrChannelFull // 没有合适的Worker
}

// 辅助类型和方法

// AnalysisResult 分析结果
type AnalysisResult struct {
	Summary              string            `json:"summary"`
	Intent               string            `json:"intent"`
	Entities             map[string]string `json:"entities"`
	Complexity           string            `json:"complexity"`
	RequiredCapabilities []string          `json:"required_capabilities"`
}

// TaskPlan 任务计划
type TaskPlan struct {
	ID                   string         `json:"id"`
	Name                 string         `json:"name"`
	Type                 string         `json:"type"`
	Priority             state.Priority `json:"priority"`
	Description          string         `json:"description"`
	RequiredCapabilities []string       `json:"required_capabilities"`
	EstimatedDuration    time.Duration  `json:"estimated_duration"`
	Dependencies         []string       `json:"dependencies"`
}

// buildAnalysisPrompt 构建分析提示
func (o *Orchestrator) buildAnalysisPrompt(request string) string {
	return "请分析以下用户请求，并提供详细的分析结果：\n\n" + request
}

// parseAnalysisResponse 解析分析响应
func (o *Orchestrator) parseAnalysisResponse(response string) AnalysisResult {
	// 这里应该实现更复杂的响应解析逻辑
	return AnalysisResult{
		Summary:              response,
		Intent:               "general_request",
		Entities:             make(map[string]string),
		Complexity:           "medium",
		RequiredCapabilities: []string{"basic_operation"},
	}
}

// workerCanHandleTask 检查Worker是否能处理任务
func (o *Orchestrator) workerCanHandleTask(worker *communication.ComponentInfo, task TaskPlan) bool {
	for _, required := range task.RequiredCapabilities {
		hasCapability := slices.Contains(worker.Capabilities, required)
		if !hasCapability {
			return false
		}
	}
	return true
}

// 事件处理器

// handleTaskCompleted 处理任务完成事件
func (o *Orchestrator) handleTaskCompleted(event *communication.TaskEvent) {
	o.logger.Info("处理任务完成事件", "task_id", event.TaskID)

	// 检查是否所有任务都完成了
	if o.taskDAG.IsDAGCompleted() {
		o.logger.Info("所有任务已完成，开始评估阶段")
		err := o.stateManager.SetSystemState(state.SystemStateEvaluating, "开始评估执行结果")
		if err != nil {
			o.logger.Error("设置系统状态失败", "error", err)
		}

		// 简化版本：直接回到空闲状态
		go func() {
			time.Sleep(2 * time.Second)
			err := o.stateManager.SetSystemState(state.SystemStateIdle, "评估完成，系统空闲")
			if err != nil {
				o.logger.Error("设置系统状态失败", "error", err)
			}
			o.currentRequest = None[string]()
		}()
	}
}

// handleTaskFailed 处理任务失败事件
func (o *Orchestrator) handleTaskFailed(event *communication.TaskEvent) {
	o.logger.Error("处理任务失败事件", "task_id", event.TaskID)

	// 这里可以实现重试逻辑或错误恢复
}

// handleWorkerRegistered 处理Worker注册事件
func (o *Orchestrator) handleWorkerRegistered(event *communication.ComponentEvent) {
	o.logger.Info("新Worker注册", "worker_id", event.ComponentID, "type", event.ComponentType)

	// 检查是否有等待的任务可以分配给新Worker
	readyTasks := o.taskDAG.GetReadyTasks()
	o.logger.Info("检查待分配任务", "ready_task_count", len(readyTasks))
}

// checkSystemHealth 检查系统健康状态
func (o *Orchestrator) checkSystemHealth() {
	// 检查组件状态、资源使用等
	stats := o.stateManager.GetStatistics()
	o.logger.Debug("系统健康检查", "stats", stats)
}

// monitorTaskExecution 监控任务执行
func (o *Orchestrator) monitorTaskExecution() {
	// 检查长时间运行的任务、超时任务等
	runningTasks := o.stateManager.ListTasksByState(state.TaskStateRunning)
	o.logger.Debug("任务执行监控", "running_tasks", len(runningTasks))
}

// sendHeartbeat 发送心跳
func (o *Orchestrator) sendHeartbeat() {
	if err := o.registry.Heartbeat(o.id); err != nil {
		o.logger.Error("发送心跳失败", "error", err)
	}
}
