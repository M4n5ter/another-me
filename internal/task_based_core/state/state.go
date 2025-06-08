package state

import (
	"context"
	"log/slog"
	"sync"
	"time"

	. "github.com/m4n5ter/another-me/pkg/option"
)

// SystemState 表示整个系统的主要状态
type SystemState int

const (
	SystemStateIdle         SystemState = iota // 空闲状态，等待任务
	SystemStateAnalyzing                       // 分析用户请求
	SystemStatePlanning                        // 制定执行计划
	SystemStateExecuting                       // 执行任务中
	SystemStateEvaluating                      // 评估执行结果
	SystemStateLearning                        // 学习和优化
	SystemStateError                           // 系统错误状态
	SystemStateMaintenance                     // 维护状态
	SystemStateShuttingDown                    // 关闭中
	SystemStateUnknown      = "未知系统状态"
)

// String 返回系统状态的字符串表示
func (s SystemState) String() string {
	switch s {
	case SystemStateIdle:
		return "空闲"
	case SystemStateAnalyzing:
		return "分析中"
	case SystemStatePlanning:
		return "规划中"
	case SystemStateExecuting:
		return "执行中"
	case SystemStateEvaluating:
		return "评估中"
	case SystemStateLearning:
		return "学习中"
	case SystemStateError:
		return "错误"
	case SystemStateMaintenance:
		return "维护中"
	case SystemStateShuttingDown:
		return "关闭中"
	default:
		return SystemStateUnknown
	}
}

// TaskState 表示任务的执行状态
type TaskState int

const (
	TaskStatePending     TaskState = iota // 等待执行
	TaskStateAnalyzing                    // 分析任务
	TaskStateDecomposing                  // 分解子任务
	TaskStateScheduling                   // 调度Worker
	TaskStateRunning                      // 运行中
	TaskStateSuspended                    // 暂停
	TaskStateCompleted                    // 已完成
	TaskStateFailed                       // 执行失败
	TaskStateCancelled                    // 已取消
	TaskStateRetrying                     // 重试中
	TaskStateUnknown     = "未知任务状态"
)

// String 返回任务状态的字符串表示
func (t TaskState) String() string {
	switch t {
	case TaskStatePending:
		return "等待执行"
	case TaskStateAnalyzing:
		return "分析任务"
	case TaskStateDecomposing:
		return "分解子任务"
	case TaskStateScheduling:
		return "调度Worker"
	case TaskStateRunning:
		return "运行中"
	case TaskStateSuspended:
		return "暂停"
	case TaskStateCompleted:
		return "已完成"
	case TaskStateFailed:
		return "执行失败"
	case TaskStateCancelled:
		return "已取消"
	case TaskStateRetrying:
		return "重试中"
	default:
		return TaskStateUnknown
	}
}

// WorkerState 表示Worker的状态
type WorkerState int

const (
	WorkerStateIdle         WorkerState = iota // 空闲
	WorkerStateInitializing                    // 初始化中
	WorkerStateRunning                         // 运行中
	WorkerStateBusy                            // 忙碌
	WorkerStateWaiting                         // 等待资源
	WorkerStateCompleted                       // 任务完成
	WorkerStateError                           // 错误状态
	WorkerStateTerminating                     // 终止中
	WorkerStateTerminated                      // 已终止
	WorkerStateUnknown      = "未知Worker状态"
)

// String 返回Worker状态的字符串表示
func (w WorkerState) String() string {
	switch w {
	case WorkerStateIdle:
		return "空闲"
	case WorkerStateInitializing:
		return "初始化中"
	case WorkerStateRunning:
		return "运行中"
	case WorkerStateBusy:
		return "忙碌"
	case WorkerStateWaiting:
		return "等待资源"
	case WorkerStateCompleted:
		return "任务完成"
	case WorkerStateError:
		return "错误状态"
	case WorkerStateTerminating:
		return "终止中"
	case WorkerStateTerminated:
		return "已终止"
	default:
		return WorkerStateUnknown
	}
}

// Priority 任务优先级
type Priority int

const (
	PriorityLow Priority = iota
	PriorityNormal
	PriorityHigh
	PriorityCritical
	PriorityUnknown = "未知优先级"
)

// String 返回优先级的字符串表示
func (p Priority) String() string {
	switch p {
	case PriorityLow:
		return "低"
	case PriorityNormal:
		return "正常"
	case PriorityHigh:
		return "高"
	case PriorityCritical:
		return "紧急"
	default:
		return PriorityUnknown
	}
}

// TaskInfo 任务信息
type TaskInfo struct {
	ID          string            `json:"id"`           // 任务ID
	Name        string            `json:"name"`         // 任务名称
	Description string            `json:"description"`  // 任务描述
	State       TaskState         `json:"state"`        // 任务状态
	Priority    Priority          `json:"priority"`     // 优先级
	Progress    float64           `json:"progress"`     // 进度百分比 (0-100)
	CreatedAt   time.Time         `json:"created_at"`   // 创建时间
	StartedAt   Option[time.Time] `json:"started_at"`   // 开始时间
	CompletedAt Option[time.Time] `json:"completed_at"` // 完成时间
	Error       Option[string]    `json:"error"`        // 错误信息
	Metadata    map[string]any    `json:"metadata"`     // 任务元数据
	SubTasks    []string          `json:"sub_tasks"`    // 子任务ID列表
	TaskResult  Option[any]       `json:"task_result"`  // 任务结果，如果当前任务是末端任务，则保存结果
	ParentTask  Option[string]    `json:"parent_task"`  // 父任务ID
}

// WorkerInfo Worker信息
type WorkerInfo struct {
	ID          string             `json:"id"`           // Worker ID
	Type        string             `json:"type"`         // Worker类型 (web_ui, data_analysis, file_system, temporary)
	State       WorkerState        `json:"state"`        // Worker状态
	CurrentTask Option[string]     `json:"current_task"` // 当前执行的任务ID
	CreatedAt   time.Time          `json:"created_at"`   // 创建时间
	LastActive  time.Time          `json:"last_active"`  // 最后活跃时间
	Tools       []string           `json:"tools"`        // 可用工具列表
	Metadata    map[string]any     `json:"metadata"`     // Worker元数据
	Performance PerformanceMetrics `json:"performance"`  // 性能指标
}

// PerformanceMetrics 性能指标
type PerformanceMetrics struct {
	TasksCompleted int            `json:"tasks_completed"` // 完成任务数
	TasksFailed    int            `json:"tasks_failed"`    // 失败任务数
	AvgDuration    time.Duration  `json:"avg_duration"`    // 平均执行时间
	LastError      Option[string] `json:"last_error"`      // 最后错误
	CPUUsage       float64        `json:"cpu_usage"`       // CPU使用率
	MemoryUsage    float64        `json:"memory_usage"`    // 内存使用率
}

// SystemInfo 系统整体信息
type SystemInfo struct {
	State          SystemState    `json:"state"`           // 系统状态
	ActiveTasks    int            `json:"active_tasks"`    // 活跃任务数
	PendingTasks   int            `json:"pending_tasks"`   // 等待任务数
	CompletedTasks int            `json:"completed_tasks"` // 完成任务数
	FailedTasks    int            `json:"failed_tasks"`    // 失败任务数
	ActiveWorkers  int            `json:"active_workers"`  // 活跃Worker数
	IdleWorkers    int            `json:"idle_workers"`    // 空闲Worker数
	SystemLoad     float64        `json:"system_load"`     // 系统负载
	Uptime         time.Duration  `json:"uptime"`          // 运行时间
	LastError      Option[string] `json:"last_error"`      // 最后错误
	Metadata       map[string]any `json:"metadata"`        // 系统元数据
}

// StateTransition 状态转换事件
type StateTransition struct {
	EntityType   string    `json:"entity_type"`   // 实体类型 (system, task, worker)
	EntityID     string    `json:"entity_id"`     // 实体ID
	FromState    string    `json:"from_state"`    // 原状态
	ToState      string    `json:"to_state"`      // 目标状态
	Reason       string    `json:"reason"`        // 转换原因
	Timestamp    time.Time `json:"timestamp"`     // 转换时间
	TriggeredBy  string    `json:"triggered_by"`  // 触发者
	Success      bool      `json:"success"`       // 是否成功
	ErrorMessage string    `json:"error_message"` // 错误消息
}

// StateManager 状态管理器
type StateManager struct {
	mu             sync.RWMutex
	logger         *slog.Logger
	systemInfo     SystemInfo
	tasks          map[string]*TaskInfo
	workers        map[string]*WorkerInfo
	transitions    []StateTransition
	maxTransitions int
	startTime      time.Time
}

// NewStateManager 创建新的状态管理器
func NewStateManager() *StateManager {
	return &StateManager{
		logger:         slog.Default().WithGroup("state_manager"),
		systemInfo:     SystemInfo{State: SystemStateIdle, Metadata: make(map[string]any)},
		tasks:          make(map[string]*TaskInfo),
		workers:        make(map[string]*WorkerInfo),
		transitions:    make([]StateTransition, 0),
		maxTransitions: 1000, // 最多保留1000条状态转换记录
		startTime:      time.Now(),
	}
}

// 接口实现检查
var _ StateManagerInterface = (*StateManager)(nil)

// StateManagerInterface 状态管理器接口
type StateManagerInterface interface {
	// 系统状态管理
	GetSystemState() SystemState
	SetSystemState(state SystemState, reason string) error
	GetSystemInfo() SystemInfo
	UpdateSystemMetadata(key string, value any)

	// 任务状态管理
	CreateTask(task *TaskInfo) error
	GetTask(taskID string) (*TaskInfo, error)
	UpdateTaskState(taskID string, state TaskState, reason string) error
	UpdateTaskProgress(taskID string, progress float64, result Option[any], errorMsg Option[string]) error
	DeleteTask(taskID string) error
	ListTasks() []*TaskInfo
	ListTasksByState(state TaskState) []*TaskInfo

	// Worker状态管理
	RegisterWorker(worker *WorkerInfo) error
	GetWorker(workerID string) (*WorkerInfo, error)
	UpdateWorkerState(workerID string, state WorkerState, reason string) error
	AssignTaskToWorker(workerID, taskID string) error
	UnregisterWorker(workerID, reason string) error
	ListWorkers() []*WorkerInfo
	ListWorkersByState(state WorkerState) []*WorkerInfo

	// 状态转换历史
	GetStateTransitions(limit int) []StateTransition
	GetStateTransitionsByEntity(entityType, entityID string, limit int) []StateTransition

	// 统计信息
	GetStatistics() map[string]any
}

// GetSystemState 获取系统当前状态
func (sm *StateManager) GetSystemState() SystemState {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.systemInfo.State
}

// SetSystemState 设置系统状态
func (sm *StateManager) SetSystemState(state SystemState, reason string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	oldState := sm.systemInfo.State
	sm.systemInfo.State = state

	// 记录状态转换
	transition := StateTransition{
		EntityType:  "system",
		EntityID:    "main",
		FromState:   oldState.String(),
		ToState:     state.String(),
		Reason:      reason,
		Timestamp:   time.Now(),
		TriggeredBy: "system",
		Success:     true,
	}
	sm.addTransition(transition)

	sm.logger.Info("系统状态转换",
		"from", oldState.String(),
		"to", state.String(),
		"reason", reason)

	return nil
}

// GetSystemInfo 获取系统完整信息
func (sm *StateManager) GetSystemInfo() SystemInfo {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	info := sm.systemInfo
	info.Uptime = time.Since(sm.startTime)

	// 统计任务和Worker数量
	for _, task := range sm.tasks {
		switch task.State {
		case TaskStateRunning, TaskStateScheduling:
			info.ActiveTasks++
		case TaskStatePending:
			info.PendingTasks++
		case TaskStateCompleted:
			info.CompletedTasks++
		case TaskStateFailed:
			info.FailedTasks++
		}
	}

	for _, worker := range sm.workers {
		switch worker.State {
		case WorkerStateRunning, WorkerStateBusy:
			info.ActiveWorkers++
		case WorkerStateIdle:
			info.IdleWorkers++
		}
	}

	return info
}

// UpdateSystemMetadata 更新系统元数据
func (sm *StateManager) UpdateSystemMetadata(key string, value any) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.systemInfo.Metadata[key] = value
}

// CreateTask 创建新任务
func (sm *StateManager) CreateTask(task *TaskInfo) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, exists := sm.tasks[task.ID]; exists {
		return NewStateError(ErrorTypeTaskExists, "任务已存在", task.ID)
	}

	// 设置创建时间
	if task.CreatedAt.IsZero() {
		task.CreatedAt = time.Now()
	}

	// 初始化元数据
	if task.Metadata == nil {
		task.Metadata = make(map[string]any)
	}

	sm.tasks[task.ID] = task

	// 记录状态转换
	transition := StateTransition{
		EntityType:  "task",
		EntityID:    task.ID,
		FromState:   "",
		ToState:     task.State.String(),
		Reason:      "任务创建",
		Timestamp:   time.Now(),
		TriggeredBy: "orchestrator",
		Success:     true,
	}
	sm.addTransition(transition)

	sm.logger.Info("任务创建",
		"task_id", task.ID,
		"name", task.Name,
		"priority", task.Priority.String())

	return nil
}

// GetTask 获取任务信息
func (sm *StateManager) GetTask(taskID string) (*TaskInfo, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	task, exists := sm.tasks[taskID]
	if !exists {
		return nil, NewStateError(ErrorTypeTaskNotFound, "任务不存在", taskID)
	}

	// 返回任务的副本
	taskCopy := *task
	return &taskCopy, nil
}

// UpdateTaskState 更新任务状态
func (sm *StateManager) UpdateTaskState(taskID string, state TaskState, reason string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	task, exists := sm.tasks[taskID]
	if !exists {
		return NewStateError(ErrorTypeTaskNotFound, "任务不存在", taskID)
	}

	oldState := task.State
	task.State = state

	// 更新时间戳
	now := time.Now()
	if state == TaskStateRunning && !task.StartedAt.IsSome() {
		task.StartedAt = Some(now)
	}
	if (state == TaskStateCompleted || state == TaskStateFailed || state == TaskStateCancelled) && !task.CompletedAt.IsSome() {
		task.CompletedAt = Some(now)
	}

	// 记录状态转换
	transition := StateTransition{
		EntityType:  "task",
		EntityID:    taskID,
		FromState:   oldState.String(),
		ToState:     state.String(),
		Reason:      reason,
		Timestamp:   now,
		TriggeredBy: "orchestrator",
		Success:     true,
	}
	sm.addTransition(transition)

	sm.logger.Info("任务状态更新",
		"task_id", taskID,
		"from", oldState.String(),
		"to", state.String(),
		"reason", reason)

	return nil
}

// UpdateTaskProgress 更新任务进度
func (sm *StateManager) UpdateTaskProgress(taskID string, progress float64, result Option[any], errorMsg Option[string]) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	task, exists := sm.tasks[taskID]
	if !exists {
		return NewStateError(ErrorTypeTaskNotFound, "任务不存在", taskID)
	}

	// 限制进度在0-100之间
	if progress < 0 {
		progress = 0
	} else if progress > 100 {
		progress = 100
	}

	task.Progress = progress
	task.TaskResult = result
	task.Error = errorMsg
	sm.logger.Debug("任务进度更新",
		"task_id", taskID,
		"progress", progress)

	return nil
}

// DeleteTask 删除任务
func (sm *StateManager) DeleteTask(taskID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, exists := sm.tasks[taskID]; !exists {
		return NewStateError(ErrorTypeTaskNotFound, "任务不存在", taskID)
	}

	delete(sm.tasks, taskID)

	sm.logger.Info("任务删除", "task_id", taskID)
	return nil
}

// ListTasks 列出所有任务
func (sm *StateManager) ListTasks() []*TaskInfo {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	tasks := make([]*TaskInfo, 0, len(sm.tasks))
	for _, task := range sm.tasks {
		taskCopy := *task
		tasks = append(tasks, &taskCopy)
	}
	return tasks
}

// ListTasksByState 按状态列出任务
func (sm *StateManager) ListTasksByState(state TaskState) []*TaskInfo {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	var tasks []*TaskInfo
	for _, task := range sm.tasks {
		if task.State == state {
			taskCopy := *task
			tasks = append(tasks, &taskCopy)
		}
	}
	return tasks
}

// RegisterWorker 注册Worker
func (sm *StateManager) RegisterWorker(worker *WorkerInfo) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, exists := sm.workers[worker.ID]; exists {
		return NewStateError(ErrorTypeWorkerExists, "Worker已存在", worker.ID)
	}

	// 设置创建时间和最后活跃时间
	if worker.CreatedAt.IsZero() {
		worker.CreatedAt = time.Now()
	}
	worker.LastActive = time.Now()

	// 初始化元数据
	if worker.Metadata == nil {
		worker.Metadata = make(map[string]any)
	}

	sm.workers[worker.ID] = worker

	// 记录状态转换
	transition := StateTransition{
		EntityType:  "worker",
		EntityID:    worker.ID,
		FromState:   "",
		ToState:     worker.State.String(),
		Reason:      "Worker注册",
		Timestamp:   time.Now(),
		TriggeredBy: "orchestrator",
		Success:     true,
	}
	sm.addTransition(transition)

	sm.logger.Info("Worker注册",
		"worker_id", worker.ID,
		"type", worker.Type,
		"tools", len(worker.Tools))

	return nil
}

// GetWorker 获取Worker信息
func (sm *StateManager) GetWorker(workerID string) (*WorkerInfo, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	worker, exists := sm.workers[workerID]
	if !exists {
		return nil, NewStateError(ErrorTypeWorkerNotFound, "Worker不存在", workerID)
	}

	// 返回Worker的副本
	workerCopy := *worker
	return &workerCopy, nil
}

// UpdateWorkerState 更新Worker状态
func (sm *StateManager) UpdateWorkerState(workerID string, state WorkerState, reason string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	worker, exists := sm.workers[workerID]
	if !exists {
		return NewStateError(ErrorTypeWorkerNotFound, "Worker不存在", workerID)
	}

	oldState := worker.State
	worker.State = state
	worker.LastActive = time.Now()

	// 记录状态转换
	transition := StateTransition{
		EntityType:  "worker",
		EntityID:    workerID,
		FromState:   oldState.String(),
		ToState:     state.String(),
		Reason:      reason,
		Timestamp:   time.Now(),
		TriggeredBy: "orchestrator",
		Success:     true,
	}
	sm.addTransition(transition)

	sm.logger.Info("Worker状态更新",
		"worker_id", workerID,
		"from", oldState.String(),
		"to", state.String(),
		"reason", reason)

	return nil
}

// AssignTaskToWorker 将任务分配给Worker
func (sm *StateManager) AssignTaskToWorker(workerID, taskID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	worker, exists := sm.workers[workerID]
	if !exists {
		return NewStateError(ErrorTypeWorkerNotFound, "Worker不存在", workerID)
	}

	if _, exists := sm.tasks[taskID]; !exists {
		return NewStateError(ErrorTypeTaskNotFound, "任务不存在", taskID)
	}

	worker.CurrentTask = Some(taskID)
	worker.LastActive = time.Now()

	sm.logger.Info("任务分配",
		"worker_id", workerID,
		"task_id", taskID)

	return nil
}

// UnregisterWorker 注销Worker
func (sm *StateManager) UnregisterWorker(workerID, reason string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, exists := sm.workers[workerID]; !exists {
		return NewStateError(ErrorTypeWorkerNotFound, "Worker不存在", workerID)
	}

	delete(sm.workers, workerID)

	// 记录状态转换
	transition := StateTransition{
		EntityType:  "worker",
		EntityID:    workerID,
		FromState:   "registered",
		ToState:     "unregistered",
		Reason:      reason,
		Timestamp:   time.Now(),
		TriggeredBy: "orchestrator",
		Success:     true,
	}
	sm.addTransition(transition)

	sm.logger.Info("Worker注销", "worker_id", workerID, "reason", reason)
	return nil
}

// ListWorkers 列出所有Worker
func (sm *StateManager) ListWorkers() []*WorkerInfo {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	workers := make([]*WorkerInfo, 0, len(sm.workers))
	for _, worker := range sm.workers {
		workerCopy := *worker
		workers = append(workers, &workerCopy)
	}
	return workers
}

// ListWorkersByState 按状态列出Worker
func (sm *StateManager) ListWorkersByState(state WorkerState) []*WorkerInfo {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	var workers []*WorkerInfo
	for _, worker := range sm.workers {
		if worker.State == state {
			workerCopy := *worker
			workers = append(workers, &workerCopy)
		}
	}
	return workers
}

// GetStateTransitions 获取状态转换历史
func (sm *StateManager) GetStateTransitions(limit int) []StateTransition {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if limit <= 0 || limit > len(sm.transitions) {
		limit = len(sm.transitions)
	}

	// 返回最近的转换记录
	start := len(sm.transitions) - limit
	transitions := make([]StateTransition, limit)
	copy(transitions, sm.transitions[start:])
	return transitions
}

// GetStateTransitionsByEntity 获取特定实体的状态转换历史
func (sm *StateManager) GetStateTransitionsByEntity(entityType, entityID string, limit int) []StateTransition {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	var transitions []StateTransition
	count := 0

	// 从最新的记录开始倒序查找
	for i := len(sm.transitions) - 1; i >= 0 && count < limit; i-- {
		if sm.transitions[i].EntityType == entityType && sm.transitions[i].EntityID == entityID {
			transitions = append([]StateTransition{sm.transitions[i]}, transitions...)
			count++
		}
	}

	return transitions
}

// GetStatistics 获取统计信息
func (sm *StateManager) GetStatistics() map[string]any {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	stats := make(map[string]any)

	// 任务统计
	taskStats := make(map[string]int)
	for _, task := range sm.tasks {
		taskStats[task.State.String()]++
	}
	stats["tasks"] = taskStats

	// Worker统计
	workerStats := make(map[string]int)
	for _, worker := range sm.workers {
		workerStats[worker.State.String()]++
	}
	stats["workers"] = workerStats

	// 系统统计
	stats["system"] = map[string]any{
		"state":             sm.systemInfo.State.String(),
		"uptime_seconds":    int(time.Since(sm.startTime).Seconds()),
		"total_tasks":       len(sm.tasks),
		"total_workers":     len(sm.workers),
		"total_transitions": len(sm.transitions),
	}

	return stats
}

// addTransition 添加状态转换记录（内部方法，调用前需加锁）
func (sm *StateManager) addTransition(transition StateTransition) {
	sm.transitions = append(sm.transitions, transition)

	// 保持转换记录数量在限制内
	if len(sm.transitions) > sm.maxTransitions {
		// 删除最旧的记录
		copy(sm.transitions, sm.transitions[1:])
		sm.transitions = sm.transitions[:sm.maxTransitions]
	}
}

// StateError 状态管理错误
type StateError struct {
	Type     ErrorType `json:"type"`
	Message  string    `json:"message"`
	EntityID string    `json:"entity_id"`
}

// Error 实现error接口
func (e *StateError) Error() string {
	return e.Message
}

// NewStateError 创建状态错误
func NewStateError(errorType ErrorType, message, entityID string) *StateError {
	return &StateError{
		Type:     errorType,
		Message:  message,
		EntityID: entityID,
	}
}

// ErrorType 错误类型
type ErrorType int

const (
	ErrorTypeTaskNotFound ErrorType = iota
	ErrorTypeTaskExists
	ErrorTypeWorkerNotFound
	ErrorTypeWorkerExists
	ErrorTypeInvalidState
	ErrorTypeOperationFailed
)

// String 返回错误类型的字符串表示
func (e ErrorType) String() string {
	switch e {
	case ErrorTypeTaskNotFound:
		return "任务不存在"
	case ErrorTypeTaskExists:
		return "任务已存在"
	case ErrorTypeWorkerNotFound:
		return "Worker不存在"
	case ErrorTypeWorkerExists:
		return "Worker已存在"
	case ErrorTypeInvalidState:
		return "无效状态"
	case ErrorTypeOperationFailed:
		return "操作失败"
	default:
		return "未知错误"
	}
}

// CanTransition 检查状态转换是否合法
func CanTransition(from, to SystemState) bool {
	validTransitions := map[SystemState][]SystemState{
		SystemStateIdle: {
			SystemStateAnalyzing,
			SystemStateMaintenance,
			SystemStateShuttingDown,
		},
		SystemStateAnalyzing: {
			SystemStatePlanning,
			SystemStateIdle,
			SystemStateError,
		},
		SystemStatePlanning: {
			SystemStateExecuting,
			SystemStateAnalyzing,
			SystemStateError,
		},
		SystemStateExecuting: {
			SystemStateEvaluating,
			SystemStateError,
			SystemStatePlanning, // 可以重新规划
		},
		SystemStateEvaluating: {
			SystemStateLearning,
			SystemStateIdle,
			SystemStateError,
		},
		SystemStateLearning: {
			SystemStateIdle,
			SystemStateError,
		},
		SystemStateError: {
			SystemStateIdle,
			SystemStateMaintenance,
			SystemStateShuttingDown,
		},
		SystemStateMaintenance: {
			SystemStateIdle,
			SystemStateShuttingDown,
		},
		SystemStateShuttingDown: {}, // 终态，不可转换
	}

	allowedStates, exists := validTransitions[from]
	if !exists {
		return false
	}

	for _, allowedState := range allowedStates {
		if allowedState == to {
			return true
		}
	}
	return false
}

// CanTransitionTask 检查任务状态转换是否合法
func CanTransitionTask(from, to TaskState) bool {
	validTransitions := map[TaskState][]TaskState{
		TaskStatePending: {
			TaskStateAnalyzing,
			TaskStateCancelled,
		},
		TaskStateAnalyzing: {
			TaskStateDecomposing,
			TaskStateFailed,
			TaskStateCancelled,
		},
		TaskStateDecomposing: {
			TaskStateScheduling,
			TaskStateFailed,
			TaskStateCancelled,
		},
		TaskStateScheduling: {
			TaskStateRunning,
			TaskStateFailed,
			TaskStateCancelled,
		},
		TaskStateRunning: {
			TaskStateCompleted,
			TaskStateFailed,
			TaskStateSuspended,
			TaskStateCancelled,
		},
		TaskStateSuspended: {
			TaskStateRunning,
			TaskStateCancelled,
		},
		TaskStateFailed: {
			TaskStateRetrying,
			TaskStateCancelled,
		},
		TaskStateRetrying: {
			TaskStateRunning,
			TaskStateFailed,
			TaskStateCancelled,
		},
		TaskStateCompleted: {}, // 终态
		TaskStateCancelled: {}, // 终态
	}

	allowedStates, exists := validTransitions[from]
	if !exists {
		return false
	}

	for _, allowedState := range allowedStates {
		if allowedState == to {
			return true
		}
	}
	return false
}

// CanTransitionWorker 检查Worker状态转换是否合法
func CanTransitionWorker(from, to WorkerState) bool {
	validTransitions := map[WorkerState][]WorkerState{
		WorkerStateIdle: {
			WorkerStateRunning,
			WorkerStateTerminating,
		},
		WorkerStateInitializing: {
			WorkerStateIdle,
			WorkerStateError,
			WorkerStateTerminating,
		},
		WorkerStateRunning: {
			WorkerStateBusy,
			WorkerStateCompleted,
			WorkerStateError,
			WorkerStateTerminating,
		},
		WorkerStateBusy: {
			WorkerStateRunning,
			WorkerStateWaiting,
			WorkerStateCompleted,
			WorkerStateError,
			WorkerStateTerminating,
		},
		WorkerStateWaiting: {
			WorkerStateRunning,
			WorkerStateError,
			WorkerStateTerminating,
		},
		WorkerStateCompleted: {
			WorkerStateIdle,
			WorkerStateTerminating,
		},
		WorkerStateError: {
			WorkerStateIdle,
			WorkerStateTerminating,
		},
		WorkerStateTerminating: {
			WorkerStateTerminated,
		},
		WorkerStateTerminated: {}, // 终态
	}

	allowedStates, exists := validTransitions[from]
	if !exists {
		return false
	}

	for _, allowedState := range allowedStates {
		if allowedState == to {
			return true
		}
	}
	return false
}

// GetGlobalStateManager 获取全局状态管理器单例
var (
	globalStateManager *StateManager
	once               sync.Once
)

func GetGlobalStateManager() *StateManager {
	once.Do(func() {
		globalStateManager = NewStateManager()
	})
	return globalStateManager
}

// Context 相关的辅助函数

type stateManagerKey struct{}

// ContextWithStateManager 将StateManager添加到context中
func ContextWithStateManager(ctx context.Context, sm *StateManager) context.Context {
	return context.WithValue(ctx, stateManagerKey{}, sm)
}

// StateManagerFromContext 从context中获取StateManager
func StateManagerFromContext(ctx context.Context) (*StateManager, bool) {
	sm, ok := ctx.Value(stateManagerKey{}).(*StateManager)
	return sm, ok
}
