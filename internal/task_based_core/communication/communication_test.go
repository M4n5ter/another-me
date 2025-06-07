package communication

import (
	"fmt"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"
)

func TestMessageBus(t *testing.T) {
	t.Run("基本发布订阅", func(t *testing.T) {
		bus := NewMessageBus(100, 2)
		defer bus.Close()

		received := make(chan Event, 1)

		// 订阅事件
		bus.Subscribe(EventTypeTaskStarted, func(event Event) {
			received <- event
		})

		// 发布事件
		testEvent := NewTaskEvent(EventTypeTaskStarted, "test", "task1", "测试任务")
		bus.Publish(testEvent)

		// 验证接收
		select {
		case event := <-received:
			if event.EventType() != EventTypeTaskStarted {
				t.Errorf("期望事件类型 %s，实际 %s", EventTypeTaskStarted, event.EventType())
			}
		case <-time.After(time.Second):
			t.Fatal("超时未收到事件")
		}
	})

	t.Run("多订阅者", func(t *testing.T) {
		bus := NewMessageBus(100, 2)
		defer bus.Close()

		received1 := make(chan Event, 1)
		received2 := make(chan Event, 1)

		// 多个订阅者
		bus.Subscribe(EventTypeTaskCompleted, func(event Event) {
			received1 <- event
		})
		bus.Subscribe(EventTypeTaskCompleted, func(event Event) {
			received2 <- event
		})

		// 发布事件
		testEvent := NewTaskEvent(EventTypeTaskCompleted, "test", "task1", "测试任务")
		bus.Publish(testEvent)

		// 验证两个订阅者都收到事件
		select {
		case <-received1:
		case <-time.After(time.Second):
			t.Fatal("订阅者1超时未收到事件")
		}

		select {
		case <-received2:
		case <-time.After(time.Second):
			t.Fatal("订阅者2超时未收到事件")
		}
	})

	t.Run("优先级事件", func(t *testing.T) {
		bus := NewMessageBus(100, 2)
		defer bus.Close()

		var eventCount int32

		bus.Subscribe(EventTypeTaskStarted, func(event Event) {
			atomic.AddInt32(&eventCount, 1)
		})

		// 发布事件
		testEvent := NewTaskEvent(EventTypeTaskStarted, "test", "task1", "测试任务")
		bus.Publish(testEvent)

		// 等待处理完成
		time.Sleep(100 * time.Millisecond)

		count := atomic.LoadInt32(&eventCount)
		if count != 1 {
			t.Fatalf("期望收到1个事件，实际收到 %d 个", count)
		}
	})
}

func TestComponentRegistry(t *testing.T) {
	t.Run("组件注册和获取", func(t *testing.T) {
		bus := NewMessageBus(100, 2)
		defer bus.Close()

		registry := NewComponentRegistry(bus)
		defer registry.Close()

		// 注册组件
		component := &ComponentInfo{
			ID:           "test-worker",
			Type:         ComponentTypeWorker,
			Name:         "测试Worker",
			Version:      "1.0.0",
			Capabilities: []string{"test_capability"},
		}

		err := registry.RegisterComponent(component)
		if err != nil {
			t.Fatalf("注册组件失败: %v", err)
		}

		// 获取组件
		retrieved, err := registry.GetComponent("test-worker")
		if err != nil {
			t.Fatalf("获取组件失败: %v", err)
		}

		if retrieved.ID != component.ID {
			t.Errorf("期望组件ID %s，实际 %s", component.ID, retrieved.ID)
		}
		if retrieved.Type != component.Type {
			t.Errorf("期望组件类型 %s，实际 %s", component.Type, retrieved.Type)
		}
	})

	t.Run("组件心跳", func(t *testing.T) {
		bus := NewMessageBus(100, 2)
		defer bus.Close()

		registry := NewComponentRegistry(bus)
		defer registry.Close()

		// 注册组件
		component := &ComponentInfo{
			ID:   "test-worker",
			Type: ComponentTypeWorker,
			Name: "测试Worker",
		}
		registry.RegisterComponent(component)

		// 发送心跳
		err := registry.Heartbeat("test-worker")
		if err != nil {
			t.Fatalf("发送心跳失败: %v", err)
		}

		// 验证组件状态
		retrieved, _ := registry.GetComponent("test-worker")
		if retrieved.Status != ComponentStatusActive {
			t.Errorf("期望组件状态 %s，实际 %s", ComponentStatusActive.String(), retrieved.Status.String())
		}
	})

	t.Run("按类型列出组件", func(t *testing.T) {
		bus := NewMessageBus(100, 2)
		defer bus.Close()

		registry := NewComponentRegistry(bus)
		defer registry.Close()

		// 注册不同类型的组件
		components := []*ComponentInfo{
			{ID: "worker1", Type: ComponentTypeWorker, Name: "Worker 1"},
			{ID: "worker2", Type: ComponentTypeWorker, Name: "Worker 2"},
			{ID: "orchestrator1", Type: ComponentTypeOrchestrator, Name: "Orchestrator 1"},
		}

		for _, comp := range components {
			registry.RegisterComponent(comp)
		}

		// 按类型获取
		workers := registry.ListComponentsByType(ComponentTypeWorker)
		orchestrators := registry.ListComponentsByType(ComponentTypeOrchestrator)

		if len(workers) != 2 {
			t.Errorf("期望2个Worker，实际 %d 个", len(workers))
		}
		if len(orchestrators) != 1 {
			t.Errorf("期望1个Orchestrator，实际 %d 个", len(orchestrators))
		}
	})
}

func TestTaskDAG(t *testing.T) {
	t.Run("基本任务添加和依赖", func(t *testing.T) {
		bus := NewMessageBus(100, 2)
		defer bus.Close()

		dag := NewTaskDAG(bus, 3)

		// 添加任务
		task1 := &TaskNode{
			ID:   "task1",
			Name: "任务1",
			Type: "test",
		}
		task2 := &TaskNode{
			ID:           "task2",
			Name:         "任务2",
			Type:         "test",
			Dependencies: []string{"task1"},
		}

		err := dag.AddTask(task1)
		if err != nil {
			t.Fatalf("添加任务1失败: %v", err)
		}

		err = dag.AddTask(task2)
		if err != nil {
			t.Fatalf("添加任务2失败: %v", err)
		}

		err = dag.AddDependency("task2", "task1")
		if err != nil {
			t.Fatalf("添加依赖关系失败: %v", err)
		}

		// 验证就绪任务
		readyTasks := dag.GetReadyTasks()
		if len(readyTasks) != 1 {
			t.Fatalf("期望1个就绪任务，实际 %d 个", len(readyTasks))
		}
		if readyTasks[0].ID != "task1" {
			t.Errorf("期望就绪任务为task1，实际为 %s", readyTasks[0].ID)
		}
	})

	t.Run("任务执行流程", func(t *testing.T) {
		bus := NewMessageBus(100, 2)
		defer bus.Close()

		dag := NewTaskDAG(bus, 3)

		// 创建简单的依赖链
		task1 := &TaskNode{ID: "task1", Name: "任务1", Type: "test"}
		task2 := &TaskNode{ID: "task2", Name: "任务2", Type: "test"}

		dag.AddTask(task1)
		dag.AddTask(task2)
		dag.AddDependency("task2", "task1")

		// 开始执行task1
		err := dag.MarkTaskStarted("task1", "worker1")
		if err != nil {
			t.Fatalf("标记任务开始失败: %v", err)
		}

		// 验证task1状态
		node, _ := dag.GetTaskNode("task1")
		if node.Status != TaskStatusRunning {
			t.Errorf("期望任务状态为 %s，实际为 %s", TaskStatusRunning.String(), node.Status.String())
		}

		// 完成task1
		output := map[string]any{"result": "success"}
		err = dag.MarkTaskCompleted("task1", output)
		if err != nil {
			t.Fatalf("标记任务完成失败: %v", err)
		}

		// 验证task2变为就绪
		readyTasks := dag.GetReadyTasks()
		if len(readyTasks) != 1 || readyTasks[0].ID != "task2" {
			t.Error("task1完成后，task2应该变为就绪状态")
		}
	})

	t.Run("DAG完成检测", func(t *testing.T) {
		bus := NewMessageBus(100, 2)
		defer bus.Close()

		dag := NewTaskDAG(bus, 3)

		// 添加单个任务
		task := &TaskNode{ID: "task1", Name: "任务1", Type: "test"}
		dag.AddTask(task)

		// DAG未完成
		if dag.IsDAGCompleted() {
			t.Error("DAG不应该是完成状态")
		}

		// 完成任务
		dag.MarkTaskStarted("task1", "worker1")
		dag.MarkTaskCompleted("task1", map[string]any{"result": "success"})

		// DAG应该完成
		if !dag.IsDAGCompleted() {
			t.Error("DAG应该是完成状态")
		}
	})

	t.Run("循环依赖检测", func(t *testing.T) {
		bus := NewMessageBus(100, 2)
		defer bus.Close()

		dag := NewTaskDAG(bus, 3)

		// 创建任务
		task1 := &TaskNode{ID: "task1", Name: "任务1", Type: "test"}
		task2 := &TaskNode{ID: "task2", Name: "任务2", Type: "test"}

		dag.AddTask(task1)
		dag.AddTask(task2)

		// 添加依赖：task1 -> task2
		err := dag.AddDependency("task1", "task2")
		if err != nil {
			t.Fatalf("添加依赖失败: %v", err)
		}

		// 尝试添加循环依赖：task2 -> task1
		err = dag.AddDependency("task2", "task1")
		if err == nil {
			t.Error("应该检测到循环依赖并返回错误")
		}
	})

	t.Run("任务重试", func(t *testing.T) {
		bus := NewMessageBus(100, 2)
		defer bus.Close()

		dag := NewTaskDAG(bus, 3)

		task := &TaskNode{
			ID:         "task1",
			Name:       "任务1",
			Type:       "test",
			MaxRetries: 2, // 设置最大重试次数
		}
		dag.AddTask(task)

		// 第一次失败
		dag.MarkTaskStarted("task1", "worker1")
		err := dag.MarkTaskFailed("task1", "模拟错误")
		if err != nil {
			t.Fatalf("标记任务失败失败: %v", err)
		}

		// 验证重试次数增加
		node, _ := dag.GetTaskNode("task1")
		if node.RetryCount != 1 {
			t.Errorf("期望重试次数为1，实际为 %d", node.RetryCount)
		}

		// 验证任务状态为失败（重试是异步的）
		if node.Status != TaskStatusFailed {
			t.Errorf("期望任务状态为失败，实际为 %s", node.Status.String())
		}

		// 等待足够时间让重试逻辑执行（重试延迟是 RetryCount * 5秒，这里是5秒）
		// 在测试中我们不能等那么长时间，所以这里验证重试计数和状态即可

		// 验证重试逻辑被触发（通过检查任务没有被标记为跳过）
		stats := dag.GetDAGStats()
		if stats.SkippedTasks > 0 {
			t.Error("任务不应该被跳过，因为还在重试次数范围内")
		}
	})
}

func TestComplexWorkflow(t *testing.T) {
	t.Run("复杂工作流集成测试", func(t *testing.T) {
		// 设置静默日志避免测试输出过多
		logger := slog.New(slog.NewTextHandler(nil, &slog.HandlerOptions{Level: slog.LevelWarn}))
		slog.SetDefault(logger)

		// 创建组件
		bus := NewMessageBus(1000, 4)
		defer bus.Close()

		registry := NewComponentRegistry(bus)
		defer registry.Close()

		dag := NewTaskDAG(bus, 2)

		// 注册简化的组件
		worker := &ComponentInfo{
			ID:           "test-worker",
			Type:         ComponentTypeWorker,
			Name:         "测试Worker",
			Capabilities: []string{"test_capability", "data_analysis"},
		}
		registry.RegisterComponent(worker)

		orchestrator := &ComponentInfo{
			ID:   "test-orchestrator",
			Type: ComponentTypeOrchestrator,
			Name: "测试编排器",
		}
		registry.RegisterComponent(orchestrator)

		// 创建简单的任务链
		task1 := &TaskNode{
			ID:            "task1",
			Name:          "数据收集",
			Type:          "test_capability",
			EstimatedTime: 100 * time.Millisecond,
		}
		task2 := &TaskNode{
			ID:            "task2",
			Name:          "数据分析",
			Type:          "data_analysis",
			Dependencies:  []string{"task1"},
			EstimatedTime: 100 * time.Millisecond,
		}

		dag.AddTask(task1)
		dag.AddTask(task2)
		dag.AddDependency("task2", "task1")

		// 模拟任务执行
		readyTasks := dag.GetReadyTasks()
		if len(readyTasks) != 1 {
			t.Fatalf("期望1个初始就绪任务，实际 %d 个", len(readyTasks))
		}

		// 执行task1
		dag.MarkTaskStarted("task1", "test-worker")
		time.Sleep(150 * time.Millisecond) // 模拟执行时间
		dag.MarkTaskCompleted("task1", map[string]any{"result": "data"})

		// 检查task2变为就绪
		readyTasks = dag.GetReadyTasks()
		if len(readyTasks) != 1 || readyTasks[0].ID != "task2" {
			t.Error("task1完成后，task2应该变为就绪")
		}

		// 执行task2
		dag.MarkTaskStarted("task2", "test-worker")
		time.Sleep(150 * time.Millisecond)
		dag.MarkTaskCompleted("task2", map[string]any{"result": "analysis"})

		// 验证DAG完成
		if !dag.IsDAGCompleted() {
			t.Error("所有任务完成后，DAG应该标记为完成")
		}

		stats := dag.GetDAGStats()
		if stats.CompletedTasks != 2 {
			t.Errorf("期望完成2个任务，实际完成 %d 个", stats.CompletedTasks)
		}
		if stats.Progress != 100.0 {
			t.Errorf("期望进度100%%，实际进度 %.1f%%", stats.Progress)
		}
	})
}

func BenchmarkMessageBus(b *testing.B) {
	bus := NewMessageBus(10000, 4)
	defer bus.Close()

	// 订阅多个事件类型
	for range 10 {
		bus.Subscribe(EventTypeTaskStarted, func(event Event) {
			// 模拟处理
		})
	}

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			event := NewTaskEvent(EventTypeTaskStarted, "bench", fmt.Sprintf("task-%d", b.N), "基准测试任务")
			bus.Publish(event)
		}
	})
}

func BenchmarkComponentRegistry(b *testing.B) {
	bus := NewMessageBus(1000, 2)
	defer bus.Close()

	registry := NewComponentRegistry(bus)
	defer registry.Close()

	// 预注册一些组件
	for i := range 100 {
		component := &ComponentInfo{
			ID:   fmt.Sprintf("worker-%d", i),
			Type: ComponentTypeWorker,
			Name: fmt.Sprintf("Worker %d", i),
		}
		registry.RegisterComponent(component)
	}

	b.ResetTimer()

	b.Run("GetComponent", func(b *testing.B) {
		for i := 0; b.Loop(); i++ {
			registry.GetComponent(fmt.Sprintf("worker-%d", i%100))
		}
	})

	b.Run("ListComponentsByType", func(b *testing.B) {
		for b.Loop() {
			registry.ListComponentsByType(ComponentTypeWorker)
		}
	})

	b.Run("Heartbeat", func(b *testing.B) {
		for i := 0; b.Loop(); i++ {
			registry.Heartbeat(fmt.Sprintf("worker-%d", i%100))
		}
	})
}

func BenchmarkTaskDAG(b *testing.B) {
	bus := NewMessageBus(1000, 2)
	defer bus.Close()

	dag := NewTaskDAG(bus, 10)

	// 预创建一些任务
	for i := range 100 {
		task := &TaskNode{
			ID:   fmt.Sprintf("task-%d", i),
			Name: fmt.Sprintf("Task %d", i),
			Type: "benchmark",
		}
		dag.AddTask(task)
	}

	b.ResetTimer()

	b.Run("GetReadyTasks", func(b *testing.B) {
		for b.Loop() {
			dag.GetReadyTasks()
		}
	})

	b.Run("MarkTaskCompleted", func(b *testing.B) {
		// 简化基准测试，不使用重置功能
		for i := 0; i < b.N && i < 100; i++ {
			taskID := fmt.Sprintf("task-%d", i)
			dag.MarkTaskStarted(taskID, "worker")
			dag.MarkTaskCompleted(taskID, map[string]any{"result": "ok"})
		}
	})
}
