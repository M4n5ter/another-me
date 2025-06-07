package communication

import (
	"time"

	. "github.com/m4n5ter/another-me/pkg/option"
)

// Event 事件接口
type Event interface {
	// EventType 返回事件类型
	EventType() EventType
	// EventID 返回事件唯一ID
	EventID() string
	// Source 返回事件源组件ID
	Source() string
	// Timestamp 返回事件时间戳
	Timestamp() time.Time
	// Metadata 返回事件元数据
	Metadata() map[string]any
}

// EventType 事件类型枚举
type EventType string

const (
	// 任务相关事件

	EventTypeTaskCreated   EventType = "task_created"   // 任务创建
	EventTypeTaskStarted   EventType = "task_started"   // 任务开始
	EventTypeTaskCompleted EventType = "task_completed" // 任务完成
	EventTypeTaskFailed    EventType = "task_failed"    // 任务失败
	EventTypeTaskCancelled EventType = "task_cancelled" // 任务取消
	EventTypeTaskRetry     EventType = "task_retry"     // 任务重试
	EventTypeTaskProgress  EventType = "task_progress"  // 任务进度更新

	// Worker相关事件

	EventTypeWorkerRegistered   EventType = "worker_registered"   // Worker注册
	EventTypeWorkerUnregistered EventType = "worker_unregistered" // Worker注销
	EventTypeWorkerIdle         EventType = "worker_idle"         // Worker空闲
	EventTypeWorkerBusy         EventType = "worker_busy"         // Worker忙碌
	EventTypeWorkerError        EventType = "worker_error"        // Worker错误

	// 系统相关事件

	EventTypeSystemStateChanged EventType = "system_state_changed" // 系统状态变更
	EventTypeResourceAlert      EventType = "resource_alert"       // 资源告警
	EventTypePerformanceReport  EventType = "performance_report"   // 性能报告

	// 通信相关事件

	EventTypeComponentRegistered   EventType = "component_registered"   // 组件注册
	EventTypeComponentUnregistered EventType = "component_unregistered" // 组件注销
	EventTypeHeartbeat             EventType = "heartbeat"              // 心跳
)

// BaseEvent 基础事件实现
type BaseEvent struct {
	ID       string         `json:"id"`        // 事件ID
	Type     EventType      `json:"type"`      // 事件类型
	SourceID string         `json:"source_id"` // 源组件ID
	Time     time.Time      `json:"timestamp"` // 时间戳
	Meta     map[string]any `json:"metadata"`  // 元数据
}

// EventType 实现Event接口
func (e *BaseEvent) EventType() EventType {
	return e.Type
}

// EventID 实现Event接口
func (e *BaseEvent) EventID() string {
	return e.ID
}

// Source 实现Event接口
func (e *BaseEvent) Source() string {
	return e.SourceID
}

// Timestamp 实现Event接口
func (e *BaseEvent) Timestamp() time.Time {
	return e.Time
}

// Metadata 实现Event接口
func (e *BaseEvent) Metadata() map[string]any {
	return e.Meta
}

// TaskEvent 任务事件
type TaskEvent struct {
	BaseEvent
	TaskID       string         `json:"task_id"`      // 任务ID
	TaskName     string         `json:"task_name"`    // 任务名称
	WorkerID     Option[string] `json:"worker_id"`    // 执行Worker ID
	Progress     float64        `json:"progress"`     // 任务进度 (0-100)
	Result       Option[string] `json:"result"`       // 任务结果
	ErrorMsg     Option[string] `json:"error_msg"`    // 错误信息
	RetryCount   int            `json:"retry_count"`  // 重试次数
	Dependencies []string       `json:"dependencies"` // 依赖任务ID列表
}

// WorkerEvent Worker事件
type WorkerEvent struct {
	BaseEvent
	WorkerID      string         `json:"worker_id"`      // Worker ID
	WorkerType    string         `json:"worker_type"`    // Worker类型
	CurrentTask   Option[string] `json:"current_task"`   // 当前任务ID
	Capabilities  []string       `json:"capabilities"`   // 能力列表
	ResourceUsage ResourceUsage  `json:"resource_usage"` // 资源使用情况
	ErrorMsg      Option[string] `json:"error_msg"`      // 错误信息
}

// SystemEvent 系统事件
type SystemEvent struct {
	BaseEvent
	PreviousState Option[string] `json:"previous_state"` // 前一状态
	CurrentState  string         `json:"current_state"`  // 当前状态
	Reason        string         `json:"reason"`         // 状态变更原因
	SystemLoad    float64        `json:"system_load"`    // 系统负载
}

// ComponentEvent 组件事件
type ComponentEvent struct {
	BaseEvent
	ComponentID   string         `json:"component_id"`   // 组件ID
	ComponentType ComponentType  `json:"component_type"` // 组件类型
	Capabilities  []string       `json:"capabilities"`   // 组件能力
	Config        map[string]any `json:"config"`         // 组件配置
}

// ResourceUsage 资源使用情况
type ResourceUsage struct {
	CPUPercent    float64 `json:"cpu_percent"`    // CPU使用率
	MemoryPercent float64 `json:"memory_percent"` // 内存使用率
	DiskPercent   float64 `json:"disk_percent"`   // 磁盘使用率
	NetworkIO     int64   `json:"network_io"`     // 网络IO
}

// ComponentType 组件类型
type ComponentType string

const (
	ComponentTypeOrchestrator ComponentType = "orchestrator" // 编排器
	ComponentTypeWorker       ComponentType = "worker"       // 工作器
	ComponentTypeScheduler    ComponentType = "scheduler"    // 调度器
	ComponentTypeMonitor      ComponentType = "monitor"      // 监控器
	ComponentTypeStorage      ComponentType = "storage"      // 存储器
)

// EventPriority 事件优先级
type EventPriority int

const (
	PriorityLow      EventPriority = iota // 低优先级
	PriorityNormal                        // 正常优先级
	PriorityHigh                          // 高优先级
	PriorityCritical                      // 紧急优先级
)

// String 返回优先级字符串表示
func (p EventPriority) String() string {
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
		return "未知"
	}
}

// PriorityEvent 带优先级的事件
type PriorityEvent struct {
	Event    Event         `json:"event"`    // 原始事件
	Priority EventPriority `json:"priority"` // 事件优先级
}

// EventType 实现Event接口
func (p *PriorityEvent) EventType() EventType {
	return p.Event.EventType()
}

// EventID 实现Event接口
func (p *PriorityEvent) EventID() string {
	return p.Event.EventID()
}

// Source 实现Event接口
func (p *PriorityEvent) Source() string {
	return p.Event.Source()
}

// Timestamp 实现Event接口
func (p *PriorityEvent) Timestamp() time.Time {
	return p.Event.Timestamp()
}

// Metadata 实现Event接口
func (p *PriorityEvent) Metadata() map[string]any {
	return p.Event.Metadata()
}

// TaskDependencyEvent 任务依赖事件
type TaskDependencyEvent struct {
	BaseEvent
	TaskID       string   `json:"task_id"`      // 任务ID
	Dependencies []string `json:"dependencies"` // 依赖的任务ID列表
	Ready        bool     `json:"ready"`        // 是否准备就绪
	BlockedBy    []string `json:"blocked_by"`   // 被哪些任务阻塞
}

// NewTaskEvent 创建任务事件
func NewTaskEvent(eventType EventType, sourceID, taskID, taskName string) *TaskEvent {
	return &TaskEvent{
		BaseEvent: BaseEvent{
			ID:       generateEventID(),
			Type:     eventType,
			SourceID: sourceID,
			Time:     time.Now(),
			Meta:     make(map[string]any),
		},
		TaskID:   taskID,
		TaskName: taskName,
	}
}

// NewWorkerEvent 创建Worker事件
func NewWorkerEvent(eventType EventType, sourceID, workerID, workerType string) *WorkerEvent {
	return &WorkerEvent{
		BaseEvent: BaseEvent{
			ID:       generateEventID(),
			Type:     eventType,
			SourceID: sourceID,
			Time:     time.Now(),
			Meta:     make(map[string]any),
		},
		WorkerID:   workerID,
		WorkerType: workerType,
	}
}

// NewSystemEvent 创建系统事件
func NewSystemEvent(eventType EventType, sourceID, currentState, reason string) *SystemEvent {
	return &SystemEvent{
		BaseEvent: BaseEvent{
			ID:       generateEventID(),
			Type:     eventType,
			SourceID: sourceID,
			Time:     time.Now(),
			Meta:     make(map[string]any),
		},
		CurrentState: currentState,
		Reason:       reason,
	}
}

// NewComponentEvent 创建组件事件
func NewComponentEvent(eventType EventType, sourceID, componentID string, componentType ComponentType) *ComponentEvent {
	return &ComponentEvent{
		BaseEvent: BaseEvent{
			ID:       generateEventID(),
			Type:     eventType,
			SourceID: sourceID,
			Time:     time.Now(),
			Meta:     make(map[string]any),
		},
		ComponentID:   componentID,
		ComponentType: componentType,
		Config:        make(map[string]any),
	}
}

// NewTaskDependencyEvent 创建任务依赖事件
func NewTaskDependencyEvent(sourceID, taskID string, dependencies []string) *TaskDependencyEvent {
	return &TaskDependencyEvent{
		BaseEvent: BaseEvent{
			ID:       generateEventID(),
			Type:     "task_dependency_check",
			SourceID: sourceID,
			Time:     time.Now(),
			Meta:     make(map[string]any),
		},
		TaskID:       taskID,
		Dependencies: dependencies,
		BlockedBy:    make([]string, 0),
	}
}

// generateEventID 生成事件ID
func generateEventID() string {
	return "event_" + time.Now().Format("20060102_150405") + "_" + randomString(8)
}

// randomString 生成随机字符串
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}
