package communication

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"time"

	. "github.com/m4n5ter/another-me/pkg/option"
)

// ExampleComplexWorkflow 演示复杂的任务执行流程
func ExampleComplexWorkflow() {
	logger := slog.Default().WithGroup("communication_example")
	logger.Info("开始复杂工作流演示")

	// 1. 创建消息总线
	eventBus := NewMessageBus(1000, 4) // 1000个事件缓冲，4个工作协程
	defer eventBus.Close()

	// 2. 创建组件注册表
	registry := NewComponentRegistry(eventBus)
	defer registry.Close()

	// 3. 创建任务DAG管理器
	dag := NewTaskDAG(eventBus, 3) // 最多3个并发任务

	// 4. 模拟注册各种组件
	registerComponents(registry, logger)

	// 5. 创建复杂的任务依赖图
	createComplexTaskDAG(dag, logger)

	// 6. 模拟Orchestrator组件
	orchestratorCtx, orchestratorCancel := context.WithCancel(context.Background())
	go simulateOrchestrator(orchestratorCtx, eventBus, registry, dag, logger)

	// 7. 模拟Worker组件
	workerCtx, workerCancel := context.WithCancel(context.Background())
	go simulateWorkers(workerCtx, eventBus, registry, dag, logger)

	// 8. 运行工作流并观察进度
	runWorkflowDemo(eventBus, registry, dag, logger)

	// 9. 清理资源
	orchestratorCancel()
	workerCancel()

	// 10. 打印最终统计信息
	printFinalStats(eventBus, registry, dag, logger)

	logger.Info("复杂工作流演示完成")
}

// registerComponents 注册各种组件
//
//nolint:errcheck // 示例可以忽略错误检查
func registerComponents(registry *ComponentRegistry, logger *slog.Logger) {
	logger.Info("=== 注册系统组件 ===")

	// 注册Orchestrator
	orchestrator := &ComponentInfo{
		ID:           "orchestrator-main",
		Type:         ComponentTypeOrchestrator,
		Name:         "主编排器",
		Version:      "1.0.0",
		Capabilities: []string{"task_planning", "task_decomposition", "resource_allocation"},
		Config: map[string]any{
			"max_concurrent_tasks": 10,
			"planning_algorithm":   "dependency_aware",
		},
	}
	registry.RegisterComponent(orchestrator)

	// 注册多个Worker
	workers := []*ComponentInfo{
		{
			ID:           "web-worker-01",
			Type:         ComponentTypeWorker,
			Name:         "网页操作Worker #1",
			Version:      "2.1.0",
			Capabilities: []string{"web_scraping", "form_filling", "navigation"},
			Config: map[string]any{
				"browser":         "chrome",
				"max_tabs":        5,
				"timeout_seconds": 30,
			},
		},
		{
			ID:           "data-worker-01",
			Type:         ComponentTypeWorker,
			Name:         "数据分析Worker #1",
			Version:      "1.5.0",
			Capabilities: []string{"data_analysis", "visualization", "statistical_analysis"},
			Config: map[string]any{
				"pandas_version": "2.0.0",
				"memory_limit":   "8GB",
			},
		},
		{
			ID:           "file-worker-01",
			Type:         ComponentTypeWorker,
			Name:         "文件操作Worker #1",
			Version:      "1.2.0",
			Capabilities: []string{"file_operations", "document_generation", "data_export"},
			Config: map[string]any{
				"temp_dir":      "/tmp/worker",
				"max_file_size": "100MB",
			},
		},
	}

	for _, worker := range workers {
		registry.RegisterComponent(worker)
	}

	// 注册监控组件
	monitor := &ComponentInfo{
		ID:           "monitor-main",
		Type:         ComponentTypeMonitor,
		Name:         "系统监控器",
		Version:      "1.0.0",
		Capabilities: []string{"performance_monitoring", "health_check", "alerting"},
		Config: map[string]any{
			"metrics_interval": 10,
			"alert_threshold":  0.85,
		},
	}
	registry.RegisterComponent(monitor)

	logger.Info("组件注册完成", "total_components", len(registry.ListComponents()))
}

// createComplexTaskDAG 创建复杂的任务依赖图
//
//nolint:errcheck // 示例可以忽略错误检查
func createComplexTaskDAG(dag *TaskDAG, logger *slog.Logger) {
	logger.Info("=== 创建复杂任务依赖图 ===")

	// 第一层：数据收集任务（可并行）
	collectTasks := []*TaskNode{
		{
			ID:            "collect-web-data",
			Name:          "收集网页数据",
			Type:          "web_scraping",
			Priority:      1,
			EstimatedTime: 30 * time.Second,
			Metadata: map[string]any{
				"target_urls": []string{"https://example1.com", "https://example2.com"},
				"data_types":  []string{"user_stats", "content"},
			},
		},
		{
			ID:            "collect-api-data",
			Name:          "收集API数据",
			Type:          "api_call",
			Priority:      1,
			EstimatedTime: 20 * time.Second,
			Metadata: map[string]any{
				"api_endpoints": []string{"/users", "/orders", "/products"},
				"rate_limit":    100,
			},
		},
		{
			ID:            "collect-file-data",
			Name:          "收集文件数据",
			Type:          "file_processing",
			Priority:      1,
			EstimatedTime: 15 * time.Second,
			Metadata: map[string]any{
				"file_paths": []string{"/data/sales.csv", "/data/users.json"},
				"encoding":   "utf-8",
			},
		},
	}

	for _, task := range collectTasks {
		dag.AddTask(task)
	}

	// 第二层：数据清洗任务（依赖第一层）
	cleanTasks := []*TaskNode{
		{
			ID:            "clean-web-data",
			Name:          "清洗网页数据",
			Type:          "data_cleaning",
			Priority:      2,
			EstimatedTime: 25 * time.Second,
			Dependencies:  []string{"collect-web-data"},
			Metadata: map[string]any{
				"operations": []string{"remove_duplicates", "normalize_text", "extract_entities"},
			},
		},
		{
			ID:            "clean-api-data",
			Name:          "清洗API数据",
			Type:          "data_cleaning",
			Priority:      2,
			EstimatedTime: 20 * time.Second,
			Dependencies:  []string{"collect-api-data"},
			Metadata: map[string]any{
				"operations": []string{"validate_schema", "fill_missing", "standardize_format"},
			},
		},
		{
			ID:            "clean-file-data",
			Name:          "清洗文件数据",
			Type:          "data_cleaning",
			Priority:      2,
			EstimatedTime: 18 * time.Second,
			Dependencies:  []string{"collect-file-data"},
			Metadata: map[string]any{
				"operations": []string{"parse_dates", "convert_types", "remove_outliers"},
			},
		},
	}

	for _, task := range cleanTasks {
		dag.AddTask(task)
		// 添加依赖关系
		for _, depID := range task.Dependencies {
			dag.AddDependency(task.ID, depID)
		}
	}

	// 第三层：数据合并任务（依赖所有清洗任务）
	mergeTask := &TaskNode{
		ID:            "merge-all-data",
		Name:          "合并所有数据",
		Type:          "data_merge",
		Priority:      3,
		EstimatedTime: 30 * time.Second,
		Dependencies:  []string{"clean-web-data", "clean-api-data", "clean-file-data"},
		Metadata: map[string]any{
			"merge_strategy": "outer_join",
			"join_keys":      []string{"user_id", "timestamp"},
		},
	}
	dag.AddTask(mergeTask)
	for _, depID := range mergeTask.Dependencies {
		dag.AddDependency(mergeTask.ID, depID)
	}

	// 第四层：分析任务（依赖合并任务，可并行）
	analysisTasks := []*TaskNode{
		{
			ID:            "statistical-analysis",
			Name:          "统计分析",
			Type:          "statistical_analysis",
			Priority:      4,
			EstimatedTime: 40 * time.Second,
			Dependencies:  []string{"merge-all-data"},
			Metadata: map[string]any{
				"methods": []string{"correlation", "regression", "clustering"},
			},
		},
		{
			ID:            "trend-analysis",
			Name:          "趋势分析",
			Type:          "trend_analysis",
			Priority:      4,
			EstimatedTime: 35 * time.Second,
			Dependencies:  []string{"merge-all-data"},
			Metadata: map[string]any{
				"time_period": "last_6_months",
				"granularity": "daily",
			},
		},
	}

	for _, task := range analysisTasks {
		dag.AddTask(task)
		for _, depID := range task.Dependencies {
			dag.AddDependency(task.ID, depID)
		}
	}

	// 第五层：报告生成（依赖所有分析任务）
	reportTask := &TaskNode{
		ID:            "generate-report",
		Name:          "生成分析报告",
		Type:          "report_generation",
		Priority:      5,
		EstimatedTime: 25 * time.Second,
		Dependencies:  []string{"statistical-analysis", "trend-analysis"},
		Metadata: map[string]any{
			"format":         "pdf",
			"include_charts": true,
			"template":       "executive_summary",
		},
	}
	dag.AddTask(reportTask)
	for _, depID := range reportTask.Dependencies {
		dag.AddDependency(reportTask.ID, depID)
	}

	stats := dag.GetDAGStats()
	logger.Info("任务DAG创建完成",
		"total_tasks", stats.TotalTasks,
		"ready_tasks", stats.ReadyTasks,
		"estimated_time", stats.EstimatedTime)
}

// simulateOrchestrator 模拟Orchestrator组件行为
//
//nolint:errcheck // 示例可以忽略错误检查
func simulateOrchestrator(ctx context.Context, eventBus *MessageBus, registry *ComponentRegistry, dag *TaskDAG, logger *slog.Logger) {
	logger.Info("=== 启动Orchestrator模拟 ===")

	// 获取Orchestrator的消息通道
	msgChan, err := registry.GetComponentMessageChannel("orchestrator-main")
	if err != nil {
		logger.Error("获取Orchestrator消息通道失败", "error", err)
		return
	}

	// 订阅任务相关事件
	eventBus.Subscribe(EventTypeTaskCompleted, func(event Event) {
		if taskEvent, ok := event.(*TaskEvent); ok {
			logger.Info("Orchestrator收到任务完成事件",
				"task_id", taskEvent.TaskID,
				"worker_id", taskEvent.WorkerID.TakeOr("unknown"))

			// 发布新的就绪任务
			readyTasks := dag.GetReadyTasks()
			for _, task := range readyTasks {
				taskStartEvent := NewTaskEvent(EventTypeTaskStarted, "orchestrator-main", task.ID, task.Name)
				eventBus.Publish(taskStartEvent)
			}
		}
	})

	// 定期检查DAG状态并调度任务
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("Orchestrator模拟退出")
			return
		case <-ticker.C:
			// 检查就绪任务并分配给Worker
			readyTasks := dag.GetReadyTasks()
			if len(readyTasks) > 0 {
				logger.Info("发现就绪任务", "count", len(readyTasks))
				for _, task := range readyTasks {
					assignTaskToWorker(task, registry, dag, eventBus, logger)
				}
			}

			// 检查DAG是否完成
			if dag.IsDAGCompleted() {
				logger.Info("所有任务已完成！")
				stats := dag.GetDAGStats()
				logger.Info("最终DAG统计", "stats", stats)
				return
			}
		case msg := <-msgChan:
			logger.Debug("Orchestrator收到消息", "event_type", string(msg.EventType()))
		}
	}
}

// simulateWorkers 模拟Worker组件行为
//
//nolint:errcheck // 示例可以忽略错误检查
func simulateWorkers(ctx context.Context, eventBus *MessageBus, registry *ComponentRegistry, dag *TaskDAG, logger *slog.Logger) {
	logger.Info("=== 启动Worker模拟 ===")

	workers := registry.ListComponentsByType(ComponentTypeWorker)

	// 为每个Worker启动独立的处理协程
	for _, worker := range workers {
		go simulateWorker(ctx, worker.ID, eventBus, registry, dag, logger)
	}

	// 定期发送心跳
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("Worker模拟退出")
			return
		case <-ticker.C:
			// 发送心跳给所有Worker
			for _, worker := range workers {
				heartbeatEvent := NewComponentEvent(EventTypeHeartbeat, worker.ID, worker.ID, ComponentTypeWorker)
				eventBus.Publish(heartbeatEvent)
			}
		}
	}
}

// simulateWorker 模拟单个Worker
func simulateWorker(ctx context.Context, workerID string, eventBus *MessageBus, registry *ComponentRegistry, dag *TaskDAG, logger *slog.Logger) {
	workerLogger := logger.With("worker_id", workerID)

	// 获取Worker的消息通道
	msgChan, err := registry.GetComponentMessageChannel(workerID)
	if err != nil {
		workerLogger.Error("获取Worker消息通道失败", "error", err)
		return
	}

	// 订阅任务开始事件
	eventBus.Subscribe(EventTypeTaskStarted, func(event Event) {
		if taskEvent, ok := event.(*TaskEvent); ok {
			// 检查是否适合这个Worker执行
			if canWorkerExecuteTask(workerID, taskEvent.TaskID, registry, dag) {
				executeTask(workerID, taskEvent.TaskID, eventBus, registry, dag, workerLogger)
			}
		}
	})

	for {
		select {
		case <-ctx.Done():
			workerLogger.Info("Worker退出")
			return
		case msg := <-msgChan:
			workerLogger.Debug("Worker收到消息", "event_type", string(msg.EventType()))
		}
	}
}

// assignTaskToWorker 分配任务给Worker
//
//nolint:errcheck // 示例可以忽略错误检查
func assignTaskToWorker(task *TaskNode, registry *ComponentRegistry, dag *TaskDAG, eventBus *MessageBus, logger *slog.Logger) {
	// 根据任务类型选择合适的Worker
	var selectedWorker *ComponentInfo
	workers := registry.ListComponentsByType(ComponentTypeWorker)

	for _, worker := range workers {
		if worker.Status == ComponentStatusActive || worker.Status == ComponentStatusIdle {
			// 检查Worker是否有执行此类型任务的能力
			if hasCapabilityForTask(worker, task) {
				selectedWorker = worker
				break
			}
		}
	}

	if selectedWorker == nil {
		logger.Warn("没有找到合适的Worker", "task_id", task.ID, "task_type", task.Type)
		return
	}

	// 标记任务开始
	dag.MarkTaskStarted(task.ID, selectedWorker.ID)
	registry.UpdateComponentStatus(selectedWorker.ID, ComponentStatusBusy)

	logger.Info("任务分配",
		"task_id", task.ID,
		"worker_id", selectedWorker.ID,
		"task_type", task.Type)

	// 发布任务开始事件
	taskEvent := NewTaskEvent(EventTypeTaskStarted, "orchestrator-main", task.ID, task.Name)
	taskEvent.WorkerID = Some(selectedWorker.ID)
	eventBus.Publish(taskEvent)
}

// canWorkerExecuteTask 检查Worker是否能执行任务
func canWorkerExecuteTask(workerID, taskID string, registry *ComponentRegistry, dag *TaskDAG) bool {
	worker, err := registry.GetComponent(workerID)
	if err != nil {
		return false
	}

	task, err := dag.GetTaskNode(taskID)
	if err != nil {
		return false
	}

	return hasCapabilityForTask(worker, task)
}

// hasCapabilityForTask 检查Worker是否有执行任务的能力
func hasCapabilityForTask(worker *ComponentInfo, task *TaskNode) bool {
	taskTypeMap := map[string][]string{
		"web_scraping":         {"web_scraping", "navigation"},
		"api_call":             {"web_scraping"}, // Web Worker也可以处理API调用
		"file_processing":      {"file_operations"},
		"data_cleaning":        {"data_analysis"},
		"data_merge":           {"data_analysis"},
		"statistical_analysis": {"data_analysis", "statistical_analysis"},
		"trend_analysis":       {"data_analysis"},
		"report_generation":    {"document_generation", "data_export"},
	}

	requiredCapabilities, exists := taskTypeMap[task.Type]
	if !exists {
		return false
	}

	for _, required := range requiredCapabilities {
		hasCapability := slices.Contains(worker.Capabilities, required)
		if hasCapability {
			return true // 只要有一个匹配的能力就可以
		}
	}

	return false
}

// executeTask 执行任务
//
//nolint:errcheck // 示例可以忽略错误检查
func executeTask(workerID, taskID string, eventBus *MessageBus, registry *ComponentRegistry, dag *TaskDAG, logger *slog.Logger) {
	logger.Info("开始执行任务", "task_id", taskID)

	// 获取任务信息
	task, err := dag.GetTaskNode(taskID)
	if err != nil {
		logger.Error("获取任务失败", "error", err)
		return
	}

	// 模拟任务执行
	executionTime := task.EstimatedTime
	if executionTime == 0 {
		executionTime = 10 * time.Second // 默认执行时间
	}

	// 模拟可能的执行失败（10%几率）
	if time.Now().UnixNano()%10 == 0 {
		time.Sleep(executionTime / 3) // 部分执行时间

		// 任务失败
		errorMsg := fmt.Sprintf("Worker %s 执行任务 %s 时发生模拟错误", workerID, taskID)
		dag.MarkTaskFailed(taskID, errorMsg)
		registry.UpdateComponentStatus(workerID, ComponentStatusError)

		// 发布任务失败事件
		taskEvent := NewTaskEvent(EventTypeTaskFailed, workerID, taskID, task.Name)
		taskEvent.WorkerID = Some(workerID)
		taskEvent.ErrorMsg = Some(errorMsg)
		eventBus.Publish(taskEvent)

		logger.Error("任务执行失败", "task_id", taskID, "error", errorMsg)

		// Worker恢复正常状态（模拟错误恢复）
		time.Sleep(2 * time.Second)
		registry.UpdateComponentStatus(workerID, ComponentStatusActive)
		return
	}

	// 正常执行
	time.Sleep(executionTime)

	// 生成任务输出
	output := map[string]any{
		"executed_by":    workerID,
		"execution_time": executionTime.Seconds(),
		"status":         "success",
		"result_size":    1000 + time.Now().UnixNano()%5000, // 模拟结果大小
		"timestamp":      time.Now().Unix(),
	}

	// 标记任务完成
	dag.MarkTaskCompleted(taskID, output)
	registry.UpdateComponentStatus(workerID, ComponentStatusIdle)

	// 发布任务完成事件
	taskEvent := NewTaskEvent(EventTypeTaskCompleted, workerID, taskID, task.Name)
	taskEvent.WorkerID = Some(workerID)
	taskEvent.Result = Some(any(output))
	eventBus.Publish(taskEvent)

	logger.Info("任务执行完成",
		"task_id", taskID,
		"execution_time", executionTime,
		"output_size", output["result_size"])
}

// runWorkflowDemo 运行工作流演示
func runWorkflowDemo(eventBus *MessageBus, registry *ComponentRegistry, dag *TaskDAG, logger *slog.Logger) {
	logger.Info("=== 开始工作流执行 ===")

	// 启动初始任务（没有依赖的任务）
	readyTasks := dag.GetReadyTasks()
	logger.Info("初始就绪任务", "count", len(readyTasks))

	for _, task := range readyTasks {
		assignTaskToWorker(task, registry, dag, eventBus, logger)
	}

	// 监控进度
	progressTicker := time.NewTicker(10 * time.Second)
	defer progressTicker.Stop()

	startTime := time.Now()
	maxDuration := 5 * time.Minute // 最大等待时间

	for {
		select {
		case <-progressTicker.C:
			stats := dag.GetDAGStats()
			logger.Info("工作流进度",
				"progress", fmt.Sprintf("%.1f%%", stats.Progress),
				"completed", stats.CompletedTasks,
				"running", stats.RunningTasks,
				"failed", stats.FailedTasks,
				"elapsed", time.Since(startTime).Round(time.Second))

			// 检查是否完成
			if dag.IsDAGCompleted() {
				logger.Info("工作流执行完成！",
					"total_time", time.Since(startTime).Round(time.Second))
				return
			}

			// 检查超时
			if time.Since(startTime) > maxDuration {
				logger.Warn("工作流执行超时")
				return
			}

		default:
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// printFinalStats 打印最终统计信息
func printFinalStats(eventBus *MessageBus, registry *ComponentRegistry, dag *TaskDAG, logger *slog.Logger) {
	logger.Info("=== 最终统计信息 ===")

	// 消息总线统计
	busStats := eventBus.GetStats()
	logger.Info("消息总线统计",
		"total_events", busStats.TotalEvents,
		"processed_events", busStats.ProcessedEvents,
		"failed_events", busStats.FailedEvents,
		"active_subscribers", busStats.ActiveSubscribers,
		"average_latency", busStats.AverageLatency)

	// 组件注册表统计
	registryStats := registry.GetRegistryStats()
	logger.Info("组件注册表统计", "stats", registryStats)

	// DAG统计
	dagStats := dag.GetDAGStats()
	logger.Info("任务DAG统计",
		"total_tasks", dagStats.TotalTasks,
		"completed_tasks", dagStats.CompletedTasks,
		"failed_tasks", dagStats.FailedTasks,
		"progress", fmt.Sprintf("%.1f%%", dagStats.Progress))

	// 列出所有任务的最终状态
	logger.Info("=== 任务执行详情 ===")
	allTasks := []TaskStatus{TaskStatusCompleted, TaskStatusFailed, TaskStatusCancelled, TaskStatusSkipped}
	for _, status := range allTasks {
		tasks := dag.GetTasksByStatus(status)
		if len(tasks) > 0 {
			logger.Info("任务执行详情", "status", status.String(), "count", len(tasks))
			for _, task := range tasks {
				logger.Info("任务详情",
					"task_id", task.ID,
					"name", task.Name,
					"type", task.Type,
					"retry_count", task.RetryCount)
			}
		}
	}
}
