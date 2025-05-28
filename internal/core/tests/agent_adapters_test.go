package tests

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/m4n5ter/another-me/internal/core"
	"github.com/m4n5ter/another-me/internal/core/agents"
	"github.com/m4n5ter/another-me/pkg/llminterface"
	"github.com/m4n5ter/another-me/pkg/toolcore"
)

// MockChatAdapter 模拟LLM适配器
type MockChatAdapter struct {
	mock.Mock
}

func (m *MockChatAdapter) Chat(ctx context.Context, input llminterface.ChatInput) (<-chan llminterface.ChatOutputChunk, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(<-chan llminterface.ChatOutputChunk), args.Error(1)
}

func (m *MockChatAdapter) RegisterTools(ctx context.Context, registry *toolcore.Registry) error {
	args := m.Called(ctx, registry)
	return args.Error(0)
}

func (m *MockChatAdapter) GetFrameworkName() string {
	args := m.Called()
	return args.String(0)
}

// MockTool 模拟工具
type MockTool struct {
	mock.Mock
	name string
}

func (m *MockTool) Schema(ctx context.Context) (toolcore.ToolSchema, error) {
	args := m.Called(ctx)
	return args.Get(0).(toolcore.ToolSchema), args.Error(1)
}

func (m *MockTool) Call(ctx context.Context, inputJSON string) (string, error) {
	args := m.Called(ctx, inputJSON)
	return args.String(0), args.Error(1)
}

func TestGUIAgentAdapterBasics(t *testing.T) {
	// 注意：这个测试需要GUI环境，在CI中可能会失败
	// 这里主要测试接口实现的正确性
	
	mockLLM := &MockChatAdapter{}
	
	// 由于GUIAgent需要实际的GUI环境，我们只测试基本的接口方法
	adapter, err := agents.NewGUIAgentAdapter(context.Background(), mockLLM)
	if err != nil {
		// 在没有GUI环境的情况下，这是预期的
		t.Skipf("跳过GUI测试，需要GUI环境: %v", err)
		return
	}
	
	// 测试基本属性
	assert.Equal(t, "GUIAgent", adapter.Name())
	assert.Equal(t, core.AgentTypeGUI, adapter.Type())
	
	// 测试能力描述
	capabilities := adapter.GetCapabilities()
	assert.NotEmpty(t, capabilities)
	assert.Contains(t, capabilities, "GUI操作自动化")
}

func TestGUIAgentAdapterExecuteWithMissingInstruction(t *testing.T) {
	mockLLM := &MockChatAdapter{}
	
	adapter, err := agents.NewGUIAgentAdapter(context.Background(), mockLLM)
	if err != nil {
		t.Skipf("跳过GUI测试，需要GUI环境: %v", err)
		return
	}
	
	// 测试缺少instruction参数的情况
	task := core.Task{
		ID:          "test-task-001",
		Type:        "gui_click",
		Description: "测试任务",
		AgentType:   core.AgentTypeGUI,
		Parameters:  map[string]any{}, // 缺少instruction
		CreatedAt:   time.Now(),
	}
	
	result, err := adapter.Execute(context.Background(), task, nil)
	
	assert.Error(t, err)
	assert.Equal(t, core.ExecutionStatusFailure, result.Status)
	assert.Contains(t, result.Error, "instruction")
}

func TestReActAgentAdapterBasics(t *testing.T) {
	mockLLM := &MockChatAdapter{}
	
	// 创建模拟工具注册表
	registry := toolcore.NewRegistry()
	mockTool := &MockTool{name: "test_tool"}
	
	// 设置模拟工具的Schema方法
	mockTool.On("Schema", mock.Anything).Return(toolcore.ToolSchema{
		Name: "test_tool",
		Descriptions: map[string]string{
			"en": "Test tool",
			"zh": "测试工具",
		},
	}, nil)
	
	registry.Register(context.Background(), mockTool)
	
	adapter, err := agents.NewReActAgentAdapter(
		context.Background(),
		mockLLM,
		registry,
		"You are a helpful assistant.",
		5,
	)
	
	assert.NoError(t, err)
	assert.NotNil(t, adapter)
	
	// 测试基本属性
	assert.Equal(t, "ReActAgent", adapter.Name())
	assert.Equal(t, core.AgentTypeReAct, adapter.Type())
	
	// 测试可用性检查
	assert.True(t, adapter.IsAvailable(context.Background()))
	
	// 测试能力描述
	capabilities := adapter.GetCapabilities()
	assert.NotEmpty(t, capabilities)
	assert.Contains(t, capabilities, "基于ReAct范式的推理和行动")
	assert.Contains(t, capabilities, "支持1个工具")
	assert.Contains(t, capabilities, "工具: test_tool")
}

func TestReActAgentAdapterWithEmptyRegistry(t *testing.T) {
	mockLLM := &MockChatAdapter{}
	
	// 创建空的工具注册表
	registry := toolcore.NewRegistry()
	
	adapter, err := agents.NewReActAgentAdapter(
		context.Background(),
		mockLLM,
		registry,
		"You are a helpful assistant.",
		5,
	)
	
	assert.NoError(t, err)
	assert.NotNil(t, adapter)
	
	// 空注册表应该被认为是不可用的
	assert.False(t, adapter.IsAvailable(context.Background()))
}

func TestReActAgentAdapterWithNilRegistry(t *testing.T) {
	mockLLM := &MockChatAdapter{}
	
	// 传入nil注册表应该返回错误
	adapter, err := agents.NewReActAgentAdapter(
		context.Background(),
		mockLLM,
		nil,
		"You are a helpful assistant.",
		5,
	)
	
	assert.Error(t, err)
	assert.Nil(t, adapter)
	assert.Contains(t, err.Error(), "toolRegistry不能为空")
}

func TestAgentTypeConstants(t *testing.T) {
	// 测试Agent类型常量
	assert.Equal(t, core.AgentType("gui"), core.AgentTypeGUI)
	assert.Equal(t, core.AgentType("react"), core.AgentTypeReAct)
	assert.Equal(t, core.AgentType("unknown"), core.AgentTypeUnknown)
} 