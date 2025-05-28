package core

import (
	"context"
	"time"
)

// Agent 定义Agent的通用接口
// 所有Agent（GUI Agent、ReAct Agent等）都需要实现此接口
type Agent interface {
	// Execute 执行具体任务
	// ctx: 上下文，用于取消和超时控制
	// task: 要执行的任务
	// initialContext: 任务执行的初始上下文
	Execute(ctx context.Context, task Task, initialContext map[string]any) (ExecutionResult, error)

	// Name 返回Agent的名称
	Name() string

	// Type 返回Agent的类型
	Type() AgentType

	// IsAvailable 检查Agent是否可用
	IsAvailable(ctx context.Context) bool

	// GetCapabilities 获取Agent的能力描述
	GetCapabilities() []string
}

// MindscapeService 定义与Mindscape服务交互的接口
// MindscapeConnector实现此接口，供其他组件依赖注入和调用
type MindscapeService interface {
	// StoreMemory 存储记忆到Mindscape
	StoreMemory(ctx context.Context, memoryData MemoryItem) error

	// RetrieveMemories 从Mindscape检索相关记忆
	// queryContext: 查询上下文，包含查询关键词、时间范围等
	RetrieveMemories(ctx context.Context, queryContext map[string]any) ([]MemoryItem, error)

	// DelegateMonitoringTask 委托监控任务给Mindscape
	// 返回监控任务ID
	DelegateMonitoringTask(ctx context.Context, taskDetails MonitoringTask) (string, error)

	// ClearOrUpdateMonitoringTasks 清除或更新监控任务
	ClearOrUpdateMonitoringTasks(ctx context.Context, taskUpdate TaskUpdate) error

	// SetupWakeUpListener 设置唤醒监听器
	// handler: 处理唤醒事件的回调函数
	SetupWakeUpListener(handler func(wakeupData WakeupEvent) error) error

	// IsHealthy 检查Mindscape连接是否健康
	IsHealthy(ctx context.Context) bool

	// GetUserProfile 获取用户画像
	GetUserProfile(ctx context.Context, userID string) (*MemoryItem, error)

	// UpdateUserProfile 更新用户画像
	UpdateUserProfile(ctx context.Context, userID string, profileData map[string]any) error
}

// DecisionMaker 定义决策器接口
// 负责根据当前上下文决定下一步行动
type DecisionMaker interface {
	// MakeDecision 进行决策
	// 整合当前上下文信息，决定是否执行任务或进入监控模式
	MakeDecision(ctx context.Context, decisionContext DecisionContext) (DecisionResult, error)

	// EvaluateTaskPriority 评估任务优先级
	EvaluateTaskPriority(ctx context.Context, tasks []Task) ([]Task, error)

	// DefineMonitoringConditions 定义监控条件
	// 当无明确任务时，定义需要Mindscape监控的条件
	DefineMonitoringConditions(ctx context.Context, systemState SystemState) ([]MonitoringTask, error)

	// AnalyzeWakeupEvent 分析唤醒事件
	// 根据唤醒事件决定后续行动
	AnalyzeWakeupEvent(ctx context.Context, wakeupEvent WakeupEvent) (DecisionResult, error)

	// GetDecisionHistory 获取决策历史
	GetDecisionHistory(ctx context.Context, limit int) ([]DecisionResult, error)
}

// AgentDispatcher 定义Agent调度器接口
// 负责管理Agent实例和任务分发
type AgentDispatcher interface {
	// RegisterAgent 注册Agent
	RegisterAgent(agent Agent) error

	// UnregisterAgent 注销Agent
	UnregisterAgent(agentType AgentType) error

	// DispatchTask 分发任务给合适的Agent
	DispatchTask(ctx context.Context, task Task) (ExecutionResult, error)

	// GetAvailableAgents 获取可用的Agent列表
	GetAvailableAgents(ctx context.Context) ([]Agent, error)

	// GetAgentByType 根据类型获取Agent
	GetAgentByType(agentType AgentType) (Agent, error)

	// IsAgentAvailable 检查指定类型的Agent是否可用
	IsAgentAvailable(ctx context.Context, agentType AgentType) bool

	// GetAgentStatus 获取Agent状态
	GetAgentStatus(ctx context.Context, agentType AgentType) (map[string]any, error)
}

// MainLoop 定义主循环接口
// 系统的心跳，驱动持续运行和状态管理
type MainLoop interface {
	// Start 启动主循环
	Start(ctx context.Context) error

	// Stop 停止主循环
	Stop(ctx context.Context) error

	// IsRunning 检查主循环是否正在运行
	IsRunning() bool

	// GetSystemState 获取当前系统状态
	GetSystemState() SystemState

	// OnWakeupEvent 处理唤醒事件的回调
	OnWakeupEvent(wakeupEvent WakeupEvent) error

	// EnterWaitMode 进入等待模式
	EnterWaitMode(ctx context.Context, monitoringTasks []MonitoringTask) error

	// ExitWaitMode 退出等待模式
	ExitWaitMode(ctx context.Context) error

	// GetExecutionHistory 获取执行历史
	GetExecutionHistory(limit int) []ExecutionResult
}

// WakeupListener 定义唤醒监听器接口
// 用于接收Mindscape的唤醒信号
type WakeupListener interface {
	// Start 启动监听器
	Start(ctx context.Context) error

	// Stop 停止监听器
	Stop(ctx context.Context) error

	// SetHandler 设置唤醒事件处理器
	SetHandler(handler func(WakeupEvent) error)

	// IsListening 检查是否正在监听
	IsListening() bool

	// GetListenAddress 获取监听地址（如Webhook URL）
	GetListenAddress() string
}

// TaskEvaluator 定义任务评估器接口
// 用于评估任务的完成情况和结果质量
type TaskEvaluator interface {
	// EvaluateTaskCompletion 评估任务是否完成
	EvaluateTaskCompletion(ctx context.Context, task Task, result ExecutionResult) (bool, float64, error)

	// EvaluateResultQuality 评估执行结果的质量
	EvaluateResultQuality(ctx context.Context, task Task, result ExecutionResult) (float64, string, error)

	// SuggestImprovements 建议改进措施
	SuggestImprovements(ctx context.Context, task Task, result ExecutionResult) ([]string, error)
}

// MemoryManager 定义记忆管理器接口
// 提供本地记忆缓存和管理功能
type MemoryManager interface {
	// CacheMemory 缓存记忆到本地
	CacheMemory(ctx context.Context, memory MemoryItem) error

	// GetCachedMemories 获取缓存的记忆
	GetCachedMemories(ctx context.Context, query map[string]any) ([]MemoryItem, error)

	// ClearCache 清除缓存
	ClearCache(ctx context.Context) error

	// GetCacheStats 获取缓存统计信息
	GetCacheStats() map[string]any

	// SyncWithMindscape 与Mindscape同步记忆
	SyncWithMindscape(ctx context.Context) error
}

// ConfigManager 定义配置管理器接口
type ConfigManager interface {
	// GetConfig 获取配置
	GetConfig(key string) (any, error)

	// SetConfig 设置配置
	SetConfig(key string, value any) error

	// ReloadConfig 重新加载配置
	ReloadConfig() error

	// GetAllConfigs 获取所有配置
	GetAllConfigs() map[string]any
}

// HealthChecker 定义健康检查接口
type HealthChecker interface {
	// CheckHealth 检查组件健康状态
	CheckHealth(ctx context.Context) (HealthStatus, error)

	// GetHealthStatus 获取健康状态
	GetHealthStatus() HealthStatus

	// RegisterHealthCheck 注册健康检查
	RegisterHealthCheck(name string, checker func(ctx context.Context) error)
}

// HealthStatus 健康状态
type HealthStatus struct {
	IsHealthy    bool           `json:"is_healthy"`
	LastCheck    time.Time      `json:"last_check"`
	Details      map[string]any `json:"details"`
	ErrorMessage string         `json:"error_message,omitempty"`
}

// Logger 定义日志接口
type Logger interface {
	// Debug 调试日志
	Debug(msg string, args ...any)

	// Info 信息日志
	Info(msg string, args ...any)

	// Warn 警告日志
	Warn(msg string, args ...any)

	// Error 错误日志
	Error(msg string, args ...any)

	// WithContext 添加上下文
	WithContext(ctx context.Context) Logger

	// WithFields 添加字段
	WithFields(fields map[string]any) Logger
}
