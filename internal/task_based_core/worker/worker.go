package worker

import (
	"context"
	"time"

	. "github.com/m4n5ter/another-me/pkg/option"
)

// Worker 接口定义
type Worker interface {
	// GetID 获取Worker ID
	GetID() string
	// GetType 获取Worker类型
	GetType() WorkerType
	// GetCapabilities 获取Worker能力列表
	GetCapabilities() []string
	// Execute 执行任务
	Execute(ctx context.Context, task *Task) (*TaskResult, error)
	// IsReady 检查Worker是否就绪
	IsReady() bool
	// Shutdown 关闭Worker
	Shutdown(ctx context.Context) error
}

// WorkerType Worker类型
type WorkerType string

const (
	WorkerTypeWebUI        WorkerType = "web_ui"
	WorkerTypeDataAnalysis WorkerType = "data_analysis"
	WorkerTypeFileSystem   WorkerType = "file_system"
	WorkerTypeTemporary    WorkerType = "temporary"
)

// String 返回Worker类型的字符串表示
func (wt WorkerType) String() string {
	return string(wt)
}

// Task 任务结构
type Task struct {
	ID         string                `json:"id"`         // 任务ID
	Name       string                `json:"name"`       // 任务名称
	Type       string                `json:"type"`       // 任务类型
	Parameters map[string]any        `json:"parameters"` // 任务参数
	Timeout    Option[time.Duration] `json:"timeout"`    // 超时时间
	Priority   int                   `json:"priority"`   // 优先级
	Metadata   map[string]any        `json:"metadata"`   // 元数据
}

// TaskResult 任务执行结果
type TaskResult struct {
	TaskID   string         `json:"task_id"`  // 任务ID
	Success  bool           `json:"success"`  // 执行是否成功
	Result   map[string]any `json:"result"`   // 执行结果
	Error    Option[string] `json:"error"`    // 错误信息
	Duration time.Duration  `json:"duration"` // 执行时长
	Metadata map[string]any `json:"metadata"` // 结果元数据
}

// BaseWorker 基础Worker实现
type BaseWorker struct {
	id           string
	workerType   WorkerType
	capabilities []string
	ready        bool
}

// NewBaseWorker 创建基础Worker
func NewBaseWorker(id string, workerType WorkerType, capabilities ...string) *BaseWorker {
	return &BaseWorker{
		id:           id,
		workerType:   workerType,
		capabilities: capabilities,
		ready:        true,
	}
}

// GetID 实现Worker接口
func (bw *BaseWorker) GetID() string {
	return bw.id
}

// GetType 实现Worker接口
func (bw *BaseWorker) GetType() WorkerType {
	return bw.workerType
}

// GetCapabilities 实现Worker接口
func (bw *BaseWorker) GetCapabilities() []string {
	return bw.capabilities
}

// IsReady 实现Worker接口
func (bw *BaseWorker) IsReady() bool {
	return bw.ready
}

// SetReady 设置Worker就绪状态
func (bw *BaseWorker) SetReady(ready bool) {
	bw.ready = ready
}
