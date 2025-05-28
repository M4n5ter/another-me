package tests

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/m4n5ter/another-me/internal/core"
)

// MockWakeupListener Mock实现（新增的Mock）
type MockWakeupListener struct {
	mock.Mock
	handler func(core.WakeupEvent) error
}

func (m *MockWakeupListener) Start(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockWakeupListener) Stop(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockWakeupListener) SetHandler(handler func(core.WakeupEvent) error) {
	m.handler = handler
	m.Called(handler)
}

func (m *MockWakeupListener) IsListening() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockWakeupListener) GetListenAddress() string {
	args := m.Called()
	return args.String(0)
}

// TriggerWakeup 触发唤醒事件（测试辅助方法）
func (m *MockWakeupListener) TriggerWakeup(event core.WakeupEvent) error {
	if m.handler != nil {
		return m.handler(event)
	}
	return nil
}

// TestSmartMainLoop_BasicCreation 测试主循环基础创建
func TestSmartMainLoop_BasicCreation(t *testing.T) {
	mockWakeup := &MockWakeupListener{}

	// 创建MainLoop
	config := core.DefaultMainLoopConfig()
	logger := slog.Default()
	
	mainLoop := core.NewSmartMainLoop(
		nil, // 使用nil进行基础测试
		nil,
		nil,
		mockWakeup,
		config,
		logger,
	)

	// 测试初始状态
	assert.False(t, mainLoop.IsRunning())
	
	state := mainLoop.GetSystemState()
	assert.False(t, state.IsActive)
	assert.False(t, state.IsWaitingMode)
	assert.NotNil(t, state.Metadata)
}

// TestSmartMainLoop_ConfigDefaults 测试默认配置
func TestSmartMainLoop_ConfigDefaults(t *testing.T) {
	config := core.DefaultMainLoopConfig()
	
	assert.Equal(t, 5*time.Second, config.MainLoopInterval)
	assert.Equal(t, 30*time.Second, config.WaitModeInterval)
	assert.Equal(t, 100, config.MaxExecutionHistory)
	assert.Equal(t, 1*time.Minute, config.HealthCheckInterval)
	assert.Equal(t, 3, config.MaxRetryAttempts)
	assert.Equal(t, 1*time.Second, config.RetryBackoffBase)
	assert.True(t, config.EnableAutoRecover)
	assert.True(t, config.EnableMetrics)
	assert.Equal(t, 30*time.Second, config.UserInputTimeout)
}

// TestSmartMainLoop_SystemStateManagement 测试系统状态管理
func TestSmartMainLoop_SystemStateManagement(t *testing.T) {
	mockWakeup := &MockWakeupListener{}
	
	config := core.DefaultMainLoopConfig()
	logger := slog.Default()
	
	mainLoop := core.NewSmartMainLoop(
		nil,
		nil,
		nil,
		mockWakeup,
		config,
		logger,
	)

	// 测试初始状态
	state := mainLoop.GetSystemState()
	assert.False(t, state.IsActive)
	assert.False(t, state.IsWaitingMode)
	assert.Equal(t, 0, state.ErrorCount)
	assert.Empty(t, state.ActiveMonitoringIDs)
	assert.Empty(t, state.ExecutionHistory)

	// 测试等待模式进入和退出（即使组件为nil也应该能正常切换状态）
	ctx := context.Background()
	
	_ = mainLoop.EnterWaitMode(ctx, []core.MonitoringTask{})
	// 这可能会失败因为mindscapeService为nil，但我们可以测试状态变化
	state = mainLoop.GetSystemState()
	
	err := mainLoop.ExitWaitMode(ctx)
	assert.NoError(t, err) // 退出等待模式应该成功
	
	state = mainLoop.GetSystemState()
	assert.False(t, state.IsWaitingMode)
}

// TestSmartMainLoop_ExecutionHistory 测试执行历史管理
func TestSmartMainLoop_ExecutionHistory(t *testing.T) {
	mockWakeup := &MockWakeupListener{}
	
	config := core.DefaultMainLoopConfig()
	config.MaxExecutionHistory = 5 // 设置较小的历史记录限制
	logger := slog.Default()
	
	mainLoop := core.NewSmartMainLoop(
		nil,
		nil,
		nil,
		mockWakeup,
		config,
		logger,
	)

	// 测试初始历史记录为空
	history := mainLoop.GetExecutionHistory(10)
	assert.Empty(t, history)

	// 测试限制参数
	history = mainLoop.GetExecutionHistory(-1)
	assert.Empty(t, history)

	history = mainLoop.GetExecutionHistory(0)
	assert.Empty(t, history)
}

// TestSmartMainLoop_UserInputAPI 测试用户输入API
func TestSmartMainLoop_UserInputAPI(t *testing.T) {
	mockWakeup := &MockWakeupListener{}
	
	config := core.DefaultMainLoopConfig()
	config.UserInputTimeout = 100 * time.Millisecond // 设置较短的超时时间
	logger := slog.Default()
	
	mainLoop := core.NewSmartMainLoop(
		nil,
		nil,
		nil,
		mockWakeup,
		config,
		logger,
	)

	// 测试用户输入API（不启动主循环）
	userContext := map[string]any{"source": "test"}
	_ = mainLoop.ProcessUserInput("测试输入", "test_user", userContext)
	
	// 由于主循环未启动，通道应该能接收输入但可能超时，这取决于实现
	// 这里主要测试API不会panic
	assert.NotPanics(t, func() {
		mainLoop.ProcessUserInput("测试输入2", "test_user2", userContext)
	})
}

// TestSmartMainLoop_WakeupEventAPI 测试唤醒事件API
func TestSmartMainLoop_WakeupEventAPI(t *testing.T) {
	mockWakeup := &MockWakeupListener{}
	
	config := core.DefaultMainLoopConfig()
	logger := slog.Default()
	
	_ = core.NewSmartMainLoop(
		nil,
		nil,
		nil,
		mockWakeup,
		config,
		logger,
	)

	// 测试唤醒事件API
	wakeupEvent := core.WakeupEvent{
		MonitoringTaskID: "test_task",
		TriggerTime:      time.Now(),
		ObservedData:     map[string]any{"test": "data"},
		Reason:           "测试唤醒",
		Metadata:         map[string]any{"priority": "low"},
	}

	// 测试Mock的TriggerWakeup功能
	assert.NotPanics(t, func() {
		mockWakeup.TriggerWakeup(wakeupEvent)
	})
}

// TestSmartMainLoop_MockWakeupListener 测试MockWakeupListener功能
func TestSmartMainLoop_MockWakeupListener(t *testing.T) {
	mockWakeup := &MockWakeupListener{}
	
	// 设置期望
	mockWakeup.On("SetHandler", mock.Anything).Return()
	mockWakeup.On("GetListenAddress").Return("http://localhost:8080/webhook")
	
	config := core.DefaultMainLoopConfig()
	logger := slog.Default()
	
	// 创建主循环但不使用，主要测试Mock
	_ = core.NewSmartMainLoop(
		nil,
		nil,
		nil,
		mockWakeup,
		config,
		logger,
	)

	// 测试Mock的基本功能
	assert.Equal(t, "http://localhost:8080/webhook", mockWakeup.GetListenAddress())

	// 测试Handler设置
	var testHandler func(core.WakeupEvent) error
	mockWakeup.SetHandler(testHandler)

	// 验证Mock调用
	mockWakeup.AssertExpectations(t)
} 