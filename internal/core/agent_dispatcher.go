package core

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"sync"
	"time"
)

// SmartAgentDispatcher 智能Agent调度器实现
type SmartAgentDispatcher struct {
	agents       map[string]Agent         // 所有注册的Agent实例 (key: Agent ID)
	agentsByType map[AgentType][]string   // 按类型分组的Agent ID列表
	agentStatus  map[string]AgentStatus   // Agent状态 (key: Agent ID)
	taskQueue    chan TaskDispatchRequest // 任务队列
	resultChan   chan ExecutionResult     // 结果通道
	logger       *slog.Logger
	config       DispatcherConfig
	mu           sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
}

// DispatcherConfig 调度器配置
type DispatcherConfig struct {
	MaxConcurrentTasks int           `json:"max_concurrent_tasks"` // 最大并发任务数
	TaskQueueSize      int           `json:"task_queue_size"`      // 任务队列大小
	ResultChanSize     int           `json:"result_chan_size"`     // 结果通道大小
	TaskTimeout        time.Duration `json:"task_timeout"`         // 任务超时时间
	RetryAttempts      int           `json:"retry_attempts"`       // 重试次数
	RetryDelay         time.Duration `json:"retry_delay"`          // 重试延迟
}

// AgentStatus Agent状态信息
type AgentStatus struct {
	IsAvailable     bool      `json:"is_available"`  // 是否可用
	LastUsed        time.Time `json:"last_used"`     // 最后使用时间
	TaskCount       int       `json:"task_count"`    // 执行任务数
	SuccessCount    int       `json:"success_count"` // 成功任务数
	FailureCount    int       `json:"failure_count"` // 失败任务数
	AverageExecTime float64   `json:"avg_exec_time"` // 平均执行时间(秒)
	HealthStatus    string    `json:"health_status"` // 健康状态
}

// TaskDispatchRequest 任务分发请求
type TaskDispatchRequest struct {
	Task           Task
	RequestID      string
	RequestTime    time.Time
	RetryCount     int
	ResultCallback func(ExecutionResult, error)
	AgentID        string
}

// DefaultDispatcherConfig 返回默认调度器配置
func DefaultDispatcherConfig() DispatcherConfig {
	return DispatcherConfig{
		MaxConcurrentTasks: 3,
		TaskQueueSize:      100,
		ResultChanSize:     50,
		TaskTimeout:        5 * time.Minute,
		RetryAttempts:      2,
		RetryDelay:         10 * time.Second,
	}
}

// NewSmartAgentDispatcher 创建新的智能Agent调度器
func NewSmartAgentDispatcher(config DispatcherConfig, logger *slog.Logger) *SmartAgentDispatcher {
	if logger == nil {
		logger = slog.Default().WithGroup("agent_dispatcher")
	}

	ctx, cancel := context.WithCancel(context.Background())

	dispatcher := &SmartAgentDispatcher{
		agents:       make(map[string]Agent),
		agentsByType: make(map[AgentType][]string),
		agentStatus:  make(map[string]AgentStatus),
		taskQueue:    make(chan TaskDispatchRequest, config.TaskQueueSize),
		resultChan:   make(chan ExecutionResult, config.ResultChanSize),
		logger:       logger,
		config:       config,
		ctx:          ctx,
		cancel:       cancel,
	}

	// 启动任务处理协程
	dispatcher.startWorkers()

	return dispatcher
}

// RegisterAgent 注册Agent
func (d *SmartAgentDispatcher) RegisterAgent(agent Agent) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if agent == nil {
		return fmt.Errorf("Agent不能为空")
	}

	agentType := agent.Type()
	agentID := fmt.Sprintf("%s_%s_%d", agentType, agent.Name(), time.Now().UnixNano())
	d.agents[agentID] = agent
	d.agentsByType[agentType] = append(d.agentsByType[agentType], agentID)
	d.agentStatus[agentID] = AgentStatus{
		IsAvailable:     true,
		LastUsed:        time.Time{},
		TaskCount:       0,
		SuccessCount:    0,
		FailureCount:    0,
		AverageExecTime: 0.0,
		HealthStatus:    "healthy",
	}

	d.logger.Info("Agent注册成功", "agent_id", agentID, "agent_name", agent.Name(), "agent_type", agentType)
	return nil
}

// RegisterAgentWithID 注册Agent并返回Agent ID
func (d *SmartAgentDispatcher) RegisterAgentWithID(agent Agent) (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if agent == nil {
		return "", fmt.Errorf("Agent不能为空")
	}

	agentType := agent.Type()
	agentID := fmt.Sprintf("%s_%s_%d", agentType, agent.Name(), time.Now().UnixNano())
	d.agents[agentID] = agent
	d.agentsByType[agentType] = append(d.agentsByType[agentType], agentID)
	d.agentStatus[agentID] = AgentStatus{
		IsAvailable:     true,
		LastUsed:        time.Time{},
		TaskCount:       0,
		SuccessCount:    0,
		FailureCount:    0,
		AverageExecTime: 0.0,
		HealthStatus:    "healthy",
	}

	d.logger.Info("Agent注册成功", "agent_id", agentID, "agent_name", agent.Name(), "agent_type", agentType)
	return agentID, nil
}

// UnregisterAgent 注销Agent
func (d *SmartAgentDispatcher) UnregisterAgent(agentID string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if _, exists := d.agents[agentID]; !exists {
		return fmt.Errorf("Agent ID %s 未注册", agentID)
	}

	agent := d.agents[agentID]
	agentType := agent.Type()
	delete(d.agents, agentID)
	delete(d.agentStatus, agentID)

	// Remove from agentsByType
	agentsByType := d.agentsByType[agentType]
	for i, id := range agentsByType {
		if id == agentID {
			d.agentsByType[agentType] = slices.Delete(agentsByType, i, i+1)
			break
		}
	}

	d.logger.Info("Agent注销成功", "agent_id", agentID)
	return nil
}

// DispatchTask 分发任务
func (d *SmartAgentDispatcher) DispatchTask(ctx context.Context, task Task) (ExecutionResult, error) {
	d.logger.Debug("开始分发任务", "task_id", task.ID, "agent_type", task.AgentType)

	// 选择最佳Agent
	agentID := d.selectBestAgent(task.AgentType)
	if agentID == "" {
		return ExecutionResult{}, fmt.Errorf("Agent %s 不可用", task.AgentType)
	}

	// 创建分发请求
	requestID := fmt.Sprintf("req_%d", time.Now().UnixNano())
	request := TaskDispatchRequest{
		Task:        task,
		RequestID:   requestID,
		RequestTime: time.Now(),
		RetryCount:  0,
		AgentID:     agentID, // 添加Agent ID到请求中
	}

	// 创建结果通道
	resultChan := make(chan ExecutionResult, 1)
	errorChan := make(chan error, 1)

	request.ResultCallback = func(result ExecutionResult, err error) {
		if err != nil {
			errorChan <- err
		} else {
			resultChan <- result
		}
	}

	// 发送到任务队列
	select {
	case d.taskQueue <- request:
		d.logger.Debug("任务已加入队列", "request_id", requestID, "agent_id", agentID)
	case <-ctx.Done():
		return ExecutionResult{}, fmt.Errorf("分发任务失败: %w", ctx.Err())
	case <-time.After(5 * time.Second):
		return ExecutionResult{}, fmt.Errorf("任务队列已满，无法分发任务")
	}

	// 等待结果
	select {
	case result := <-resultChan:
		d.logger.Info("任务执行完成", "task_id", task.ID, "status", result.Status, "agent_id", agentID)
		return result, nil
	case err := <-errorChan:
		d.logger.Error("任务执行失败", "task_id", task.ID, "error", err, "agent_id", agentID)
		return ExecutionResult{}, err
	case <-ctx.Done():
		return ExecutionResult{}, fmt.Errorf("任务执行失败: %w", ctx.Err())
	case <-time.After(d.config.TaskTimeout):
		return ExecutionResult{}, fmt.Errorf("任务执行超时")
	}
}

// selectBestAgent 选择最佳的Agent实例
func (d *SmartAgentDispatcher) selectBestAgent(agentType AgentType) string {
	d.mu.RLock()
	defer d.mu.RUnlock()

	agentIDs := d.agentsByType[agentType]
	if len(agentIDs) == 0 {
		return ""
	}

	// 过滤可用的Agent
	availableAgents := []string{}
	for _, agentID := range agentIDs {
		if status, exists := d.agentStatus[agentID]; exists {
			if status.IsAvailable && status.HealthStatus == "healthy" {
				availableAgents = append(availableAgents, agentID)
			}
		}
	}

	if len(availableAgents) == 0 {
		return ""
	}

	// 负载均衡：选择任务数最少的Agent
	bestAgentID := availableAgents[0]
	minTaskCount := d.agentStatus[bestAgentID].TaskCount

	for _, agentID := range availableAgents[1:] {
		if status := d.agentStatus[agentID]; status.TaskCount < minTaskCount {
			bestAgentID = agentID
			minTaskCount = status.TaskCount
		}
	}

	return bestAgentID
}

// GetAgentsByType 获取指定类型的所有Agent
func (d *SmartAgentDispatcher) GetAgentsByType(agentType AgentType) ([]Agent, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	agentIDs := d.agentsByType[agentType]
	agents := make([]Agent, 0, len(agentIDs))

	for _, agentID := range agentIDs {
		if agent, exists := d.agents[agentID]; exists {
			agents = append(agents, agent)
		}
	}

	return agents, nil
}

// GetAgentByID 根据ID获取Agent
func (d *SmartAgentDispatcher) GetAgentByID(agentID string) (Agent, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	agent, exists := d.agents[agentID]
	if !exists {
		return nil, fmt.Errorf("Agent ID %s 未注册", agentID)
	}

	return agent, nil
}

// GetAvailableAgents 获取可用Agent列表
func (d *SmartAgentDispatcher) GetAvailableAgents(ctx context.Context) ([]Agent, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var available []Agent
	for agentID, status := range d.agentStatus {
		if status.IsAvailable && status.HealthStatus == "healthy" {
			if agent, exists := d.agents[agentID]; exists {
				available = append(available, agent)
			}
		}
	}

	return available, nil
}

// GetAvailableAgentTypes 获取可用Agent类型列表 (为了向后兼容)
func (d *SmartAgentDispatcher) GetAvailableAgentTypes() []AgentType {
	d.mu.RLock()
	defer d.mu.RUnlock()

	return slices.Collect(maps.Keys(d.agentsByType))
}

// GetAgentByType 根据类型获取Agent
func (d *SmartAgentDispatcher) GetAgentByType(agentType AgentType) (Agent, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	agentIDs := d.agentsByType[agentType]
	if len(agentIDs) == 0 {
		return nil, fmt.Errorf("没有可用的Agent类型 %s", agentType)
	}

	agentID := agentIDs[0]
	agent, exists := d.agents[agentID]
	if !exists {
		return nil, fmt.Errorf("Agent ID %s 未注册", agentID)
	}

	return agent, nil
}

// IsAgentAvailable 检查指定类型的Agent是否可用
func (d *SmartAgentDispatcher) IsAgentAvailable(ctx context.Context, agentType AgentType) bool {
	return d.isAgentAvailable(agentType)
}

// GetAgentStatus 获取Agent状态
func (d *SmartAgentDispatcher) GetAgentStatus(ctx context.Context, agentID string) (map[string]any, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	status, exists := d.agentStatus[agentID]
	if !exists {
		return nil, fmt.Errorf("Agent ID %s 未注册", agentID)
	}

	result := map[string]any{
		"is_available":  status.IsAvailable,
		"last_used":     status.LastUsed,
		"task_count":    status.TaskCount,
		"success_count": status.SuccessCount,
		"failure_count": status.FailureCount,
		"avg_exec_time": status.AverageExecTime,
		"health_status": status.HealthStatus,
	}

	return result, nil
}

// GetAgentStatusStruct 获取Agent状态结构体 (为了向后兼容)
func (d *SmartAgentDispatcher) GetAgentStatusStruct(agentID string) (AgentStatus, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	status, exists := d.agentStatus[agentID]
	return status, exists
}

// GetAllAgentStatus 获取所有Agent状态
func (d *SmartAgentDispatcher) GetAllAgentStatus() map[string]AgentStatus {
	d.mu.RLock()
	defer d.mu.RUnlock()

	result := make(map[string]AgentStatus)
	maps.Copy(result, d.agentStatus)

	return result
}

// Shutdown 关闭调度器
func (d *SmartAgentDispatcher) Shutdown(ctx context.Context) error {
	d.logger.Info("开始关闭Agent调度器")

	// 取消上下文
	d.cancel()

	// 等待所有工作协程完成
	done := make(chan struct{})
	go func() {
		d.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		d.logger.Info("Agent调度器关闭完成")
		return nil
	case <-ctx.Done():
		d.logger.Warn("Agent调度器关闭超时")
		return fmt.Errorf("agent调度器关闭超时: %w", ctx.Err())
	}
}

// 私有方法

// startWorkers 启动工作协程
func (d *SmartAgentDispatcher) startWorkers() {
	for i := range d.config.MaxConcurrentTasks {
		d.wg.Add(1)
		go d.worker(i)
	}
}

// worker 工作协程
func (d *SmartAgentDispatcher) worker(workerID int) {
	defer d.wg.Done()

	d.logger.Debug("工作协程启动", "worker_id", workerID)

	for {
		select {
		case <-d.ctx.Done():
			d.logger.Debug("工作协程退出", "worker_id", workerID)
			return
		case request := <-d.taskQueue:
			d.processTask(workerID, request)
		}
	}
}

// processTask 处理任务
func (d *SmartAgentDispatcher) processTask(workerID int, request TaskDispatchRequest) {
	d.logger.Debug("开始处理任务", "worker_id", workerID, "task_id", request.Task.ID, "agent_id", request.AgentID)

	startTime := time.Now()

	// 更新Agent状态为忙碌
	d.updateAgentStatusByID(request.AgentID, func(status *AgentStatus) {
		status.IsAvailable = false
		status.LastUsed = startTime
		status.TaskCount++
	})

	// 获取Agent实例
	agent := d.getAgentByID(request.AgentID)
	if agent == nil {
		err := fmt.Errorf("Agent %s 不存在", request.AgentID)
		d.handleTaskFailure(request, err, startTime)
		return
	}

	// 执行任务
	taskCtx, cancel := context.WithTimeout(d.ctx, d.config.TaskTimeout)
	defer cancel()

	result, err := agent.Execute(taskCtx, request.Task, request.Task.Context)

	// 处理执行结果
	if err != nil {
		d.handleTaskFailure(request, err, startTime)
	} else {
		d.handleTaskSuccess(request, result, startTime)
	}

	// 恢复Agent可用状态
	d.updateAgentStatusByID(request.AgentID, func(status *AgentStatus) {
		status.IsAvailable = true
		duration := time.Since(startTime).Seconds()
		status.AverageExecTime = (status.AverageExecTime*float64(status.TaskCount-1) + duration) / float64(status.TaskCount)
	})
}

// handleTaskSuccess 处理任务成功
func (d *SmartAgentDispatcher) handleTaskSuccess(request TaskDispatchRequest, result ExecutionResult, startTime time.Time) {
	duration := time.Since(startTime)
	d.logger.Info("任务执行成功",
		"task_id", request.Task.ID,
		"agent_id", request.AgentID,
		"duration", duration,
		"status", result.Status)

	// 更新成功统计
	d.updateAgentStatusByID(request.AgentID, func(status *AgentStatus) {
		status.SuccessCount++
	})

	// 调用结果回调
	if request.ResultCallback != nil {
		request.ResultCallback(result, nil)
	}
}

// handleTaskFailure 处理任务失败
func (d *SmartAgentDispatcher) handleTaskFailure(request TaskDispatchRequest, err error, startTime time.Time) {
	duration := time.Since(startTime)
	d.logger.Error("任务执行失败",
		"task_id", request.Task.ID,
		"agent_id", request.AgentID,
		"duration", duration,
		"error", err,
		"retry_count", request.RetryCount)

	// 检查是否需要重试
	if request.RetryCount < d.config.RetryAttempts {
		d.retryTask(request, err)
		return
	}

	// 更新失败统计
	d.updateAgentStatusByID(request.AgentID, func(status *AgentStatus) {
		status.FailureCount++
	})

	// 调用结果回调
	if request.ResultCallback != nil {
		request.ResultCallback(ExecutionResult{}, err)
	}
}

// retryTask 重试任务
func (d *SmartAgentDispatcher) retryTask(request TaskDispatchRequest, lastErr error) {
	request.RetryCount++
	d.logger.Info("准备重试任务",
		"task_id", request.Task.ID,
		"retry_count", request.RetryCount,
		"last_error", lastErr)

	// 延迟重试
	time.AfterFunc(d.config.RetryDelay, func() {
		select {
		case d.taskQueue <- request:
			d.logger.Debug("重试任务已加入队列", "task_id", request.Task.ID)
		case <-d.ctx.Done():
			d.logger.Warn("重试任务时调度器已关闭", "task_id", request.Task.ID)
		default:
			d.logger.Error("重试任务时队列已满", "task_id", request.Task.ID)
			if request.ResultCallback != nil {
				request.ResultCallback(ExecutionResult{}, fmt.Errorf("重试失败: 队列已满"))
			}
		}
	})
}

// isAgentAvailable 检查Agent是否可用
func (d *SmartAgentDispatcher) isAgentAvailable(agentType AgentType) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()

	agentIDs := d.agentsByType[agentType]
	if len(agentIDs) == 0 {
		return false
	}

	agentID := agentIDs[0]
	status, exists := d.agentStatus[agentID]
	if !exists {
		return false
	}

	return status.IsAvailable && status.HealthStatus == "healthy"
}

// getAgentByID 获取指定ID的Agent实例
func (d *SmartAgentDispatcher) getAgentByID(agentID string) Agent {
	d.mu.RLock()
	defer d.mu.RUnlock()

	return d.agents[agentID]
}

// updateAgentStatusByID 根据Agent ID更新状态
func (d *SmartAgentDispatcher) updateAgentStatusByID(agentID string, updateFunc func(*AgentStatus)) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if status, exists := d.agentStatus[agentID]; exists {
		updateFunc(&status)
		d.agentStatus[agentID] = status
	}
}
