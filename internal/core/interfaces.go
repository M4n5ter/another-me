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

// SmartTaskOrchestrator 智能任务编排器接口
// 支持并行、串行、混合执行模式的复杂任务编排
type SmartTaskOrchestrator interface {
	// ExecutePlan 执行任务计划
	// 支持复杂的并行/串行/混合执行流程
	ExecutePlan(ctx context.Context, plan ExecutionPlan) (ExecutionState, error)

	// CreateExecutionPlan 创建执行计划
	// 根据任务列表和策略创建优化的执行计划
	CreateExecutionPlan(ctx context.Context, tasks []Task, strategy ExecutionMode) (ExecutionPlan, error)

	// MonitorExecution 监控执行状态
	// 返回当前执行状态和进度信息
	MonitorExecution(ctx context.Context, planID string) (ExecutionState, error)

	// CancelExecution 取消执行
	CancelExecution(ctx context.Context, planID string) error

	// GetExecutionHistory 获取执行历史
	GetExecutionHistory(ctx context.Context, limit int) ([]ExecutionState, error)

	// OptimizeExecutionPlan 优化执行计划
	// 基于历史性能数据优化任务调度
	OptimizeExecutionPlan(ctx context.Context, plan ExecutionPlan) (ExecutionPlan, error)

	// EstimateExecutionTime 估算执行时间
	EstimateExecutionTime(ctx context.Context, plan ExecutionPlan) (time.Duration, error)

	// GetResourceUsage 获取资源使用情况
	GetResourceUsage(ctx context.Context) (SystemMetrics, error)
}

// ContinuousDecisionEngine 持续决策引擎接口
// 基于Agent输出反馈进行持续决策，支持多轮智能决策
type ContinuousDecisionEngine interface {
	// MakeContinuousDecision 进行持续决策
	// 基于当前执行状态和Agent输出分析决定下一步行动
	MakeContinuousDecision(ctx context.Context, decisionContext ContinuousDecisionContext) (ContinuousDecisionResult, error)

	// AnalyzeAgentOutput 分析Agent输出
	// 对Agent的执行结果进行深度分析，提取关键信息
	AnalyzeAgentOutput(ctx context.Context, results []ExecutionResult) (AgentOutputAnalysis, error)

	// EvaluateContinuationStrategy 评估持续策略
	// 判断是否应该继续执行、休眠或等待用户输入
	EvaluateContinuationStrategy(ctx context.Context, executionState ExecutionState, outputAnalysis AgentOutputAnalysis) (ContinuousDecisionResult, error)

	// GenerateNextActions 生成下一步行动
	// 基于分析结果生成具体的下一步行动计划
	GenerateNextActions(ctx context.Context, analysis AgentOutputAnalysis, systemContext map[string]any) ([]Task, error)

	// UpdateDecisionHistory 更新决策历史
	UpdateDecisionHistory(ctx context.Context, decisionResult ContinuousDecisionResult) error

	// GetDecisionInsights 获取决策洞察
	// 提供决策质量分析和改进建议
	GetDecisionInsights(ctx context.Context, timeRange time.Duration) (map[string]any, error)

	// ConfigureStrategy 配置决策策略
	// 动态配置决策引擎的行为策略
	ConfigureStrategy(ctx context.Context, strategy ContinuationStrategy) error
}

// FeedbackAnalyzer 反馈分析器接口
// 专门用于分析Agent执行结果的深度反馈
type FeedbackAnalyzer interface {
	// AnalyzeExecutionResults 分析执行结果
	// 深度分析多个Agent的执行结果，提取模式和洞察
	AnalyzeExecutionResults(ctx context.Context, results []ExecutionResult) (AgentOutputAnalysis, error)

	// DetectPatterns 检测执行模式
	// 识别任务执行中的成功模式和失败模式
	DetectPatterns(ctx context.Context, history []ExecutionResult) ([]string, error)

	// PredictNextSteps 预测下一步操作
	// 基于历史数据和当前结果预测最优的下一步操作
	PredictNextSteps(ctx context.Context, currentResults []ExecutionResult, systemState SystemState) ([]string, error)

	// AssessRisk 评估风险
	// 分析当前操作可能的风险和影响
	AssessRisk(ctx context.Context, proposedActions []Task, systemState SystemState) (RiskAssessment, error)

	// GenerateInsights 生成洞察
	// 从执行结果中生成可执行的洞察和建议
	GenerateInsights(ctx context.Context, analysis AgentOutputAnalysis) ([]string, error)
}

// TaskFlowManager 任务流管理器接口
// 管理复杂的任务依赖关系和执行流程
type TaskFlowManager interface {
	// CreateTaskFlow 创建任务流
	// 基于任务依赖关系创建优化的执行流程
	CreateTaskFlow(ctx context.Context, tasks []Task, dependencies map[string][]string) (ExecutionPlan, error)

	// ValidateTaskFlow 验证任务流
	// 检查任务流的合理性和可执行性
	ValidateTaskFlow(ctx context.Context, plan ExecutionPlan) (bool, []string, error)

	// OptimizeTaskFlow 优化任务流
	// 基于历史性能和资源情况优化任务执行顺序
	OptimizeTaskFlow(ctx context.Context, plan ExecutionPlan, systemMetrics SystemMetrics) (ExecutionPlan, error)

	// ResolveTaskDependencies 解析任务依赖
	// 解析和优化任务间的依赖关系
	ResolveTaskDependencies(ctx context.Context, tasks []Task) (map[string][]string, error)

	// TrackTaskFlow 跟踪任务流执行
	// 实时跟踪任务流的执行状态和进度
	TrackTaskFlow(ctx context.Context, planID string) (ExecutionState, error)
}
