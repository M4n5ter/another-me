package core

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/m4n5ter/another-me/internal/core/types"
	. "github.com/m4n5ter/another-me/pkg/option"
)

// smartTaskOrchestrator 智能任务编排器实现
type smartTaskOrchestrator struct {
	agentDispatcher    AgentDispatcher
	continuousDecision ContinuousDecisionEngine
	feedbackAnalyzer   FeedbackAnalyzer
	config             OrchestratorConfig
	logger             *slog.Logger

	// 执行状态管理
	activeExecutions map[string]*types.ExecutionState
	executionHistory []types.ExecutionState
	resourceMetrics  types.SystemMetrics

	// 并发控制
	mu                  sync.RWMutex
	semaphore           chan struct{} // 并发控制信号量
	executionResultChan chan TaskExecutionEvent
}

// OrchestratorConfig 编排器配置
type OrchestratorConfig struct {
	MaxConcurrentTasks   int           `json:"max_concurrent_tasks"`
	TaskExecutionTimeout time.Duration `json:"task_execution_timeout"`
	MaxExecutionHistory  int           `json:"max_execution_history"`
	EnableMetrics        bool          `json:"enable_metrics"`
	EnableOptimization   bool          `json:"enable_optimization"`
}

// TaskExecutionEvent 任务执行事件
type TaskExecutionEvent struct {
	PlanID    string                `json:"plan_id"`
	StepID    string                `json:"step_id"`
	TaskID    string                `json:"task_id"`
	Result    types.ExecutionResult `json:"result"`
	Timestamp time.Time             `json:"timestamp"`
}

// NewSmartTaskOrchestrator 创建新的智能任务编排器
func NewSmartTaskOrchestrator(
	agentDispatcher AgentDispatcher,
	continuousDecision ContinuousDecisionEngine,
	feedbackAnalyzer FeedbackAnalyzer,
	config OrchestratorConfig,
	logger *slog.Logger,
) SmartTaskOrchestrator {
	if logger == nil {
		logger = slog.Default().WithGroup("orchestrator")
	}

	return &smartTaskOrchestrator{
		agentDispatcher:     agentDispatcher,
		continuousDecision:  continuousDecision,
		feedbackAnalyzer:    feedbackAnalyzer,
		config:              config,
		logger:              logger,
		activeExecutions:    make(map[string]*types.ExecutionState),
		executionHistory:    make([]types.ExecutionState, 0, config.MaxExecutionHistory),
		semaphore:           make(chan struct{}, config.MaxConcurrentTasks),
		executionResultChan: make(chan TaskExecutionEvent, 100),
	}
}

var _ SmartTaskOrchestrator = (*smartTaskOrchestrator)(nil)

// ExecutePlan 执行任务计划
func (o *smartTaskOrchestrator) ExecutePlan(ctx context.Context, plan types.ExecutionPlan) (types.ExecutionState, error) {
	planCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// 设置全局超时
	if plan.GlobalTimeout.IsSome() {
		timeoutCtx, timeoutCancel := context.WithTimeout(planCtx, plan.GlobalTimeout.Unwrap())
		defer timeoutCancel()
		planCtx = timeoutCtx
	}

	// 初始化执行状态
	executionState := types.ExecutionState{
		PlanID:             plan.ID,
		CurrentStepIndex:   0,
		StepResults:        make([]types.StepResult, 0, len(plan.Steps)),
		StartTime:          time.Now(),
		Status:             types.ExecutionStatusInProgress,
		IterationCount:     0,
		TotalTaskCount:     o.countTotalTasks(plan.Steps),
		CompletedTaskCount: 0,
		FailedTaskCount:    0,
		Metadata: map[string]any{
			"plan_id":    plan.ID,
			"created_at": plan.CreatedAt,
		},
	}

	// 注册执行状态
	o.mu.Lock()
	o.activeExecutions[plan.ID] = &executionState
	o.mu.Unlock()

	defer func() {
		o.mu.Lock()
		delete(o.activeExecutions, plan.ID)
		o.addExecutionHistory(executionState)
		o.mu.Unlock()
	}()

	o.logger.Info("开始执行任务计划",
		"plan_id", plan.ID,
		"total_steps", len(plan.Steps),
		"total_tasks", executionState.TotalTaskCount)

	// 执行计划的主循环
	for {
		// 检查是否应该停止
		select {
		case <-planCtx.Done():
			executionState.Status = types.ExecutionStatusCancelled
			return executionState, planCtx.Err()
		default:
		}

		// 执行当前步骤
		if executionState.CurrentStepIndex >= len(plan.Steps) {
			// 所有步骤执行完毕
			executionState.Status = types.ExecutionStatusSuccess
			break
		}

		currentStep := plan.Steps[executionState.CurrentStepIndex]
		stepResult, err := o.executeStep(planCtx, currentStep, &executionState)

		// 更新执行状态
		o.mu.Lock()
		executionState.StepResults = append(executionState.StepResults, stepResult)
		executionState.CompletedTaskCount += len(stepResult.TaskResults)
		for _, result := range stepResult.TaskResults {
			if result.Status == types.ExecutionStatusFailure {
				executionState.FailedTaskCount++
			}
		}
		o.mu.Unlock()

		if err != nil && !currentStep.ContinueOnFailure {
			executionState.Status = types.ExecutionStatusFailure
			o.logger.Error("步骤执行失败，停止计划执行",
				"plan_id", plan.ID,
				"step_id", currentStep.ID,
				"error", err)
			break
		}

		// 持续决策：判断是否继续执行
		shouldContinue, nextPlan, err := o.evaluateContinuation(planCtx, plan, executionState)
		if err != nil {
			o.logger.Warn("持续决策失败", "error", err)
		}

		if !shouldContinue {
			o.logger.Info("决策引擎建议停止执行", "plan_id", plan.ID)
			executionState.Status = types.ExecutionStatusSuccess
			break
		}

		// 如果有新的执行计划，递归执行
		if nextPlan.IsSome() {
			newPlan := nextPlan.Unwrap()
			o.logger.Info("执行新生成的计划",
				"original_plan_id", plan.ID,
				"new_plan_id", newPlan.ID)

			_, err := o.ExecutePlan(planCtx, newPlan)
			if err != nil {
				o.logger.Warn("新计划执行失败", "error", err)
			}
		}

		executionState.CurrentStepIndex++
		executionState.IterationCount++

		// 检查是否达到最大迭代次数
		if plan.ContinuationStrategy.MaxIterations > 0 &&
			executionState.IterationCount >= plan.ContinuationStrategy.MaxIterations {
			o.logger.Info("达到最大迭代次数，停止执行",
				"plan_id", plan.ID,
				"iterations", executionState.IterationCount)
			executionState.Status = types.ExecutionStatusSuccess
			break
		}
	}

	o.logger.Info("任务计划执行完成",
		"plan_id", plan.ID,
		"status", executionState.Status,
		"completed_tasks", executionState.CompletedTaskCount,
		"failed_tasks", executionState.FailedTaskCount)

	return executionState, nil
}

// executeStep 执行单个步骤
func (o *smartTaskOrchestrator) executeStep(ctx context.Context, step types.ExecutionStep, executionState *types.ExecutionState) (types.StepResult, error) {
	stepCtx := ctx
	if step.Timeout.IsSome() {
		timeoutCtx, cancel := context.WithTimeout(ctx, step.Timeout.Unwrap())
		defer cancel()
		stepCtx = timeoutCtx
	}

	stepResult := types.StepResult{
		StepID:      step.ID,
		TaskResults: make([]types.ExecutionResult, 0, len(step.Tasks)),
		Status:      types.ExecutionStatusInProgress,
		StartTime:   time.Now(),
		RetryCount:  0,
	}

	o.logger.Info("开始执行步骤",
		"step_id", step.ID,
		"mode", step.Mode,
		"task_count", len(step.Tasks))

	var err error
	for retryCount := 0; retryCount <= step.MaxRetries; retryCount++ {
		stepResult.RetryCount = retryCount

		switch step.Mode {
		case types.ExecutionModeSerial:
			err = o.executeTasksSerial(stepCtx, step.Tasks, &stepResult)
		case types.ExecutionModeParallel:
			err = o.executeTasksParallel(stepCtx, step.Tasks, &stepResult)
		case types.ExecutionModeMixed:
			err = o.executeTasksMixed(stepCtx, step.Tasks, &stepResult)
		default:
			err = fmt.Errorf("不支持的执行模式: %s", step.Mode)
		}

		if err == nil {
			stepResult.Status = types.ExecutionStatusSuccess
			break
		}

		if retryCount < step.MaxRetries {
			o.logger.Warn("步骤执行失败，准备重试",
				"step_id", step.ID,
				"retry_count", retryCount,
				"error", err)

			stepResult.ErrorMessages = append(stepResult.ErrorMessages, err.Error())

			// 重试退避
			select {
			case <-time.After(time.Duration(retryCount+1) * time.Second):
			case <-stepCtx.Done():
				stepResult.Status = types.ExecutionStatusCancelled
				stepResult.EndTime = time.Now()
				return stepResult, stepCtx.Err()
			}
		}
	}

	if err != nil {
		stepResult.Status = types.ExecutionStatusFailure
		stepResult.ErrorMessages = append(stepResult.ErrorMessages, err.Error())
	}

	stepResult.EndTime = time.Now()

	o.logger.Info("步骤执行完成",
		"step_id", step.ID,
		"status", stepResult.Status,
		"duration", stepResult.EndTime.Sub(stepResult.StartTime),
		"task_count", len(stepResult.TaskResults))

	return stepResult, err
}

// executeTasksSerial 串行执行任务
func (o *smartTaskOrchestrator) executeTasksSerial(ctx context.Context, tasks []types.Task, stepResult *types.StepResult) error {
	for _, task := range tasks {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// 获取并发控制信号量
		select {
		case o.semaphore <- struct{}{}:
		case <-ctx.Done():
			return ctx.Err()
		}

		result, err := o.executeTaskWithTimeout(ctx, task)
		<-o.semaphore // 释放信号量

		stepResult.TaskResults = append(stepResult.TaskResults, result)

		if err != nil && result.Status == types.ExecutionStatusFailure {
			return fmt.Errorf("任务 %s 执行失败: %w", task.ID, err)
		}
	}
	return nil
}

// executeTasksParallel 并行执行任务
func (o *smartTaskOrchestrator) executeTasksParallel(ctx context.Context, tasks []types.Task, stepResult *types.StepResult) error {
	var wg sync.WaitGroup
	resultChan := make(chan types.ExecutionResult, len(tasks))
	errorChan := make(chan error, len(tasks))

	for _, task := range tasks {
		wg.Add(1)
		go func(t types.Task) {
			defer wg.Done()

			// 获取并发控制信号量
			select {
			case o.semaphore <- struct{}{}:
			case <-ctx.Done():
				errorChan <- ctx.Err()
				return
			}
			defer func() { <-o.semaphore }()

			result, err := o.executeTaskWithTimeout(ctx, t)
			resultChan <- result
			if err != nil {
				errorChan <- err
			}
		}(task)
	}

	// 等待所有任务完成
	go func() {
		wg.Wait()
		close(resultChan)
		close(errorChan)
	}()

	// 收集结果
	var lastError error
	for result := range resultChan {
		stepResult.TaskResults = append(stepResult.TaskResults, result)
	}

	for err := range errorChan {
		lastError = err
	}

	return lastError
}

// executeTasksMixed 混合执行任务（根据任务依赖关系）
func (o *smartTaskOrchestrator) executeTasksMixed(ctx context.Context, tasks []types.Task, stepResult *types.StepResult) error {
	// 这里可以实现更复杂的依赖关系解析和混合执行逻辑
	// 目前简化为并行执行
	return o.executeTasksParallel(ctx, tasks, stepResult)
}

// executeTaskWithTimeout 执行单个任务（带超时）
func (o *smartTaskOrchestrator) executeTaskWithTimeout(ctx context.Context, task types.Task) (types.ExecutionResult, error) {
	taskCtx, cancel := context.WithTimeout(ctx, o.config.TaskExecutionTimeout)
	defer cancel()

	o.logger.Debug("开始执行任务",
		"task_id", task.ID,
		"task_type", task.Type,
		"agent_type", task.AgentType)

	result, err := o.agentDispatcher.DispatchTask(taskCtx, task)
	if err != nil {
		o.logger.Error("任务执行失败",
			"task_id", task.ID,
			"error", err)

		// 创建失败结果
		result = types.ExecutionResult{
			TaskID:    task.ID,
			Status:    types.ExecutionStatusFailure,
			Error:     err.Error(),
			StartTime: time.Now(),
			EndTime:   time.Now(),
			Metadata:  map[string]any{"error": err.Error()},
		}
	}

	return result, err
}

// CreateExecutionPlan 创建执行计划
func (o *smartTaskOrchestrator) CreateExecutionPlan(ctx context.Context, tasks []types.Task, strategy types.ExecutionMode) (types.ExecutionPlan, error) {
	planID := uuid.New().String()

	o.logger.Info("创建执行计划",
		"plan_id", planID,
		"task_count", len(tasks),
		"strategy", strategy)

	plan := types.ExecutionPlan{
		ID:        planID,
		CreatedAt: time.Now(),
		Context:   map[string]any{"strategy": strategy},
		ContinuationStrategy: types.ContinuationStrategy{
			MaxIterations:        5,
			IdleThreshold:        30 * time.Second,
			FeedbackAnalysisType: types.FeedbackAnalysisLLM,
		},
	}

	// 根据策略创建执行步骤
	switch strategy {
	case types.ExecutionModeSerial:
		for i, task := range tasks {
			step := types.ExecutionStep{
				ID:                fmt.Sprintf("step_%d", i),
				Mode:              types.ExecutionModeSerial,
				Tasks:             []types.Task{task},
				MaxRetries:        2,
				ContinueOnFailure: false,
			}
			plan.Steps = append(plan.Steps, step)
		}
	case types.ExecutionModeParallel:
		step := types.ExecutionStep{
			ID:                "parallel_step",
			Mode:              types.ExecutionModeParallel,
			Tasks:             tasks,
			MaxRetries:        2,
			ContinueOnFailure: true,
		}
		plan.Steps = []types.ExecutionStep{step}
	case types.ExecutionModeMixed:
		// 简单的混合策略：按优先级分组
		plan.Steps = o.createMixedSteps(tasks)
	}

	return plan, nil
}

// createMixedSteps 创建混合执行步骤
func (o *smartTaskOrchestrator) createMixedSteps(tasks []types.Task) []types.ExecutionStep {
	// 按优先级分组任务
	priorityGroups := make(map[int][]types.Task)
	for _, task := range tasks {
		priority := task.Priority
		priorityGroups[priority] = append(priorityGroups[priority], task)
	}

	// 为每个优先级组创建步骤
	var steps []types.ExecutionStep
	stepIndex := 0
	for priority := 10; priority >= 0; priority-- { // 从高优先级到低优先级
		if tasks, exists := priorityGroups[priority]; exists {
			mode := types.ExecutionModeParallel
			if len(tasks) == 1 {
				mode = types.ExecutionModeSerial
			}

			step := types.ExecutionStep{
				ID:                fmt.Sprintf("priority_%d_step_%d", priority, stepIndex),
				Mode:              mode,
				Tasks:             tasks,
				MaxRetries:        2,
				ContinueOnFailure: true,
			}
			steps = append(steps, step)
			stepIndex++
		}
	}

	return steps
}

// MonitorExecution 监控执行状态
func (o *smartTaskOrchestrator) MonitorExecution(ctx context.Context, planID string) (types.ExecutionState, error) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	if state, exists := o.activeExecutions[planID]; exists {
		return *state, nil
	}

	return types.ExecutionState{}, fmt.Errorf("执行计划 %s 不存在或已完成", planID)
}

// CancelExecution 取消执行
func (o *smartTaskOrchestrator) CancelExecution(ctx context.Context, planID string) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if state, exists := o.activeExecutions[planID]; exists {
		state.Status = types.ExecutionStatusCancelled
		o.logger.Info("取消执行计划", "plan_id", planID)
		return nil
	}

	return fmt.Errorf("执行计划 %s 不存在或已完成", planID)
}

// GetExecutionHistory 获取执行历史
func (o *smartTaskOrchestrator) GetExecutionHistory(ctx context.Context, limit int) ([]types.ExecutionState, error) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	if limit <= 0 || limit > len(o.executionHistory) {
		limit = len(o.executionHistory)
	}

	start := len(o.executionHistory) - limit
	if start < 0 {
		start = 0
	}

	return o.executionHistory[start:], nil
}

// OptimizeExecutionPlan 优化执行计划
func (o *smartTaskOrchestrator) OptimizeExecutionPlan(ctx context.Context, plan types.ExecutionPlan) (types.ExecutionPlan, error) {
	if !o.config.EnableOptimization {
		return plan, nil
	}

	// 基于历史性能数据进行优化
	// 这里可以实现更复杂的优化逻辑
	o.logger.Info("优化执行计划", "plan_id", plan.ID)

	// 简单的优化：调整并发数
	optimizedPlan := plan
	for i, step := range optimizedPlan.Steps {
		if step.Mode == types.ExecutionModeParallel && len(step.Tasks) > o.config.MaxConcurrentTasks {
			// 将大的并行步骤拆分为多个较小的步骤
			optimizedPlan.Steps = o.splitLargeParallelStep(step, i, optimizedPlan.Steps)
		}
	}

	return optimizedPlan, nil
}

// splitLargeParallelStep 拆分大的并行步骤
func (o *smartTaskOrchestrator) splitLargeParallelStep(step types.ExecutionStep, index int, steps []types.ExecutionStep) []types.ExecutionStep {
	chunkSize := o.config.MaxConcurrentTasks
	var newSteps []types.ExecutionStep

	// 添加之前的步骤
	newSteps = append(newSteps, steps[:index]...)

	// 拆分当前步骤
	for i := 0; i < len(step.Tasks); i += chunkSize {
		end := i + chunkSize
		if end > len(step.Tasks) {
			end = len(step.Tasks)
		}

		newStep := types.ExecutionStep{
			ID:                fmt.Sprintf("%s_chunk_%d", step.ID, i/chunkSize),
			Mode:              types.ExecutionModeParallel,
			Tasks:             step.Tasks[i:end],
			Dependencies:      step.Dependencies,
			MaxRetries:        step.MaxRetries,
			Timeout:           step.Timeout,
			ContinueOnFailure: step.ContinueOnFailure,
		}
		newSteps = append(newSteps, newStep)
	}

	// 添加之后的步骤
	newSteps = append(newSteps, steps[index+1:]...)

	return newSteps
}

// EstimateExecutionTime 估算执行时间
func (o *smartTaskOrchestrator) EstimateExecutionTime(ctx context.Context, plan types.ExecutionPlan) (time.Duration, error) {
	var totalDuration time.Duration

	for _, step := range plan.Steps {
		var stepDuration time.Duration

		switch step.Mode {
		case types.ExecutionModeSerial:
			// 串行执行：累加所有任务时间
			for _, task := range step.Tasks {
				taskDuration := o.estimateTaskDuration(task)
				stepDuration += taskDuration
			}
		case types.ExecutionModeParallel:
			// 并行执行：取最长的任务时间
			var maxTaskDuration time.Duration
			for _, task := range step.Tasks {
				taskDuration := o.estimateTaskDuration(task)
				if taskDuration > maxTaskDuration {
					maxTaskDuration = taskDuration
				}
			}
			stepDuration = maxTaskDuration
		case types.ExecutionModeMixed:
			// 混合模式：估算为并行模式
			var maxTaskDuration time.Duration
			for _, task := range step.Tasks {
				taskDuration := o.estimateTaskDuration(task)
				if taskDuration > maxTaskDuration {
					maxTaskDuration = taskDuration
				}
			}
			stepDuration = maxTaskDuration
		}

		totalDuration += stepDuration
	}

	// 添加一些缓冲时间（20%）
	totalDuration = time.Duration(float64(totalDuration) * 1.2)

	return totalDuration, nil
}

// estimateTaskDuration 估算单个任务执行时间
func (o *smartTaskOrchestrator) estimateTaskDuration(task types.Task) time.Duration {
	// 基于任务类型和历史数据估算
	// 这里使用简单的默认值，实际可以基于历史统计
	switch task.AgentType {
	case types.AgentTypeGUI:
		return 10 * time.Second
	case types.AgentTypeReAct:
		return 30 * time.Second
	default:
		return 15 * time.Second
	}
}

// GetResourceUsage 获取资源使用情况
func (o *smartTaskOrchestrator) GetResourceUsage(ctx context.Context) (types.SystemMetrics, error) {
	if !o.config.EnableMetrics {
		return types.SystemMetrics{}, nil
	}

	// 计算当前活跃Agent数量
	activeAgentCount := 0
	if agents, err := o.agentDispatcher.GetAvailableAgents(ctx); err == nil {
		activeAgentCount = len(agents)
	}

	// 获取系统资源使用情况
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	metrics := types.SystemMetrics{
		CPUUsage:            0.0, // 需要实际的CPU监控
		MemoryUsage:         float64(memStats.Alloc) / float64(memStats.Sys) * 100,
		ActiveAgentCount:    activeAgentCount,
		AverageResponseTime: o.calculateAverageResponseTime(),
		ErrorRate:           o.calculateErrorRate(),
		ThroughputPerHour:   o.calculateThroughput(),
	}

	o.mu.Lock()
	o.resourceMetrics = metrics
	o.mu.Unlock()

	return metrics, nil
}

// countTotalTasks 计算计划中的总任务数
func (o *smartTaskOrchestrator) countTotalTasks(steps []types.ExecutionStep) int {
	total := 0
	for _, step := range steps {
		total += len(step.Tasks)
	}
	return total
}

// addExecutionHistory 添加执行历史
func (o *smartTaskOrchestrator) addExecutionHistory(state types.ExecutionState) {
	if len(o.executionHistory) >= o.config.MaxExecutionHistory {
		o.executionHistory = o.executionHistory[1:]
	}
	o.executionHistory = append(o.executionHistory, state)
}

// evaluateContinuation 评估是否继续执行
func (o *smartTaskOrchestrator) evaluateContinuation(ctx context.Context, plan types.ExecutionPlan, state types.ExecutionState) (bool, Option[types.ExecutionPlan], error) {
	if o.continuousDecision == nil {
		// 如果没有持续决策引擎，使用简单逻辑
		return state.CurrentStepIndex < len(plan.Steps), None[types.ExecutionPlan](), nil
	}

	// 构建持续决策上下文
	decisionContext := types.ContinuousDecisionContext{
		ExecutionState: state,
		StepResults:    state.StepResults,
		SystemMetrics:  o.resourceMetrics,
		Timestamp:      time.Now(),
	}

	// 如果有反馈分析器，分析输出
	if o.feedbackAnalyzer != nil {
		var allResults []types.ExecutionResult
		for _, stepResult := range state.StepResults {
			allResults = append(allResults, stepResult.TaskResults...)
		}

		if analysis, err := o.feedbackAnalyzer.AnalyzeExecutionResults(ctx, allResults); err == nil {
			decisionContext.AgentOutputAnalysis = analysis
		}
	}

	// 进行持续决策
	result, err := o.continuousDecision.MakeContinuousDecision(ctx, decisionContext)
	if err != nil {
		return false, None[types.ExecutionPlan](), err
	}

	return result.ShouldContinue, result.NextExecutionPlan, nil
}

// calculateAverageResponseTime 计算平均响应时间
func (o *smartTaskOrchestrator) calculateAverageResponseTime() time.Duration {
	if len(o.executionHistory) == 0 {
		return 0
	}

	var totalDuration time.Duration
	totalTasks := 0

	for _, execution := range o.executionHistory {
		for _, stepResult := range execution.StepResults {
			totalDuration += stepResult.EndTime.Sub(stepResult.StartTime)
			totalTasks += len(stepResult.TaskResults)
		}
	}

	if totalTasks == 0 {
		return 0
	}

	return totalDuration / time.Duration(totalTasks)
}

// calculateErrorRate 计算错误率
func (o *smartTaskOrchestrator) calculateErrorRate() float64 {
	if len(o.executionHistory) == 0 {
		return 0.0
	}

	totalTasks := 0
	failedTasks := 0

	for _, execution := range o.executionHistory {
		totalTasks += execution.TotalTaskCount
		failedTasks += execution.FailedTaskCount
	}

	if totalTasks == 0 {
		return 0.0
	}

	return float64(failedTasks) / float64(totalTasks) * 100
}

// calculateThroughput 计算吞吐量
func (o *smartTaskOrchestrator) calculateThroughput() int {
	if len(o.executionHistory) == 0 {
		return 0
	}

	// 计算最近一小时的任务完成数
	oneHourAgo := time.Now().Add(-time.Hour)
	completedTasks := 0

	for _, execution := range o.executionHistory {
		if execution.StartTime.After(oneHourAgo) {
			completedTasks += execution.CompletedTaskCount
		}
	}

	return completedTasks
}
