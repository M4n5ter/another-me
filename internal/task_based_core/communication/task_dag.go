package communication

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	. "github.com/m4n5ter/another-me/pkg/option"
)

// TaskDAG 任务依赖图管理器
type TaskDAG struct {
	mu     sync.RWMutex
	logger *slog.Logger

	// 图结构
	nodes        map[string]*TaskNode // 任务节点
	edges        map[string][]string  // 依赖关系: taskID -> [dependentTaskIDs]
	reverseEdges map[string][]string  // 反向依赖: taskID -> [dependencyTaskIDs]

	// 状态管理
	completedTasks map[string]bool // 已完成的任务
	failedTasks    map[string]bool // 失败的任务
	runningTasks   map[string]bool // 正在运行的任务

	// 事件通信
	eventBus       *MessageBus // 消息总线
	readyTasksChan chan string // 就绪任务通道

	// 配置
	maxConcurrent int // 最大并发任务数
}

// TaskNode 任务节点
type TaskNode struct {
	ID            string         `json:"id"`             // 任务ID
	Name          string         `json:"name"`           // 任务名称
	Type          string         `json:"type"`           // 任务类型
	Priority      int            `json:"priority"`       // 优先级
	EstimatedTime time.Duration  `json:"estimated_time"` // 预估执行时间
	Metadata      map[string]any `json:"metadata"`       // 任务元数据

	// 依赖关系
	Dependencies []string `json:"dependencies"` // 依赖的任务ID
	Dependents   []string `json:"dependents"`   // 依赖此任务的任务ID

	// 状态信息
	Status      TaskStatus        `json:"status"`       // 任务状态
	CreatedAt   time.Time         `json:"created_at"`   // 创建时间
	StartedAt   Option[time.Time] `json:"started_at"`   // 开始时间
	CompletedAt Option[time.Time] `json:"completed_at"` // 完成时间
	WorkerID    Option[string]    `json:"worker_id"`    // 分配的Worker ID
	RetryCount  int               `json:"retry_count"`  // 重试次数
	MaxRetries  int               `json:"max_retries"`  // 最大重试次数

	// 结果信息
	Output       map[string]any `json:"output"`        // 任务输出
	ErrorMessage Option[string] `json:"error_message"` // 错误信息
}

// TaskStatus 任务状态
type TaskStatus int

const (
	TaskStatusPending   TaskStatus = iota // 等待依赖
	TaskStatusReady                       // 准备执行
	TaskStatusRunning                     // 正在执行
	TaskStatusCompleted                   // 已完成
	TaskStatusFailed                      // 执行失败
	TaskStatusCancelled                   // 已取消
	TaskStatusSkipped                     // 已跳过
)

// String 返回任务状态字符串表示
func (s TaskStatus) String() string {
	switch s {
	case TaskStatusPending:
		return "等待依赖"
	case TaskStatusReady:
		return "准备执行"
	case TaskStatusRunning:
		return "正在执行"
	case TaskStatusCompleted:
		return "已完成"
	case TaskStatusFailed:
		return "执行失败"
	case TaskStatusCancelled:
		return "已取消"
	case TaskStatusSkipped:
		return "已跳过"
	default:
		return "未知状态"
	}
}

// DAGStats DAG统计信息
type DAGStats struct {
	TotalTasks     int           `json:"total_tasks"`     // 总任务数
	PendingTasks   int           `json:"pending_tasks"`   // 等待任务数
	ReadyTasks     int           `json:"ready_tasks"`     // 就绪任务数
	RunningTasks   int           `json:"running_tasks"`   // 运行中任务数
	CompletedTasks int           `json:"completed_tasks"` // 已完成任务数
	FailedTasks    int           `json:"failed_tasks"`    // 失败任务数
	CancelledTasks int           `json:"cancelled_tasks"` // 取消任务数
	SkippedTasks   int           `json:"skipped_tasks"`   // 跳过任务数
	Progress       float64       `json:"progress"`        // 整体进度
	EstimatedTime  time.Duration `json:"estimated_time"`  // 预估剩余时间
}

// NewTaskDAG 创建新的任务DAG
func NewTaskDAG(eventBus *MessageBus, maxConcurrent int) *TaskDAG {
	dag := &TaskDAG{
		logger:         slog.Default().WithGroup("task_dag"),
		nodes:          make(map[string]*TaskNode),
		edges:          make(map[string][]string),
		reverseEdges:   make(map[string][]string),
		completedTasks: make(map[string]bool),
		failedTasks:    make(map[string]bool),
		runningTasks:   make(map[string]bool),
		eventBus:       eventBus,
		readyTasksChan: make(chan string, 100),
		maxConcurrent:  maxConcurrent,
	}

	// 订阅任务相关事件
	dag.subscribeToEvents()

	return dag
}

// AddTask 添加任务到DAG
func (dag *TaskDAG) AddTask(node *TaskNode) error {
	dag.mu.Lock()
	defer dag.mu.Unlock()

	if _, exists := dag.nodes[node.ID]; exists {
		return fmt.Errorf("任务 %s 已存在", node.ID)
	}

	// 初始化节点
	if node.CreatedAt.IsZero() {
		node.CreatedAt = time.Now()
	}
	if node.Metadata == nil {
		node.Metadata = make(map[string]any)
	}
	if node.Output == nil {
		node.Output = make(map[string]any)
	}
	if node.MaxRetries == 0 {
		node.MaxRetries = 3 // 默认最大重试3次
	}

	// 设置初始状态
	if len(node.Dependencies) == 0 {
		node.Status = TaskStatusReady
	} else {
		node.Status = TaskStatusPending
	}

	dag.nodes[node.ID] = node
	dag.edges[node.ID] = make([]string, 0)
	dag.reverseEdges[node.ID] = make([]string, 0)

	dag.logger.Info("任务添加到DAG",
		"task_id", node.ID,
		"task_name", node.Name,
		"dependencies", len(node.Dependencies))

	return nil
}

// AddDependency 添加任务依赖关系
func (dag *TaskDAG) AddDependency(dependentID, dependencyID string) error {
	dag.mu.Lock()
	defer dag.mu.Unlock()

	// 检查任务是否存在
	if _, exists := dag.nodes[dependentID]; !exists {
		return fmt.Errorf("依赖任务 %s 不存在", dependentID)
	}
	if _, exists := dag.nodes[dependencyID]; !exists {
		return fmt.Errorf("被依赖任务 %s 不存在", dependencyID)
	}

	// 检查是否会产生循环依赖
	if dag.wouldCreateCycle(dependentID, dependencyID) {
		return fmt.Errorf("添加依赖关系会产生循环依赖: %s -> %s", dependentID, dependencyID)
	}

	// 添加依赖关系
	dag.edges[dependencyID] = append(dag.edges[dependencyID], dependentID)
	dag.reverseEdges[dependentID] = append(dag.reverseEdges[dependentID], dependencyID)

	// 更新节点信息
	dag.nodes[dependentID].Dependencies = append(dag.nodes[dependentID].Dependencies, dependencyID)
	dag.nodes[dependencyID].Dependents = append(dag.nodes[dependencyID].Dependents, dependentID)

	// 更新依赖任务状态
	if dag.nodes[dependentID].Status == TaskStatusReady && len(dag.nodes[dependentID].Dependencies) > 0 {
		dag.nodes[dependentID].Status = TaskStatusPending
	}

	dag.logger.Info("添加任务依赖关系",
		"dependent", dependentID,
		"dependency", dependencyID)

	return nil
}

// GetReadyTasks 获取所有就绪的任务
func (dag *TaskDAG) GetReadyTasks() []*TaskNode {
	dag.mu.RLock()
	defer dag.mu.RUnlock()

	var readyTasks []*TaskNode
	for _, node := range dag.nodes {
		if node.Status == TaskStatusReady {
			readyTasks = append(readyTasks, node)
		}
	}

	return readyTasks
}

// MarkTaskStarted 标记任务开始执行
func (dag *TaskDAG) MarkTaskStarted(taskID, workerID string) error {
	dag.mu.Lock()
	defer dag.mu.Unlock()

	node, exists := dag.nodes[taskID]
	if !exists {
		return fmt.Errorf("任务 %s 不存在", taskID)
	}

	if node.Status != TaskStatusReady {
		return fmt.Errorf("任务 %s 状态不正确，当前状态: %s", taskID, node.Status.String())
	}

	node.Status = TaskStatusRunning
	node.StartedAt = Some(time.Now())
	node.WorkerID = Some(workerID)
	dag.runningTasks[taskID] = true

	dag.logger.Info("任务开始执行",
		"task_id", taskID,
		"worker_id", workerID)

	return nil
}

// MarkTaskCompleted 标记任务完成
func (dag *TaskDAG) MarkTaskCompleted(taskID string, output map[string]any) error {
	dag.mu.Lock()
	defer dag.mu.Unlock()

	node, exists := dag.nodes[taskID]
	if !exists {
		return fmt.Errorf("任务 %s 不存在", taskID)
	}

	node.Status = TaskStatusCompleted
	node.CompletedAt = Some(time.Now())
	if output != nil {
		node.Output = output
	}

	delete(dag.runningTasks, taskID)
	dag.completedTasks[taskID] = true

	// 检查并更新依赖此任务的任务状态
	dag.updateDependentTasks(taskID)

	dag.logger.Info("任务完成",
		"task_id", taskID,
		"duration", time.Since(node.StartedAt.TakeOr(time.Now())))

	return nil
}

// MarkTaskFailed 标记任务失败
func (dag *TaskDAG) MarkTaskFailed(taskID, errorMsg string) error {
	dag.mu.Lock()
	defer dag.mu.Unlock()

	node, exists := dag.nodes[taskID]
	if !exists {
		return fmt.Errorf("任务 %s 不存在", taskID)
	}

	node.Status = TaskStatusFailed
	node.CompletedAt = Some(time.Now())
	node.ErrorMessage = Some(errorMsg)
	node.RetryCount++

	delete(dag.runningTasks, taskID)
	dag.failedTasks[taskID] = true

	dag.logger.Error("任务失败",
		"task_id", taskID,
		"error", errorMsg,
		"retry_count", node.RetryCount)

	// 检查是否需要重试
	if node.RetryCount < node.MaxRetries {
		// 异步调度重试任务
		go dag.scheduleRetry(taskID)
	} else {
		// 标记依赖此任务的所有任务为跳过
		dag.markDependentTasksSkipped(taskID)
	}

	return nil
}

// RetryTask 重试任务
func (dag *TaskDAG) RetryTask(taskID string) error {
	dag.mu.Lock()
	defer dag.mu.Unlock()

	node, exists := dag.nodes[taskID]
	if !exists {
		return fmt.Errorf("任务 %s 不存在", taskID)
	}

	if node.Status != TaskStatusFailed {
		return fmt.Errorf("只能重试失败的任务，当前状态: %s", node.Status.String())
	}

	// 重置任务状态
	node.Status = TaskStatusReady
	node.StartedAt = None[time.Time]()
	node.CompletedAt = None[time.Time]()
	node.WorkerID = None[string]()
	node.ErrorMessage = None[string]()

	delete(dag.failedTasks, taskID)

	dag.logger.Info("任务重试",
		"task_id", taskID,
		"retry_count", node.RetryCount)

	// 通知有新的就绪任务
	select {
	case dag.readyTasksChan <- taskID:
	default:
		// 通道满了，记录日志但不阻塞
		dag.logger.Warn("就绪任务通道已满", "task_id", taskID)
	}

	return nil
}

// GetTaskOutput 获取任务输出
func (dag *TaskDAG) GetTaskOutput(taskID string) (map[string]any, error) {
	dag.mu.RLock()
	defer dag.mu.RUnlock()

	node, exists := dag.nodes[taskID]
	if !exists {
		return nil, fmt.Errorf("任务 %s 不存在", taskID)
	}

	if node.Status != TaskStatusCompleted {
		return nil, fmt.Errorf("任务 %s 尚未完成，当前状态: %s", taskID, node.Status.String())
	}

	return node.Output, nil
}

// GetDAGStats 获取DAG统计信息
func (dag *TaskDAG) GetDAGStats() DAGStats {
	dag.mu.RLock()
	defer dag.mu.RUnlock()

	stats := DAGStats{
		TotalTasks: len(dag.nodes),
	}

	var totalEstimatedTime time.Duration

	for _, node := range dag.nodes {
		switch node.Status {
		case TaskStatusPending:
			stats.PendingTasks++
		case TaskStatusReady:
			stats.ReadyTasks++
		case TaskStatusRunning:
			stats.RunningTasks++
		case TaskStatusCompleted:
			stats.CompletedTasks++
		case TaskStatusFailed:
			stats.FailedTasks++
		case TaskStatusCancelled:
			stats.CancelledTasks++
		case TaskStatusSkipped:
			stats.SkippedTasks++
		}

		if node.Status != TaskStatusCompleted {
			totalEstimatedTime += node.EstimatedTime
		}
	}

	if stats.TotalTasks > 0 {
		stats.Progress = float64(stats.CompletedTasks) / float64(stats.TotalTasks) * 100
	}
	stats.EstimatedTime = totalEstimatedTime

	return stats
}

// GetTasksByStatus 按状态获取任务
func (dag *TaskDAG) GetTasksByStatus(status TaskStatus) []*TaskNode {
	dag.mu.RLock()
	defer dag.mu.RUnlock()

	var tasks []*TaskNode
	for _, node := range dag.nodes {
		if node.Status == status {
			taskCopy := *node
			tasks = append(tasks, &taskCopy)
		}
	}

	return tasks
}

// GetTaskNode 获取任务节点
func (dag *TaskDAG) GetTaskNode(taskID string) (*TaskNode, error) {
	dag.mu.RLock()
	defer dag.mu.RUnlock()

	node, exists := dag.nodes[taskID]
	if !exists {
		return nil, fmt.Errorf("任务 %s 不存在", taskID)
	}

	// 返回副本
	nodeCopy := *node
	return &nodeCopy, nil
}

// IsDAGCompleted 检查DAG是否全部完成
func (dag *TaskDAG) IsDAGCompleted() bool {
	dag.mu.RLock()
	defer dag.mu.RUnlock()

	for _, node := range dag.nodes {
		if node.Status == TaskStatusPending ||
			node.Status == TaskStatusReady ||
			node.Status == TaskStatusRunning {
			return false
		}
	}

	return true
}

// CancelTask 取消任务
func (dag *TaskDAG) CancelTask(taskID string) error {
	dag.mu.Lock()
	defer dag.mu.Unlock()

	node, exists := dag.nodes[taskID]
	if !exists {
		return fmt.Errorf("任务 %s 不存在", taskID)
	}

	if node.Status == TaskStatusCompleted {
		return fmt.Errorf("任务 %s 已完成，无法取消", taskID)
	}

	node.Status = TaskStatusCancelled
	node.CompletedAt = Some(time.Now())

	delete(dag.runningTasks, taskID)

	// 标记依赖此任务的所有任务为跳过
	dag.markDependentTasksSkipped(taskID)

	dag.logger.Info("任务已取消", "task_id", taskID)

	return nil
}

// WatchReadyTasks 监听就绪任务通道
func (dag *TaskDAG) WatchReadyTasks(ctx context.Context) <-chan string {
	return dag.readyTasksChan
}

// 内部方法

// wouldCreateCycle 检查添加依赖关系是否会产生循环依赖
func (dag *TaskDAG) wouldCreateCycle(dependentID, dependencyID string) bool {
	// 如果添加 dependentID -> dependencyID 的依赖
	// 需要检查从 dependencyID 开始，能否通过现有的依赖路径到达 dependentID
	// 如果能到达，就说明会形成循环
	visited := make(map[string]bool)
	return dag.hasCycle(dependencyID, dependentID, visited)
}

// hasCycle 深度优先搜索检查循环依赖
func (dag *TaskDAG) hasCycle(current, target string, visited map[string]bool) bool {
	if current == target {
		return true
	}

	if visited[current] {
		return false
	}

	visited[current] = true

	// 遍历当前节点的所有依赖（反向边）
	// reverseEdges[current] 包含 current 依赖的所有任务
	for _, dependency := range dag.reverseEdges[current] {
		if dag.hasCycle(dependency, target, visited) {
			return true
		}
	}

	return false
}

// updateDependentTasks 更新依赖此任务的任务状态
func (dag *TaskDAG) updateDependentTasks(completedTaskID string) {
	for _, dependentID := range dag.edges[completedTaskID] {
		node := dag.nodes[dependentID]
		if node.Status == TaskStatusPending {
			// 检查所有依赖是否都已完成
			allDepsCompleted := true
			for _, depID := range node.Dependencies {
				if !dag.completedTasks[depID] {
					allDepsCompleted = false
					break
				}
			}

			if allDepsCompleted {
				node.Status = TaskStatusReady
				dag.logger.Info("任务就绪", "task_id", dependentID)

				// 通知有新的就绪任务
				select {
				case dag.readyTasksChan <- dependentID:
				default:
					dag.logger.Warn("就绪任务通道已满", "task_id", dependentID)
				}
			}
		}
	}
}

// markDependentTasksSkipped 标记依赖此任务的所有任务为跳过
func (dag *TaskDAG) markDependentTasksSkipped(failedTaskID string) {
	visited := make(map[string]bool)
	dag.markSkippedRecursive(failedTaskID, visited)
}

// markSkippedRecursive 递归标记跳过的任务
func (dag *TaskDAG) markSkippedRecursive(taskID string, visited map[string]bool) {
	if visited[taskID] {
		return
	}
	visited[taskID] = true

	for _, dependentID := range dag.edges[taskID] {
		node := dag.nodes[dependentID]
		if node.Status == TaskStatusPending || node.Status == TaskStatusReady {
			node.Status = TaskStatusSkipped
			node.CompletedAt = Some(time.Now())
			dag.logger.Info("任务跳过", "task_id", dependentID, "reason", "依赖任务失败")
		}

		// 递归处理
		dag.markSkippedRecursive(dependentID, visited)
	}
}

// scheduleRetry 调度任务重试
func (dag *TaskDAG) scheduleRetry(taskID string) {
	dag.mu.RLock()
	node, exists := dag.nodes[taskID]
	if !exists {
		dag.mu.RUnlock()
		return
	}

	// 计算重试延迟时间（指数退避）
	retryDelay := time.Duration(node.RetryCount) * time.Second * 5
	dag.mu.RUnlock()

	dag.logger.Info("计划任务重试",
		"task_id", taskID,
		"delay", retryDelay,
		"retry_count", node.RetryCount)

	// 延迟后重试
	time.Sleep(retryDelay)

	// 调用RetryTask方法进行重试
	if err := dag.RetryTask(taskID); err != nil {
		dag.logger.Error("任务重试失败", "task_id", taskID, "error", err)
	}
}

// subscribeToEvents 订阅事件
func (dag *TaskDAG) subscribeToEvents() {
	if dag.eventBus != nil {
		// 订阅任务完成事件
		dag.eventBus.Subscribe(EventTypeTaskCompleted, func(event Event) {
			if taskEvent, ok := event.(*TaskEvent); ok {
				err := dag.MarkTaskCompleted(taskEvent.TaskID, make(map[string]any))
				if err != nil {
					dag.logger.Error("任务完成事件处理失败", "task_id", taskEvent.TaskID, "error", err)
				}
			}
		})

		// 订阅任务失败事件
		dag.eventBus.Subscribe(EventTypeTaskFailed, func(event Event) {
			if taskEvent, ok := event.(*TaskEvent); ok {
				errorMsg := taskEvent.ErrorMsg.TakeOr("未知错误")
				err := dag.MarkTaskFailed(taskEvent.TaskID, errorMsg)
				if err != nil {
					dag.logger.Error("任务失败事件处理失败", "task_id", taskEvent.TaskID, "error", err)
				}
			}
		})
	}
}
