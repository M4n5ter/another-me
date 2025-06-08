package state

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	. "github.com/m4n5ter/another-me/pkg/option"
)

// ExampleUsage 展示状态机的完整使用流程
func ExampleUsage() {
	logger := slog.Default().WithGroup("example")
	logger.Info("开始状态机使用示例")

	// 创建状态管理器
	sm := NewStateManager()

	// 模拟系统启动到执行任务的完整流程
	demonstrateSystemWorkflow(sm, logger)

	// 模拟多任务并发执行
	demonstrateConcurrentTasks(sm, logger)

	// 模拟错误处理和恢复
	demonstrateErrorHandling(sm, logger)

	// 展示状态查询和统计
	demonstrateStateInquiry(sm, logger)

	logger.Info("状态机使用示例完成")
}

// demonstrateSystemWorkflow 演示系统完整工作流程
//
//nolint:errcheck // 示例可以忽略错误检查
func demonstrateSystemWorkflow(sm *StateManager, logger *slog.Logger) {
	logger.Info("=== 演示系统工作流程 ===")

	// 1. 用户提交请求，系统开始分析
	sm.SetSystemState(SystemStateAnalyzing, "用户请求：帮我分析网站数据并生成报告")
	logger.Info("系统状态", "state", sm.GetSystemState().String())

	// 2. 分析完成，开始制定计划
	time.Sleep(100 * time.Millisecond) // 模拟分析时间
	sm.SetSystemState(SystemStatePlanning, "分析完成，开始制定执行计划")

	// 3. 创建主任务
	mainTask := &TaskInfo{
		ID:          "main-task-001",
		Name:        "网站数据分析报告",
		Description: "爬取网站数据，进行分析并生成可视化报告",
		State:       TaskStatePending,
		Priority:    PriorityHigh,
		Metadata: map[string]any{
			"target_url": "https://example.com",
			"data_types": []string{"user_stats", "sales_data", "traffic"},
		},
	}
	sm.CreateTask(mainTask)

	// 4. 分解子任务
	sm.UpdateTaskState("main-task-001", TaskStateDecomposing, "分解为具体子任务")

	subTasks := []TaskInfo{
		{
			ID:         "subtask-001",
			Name:       "数据爬取",
			State:      TaskStatePending,
			Priority:   PriorityNormal,
			ParentTask: Some("main-task-001"),
			Metadata:   map[string]any{"type": "web_scraping"},
		},
		{
			ID:         "subtask-002",
			Name:       "数据分析",
			State:      TaskStatePending,
			Priority:   PriorityNormal,
			ParentTask: Some("main-task-001"),
			Metadata:   map[string]any{"type": "data_analysis"},
		},
		{
			ID:         "subtask-003",
			Name:       "报告生成",
			State:      TaskStatePending,
			Priority:   PriorityNormal,
			ParentTask: Some("main-task-001"),
			Metadata:   map[string]any{"type": "report_generation"},
		},
	}

	for _, task := range subTasks {
		taskCopy := task
		sm.CreateTask(&taskCopy)
	}

	// 5. 注册专门的Worker
	workers := []WorkerInfo{
		{
			ID:    "web-scraper-001",
			Type:  "web_ui",
			State: WorkerStateIdle,
			Tools: []string{"navigate", "click", "extract_data", "scroll"},
			Metadata: map[string]any{
				"specialization": "web_scraping",
				"max_concurrent": 3,
			},
		},
		{
			ID:    "data-analyst-001",
			Type:  "data_analysis",
			State: WorkerStateIdle,
			Tools: []string{"pandas", "numpy", "matplotlib", "seaborn"},
			Metadata: map[string]any{
				"specialization": "statistical_analysis",
			},
		},
		{
			ID:    "report-generator-001",
			Type:  "file_system",
			State: WorkerStateIdle,
			Tools: []string{"create_file", "write_content", "generate_pdf"},
			Metadata: map[string]any{
				"specialization": "document_generation",
			},
		},
	}

	for _, worker := range workers {
		workerCopy := worker
		sm.RegisterWorker(&workerCopy)
	}

	// 6. 系统进入执行状态
	sm.SetSystemState(SystemStateExecuting, "开始执行任务")

	// 7. 调度任务执行
	sm.UpdateTaskState("main-task-001", TaskStateScheduling, "调度子任务")

	// 分配任务给Worker
	taskWorkerPairs := [][2]string{
		{"subtask-001", "web-scraper-001"},
		{"subtask-002", "data-analyst-001"},
		{"subtask-003", "report-generator-001"},
	}

	for _, pair := range taskWorkerPairs {
		taskID, workerID := pair[0], pair[1]
		sm.AssignTaskToWorker(workerID, taskID)
		sm.UpdateWorkerState(workerID, WorkerStateRunning, fmt.Sprintf("开始执行任务 %s", taskID))
		sm.UpdateTaskState(taskID, TaskStateRunning, fmt.Sprintf("由Worker %s 执行", workerID))
	}

	// 8. 模拟任务执行进度
	for i := 1; i <= 10; i++ {
		progress := float64(i * 10)
		for _, pair := range taskWorkerPairs {
			taskID := pair[0]
			sm.UpdateTaskProgress(taskID, progress, None[any](), None[string]())
		}
		time.Sleep(50 * time.Millisecond)

		if i == 10 {
			// 任务完成
			for _, pair := range taskWorkerPairs {
				taskID, workerID := pair[0], pair[1]
				sm.UpdateTaskState(taskID, TaskStateCompleted, "任务执行完成")
				sm.UpdateWorkerState(workerID, WorkerStateCompleted, "任务完成")
			}
		}
	}

	// 9. 主任务完成
	sm.UpdateTaskState("main-task-001", TaskStateCompleted, "所有子任务完成")

	// 10. 系统进入评估阶段
	sm.SetSystemState(SystemStateEvaluating, "评估任务执行结果")
	time.Sleep(100 * time.Millisecond)

	// 11. 学习阶段
	sm.SetSystemState(SystemStateLearning, "从执行经验中学习")
	time.Sleep(100 * time.Millisecond)

	// 12. 回到空闲状态
	sm.SetSystemState(SystemStateIdle, "任务完成，系统空闲")

	logger.Info("系统工作流程演示完成")
}

// demonstrateConcurrentTasks 演示并发任务处理
//
//nolint:errcheck // 示例可以忽略错误检查
func demonstrateConcurrentTasks(sm *StateManager, logger *slog.Logger) {
	logger.Info("=== 演示并发任务处理 ===")

	sm.SetSystemState(SystemStateAnalyzing, "处理多个并发请求")

	// 创建多个并发任务
	concurrentTasks := []TaskInfo{
		{
			ID:       "concurrent-task-1",
			Name:     "邮件处理",
			State:    TaskStatePending,
			Priority: PriorityNormal,
			Metadata: map[string]any{"type": "email_processing"},
		},
		{
			ID:       "concurrent-task-2",
			Name:     "数据备份",
			State:    TaskStatePending,
			Priority: PriorityLow,
			Metadata: map[string]any{"type": "data_backup"},
		},
		{
			ID:       "concurrent-task-3",
			Name:     "系统监控",
			State:    TaskStatePending,
			Priority: PriorityCritical,
			Metadata: map[string]any{"type": "system_monitoring"},
		},
	}

	for _, task := range concurrentTasks {
		taskCopy := task
		sm.CreateTask(&taskCopy)
	}

	// 创建临时Worker处理这些任务
	tempWorkers := []WorkerInfo{
		{
			ID:    "temp-worker-1",
			Type:  "temporary",
			State: WorkerStateInitializing,
			Tools: []string{"email_api", "imap", "smtp"},
		},
		{
			ID:    "temp-worker-2",
			Type:  "temporary",
			State: WorkerStateInitializing,
			Tools: []string{"rsync", "tar", "gzip"},
		},
		{
			ID:    "temp-worker-3",
			Type:  "temporary",
			State: WorkerStateInitializing,
			Tools: []string{"system_stats", "alerts", "notifications"},
		},
	}

	for _, worker := range tempWorkers {
		workerCopy := worker
		sm.RegisterWorker(&workerCopy)
		sm.UpdateWorkerState(workerCopy.ID, WorkerStateIdle, "初始化完成")
	}

	// 根据优先级分配任务（优先级高的先执行）
	sm.SetSystemState(SystemStateExecuting, "执行并发任务")

	// 紧急任务优先
	sm.AssignTaskToWorker("temp-worker-3", "concurrent-task-3")
	sm.UpdateTaskState("concurrent-task-3", TaskStateRunning, "紧急任务优先执行")
	sm.UpdateWorkerState("temp-worker-3", WorkerStateRunning, "执行紧急监控任务")

	// 然后处理其他任务
	sm.AssignTaskToWorker("temp-worker-1", "concurrent-task-1")
	sm.UpdateTaskState("concurrent-task-1", TaskStateRunning, "开始处理邮件")
	sm.UpdateWorkerState("temp-worker-1", WorkerStateRunning, "处理邮件任务")

	sm.AssignTaskToWorker("temp-worker-2", "concurrent-task-2")
	sm.UpdateTaskState("concurrent-task-2", TaskStateRunning, "开始数据备份")
	sm.UpdateWorkerState("temp-worker-2", WorkerStateRunning, "执行备份任务")

	// 模拟并发执行
	time.Sleep(200 * time.Millisecond)

	// 完成任务
	for i, task := range concurrentTasks {
		sm.UpdateTaskState(task.ID, TaskStateCompleted, "并发任务完成")
		workerID := fmt.Sprintf("temp-worker-%d", i+1)
		sm.UpdateWorkerState(workerID, WorkerStateCompleted, "任务完成")
		// 临时Worker完成后销毁
		sm.UnregisterWorker(workerID, "临时Worker任务完成")
	}

	sm.SetSystemState(SystemStateIdle, "并发任务处理完成")
	logger.Info("并发任务处理演示完成")
}

// demonstrateErrorHandling 演示错误处理和恢复
//
//nolint:errcheck // 示例可以忽略错误检查
func demonstrateErrorHandling(sm *StateManager, logger *slog.Logger) {
	logger.Info("=== 演示错误处理和恢复 ===")

	// 创建一个会失败的任务
	errorTask := &TaskInfo{
		ID:          "error-task-001",
		Name:        "网络请求任务",
		Description: "这个任务会遇到网络错误",
		State:       TaskStatePending,
		Priority:    PriorityNormal,
		Metadata:    map[string]any{"url": "https://invalid-url.com"},
	}
	sm.CreateTask(errorTask)

	// 创建Worker执行任务
	errorWorker := &WorkerInfo{
		ID:    "error-prone-worker",
		Type:  "web_ui",
		State: WorkerStateIdle,
		Tools: []string{"http_request", "retry"},
	}
	sm.RegisterWorker(errorWorker)

	sm.SetSystemState(SystemStateExecuting, "执行可能失败的任务")

	// 分配任务
	sm.AssignTaskToWorker("error-prone-worker", "error-task-001")
	sm.UpdateTaskState("error-task-001", TaskStateRunning, "开始执行网络请求")
	sm.UpdateWorkerState("error-prone-worker", WorkerStateRunning, "执行网络请求")

	// 模拟执行失败
	time.Sleep(100 * time.Millisecond)
	sm.UpdateTaskState("error-task-001", TaskStateFailed, "网络连接超时")
	sm.UpdateWorkerState("error-prone-worker", WorkerStateError, "网络请求失败")

	// 系统检测到错误
	sm.SetSystemState(SystemStateError, "检测到任务执行失败")

	// 分析失败原因并重试
	logger.Info("分析失败原因", "task", "error-task-001", "reason", "网络连接超时")

	// 尝试恢复
	sm.UpdateTaskState("error-task-001", TaskStateRetrying, "重试任务执行")
	sm.UpdateWorkerState("error-prone-worker", WorkerStateIdle, "Worker恢复正常")

	// 重新执行
	sm.SetSystemState(SystemStateExecuting, "重试任务执行")
	sm.UpdateTaskState("error-task-001", TaskStateRunning, "重试执行（使用备用URL）")
	sm.UpdateWorkerState("error-prone-worker", WorkerStateRunning, "使用备用策略重试")

	// 模拟重试成功
	time.Sleep(100 * time.Millisecond)
	sm.UpdateTaskState("error-task-001", TaskStateCompleted, "重试成功完成")
	sm.UpdateWorkerState("error-prone-worker", WorkerStateCompleted, "重试成功")

	// 系统恢复正常
	sm.SetSystemState(SystemStateEvaluating, "评估恢复结果")
	time.Sleep(50 * time.Millisecond)
	sm.SetSystemState(SystemStateIdle, "系统恢复正常")

	logger.Info("错误处理和恢复演示完成")
}

// demonstrateStateInquiry 演示状态查询和统计
func demonstrateStateInquiry(sm *StateManager, logger *slog.Logger) {
	logger.Info("=== 演示状态查询和统计 ===")

	// 获取系统整体信息
	systemInfo := sm.GetSystemInfo()
	logger.Info("系统信息",
		"state", systemInfo.State.String(),
		"uptime", systemInfo.Uptime.String(),
		"active_tasks", systemInfo.ActiveTasks,
		"completed_tasks", systemInfo.CompletedTasks,
		"active_workers", systemInfo.ActiveWorkers,
	)

	// 获取所有任务
	allTasks := sm.ListTasks()
	logger.Info("任务总览", "total_tasks", len(allTasks))

	// 按状态分组统计任务
	tasksByState := make(map[TaskState]int)
	for _, task := range allTasks {
		tasksByState[task.State]++
	}

	for state, count := range tasksByState {
		logger.Info("任务状态统计", "state", state.String(), "count", count)
	}

	// 获取所有Worker
	allWorkers := sm.ListWorkers()
	logger.Info("Worker总览", "total_workers", len(allWorkers))

	// 获取状态转换历史
	transitions := sm.GetStateTransitions(10)
	logger.Info("最近状态转换", "count", len(transitions))

	for i, transition := range transitions {
		if i < 5 { // 只显示前5条
			logger.Info("状态转换历史",
				"entity", transition.EntityType,
				"id", transition.EntityID,
				"from", transition.FromState,
				"to", transition.ToState,
				"reason", transition.Reason,
				"time", transition.Timestamp.Format("15:04:05"),
			)
		}
	}

	// 获取统计信息
	stats := sm.GetStatistics()
	logger.Info("系统统计", "stats", stats)

	logger.Info("状态查询和统计演示完成")
}

// DemonstrateWithContext 演示在上下文中使用状态管理器
func DemonstrateWithContext() {
	logger := slog.Default().WithGroup("context_demo")
	logger.Info("演示上下文使用")

	sm := NewStateManager()
	ctx := ContextWithStateManager(context.Background(), sm)

	// 模拟在不同组件中使用状态管理器
	orchestratorComponent(ctx, logger)
	workerComponent(ctx, logger)
	monitoringComponent(ctx, logger)
}

// orchestratorComponent 模拟Orchestrator组件
//
//nolint:errcheck // 示例可以忽略错误检查
func orchestratorComponent(ctx context.Context, logger *slog.Logger) {
	sm, ok := StateManagerFromContext(ctx)
	if !ok {
		logger.Error("无法从上下文获取状态管理器")
		return
	}

	logger.Info("Orchestrator: 创建新任务")
	task := &TaskInfo{
		ID:       "context-task-001",
		Name:     "上下文演示任务",
		State:    TaskStatePending,
		Priority: PriorityNormal,
		Metadata: make(map[string]any),
	}
	sm.CreateTask(task)
	sm.SetSystemState(SystemStateAnalyzing, "Orchestrator开始分析任务")
}

// workerComponent 模拟Worker组件
//
//nolint:errcheck // 示例可以忽略错误检查
func workerComponent(ctx context.Context, logger *slog.Logger) {
	sm, ok := StateManagerFromContext(ctx)
	if !ok {
		logger.Error("无法从上下文获取状态管理器")
		return
	}

	logger.Info("Worker: 注册并执行任务")
	worker := &WorkerInfo{
		ID:    "context-worker-001",
		Type:  "temporary",
		State: WorkerStateIdle,
		Tools: []string{"generic_tool"},
	}
	sm.RegisterWorker(worker)
	sm.AssignTaskToWorker("context-worker-001", "context-task-001")
	sm.UpdateTaskState("context-task-001", TaskStateRunning, "Worker开始执行")
}

// monitoringComponent 模拟监控组件
func monitoringComponent(ctx context.Context, logger *slog.Logger) {
	sm, ok := StateManagerFromContext(ctx)
	if !ok {
		logger.Error("无法从上下文获取状态管理器")
		return
	}

	logger.Info("监控: 检查系统状态")
	systemInfo := sm.GetSystemInfo()
	logger.Info("监控报告",
		"system_state", systemInfo.State.String(),
		"total_tasks", len(sm.ListTasks()),
		"total_workers", len(sm.ListWorkers()),
	)
}
