package state

import (
	"fmt"
	"testing"

	. "github.com/m4n5ter/another-me/pkg/option"
)

// TestStateManager 测试状态管理器的基本功能
func TestStateManager(t *testing.T) {
	sm := NewStateManager()

	// 测试系统状态
	t.Run("SystemState", func(t *testing.T) {
		// 初始状态应该是Idle
		if sm.GetSystemState() != SystemStateIdle {
			t.Errorf("期望初始状态为Idle，实际为 %v", sm.GetSystemState())
		}

		// 测试状态转换
		err := sm.SetSystemState(SystemStateAnalyzing, "开始分析用户请求")
		if err != nil {
			t.Errorf("设置系统状态失败: %v", err)
		}

		if sm.GetSystemState() != SystemStateAnalyzing {
			t.Errorf("期望状态为Analyzing，实际为 %v", sm.GetSystemState())
		}
	})

	// 测试任务管理
	t.Run("TaskManagement", func(t *testing.T) {
		task := &TaskInfo{
			ID:          "task-001",
			Name:        "测试任务",
			Description: "这是一个测试任务",
			State:       TaskStatePending,
			Priority:    PriorityNormal,
			Progress:    0,
			Metadata:    make(map[string]any),
		}

		// 创建任务
		err := sm.CreateTask(task)
		if err != nil {
			t.Errorf("创建任务失败: %v", err)
		}

		// 获取任务
		retrievedTask, err := sm.GetTask("task-001")
		if err != nil {
			t.Errorf("获取任务失败: %v", err)
		}

		if retrievedTask.Name != "测试任务" {
			t.Errorf("期望任务名称为'测试任务'，实际为 %v", retrievedTask.Name)
		}

		// 更新任务状态
		err = sm.UpdateTaskState("task-001", TaskStateRunning, "开始执行任务")
		if err != nil {
			t.Errorf("更新任务状态失败: %v", err)
		}

		// 更新任务进度
		err = sm.UpdateTaskProgress("task-001", 50.0, None[any](), None[string]())
		if err != nil {
			t.Errorf("更新任务进度失败: %v", err)
		}

		// 再次获取任务验证更新
		updatedTask, err := sm.GetTask("task-001")
		if err != nil {
			t.Errorf("获取更新后的任务失败: %v", err)
		}

		if updatedTask.State != TaskStateRunning {
			t.Errorf("期望任务状态为Running，实际为 %v", updatedTask.State)
		}

		if updatedTask.Progress != 50.0 {
			t.Errorf("期望任务进度为50.0，实际为 %v", updatedTask.Progress)
		}
	})

	// 测试Worker管理
	t.Run("WorkerManagement", func(t *testing.T) {
		worker := &WorkerInfo{
			ID:          "worker-001",
			Type:        "web_ui",
			State:       WorkerStateIdle,
			Tools:       []string{"click", "type", "scroll"},
			Metadata:    make(map[string]any),
			Performance: PerformanceMetrics{},
		}

		// 注册Worker
		err := sm.RegisterWorker(worker)
		if err != nil {
			t.Errorf("注册Worker失败: %v", err)
		}

		// 获取Worker
		retrievedWorker, err := sm.GetWorker("worker-001")
		if err != nil {
			t.Errorf("获取Worker失败: %v", err)
		}

		if retrievedWorker.Type != "web_ui" {
			t.Errorf("期望Worker类型为'web_ui'，实际为 %v", retrievedWorker.Type)
		}

		// 更新Worker状态
		err = sm.UpdateWorkerState("worker-001", WorkerStateRunning, "开始执行任务")
		if err != nil {
			t.Errorf("更新Worker状态失败: %v", err)
		}

		// 分配任务给Worker
		err = sm.AssignTaskToWorker("worker-001", "task-001")
		if err != nil {
			t.Errorf("分配任务给Worker失败: %v", err)
		}

		// 验证任务分配
		updatedWorker, err := sm.GetWorker("worker-001")
		if err != nil {
			t.Errorf("获取更新后的Worker失败: %v", err)
		}

		if !updatedWorker.CurrentTask.IsSome() || updatedWorker.CurrentTask.Unwrap() != "task-001" {
			t.Errorf("期望Worker当前任务为'task-001'，实际为 %v", updatedWorker.CurrentTask)
		}
	})

	// 测试状态转换历史
	t.Run("StateTransitions", func(t *testing.T) {
		transitions := sm.GetStateTransitions(10)
		if len(transitions) == 0 {
			t.Error("期望有状态转换记录，但实际为空")
		}

		// 测试按实体获取转换历史
		taskTransitions := sm.GetStateTransitionsByEntity("task", "task-001", 5)
		if len(taskTransitions) == 0 {
			t.Error("期望有任务状态转换记录，但实际为空")
		}
	})

	// 测试统计信息
	t.Run("Statistics", func(t *testing.T) {
		stats := sm.GetStatistics()

		if stats["system"] == nil {
			t.Error("期望有系统统计信息")
		}

		if stats["tasks"] == nil {
			t.Error("期望有任务统计信息")
		}

		if stats["workers"] == nil {
			t.Error("期望有Worker统计信息")
		}
	})
}

// TestStateTransitions 测试状态转换验证
func TestStateTransitions(t *testing.T) {
	tests := []struct {
		name     string
		from     SystemState
		to       SystemState
		expected bool
	}{
		{"Idle to Analyzing", SystemStateIdle, SystemStateAnalyzing, true},
		{"Analyzing to Planning", SystemStateAnalyzing, SystemStatePlanning, true},
		{"Planning to Executing", SystemStatePlanning, SystemStateExecuting, true},
		{"Executing to Evaluating", SystemStateExecuting, SystemStateEvaluating, true},
		{"Evaluating to Learning", SystemStateEvaluating, SystemStateLearning, true},
		{"Learning to Idle", SystemStateLearning, SystemStateIdle, true},
		{"Invalid transition", SystemStateIdle, SystemStateExecuting, false},
		{"Shutdown from Error", SystemStateError, SystemStateShuttingDown, true},
		{"Invalid from Shutdown", SystemStateShuttingDown, SystemStateIdle, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CanTransition(tt.from, tt.to)
			if result != tt.expected {
				t.Errorf("CanTransition(%v, %v) = %v，期望 %v",
					tt.from, tt.to, result, tt.expected)
			}
		})
	}
}

// TestTaskStateTransitions 测试任务状态转换验证
func TestTaskStateTransitions(t *testing.T) {
	tests := []struct {
		name     string
		from     TaskState
		to       TaskState
		expected bool
	}{
		{"Pending to Analyzing", TaskStatePending, TaskStateAnalyzing, true},
		{"Analyzing to Decomposing", TaskStateAnalyzing, TaskStateDecomposing, true},
		{"Running to Completed", TaskStateRunning, TaskStateCompleted, true},
		{"Running to Failed", TaskStateRunning, TaskStateFailed, true},
		{"Failed to Retrying", TaskStateFailed, TaskStateRetrying, true},
		{"Retrying to Running", TaskStateRetrying, TaskStateRunning, true},
		{"Invalid transition", TaskStatePending, TaskStateRunning, false},
		{"Completed to Running", TaskStateCompleted, TaskStateRunning, false},
		{"Cancelled to Running", TaskStateCancelled, TaskStateRunning, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CanTransitionTask(tt.from, tt.to)
			if result != tt.expected {
				t.Errorf("CanTransitionTask(%v, %v) = %v，期望 %v",
					tt.from, tt.to, result, tt.expected)
			}
		})
	}
}

// TestWorkerStateTransitions 测试Worker状态转换验证
func TestWorkerStateTransitions(t *testing.T) {
	tests := []struct {
		name     string
		from     WorkerState
		to       WorkerState
		expected bool
	}{
		{"Idle to Running", WorkerStateIdle, WorkerStateRunning, true},
		{"Running to Busy", WorkerStateRunning, WorkerStateBusy, true},
		{"Busy to Waiting", WorkerStateBusy, WorkerStateWaiting, true},
		{"Waiting to Running", WorkerStateWaiting, WorkerStateRunning, true},
		{"Error to Idle", WorkerStateError, WorkerStateIdle, true},
		{"Any to Terminating", WorkerStateRunning, WorkerStateTerminating, true},
		{"Terminating to Terminated", WorkerStateTerminating, WorkerStateTerminated, true},
		{"Invalid transition", WorkerStateIdle, WorkerStateBusy, false},
		{"Terminated to any", WorkerStateTerminated, WorkerStateIdle, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CanTransitionWorker(tt.from, tt.to)
			if result != tt.expected {
				t.Errorf("CanTransitionWorker(%v, %v) = %v，期望 %v",
					tt.from, tt.to, result, tt.expected)
			}
		})
	}
}

// TestConcurrentAccess 测试并发访问
func TestConcurrentAccess(t *testing.T) {
	sm := NewStateManager()

	// 创建多个goroutine并发访问状态管理器
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- true }()

			// 创建任务
			task := &TaskInfo{
				ID:       fmt.Sprintf("task-%d", id),
				Name:     fmt.Sprintf("并发测试任务-%d", id),
				State:    TaskStatePending,
				Priority: PriorityNormal,
				Metadata: make(map[string]any),
			}

			err := sm.CreateTask(task)
			if err != nil {
				t.Errorf("并发创建任务失败: %v", err)
				return
			}

			// 更新任务状态
			err = sm.UpdateTaskState(task.ID, TaskStateRunning, "并发测试")
			if err != nil {
				t.Errorf("并发更新任务状态失败: %v", err)
				return
			}

			// 获取任务
			_, err = sm.GetTask(task.ID)
			if err != nil {
				t.Errorf("并发获取任务失败: %v", err)
				return
			}
		}(i)
	}

	// 等待所有goroutine完成
	for i := 0; i < 10; i++ {
		<-done
	}

	// 验证创建了10个任务
	tasks := sm.ListTasks()
	if len(tasks) != 10 {
		t.Errorf("期望创建10个任务，实际创建了 %d 个", len(tasks))
	}
}

// TestSystemInfo 测试系统信息获取
func TestSystemInfo(t *testing.T) {
	sm := NewStateManager()

	// 创建一些测试数据
	task1 := &TaskInfo{
		ID:       "task-1",
		Name:     "任务1",
		State:    TaskStateRunning,
		Priority: PriorityNormal,
		Metadata: make(map[string]any),
	}
	task2 := &TaskInfo{
		ID:       "task-2",
		Name:     "任务2",
		State:    TaskStateCompleted,
		Priority: PriorityHigh,
		Metadata: make(map[string]any),
	}

	worker1 := &WorkerInfo{
		ID:          "worker-1",
		Type:        "web_ui",
		State:       WorkerStateRunning,
		Tools:       []string{"click"},
		Metadata:    make(map[string]any),
		Performance: PerformanceMetrics{},
	}

	sm.CreateTask(task1)
	sm.CreateTask(task2)
	sm.RegisterWorker(worker1)

	// 获取系统信息
	info := sm.GetSystemInfo()

	// 验证统计信息
	if info.ActiveTasks != 1 {
		t.Errorf("期望活跃任务数为1，实际为 %d", info.ActiveTasks)
	}

	if info.CompletedTasks != 1 {
		t.Errorf("期望完成任务数为1，实际为 %d", info.CompletedTasks)
	}

	if info.ActiveWorkers != 1 {
		t.Errorf("期望活跃Worker数为1，实际为 %d", info.ActiveWorkers)
	}

	if info.Uptime <= 0 {
		t.Error("系统运行时间应该大于0")
	}
}

// BenchmarkStateManager 性能基准测试
func BenchmarkStateManager(b *testing.B) {
	sm := NewStateManager()

	b.Run("CreateTask", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			task := &TaskInfo{
				ID:       fmt.Sprintf("task-%d", i),
				Name:     fmt.Sprintf("基准测试任务-%d", i),
				State:    TaskStatePending,
				Priority: PriorityNormal,
				Metadata: make(map[string]any),
			}
			sm.CreateTask(task)
		}
	})

	b.Run("GetTask", func(b *testing.B) {
		// 预创建一些任务
		for i := 0; i < 1000; i++ {
			task := &TaskInfo{
				ID:       fmt.Sprintf("bench-task-%d", i),
				Name:     fmt.Sprintf("基准测试任务-%d", i),
				State:    TaskStatePending,
				Priority: PriorityNormal,
				Metadata: make(map[string]any),
			}
			sm.CreateTask(task)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			sm.GetTask(fmt.Sprintf("bench-task-%d", i%1000))
		}
	})

	b.Run("UpdateTaskState", func(b *testing.B) {
		// 预创建一些任务
		for i := 0; i < 1000; i++ {
			task := &TaskInfo{
				ID:       fmt.Sprintf("update-task-%d", i),
				Name:     fmt.Sprintf("更新测试任务-%d", i),
				State:    TaskStatePending,
				Priority: PriorityNormal,
				Metadata: make(map[string]any),
			}
			sm.CreateTask(task)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			sm.UpdateTaskState(fmt.Sprintf("update-task-%d", i%1000), TaskStateRunning, "基准测试")
		}
	})
}
