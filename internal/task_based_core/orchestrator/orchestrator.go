package orchestrator

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	json "github.com/json-iterator/go"

	"github.com/m4n5ter/another-me/internal/task_based_core/communication"
	"github.com/m4n5ter/another-me/internal/task_based_core/state"
	"github.com/m4n5ter/another-me/internal/task_based_core/worker"
	"github.com/m4n5ter/another-me/pkg/llminterface"
	. "github.com/m4n5ter/another-me/pkg/option"
	"github.com/m4n5ter/another-me/pkg/toolcore"
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

	// 工具注册表，一般用来给临时Worker提供工具
	toolRegistry *toolcore.Registry

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
	toolRegistry *toolcore.Registry,
) *Orchestrator {
	return &Orchestrator{
		id:           id,
		logger:       slog.Default().WithGroup("orchestrator").With("id", id),
		stateManager: stateManager,
		eventBus:     eventBus,
		registry:     registry,
		taskDAG:      taskDAG,
		llmAdapter:   llmAdapter,
		toolRegistry: toolRegistry,
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

// ProcessUserRequest 处理用户请求 - 智能化四阶段流程
func (o *Orchestrator) ProcessUserRequest(request string) error {
	o.logger.Info("开始智能化处理用户请求", "request", request)

	// 设置当前请求
	o.currentRequest = Some(request)

	// ========== 阶段 1: 任务请求丰富化 ==========
	if err := o.stateManager.SetSystemState(state.SystemStateAnalyzing, "阶段1: 任务请求丰富化"); err != nil {
		return fmt.Errorf("设置系统状态失败: %w", err)
	}

	enrichmentResult, err := o.enrichTaskRequest(request)
	if err != nil {
		o.logger.Error("任务请求丰富化失败", "error", err)
		if err := o.stateManager.SetSystemState(state.SystemStateError, "任务请求丰富化失败"); err != nil {
			return fmt.Errorf("设置系统状态失败: %w", err)
		}
		return fmt.Errorf("任务请求丰富化失败: %w", err)
	}

	// ========== 阶段 2: 任务分析 ==========
	if err := o.stateManager.SetSystemState(state.SystemStateAnalyzing, "阶段2: 深度任务分析"); err != nil {
		return fmt.Errorf("设置系统状态失败: %w", err)
	}

	analysisResult, err := o.analyzeEnrichedRequest(enrichmentResult)
	if err != nil {
		o.logger.Error("任务分析失败", "error", err)
		if err := o.stateManager.SetSystemState(state.SystemStateError, "任务分析失败"); err != nil {
			return fmt.Errorf("设置系统状态失败: %w", err)
		}
		return fmt.Errorf("任务分析失败: %w", err)
	}

	// ========== 阶段 3: Worker-任务映射 ==========
	if err := o.stateManager.SetSystemState(state.SystemStatePlanning, "阶段3: Worker任务映射与资源规划"); err != nil {
		return fmt.Errorf("设置系统状态失败: %w", err)
	}

	mappingResult, err := o.mapTasksToWorkers(enrichmentResult, analysisResult)
	if err != nil {
		o.logger.Error("Worker任务映射失败", "error", err)
		if err := o.stateManager.SetSystemState(state.SystemStateError, "Worker任务映射失败"); err != nil {
			return fmt.Errorf("设置系统状态失败: %w", err)
		}
		return fmt.Errorf("worker任务映射失败: %w", err)
	}

	// ========== 阶段 4: 临时Worker创建与任务执行 ==========
	if err := o.stateManager.SetSystemState(state.SystemStateExecuting, "阶段4: 创建临时Worker并开始执行"); err != nil {
		return fmt.Errorf("设置系统状态失败: %w", err)
	}

	if err := o.executeWithMappingResult(mappingResult); err != nil {
		o.logger.Error("任务执行失败", "error", err)
		if err := o.stateManager.SetSystemState(state.SystemStateError, "任务执行失败"); err != nil {
			return fmt.Errorf("设置系统状态失败: %w", err)
		}
		return fmt.Errorf("任务执行失败: %w", err)
	}

	o.logger.Info("智能化用户请求处理完成，任务已开始执行")
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

// ========== 智能化处理流程方法 ==========

// enrichTaskRequest 阶段1: 任务请求丰富化
func (o *Orchestrator) enrichTaskRequest(userInput string) (*TaskRequestEnrichmentResponse, error) {
	o.logger.Info("开始任务请求丰富化", "user_input", userInput)

	// 准备丰富请求
	enrichmentRequest := TaskRequestEnrichmentRequest{
		UserInput: userInput,
	}

	// 构建系统提示
	systemPrompt := `你是一个智能任务请求分析专家。用户可能会提供简单或不够详细的任务描述，你的任务是：

1. **理解用户意图**：深入理解用户真正想要完成的目标
2. **识别具体目标**：将模糊的描述转化为具体可执行的目标
3. **补充细节**：基于常见场景和最佳实践，合理补充必要的细节
4. **明确范围**：清晰定义任务的边界和范围
5. **识别约束**：发现可能的限制条件和约束
6. **定义成功标准**：明确什么算作任务完成

请基于用户输入和可用资源，提供一个详细、明确、可执行的任务描述。`

	// 构建用户提示
	requestJSON, err := json.MarshalToString(enrichmentRequest)
	if err != nil {
		return nil, fmt.Errorf("序列化丰富化请求失败: %w", err)
	}

	userPrompt := fmt.Sprintf("请分析并丰富以下用户任务请求：\n%s", requestJSON)

	// 构建ChatInput
	input := llminterface.ChatInput{
		Messages: []llminterface.InputMessage{
			llminterface.SystemInputMessage(systemPrompt),
			llminterface.UserInputMessageText(userPrompt),
		},
		ConversationID: fmt.Sprintf("task_enrichment_%s", time.Now().Format("20060102-150405")),
	}

	// 使用ProduceJSON获取结构化响应
	response, err := o.llmAdapter.ProduceJSON(o.ctx, input, Some(*CreateTaskRequestEnrichmentSchema()))
	if err != nil {
		return nil, fmt.Errorf("调用LLM进行任务丰富化失败: %w", err)
	}

	// 解析响应
	var enrichmentResponse TaskRequestEnrichmentResponse
	if err := json.UnmarshalFromString(response, &enrichmentResponse); err != nil {
		return nil, fmt.Errorf("解析任务丰富化响应失败: %w", err)
	}

	o.logger.Info("任务请求丰富化完成",
		"enriched_description", enrichmentResponse.EnrichedDescription,
		"identified_goals_count", len(enrichmentResponse.IdentifiedGoals),
		"scope", enrichmentResponse.Scope,
		"success_criteria_count", len(enrichmentResponse.SuccessCriteria),
	)

	return &enrichmentResponse, nil
}

// analyzeEnrichedRequest 阶段2: 基于丰富化结果进行深度任务分析
func (o *Orchestrator) analyzeEnrichedRequest(enrichmentResult *TaskRequestEnrichmentResponse) (*TaskAnalysisResponse, error) {
	o.logger.Info("开始深度任务分析")

	// 获取可用Worker信息
	availableWorkers := o.getAvailableWorkers()

	// 准备任务分析请求 - 基于丰富化结果
	analysisRequest := TaskAnalysisRequest{
		TaskID:      "analysis-" + time.Now().Format("20060102-150405"),
		Name:        "深度任务分析",
		Description: enrichmentResult.EnrichedDescription,
		Priority:    "normal",
		Metadata: map[string]any{
			"enrichment_result": *enrichmentResult,
			"available_workers": availableWorkers,
			"identified_goals":  enrichmentResult.IdentifiedGoals,
			"success_criteria":  enrichmentResult.SuccessCriteria,
			"constraints":       enrichmentResult.Constraints,
		},
	}

	// 构建系统提示
	systemPrompt := `你是一个高级任务分析专家。基于已经丰富化的任务描述，你需要进行深度的技术分析：

**分析重点：**
1. **任务分解评估**：判断是否需要分解为子任务，以及如何分解
2. **技术复杂度**：评估任务的技术难度和复杂程度
3. **资源需求**：确定需要哪种类型的Worker和能力
4. **执行时间**：基于任务复杂度进行准确的时间估算
5. **依赖关系**：识别子任务间的依赖关系和执行顺序
6. **风险识别**：识别潜在的技术风险和挑战

请基于丰富化的任务信息，提供详细的技术分析结果。`

	// 构建用户提示
	requestJSON, err := json.MarshalToString(analysisRequest)
	if err != nil {
		return nil, fmt.Errorf("序列化任务分析请求失败: %w", err)
	}

	userPrompt := fmt.Sprintf("请对以下丰富化的任务进行深度技术分析：\n%s", requestJSON)

	// 构建ChatInput
	input := llminterface.ChatInput{
		Messages: []llminterface.InputMessage{
			llminterface.SystemInputMessage(systemPrompt),
			llminterface.UserInputMessageText(userPrompt),
		},
		ConversationID: fmt.Sprintf("task_analysis_%s", analysisRequest.TaskID),
	}

	// 使用ProduceJSON获取结构化响应
	response, err := o.llmAdapter.ProduceJSON(o.ctx, input, Some(*CreateTaskAnalysisSchema()))
	if err != nil {
		return nil, fmt.Errorf("调用LLM进行深度分析失败: %w", err)
	}

	// 解析响应
	var analysisResponse TaskAnalysisResponse
	if err := json.UnmarshalFromString(response, &analysisResponse); err != nil {
		return nil, fmt.Errorf("解析深度分析响应失败: %w", err)
	}

	o.logger.Info("深度任务分析完成",
		"requires_decomposition", analysisResponse.RequiresDecomposition,
		"complexity", analysisResponse.EstimatedComplexity,
		"sub_tasks_count", len(analysisResponse.SubTasks),
		"estimated_duration", analysisResponse.EstimatedDuration,
		"required_worker_type", analysisResponse.RequiredWorkerType,
		"risk_level", analysisResponse.RiskAssessment.Level,
	)

	return &analysisResponse, nil
}

// mapTasksToWorkers 阶段3: Worker任务映射与资源规划
func (o *Orchestrator) mapTasksToWorkers(enrichmentResult *TaskRequestEnrichmentResponse, analysisResult *TaskAnalysisResponse) (*WorkerTaskMappingResponse, error) {
	o.logger.Info("开始Worker任务映射与资源规划")

	// 获取可用Worker信息
	availableWorkers := o.getAvailableWorkers()

	// 准备映射请求
	mappingRequest := WorkerTaskMappingRequest{
		EnrichmentResult: *enrichmentResult,
		AnalysisResult:   *analysisResult,
		AvailableWorkers: availableWorkers,
	}

	// 构建系统提示
	systemPrompt := `你是一个智能资源调度专家。你需要将分析出的任务分配给合适的Worker，并为无法满足的需求创建临时Worker。

**任务分配原则：**
1. **能力匹配**：确保Worker具备完成任务所需的能力
2. **负载均衡**：考虑Worker当前的工作负载
3. **依赖关系**：正确设置任务间的依赖关系
4. **效率优化**：选择最适合和最高效的Worker

**临时Worker创建指导：**
- 当现有Worker无法满足特定需求时，设计新的临时Worker
- 为临时Worker编写精心设计的系统提示词，使其能够专业地完成特定任务
- 定义临时Worker的能力范围和生命周期
- 考虑临时Worker与现有系统的集成方式

**决策策略：**
- 优先使用现有Worker
- 对于复杂或特殊任务，创建专门的临时Worker
- 确保任务分配的合理性和可执行性

请基于任务分析结果和可用资源，制定最优的执行策略。`

	// 构建用户提示
	requestJSON, err := json.MarshalToString(mappingRequest)
	if err != nil {
		return nil, fmt.Errorf("序列化映射请求失败: %w", err)
	}

	userPrompt := fmt.Sprintf("请为以下任务制定Worker分配策略：\n%s", requestJSON)

	// 构建ChatInput
	input := llminterface.ChatInput{
		Messages: []llminterface.InputMessage{
			llminterface.SystemInputMessage(systemPrompt),
			llminterface.UserInputMessageText(userPrompt),
		},
		ConversationID: fmt.Sprintf("task_mapping_%s", time.Now().Format("20060102-150405")),
	}

	// 使用ProduceJSON获取结构化响应
	response, err := o.llmAdapter.ProduceJSON(o.ctx, input, Some(*CreateWorkerTaskMappingSchema()))
	if err != nil {
		return nil, fmt.Errorf("调用LLM进行任务映射失败: %w", err)
	}

	// 解析响应
	var mappingResponse WorkerTaskMappingResponse
	if err := json.UnmarshalFromString(response, &mappingResponse); err != nil {
		return nil, fmt.Errorf("解析任务映射响应失败: %w", err)
	}

	o.logger.Info("Worker任务映射完成",
		"task_assignments_count", len(mappingResponse.TaskAssignments),
		"new_workers_required", len(mappingResponse.RequiredNewWorkers),
		"unassigned_tasks", len(mappingResponse.UnassignedTasks),
		"estimated_completion", mappingResponse.EstimatedCompletion,
	)

	return &mappingResponse, nil
}

// executeWithMappingResult 阶段4: 基于映射结果执行任务
func (o *Orchestrator) executeWithMappingResult(mappingResult *WorkerTaskMappingResponse) error {
	o.logger.Info("开始基于映射结果执行任务")

	// 步骤4.1: 创建临时Worker
	if len(mappingResult.RequiredNewWorkers) > 0 {
		if err := o.createTemporaryWorkers(mappingResult.RequiredNewWorkers); err != nil {
			return fmt.Errorf("创建临时Worker失败: %w", err)
		}
	}

	// 步骤4.2: 执行任务分配
	if err := o.executeTaskAssignments(mappingResult.TaskAssignments); err != nil {
		return fmt.Errorf("执行任务分配失败: %w", err)
	}

	// 步骤4.3: 处理未分配的任务
	if len(mappingResult.UnassignedTasks) > 0 {
		o.handleUnassignedTasks(mappingResult.UnassignedTasks)
	}

	o.logger.Info("任务执行初始化完成")
	return nil
}

// 辅助方法

// createTemporaryWorkers 创建临时Worker
func (o *Orchestrator) createTemporaryWorkers(newWorkerRequests []NewWorkerRequest) error {
	o.logger.Info("开始创建临时Worker", "count", len(newWorkerRequests))

	for _, workerReq := range newWorkerRequests {
		o.logger.Info("创建临时Worker",
			"worker_name", workerReq.WorkerName,
			"worker_type", workerReq.WorkerType,
			"capabilities", workerReq.RequiredCapabilities,
			"tasks_to_handle", workerReq.TasksToHandle,
			"estimated_lifetime", workerReq.EstimatedLifetime,
		)

		tempWorker, err := worker.NewTemporaryWorker(
			workerReq.WorkerID,
			o.stateManager,
			o.eventBus,
			o.registry,
			workerReq.SystemPrompt,
			workerReq.RequiredCapabilities,
			o.llmAdapter,
			nil,
		)
		if err != nil {
			o.logger.Error("failed to create temporary worker", "error", err)
			return fmt.Errorf("failed to create temporary worker: %w", err)
		}

		err = tempWorker.Start(o.ctx)
		if err != nil {
			o.logger.Error("failed to start temporary worker", "error", err)
			return fmt.Errorf("failed to start temporary worker: %w", err)
		}

		o.logger.Info("临时Worker系统提示词",
			"worker_name", workerReq.WorkerName,
			"system_prompt", workerReq.SystemPrompt,
		)
	}

	return nil
}

// executeTaskAssignments 执行任务分配
func (o *Orchestrator) executeTaskAssignments(assignments []TaskAssignment) error {
	o.logger.Info("开始执行任务分配", "assignments_count", len(assignments))

	for _, assignment := range assignments {
		o.logger.Info("分配任务",
			"task_id", assignment.TaskID,
			"task_name", assignment.TaskName,
			"worker_id", assignment.AssignedWorkerID,
			"worker_type", assignment.WorkerType,
			"priority", assignment.Priority,
			"estimated_time", assignment.EstimatedTime,
			"reason", assignment.AssignmentReason,
		)

		taskInfo := state.TaskInfo{
			ID:          assignment.TaskID,
			Name:        assignment.TaskName,
			Description: o.currentRequest.TakeOr("Unknown Task"), // TODO: 可能需要改进
			State:       state.TaskStatePending,
			CreatedAt:   time.Now(),
		}

		err := o.stateManager.CreateTask(&taskInfo)
		if err != nil {
			o.logger.Error("failed to create task", "error", err)
			return fmt.Errorf("failed to create task: %w", err)
		}

		err = o.stateManager.AssignTaskToWorker(assignment.AssignedWorkerID, assignment.TaskID)
		if err != nil {
			o.logger.Error("failed to assign task to worker", "error", err)
			return fmt.Errorf("failed to assign task to worker: %w", err)
		}

		taskEvent := communication.NewTaskEvent(
			communication.EventTypeTaskStarted,
			o.id,
			assignment.TaskID,
			assignment.TaskName,
		)

		err = o.eventBus.Publish(taskEvent)
		if err != nil {
			o.logger.Error("failed to publish task event", "error", err)
			return fmt.Errorf("failed to publish task event: %w", err)
		}

		o.logger.Info("task assigned to worker", "task_id", assignment.TaskID, "worker_id", assignment.AssignedWorkerID)
	}

	return nil
}

// handleUnassignedTasks 处理未分配的任务
func (o *Orchestrator) handleUnassignedTasks(unassignedTasks []UnassignedTask) {
	o.logger.Warn("存在未分配的任务", "count", len(unassignedTasks))

	for _, task := range unassignedTasks {
		o.logger.Warn("未分配任务",
			"task_id", task.TaskID,
			"task_name", task.TaskName,
			"reason", task.Reason,
			"suggestions", task.Suggestions,
		)
	}
}

// getAvailableWorkers 获取可用Worker
func (o *Orchestrator) getAvailableWorkers() []AvailableWorker {
	workersComp := o.registry.ListComponentsByType(communication.ComponentTypeWorker)
	workers := make([]worker.Worker, 0, len(workersComp))
	for _, workerComp := range workersComp {
		worker, ok := workerComp.Metadata["instance"].(worker.Worker)
		if !ok {
			continue
		}
		workers = append(workers, worker)
	}
	availableWorkers := make([]AvailableWorker, len(workers))
	for i, worker := range workers {
		availableWorkers[i] = AvailableWorker{
			ID:           worker.GetID(),
			Type:         worker.GetType(),
			State:        worker.GetStatus().State.String(),
			Capabilities: worker.GetCapabilities(),
		}
	}
	return availableWorkers
}

// 事件处理器

// handleTaskCompleted 处理任务完成事件
func (o *Orchestrator) handleTaskCompleted(event *communication.TaskEvent) {
	o.logger.Info("处理任务完成事件", "task_id", event.TaskID)

	// 通知TaskDAG任务已完成
	if err := o.taskDAG.MarkTaskCompleted(event.TaskID, map[string]any{"result": event.Result}); err != nil {
		o.logger.Error("标记任务完成失败", "task_id", event.TaskID, "error", err)
	}

	// 检查是否有等待此任务的其他任务可以开始执行
	readyTasks := o.taskDAG.GetReadyTasks()
	for _, readyTask := range readyTasks {
		o.logger.Info("发现就绪任务", "task_id", readyTask.ID, "task_name", readyTask.Name)
		// 在新的架构中，任务执行由映射结果驱动，这里只记录
		// TODO: 可以实现更复杂的动态任务调度逻辑
	}

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

// DemoComplexTaskProcessing 演示复杂任务处理能力
func (o *Orchestrator) DemoComplexTaskProcessing(ctx context.Context) error {
	o.logger.Info("开始演示复杂任务处理")

	// 模拟一个复杂的用户请求
	complexRequest := `
	请帮我处理一个数据分析项目：
	1. 首先从多个Excel文件读取销售数据
	2. 同时从网站下载最新的产品价格信息
	3. 将销售数据和价格数据进行合并分析
	4. 生成统计图表
	5. 最后发送分析报告给管理层
	
	注意：数据合并必须等待前两步完成，图表生成需要合并分析的结果，发送报告需要等待图表完成。
	`

	// 使用新的智能化四阶段流程处理复杂请求
	err := o.ProcessUserRequest(complexRequest)
	if err != nil {
		return fmt.Errorf("处理复杂任务演示失败: %w", err)
	}

	o.logger.Info("复杂任务演示完成，任务已通过智能化流程开始执行")
	return nil
}
