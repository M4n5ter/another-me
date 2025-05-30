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
	"github.com/m4n5ter/another-me/pkg/i18n"
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
	// DecisionMaker配置
	SystemPrompt    string        `json:"system_prompt"`    // 系统提示词
	PromptTemplate  string        `json:"prompt_template"`  // 提示词模板
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
		PromptTemplate: `
请基于以下信息进行决策分析：

**用户输入**: {{.UserInput}}

**相关记忆**: {{.RelevantMemories}}

**当前上下文**: {{.SystemContext}}

**唤醒事件**: {{.WakeupEvent}}

按照以下JSON格式调用决策工具输出决策结果：
{
  "should_execute_task": true/false,
  "selected_agent_type": "gui" | "react",
  "task_type": "任务类型",
  "task_description": "任务描述",
  "task_priority": 1-10,
  "task_parameters": {},
  "should_setup_monitoring": true/false,
  "monitoring_conditions": [],
  "should_enter_wait_mode": true/false,
  "reasoning_steps": ["推理步骤1", "推理步骤2"],
  "confidence": 0.0-1.0,
  "expected_duration_minutes": 5
}

**决策原则**:
1. 仔细分析用户意图，选择最合适的Agent类型
2. GUI Agent适用于：界面操作、点击、输入、截图等
3. ReAct Agent适用于：搜索、分析、计算、工具调用等
4. 根据任务紧急程度设置优先级
5. 考虑是否需要设置监控任务
6. 给出详细的推理过程和置信度评估`,
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

// NewSmartDecisionMaker 创建新的智能决策引擎
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
	dicisionMakerTool := decisionmaker.NewMakeDecisionTool(i18n.GlobalManager)
	err := registry.Register(context.Background(), dicisionMakerTool)
	if err != nil {
		logger.Error("注册决策工具失败，无法继续初始化", "error", err)
		return nil
	}

	llmAdapter.RegisterTools(context.Background(), registry)

	return &SmartDecisionMaker{
		agent:            agent,
		llmAdapter:       llmAdapter,
		mindscapeService: mindscapeService,
		logger:           logger,
		config:           config,
	}
}

var _ DecisionMaker = (*SmartDecisionMaker)(nil)

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
	dm.logger.Info("开始AI驱动的决策分析")

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
	dm.logger.Info("开始AI分析用户输入", "user_input", userInput)

	// 1. 检索相关记忆
	memories, err := dm.retrieveRelevantMemories(ctx, decisionCtx)
	if err != nil {
		dm.logger.Warn("检索相关记忆失败", "error", err)
		memories = []types.MemoryItem{} // 继续执行，但没有记忆辅助
	}

	// 2. 构建Agent决策提示
	promptData := DecisionPromptData{
		UserInput:        userInput,
		RelevantMemories: memories,
		SystemContext:    decisionCtx.SystemState,
		WakeupEvent:      nil,
	}

	// 3. 调用Agent进行智能决策
	agentResult, err := dm.invokeAgentForDecision(ctx, promptData)
	if err != nil {
		return dm.createFallbackDecision(userInput, err), nil
	}

	// 4. 转换Agent结果为标准决策结果
	result := dm.convertAgentResultToDecisionResult(agentResult, userInput)

	// 5. 验证和后处理决策结果
	result = dm.validateAndEnhanceDecision(result, decisionCtx)

	dm.logger.Info("AI决策分析完成",
		"should_execute", result.ShouldExecuteTask,
		"agent_type", dm.getAgentTypeFromResult(agentResult),
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
	promptData := DecisionPromptData{
		UserInput:        fmt.Sprintf("监控任务唤醒: %s", wakeupEvent.Reason),
		RelevantMemories: memories,
		SystemContext: map[string]any{
			"wakeup_trigger_time": wakeupEvent.TriggerTime,
			"observed_data":       wakeupEvent.ObservedData,
			"metadata":            wakeupEvent.Metadata,
		},
		WakeupEvent: &wakeupEvent,
	}

	// 3. 调用Agent进行智能决策
	agentResult, err := dm.invokeAgentForDecision(ctx, promptData)
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
	promptData := map[string]any{
		"task_description":      task.Description,
		"task_type":             task.Type,
		"task_parameters":       task.Parameters,
		"available_agent_types": availableAgentTypes,
		"context":               task.Context,
	}

	prompt := dm.buildAgentSelectionPrompt(promptData)

	// 创建Agent类型选择任务
	response, err := llminterface.ChatAndGetFullResponse(ctx, dm.llmAdapter, llminterface.ChatInput{
		Messages: []llminterface.InputMessage{
			llminterface.SystemInputMessage(dm.config.SystemPrompt),
			llminterface.UserInputMessageText(prompt),
		},
		ConversationID: uuid.NewString(),
	})
	if err != nil {
		return types.AgentTypeUnknown, fmt.Errorf("Agent类型选择失败: %w", err)
	}

	if !response.HasToolCalls() {
		return types.AgentTypeUnknown, fmt.Errorf("Agent类型选择失败: 没有工具调用")
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
			llminterface.SystemInputMessage(dm.config.SystemPrompt),
			llminterface.UserInputMessageText(prompt),
		},
		ConversationID: uuid.NewString(),
	})
	if err != nil {
		return nil, err
	}

	// 解析优先级结果并排序
	prioritizedTasks := dm.parsePriorityEvaluationResult(response.FullText, tasks)

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
			llminterface.SystemInputMessage(dm.config.SystemPrompt),
			llminterface.UserInputMessageText(prompt),
		},
		ConversationID: uuid.NewString(),
	})
	if err != nil {
		return nil, err
	}

	// 解析监控条件结果
	monitoringTasks := dm.parseMonitoringDefinitionResult(response.FullText)

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
		return []types.DecisionResult{}, err
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

func (dm *SmartDecisionMaker) invokeAgentForDecision(ctx context.Context, promptData DecisionPromptData) (*AgentDecisionResult, error) {
	// 构建决策提示
	prompt := dm.buildDecisionPrompt(promptData)

	// 调用 LLM 进行决策
	response, err := llminterface.ChatAndGetFullResponse(ctx, dm.llmAdapter, llminterface.ChatInput{
		Messages: []llminterface.InputMessage{
			llminterface.SystemInputMessage(dm.config.SystemPrompt),
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

func (dm *SmartDecisionMaker) buildDecisionPrompt(data DecisionPromptData) string {
	template := dm.config.PromptTemplate

	// 简单的模板替换
	template = strings.ReplaceAll(template, "{{.UserInput}}", data.UserInput)
	template = strings.ReplaceAll(template, "{{.RelevantMemories}}", dm.formatMemories(data.RelevantMemories))
	template = strings.ReplaceAll(template, "{{.SystemContext}}", dm.formatContext(data.SystemContext))

	if data.WakeupEvent != nil {
		template = strings.ReplaceAll(template, "{{.WakeupEvent}}", dm.formatWakeupEvent(*data.WakeupEvent))
	} else {
		template = strings.ReplaceAll(template, "{{.WakeupEvent}}", "无")
	}

	return template
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

	contextJson, err := json.MarshalIndent(context, "", "  ")
	if err != nil {
		return fmt.Sprintf("上下文格式化错误: %v", err)
	}
	return string(contextJson)
}

func (dm *SmartDecisionMaker) formatWakeupEvent(event types.WakeupEvent) string {
	eventJson, err := json.MarshalIndent(event, "", "  ")
	if err != nil {
		return fmt.Sprintf("唤醒事件格式化错误: %v", err)
	}
	return string(eventJson)
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

func (dm *SmartDecisionMaker) getAgentTypeFromResult(result *AgentDecisionResult) types.AgentType {
	return dm.parseAgentType(result.SelectedAgentType)
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

// 以下是其他辅助方法的简化实现，主要由AI Agent处理

func (dm *SmartDecisionMaker) buildAgentSelectionPrompt(data map[string]any) string {
	return fmt.Sprintf(`请为以下任务选择最合适的Agent类型：

任务描述: %v
任务类型: %v
可用Agent类型: %v

请返回最合适的Agent类型（gui 或 react）`,
		data["task_description"], data["task_type"], data["available_agent_types"])
}

func (dm *SmartDecisionMaker) parseAgentSelectionResult(output any, availableAgents []types.AgentType) types.AgentType {
	outputStr := strings.ToLower(fmt.Sprintf("%v", output))

	if strings.Contains(outputStr, "gui") {
		if slices.Contains(availableAgents, types.AgentTypeGUI) {
			return types.AgentTypeGUI
		}
	}

	if strings.Contains(outputStr, "react") {
		if slices.Contains(availableAgents, types.AgentTypeReAct) {
			return types.AgentTypeReAct
		}
	}

	return availableAgents[0]
}

func (dm *SmartDecisionMaker) buildPriorityEvaluationPrompt(tasks []types.Task) string {
	tasksJson, _ := json.MarshalIndent(tasks, "", "  ")
	return fmt.Sprintf(`请为以下任务评估优先级并排序（1-10，10为最高）：

%s

请返回按优先级排序的任务ID列表`, string(tasksJson))
}

func (dm *SmartDecisionMaker) parsePriorityEvaluationResult(output any, tasks []types.Task) []types.Task {
	// 简化实现：如果解析失败，保持原序
	return tasks
}

func (dm *SmartDecisionMaker) buildMonitoringDefinitionPrompt(systemState types.SystemState) string {
	stateJson, _ := json.MarshalIndent(systemState, "", "  ")
	return fmt.Sprintf(`基于以下系统状态，定义合适的监控条件：

%s

请返回JSON格式的监控任务定义`, string(stateJson))
}

func (dm *SmartDecisionMaker) parseMonitoringDefinitionResult(output any) []types.MonitoringTask {
	// 简化实现：返回默认监控任务
	return []types.MonitoringTask{}
}

func (dm *SmartDecisionMaker) createDefaultMonitoringTasks(systemState types.SystemState) []types.MonitoringTask {
	return []types.MonitoringTask{
		{
			ID:                  None[string](),
			Description:         "默认系统监控",
			MindscapeTaskType:   "default_monitoring",
			Conditions:          []types.MonitorCondition{},
			TargetData:          []string{"status"},
			NotificationMethods: []string{"webhook"},
			WebhookURL:          None[string](),
			MQTopic:             None[string](),
			MaxRetries:          Some(3),
			IsEnabled:           true,
			CreatedAt:           time.Now(),
			UpdatedAt:           time.Now(),
		},
	}
}
