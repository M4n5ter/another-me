package core

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	json "github.com/json-iterator/go"

	decisionmaker "github.com/m4n5ter/another-me/internal/core/tools/decision_maker"
	"github.com/m4n5ter/another-me/internal/core/types"
	"github.com/m4n5ter/another-me/pkg/llminterface"
	. "github.com/m4n5ter/another-me/pkg/option"
	"github.com/m4n5ter/another-me/pkg/reactagent"
	"github.com/m4n5ter/another-me/pkg/toolcore"
)

// SmartDecisionMaker 智能决策引擎实现 - 基于Agent的智能决策
type SmartDecisionMaker struct {
	agent            *reactagent.ReActAgent
	llmAdapter       llminterface.ChatAdapter
	mindscapeService MindscapeService
	logger           *slog.Logger
	config           DecisionMakerConfig
}

// DecisionMakerConfig 决策引擎配置
type DecisionMakerConfig struct {
	// 决策超时配置
	DecisionTimeout time.Duration `json:"decision_timeout"` // 决策超时时间
	MaxRetries      int           `json:"max_retries"`      // 最大重试次数

	// 记忆检索配置
	MemoryQueryLimit         int     `json:"memory_query_limit"`         // 记忆查询数量限制
	MemoryRelevanceThreshold float64 `json:"memory_relevance_threshold"` // 记忆相关性阈值

	// Agent选择配置
	DefaultAgent types.AgentType `json:"default_agent"` // 默认Agent类型

	// 监控策略配置
	DefaultMonitoringTTL time.Duration `json:"default_monitoring_ttl"` // 默认监控时长
	MaxMonitoringTasks   int           `json:"max_monitoring_tasks"`   // 最大监控任务数

	// 决策置信度配置
	MinConfidenceThreshold float64 `json:"min_confidence_threshold"` // 最小置信度阈值
}

// DefaultDecisionMakerConfig 返回默认决策引擎配置
func DefaultDecisionMakerConfig() DecisionMakerConfig {
	return DecisionMakerConfig{
		DecisionTimeout:          30 * time.Second,
		MaxRetries:               3,
		MemoryQueryLimit:         20,
		MemoryRelevanceThreshold: 0.6,
		DefaultAgent:             types.AgentTypeReAct,
		DefaultMonitoringTTL:     24 * time.Hour,
		MaxMonitoringTasks:       10,
		MinConfidenceThreshold:   0.3,
	}
}

// NewSmartDecisionMaker 创建决策者
func NewSmartDecisionMaker(
	agent *reactagent.ReActAgent,
	llmAdapter llminterface.ChatAdapter,
	mindscapeService MindscapeService,
	config DecisionMakerConfig,
	logger *slog.Logger,
) *SmartDecisionMaker {
	if logger == nil {
		logger = slog.Default().WithGroup("smart_decision_maker")
	}

	registry := toolcore.NewRegistry()
	err := decisionmaker.RegistDecisionMakerTools(context.Background(), registry)
	if err != nil {
		logger.Error("注册决策工具失败，无法继续初始化", "error", err)
		return nil
	}

	err = llmAdapter.RegisterTools(context.Background(), registry)
	if err != nil {
		logger.Error("注册决策工具失败，无法继续初始化", "error", err)
		return nil
	}

	return &SmartDecisionMaker{
		agent:            agent,
		llmAdapter:       llmAdapter,
		mindscapeService: mindscapeService,
		logger:           logger,
		config:           config,
	}
}

var _ DecisionMaker = (*SmartDecisionMaker)(nil)

// getSystemPrompt 系统提示词
func (dm *SmartDecisionMaker) getSystemPrompt() string {
	return `你是一个智能决策引擎，专门分析用户意图并做出最优决策。你的核心职责是：

**核心能力**：
1. 理解用户输入的真实意图和需求
2. 分析当前系统状态和相关记忆
3. 选择最合适的执行方案和Agent类型
4. 评估任务优先级和设置监控策略
5. 提供清晰的推理过程和置信度评估

**决策原则**：
- 准确性：确保理解用户真实需求，避免误解
- 效率性：选择最高效的执行路径和Agent类型
- 可靠性：设置适当的监控和容错机制
- 透明性：提供详细的决策推理过程

**Agent类型选择指南**：
- GUI Agent：适用于界面操作、点击、输入、截图、自动化操作等视觉交互任务
- ReAct Agent：适用于搜索、分析、计算、API调用、逻辑推理等工具调用任务

**输出要求**：
- 必须使用指定的工具来提供结构化输出
- 给出清晰的推理步骤和置信度评估
- 根据任务特性选择合适的优先级和监控策略

请根据具体场景调用相应的工具来提供格式化的决策结果。`
}

// DecisionPromptData 决策提示词数据结构
type DecisionPromptData struct {
	UserInput        string             `json:"user_input"`
	RelevantMemories []types.MemoryItem `json:"relevant_memories"`
	SystemContext    map[string]any     `json:"system_context"`
	WakeupEvent      *types.WakeupEvent `json:"wakeup_event,omitempty"`
}

// AgentDecisionResult Agent返回的决策结果结构
type AgentDecisionResult struct {
	ShouldExecuteTask       bool                     `json:"should_execute_task" jsonschema:"title=should_execute_task,description=Whether to execute the task,example=true,example=false,default=false"`
	SelectedAgentType       string                   `json:"selected_agent_type"`
	TaskType                string                   `json:"task_type"`
	TaskDescription         string                   `json:"task_description"`
	TaskPriority            int                      `json:"task_priority"`
	TaskParameters          map[string]any           `json:"task_parameters"`
	ShouldSetupMonitoring   bool                     `json:"should_setup_monitoring"`
	MonitoringConditions    []types.MonitorCondition `json:"monitoring_conditions"`
	ShouldEnterWaitMode     bool                     `json:"should_enter_wait_mode"`
	ReasoningSteps          []string                 `json:"reasoning_steps"`
	Confidence              float64                  `json:"confidence"`
	ExpectedDurationMinutes int                      `json:"expected_duration_minutes"`
}

// MakeDecision 进行决策 - 实现DecisionMaker接口
func (dm *SmartDecisionMaker) MakeDecision(ctx context.Context, decisionContext types.DecisionContext) (types.DecisionResult, error) {
	dm.logger.Info("开始决策分析")

	// 设置决策超时
	decisionCtx, cancel := context.WithTimeout(ctx, dm.config.DecisionTimeout)
	defer cancel()

	if decisionContext.WakeupEvent.IsSome() {
		return dm.HandleWakeupEvent(decisionCtx, decisionContext.WakeupEvent.Unwrap())
	} else {
		return dm.AnalyzeUserInput(decisionCtx, decisionContext)
	}
}

// AnalyzeUserInput 分析用户输入，通过Agent进行智能决策
func (dm *SmartDecisionMaker) AnalyzeUserInput(ctx context.Context, decisionCtx types.DecisionContext) (types.DecisionResult, error) {
	userInput := extractUserInputFromContext(decisionCtx)
	dm.logger.Info("开始分析用户输入", "user_input", userInput)

	// 1. 检索相关记忆
	memories, err := dm.retrieveRelevantMemories(ctx, decisionCtx)
	if err != nil {
		dm.logger.Warn("检索相关记忆失败", "error", err)
		memories = []types.MemoryItem{} // 继续执行，但没有记忆辅助
	}

	// 2. 构建决策提示词
	prompt := dm.buildMainDecisionPrompt(userInput, memories, decisionCtx.SystemState)

	// 3. 调用LLM进行决策
	agentResult, err := dm.invokeLLMForDecision(ctx, prompt)
	if err != nil {
		return dm.createFallbackDecision(userInput, err), nil
	}

	// 4. 转换Agent结果为决策结果
	result := dm.convertAgentResultToDecisionResult(agentResult, userInput)

	// 5. 验证和后处理决策结果
	result = dm.validateAndEnhanceDecision(result, decisionCtx)

	dm.logger.Info("决策分析完成",
		"should_execute", result.ShouldExecuteTask,
		"agent_type", dm.parseAgentType(agentResult.SelectedAgentType),
		"confidence", result.Confidence)

	return result, nil
}

// HandleWakeupEvent 处理唤醒事件，通过Agent进行智能分析
func (dm *SmartDecisionMaker) HandleWakeupEvent(ctx context.Context, wakeupEvent types.WakeupEvent) (types.DecisionResult, error) {
	dm.logger.Info("开始AI分析唤醒事件", "task_id", wakeupEvent.MonitoringTaskID, "reason", wakeupEvent.Reason)

	// 1. 检索相关记忆
	memories, err := dm.retrieveWakeupRelatedMemories(ctx, wakeupEvent)
	if err != nil {
		dm.logger.Warn("检索唤醒相关记忆失败", "error", err)
		memories = []types.MemoryItem{}
	}

	// 2. 构建唤醒事件决策提示
	wakeupInput := fmt.Sprintf("监控任务唤醒: %s", wakeupEvent.Reason)
	systemContext := map[string]any{
		"wakeup_trigger_time": wakeupEvent.TriggerTime,
		"observed_data":       wakeupEvent.ObservedData,
		"metadata":            wakeupEvent.Metadata,
	}

	prompt := dm.buildWakeupDecisionPrompt(wakeupInput, memories, systemContext, wakeupEvent)

	// 3. 调用LLM进行智能决策
	agentResult, err := dm.invokeLLMForDecision(ctx, prompt)
	if err != nil {
		return dm.createFallbackWakeupDecision(wakeupEvent, err), nil
	}

	// 4. 转换为决策结果
	result := dm.convertAgentResultToDecisionResult(agentResult, wakeupEvent.Reason)

	dm.logger.Info("AI唤醒事件分析完成",
		"should_execute", result.ShouldExecuteTask,
		"confidence", result.Confidence)

	return result, nil
}

// SelectAgent 根据任务上下文选择最适合的AgentType
func (dm *SmartDecisionMaker) SelectAgent(ctx context.Context, task types.Task, availableAgentTypes []types.AgentType) (types.AgentType, error) {
	dm.logger.Debug("选择Agent类型", "task_type", task.Type, "available_agents", availableAgentTypes)

	if len(availableAgentTypes) == 0 {
		return types.AgentTypeUnknown, fmt.Errorf("没有可用的Agent")
	}

	// 构建Agent类型选择提示
	prompt := dm.buildAgentSelectionPrompt(task, availableAgentTypes)

	// 创建Agent类型选择任务
	response, err := llminterface.ChatAndGetFullResponse(ctx, dm.llmAdapter, llminterface.ChatInput{
		Messages: []llminterface.InputMessage{
			llminterface.SystemInputMessage(dm.getSystemPrompt()),
			llminterface.UserInputMessageText(prompt),
		},
		ConversationID: uuid.NewString(),
	})
	if err != nil {
		return types.AgentTypeUnknown, fmt.Errorf("agent类型选择失败: %w", err)
	}

	if !response.HasToolCalls() {
		return types.AgentTypeUnknown, fmt.Errorf("agent类型选择失败: 没有工具调用")
	}

	// 解析Agent选择结果
	selectedAgentType := dm.parseAgentSelectionResult(response.GetToolCalls()[0].Arguments, availableAgentTypes)

	dm.logger.Info("Agent类型选择完成", "selected_agent_type", selectedAgentType)
	return selectedAgentType, nil
}

// EvaluateTaskPriority 评估任务优先级
func (dm *SmartDecisionMaker) EvaluateTaskPriority(ctx context.Context, tasks []types.Task) ([]types.Task, error) {
	if len(tasks) == 0 {
		return tasks, nil
	}

	dm.logger.Debug("AI评估任务优先级", "task_count", len(tasks))

	// 构建优先级评估提示
	prompt := dm.buildPriorityEvaluationPrompt(tasks)

	// 调用LLM进行优先级评估
	response, err := llminterface.ChatAndGetFullResponse(ctx, dm.llmAdapter, llminterface.ChatInput{
		Messages: []llminterface.InputMessage{
			llminterface.SystemInputMessage(dm.getSystemPrompt()),
			llminterface.UserInputMessageText(prompt),
		},
		ConversationID: uuid.NewString(),
	})
	if err != nil {
		return nil, fmt.Errorf("监控定义失败: %w", err)
	}

	if !response.HasToolCalls() {
		return nil, fmt.Errorf("优先级评估失败: 没有工具调用")
	}

	// 解析优先级结果并排序
	prioritizedTasks := dm.parsePriorityEvaluationResult(response.GetToolCalls()[0].Arguments, tasks)

	dm.logger.Info("AI任务优先级评估完成", "task_count", len(prioritizedTasks))
	return prioritizedTasks, nil
}

// DefineMonitoringConditions 定义监控条件
func (dm *SmartDecisionMaker) DefineMonitoringConditions(ctx context.Context, systemState types.SystemState) ([]types.MonitoringTask, error) {
	dm.logger.Debug("AI定义监控条件")

	// 构建监控条件定义提示
	prompt := dm.buildMonitoringDefinitionPrompt(systemState)

	response, err := llminterface.ChatAndGetFullResponse(ctx, dm.llmAdapter, llminterface.ChatInput{
		Messages: []llminterface.InputMessage{
			llminterface.SystemInputMessage(dm.getSystemPrompt()),
			llminterface.UserInputMessageText(prompt),
		},
		ConversationID: uuid.NewString(),
	})
	if err != nil {
		return nil, fmt.Errorf("优先级评估失败: %w", err)
	}

	if !response.HasToolCalls() {
		return nil, fmt.Errorf("监控定义失败: 没有工具调用")
	}

	// 解析监控条件结果
	monitoringTasks := dm.parseMonitoringDefinitionResult(response.GetToolCalls()[0].Arguments)

	dm.logger.Info("AI监控条件定义完成", "monitoring_task_count", len(monitoringTasks))
	return monitoringTasks, nil
}

// AnalyzeWakeupEvent 分析唤醒事件
func (dm *SmartDecisionMaker) AnalyzeWakeupEvent(ctx context.Context, wakeupEvent types.WakeupEvent) (types.DecisionResult, error) {
	return dm.HandleWakeupEvent(ctx, wakeupEvent)
}

// GetDecisionHistory 获取决策历史
func (dm *SmartDecisionMaker) GetDecisionHistory(ctx context.Context, limit int) ([]types.DecisionResult, error) {
	dm.logger.Debug("获取决策历史", "limit", limit)

	// 从Mindscape检索决策历史
	queryContext := map[string]any{
		"type":     "decision_history",
		"limit":    limit,
		"order_by": "created_at",
		"order":    "desc",
	}

	memories, err := dm.mindscapeService.RetrieveMemories(ctx, queryContext)
	if err != nil {
		dm.logger.Error("检索决策历史失败", "error", err)
		return []types.DecisionResult{}, fmt.Errorf("检索决策历史失败: %w", err)
	}

	// 转换记忆为决策结果
	var decisionHistory []types.DecisionResult
	for _, memory := range memories {
		if decisionResult, ok := convertMemoryToDecisionResult(memory); ok {
			decisionHistory = append(decisionHistory, decisionResult)
		}
	}

	if len(decisionHistory) > limit {
		decisionHistory = decisionHistory[:limit]
	}

	dm.logger.Info("决策历史获取完成", "count", len(decisionHistory))
	return decisionHistory, nil
}

// 私有方法实现

// invokeLLMForDecision 调用LLM进行决策
func (dm *SmartDecisionMaker) invokeLLMForDecision(ctx context.Context, prompt string) (*AgentDecisionResult, error) {
	response, err := llminterface.ChatAndGetFullResponse(ctx, dm.llmAdapter, llminterface.ChatInput{
		Messages: []llminterface.InputMessage{
			llminterface.SystemInputMessage(dm.getSystemPrompt()),
			llminterface.UserInputMessageText(prompt),
		},
		ConversationID: uuid.NewString(),
	})
	if err != nil {
		return nil, fmt.Errorf("决策失败: %w", err)
	}

	if !response.HasToolCalls() {
		return nil, fmt.Errorf("决策失败: 没有工具调用")
	}

	// 解析Agent输出的JSON结果
	var agentResult AgentDecisionResult
	if err := json.UnmarshalFromString(response.GetToolCalls()[0].Arguments, &agentResult); err != nil {
		// 如果JSON解析失败，尝试从文本中提取关键信息
		dm.logger.Warn("JSON解析失败，尝试文本解析", "error", err, "output", response.GetToolCalls()[0].Arguments)
		return dm.parseDecisionFromText(response.GetToolCalls()[0].Arguments), nil
	}

	return &agentResult, nil
}

// buildMainDecisionPrompt 构建主要决策提示词
func (dm *SmartDecisionMaker) buildMainDecisionPrompt(userInput string, memories []types.MemoryItem, systemContext map[string]any) string {
	return fmt.Sprintf(`请分析以下用户请求并做出决策：

**用户输入**: %s

**相关记忆**: 
%s

**当前系统状态**: 
%s

请仔细分析用户的真实意图，考虑以下因素：
1. 用户想要完成什么任务？
2. 这个任务最适合哪种Agent类型（GUI操作 vs ReAct推理）？
3. 任务的紧急程度和重要性如何？
4. 是否需要设置监控来跟踪执行状态？
5. 预期的执行时间是多少？

请使用 decision_maker 工具提供完整的决策结果，包括详细的推理步骤和置信度评估。`,
		userInput,
		dm.formatMemories(memories),
		dm.formatContext(systemContext))
}

// buildWakeupDecisionPrompt 构建唤醒事件决策提示词
func (dm *SmartDecisionMaker) buildWakeupDecisionPrompt(wakeupInput string, memories []types.MemoryItem, systemContext map[string]any, wakeupEvent types.WakeupEvent) string {
	return fmt.Sprintf(`分析以下监控唤醒事件并决定后续行动：

**唤醒原因**: %s
**监控任务ID**: %s
**触发时间**: %s

**相关记忆**: 
%s

**观测数据**: 
%s

**系统上下文**: 
%s

请分析：
1. 这个唤醒事件的性质和严重程度
2. 是否需要立即采取行动？
3. 如果需要行动，应该使用什么类型的Agent？
4. 是否需要调整或重新设置监控条件？

请使用 decision_maker 工具提供决策结果。`,
		wakeupEvent.Reason,
		wakeupEvent.MonitoringTaskID,
		wakeupEvent.TriggerTime.Format(time.RFC3339),
		dm.formatMemories(memories),
		dm.formatContext(wakeupEvent.ObservedData),
		dm.formatContext(systemContext))
}

func (dm *SmartDecisionMaker) formatMemories(memories []types.MemoryItem) string {
	if len(memories) == 0 {
		return "无相关记忆"
	}

	var formatted strings.Builder
	for i, memory := range memories {
		formatted.WriteString(fmt.Sprintf("%d. %s (重要性: %.2f)\n", i+1, memory.Content, memory.Importance))
	}
	return formatted.String()
}

func (dm *SmartDecisionMaker) formatContext(context map[string]any) string {
	if len(context) == 0 {
		return "无上下文信息"
	}

	contextJSON, err := json.MarshalIndent(context, "", "  ")
	if err != nil {
		return fmt.Sprintf("上下文格式化错误: %v", err)
	}
	return string(contextJSON)
}

func (dm *SmartDecisionMaker) formatTaskParameters(params map[string]any) string {
	if len(params) == 0 {
		return "无参数"
	}

	paramsJSON, err := json.MarshalIndent(params, "", "  ")
	if err != nil {
		return fmt.Sprintf("参数格式化错误: %v", err)
	}
	return string(paramsJSON)
}

func (dm *SmartDecisionMaker) formatSystemState(state types.SystemState) string {
	stateJSON, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Sprintf("系统状态格式化错误: %v", err)
	}
	return string(stateJSON)
}

func (dm *SmartDecisionMaker) formatWakeupEvent(event types.WakeupEvent) string {
	eventJSON, err := json.MarshalIndent(event, "", "  ")
	if err != nil {
		return fmt.Sprintf("唤醒事件格式化错误: %v", err)
	}
	return string(eventJSON)
}

func (dm *SmartDecisionMaker) parseDecisionFromText(output string) *AgentDecisionResult {
	// 当JSON解析失败时的文本解析后备方案
	result := &AgentDecisionResult{
		ShouldExecuteTask:       true,
		SelectedAgentType:       "react",
		TaskType:                "general",
		TaskDescription:         output,
		TaskPriority:            5,
		TaskParameters:          map[string]any{},
		ShouldSetupMonitoring:   false,
		MonitoringConditions:    []types.MonitorCondition{},
		ShouldEnterWaitMode:     false,
		ReasoningSteps:          []string{"基于文本解析的默认决策"},
		Confidence:              0.5,
		ExpectedDurationMinutes: 5,
	}

	// 简单的关键词分析
	outputLower := strings.ToLower(output)
	if strings.Contains(outputLower, "gui") || strings.Contains(outputLower, "界面") || strings.Contains(outputLower, "点击") {
		result.SelectedAgentType = "gui"
	}

	if strings.Contains(outputLower, "不执行") || strings.Contains(outputLower, "等待") {
		result.ShouldExecuteTask = false
		result.ShouldEnterWaitMode = true
	}

	return result
}

func (dm *SmartDecisionMaker) convertAgentResultToDecisionResult(agentResult *AgentDecisionResult, userInput string) types.DecisionResult {
	result := types.DecisionResult{
		ShouldExecuteTask:   agentResult.ShouldExecuteTask,
		ShouldEnterWaitMode: agentResult.ShouldEnterWaitMode,
		ReasoningSteps:      agentResult.ReasoningSteps,
		Confidence:          agentResult.Confidence,
		MonitoringTasks:     []types.MonitoringTask{},
	}

	// 设置预期执行时间
	if agentResult.ExpectedDurationMinutes > 0 {
		result.ExpectedDuration = Some(time.Duration(agentResult.ExpectedDurationMinutes) * time.Minute)
	} else {
		result.ExpectedDuration = Some(5 * time.Minute)
	}

	// 如果需要执行任务，创建任务
	if agentResult.ShouldExecuteTask {
		task := types.Task{
			ID:          generateTaskID(),
			Type:        agentResult.TaskType,
			Description: agentResult.TaskDescription,
			Priority:    agentResult.TaskPriority,
			AgentType:   dm.parseAgentType(agentResult.SelectedAgentType),
			Parameters:  agentResult.TaskParameters,
			Context: map[string]any{
				"generated_by": "smart_decision_maker",
				"user_input":   userInput,
			},
			CreatedAt: time.Now(),
		}
		result.Task = Some(task)
	} else {
		result.Task = None[types.Task]()
	}

	// 如果需要设置监控，创建监控任务
	if agentResult.ShouldSetupMonitoring {
		monitoringTask := types.MonitoringTask{
			ID:                  None[string](),
			Description:         fmt.Sprintf("基于决策的监控任务: %s", agentResult.TaskDescription),
			MindscapeTaskType:   "smart_monitoring",
			Conditions:          agentResult.MonitoringConditions,
			TargetData:          []string{"status", "result", "timestamp"},
			NotificationMethods: []string{"webhook"},
			WebhookURL:          None[string](),
			MQTopic:             None[string](),
			MaxRetries:          Some(3),
			IsEnabled:           true,
			CreatedAt:           time.Now(),
			UpdatedAt:           time.Now(),
		}
		result.MonitoringTasks = []types.MonitoringTask{monitoringTask}
	}

	return result
}

func (dm *SmartDecisionMaker) validateAndEnhanceDecision(result types.DecisionResult, ctx types.DecisionContext) types.DecisionResult {
	// 验证置信度
	if result.Confidence < dm.config.MinConfidenceThreshold {
		dm.logger.Warn("决策置信度过低", "confidence", result.Confidence)
		result.ShouldEnterWaitMode = true
		result.ReasoningSteps = append(result.ReasoningSteps,
			fmt.Sprintf("置信度过低(%.2f)，进入等待模式", result.Confidence))
	}

	// 验证任务参数
	if result.Task.IsSome() {
		task := result.Task.Unwrap()
		if task.Parameters == nil {
			task.Parameters = map[string]any{}
		}

		// 确保GUI任务有必要的参数
		if task.AgentType == types.AgentTypeGUI {
			if _, exists := task.Parameters["instruction"]; !exists {
				task.Parameters["instruction"] = task.Description
			}
		}

		result.Task = Some(task)
	}

	return result
}

func (dm *SmartDecisionMaker) createFallbackDecision(userInput string, err error) types.DecisionResult {
	dm.logger.Error("AI决策失败，使用后备决策", "error", err)

	return types.DecisionResult{
		ShouldExecuteTask:   false,
		Task:                None[types.Task](),
		MonitoringTasks:     []types.MonitoringTask{},
		ShouldEnterWaitMode: true,
		ReasoningSteps:      []string{fmt.Sprintf("AI决策失败: %v，进入等待模式", err)},
		Confidence:          0.1,
		ExpectedDuration:    None[time.Duration](),
	}
}

func (dm *SmartDecisionMaker) createFallbackWakeupDecision(wakeupEvent types.WakeupEvent, err error) types.DecisionResult {
	dm.logger.Error("AI唤醒事件分析失败，使用后备决策", "error", err)

	return types.DecisionResult{
		ShouldExecuteTask:   false,
		Task:                None[types.Task](),
		MonitoringTasks:     []types.MonitoringTask{},
		ShouldEnterWaitMode: true,
		ReasoningSteps:      []string{fmt.Sprintf("AI唤醒分析失败: %v，继续等待", err)},
		Confidence:          0.1,
		ExpectedDuration:    None[time.Duration](),
	}
}

// 工具方法

func (dm *SmartDecisionMaker) retrieveRelevantMemories(ctx context.Context, decisionCtx types.DecisionContext) ([]types.MemoryItem, error) {
	userInput := extractUserInputFromContext(decisionCtx)
	queryContext := map[string]any{
		"query":     userInput,
		"limit":     dm.config.MemoryQueryLimit,
		"min_score": int(dm.config.MemoryRelevanceThreshold * 100),
	}

	if userID, exists := decisionCtx.SystemState["user_id"]; exists {
		queryContext["user_id"] = userID
	}

	return dm.mindscapeService.RetrieveMemories(ctx, queryContext)
}

func (dm *SmartDecisionMaker) retrieveWakeupRelatedMemories(ctx context.Context, wakeupEvent types.WakeupEvent) ([]types.MemoryItem, error) {
	queryContext := map[string]any{
		"query":    wakeupEvent.Reason,
		"task_id":  wakeupEvent.MonitoringTaskID,
		"limit":    dm.config.MemoryQueryLimit,
		"keywords": []string{wakeupEvent.MonitoringTaskID, wakeupEvent.Reason},
	}

	return dm.mindscapeService.RetrieveMemories(ctx, queryContext)
}

func (dm *SmartDecisionMaker) parseAgentType(agentTypeStr string) types.AgentType {
	switch strings.ToLower(agentTypeStr) {
	case "gui":
		return types.AgentTypeGUI
	case "react":
		return types.AgentTypeReAct
	default:
		return dm.config.DefaultAgent
	}
}

func extractUserInputFromContext(ctx types.DecisionContext) string {
	if ctx.WakeupEvent.IsSome() {
		return fmt.Sprintf("监控任务触发: %s", ctx.WakeupEvent.Unwrap().Reason)
	}

	if userInput, exists := ctx.SystemState["user_input"]; exists {
		if str, ok := userInput.(string); ok {
			return str
		}
	}

	if userInput, exists := ctx.SystemState["instruction"]; exists {
		if str, ok := userInput.(string); ok {
			return str
		}
	}

	return "系统自动决策"
}

func generateTaskID() string {
	return fmt.Sprintf("task_%d", time.Now().UnixNano())
}

func convertMemoryToDecisionResult(memory types.MemoryItem) (types.DecisionResult, bool) {
	if memory.Type != "decision_history" {
		return types.DecisionResult{}, false
	}

	result := types.DecisionResult{
		ShouldExecuteTask:   false,
		ShouldEnterWaitMode: true,
		MonitoringTasks:     []types.MonitoringTask{},
		ReasoningSteps:      []string{"从历史记忆恢复的决策"},
		Confidence:          0.5,
		ExpectedDuration:    None[time.Duration](),
	}

	if context, ok := memory.Metadata["decision_result"].(map[string]any); ok {
		if shouldExecute, ok := context["should_execute_task"].(bool); ok {
			result.ShouldExecuteTask = shouldExecute
		}
		if confidence, ok := context["confidence"].(float64); ok {
			result.Confidence = confidence
		}
	}

	return result, true
}

// buildAgentSelectionPrompt 构建Agent选择提示词
func (dm *SmartDecisionMaker) buildAgentSelectionPrompt(task types.Task, availableAgentTypes []types.AgentType) string {
	agentTypesStr := make([]string, len(availableAgentTypes))
	for i, agentType := range availableAgentTypes {
		agentTypesStr[i] = string(agentType)
	}

	return fmt.Sprintf(`请为以下任务选择最合适的Agent类型：

**任务信息**:
- 类型: %s
- 描述: %s
- 参数: %s
- 上下文: %s

**可用Agent类型**: %s

**选择标准**:
- GUI Agent: 适用于界面操作、点击、输入、截图等视觉交互
- ReAct Agent: 适用于搜索、分析、计算、API调用等逻辑推理

请使用 select_agent 工具提供你的选择，包括选择理由和置信度。`,
		task.Type,
		task.Description,
		dm.formatTaskParameters(task.Parameters),
		dm.formatContext(task.Context),
		strings.Join(agentTypesStr, ", "))
}

func (dm *SmartDecisionMaker) parseAgentSelectionResult(arguments string, availableAgents []types.AgentType) types.AgentType {
	var selection decisionmaker.AgentSelection
	if err := json.UnmarshalFromString(arguments, &selection); err != nil {
		dm.logger.Warn("解析Agent选择结果失败", "error", err)
		return dm.config.DefaultAgent
	}

	selectedType := dm.parseAgentType(string(selection.SelectedAgentType))
	if slices.Contains(availableAgents, selectedType) {
		return selectedType
	}

	return dm.config.DefaultAgent
}

// buildPriorityEvaluationPrompt 构建优先级评估提示词
func (dm *SmartDecisionMaker) buildPriorityEvaluationPrompt(tasks []types.Task) string {
	tasksInfo := make([]string, len(tasks))
	for i, task := range tasks {
		tasksInfo[i] = fmt.Sprintf("- ID: %s, 类型: %s, 描述: %s, 当前优先级: %d",
			task.ID, task.Type, task.Description, task.Priority)
	}

	return fmt.Sprintf(`请评估以下任务的优先级并重新排序：

**任务列表**:
%s

**评估标准**:
1. 紧急程度 (用户等待时间、时效性)
2. 重要程度 (对系统/用户的影响)
3. 依赖关系 (是否阻塞其他任务)
4. 资源需求 (执行难度和时间)

优先级范围: 1-10 (10为最高优先级)

请使用 evaluate_priority 工具提供排序结果，包括每个任务的新优先级和整体推理。`,
		strings.Join(tasksInfo, "\n"))
}

func (dm *SmartDecisionMaker) parsePriorityEvaluationResult(arguments string, originalTasks []types.Task) []types.Task {
	var evaluation decisionmaker.TaskPriorityEvaluation
	if err := json.UnmarshalFromString(arguments, &evaluation); err != nil {
		dm.logger.Warn("解析优先级评估结果失败", "error", err)
		return originalTasks
	}

	// 创建任务ID到任务的映射
	taskMap := make(map[string]types.Task)
	for _, task := range originalTasks {
		taskMap[task.ID] = task
	}

	// 根据评估结果重新排序任务
	var sortedTasks []types.Task
	for _, priorityItem := range evaluation.PrioritizedTasks {
		if task, exists := taskMap[priorityItem.TaskID]; exists {
			task.Priority = priorityItem.Priority
			sortedTasks = append(sortedTasks, task)
			delete(taskMap, priorityItem.TaskID)
		}
	}

	// 添加未在评估中的任务
	for _, task := range taskMap {
		sortedTasks = append(sortedTasks, task)
	}

	return sortedTasks
}

// buildMonitoringDefinitionPrompt 构建监控定义提示词
func (dm *SmartDecisionMaker) buildMonitoringDefinitionPrompt(systemState types.SystemState) string {
	return fmt.Sprintf(`基于当前系统状态，定义合适的监控任务：

**系统状态**: 
%s

**监控目标**:
1. 识别系统异常和性能问题
2. 跟踪重要任务的执行状态
3. 监控用户交互和反馈
4. 检测安全威胁和数据泄露

**监控类型**:
- 实时监控: 需要即时响应的关键指标
- 定期检查: 周期性状态评估
- 事件驱动: 基于特定触发条件

请使用 define_monitoring 工具定义监控任务，包括监控条件、触发阈值、通知方式等。`,
		dm.formatSystemState(systemState))
}

func (dm *SmartDecisionMaker) parseMonitoringDefinitionResult(arguments string) []types.MonitoringTask {
	var definition decisionmaker.MonitoringDefinition
	if err := json.UnmarshalFromString(arguments, &definition); err != nil {
		dm.logger.Warn("解析监控定义结果失败", "error", err)
		return []types.MonitoringTask{}
	}

	return definition.MonitoringTasks
}
