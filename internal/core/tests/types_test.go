package tests

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/m4n5ter/another-me/internal/core/types"
	. "github.com/m4n5ter/another-me/pkg/option"
)

func TestAgentType(t *testing.T) {
	assert.Equal(t, types.AgentType("gui"), types.AgentTypeGUI)
	assert.Equal(t, types.AgentType("react"), types.AgentTypeReAct)
	assert.Equal(t, types.AgentType("unknown"), types.AgentTypeUnknown)
}

func TestTask(t *testing.T) {
	now := time.Now()
	task := types.Task{
		ID:          "test-task-001",
		Type:        "gui_click",
		Description: "点击指定按钮",
		AgentType:   types.AgentTypeGUI,
		Parameters: map[string]any{
			"coordinates": []int{100, 200},
			"button":      "left",
		},
		Priority:  1,
		CreatedAt: now,
		Context: map[string]any{
			"screenshot_url": "data:image/png;base64,test123",
		},
	}

	assert.Equal(t, "test-task-001", task.ID)
	assert.Equal(t, "gui_click", task.Type)
	assert.Equal(t, types.AgentTypeGUI, task.AgentType)
	assert.Equal(t, 1, task.Priority)
	assert.Equal(t, now, task.CreatedAt)
	assert.Contains(t, task.Parameters, "coordinates")
	assert.Contains(t, task.Context, "screenshot_url")
}

func TestExecutionResult(t *testing.T) {
	startTime := time.Now()
	endTime := startTime.Add(5 * time.Second)

	result := types.ExecutionResult{
		TaskID:       "test-task-001",
		Status:       types.ExecutionStatusSuccess,
		Output:       "操作已成功完成",
		Observations: []string{"检测到按钮", "执行点击操作", "确认操作成功"},
		Error:        "",
		StartTime:    startTime,
		EndTime:      endTime,
		Metadata: map[string]any{
			"execution_time_ms": 5000,
			"agent_type":        "gui",
		},
	}

	assert.Equal(t, "test-task-001", result.TaskID)
	assert.Equal(t, types.ExecutionStatusSuccess, result.Status)
	assert.Equal(t, "操作已成功完成", result.Output)
	assert.Len(t, result.Observations, 3)
	assert.Empty(t, result.Error)
	assert.Equal(t, startTime, result.StartTime)
	assert.Equal(t, endTime, result.EndTime)
	assert.Equal(t, 5000, result.Metadata["execution_time_ms"])
}

func TestExecutionStatus(t *testing.T) {
	assert.Equal(t, types.ExecutionStatus("success"), types.ExecutionStatusSuccess)
	assert.Equal(t, types.ExecutionStatus("failure"), types.ExecutionStatusFailure)
	assert.Equal(t, types.ExecutionStatus("in_progress"), types.ExecutionStatusInProgress)
	assert.Equal(t, types.ExecutionStatus("cancelled"), types.ExecutionStatusCancelled)
}

func TestMonitoringTask(t *testing.T) {
	now := time.Now()
	monitoringTask := types.MonitoringTask{
		ID:                None[string](), // 初始创建时为None，由Mindscape分配
		Description:       "监控VSCode启动",
		MindscapeTaskType: "application_monitor",
		Conditions: []types.MonitorCondition{
			{
				Type:     "application_start",
				Property: "application_name",
				Operator: "equals",
				Value:    "Code",
			},
		},
		TargetData:          []string{"window_title", "process_id"},
		NotificationMethods: []string{"webhook"},
		WebhookURL:          Some("http://localhost:8080/wakeup"),
		MQTopic:             None[string](),
		MaxRetries:          Some(3),
		IsEnabled:           true,
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	assert.True(t, monitoringTask.ID.IsNone())
	assert.Equal(t, "监控VSCode启动", monitoringTask.Description)
	assert.Equal(t, "application_monitor", monitoringTask.MindscapeTaskType)
	assert.Len(t, monitoringTask.Conditions, 1)
	assert.Equal(t, "application_start", monitoringTask.Conditions[0].Type)
	assert.Contains(t, monitoringTask.TargetData, "window_title")
	assert.True(t, monitoringTask.WebhookURL.IsSome())
	assert.Equal(t, "http://localhost:8080/wakeup", monitoringTask.WebhookURL.Unwrap())
	assert.True(t, monitoringTask.IsEnabled)
}

func TestMonitorCondition(t *testing.T) {
	condition := types.MonitorCondition{
		Type:     "text_on_screen",
		Property: "text_pattern",
		Operator: "contains",
		Value:    "Error:",
	}

	assert.Equal(t, "text_on_screen", condition.Type)
	assert.Equal(t, "text_pattern", condition.Property)
	assert.Equal(t, "contains", condition.Operator)
	assert.Equal(t, "Error:", condition.Value)
}

func TestTaskUpdate(t *testing.T) {
	taskUpdate := types.TaskUpdate{
		TasksToUpdate: []types.MonitoringTask{
			{
				ID:          Some("task-001"),
				Description: "更新的监控任务",
				IsEnabled:   false,
			},
		},
		TaskIDsToDelete: []string{"task-002", "task-003"},
	}

	assert.Len(t, taskUpdate.TasksToUpdate, 1)
	assert.Len(t, taskUpdate.TaskIDsToDelete, 2)
	assert.Equal(t, "task-001", taskUpdate.TasksToUpdate[0].ID.Unwrap())
	assert.False(t, taskUpdate.TasksToUpdate[0].IsEnabled)
	assert.Contains(t, taskUpdate.TaskIDsToDelete, "task-002")
}

func TestWakeupEvent(t *testing.T) {
	triggerTime := time.Now()
	wakeupEvent := types.WakeupEvent{
		MonitoringTaskID: "task-001",
		TriggerTime:      triggerTime,
		ObservedData: map[string]any{
			"application_name": "Code",
			"window_title":     "another-me - Visual Studio Code",
			"process_id":       12345,
		},
		Reason: "VSCode启动检测到",
		Metadata: map[string]any{
			"detector_version": "1.0.0",
			"confidence":       0.95,
		},
	}

	assert.Equal(t, "task-001", wakeupEvent.MonitoringTaskID)
	assert.Equal(t, triggerTime, wakeupEvent.TriggerTime)
	assert.Equal(t, "Code", wakeupEvent.ObservedData["application_name"])
	assert.Equal(t, "VSCode启动检测到", wakeupEvent.Reason)
	assert.Equal(t, 0.95, wakeupEvent.Metadata["confidence"])
}

func TestMemoryItem(t *testing.T) {
	timestamp := time.Now()
	memoryItem := types.MemoryItem{
		ID:         "memory-001",
		Timestamp:  timestamp,
		Type:       types.MemoryTypeObservation,
		Content:    "用户打开了VSCode，开始编程工作",
		Keywords:   []string{"vscode", "编程", "工作"},
		Importance: 0.8,
		RelatedIDs: []string{"memory-002", "memory-003"},
		UserID:     "user-001",
		Metadata: map[string]any{
			"source":     "gui_agent",
			"confidence": 0.9,
		},
	}

	assert.Equal(t, "memory-001", memoryItem.ID)
	assert.Equal(t, timestamp, memoryItem.Timestamp)
	assert.Equal(t, types.MemoryTypeObservation, memoryItem.Type)
	assert.Contains(t, memoryItem.Keywords, "vscode")
	assert.Equal(t, 0.8, memoryItem.Importance)
	assert.Len(t, memoryItem.RelatedIDs, 2)
	assert.Equal(t, "user-001", memoryItem.UserID)
}

func TestMemoryTypeConstants(t *testing.T) {
	assert.Equal(t, "observation", types.MemoryTypeObservation)
	assert.Equal(t, "user_preference", types.MemoryTypeUserPref)
	assert.Equal(t, "task_summary", types.MemoryTypeTaskSummary)
	assert.Equal(t, "error_log", types.MemoryTypeErrorLog)
	assert.Equal(t, "user_profile", types.MemoryTypeUserProfile)
	assert.Equal(t, "system_event", types.MemoryTypeSystemEvent)
}

func TestDecisionContext(t *testing.T) {
	timestamp := time.Now()
	wakeupEvent := types.WakeupEvent{
		MonitoringTaskID: "task-001",
		TriggerTime:      timestamp,
		Reason:           "测试唤醒",
	}

	decisionContext := types.DecisionContext{
		WakeupEvent:         Some(wakeupEvent),
		RetrievedMemories:   []types.MemoryItem{},
		SystemState:         map[string]any{"active": true},
		LastExecutionResult: None[types.ExecutionResult](),
		Timestamp:           timestamp,
	}

	assert.True(t, decisionContext.WakeupEvent.IsSome())
	assert.Equal(t, "task-001", decisionContext.WakeupEvent.Unwrap().MonitoringTaskID)
	assert.Empty(t, decisionContext.RetrievedMemories)
	assert.True(t, decisionContext.SystemState["active"].(bool))
	assert.True(t, decisionContext.LastExecutionResult.IsNone())
}

func TestDecisionResult(t *testing.T) {
	task := types.Task{
		ID:          "task-001",
		Description: "测试任务",
		AgentType:   types.AgentTypeGUI,
	}

	decisionResult := types.DecisionResult{
		ShouldExecuteTask:   true,
		Task:                Some(task),
		MonitoringTasks:     []types.MonitoringTask{},
		ShouldEnterWaitMode: false,
		ReasoningSteps:      []string{"分析唤醒事件", "确定需要执行任务", "选择GUI Agent"},
		Confidence:          0.85,
		ExpectedDuration:    Some(30 * time.Second),
	}

	assert.True(t, decisionResult.ShouldExecuteTask)
	assert.True(t, decisionResult.Task.IsSome())
	assert.Equal(t, "task-001", decisionResult.Task.Unwrap().ID)
	assert.False(t, decisionResult.ShouldEnterWaitMode)
	assert.Len(t, decisionResult.ReasoningSteps, 3)
	assert.Equal(t, 0.85, decisionResult.Confidence)
	assert.Equal(t, 30*time.Second, decisionResult.ExpectedDuration.Unwrap())
}

func TestSystemState(t *testing.T) {
	startTime := time.Now()
	lastActivity := startTime.Add(10 * time.Minute)

	systemState := types.SystemState{
		IsActive:            true,
		IsWaitingMode:       false,
		CurrentTask:         None[types.Task](),
		ActiveMonitoringIDs: []string{"monitor-001", "monitor-002"},
		LastActivity:        lastActivity,
		StartTime:           startTime,
		ExecutionHistory:    []types.ExecutionResult{},
		ErrorCount:          0,
		Metadata: map[string]any{
			"version": "1.0.0",
			"uptime":  10 * time.Minute,
		},
	}

	assert.True(t, systemState.IsActive)
	assert.False(t, systemState.IsWaitingMode)
	assert.True(t, systemState.CurrentTask.IsNone())
	assert.Len(t, systemState.ActiveMonitoringIDs, 2)
	assert.Equal(t, lastActivity, systemState.LastActivity)
	assert.Equal(t, 0, systemState.ErrorCount)
	assert.Equal(t, "1.0.0", systemState.Metadata["version"])
}
