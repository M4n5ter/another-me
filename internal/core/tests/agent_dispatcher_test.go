package tests

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/m4n5ter/another-me/internal/core"
	"github.com/m4n5ter/another-me/internal/core/types"
)

// 重用MockAgent，但增加名称区分
type NamedMockAgent struct {
	mock.Mock
	name string
}

func NewNamedMockAgent(name string) *NamedMockAgent {
	return &NamedMockAgent{name: name}
}

func (m *NamedMockAgent) Execute(ctx context.Context, task types.Task, initialContext map[string]any) (types.ExecutionResult, error) {
	args := m.Called(ctx, task, initialContext)
	return args.Get(0).(types.ExecutionResult), args.Error(1)
}

func (m *NamedMockAgent) Name() string {
	return m.name
}

func (m *NamedMockAgent) Type() types.AgentType {
	args := m.Called()
	return args.Get(0).(types.AgentType)
}

func (m *NamedMockAgent) IsAvailable(ctx context.Context) bool {
	args := m.Called(ctx)
	return args.Bool(0)
}

func (m *NamedMockAgent) GetCapabilities() []string {
	args := m.Called()
	return args.Get(0).([]string)
}

func TestMultipleAgentsOfSameType(t *testing.T) {
	config := core.DefaultDispatcherConfig()
	dispatcher := core.NewSmartAgentDispatcher(config, nil)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		dispatcher.Shutdown(ctx)
	}()

	// 创建多个ReAct Agent
	sysOpsAgent := NewNamedMockAgent("SysOps")
	socialAgent := NewNamedMockAgent("SocialMedia")
	financeAgent := NewNamedMockAgent("Finance")

	// 设置mock期望
	sysOpsAgent.On("Type").Return(types.AgentTypeReAct)
	socialAgent.On("Type").Return(types.AgentTypeReAct)
	financeAgent.On("Type").Return(types.AgentTypeReAct)

	// 注册所有Agent
	err := dispatcher.RegisterAgent(sysOpsAgent)
	assert.NoError(t, err)

	err = dispatcher.RegisterAgent(socialAgent)
	assert.NoError(t, err)

	err = dispatcher.RegisterAgent(financeAgent)
	assert.NoError(t, err)

	// 验证可以获取同类型的多个Agent
	reactAgents, err := dispatcher.GetAgentsByType(types.AgentTypeReAct)
	assert.NoError(t, err)
	assert.Len(t, reactAgents, 3)

	// 验证Agent名称不同
	agentNames := make(map[string]bool)
	for _, agent := range reactAgents {
		agentNames[agent.Name()] = true
	}
	assert.True(t, agentNames["SysOps"])
	assert.True(t, agentNames["SocialMedia"])
	assert.True(t, agentNames["Finance"])

	sysOpsAgent.AssertExpectations(t)
	socialAgent.AssertExpectations(t)
	financeAgent.AssertExpectations(t)
}

func TestAgentLoadBalancing(t *testing.T) {
	config := core.DefaultDispatcherConfig()
	config.TaskTimeout = 2 * time.Second
	dispatcher := core.NewSmartAgentDispatcher(config, nil)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		dispatcher.Shutdown(ctx)
	}()

	// 创建两个GUI Agent
	agent1 := NewNamedMockAgent("GUI-Agent-1")
	agent2 := NewNamedMockAgent("GUI-Agent-2")

	agent1.On("Type").Return(types.AgentTypeGUI)
	agent2.On("Type").Return(types.AgentTypeGUI)

	// 注册Agent
	err := dispatcher.RegisterAgent(agent1)
	assert.NoError(t, err)

	err = dispatcher.RegisterAgent(agent2)
	assert.NoError(t, err)

	// 设置mock期望 - 每个Agent处理一个任务
	expectedResult := types.ExecutionResult{
		Status:    types.ExecutionStatusSuccess,
		Output:    "任务完成",
		StartTime: time.Now(),
		EndTime:   time.Now(),
	}

	agent1.On("Execute", mock.Anything, mock.Anything, mock.Anything).
		Return(expectedResult, nil).Once()
	
	agent2.On("Execute", mock.Anything, mock.Anything, mock.Anything).
		Return(expectedResult, nil).Once()

	// 创建两个任务
	task1 := types.Task{
		ID:        "load-balance-task-1",
		Type:      "gui_click",
		AgentType: types.AgentTypeGUI,
		CreatedAt: time.Now(),
		Context:   map[string]any{},
	}

	task2 := types.Task{
		ID:        "load-balance-task-2", 
		Type:      "gui_click",
		AgentType: types.AgentTypeGUI,
		CreatedAt: time.Now(),
		Context:   map[string]any{},
	}

	// 分发任务
	ctx := context.Background()
	result1, err1 := dispatcher.DispatchTask(ctx, task1)
	assert.NoError(t, err1)
	assert.Equal(t, types.ExecutionStatusSuccess, result1.Status)

	result2, err2 := dispatcher.DispatchTask(ctx, task2)
	assert.NoError(t, err2)
	assert.Equal(t, types.ExecutionStatusSuccess, result2.Status)

	// 验证两个Agent都被使用了
	agent1.AssertExpectations(t)
	agent2.AssertExpectations(t)
}

func TestRegisterAgentWithID(t *testing.T) {
	config := core.DefaultDispatcherConfig()
	dispatcher := core.NewSmartAgentDispatcher(config, nil)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		dispatcher.Shutdown(ctx)
	}()

	agent := NewNamedMockAgent("TestAgent")
	agent.On("Type").Return(types.AgentTypeReAct)

	// 使用RegisterAgentWithID注册
	agentID, err := dispatcher.RegisterAgentWithID(agent)
	assert.NoError(t, err)
	assert.NotEmpty(t, agentID)
	assert.Contains(t, agentID, "react")
	assert.Contains(t, agentID, "TestAgent")

	// 验证可以通过ID获取Agent
	retrievedAgent, err := dispatcher.GetAgentByID(agentID)
	assert.NoError(t, err)
	assert.Equal(t, agent, retrievedAgent)
	assert.Equal(t, "TestAgent", retrievedAgent.Name())

	agent.AssertExpectations(t)
}

func TestUnregisterAgentByID(t *testing.T) {
	config := core.DefaultDispatcherConfig()
	dispatcher := core.NewSmartAgentDispatcher(config, nil)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		dispatcher.Shutdown(ctx)
	}()

	// 注册两个同类型Agent
	agent1 := NewNamedMockAgent("Agent1")
	agent2 := NewNamedMockAgent("Agent2")

	agent1.On("Type").Return(types.AgentTypeGUI)
	agent2.On("Type").Return(types.AgentTypeGUI)

	agentID1, err := dispatcher.RegisterAgentWithID(agent1)
	assert.NoError(t, err)

	_, err = dispatcher.RegisterAgentWithID(agent2)
	assert.NoError(t, err)

	// 验证都注册成功
	guiAgents, err := dispatcher.GetAgentsByType(types.AgentTypeGUI)
	assert.NoError(t, err)
	assert.Len(t, guiAgents, 2)

	// 注销一个Agent
	err = dispatcher.UnregisterAgent(agentID1)
	assert.NoError(t, err)

	// 验证只剩一个
	guiAgents, err = dispatcher.GetAgentsByType(types.AgentTypeGUI)
	assert.NoError(t, err)
	assert.Len(t, guiAgents, 1)
	assert.Equal(t, "Agent2", guiAgents[0].Name())

	// 尝试获取已注销的Agent应该失败
	_, err = dispatcher.GetAgentByID(agentID1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "未注册")

	agent1.AssertExpectations(t)
	agent2.AssertExpectations(t)
}

func TestSpecializedAgentSelection(t *testing.T) {
	config := core.DefaultDispatcherConfig()
	dispatcher := core.NewSmartAgentDispatcher(config, nil)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		dispatcher.Shutdown(ctx)
	}()

	// 创建专门化的Agent
	sysOpsAgent := NewNamedMockAgent("SystemOperations")
	socialAgent := NewNamedMockAgent("SocialMediaAnalyst")
	financeAgent := NewNamedMockAgent("FinancialAnalyst")

	sysOpsAgent.On("Type").Return(types.AgentTypeReAct)
	socialAgent.On("Type").Return(types.AgentTypeReAct)
	financeAgent.On("Type").Return(types.AgentTypeReAct)

	// 注册所有Agent
	sysOpsID, _ := dispatcher.RegisterAgentWithID(sysOpsAgent)
	socialID, _ := dispatcher.RegisterAgentWithID(socialAgent)
	financeID, _ := dispatcher.RegisterAgentWithID(financeAgent)

	// 验证可以通过ID获取特定的Agent
	retrievedSysOps, err := dispatcher.GetAgentByID(sysOpsID)
	assert.NoError(t, err)
	assert.Equal(t, "SystemOperations", retrievedSysOps.Name())

	retrievedSocial, err := dispatcher.GetAgentByID(socialID)
	assert.NoError(t, err)
	assert.Equal(t, "SocialMediaAnalyst", retrievedSocial.Name())

	retrievedFinance, err := dispatcher.GetAgentByID(financeID)
	assert.NoError(t, err)
	assert.Equal(t, "FinancialAnalyst", retrievedFinance.Name())

	// 验证类型获取包含所有Agent
	reactAgents, err := dispatcher.GetAgentsByType(types.AgentTypeReAct)
	assert.NoError(t, err)
	assert.Len(t, reactAgents, 3)

	sysOpsAgent.AssertExpectations(t)
	socialAgent.AssertExpectations(t)
	financeAgent.AssertExpectations(t)
} 