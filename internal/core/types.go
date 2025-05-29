package core

import (
	"time"

	. "github.com/m4n5ter/another-me/pkg/option"
)

// AgentType 定义Agent的类型
type AgentType string

const (
	AgentTypeGUI     AgentType = "gui"
	AgentTypeReAct   AgentType = "react"
	AgentTypeUnknown AgentType = "unknown"
)

// Task 描述一个需要Agent执行的具体任务
type Task struct {
	ID          string         `json:"id"`          // 任务唯一标识
	Type        string         `json:"type"`        // 任务类型 (例如 "gui_click", "react_plan_and_execute")
	Description string         `json:"description"` // 任务的自然语言描述
	AgentType   AgentType      `json:"agent_type"`  // 指定执行此任务的Agent类型
	Parameters  map[string]any `json:"parameters"`  // 任务执行所需的具体参数
	Priority    int            `json:"priority"`    // 任务优先级
	CreatedAt   time.Time      `json:"created_at"`  // 任务创建时间
	Context     map[string]any `json:"context"`     // 任务执行上下文
}

// ExecutionResult Agent执行任务后的结果
type ExecutionResult struct {
	TaskID       string          `json:"task_id"`      // 对应的任务ID
	Status       ExecutionStatus `json:"status"`       // 执行状态
	Output       any             `json:"output"`       // Agent执行的主要产出物
	Observations []string        `json:"observations"` // Agent在执行过程中的重要观察
	Error        string          `json:"error"`        // 如果执行失败，记录错误信息
	StartTime    time.Time       `json:"start_time"`   // 执行开始时间
	EndTime      time.Time       `json:"end_time"`     // 执行结束时间
	Metadata     map[string]any  `json:"metadata"`     // 其他性能指标或元数据
}

// ExecutionStatus 定义执行状态
type ExecutionStatus string

const (
	ExecutionStatusSuccess    ExecutionStatus = "success"
	ExecutionStatusFailure    ExecutionStatus = "failure"
	ExecutionStatusInProgress ExecutionStatus = "in_progress"
	ExecutionStatusCancelled  ExecutionStatus = "cancelled"
)

// MonitoringTask 定义一个需要委托给Mindscape的监控任务
type MonitoringTask struct {
	ID                  Option[string]     `json:"id"`                   // 监控任务的唯一ID (由Mindscape生成并返回)
	Description         string             `json:"description"`          // 监控任务的自然语言描述
	MindscapeTaskType   string             `json:"mindscape_task_type"`  // 期望在Mindscape中使用的任务类型
	Conditions          []MonitorCondition `json:"conditions"`           // 触发唤醒的一组条件
	TargetData          []string           `json:"target_data"`          // 需要采集并返回的数据点
	NotificationMethods []string           `json:"notification_methods"` // 通知方式
	WebhookURL          Option[string]     `json:"webhook_url"`          // Webhook回调URL
	MQTopic             Option[string]     `json:"mq_topic"`             // 消息队列主题
	MaxRetries          Option[int]        `json:"max_retries"`          // 通知传递的最大重试次数
	IsEnabled           bool               `json:"is_enabled"`           // 任务是否启用
	CreatedAt           time.Time          `json:"created_at"`           // 创建时间
	UpdatedAt           time.Time          `json:"updated_at"`           // 更新时间
}

// MonitorCondition 定义监控条件
type MonitorCondition struct {
	Type     string `json:"type"`     // 条件类型 e.g., "application_start", "text_on_screen", "user_idle_then_active"
	Property string `json:"property"` // 属性名 e.g., "application_name", "text_pattern", "idle_duration_seconds"
	Operator string `json:"operator"` // 操作符 e.g., "equals", "contains", "greater_than"
	Value    any    `json:"value"`    // 比较值
}

// TaskUpdate 用于更新或清除Mindscape中的监控任务
type TaskUpdate struct {
	TasksToUpdate   []MonitoringTask `json:"tasks_to_update"`    // 需要更新的监控任务详情
	TaskIDsToDelete []string         `json:"task_ids_to_delete"` // 需要删除的监控任务ID
}

// WakeupEvent Mindscape唤醒"Another Me"时传递的数据
type WakeupEvent struct {
	MonitoringTaskID string         `json:"monitoring_task_id"` // 触发唤醒的监控任务ID
	TriggerTime      time.Time      `json:"trigger_time"`       // 唤醒条件满足的时间
	ObservedData     map[string]any `json:"observed_data"`      // Mindscape观测到的数据
	Reason           string         `json:"reason"`             // 简述唤醒原因
	Metadata         map[string]any `json:"metadata"`           // 其他元数据
}

// MemoryItem 表示存储在Mindscape中的一条记忆
type MemoryItem struct {
	ID         string         `json:"id"`          // 记忆唯一标识
	Timestamp  time.Time      `json:"timestamp"`   // 记忆产生的时间
	Type       string         `json:"type"`        // 记忆类型
	Content    any            `json:"content"`     // 记忆的具体内容
	Keywords   []string       `json:"keywords"`    // 用于检索的关键词
	Importance float64        `json:"importance"`  // 记忆的重要性评分 (0.0 - 1.0)
	RelatedIDs []string       `json:"related_ids"` // 与此记忆相关的其他记忆ID
	UserID     string         `json:"user_id"`     // 关联的用户ID
	Metadata   map[string]any `json:"metadata"`    // 其他元数据
}

// MemoryType 定义记忆类型常量
const (
	MemoryTypeObservation = "observation"
	MemoryTypeUserPref    = "user_preference"
	MemoryTypeTaskSummary = "task_summary"
	MemoryTypeErrorLog    = "error_log"
	MemoryTypeUserProfile = "user_profile"
	MemoryTypeSystemEvent = "system_event"
)

// DecisionContext 决策上下文信息
type DecisionContext struct {
	WakeupEvent         Option[WakeupEvent]     `json:"wakeup_event"`          // 唤醒事件数据
	RetrievedMemories   []MemoryItem            `json:"retrieved_memories"`    // 检索到的相关记忆
	SystemState         map[string]any          `json:"system_state"`          // 系统内部状态
	LastExecutionResult Option[ExecutionResult] `json:"last_execution_result"` // 上次执行结果
	Timestamp           time.Time               `json:"timestamp"`             // 决策时间戳
}

// DecisionResult 决策结果
type DecisionResult struct {
	ShouldExecuteTask   bool                  `json:"should_execute_task"`    // 是否应该执行任务
	Task                Option[Task]          `json:"task"`                   // 要执行的任务（如果有）
	MonitoringTasks     []MonitoringTask      `json:"monitoring_tasks"`       // 需要注册的监控任务
	ShouldEnterWaitMode bool                  `json:"should_enter_wait_mode"` // 是否进入等待模式
	ReasoningSteps      []string              `json:"reasoning_steps"`        // 决策推理步骤
	Confidence          float64               `json:"confidence"`             // 决策置信度
	ExpectedDuration    Option[time.Duration] `json:"expected_duration"`      // 预期执行时长
}

// SystemState 系统状态
type SystemState struct {
	IsActive            bool              `json:"is_active"`             // 系统是否处于活跃状态
	IsWaitingMode       bool              `json:"is_waiting_mode"`       // 是否处于等待模式
	CurrentTask         Option[Task]      `json:"current_task"`          // 当前正在执行的任务
	ActiveMonitoringIDs []string          `json:"active_monitoring_ids"` // 当前活跃的监控任务ID
	LastActivity        time.Time         `json:"last_activity"`         // 最后活动时间
	StartTime           time.Time         `json:"start_time"`            // 系统启动时间
	ExecutionHistory    []ExecutionResult `json:"execution_history"`     // 最近的执行历史
	ErrorCount          int               `json:"error_count"`           // 错误计数
	Metadata            map[string]any    `json:"metadata"`              // 其他系统元数据
}

// ExecutionMode 定义任务执行模式
type ExecutionMode string

const (
	ExecutionModeSerial   ExecutionMode = "serial"   // 串行执行
	ExecutionModeParallel ExecutionMode = "parallel" // 并行执行
	ExecutionModeMixed    ExecutionMode = "mixed"    // 混合执行（包含串行和并行步骤）
)

// ExecutionStep 执行步骤，支持串行和并行的混合执行
type ExecutionStep struct {
	ID           string             `json:"id"`            // 步骤唯一标识
	Mode         ExecutionMode      `json:"mode"`          // 执行模式
	Tasks        []Task             `json:"tasks"`         // 此步骤包含的任务列表
	Dependencies []string           `json:"dependencies"`  // 依赖的步骤ID
	MaxRetries   int                `json:"max_retries"`   // 最大重试次数
	Timeout      Option[time.Duration] `json:"timeout"`   // 步骤超时时间
	ContinueOnFailure bool          `json:"continue_on_failure"` // 失败时是否继续
}

// ExecutionPlan 执行计划，定义完整的任务执行流程
type ExecutionPlan struct {
	ID                   string                `json:"id"`                    // 计划唯一标识
	Steps                []ExecutionStep       `json:"steps"`                 // 执行步骤列表
	GlobalTimeout        Option[time.Duration] `json:"global_timeout"`        // 全局超时时间
	ContinuationStrategy ContinuationStrategy  `json:"continuation_strategy"` // 持续执行策略
	CreatedAt            time.Time             `json:"created_at"`            // 创建时间
	Context              map[string]any        `json:"context"`               // 执行上下文
}

// ContinuationStrategy 持续执行策略
type ContinuationStrategy struct {
	MaxIterations        int                   `json:"max_iterations"`         // 最大迭代次数
	ContinueConditions   []ContinueCondition   `json:"continue_conditions"`    // 继续执行的条件
	StopConditions       []StopCondition       `json:"stop_conditions"`        // 停止执行的条件
	IdleThreshold        time.Duration         `json:"idle_threshold"`         // 空闲阈值
	FeedbackAnalysisType FeedbackAnalysisType  `json:"feedback_analysis_type"` // 反馈分析类型
}

// ContinueCondition 继续执行的条件
type ContinueCondition struct {
	Type        string `json:"type"`        // 条件类型 (e.g., "output_contains", "execution_successful", "custom_logic")
	Property    string `json:"property"`    // 检查的属性
	Operator    string `json:"operator"`    // 操作符
	Value       any    `json:"value"`       // 期望值
	Description string `json:"description"` // 条件描述
}

// StopCondition 停止执行的条件
type StopCondition struct {
	Type        string `json:"type"`        // 条件类型
	Property    string `json:"property"`    // 检查的属性
	Operator    string `json:"operator"`    // 操作符
	Value       any    `json:"value"`       // 期望值
	Description string `json:"description"` // 条件描述
}

// FeedbackAnalysisType 反馈分析类型
type FeedbackAnalysisType string

const (
	FeedbackAnalysisSimple    FeedbackAnalysisType = "simple"    // 简单规则匹配
	FeedbackAnalysisLLM       FeedbackAnalysisType = "llm"       // LLM分析
	FeedbackAnalysisHybrid    FeedbackAnalysisType = "hybrid"    // 混合分析
)

// ExecutionState 执行状态跟踪
type ExecutionState struct {
	PlanID           string              `json:"plan_id"`            // 执行计划ID
	CurrentStepIndex int                 `json:"current_step_index"` // 当前执行步骤索引
	StepResults      []StepResult        `json:"step_results"`       // 各步骤执行结果
	StartTime        time.Time           `json:"start_time"`         // 开始执行时间
	Status           ExecutionStatus     `json:"status"`             // 总体执行状态
	IterationCount   int                 `json:"iteration_count"`    // 迭代次数
	TotalTaskCount   int                 `json:"total_task_count"`   // 总任务数
	CompletedTaskCount int               `json:"completed_task_count"` // 已完成任务数
	FailedTaskCount  int                 `json:"failed_task_count"`  // 失败任务数
	Metadata         map[string]any      `json:"metadata"`           // 执行元数据
}

// StepResult 步骤执行结果
type StepResult struct {
	StepID        string            `json:"step_id"`        // 步骤ID
	TaskResults   []ExecutionResult `json:"task_results"`   // 任务执行结果
	Status        ExecutionStatus   `json:"status"`         // 步骤状态
	StartTime     time.Time         `json:"start_time"`     // 步骤开始时间
	EndTime       time.Time         `json:"end_time"`       // 步骤结束时间
	RetryCount    int               `json:"retry_count"`    // 重试次数
	ErrorMessages []string          `json:"error_messages"` // 错误信息
}

// ContinuousDecisionContext 持续决策上下文
type ContinuousDecisionContext struct {
	InitialContext      DecisionContext       `json:"initial_context"`       // 初始决策上下文
	ExecutionState      ExecutionState        `json:"execution_state"`       // 当前执行状态
	StepResults         []StepResult          `json:"step_results"`          // 步骤执行结果
	AgentOutputAnalysis AgentOutputAnalysis   `json:"agent_output_analysis"` // Agent输出分析
	SystemMetrics       SystemMetrics         `json:"system_metrics"`        // 系统指标
	Timestamp           time.Time             `json:"timestamp"`             // 决策时间
}

// AgentOutputAnalysis Agent输出分析结果
type AgentOutputAnalysis struct {
	KeyFindings         []string        `json:"key_findings"`          // 关键发现
	ActionableInsights  []string        `json:"actionable_insights"`   // 可执行洞察
	RequiresUserInput   bool            `json:"requires_user_input"`   // 是否需要用户输入
	ConfidenceLevel     float64         `json:"confidence_level"`      // 置信度
	RecommendedActions  []string        `json:"recommended_actions"`   // 推荐行动
	RiskAssessment      RiskAssessment  `json:"risk_assessment"`       // 风险评估
	NextStepSuggestions []string        `json:"next_step_suggestions"` // 下一步建议
}

// RiskAssessment 风险评估
type RiskAssessment struct {
	Level       string   `json:"level"`       // 风险级别 (low, medium, high)
	Factors     []string `json:"factors"`     // 风险因素
	Mitigation  []string `json:"mitigation"`  // 缓解措施
	Description string   `json:"description"` // 风险描述
}

// SystemMetrics 系统指标
type SystemMetrics struct {
	CPUUsage         float64       `json:"cpu_usage"`         // CPU使用率
	MemoryUsage      float64       `json:"memory_usage"`      // 内存使用率
	ActiveAgentCount int           `json:"active_agent_count"` // 活跃Agent数量
	AverageResponseTime time.Duration `json:"average_response_time"` // 平均响应时间
	ErrorRate        float64       `json:"error_rate"`        // 错误率
	ThroughputPerHour int          `json:"throughput_per_hour"` // 每小时处理量
}

// ContinuousDecisionResult 持续决策结果
type ContinuousDecisionResult struct {
	ShouldContinue       bool                  `json:"should_continue"`        // 是否应该继续执行
	NextExecutionPlan    Option[ExecutionPlan] `json:"next_execution_plan"`    // 下一个执行计划
	ShouldEnterWaitMode  bool                  `json:"should_enter_wait_mode"` // 是否进入等待模式
	MonitoringTasks      []MonitoringTask      `json:"monitoring_tasks"`       // 新的监控任务
	ReasoningSteps       []string              `json:"reasoning_steps"`        // 决策推理步骤
	Confidence           float64               `json:"confidence"`             // 决策置信度
	EstimatedDuration    Option[time.Duration] `json:"estimated_duration"`     // 预计执行时长
	Priority             int                   `json:"priority"`               // 优先级
	RequiresUserApproval bool                  `json:"requires_user_approval"` // 是否需要用户批准
}
