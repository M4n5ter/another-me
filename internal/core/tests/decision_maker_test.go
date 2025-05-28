package tests

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/m4n5ter/another-me/internal/core"
)

// MockMindscapeService 模拟Mindscape服务
type MockMindscapeService struct {
	mock.Mock
}

func (m *MockMindscapeService) StoreMemory(ctx context.Context, memoryData core.MemoryItem) error {
	args := m.Called(ctx, memoryData)
	return args.Error(0)
}

func (m *MockMindscapeService) RetrieveMemories(ctx context.Context, queryContext map[string]any) ([]core.MemoryItem, error) {
	args := m.Called(ctx, queryContext)
	return args.Get(0).([]core.MemoryItem), args.Error(1)
}

func (m *MockMindscapeService) DelegateMonitoringTask(ctx context.Context, taskDetails core.MonitoringTask) (string, error) {
	args := m.Called(ctx, taskDetails)
	return args.String(0), args.Error(1)
}

func (m *MockMindscapeService) ClearOrUpdateMonitoringTasks(ctx context.Context, taskUpdate core.TaskUpdate) error {
	args := m.Called(ctx, taskUpdate)
	return args.Error(0)
}

func (m *MockMindscapeService) SetupWakeUpListener(handler func(core.WakeupEvent) error) error {
	args := m.Called(handler)
	return args.Error(0)
}

func (m *MockMindscapeService) IsHealthy(ctx context.Context) bool {
	args := m.Called(ctx)
	return args.Bool(0)
}

func (m *MockMindscapeService) GetUserProfile(ctx context.Context, userID string) (*core.MemoryItem, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(*core.MemoryItem), args.Error(1)
}

func (m *MockMindscapeService) UpdateUserProfile(ctx context.Context, userID string, profileData map[string]any) error {
	args := m.Called(ctx, userID, profileData)
	return args.Error(0)
}

func TestDefaultDecisionMakerConfig(t *testing.T) {
	config := core.DefaultDecisionMakerConfig()

	assert.Equal(t, 20, config.MemoryQueryLimit)
	assert.Equal(t, 0.6, config.MemoryRelevanceThreshold)
	assert.Equal(t, core.AgentTypeReAct, config.DefaultAgent)
	assert.Equal(t, 1.0, config.GUIKeywordWeight)
	assert.Equal(t, 1.0, config.ReActKeywordWeight)
	assert.Contains(t, config.MonitoringKeywords, "监控")
	assert.Equal(t, 24*time.Hour, config.DefaultMonitoringTTL)
	assert.Equal(t, 10, config.MaxMonitoringTasks)
}

func TestNewSmartDecisionMaker(t *testing.T) {
	mockMindscape := &MockMindscapeService{}
	config := core.DefaultDecisionMakerConfig()

	decisionMaker := core.NewSmartDecisionMaker(mockMindscape, config, nil)

	assert.NotNil(t, decisionMaker)
}

func TestAnalyzeUserInput_GUITask(t *testing.T) {
	mockMindscape := &MockMindscapeService{}
	config := core.DefaultDecisionMakerConfig()
	decisionMaker := core.NewSmartDecisionMaker(mockMindscape, config, nil)

	// 模拟返回空记忆
	mockMindscape.On("RetrieveMemories", mock.Anything, mock.Anything).Return([]core.MemoryItem{}, nil)

	ctx := context.Background()
	decisionCtx := core.DecisionContext{
		SystemState: map[string]any{
			"user_input": "请点击屏幕上的登录按钮",
		},
		RetrievedMemories: []core.MemoryItem{},
		Timestamp:        time.Now(),
	}

	result, err := decisionMaker.AnalyzeUserInput(ctx, decisionCtx)

	assert.NoError(t, err)
	assert.True(t, result.ShouldExecuteTask)
	assert.True(t, result.Task.IsSome())
	
	task := result.Task.Unwrap()
	assert.Equal(t, core.AgentTypeGUI, task.AgentType)
	assert.Contains(t, task.Type, "gui")
	assert.Contains(t, task.Description, "点击")
	
	mockMindscape.AssertExpectations(t)
}

func TestAnalyzeUserInput_ReActTask(t *testing.T) {
	mockMindscape := &MockMindscapeService{}
	config := core.DefaultDecisionMakerConfig()
	decisionMaker := core.NewSmartDecisionMaker(mockMindscape, config, nil)

	// 模拟返回空记忆
	mockMindscape.On("RetrieveMemories", mock.Anything, mock.Anything).Return([]core.MemoryItem{}, nil)

	ctx := context.Background()
	decisionCtx := core.DecisionContext{
		SystemState: map[string]any{
			"user_input": "请搜索今天的天气信息",
		},
		RetrievedMemories: []core.MemoryItem{},
		Timestamp:        time.Now(),
	}

	result, err := decisionMaker.AnalyzeUserInput(ctx, decisionCtx)

	assert.NoError(t, err)
	assert.True(t, result.ShouldExecuteTask)
	assert.True(t, result.Task.IsSome())
	
	task := result.Task.Unwrap()
	assert.Equal(t, core.AgentTypeReAct, task.AgentType)
	assert.Contains(t, task.Type, "react")
	assert.Contains(t, task.Description, "搜索")
	
	mockMindscape.AssertExpectations(t)
}

func TestAnalyzeUserInput_MonitoringTask(t *testing.T) {
	mockMindscape := &MockMindscapeService{}
	config := core.DefaultDecisionMakerConfig()
	decisionMaker := core.NewSmartDecisionMaker(mockMindscape, config, nil)

	// 模拟返回空记忆
	mockMindscape.On("RetrieveMemories", mock.Anything, mock.Anything).Return([]core.MemoryItem{}, nil)

	ctx := context.Background()
	decisionCtx := core.DecisionContext{
		SystemState: map[string]any{
			"user_input": "请监控网站的状态变化",
		},
		RetrievedMemories: []core.MemoryItem{},
		Timestamp:        time.Now(),
	}

	result, err := decisionMaker.AnalyzeUserInput(ctx, decisionCtx)

	assert.NoError(t, err)
	assert.True(t, result.ShouldExecuteTask)
	assert.NotEmpty(t, result.MonitoringTasks)
	assert.Contains(t, result.ReasoningSteps[0], "监控需求")
	
	monitoringTask := result.MonitoringTasks[0]
	assert.Contains(t, monitoringTask.Description, "监控")
	assert.True(t, monitoringTask.IsEnabled)
	
	mockMindscape.AssertExpectations(t)
}

func TestSelectAgent_WithAvailableAgents(t *testing.T) {
	mockMindscape := &MockMindscapeService{}
	config := core.DefaultDecisionMakerConfig()
	decisionMaker := core.NewSmartDecisionMaker(mockMindscape, config, nil)

	ctx := context.Background()
	task := core.Task{
		ID:          "test-task-001",
		Type:        "gui_click",
		Description: "点击按钮",
		AgentType:   core.AgentTypeGUI,
		Parameters:  map[string]any{},
		CreatedAt:   time.Now(),
	}

	availableAgents := []core.AgentType{core.AgentTypeGUI, core.AgentTypeReAct}

	selectedAgent, err := decisionMaker.SelectAgent(ctx, task, availableAgents)

	assert.NoError(t, err)
	assert.Equal(t, core.AgentTypeGUI, selectedAgent)
}

func TestSelectAgent_NoAvailableAgents(t *testing.T) {
	mockMindscape := &MockMindscapeService{}
	config := core.DefaultDecisionMakerConfig()
	decisionMaker := core.NewSmartDecisionMaker(mockMindscape, config, nil)

	ctx := context.Background()
	task := core.Task{
		ID:          "test-task-001",
		Type:        "gui_click",
		Description: "点击按钮",
		AgentType:   core.AgentTypeGUI,
		Parameters:  map[string]any{},
		CreatedAt:   time.Now(),
	}

	availableAgents := []core.AgentType{}

	selectedAgent, err := decisionMaker.SelectAgent(ctx, task, availableAgents)

	assert.Error(t, err)
	assert.Equal(t, core.AgentTypeUnknown, selectedAgent)
	assert.Contains(t, err.Error(), "没有可用的Agent")
}

func TestDefineMonitoringConditions(t *testing.T) {
	mockMindscape := &MockMindscapeService{}
	config := core.DefaultDecisionMakerConfig()
	decisionMaker := core.NewSmartDecisionMaker(mockMindscape, config, nil)

	ctx := context.Background()
	userInput := "每5分钟检查网站状态变化"
	context := map[string]any{
		"target_url": "https://example.com",
	}

	conditions, err := decisionMaker.DefineMonitoringConditions(ctx, userInput, context)

	assert.NoError(t, err)
	assert.NotEmpty(t, conditions)
	
	// 检查时间条件
	hasTimeCondition := false
	hasChangeCondition := false
	
	for _, condition := range conditions {
		if condition.Type == "time_interval" {
			hasTimeCondition = true
			assert.Contains(t, condition.Value, "5")
		}
		if condition.Type == "change_detection" {
			hasChangeCondition = true
		}
	}
	
	assert.True(t, hasTimeCondition, "应该有时间条件")
	assert.True(t, hasChangeCondition, "应该有变化检测条件")
}

func TestMakeDecisionBasedOnMemory(t *testing.T) {
	mockMindscape := &MockMindscapeService{}
	config := core.DefaultDecisionMakerConfig()
	decisionMaker := core.NewSmartDecisionMaker(mockMindscape, config, nil)

	// 模拟返回空记忆（因为会再次调用RetrieveMemories）
	mockMindscape.On("RetrieveMemories", mock.Anything, mock.Anything).Return([]core.MemoryItem{}, nil)

	ctx := context.Background()
	memories := []core.MemoryItem{
		{
			ID:        "memory-001",
			Timestamp: time.Now(),
			Type:      core.MemoryTypeTaskSummary,
			Content:   "成功执行GUI点击任务",
			Keywords:  []string{"GUI", "点击", "成功"},
			Importance: 0.8,
			UserID:    "user-001",
			Metadata: map[string]any{
				"agent_type": "gui",
				"success":    true,
				"task_type":  "gui_click",
			},
		},
	}

	currentContext := core.DecisionContext{
		SystemState: map[string]any{
			"user_input": "点击登录按钮",
		},
		RetrievedMemories: []core.MemoryItem{},
		Timestamp:        time.Now(),
	}

	result, err := decisionMaker.MakeDecisionBasedOnMemory(ctx, memories, currentContext)

	assert.NoError(t, err)
	assert.True(t, result.ShouldExecuteTask)
	assert.True(t, result.Task.IsSome())
	
	task := result.Task.Unwrap()
	assert.Equal(t, core.AgentTypeGUI, task.AgentType)
	
	mockMindscape.AssertExpectations(t)
}

func TestHandleWakeupEvent(t *testing.T) {
	mockMindscape := &MockMindscapeService{}
	config := core.DefaultDecisionMakerConfig()
	decisionMaker := core.NewSmartDecisionMaker(mockMindscape, config, nil)

	// 模拟返回相关记忆
	mockMemories := []core.MemoryItem{
		{
			ID:        "memory-001",
			Timestamp: time.Now(),
			Type:      core.MemoryTypeObservation,
			Content:   "网站状态检查",
			Keywords:  []string{"网站", "监控"},
			Importance: 0.9,
			UserID:    "user-001",
			Metadata:  map[string]any{"source": "monitoring"},
		},
	}
	mockMindscape.On("RetrieveMemories", mock.Anything, mock.Anything).Return(mockMemories, nil)

	ctx := context.Background()
	wakeupEvent := core.WakeupEvent{
		MonitoringTaskID: "task-monitoring-001",
		TriggerTime:      time.Now(),
		ObservedData: map[string]any{
			"url":    "https://example.com",
			"status": 500,
		},
		Reason:   "网站状态异常",
		Metadata: map[string]any{"alert_level": "high"},
	}

	result, err := decisionMaker.HandleWakeupEvent(ctx, wakeupEvent)

	assert.NoError(t, err)
	assert.True(t, result.ShouldExecuteTask)
	assert.True(t, result.Task.IsSome())
	
	task := result.Task.Unwrap()
	assert.Contains(t, task.Description, "监控任务触发")
	assert.Contains(t, task.Description, "网站状态异常")
	
	mockMindscape.AssertExpectations(t)
}

func TestDecisionMakerWithHistoricalPreference(t *testing.T) {
	mockMindscape := &MockMindscapeService{}
	config := core.DefaultDecisionMakerConfig()
	decisionMaker := core.NewSmartDecisionMaker(mockMindscape, config, nil)

	// 模拟返回有GUI偏好的历史记忆
	historicalMemories := []core.MemoryItem{
		{
			ID:        "memory-001",
			Timestamp: time.Now(),
			Type:      core.MemoryTypeTaskSummary,
			Content:   "GUI任务执行成功",
			Metadata: map[string]any{
				"agent_type": "gui",
				"success":    true,
			},
		},
		{
			ID:        "memory-002",
			Timestamp: time.Now(),
			Type:      core.MemoryTypeTaskSummary,
			Content:   "GUI任务执行成功",
			Metadata: map[string]any{
				"agent_type": "gui",
				"success":    true,
			},
		},
	}
	mockMindscape.On("RetrieveMemories", mock.Anything, mock.Anything).Return(historicalMemories, nil)

	ctx := context.Background()
	decisionCtx := core.DecisionContext{
		SystemState: map[string]any{
			"user_input": "执行一个通用任务", // 模糊指令，应该基于历史偏好选择GUI
		},
		RetrievedMemories: []core.MemoryItem{},
		Timestamp:        time.Now(),
	}

	result, err := decisionMaker.AnalyzeUserInput(ctx, decisionCtx)

	assert.NoError(t, err)
	assert.True(t, result.ShouldExecuteTask)
	assert.True(t, result.Task.IsSome())
	
	task := result.Task.Unwrap()
	// 由于历史偏好GUI，应该倾向于选择GUI Agent
	assert.Equal(t, core.AgentTypeGUI, task.AgentType)
	
	mockMindscape.AssertExpectations(t)
} 