package core

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/m4n5ter/another-me/internal/core/types"
	. "github.com/m4n5ter/another-me/pkg/option"
)

// SmartMainLoop 智能主循环实现
type SmartMainLoop struct {
	// 核心组件
	mindscapeService MindscapeService
	decisionMaker    DecisionMaker
	agentDispatcher  AgentDispatcher
	wakeupListener   WakeupListener

	// 智能编排组件
	taskOrchestrator   SmartTaskOrchestrator
	continuousDecision ContinuousDecisionEngine
	feedbackAnalyzer   FeedbackAnalyzer

	// 状态管理
	systemState      types.SystemState
	isRunning        bool
	isWaitMode       bool
	executionHistory []types.ExecutionResult

	// 当前执行状态
	currentExecutionPlan  Option[types.ExecutionPlan]
	currentExecutionState Option[types.ExecutionState]

	// 配置
	config MainLoopConfig
	logger *slog.Logger

	// 并发控制
	mu     sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// 通道
	userInputChan   chan UserInputEvent
	wakeupEventChan chan types.WakeupEvent
	stopChan        chan struct{}
}

// MainLoopConfig 主循环配置
type MainLoopConfig struct {
	// 循环间隔
	MainLoopInterval time.Duration `json:"main_loop_interval"` // 主循环检查间隔
	WaitModeInterval time.Duration `json:"wait_mode_interval"` // 等待模式检查间隔

	// 历史记录
	MaxExecutionHistory int `json:"max_execution_history"` // 最大执行历史记录数

	// 健康检查
	HealthCheckInterval time.Duration `json:"health_check_interval"` // 健康检查间隔

	// 错误处理
	MaxRetryAttempts int           `json:"max_retry_attempts"` // 最大重试次数
	RetryBackoffBase time.Duration `json:"retry_backoff_base"` // 重试退避基础时间

	// 系统配置
	EnableAutoRecover bool `json:"enable_auto_recover"` // 是否启用自动恢复
	EnableMetrics     bool `json:"enable_metrics"`      // 是否启用指标收集

	// 用户交互
	UserInputTimeout time.Duration `json:"user_input_timeout"` // 用户输入超时时间

	// 并发控制
	MaxConcurrentTasks   int           `json:"max_concurrent_tasks"`   // 最大并发任务数
	TaskExecutionTimeout time.Duration `json:"task_execution_timeout"` // 单个任务执行超时时间
}

// UserInputEvent 用户输入事件
type UserInputEvent struct {
	Input     string         `json:"input"`
	UserID    string         `json:"user_id"`
	Context   map[string]any `json:"context"`
	Timestamp time.Time      `json:"timestamp"`
}

// DefaultMainLoopConfig 返回默认主循环配置
func DefaultMainLoopConfig() MainLoopConfig {
	return MainLoopConfig{
		MainLoopInterval:     5 * time.Second,
		WaitModeInterval:     30 * time.Second,
		MaxExecutionHistory:  100,
		HealthCheckInterval:  1 * time.Minute,
		MaxRetryAttempts:     3,
		RetryBackoffBase:     1 * time.Second,
		EnableAutoRecover:    true,
		EnableMetrics:        true,
		UserInputTimeout:     30 * time.Second,
		MaxConcurrentTasks:   10,
		TaskExecutionTimeout: 30 * time.Second,
	}
}

// NewSmartMainLoop 创建新的智能主循环
func NewSmartMainLoop(
	mindscapeService MindscapeService,
	decisionMaker DecisionMaker,
	agentDispatcher AgentDispatcher,
	wakeupListener WakeupListener,
	taskOrchestrator SmartTaskOrchestrator,
	continuousDecision ContinuousDecisionEngine,
	feedbackAnalyzer FeedbackAnalyzer,
	config MainLoopConfig,
	logger *slog.Logger,
) *SmartMainLoop {
	if logger == nil {
		logger = slog.Default().WithGroup("main_loop")
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &SmartMainLoop{
		mindscapeService:   mindscapeService,
		decisionMaker:      decisionMaker,
		agentDispatcher:    agentDispatcher,
		wakeupListener:     wakeupListener,
		taskOrchestrator:   taskOrchestrator,
		continuousDecision: continuousDecision,
		feedbackAnalyzer:   feedbackAnalyzer,
		systemState: types.SystemState{
			IsActive:            false,
			IsWaitingMode:       false,
			CurrentTask:         None[types.Task](),
			ActiveMonitoringIDs: []string{},
			LastActivity:        time.Now(),
			StartTime:           time.Now(),
			ExecutionHistory:    []types.ExecutionResult{},
			ErrorCount:          0,
			Metadata:            map[string]any{},
		},
		isRunning:             false,
		isWaitMode:            false,
		executionHistory:      make([]types.ExecutionResult, 0, config.MaxExecutionHistory),
		currentExecutionPlan:  None[types.ExecutionPlan](),
		currentExecutionState: None[types.ExecutionState](),
		config:                config,
		logger:                logger,
		ctx:                   ctx,
		cancel:                cancel,
		userInputChan:         make(chan UserInputEvent, 10),
		wakeupEventChan:       make(chan types.WakeupEvent, 10),
		stopChan:              make(chan struct{}),
	}
}

var _ MainLoop = (*SmartMainLoop)(nil)

// Start 启动主循环
func (ml *SmartMainLoop) Start(ctx context.Context) error {
	ml.mu.Lock()
	defer ml.mu.Unlock()

	if ml.isRunning {
		return fmt.Errorf("主循环已经在运行")
	}

	ml.logger.Info("开始启动主循环")

	// 初始化系统组件
	if err := ml.initializeSystem(ctx); err != nil {
		return fmt.Errorf("系统初始化失败: %w", err)
	}

	// 设置唤醒事件监听器
	if err := ml.setupWakeupListener(); err != nil {
		return fmt.Errorf("设置唤醒监听器失败: %w", err)
	}

	// 启动各种协程
	ml.isRunning = true
	ml.systemState.IsActive = true
	ml.systemState.StartTime = time.Now()
	ml.systemState.LastActivity = time.Now()

	// 启动主循环协程
	ml.wg.Add(1)
	go ml.mainLoop()

	// 启动健康检查协程
	if ml.config.EnableMetrics {
		ml.wg.Add(1)
		go ml.healthCheckLoop()
	}

	// 启动唤醒监听器
	if ml.wakeupListener != nil {
		if err := ml.wakeupListener.Start(ctx); err != nil {
			ml.logger.Warn("启动唤醒监听器失败", "error", err)
		}
	}

	ml.logger.Info("主循环启动成功")
	return nil
}

// Stop 停止主循环
func (ml *SmartMainLoop) Stop(ctx context.Context) error {
	ml.mu.Lock()
	defer ml.mu.Unlock()

	if !ml.isRunning {
		return fmt.Errorf("主循环未在运行")
	}

	ml.logger.Info("开始停止主循环")

	// 发送停止信号
	close(ml.stopChan)
	ml.cancel()

	// 停止唤醒监听器
	if ml.wakeupListener != nil {
		if err := ml.wakeupListener.Stop(ctx); err != nil {
			ml.logger.Warn("停止唤醒监听器失败", "error", err)
		}
	}

	// 等待所有协程完成
	done := make(chan struct{})
	go func() {
		ml.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		ml.logger.Info("主循环停止成功")
	case <-ctx.Done():
		ml.logger.Warn("主循环停止超时")
		return ctx.Err()
	}

	ml.isRunning = false
	ml.systemState.IsActive = false
	return nil
}

// IsRunning 检查主循环是否正在运行
func (ml *SmartMainLoop) IsRunning() bool {
	ml.mu.RLock()
	defer ml.mu.RUnlock()
	return ml.isRunning
}

// GetSystemState 获取当前系统状态
func (ml *SmartMainLoop) GetSystemState() types.SystemState {
	ml.mu.RLock()
	defer ml.mu.RUnlock()
	return ml.systemState
}

// OnWakeupEvent 处理唤醒事件
func (ml *SmartMainLoop) OnWakeupEvent(wakeupEvent types.WakeupEvent) error {
	ml.logger.Info("收到唤醒事件",
		"task_id", wakeupEvent.MonitoringTaskID,
		"reason", wakeupEvent.Reason)

	select {
	case ml.wakeupEventChan <- wakeupEvent:
		return nil
	case <-time.After(5 * time.Second):
		return fmt.Errorf("唤醒事件处理超时")
	}
}

// EnterWaitMode 进入等待模式
func (ml *SmartMainLoop) EnterWaitMode(ctx context.Context, monitoringTasks []types.MonitoringTask) error {
	ml.mu.Lock()
	defer ml.mu.Unlock()

	ml.logger.Info("进入等待模式", "monitoring_tasks_count", len(monitoringTasks))

	// 委托监控任务给Mindscape
	taskIDs := make([]string, 0, len(monitoringTasks))
	for _, task := range monitoringTasks {
		taskID, err := ml.mindscapeService.DelegateMonitoringTask(ctx, task)
		if err != nil {
			ml.logger.Error("委托监控任务失败", "error", err, "task", task.Description)
			continue
		}
		taskIDs = append(taskIDs, taskID)
		ml.logger.Debug("监控任务委托成功", "task_id", taskID)
	}

	ml.isWaitMode = true
	ml.systemState.IsWaitingMode = true
	ml.systemState.ActiveMonitoringIDs = taskIDs
	ml.systemState.LastActivity = time.Now()
	ml.systemState.Metadata["monitoring_tasks_count"] = len(monitoringTasks)

	return nil
}

// ExitWaitMode 退出等待模式
func (ml *SmartMainLoop) ExitWaitMode(ctx context.Context) error {
	ml.mu.Lock()
	defer ml.mu.Unlock()

	ml.logger.Info("退出等待模式")

	ml.isWaitMode = false
	ml.systemState.IsWaitingMode = false
	ml.systemState.ActiveMonitoringIDs = []string{}
	ml.systemState.LastActivity = time.Now()
	delete(ml.systemState.Metadata, "monitoring_tasks_count")

	return nil
}

// GetExecutionHistory 获取执行历史
func (ml *SmartMainLoop) GetExecutionHistory(limit int) []types.ExecutionResult {
	ml.mu.RLock()
	defer ml.mu.RUnlock()

	if limit <= 0 || limit > len(ml.executionHistory) {
		limit = len(ml.executionHistory)
	}

	// 返回最近的执行记录
	start := len(ml.executionHistory) - limit
	result := make([]types.ExecutionResult, limit)
	copy(result, ml.executionHistory[start:])

	return result
}

// ProcessUserInput 处理用户输入（公共方法）
func (ml *SmartMainLoop) ProcessUserInput(input, userID string, context map[string]any) error {
	event := UserInputEvent{
		Input:     input,
		UserID:    userID,
		Context:   context,
		Timestamp: time.Now(),
	}

	select {
	case ml.userInputChan <- event:
		return nil
	case <-time.After(ml.config.UserInputTimeout):
		return fmt.Errorf("用户输入处理超时")
	}
}

// 私有方法

// initializeSystem 初始化系统组件
func (ml *SmartMainLoop) initializeSystem(ctx context.Context) error {
	ml.logger.Info("开始初始化系统组件")

	// 检查Mindscape连接
	if !ml.mindscapeService.IsHealthy(ctx) {
		return fmt.Errorf("Mindscape服务不健康")
	}

	// 检查Agent调度器
	availableAgents, err := ml.agentDispatcher.GetAvailableAgents(ctx)
	if err != nil {
		return fmt.Errorf("获取可用Agent失败: %w", err)
	}

	if len(availableAgents) == 0 {
		ml.logger.Warn("没有可用的Agent，系统将以有限功能运行")
	} else {
		ml.logger.Info("系统初始化完成", "available_agents", len(availableAgents))
	}

	ml.systemState.Metadata["available_agents"] = len(availableAgents)
	ml.systemState.Metadata["mindscape_healthy"] = true

	return nil
}

// setupWakeupListener 设置唤醒监听器
func (ml *SmartMainLoop) setupWakeupListener() error {
	if ml.wakeupListener == nil {
		ml.logger.Warn("未提供唤醒监听器，将无法接收监控触发事件")
		return nil
	}

	// 设置处理器
	ml.wakeupListener.SetHandler(ml.OnWakeupEvent)

	ml.logger.Info("唤醒监听器设置完成",
		"listen_address", ml.wakeupListener.GetListenAddress())

	return nil
}

// mainLoop 主循环
func (ml *SmartMainLoop) mainLoop() {
	defer ml.wg.Done()

	ml.logger.Info("主循环开始运行")

	ticker := time.NewTicker(ml.config.MainLoopInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ml.stopChan:
			ml.logger.Info("收到停止信号，主循环退出")
			return

		case <-ml.ctx.Done():
			ml.logger.Info("上下文取消，主循环退出")
			return

		case userInput := <-ml.userInputChan:
			ml.handleUserInput(userInput)

		case wakeupEvent := <-ml.wakeupEventChan:
			ml.handleWakeupEvent(wakeupEvent)

		case <-ticker.C:
			ml.performRoutineCheck()
		}
	}
}

// handleUserInput 处理用户输入
func (ml *SmartMainLoop) handleUserInput(userInput UserInputEvent) {
	ml.logger.Info("处理用户输入", "user_id", userInput.UserID, "input", userInput.Input)

	ctx, cancel := context.WithTimeout(ml.ctx, ml.config.UserInputTimeout)
	defer cancel()

	// 退出等待模式（如果在等待模式）
	if ml.isWaitMode {
		ml.ExitWaitMode(ctx)
	}

	// 使用智能编排系统处理用户输入
	ml.executeSmartWorkflow(ctx, userInput, None[types.WakeupEvent]())
}

// executeSmartWorkflow 执行智能工作流
func (ml *SmartMainLoop) executeSmartWorkflow(ctx context.Context, userInput UserInputEvent, wakeupEvent Option[types.WakeupEvent]) {
	// 构建初始决策上下文
	decisionCtx := types.DecisionContext{
		WakeupEvent:       wakeupEvent,
		RetrievedMemories: []types.MemoryItem{}, // 将由DecisionMaker填充
		SystemState: map[string]any{
			"user_input": userInput.Input,
			"user_id":    userInput.UserID,
			"is_active":  ml.systemState.IsActive,
			"is_waiting": ml.systemState.IsWaitingMode,
		},
		Timestamp: userInput.Timestamp,
	}

	// 添加用户提供的上下文
	for k, v := range userInput.Context {
		decisionCtx.SystemState[k] = v
	}

	// 使用标准的MakeDecision方法进行初始决策
	initialDecision, err := ml.decisionMaker.MakeDecision(ctx, decisionCtx)
	if err != nil {
		ml.logger.Error("初始决策分析失败", "error", err)
		return
	}

	// 如果没有任务需要执行，直接处理监控或等待模式
	if !initialDecision.ShouldExecuteTask || initialDecision.Task.IsNone() {
		ml.handleNonExecutionDecision(ctx, initialDecision)
		return
	}

	// 将单个任务转换为任务列表
	tasks := []types.Task{initialDecision.Task.Unwrap()}

	// 开始智能任务编排和持续执行循环
	err = ml.startContinuousExecution(ctx, tasks, userInput)
	if err != nil {
		ml.logger.Error("持续执行失败", "error", err)
	}
}

// startContinuousExecution 开始持续执行循环
func (ml *SmartMainLoop) startContinuousExecution(ctx context.Context, initialTasks []types.Task, userInput UserInputEvent) error {
	iterationCount := 0
	maxIterations := 10 // 防止无限循环

	currentTasks := initialTasks

	for iterationCount < maxIterations {
		iterationCount++
		ml.logger.Info("开始执行迭代",
			"iteration", iterationCount,
			"task_count", len(currentTasks))

		// 创建执行计划
		executionPlan, err := ml.createOptimalExecutionPlan(ctx, currentTasks)
		if err != nil {
			return fmt.Errorf("创建执行计划失败: %w", err)
		}

		// 保存当前执行计划
		ml.mu.Lock()
		ml.currentExecutionPlan = Some(executionPlan)
		ml.mu.Unlock()

		// 执行任务计划
		executionState, err := ml.taskOrchestrator.ExecutePlan(ctx, executionPlan)
		if err != nil {
			ml.logger.Error("执行计划失败", "error", err)
			// 即使执行失败，也要保存执行状态以供分析
		}

		// 保存执行状态
		ml.mu.Lock()
		ml.currentExecutionState = Some(executionState)
		ml.mu.Unlock()

		// 更新系统执行历史
		ml.updateSystemExecutionHistory(executionState)

		// 进行持续决策分析
		shouldContinue, nextTasks, err := ml.performContinuousDecision(ctx, executionState, userInput)
		if err != nil {
			ml.logger.Warn("持续决策分析失败", "error", err)
			break
		}

		if !shouldContinue {
			ml.logger.Info("决策引擎建议停止执行",
				"iteration", iterationCount,
				"total_completed", executionState.CompletedTaskCount)
			break
		}

		if len(nextTasks) == 0 {
			ml.logger.Info("没有更多任务需要执行", "iteration", iterationCount)
			break
		}

		currentTasks = nextTasks
		ml.logger.Info("准备执行下一轮任务",
			"iteration", iterationCount+1,
			"next_task_count", len(nextTasks))
	}

	// 清理当前执行状态
	ml.mu.Lock()
	ml.currentExecutionPlan = None[types.ExecutionPlan]()
	ml.currentExecutionState = None[types.ExecutionState]()
	ml.mu.Unlock()

	ml.logger.Info("持续执行循环完成",
		"total_iterations", iterationCount)

	return nil
}

// createOptimalExecutionPlan 创建最优执行计划
func (ml *SmartMainLoop) createOptimalExecutionPlan(ctx context.Context, tasks []types.Task) (types.ExecutionPlan, error) {
	if ml.taskOrchestrator == nil {
		// 如果没有智能编排器，创建简单的串行计划
		return ml.createSimpleExecutionPlan(tasks), nil
	}

	// 根据任务特性决定执行模式
	executionMode := ml.determineExecutionMode(tasks)

	// 使用智能编排器创建计划
	plan, err := ml.taskOrchestrator.CreateExecutionPlan(ctx, tasks, executionMode)
	if err != nil {
		return types.ExecutionPlan{}, err
	}

	// 优化执行计划
	if ml.taskOrchestrator != nil {
		optimizedPlan, err := ml.taskOrchestrator.OptimizeExecutionPlan(ctx, plan)
		if err != nil {
			ml.logger.Warn("执行计划优化失败", "error", err)
			return plan, nil
		}
		return optimizedPlan, nil
	}

	return plan, nil
}

// determineExecutionMode 确定执行模式
func (ml *SmartMainLoop) determineExecutionMode(tasks []types.Task) types.ExecutionMode {
	if len(tasks) == 1 {
		return types.ExecutionModeSerial
	}

	// 检查是否有依赖关系
	hasGUITasks := false
	hasReActTasks := false

	for _, task := range tasks {
		switch task.AgentType {
		case types.AgentTypeGUI:
			hasGUITasks = true
		case types.AgentTypeReAct:
			hasReActTasks = true
		}
	}

	// GUI任务通常需要串行执行，ReAct任务可以并行
	if hasGUITasks && hasReActTasks {
		return types.ExecutionModeMixed
	} else if hasGUITasks {
		return types.ExecutionModeSerial
	} else {
		return types.ExecutionModeParallel
	}
}

// createSimpleExecutionPlan 创建简单的执行计划
func (ml *SmartMainLoop) createSimpleExecutionPlan(tasks []types.Task) types.ExecutionPlan {
	planID := fmt.Sprintf("simple_plan_%d", time.Now().Unix())

	var steps []types.ExecutionStep
	for i, task := range tasks {
		step := types.ExecutionStep{
			ID:                fmt.Sprintf("step_%d", i),
			Mode:              types.ExecutionModeSerial,
			Tasks:             []types.Task{task},
			MaxRetries:        2,
			ContinueOnFailure: false,
		}
		steps = append(steps, step)
	}

	return types.ExecutionPlan{
		ID:        planID,
		Steps:     steps,
		CreatedAt: time.Now(),
		Context:   map[string]any{"mode": "simple"},
		ContinuationStrategy: types.ContinuationStrategy{
			MaxIterations:        3,
			IdleThreshold:        30 * time.Second,
			FeedbackAnalysisType: types.FeedbackAnalysisSimple,
		},
	}
}

// performContinuousDecision 执行持续决策
func (ml *SmartMainLoop) performContinuousDecision(ctx context.Context, executionState types.ExecutionState, userInput UserInputEvent) (bool, []types.Task, error) {
	if ml.continuousDecision == nil && ml.feedbackAnalyzer == nil {
		// 如果没有持续决策引擎，使用简单逻辑
		return ml.performSimpleContinuousDecision(executionState)
	}

	// 收集所有执行结果
	var allResults []types.ExecutionResult
	for _, stepResult := range executionState.StepResults {
		allResults = append(allResults, stepResult.TaskResults...)
	}

	// 分析Agent输出
	var outputAnalysis types.AgentOutputAnalysis
	if ml.feedbackAnalyzer != nil {
		analysis, err := ml.feedbackAnalyzer.AnalyzeExecutionResults(ctx, allResults)
		if err != nil {
			ml.logger.Warn("Agent输出分析失败", "error", err)
		} else {
			outputAnalysis = analysis
		}
	}

	// 获取系统指标
	var systemMetrics types.SystemMetrics
	if ml.taskOrchestrator != nil {
		metrics, err := ml.taskOrchestrator.GetResourceUsage(ctx)
		if err != nil {
			ml.logger.Warn("获取系统指标失败", "error", err)
		} else {
			systemMetrics = metrics
		}
	}

	// 构建持续决策上下文
	decisionContext := types.ContinuousDecisionContext{
		InitialContext: types.DecisionContext{
			SystemState: map[string]any{
				"user_input": userInput.Input,
				"user_id":    userInput.UserID,
			},
			Timestamp: time.Now(),
		},
		ExecutionState:      executionState,
		StepResults:         executionState.StepResults,
		AgentOutputAnalysis: outputAnalysis,
		SystemMetrics:       systemMetrics,
		Timestamp:           time.Now(),
	}

	// 进行持续决策
	if ml.continuousDecision != nil {
		result, err := ml.continuousDecision.MakeContinuousDecision(ctx, decisionContext)
		if err != nil {
			return false, nil, err
		}

		// 处理持续决策结果
		var nextTasks []types.Task
		if result.NextExecutionPlan.IsSome() {
			nextPlan := result.NextExecutionPlan.Unwrap()
			// 从执行计划中提取任务
			for _, step := range nextPlan.Steps {
				nextTasks = append(nextTasks, step.Tasks...)
			}
		}

		// 处理新的监控任务
		if len(result.MonitoringTasks) > 0 {
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				ml.EnterWaitMode(ctx, result.MonitoringTasks)
			}()
		}

		return result.ShouldContinue, nextTasks, nil
	}

	return false, nil, fmt.Errorf("没有可用的持续决策引擎")
}

// performSimpleContinuousDecision 执行简单的持续决策
func (ml *SmartMainLoop) performSimpleContinuousDecision(executionState types.ExecutionState) (bool, []types.Task, error) {
	// 简单逻辑：如果有失败的任务，不继续执行
	if executionState.FailedTaskCount > 0 {
		ml.logger.Info("检测到失败任务，停止持续执行",
			"failed_count", executionState.FailedTaskCount)
		return false, nil, nil
	}

	// 简单逻辑：如果所有任务都成功完成，也停止执行
	if executionState.Status == types.ExecutionStatusSuccess {
		ml.logger.Info("所有任务成功完成，停止持续执行")
		return false, nil, nil
	}

	return false, nil, nil
}

// updateSystemExecutionHistory 更新系统执行历史
func (ml *SmartMainLoop) updateSystemExecutionHistory(executionState types.ExecutionState) {
	ml.mu.Lock()
	defer ml.mu.Unlock()

	// 将执行状态中的所有结果添加到系统历史
	for _, stepResult := range executionState.StepResults {
		for _, result := range stepResult.TaskResults {
			ml.addExecutionHistory(result)

			// 添加到系统状态历史
			ml.systemState.ExecutionHistory = append(ml.systemState.ExecutionHistory, result)

			// 存储到Mindscape（异步）
			if result.TaskID != "" {
				go func(r types.ExecutionResult) {
					ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
					defer cancel()

					// 需要重构Task对象，这里创建一个最小Task
					task := types.Task{
						ID:          r.TaskID,
						Type:        "unknown",
						Description: "从执行结果推断的任务",
						CreatedAt:   r.StartTime,
					}
					ml.storeExecutionMemory(ctx, task, r)
				}(result)
			}
		}
	}

	// 保持历史记录在合理范围内
	if len(ml.systemState.ExecutionHistory) > 20 {
		ml.systemState.ExecutionHistory = ml.systemState.ExecutionHistory[len(ml.systemState.ExecutionHistory)-20:]
	}

	// 更新系统状态
	ml.systemState.LastActivity = time.Now()
	if executionState.FailedTaskCount > 0 {
		ml.systemState.ErrorCount += executionState.FailedTaskCount
	}
}

// handleNonExecutionDecision 处理非执行决策
func (ml *SmartMainLoop) handleNonExecutionDecision(ctx context.Context, decision types.DecisionResult) {
	if len(decision.MonitoringTasks) > 0 {
		// 设置监控任务
		ml.EnterWaitMode(ctx, decision.MonitoringTasks)
	} else if decision.ShouldEnterWaitMode {
		// 进入等待模式但没有具体监控任务
		ml.EnterWaitMode(ctx, []types.MonitoringTask{})
	}

	ml.logger.Info("处理非执行决策",
		"should_wait", decision.ShouldEnterWaitMode,
		"monitoring_tasks", len(decision.MonitoringTasks))
}

// handleWakeupEvent 处理唤醒事件
func (ml *SmartMainLoop) handleWakeupEvent(wakeupEvent types.WakeupEvent) {
	ml.logger.Info("处理唤醒事件", "task_id", wakeupEvent.MonitoringTaskID)

	ctx, cancel := context.WithTimeout(ml.ctx, ml.config.UserInputTimeout)
	defer cancel()

	// 退出等待模式
	if ml.isWaitMode {
		ml.ExitWaitMode(ctx)
	}

	// 创建虚拟用户输入事件用于工作流处理
	userInput := UserInputEvent{
		Input:  fmt.Sprintf("处理监控任务唤醒: %s", wakeupEvent.Reason),
		UserID: "system",
		Context: map[string]any{
			"source":             "wakeup_event",
			"monitoring_task_id": wakeupEvent.MonitoringTaskID,
			"trigger_time":       wakeupEvent.TriggerTime,
			"observed_data":      wakeupEvent.ObservedData,
		},
		Timestamp: wakeupEvent.TriggerTime,
	}

	// 使用智能编排系统处理唤醒事件
	ml.executeSmartWorkflow(ctx, userInput, Some(wakeupEvent))
}

// performRoutineCheck 执行例行检查
func (ml *SmartMainLoop) performRoutineCheck() {
	if ml.isWaitMode {
		// 在等待模式下的例行检查
		ml.logger.Debug("等待模式例行检查")

		// 检查是否需要更新监控条件
		// 这里可以添加更多的等待模式逻辑
	} else {
		// 主动模式下的例行检查
		ml.logger.Debug("主动模式例行检查")

		// 检查是否有长时间没有活动，可能需要进入等待模式
		ml.mu.RLock()
		lastActivity := ml.systemState.LastActivity
		ml.mu.RUnlock()

		if time.Since(lastActivity) > 5*time.Minute {
			ml.logger.Info("检测到长时间无活动，考虑进入等待模式")
			// 这里可以触发自动进入等待模式的逻辑
		}
	}
}

// healthCheckLoop 健康检查循环
func (ml *SmartMainLoop) healthCheckLoop() {
	defer ml.wg.Done()

	ticker := time.NewTicker(ml.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ml.stopChan:
			return
		case <-ml.ctx.Done():
			return
		case <-ticker.C:
			ml.performHealthCheck()
		}
	}
}

// performHealthCheck 执行健康检查
func (ml *SmartMainLoop) performHealthCheck() {
	ctx, cancel := context.WithTimeout(ml.ctx, 10*time.Second)
	defer cancel()

	// 检查Mindscape连接
	mindscapeHealthy := ml.mindscapeService.IsHealthy(ctx)

	// 检查Agent可用性
	availableAgents, err := ml.agentDispatcher.GetAvailableAgents(ctx)
	agentCount := 0
	if err == nil {
		agentCount = len(availableAgents)
	}

	// 更新系统状态
	ml.mu.Lock()
	ml.systemState.Metadata["mindscape_healthy"] = mindscapeHealthy
	ml.systemState.Metadata["available_agents"] = agentCount
	ml.systemState.Metadata["last_health_check"] = time.Now()
	ml.mu.Unlock()

	ml.logger.Debug("健康检查完成",
		"mindscape_healthy", mindscapeHealthy,
		"available_agents", agentCount)

	// 如果发现健康问题，可以触发恢复机制
	if !mindscapeHealthy && ml.config.EnableAutoRecover {
		ml.logger.Warn("检测到Mindscape不健康，尝试恢复")
		// 这里可以添加自动恢复逻辑
	}
}

// addExecutionHistory 添加执行历史
func (ml *SmartMainLoop) addExecutionHistory(result types.ExecutionResult) {
	ml.mu.Lock()
	defer ml.mu.Unlock()

	// 添加到历史记录
	ml.executionHistory = append(ml.executionHistory, result)

	// 保持历史记录在限制范围内
	if len(ml.executionHistory) > ml.config.MaxExecutionHistory {
		ml.executionHistory = ml.executionHistory[1:]
	}
}

// storeExecutionMemory 存储执行记忆
func (ml *SmartMainLoop) storeExecutionMemory(ctx context.Context, task types.Task, result types.ExecutionResult) {
	memoryItem := types.MemoryItem{
		ID:         fmt.Sprintf("execution_%s_%d", task.ID, time.Now().UnixNano()),
		Timestamp:  time.Now(),
		Type:       types.MemoryTypeTaskSummary,
		Content:    fmt.Sprintf("执行任务: %s, 结果: %s", task.Description, result.Status),
		Keywords:   []string{task.Type, string(task.AgentType), string(result.Status)},
		Importance: ml.calculateMemoryImportance(result),
		UserID:     ml.extractUserIDFromTask(task),
		Metadata: map[string]any{
			"task_id":    task.ID,
			"task_type":  task.Type,
			"agent_type": task.AgentType,
			"status":     result.Status,
			"duration":   result.EndTime.Sub(result.StartTime).Seconds(),
			"success":    result.Status == types.ExecutionStatusSuccess,
		},
	}

	if err := ml.mindscapeService.StoreMemory(ctx, memoryItem); err != nil {
		ml.logger.Error("存储执行记忆失败", "error", err, "task_id", task.ID)
	}
}

// calculateMemoryImportance 计算记忆重要性
func (ml *SmartMainLoop) calculateMemoryImportance(result types.ExecutionResult) float64 {
	importance := 0.5 // 基础重要性

	// 根据执行状态调整重要性
	switch result.Status {
	case types.ExecutionStatusSuccess:
		importance += 0.2
	case types.ExecutionStatusFailure:
		importance += 0.3 // 失败的记忆可能更重要，用于学习
	case types.ExecutionStatusCancelled:
		importance -= 0.1
	}

	// 确保在合理范围内
	if importance > 1.0 {
		importance = 1.0
	} else if importance < 0.1 {
		importance = 0.1
	}

	return importance
}

// extractUserIDFromTask 从任务中提取用户ID
func (ml *SmartMainLoop) extractUserIDFromTask(task types.Task) string {
	if userID, exists := task.Context["user_id"]; exists {
		if str, ok := userID.(string); ok {
			return str
		}
	}
	return "system" // 默认用户ID
}
